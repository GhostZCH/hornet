package main

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"strconv"
)

type InRespose struct {
	Status  int
	Recv    []byte
	Head    []byte
	Body    []byte
	Headers map[string][][]byte
}

func (r *InRespose) Parse(headers bool) {
	var err error

	m := RSP_REG.FindSubmatch(r.Recv)
	if len(m) == 0 {
		panic(errors.New("RSP_FORMAT_ERROR"))
	}

	r.Head, r.Body = m[2], m[3]
	r.Status, err = strconv.Atoi(string(m[1]))
	Success(err)

	// 复用
	if headers {
		r.Headers = make(map[string][][]byte)
		headers := HEADER_REG.FindAllSubmatch(r.Head, -1)
		for _, h := range headers {
			r.Headers[string(bytes.ToLower(h[1]))] = h
		}
	}
}

type OutRespose struct {
	Status  int
	Headers [][]byte
	Bodys   [][]byte
}

func (r *OutRespose) Send(conn *net.TCPConn) int {
	cl := 0
	for _, b := range r.Bodys {
		cl += len(b)
	}

	per := []byte(fmt.Sprintf(RSP_FORMAT, RSP_MAP[r.Status], cl))
	return SendHttp(conn, per, r.Headers, r.Bodys)
}
