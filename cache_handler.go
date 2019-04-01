package main

import (
	"bytes"
	"encoding/hex"
	"errors"
	"net"
	"regexp"
	"strconv"
	"time"
)

type CacheHandler struct {
	store     *Store
	heartBeat *net.UDPConn
}

func NewCacheHandler() (h *CacheHandler) {
	var err error

	addr, err := net.ResolveUDPAddr("udp", GConfig["common.heartbeat.addr"].(string))
	Success(err)

	h = new(CacheHandler)
	h.store = NewStore()
	h.heartBeat, err = net.DialUDP("udp", nil, addr)
	Success(err)

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
		span := time.Duration(GConfig["cache.span_ms"].(int)) * time.Millisecond
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

	trans.Req.ParseBasic(buf[:n])

	switch string(trans.Req.Method) {
	case "GET":
		h.get(trans)
	case "DEL":
		h.del(trans)
	case "PUT":
		h.put(trans)
	}

	trans.Rsp.Send(trans.Conn, nil)
}

func (h *CacheHandler) get(trans *Transaction) {
	trans.Req.ParseArgs()

	var key Key
	key.Hash = DecodeKey(trans.Req.Path)

	// TODO range /xxxxxxxxxxxxxxxx[?start=100&end=500]
	// 可能要跨越多个块，第一个只保存head, head不存在就算删除，head.range = 0

	item, data := h.store.Get(key)
	if item == nil {
		trans.Rsp.Status = 404
		trans.Rsp.Send(trans.Conn, nil)
		return
	}

	trans.Rsp.Status = 200
	trans.Rsp.Heads = append(trans.Rsp.Heads, data[:item.Item.HeadLen])
	trans.Rsp.Bodys = append(trans.Rsp.Bodys, data[item.Item.HeadLen:])
	trans.Rsp.Send(trans.Conn, nil)
}

func (h *CacheHandler) del(trans *Transaction) {
	trans.Rsp.Status = 200

	if trans.Req.Path != nil {
		var id Key
		n, e := hex.Decode(id.Hash[:], trans.Req.Path[1:33])
		if e != nil || n != KEY_HASH_LEN {
			panic(errors.New("ID_FORMAR_ERROR"))
		}
		// del header for range
		h.store.Del(id)
		return
	}

	trans.Req.ParseHeaders()

	if hdr, ok := trans.Req.Headers["Hornet-Group"]; ok {
		var g HKey
		if n, e := hex.Decode(g[:], hdr[2]); e != nil || n != KEY_HASH_LEN {
			panic(errors.New("GRP_FORMAR_ERROR"))
		}
		h.store.DelByGroup(g)
		return
	}

	if hdr, ok := trans.Req.Headers["Hornet-Regexp"]; ok {
		reg := regexp.MustCompile(string(hdr[2]))
		h.store.DelByRawKey(reg)
		return
	}

	panic(errors.New("NO_DEL_PARAMS"))
}

func (h *CacheHandler) put(trans *Transaction) {
	trans.Req.ParseArgs()
	trans.Req.ParseHeaders()

	var id Key
	var g HKey
	var cl int
	var raw []byte

	// TODO range

	if trans.Req.Path != nil {
		n, e := hex.Decode(id.Hash[:], trans.Req.Path)
		if e != nil || n != KEY_HASH_LEN {
			panic(errors.New("ID_FORMAR_ERROR"))
		}
	}

	if hdr, ok := trans.Req.Headers["Hornet-Group"]; ok {
		if n, e := hex.Decode(g[:], hdr[2]); e != nil || n != KEY_HASH_LEN {
			panic(errors.New("GRP_FORMAR_ERROR"))
		}
	} else {
		panic(errors.New("GRP_NOT_SET"))
	}

	if h, ok := trans.Req.Headers["Content-Length"]; ok {
		if n, e := strconv.Atoi(string(h[1])); e != nil {
			panic(e)
		} else {
			cl = n
		}
	} else {
		panic(errors.New("CONTENT_LEN_NOT_SET"))
	}

	if h, ok := trans.Req.Headers["Hornet-Raw-Key"]; ok {
		raw = h[1]
	} else {
		panic(errors.New("RAW_KEY_NOT_SET"))
	}

	for _, h := range GConfig["cache.http.header.discard"].([]interface{}) {
		delete(trans.Req.Headers, h.(string))
	}

	buf := new(bytes.Buffer)
	for _, h := range trans.Req.Headers {
		buf.Write(h[0])
		buf.Write(HTTP_END)
	}

	head := buf.Bytes()
	item, data := h.store.Add(id, len(head)+cl)
	item.Item.BodyLen = uint32(cl)
	item.Item.HeadLen = uint32(len(head))
	item.Item.Expire = 0 //TODO  Expire & etag
	item.Item.Grp = g
	item.Item.RawKeyLen = uint32(len(raw))
	copy(item.Item.RawKey[:], raw)

	copy(data, head)
	data = data[len(head):]

	copy(data, trans.Req.Body)
	data = data[len(trans.Req.Body):]

	if n, e := trans.Conn.Read(data); n != len(data) || e != nil {
		h.store.Del(id)
		panic(e) // n != len(data)
	}

	item.Putting = false
	trans.Rsp.Status = 201
}
