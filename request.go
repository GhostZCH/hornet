package main

import (
	"errors"
	"fmt"
	"net"
)

type Request struct {
	Method  []byte
	Path    []byte
	Arg     []byte
	Head    []byte
	Body    []byte
	Args    map[string][][]byte
	Headers map[string][][]byte
}

func (r *Request) Init() {
	r.Args = make(map[string][][]byte)
	r.Headers = make(map[string][][]byte)
}

func (r *Request) ParseBasic(buf []byte) {
	m := REQ_REG.FindSubmatch(buf)
	if len(m) == 0 {
		panic(errors.New("REQ_FORMAT_ERROR"))
	}

	r.Method, r.Path, r.Arg, r.Head, r.Body = m[1], m[2], m[3], m[4], m[5]
}

func (r *Request) ParseHeaders() {
	headers := HEADER_REG.FindAllSubmatch(r.Head, -1)
	for _, h := range headers {
		r.Headers[string(h[1])] = h
	}
}

func (r *Request) ParseArgs() {
	args := ARG_REQ.FindAllSubmatch(r.Head, -1)
	for _, a := range args {
		r.Headers[string(a[1])] = a
	}
}

func (r *Request) Send(conn *net.TCPConn, buf []byte) int {
	var err error
	sum, n := 0, 0

	conn.SetNoDelay(false)
	defer conn.SetNoDelay(true)

	if buf != nil {
		sum, err = conn.Write(buf)
		Success(err)
	}

	n, err = fmt.Fprintf(conn, REQ_FORMAT, r.Method, r.Path, r.Arg)
	Success(err)
	sum += n

	n, err = conn.Write(r.Head)
	Success(err)
	sum += n

	n, err = conn.Write(HTTP_END)
	Success(err)
	sum += n

	n, err = conn.Write(r.Body)
	Success(err)
	sum += n

	return sum
}
