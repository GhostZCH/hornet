package main

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"regexp"
	"strconv"
	"time"
)

type Transaction struct {
	conn  *net.TCPConn
	req   Request
	rsp   Respose
	err   error
	time  int64
	delta int64
	store *Store
}

func NewTrans(conn *net.TCPConn, store *Store) (t *Transaction) {
	t = new(Transaction)
	t.conn = conn
	t.store = store
	t.time = time.Now().UnixNano() / 1e6 // ms
	t.req.Init()
	t.rsp.Init()
	return t
}

func (t *Transaction) Finish(err error) {
	t.err = err
	t.delta = time.Now().UnixNano()/1e6 - t.time
}

func (t *Transaction) String() string {
	return fmt.Sprintf("%v %v %v %v %v %v %v %v\n",
		t.time, t.delta, t.req.Method, t.req.Path,
		t.req.Arg, t.conn.RemoteAddr(), t.rsp.Status, t.err)
}

func (t *Transaction) get() {
	t.req.ParseArgs()

	var id Key
	if n, e := hex.Decode(id.Hash[:], t.req.Path[1:33]); e != nil || n != KEY_HASH_LEN {
		panic(errors.New("ID_FORMAR_ERROR"))
	}

	// TODO range /xxxxxxxxxxxxxxxx[?start=100&end=500]
	// 可能要跨越多个块，第一个只保存head, head不存在就算删除，head.range = 0

	item, data := t.store.Get(id)
	if item == nil {
		t.rsp.Status = 404
		return
	}

	t.rsp.Status = 200
	t.rsp.Head = append(t.rsp.Head, data[:item.Item.HeadLen])
	t.rsp.Body = append(t.rsp.Body, data[item.Item.HeadLen:])
}

func (t *Transaction) del() {
	t.rsp.Status = 200

	if len(t.req.Path) == 33 {
		var id Key
		n, e := hex.Decode(id.Hash[:], t.req.Path[1:33])
		if e != nil || n != KEY_HASH_LEN {
			panic(errors.New("ID_FORMAR_ERROR"))
		}
		// del header for range
		t.store.Del(id)
		return
	}

	t.req.ParseHeaders()

	if h, ok := t.req.Headers["Hornet-Group"]; ok {
		var g HKey
		if n, e := hex.Decode(g[:], h[1]); e != nil || n != KEY_HASH_LEN {
			panic(errors.New("GRP_FORMAR_ERROR"))
		}
		t.store.DelByGroup(g)
		return
	}

	if h, ok := t.req.Headers["Hornet-Regexp"]; ok {
		reg := regexp.MustCompile(string(h[1]))
		t.store.DelByRawKey(reg)
		return
	}

	panic(errors.New("NO_DEL_PARAMS"))
}

func (t *Transaction) put() {
	t.req.ParseArgs()
	t.req.ParseHeaders()

	var id Key
	var g HKey
	var cl int
	var raw []byte

	// TODO range

	if len(t.req.Path) == 33 {
		n, e := hex.Decode(id.Hash[:], t.req.Path[1:33])
		if e != nil || n != KEY_HASH_LEN {
			panic(errors.New("ID_FORMAR_ERROR"))
		}
	}

	if h, ok := t.req.Headers["Hornet-Group"]; ok {
		if n, e := hex.Decode(g[:], h[1]); e != nil || n != KEY_HASH_LEN {
			panic(errors.New("GRP_FORMAR_ERROR"))
		}
	} else {
		panic(errors.New("GRP_NOT_SET"))
	}

	if h, ok := t.req.Headers["Content-Length"]; ok {
		if n, e := strconv.Atoi(string(h[1])); e != nil {
			panic(e)
		} else {
			cl = n
		}
	} else {
		panic(errors.New("CONTENT_LEN_NOT_SET"))
	}

	if h, ok := t.req.Headers["Hornet-Raw-Key"]; ok {
		raw = h[1]
	} else {
		panic(errors.New("RAW_KEY_NOT_SET"))
	}

	for _, h := range GConfig["http.header.discard"].([]interface{}) {
		delete(t.req.Headers, h.(string))
	}

	buf := new(bytes.Buffer)
	for _, h := range t.req.Headers {
		buf.Write(h[0])
		buf.Write(HTTP_SEMICOLON)
		buf.Write(h[1])
		buf.Write(HTTP_END)
	}

	head := buf.Bytes()
	item, data := t.store.Add(id, len(head)+cl)
	item.Item.BodyLen = uint32(cl)
	item.Item.HeadLen = uint32(len(head))
	item.Item.Expire = 0 //TODO  Expire & etag
	item.Item.Grp = g
	item.Item.RawKeyLen = uint32(len(raw))
	copy(item.Item.RawKey[:], raw)

	copy(data, head)
	data = data[len(head):]

	copy(data, t.req.Body)
	data = data[len(t.req.Body):]

	if n, e := t.conn.Read(data); n != len(data) || e != nil {
		t.store.Del(id)
		panic(e) // n != len(data)
	}

	item.Putting = false
	t.rsp.Status = 201
}

func (t *Transaction) Handle(recv []byte) {
	n, e := t.conn.Read(recv)
	Success(e)

	t.req.ParseBasic(recv[:n])

	switch string(t.req.Method) {
	case "GET":
		t.get()
	case "DEL":
		t.del()
	case "PUT":
		t.put()
	}

	t.rsp.Send(t.conn)
}
