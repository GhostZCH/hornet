package main

import (
	"bytes"
	"fmt"
	"net"
	"regexp"
	"time"
)

var REQ_REG = regexp.MustCompile("^(GET|PUT|DEL) /(.*)(\\?\\S+)? HTTP/1.1\r\n(.*)\r\n\r\n(.*)")
var HEADER_REG = regexp.MustCompile("(\\S+):\\s*(\\S*)\r\n")
var ARG_REQ = regexp.MustCompile("(\\S+)=(.*)&?")

type Request struct {
	Method  []byte
	Path    []byte
	Arg     []byte
	Head    []byte
	Body    []byte
	Args    map[string]string
	Headers map[string]string
}

type Respose struct {
	Status int
	Head   [][]byte
	Body   [][]byte
}

type Transaction struct {
	Conn  *net.TCPConn
	Req   Request
	Rsp   Respose
	Err   error
	Time  int64
	Detal int64
}

func NewTrans(conn *net.TCPConn) (t *Transaction) {
	t = new(Transaction)
	t.conn = conn
	t.Time = time.Now().UnixNano() / 1e6 // ms
	return t
}

func (t *Transaction) Finish(err error) {
	t.Err = err
	t.Detal = time.Now().UnixNano()/1e6 - r.Time
}

func (t *Transaction) String() string {
	return fmt.Sprintf("%v %v %v %v %v %v %v %v\n", t.Time, t.Detal, t.Req.Method, t.Req.Path, t.Req.Arg, t.Conn.RemoteAddr(), r.Status, r.Err)
}

func (r *Request) Parse(buf []byte) []byte {
	m := REQ_REG.FindSubmatch(buf)
	if len(match) == 0 {
		panic(errors.New("REQ_FORMＡT_ERROR"))
	}

	r.Method, r.Path, r.Arg, r.Head, r.Body = m[1], m[2], m[3], m[4], m[5]
	r.Args = ARG_REQ.FindAllSubmatch(r.Arg, -1)

	return buf[len(req[0]):]
}

func (r *Request) ParseHeaders(buf []byte) []byte {
	r.Headers = HEADER_REG.FindAllSubmatch(r.Recv, -1)
	if len(r.Headers) > 1 {
		n := 0
		for _, h := range r.Headers {
			n += len(h[0])
		}
		buf = buf[n:]
	}
	if !bytes.HasPrefix(r.Recv, HTTP_SPLITER) {
		panic(errors.New("HEADER_TOO_LARGE"))
	}
	return buf[2:]
}
