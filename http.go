package main

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"regexp"
	"strconv"
)

var HTTP_END []byte = []byte("\r\n")
var HTTP_SEMICOLON = []byte(": ")
var RSP_MAP = map[int]string{
	200: "200 OK",
	206: "206 Partial Content",
	400: "400 Bad Request",
	500: "500 Internal Server Error"}
var REQ_FORMAT string = "%s /%s HTTP/1.1\r\n"
var RSP_FORMAT string = "HTTP/1.1 %s\r\nContent-Length: %d\r\n"
var HEADER_REG = regexp.MustCompile("(\\S+):\\s*(\\S*)\r\n")
var RSP_REG = regexp.MustCompile("HTTP/1.1 (\\d+) \\w+\r\n([\\S\\s]*)\r\n([\\S\\s]*)")
var REQ_REG = regexp.MustCompile("^(GET|DELETE) (.*) HTTP/1.1\r\n([\\S\\s]*)\r\n([\\S\\s]*)")

type InRequest struct {
	Recv    []byte
	Method  []byte
	Path    []byte
	Head    []byte
	Body    []byte
	Headers map[string][][]byte
}

func (r *InRequest) Parse() {
	m := REQ_REG.FindSubmatch(r.Recv)
	if len(m) == 0 {
		panic(errors.New("REQ_FORMAT_ERROR"))
	}

	r.Method, r.Path, r.Head, r.Body = m[1], m[2], m[3], m[4]

	r.Headers = make(map[string][][]byte)
	headers := HEADER_REG.FindAllSubmatch(r.Head, -1)
	for _, h := range headers {
		r.Headers[string(bytes.ToLower(h[1]))] = h
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

type InRespose struct {
	Status  int
	Recv    []byte
	Head    []byte
	Body    []byte
	Headers map[string][][]byte
}

func (r *InRespose) Parse() {
	var err error

	m := RSP_REG.FindSubmatch(r.Recv)
	if len(m) == 0 {
		panic(errors.New("RSP_FORMAT_ERROR"))
	}

	r.Head, r.Body = m[2], m[3]
	r.Status, err = strconv.Atoi(string(m[1]))
	Success(err)

	r.Headers = make(map[string][][]byte)
	headers := HEADER_REG.FindAllSubmatch(r.Head, -1)
	for _, h := range headers {
		r.Headers[string(bytes.ToLower(h[1]))] = h
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
