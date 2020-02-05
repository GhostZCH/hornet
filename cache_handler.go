package main

import (
	"bytes"
	"crypto/md5"
	"errors"
	"fmt"
	"go.uber.org/zap"
	"hash/crc64"
	"net"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	RegRangReg = regexp.MustCompile("bytes=(\\d+)?-(\\d+)?")
	RspRangReg = regexp.MustCompile("(\\d+)-(\\d+)/(\\d+)")
)

type LockKey struct {
	id      Key
	rbSize  uint32
	rbIndex uint32
}

type Upstream struct {
	keep int
	addr *net.TCPAddr
	free chan *net.TCPConn
}

type CacheHandler struct {
	name      string
	devices   *DeviceManager
	heartBeat *net.UDPConn
	upstream  *Upstream
	lock      sync.Mutex
	pullLock  map[LockKey]sync.Mutex
}

func NewCacheHandler() (h *CacheHandler) {
	addr, err := net.ResolveUDPAddr("udp", Conf["common.heartbeat.addr"].(string))
	Success(err)

	h = new(CacheHandler)
	h.name = Conf["common.name"].(string)
	h.devices = NewDeviceManager()
	h.heartBeat, err = net.DialUDP("udp", nil, addr)
	Success(err)

	keep := Conf["cache.upstream.keep"].(int)
	h.upstream = &Upstream{
		keep: keep,
		free: make(chan *net.TCPConn, keep)}
	h.upstream.addr, err = net.ResolveTCPAddr("tcp", Conf["cache.upstream.addr"].(string))
	Success(err)

	return h
}

func (h *CacheHandler) GetCtx() interface{} {
	return make([]byte, Conf["common.http.header.maxlen"].(int))
}

func (h *CacheHandler) GetListener() string {
	return Conf["cache.addr"].(string)
}

func (h *CacheHandler) Start() {
	go func() {
		msg := []byte(Conf["cache.addr"].(string))
		span := time.Duration(Conf["cache.heartbeat_ms"].(int)) * time.Millisecond
		for {
			if _, e := h.heartBeat.Write(msg); e != nil {
				Log.Warn("heartBeat.Write", zap.NamedError("err", e))
				break
			}
			time.Sleep(span)
		}
	}()
}

func (h *CacheHandler) Close() {
	Log.Warn("close", zap.String("handler", "cache"))
	h.heartBeat.Close()
	h.devices.Close()
}

func (h *CacheHandler) Handle(trans *Transaction) {
	buf := trans.Ctx.([]byte)

	n, e := trans.Conn.Read(buf)
	Success(e)

	trans.Req.Recv = buf[:n]
	trans.Req.Parse()
	if h, ok := trans.Req.Headers["hornet-log"]; ok {
		trans.ClientMsg = string(h[2])
	}

	switch trans.Req.Method[0] {
	case 'G':
		h.get(trans)
	case 'D':
		h.del(trans)
	}

	trans.Rsp.Send(trans.Conn)
}

func (h *CacheHandler) get(trans *Transaction) {
	id := md5.Sum(trans.Req.Path)

	isRange, start, end := false, int64(0), int64(-1)
	if rg, ok := trans.Req.Headers["range"]; ok {
		isRange = true
		start, end = parse_range(rg[2])
	}

	total := 0
	var header []byte = nil
	for r := start / RangeSize; end == -1 || r*RangeSize < end; r++ {
		k := Key{ID: id, Range: uint32(r)}
		item, data, cache := h.devices.Get(k)
		if item == nil {
			h.pull(isRange, trans, k)
			item, data, cache = h.devices.Get(k)
			cache = DeviceLimit
		}

		if item == nil {
			trans.Rsp.Status = 500
			trans.Rsp.Headers = nil
			trans.Rsp.Bodys = nil
			return
		}

		total = int(item.TotalLen)
		if header == nil {
			header = data[:item.HeadLen]
		}
		if end == -1 || end > int64(item.TotalLen) {
			end = int64(item.TotalLen)
		}

		trans.SvrMsg += fmt.Sprintf("%s:%d", h.name, cache)
		via := fmt.Sprintf("X-Via-Cache: %s %d\r\n", h.name, cache)
		trans.Rsp.Headers = append(trans.Rsp.Headers, []byte(via))

		s, e := start-r*int64(RangeSize), end-r*int64(RangeSize)
		if e > RangeSize {
			e = RangeSize
		}
		if s < 0 {
			s = 0
		}
		trans.Rsp.Bodys = append(trans.Rsp.Bodys, data[s+int64(item.HeadLen):e+int64(item.HeadLen)])
	}

	trans.Rsp.Status = 200
	if isRange {
		trans.Rsp.Status = 206
		rg := fmt.Sprintf("Content-Range: %d-%d/%d", start, end, total)
		trans.Rsp.Headers = append(trans.Rsp.Headers, []byte(rg))
	}
	trans.Rsp.Headers = append(trans.Rsp.Headers, header)
}

