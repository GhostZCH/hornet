package main

import (
	"errors"
	"regexp"
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
		panic(errors.New("REQ_FORMＡT_ERROR"))
	}

	r.Method, r.Path, r.Arg, r.Head, r.Body = m[1], m[2], m[3], m[4], m[5]
}

func (r *Request) ParseHeaders() {
	headers := HEADER_REG.FindAllSubmatch(r.Head, -1)
	for _, h := range headers {
		r.Headers[string(h[0])] = h
	}
}

func (r *Request) ParseArgs() {
	args := ARG_REQ.FindAllSubmatch(r.Head, -1)
	for _, a := range args {
		r.Headers[string(a[0])] = a
	}
}
