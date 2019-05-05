package main

import (
	"regexp"
)

const (
	VERSION       int64  = 1000000
	MAGIC         int64  = 6000576210161258312 //HORNETFS
	META_VERSION  int64  = VERSION - VERSION%1000000
	FILE_NAME_FMT string = "%s/%016x.dat"
	KEY_HASH_LEN  int    = 16
	RAW_KEY_LIMIT int    = 128
	BUCKET_LIMIT  int    = 256
	RANGE_BLOCK   int    = 256 * 1204
	REQ_FORMAT    string = "%s /%s HTTP/1.1\r\n"
	RSP_FORMAT    string = "HTTP/1.1 %s\r\nServer: Hornet\r\nConnection: keep-alive\r\nContent-Length: %d\r\n"
)

var HTTP_END []byte = []byte("\r\n")
var HTTP_SEMICOLON = []byte(": ")
var RSP_MAP = map[int]string{200: "200 OK", 201: "201 Created", 404: "404 Not Found"}

var HEADER_REG = regexp.MustCompile("(\\S+):\\s*(\\S*)\r\n")
var RSP_REG = regexp.MustCompile("HTTP/1.1 (\\d+) \\w+\r\n([\\S\\s]*)\r\n([\\S\\s]*)")
var REQ_REG = regexp.MustCompile("^(GET|POST|DELETE) /([a-fA-F0-9]{32})? HTTP/1.1\r\n([\\S\\s]*)\r\n([\\S\\s]*)")