func (h *CacheHandler) del(trans *Transaction) {
	trans.Rsp.Status = 400
	group, ok := trans.Req.Headers["hornet-group"]
	if !ok {
		panic(errors.New("NO_GROUP"))
	}
	groupCRC := crc64.Checksum(group[2], nil)

	trans.Rsp.Status = 200
	// delete by group
	if bytes.Equal(trans.Req.Path, []byte("/all-items")) {
		h.devices.Del(func(item *Item) bool {
			return item.GroupCRC == groupCRC
		})
		return
	}

	// delete by id
	if !bytes.Equal(trans.Req.Path, []byte("/")) {
		id := md5.Sum(trans.Req.Path)
		h.devices.Del(func(item *Item) bool {
			return item.GroupCRC == groupCRC && item.Key.ID == id
		})
		return
	}

	// delete by type
	if hdr, ok := trans.Req.Headers["hornet-type"]; ok {
		t := crc64.Checksum(hdr[2], nil)
		h.devices.Del(func(item *Item) bool {
			return item.GroupCRC == groupCRC && item.TypeCRC == t
		})
		return
	}

	// delete by mask
	if hdr, ok := trans.Req.Headers["hornet-mask"]; ok {
		mask, e := strconv.ParseInt(string(hdr[2]), 10, 64)
		Success(e)
		h.devices.Del(func(item *Item) bool {
			return item.GroupCRC == groupCRC && item.Tag&mask != 0
		})

		return
	}

	// delete by regex
	if hdr, ok := trans.Req.Headers["hornet-regex"]; ok {
		reg := regexp.MustCompile(string(hdr[2]))
		h.devices.Del(func(item *Item) bool {
			return item.GroupCRC == groupCRC && reg.Match(item.RawKey)
		})
		return
	}

	panic(errors.New("NO_DEL_PARAMS"))
}

func (h *CacheHandler) pull(isRange bool, trans *Transaction, k Key) {
	var u *net.TCPConn

	if len(h.upstream.free) != 0 {
		u = <-h.upstream.free
	} else {
		var e error
		u, e = net.DialTCP("tcp", nil, h.upstream.addr)
		Success(e)
	}

	req := OutRequest{Method: trans.Req.Method, Path: trans.Req.Path}
	for _, v := range trans.Req.Headers {
		req.Headers = append(req.Headers, v[0])
	}

	s, e := int64(k.Range)*RangeSize, int64(k.Range)*RangeSize-1
	r := fmt.Sprintf("Range: bytes=%d-%d", s, e)
	req.Headers = append(req.Headers, []byte(r))

	req.Send(u)

	recv := make([]byte, Conf["common.http.header.maxlen"].(int))
	n, err := u.Read(recv)
	Success(err)

	rsp := InRespose{}
	rsp.Recv = recv[:n]
	rsp.Parse()

	item, head := generateItem(isRange, k, trans.Req.Path, rsp.Headers)
	data, dev := h.devices.Alloc(item)

	copy(data, head)
	data = data[len(head):]

	copy(data, rsp.Body)
	data = data[len(rsp.Body):]

	if n, e := u.Read(data); n != len(data) || e != nil {
		h.devices.DelPut(item.Key)
	}

	if len(h.upstream.free) < h.upstream.keep {
		h.upstream.free <- u
	}

	h.devices.Add(dev, item.Key)
}

func parse_range(r []byte) (start, end int64) {
	start, end = 0, -1

	m := RegRangReg.FindSubmatch(r)
	if len(m) != 3 || (len(m[1]) == 0 && len(m[2]) == 0) {
		panic(errors.New("RANGE_ERROR"))
	}

	if len(m[1]) != 0 {
		start, _ = strconv.ParseInt(string(m[1]), 10, 64)
	}

	if len(m[2]) != 0 {
		end, _ = strconv.ParseInt(string(m[2]), 10, 64)
		end++
	}

	if start >= end {
		panic(errors.New("RANGE_ERROR"))
	}

	return start, end
}

func generateItem(isRange bool, k Key, path []byte, headers map[string][][]byte) (*Item, []byte) {
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

	if !isRange {
		item.TotalLen = item.BodyLen
	} else {
		if h, ok := headers["content-range"]; !ok {
			panic(errors.New("CONTENT_RANGE_NOT_SET"))
		} else {
			if re := RspRangReg.FindSubmatch(h[2]); len(re) != 4 {
				panic(errors.New("CONTENT_RANGE_WRONG"))
			} else {
				if tmp, err := strconv.ParseInt(string(re[3]), 10, 64); err != nil {
					panic(errors.New("CONTENT_RANGE_WRONG"))
				} else {
					item.TotalLen = uint64(tmp)
				}
			}
		}
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
