package main

import (
	"bytes"
	"errors"
	"fmt"
	"net"
)

type InRequest struct {
	Recv    []byte
	Method  []byte
	Path    []byte
	Head    []byte
	Body    []byte
	Headers map[string][][]byte
}

func (r *InRequest) Parse(headers bool) {
	m := REQ_REG.FindSubmatch(r.Recv)
	if len(m) == 0 {
		panic(errors.New("REQ_FORMAT_ERROR"))
	}

	r.Method, r.Path, r.Head, r.Body = m[1], m[2], m[3], m[4]

	if headers {
		r.Headers = make(map[string][][]byte)
		headers := HEADER_REG.FindAllSubmatch(r.Head, -1)
		for _, h := range headers {
			r.Headers[string(bytes.ToLower(h[1]))] = h
		}
	}
}

type OutRequest struct {
	Method  []byte
	Path    []byte
	Headers [][]byte
	Bodys   [][]byte
}

func (r *OutRequest) Send(conn *net.TCPConn) int {
	per := []byte(fmt.Sprintf(REQ_FORMAT, r.Method, r.Path))
	return SendHttp(conn, per, r.Headers, r.Bodys)
}
