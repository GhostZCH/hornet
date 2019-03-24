package main

import (
	"regexp"
)

const (
	VERSION       int64  = 1000000
	MAGIC         int64  = 6000576210161258312 //HORNETFS
	META_VERSION  int64  = VERSION - VERSION%1000000
	DATA_FMT      string = "%s/%016x.dat"
	META_FMT      string = "%s/meta"
	KEY_HASH_LEN  int    = 16
	RAW_KEY_LIMIT int    = 128
	RANGE_BLOCK   int    = 256 * 1024
	BUCKET_LIMIT  int    = 256
	TAG_LIMIT     int    = 4
	HTTP_END      []byte = []byte("\r\n")
	REQ_FORMAT    string = "%s /%s%s HTTP/1.1\r\n"
	RSP_FORMAT    string = "HTTP/1.1 %s\r\nServer: Hornet\r\nConnection: keep-alive\r\nContent-Length: %d\r\n"
)

var HEADER_REG = regexp.MustCompile("(\\S+):\\s*(\\S*)\r\n")

var RSP_MAP = map[int]string{200: " 200 OK", 201: " 201 Created", 404: "404 Not Found"}
var RSP_REG = regexp.MustCompile("HTTP/1.1 (\\d) \\w\r\n(\\s\\S+)\r\n(\\s\\S*)")
var REQ_REG = regexp.MustCompile("^(GET|PUT|DEL) /([a-fA-F0-9]{32})?(\\?\\S+)? HTTP/1.1\r\n([\\S\\s]*)\r\n([\\S\\s]*)")
var ARG_REQ = regexp.MustCompile("(\\S+)=(.*)&?")
