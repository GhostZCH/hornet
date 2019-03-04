package main

import (
	"fmt"
)

var RSP_FORMAT = "HTTP/1.1 %d %s\r\nServer: Hornet\r\nConnection: keep-alive\r\nContent-Length: %d\r\n\r\n"

var RSP_MAP = map[int]string{200: "OK", 201: "Created", 404: "Not Found"}

type SendHandler struct {
}

func NewSendHandler() (h *SendHandler) {
	h = new(SendHandler)
	return h
}

func (h *SendHandler) Handle(t *Transaction) (ok bool) {
	defer t.Conn.SetNoDelay(true)

	cl := 0
	if t.Rsp.Body != nil {
		for _, b := range t.Rsp.Body {
			cl += len(b)
		}
	}

	t.Conn.SetNoDelay(false)
	_, e := fmt.Fprintf(t.Conn, RSP_FORMAT, t.Rsp.Status, RSP_MAP[t.Rsp.Status], cl)
	Success(e)

	if t.Rsp.Head != nil {
		for _, b := range t.Rsp.Head {
			_, e := t.Conn.Write(b)
			Success(e)
		}
	}

	if t.Rsp.Body != nil {
		for _, b := range t.Rsp.Body {
			_, e := t.Conn.Write(b)
			Success(e)
		}
	}

	return true
}
