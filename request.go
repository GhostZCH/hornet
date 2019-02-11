package main

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"regexp"
	"strconv"
	"time"
)

var HTTP_SPLITER = []byte("\r\n")

var SPC_RSP_200 []byte = []byte("HTTP/1.1 200 OK\r\nServer: Hornet\r\nConnection: keep-alive\r\nContent-Length: 0\r\n\r\n")
var SPC_RSP_201 []byte = []byte("HTTP/1.1 201 Created\r\nServer: Hornet\r\nConnection: keep-alive\r\nContent-Length: 0\r\n\r\n")
var SPC_RSP_404 []byte = []byte("HTTP/1.1 404 Not Found\r\nServer: Hornet\r\nConnection: keep-alive\r\nContent-Length: 0\r\n\r\n")
var CACHE_RSP_HEADER []byte = []byte("HTTP/1.1 200 OK\r\nServer: Hornet\r\nConnection: keep-alive\r\n")

var REQLINE_REG = regexp.MustCompile("^(GET|POST|DEL) /([0-9]+)/([0-9]+) HTTP/1.1\r\n")
var HEADR_REG = regexp.MustCompile("(.*): (.*)\r\n")

type Request struct {
	Method    string
	Dir       uint64
	ID        uint64
	Err       error
	Headers   [][][]byte
	Addr      string
	Time      int64
	DetalTime int64
	Conn      *net.TCPConn
}

func NewRequest(conn *net.TCPConn) (r *Request) {
	r = new(Request)
	r.Conn = conn
	r.Time = time.Now().UnixNano() / 1e6 // ms
	r.Method = "-"
	return r
}

func (r *Request) ParseReqLine(buf []byte) (tail []byte) {
	match := REQLINE_REG.FindSubmatch(buf)
	if len(match) == 0 {
		r.Err = errors.New("HEADER_ERROR")
		return nil
	}

	r.Method = string(match[1])
	r.Dir, _ = strconv.ParseUint(string(match[2]), 10, 64)
	r.ID, _ = strconv.ParseUint(string(match[3]), 10, 64)
	return buf[len(match[0]):]
}

func (r *Request) ParseHeaders(buf []byte) (tail []byte) {
	r.Headers = HEADR_REG.FindAllSubmatch(buf, -1)
	if len(r.Headers) > 1 {
		l := 0
		for _, h := range r.Headers {
			l += len(h[0])
		}
		tail = buf[l:]
	}
	if !bytes.HasPrefix(tail, HTTP_SPLITER) {
		r.Err = errors.New("HEADER_TOO_LARGE")
		return nil
	}
	return tail[2:]
}

func (r *Request) FindHeader(name []byte) (header [][]byte) {
	for _, h := range r.Headers {
		if h != nil && bytes.Equal(h[1], name) {
			return h
		}
	}
	return nil
}

func (r *Request) DelHeader(name []byte) {
	for i, h := range r.Headers {
		if h != nil && bytes.Equal(h[1], name) {
			r.Headers[i] = nil
		}
	}
}

func (r *Request) GenerateHeader(buf *bytes.Buffer) {
	for _, h := range r.Headers {
		if h != nil {
			buf.Write(h[0])
		}
	}
}

func (r *Request) Finish() {
	r.DetalTime = time.Now().UnixNano()/1e6 - r.Time
}

func (r *Request) String() string {
	return fmt.Sprint(r.Time, r.DetalTime, r.Dir, r.ID, r.Method, " ", r.Conn.RemoteAddr(), r.Err, "\n")
}
