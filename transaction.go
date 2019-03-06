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

var HTTP_SEMICOLON = []byte(": ")

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
	t.recv = recv
	t.conn = conn
	t.store = store
	t.Time = time.Now().UnixNano() / 1e6 // ms
	t.Req.Init()
	t.Rsp.Init()
	return t
}

func (t *Transaction) Finish(err error) {
	t.Err = err
	t.Detal = time.Now().UnixNano()/1e6 - r.Time
}

func (t *Transaction) String() string {
	return fmt.Sprintf("%v %v %v %v %v %v %v %v\n", t.Time, t.Detal, t.Req.Method, t.Req.Path, t.Req.Arg, t.Conn.RemoteAddr(), r.Status, r.Err)
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
	append(t.rsp.Head, data[:item.Item.HeadLen])
	append(t.rsp.Body, data[item.Item.HeadLen:])
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
		if n, e := hex.Decode(g, h[1]); e != nil || n != KEY_HASH_LEN {
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
		if n, e := hex.Decode(g, h[1]); e != nil || n != KEY_HASH_LEN {
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

	buf := bytes.NewBuffer()
	for _, h := range t.req.Headers {
		buf.Write(h[0])
		buf.Write(HTTP_SEMICOLON)
		buf.Write(h[1])
		buf.Write(HTTP_END)
	}

	head = buf.Bytes()
	item, data := t.store.Add(id, len(head)+cl)
	item.Item.BodyLen = cl
	item.Item.HeadLen = len(head)
	item.Item.Expire = 0 //TODO  Expire & etag
	item.Item.Grp = g
	item.Item.RawKeyLen = len(raw)
	copy(item.Item.RawKey, raw)

	copy(data, head)
	data = data[len[head]:]

	copy(data, t.req.body)
	data = data[len[t.req.body]:]

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

	switch t.req.Method {
	case "GET":
		t.get()
	case "DEL":
		t.del()
	case "PUT":
		t.put()
	}

	t.rsp.Send(t.conn)
}
