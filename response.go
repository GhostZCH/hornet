package main

import (
	"fmt"
	"net"
)

var HTTP_END = []byte("\r\n")
var RSP_MAP = map[int]string{200: " 200 OK", 201: " 201 Created", 404: "404 Not Found"}
var RSP_FORMAT = "HTTP/1.1 %s\r\nServer: Hornet\r\nConnection: keep-alive\r\nContent-Length: %d\r\n\r\n"

type Respose struct {
	Status int
	Head   [][]byte
	Body   [][]byte
}

func (r *Respose) Init() {
	r.Head = make([][]byte, 4)
	r.Head = make([][]byte, 4)
}

func (r *Respose) Send(conn *net.TCPConn) {
	cl := 0
	for _, b := range r.Body {
		cl += len(b)
	}

	conn.SetNoDelay(false)
	defer conn.SetNoDelay(true)

	Success(fmt.Fprintf(conn, RSP_FORMAT, RSP_MAP[r.Status], cl))

	for _, b := range r.Head {
		Success(conn.Write(b))
	}

	Success(conn.Write(HTTP_END))

	for _, b := range r.Body {
		Success(conn.Write(b))
	}
}
