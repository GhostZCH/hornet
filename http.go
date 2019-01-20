package main

import (
    "strconv"
	"regexp"
)

const (
	M_GET = 1
	M_PUT = 2
	M_DEL = 3
	M_CLR = 4
)


var rsp404 = []byte("HTTP/1.1 404 NOT FOUND\r\nConnection: keep-alive\r\nContent-Length: 0\r\n\r\n")

var methods = map[byte] int {'G': M_GET, 'P': M_PUT, 'D': M_DEL}
var reg = regexp.MustCompile("^(GET|PUT|DEL|CLR) /v1/([0-9]+)/([0-9]+) HTTP")

type Req struct {
	Method int
	Dir uint64
	ID uint64
}

func (r *Req) Parse(recv []byte) {
	match := reg.FindSubmatch(recv)
	r.Method, _ = methods[match[1][0]]
	r.Dir, _ = strconv.ParseUint(string(match[2]), 10, 64)
	r.ID, _ = strconv.ParseUint(string(match[3]), 10, 64)
}

