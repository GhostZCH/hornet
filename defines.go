package main

import (
	"regexp"
)

const (
	VERSION      int64  = 1000000
	META_VERSION int64  = VERSION - VERSION%1000000
	BUCKET_LIMIT int    = 1024
	RANGE_LIMIT  int    = 4 * 1024 * 1024 // snallest block size and when bigger than it we do split
	REQ_FORMAT   string = "%s /%s HTTP/1.1\r\n"
	RSP_FORMAT   string = "HTTP/1.1 %s\r\nServer: Hornet\r\nContent-Length: %d\r\n"
)

var HTTP_END []byte = []byte("\r\n")
var HTTP_SEMICOLON = []byte(": ")
var RSP_MAP = map[int]string{200: "200 OK", 201: "201 Created", 206: "206 Partial Content"}

var REQ_RANGE_REG = regexp.MustCompile("bytes=(\\d+)?-(\\d+)?")
var RSP_RANGE_REG = regexp.MustCompile("bytes (\\d+)-(\\d+)/(\\d+)")
var HEADER_REG = regexp.MustCompile("(\\S+):\\s*(\\S*)\r\n")
var RSP_REG = regexp.MustCompile("HTTP/1.1 (\\d+) \\w+\r\n([\\S\\s]*)\r\n([\\S\\s]*)")
var REQ_REG = regexp.MustCompile("^(GET|POST|DELETE) (.*) HTTP/1.1\r\n([\\S\\s]*)\r\n([\\S\\s]*)")
