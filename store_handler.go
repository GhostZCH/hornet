package main

import (
	"bytes"
	"errors"
)

type StoreHandler struct {
	store   *Store
	discard [][]byte
}

func NewStoreHandler(s *Store) (h *StoreHandler) {
	h = new(StoreHandler)
	h.store = s
	for _, hdr := range GConfig["http.header.discard"].([]interface{}) {
		append(h.discard, []byte(hdr.(string)))
	}
	return h
}

func (h *StoreHandler) Handle(t *Transaction) bool {
	if t.Rsp.Status != 0 {
		return true
	}

	switch t.Req.Method {
	case "GET":
		return h.get(t)
	case "DEL":
		return h.delete(t)
	case "POST":
		return h.post(t)
	}
}

func (h *StoreHandler) generateHeader(r *Request, buf *bytes.Buffer) {
	for _, h := range r.Headers {
		if h != nil {
			buf.Write(h[0])
		}
	}
}

func (h *StoreHandler) post(t *Transaction) bool {
	s := h.store

	//TODO write

	if item, _ := s.Get(r.Grp, r.ID); item != nil {
		r.Status = 200
	}

	for _, hdr := range h.discard {
		r.DelHeader([]byte(hdr))
	}

	var headerBuf bytes.Buffer
	headerBuf.Write(CACHE_RSP_HEADER)
	r.GenerateHeader(&headerBuf)
	headerBuf.Write(HTTP_SPLITER)

	clh := r.FindHeader([]byte("Content-Length"))
	if clh == nil {
		panic(errors.New("NO_CONTENT_LENGTH"))
	}

	cl, err := strconv.ParseInt(string(clh[2]), 10, 64)
	AssertSuccess(err)

	h := headerBuf.Bytes()
	l := int(cl) + len(h)
	buf, item := s.store.Add(r.Dir, r.ID, l)

	copy(buf, h)
	n := len(h)

	copy(buf[n:], recv)
	n += len(recv)

	recv := make([]byte, GConfig["http.body.bufsize"].(int))
	for n < l {
		if rn, err := conn.Read(recv); err != nil {
			s.store.Del(r.Dir, r.ID)
			panic(err)
		} else {
			copy(buf[n:], recv[:rn])
			n += rn
		}
	}
	item.Putting = false
	conn.Write(SPC_RSP_201)
}

func (h *StoreHandler) delete(t *Transaction) bool {
	h.store.Del(r.Dir, r.ID)
	r.Status = 200
	return true
}

func (h *StoreHandler) get(t *Transaction) bool {
	item, data := h.store.Get(rh.req.Dir, rh.req.ID)
	if data == nil || item.Putting || item {
		r.Status = 404
		return true
	} else {
		conn.Write(rsp)
		return false
	}
}
