package main

import (
	"bytes"
	"errors"
	"github.com/ghostzch/asynclogger"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/yaml.v2"
	"hash/crc64"
	"io/ioutil"
	"net"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"
)

var (
	Access *asynclogger.Logger
	Log    *asynclogger.Logger
	Conf   = make(map[string]interface{})
)

func Success(args ...interface{}) {
	if args[len(args)-1] != nil {
		panic(args[len(args)-1])
	}
}

func InitLog() {
	accessConf := &asynclogger.Conf{
		Path:      Conf["common.accesslog.path"].(string),
		MaxSize:   100 * 1024 * 1024,
		BufLimit:  Conf["common.accesslog.buf"].(int),
		QueueSize: 10,
		Level:     "info",
		ZapConf:   zapcore.EncoderConfig{}}
	Access = asynclogger.NewLogger(accessConf)

	conf := &asynclogger.Conf{
		Path:      Conf["common.log.path"].(string),
		MaxSize:   100 * 1024 * 1024,
		BufLimit:  0,
		QueueSize: 1,
		Level:     Conf["common.log.level"].(string),
		ZapConf: zapcore.EncoderConfig{
			MessageKey:   "msg",
			TimeKey:      "time",
			LevelKey:     "level",
			CallerKey:    "caller",
			EncodeTime:   zapcore.ISO8601TimeEncoder,
			EncodeCaller: zapcore.ShortCallerEncoder}}
	Log = asynclogger.NewLogger(conf)
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
	Conf = readYaml(path)
	lconf := readYaml(localPath)

	for k, v := range lconf {
		Conf[k] = v
	}
}

func SetTimeOut(conn *net.TCPConn, seconds int) {
	timeout := time.Duration(seconds) * time.Second
	Success(conn.SetDeadline(time.Now().Add(timeout)))
}

func HandleSignal(svr *Server) {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	for s := range sigs {
		Log.Warn("get signal", zap.String("sig", s.String()))
		svr.Stop()
		Access.Sync()
		Log.Sync()
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

func GenerateItem(isRange bool, k Key, path []byte, headers map[string][][]byte) (*Item, []byte) {
	item := &Item{Key: k, RawKey: path}

	if hdr, ok := headers["hornet-group"]; ok {
		item.GroupCRC = crc64.Checksum(hdr[2], nil)
	}

	if hdr, ok := headers["hornet-group"]; ok {
		item.GroupCRC = crc64.Checksum(hdr[2], nil)
	}

	if h, ok := headers["hornet-type"]; ok {
		item.TypeCRC = crc64.Checksum(h[2], nil)
	}

	if h, ok := headers["content-length"]; ok {
		if n, e := strconv.ParseInt(string(h[2]), 10, 64); e != nil {
			panic(e)
		} else {
			item.BodyLen = uint64(n)
		}
	} else {
		panic(errors.New("CONTENT_LEN_NOT_SET"))
	}

	if isRange {
		if h, ok := headers["content-range"]; ok {
			if n, e := strconv.ParseInt(string(h[2]), 10, 64); e != nil {
				panic(e)
			} else {
				item.BodyLen = uint64(n)
			}
		} else {
			panic(errors.New("CONTENT_LEN_NOT_SET"))
		}
	} else {
		item.TotalLen = item.BodyLen
	}

	if h, ok := headers["expire"]; ok {
		if d, e := ParseTime(h[2]); e == nil {
			item.Expire = d.Unix()
		} else {
			panic(e)
		}
	} else {
		item.Expire = time.Now().Unix() + 3600*12
		expireHeader := []byte("Expires: ")
		expire := FormatTime(expireHeader, time.Now().Add(time.Hour*12))
		headers["expire"] = [][]byte{expireHeader, []byte("Expires"), expire}
	}

	if h, ok := headers["hornet-tags"]; ok {
		if tag, err := strconv.ParseInt(string(h[2]), 10, 64); err != nil {
			panic(err)
		} else {
			item.Tag = tag
		}
	}

	for _, h := range Conf["cache.http.header.discard"].([]interface{}) {
		delete(headers, h.(string))
	}

	buf := new(bytes.Buffer)
	for k, v := range headers {
		if !strings.HasPrefix(k, "hornet") {
			buf.Write(v[0])
		}
	}

	head := buf.Bytes()

	item.HeadLen = uint64(len(head))

	return item, buf.Bytes()
}
