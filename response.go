package main

import (
	"errors"
	"fmt"
	"net"
	"strconv"
)

type Respose struct {
	Status  int
	Head    []byte
	Body    []byte
	InHeads map[string][][]byte
	Heads   [][]byte
	Bodys   [][]byte
}

func (r *Respose) Init() {
	r.InHeads = make(map[string][][]byte)
}

func (r *Respose) ParseBasic(buf []byte) {
	var err error

	m := RSP_REG.FindSubmatch(buf)
	if len(m) == 0 {
		panic(errors.New("RSP_FORMＡT_ERROR"))
	}

	r.Head, r.Body = m[2], m[3]
	r.Status, err = strconv.Atoi(string(m[1]))
	Success(err)
}

func (r *Respose) ParseHeaders() {
	headers := HEADER_REG.FindAllSubmatch(r.Head, -1)
	for _, h := range headers {
		r.InHeads[string(h[0])] = h
	}
}

func (r *Respose) Send(conn *net.TCPConn, buf []byte) {
	conn.SetNoDelay(false)
	defer conn.SetNoDelay(true)

	if buf != nil {
		Success(conn.Write(buf))
		return
	}

	bodyLen := 0
	for _, b := range r.Bodys {
		bodyLen += len(b)
	}

	Success(fmt.Fprintf(conn, RSP_FORMAT, RSP_MAP[r.Status], bodyLen))

	for _, b := range r.Heads {
		Success(conn.Write(b))
	}

	Success(conn.Write(HTTP_END))

	for _, b := range r.Bodys {
		Success(conn.Write(b))
	}
}
