package main

import (
	"errors"
	"fmt"
	"net"
	"regexp"
	"time"
)

type Upstream struct {
	keep int
	addr *net.TCPAddr
	free chan *net.TCPConn
}

type CacheHandler struct {
	name      string
	store     *StoreManager
	heartBeat *net.UDPConn
	upstream  *Upstream
}

func NewCacheHandler() (h *CacheHandler) {
	var err error

	addr, err := net.ResolveUDPAddr("udp", GConfig["common.heartbeat.addr"].(string))
	Success(err)

	h = new(CacheHandler)
	h.name = GConfig["common.name"].(string)
	h.store = NewStoreManager()
	h.heartBeat, err = net.DialUDP("udp", nil, addr)
	Success(err)

	if GConfig["cache.upstream"].(bool) {
		h.upstream = new(Upstream)
		h.upstream.keep = GConfig["cache.upstream.keep"].(int)
		h.upstream.addr, err = net.ResolveTCPAddr("tcp", GConfig["cache.upstream.addr"].(string))
		h.upstream.free = make(chan *net.TCPConn, h.upstream.keep)
	}

	return h
}

func (h *CacheHandler) GetCtx() interface{} {
	return make([]byte, GConfig["common.http.header.maxlen"].(int))
}

func (h *CacheHandler) GetListener() string {
	return GConfig["cache.addr"].(string)
}

func (h *CacheHandler) Start() {
	go func() {
		msg := []byte(GConfig["cache.addr"].(string))
		span := time.Duration(GConfig["cache.heartbeat_ms"].(int)) * time.Millisecond
		for {
			if _, e := h.heartBeat.Write(msg); e != nil {
				Lerror("heartBeat.Write", e)
				break
			}
			time.Sleep(span)
		}
	}()
}

func (h *CacheHandler) Close() {
	Lwarn(h, "close")
	h.heartBeat.Close()
	h.store.Close()
}

func (h *CacheHandler) Handle(trans *Transaction) {
	buf := trans.Ctx.([]byte)

	n, e := trans.Conn.Read(buf)
	Success(e)

	trans.Req.Recv = buf[:n]
	trans.Req.Parse(true)
	if h, ok := trans.Req.Headers["hornet-log"]; ok {
		trans.ClientMsg = string(h[2])
	}

	switch string(trans.Req.Method) {
	case "GET":
		h.get(trans)
	case "DELETE":
		h.del(trans)
	case "POST":
		h.post(trans)
	}

	trans.Rsp.Send(trans.Conn)
}

func (h *CacheHandler) get(trans *Transaction) {
	var key Key
	key.Hash = DecodeKey(trans.Req.Path)

	// TODO range /xxxxxxxxxxxxxxxx
	// 可能要跨越多个块，第一个只保存head, head不存在就算删除，head.range = 0

	item, data, cache := h.store.Get(key)
	if item == nil {
		trans.SvrMsg = "miss"
		if h.upstream == nil {
			trans.Rsp.Status = 404
			return
		}
		// TODO 里面的代码要和post复用
		h.pull(trans)
		item, data, cache = h.store.Get(key)
		*cache = "miss"
	}

	if item != nil {
		trans.SvrMsg = *cache
		via := fmt.Sprintf("X-Via-Cache: %s %s\r\n", h.name, *cache)
		trans.Rsp.Status = 200
		trans.Rsp.Headers = append(trans.Rsp.Headers, []byte(via))
		trans.Rsp.Headers = append(trans.Rsp.Headers, data[:item.Info.HeadLen])
		trans.Rsp.Bodys = append(trans.Rsp.Bodys, data[item.Info.HeadLen:])
		return
	}
}

func (h *CacheHandler) del(trans *Transaction) {
	trans.Rsp.Status = 200

	if trans.Req.Path != nil {
		var id Key
		id.Hash = DecodeKey(trans.Req.Path)
		h.store.Delete(id)
		return
	}

	if hdr, ok := trans.Req.Headers["hornet-group"]; ok {
		g := DecodeKey(hdr[2])
		h.store.DeleteBatch(func(item *Item) bool {
			return item.Info.Grp == g
		})
		return
	}

	if hdr, ok := trans.Req.Headers["hornet-regex"]; ok {
		reg := regexp.MustCompile(string(hdr[2]))
		h.store.DeleteBatch(func(item *Item) bool {
			return reg.Match(item.Info.RawKey[:item.Info.RawKeyLen])
		})
		return
	}

	panic(errors.New("NO_DEL_PARAMS"))
}

func (h *CacheHandler) post(trans *Transaction) {
	item, head := GenerateItem(trans.Req.Headers)

	if trans.Req.Path != nil {
		item.Info.ID.Hash = DecodeKey(trans.Req.Path)
	}

	data := h.store.Add(item)

	copy(data, head)
	data = data[len(head):]

	copy(data, trans.Req.Body)
	data = data[len(trans.Req.Body):]

	if n, e := trans.Conn.Read(data); n != len(data) || e != nil {
		h.store.Delete(item.Info.ID)
		panic(e) // n != len(data)
	}

	item.Putting = false
	trans.Rsp.Status = 201
}

func (h *CacheHandler) pull(trans *Transaction) {
	var u *net.TCPConn
	var e error

	if len(h.upstream.free) != 0 {
		u = <-h.upstream.free
	} else {
		u, e = net.DialTCP("tcp", nil, h.upstream.addr)
		Success(e)
	}

	// req := OutRequest{Method: trans.Req.Method, Path: trans.Req.Path}
	// todo modify headers

	u.Write(trans.Req.Recv)

	recv := make([]byte, GConfig["common.http.header.maxlen"].(int))
	n, err := u.Read(recv)
	Success(err)

	rsp := InRespose{}
	rsp.Recv = recv[:n]
	rsp.Parse(true)

	item, head := GenerateItem(rsp.Headers)
	// TODO range
	if trans.Req.Path != nil {
		item.Info.ID.Hash = DecodeKey(trans.Req.Path)
	}

	data := h.store.Add(item)

	copy(data, head)
	data = data[len(head):]

	copy(data, rsp.Body)
	data = data[len(rsp.Body):]

	if n, e := u.Read(data); n != len(data) || e != nil {
		h.store.Delete(item.Info.ID)
		panic(e) // n != len(data)
	}

	item.Putting = false
	trans.Rsp.Status = 201

	if len(h.upstream.free) < h.upstream.keep {
		h.upstream.free <- u
	}
}
