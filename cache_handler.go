package main

import (
	"crypto/md5"
	"errors"
	"fmt"
	"net"
	"regexp"
	"strconv"
	"sync"
	"time"
)

type LockKey struct {
	id      Key
	rbSize  uint32
	rbIndex uint32
}

type Upstream struct {
	keep int
	addr *net.TCPAddr
	free chan *net.TCPConn
}

type CacheHandler struct {
	name      string
	devices   *DeviceManager
	heartBeat *net.UDPConn
	upstream  *Upstream
	lock      sync.Mutex
	pullLock  map[LockKey]sync.Mutex
}

func NewCacheHandler() (h *CacheHandler) {
	var err error

	addr, err := net.ResolveUDPAddr("udp", GConfig["common.heartbeat.addr"].(string))
	Success(err)

	h = new(CacheHandler)
	h.name = GConfig["common.name"].(string)
	h.devices = NewDeviceManager()
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
	h.devices.Close()
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

	switch trans.Req.Method[0] {
	case 'G':
		h.get(trans)
	case 'P':
		h.get(trans)
	case 'D':
		h.del(trans)
	}

	trans.Rsp.Send(trans.Conn)
}

func (h *CacheHandler) get(trans *Transaction) {
	var ranges []int
	var start, end int

	id := md5.Sum(trans.Req.Path)
	// TODO handle request
	if rg, ok := trans.Req.Headers["range"]; ok {
		start, end = parse_range(rg[2])
	}

	item, data, cache := h.devices.Get(Key{id, 0})

	for _, rg := range ranges {
		item, data, cache := h.devices.Get(id)
		if item == nil {
			trans.SvrMsg = "miss"
			if h.upstream == nil {
				trans.Rsp.Status = 404
				return
			}

			h.pull(trans)
			item, data, cache = h.store.Get(id)
			*cache = "miss"
		}

		if item != nil {
			trans.SvrMsg = *cache
			via := fmt.Sprintf("X-Via-Cache: %s %s\r\n", h.name, *cache)
			trans.Rsp.Status = 200
			trans.Rsp.Headers = append(trans.Rsp.Headers, []byte(via))
			trans.Rsp.Headers = append(trans.Rsp.Headers, data[:item.HeadLen])
			trans.Rsp.Bodys = append(trans.Rsp.Bodys, data[item.HeadLen:])
			return
		}
	}
}

func (h *CacheHandler) del(trans *Transaction) {
	trans.Rsp.Status = 200

	if trans.Req.Path != nil {
		id := DecodeKey(trans.Req.Path)
		h.store.Delete(id)
		return
	}

	if hdr, ok := trans.Req.Headers["hornet-group"]; ok {
		h.store.DeleteBatch(func(item *Item) bool {
			return item.Grp == g
		})
		return
	}

	if hdr, ok := trans.Req.Headers["hornet-regex"]; ok {
		reg := regexp.MustCompile(string(hdr[2]))
		h.store.DeleteBatch(func(item *Item) bool {
			return reg.Match(item.RawKey)
		})
		return
	}

	panic(errors.New("NO_DEL_PARAMS"))
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
		item.ID = DecodeKey(trans.Req.Path)
	}

	data := h.store.Add(item)

	copy(data, head)
	data = data[len(head):]

	copy(data, rsp.Body)
	data = data[len(rsp.Body):]

	if n, e := u.Read(data); n != len(data) || e != nil {
		h.store.Delete(item.ID)
		panic(e) // n != len(data)
	}

	trans.Rsp.Status = 201

	if len(h.upstream.free) < h.upstream.keep {
		h.upstream.free <- u
	}
}

func parse_range(r []byte) (start, end int) {
	start, end = 0, -1

	m := REQ_RANGE_REG.FindSubmatch(r)
	if len(m) != 3 || (len(m[1]) == 0 && len(m[2]) == 0) {
		panic(errors.New("RANGE_ERROR"))
	}

	if len(m[1]) != 0 {
		start, _ = strconv.Atoi(string(m[1]))
	}

	if len(m[2]) != 0 {
		end, _ = strconv.Atoi(string(m[2]))
	}

	if start >= end {
		panic(errors.New("RANGE_ERROR"))
	}

	return start, end
}
