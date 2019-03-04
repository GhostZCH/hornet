package main

import ()

type RecvHandler struct {
	bufLen int
}

func NewRecvHandler() (h *RecvHandler) {
	h = new(RecvHandler)
	h.bufLen = GConfig["http.header.maxlen"].(int)
	return h
}

func (h *RecvHandler) Handle(t *Transaction) (ok bool) {
	buf := make([]byte, h.bufLen)

	n, e := t.Conn.Read(buf)
	Success(e)

	buf = t.Req.ParseReqLine(buf[:n])
	if t.Req.Method != "GET" {
		t.Req.Body = t.Req.ParseHeaders(buf)
	}

	return true
}
