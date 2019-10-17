package main

import (
	"bufio"
	"bytes"
	"errors"
	"hash/crc64"
	"io/ioutil"
	"log"
	"net"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"gopkg.in/yaml.v2"
)

type Logger struct {
	info         *log.Logger
	warn         *log.Logger
	err          *log.Logger
	access       chan string
	accessWriter *bufio.Writer
	accessFile   *os.File
	runFile      *os.File
}

var logger *Logger

var levelMap = map[string]int{"info": 1, "warn": 2, "error": 3}

var GConfig = make(map[string]interface{})

func Success(args ...interface{}) {
	err := args[len(args)-1]
	if args[len(args)-1] != nil {
		panic(err)
	}
}

func InitLog() {
	var err error

	if logger != nil {
		if logger.runFile != nil {
			logger.runFile.Close()
		}
		if logger.accessFile != nil {
			logger.accessWriter.Flush()
			logger.accessFile.Close()
		}
	}

	logger = new(Logger)

	// init run log
	flag := os.O_APPEND | os.O_CREATE | os.O_WRONLY

	path := GConfig["common.log.path"].(string)
	logger.runFile, err = os.OpenFile(path, flag, 0666)
	Success(err)

	var level = GConfig["common.log.level"].(string)
	var lv, ok = levelMap[level]
	if !ok {
		panic("level [" + level + "] not \"error\" \"warn\" or \"info\"")
	}

	if lv <= 3 {
		logger.err = log.New(logger.runFile, "[ERROR] ", log.LstdFlags)
	}

	if lv <= 2 {
		logger.warn = log.New(logger.runFile, "[WARN] ", log.LstdFlags)
	}

	if lv <= 1 {
		logger.info = log.New(logger.runFile, "[INFO] ", log.LstdFlags)
	}

	// init access log
	path = GConfig["common.accesslog.path"].(string)
	logger.accessFile, err = os.OpenFile(path, flag, 0666)
	Success(err)

	bufsize := GConfig["common.accesslog.buf"].(int)
	logger.access = make(chan string, 64)
	logger.accessWriter = bufio.NewWriterSize(logger.accessFile, bufsize)

	go func() {
		for {
			x := <-logger.access
			logger.accessWriter.WriteString(x)
		}
	}()
}

func Linfo(v ...interface{}) {
	if logger.info != nil {
		logger.info.Println(v...)
	}
}

func Lwarn(v ...interface{}) {
	if logger.warn != nil {
		logger.warn.Println(v...)
	}
}

func Lerror(v ...interface{}) {
	if logger.err != nil {
		logger.err.Println(v...)
	}
}

func Laccess(t *Transaction) {
	if logger.access != nil {
		logger.access <- t.String()
	}
}

func readYaml(path string) (conf map[string]interface{}) {
	if content, err := ioutil.ReadFile(path); err != nil {
		if os.IsNotExist(err) {
			return conf
		}
		panic(err)
	} else {
		Success(yaml.Unmarshal(content, &conf))
		return conf
	}
}

func LoadConf(path string, localPath string) {
	GConfig = readYaml(path)
	lconf := readYaml(localPath)

	for k, v := range lconf {
		GConfig[k] = v
	}
}

func SetTimeOut(conn *net.TCPConn, seconds int) {
	timeout := time.Duration(seconds) * time.Second
	deadline := time.Now().Add(timeout)
	Success(conn.SetDeadline(deadline))
}

func HandleSignal(svr *Server) {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGUSR2, syscall.SIGINT, syscall.SIGTERM)

	for {
		sig := <-sigs
		Lwarn("get signal ", sig)

		switch sig {
		case syscall.SIGTERM:
			fallthrough
		case syscall.SIGINT:
			svr.Stop()
			break
		case syscall.SIGUSR2:
			InitLog()
		}
	}
}

func SendHttp(conn *net.TCPConn, per []byte, headers [][]byte, bodys [][]byte) int {
	var err error
	sum, n := 0, 0

	conn.SetNoDelay(false)
	defer conn.SetNoDelay(true)

	bodyLen := 0
	for _, b := range bodys {
		bodyLen += len(b)
	}

	n, err = conn.Write(per)
	Success(err)
	sum += n

	for _, b := range headers {
		n, err = conn.Write(b)
		Success(err)
		sum += n
	}

	Success(conn.Write(HTTP_END))
	sum += len(HTTP_END)

	for _, b := range bodys {
		n, err = conn.Write(b)
		Success(err)
		sum += n
	}

	return sum
}

var GMT = []byte("GMT")

func FormatTime(dst []byte, date time.Time) []byte {
	dst = date.In(time.UTC).AppendFormat(dst, time.RFC1123)
	copy(dst[len(dst)-3:], GMT)
	return dst
}

func ParseTime(date []byte) (time.Time, error) {
	return time.Parse(time.RFC1123, string(date))
}

func GenerateItem(headers map[string][][]byte) (*Item, []byte) {
	// TODO
	item := &Item{}

	if hdr, ok := headers["hornet-group"]; ok {
		item.GroupCRC = crc64.Checksum(hdr[2], nil)
	}

	if h, ok := headers["content-length"]; ok {
		if n, e := strconv.Atoi(string(h[2])); e != nil {
			panic(e)
		} else {
			info.BodyLen = int64(n)
		}
	} else {
		panic(errors.New("CONTENT_LEN_NOT_SET"))
	}

	if h, ok := headers["hornet-raw-key"]; ok {
		info.RawKeyLen = uint32(len(h[2]))
		copy(item.Info.RawKey[:], h[2])
	}

	for _, h := range GConfig["cache.http.header.discard"].([]interface{}) {
		delete(headers, h.(string))
	}

	buf := new(bytes.Buffer)
	for k, v := range headers {
		if !strings.HasPrefix(k, "hornet") {
			buf.Write(v[0])
		}
	}

	head := buf.Bytes()

	info.HeadLen = int64(len(head))
	info.Expire = 999999999999999999

	//TODO  Expire
	//TODO  etag
	//TODO  tags

	return item, buf.Bytes()
}
