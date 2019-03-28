package main

import (
	"fmt"
	"hash/crc32"
	"net"
	"sync"
	"time"
)

type BackEnd struct {
	Name string
	Addr *net.TCPAddr
	Last time.Time
	Free chan *net.TCPConn
}

type ProxyHandler struct {
	fault     time.Duration
	backEnds  map[string]*BackEnd
	heartBeat *net.UDPConn
	crcle     *ConstHash
	lock      sync.RWMutex
}

func (be *BackEnd) Hash(i int) uint32 {
	name := []byte(fmt.Sprintf("%s#%d", be.Name, i))
	return crc32.ChecksumIEEE(name)
}

func NewProxyHandler() *ProxyHandler {
	h := new(ProxyHandler)

	h.fault = time.Duration(GConfig["proxy.fault_ms"].(int)) * time.Millisecond

	addr, err := net.ResolveUDPAddr("udp", GConfig["balancer.addr"].(string))
	Success(err)

	h.heartBeat, err = net.ListenMulticastUDP("udp", nil, addr)
	Success(err)

	return h
}

func (h *ProxyHandler) GetCtx() interface{} {
	return make([]byte, GConfig["common.http.header.maxlen"].(int))
}

func (h *ProxyHandler) GetListener() string {
	return GConfig["proxy.addr"].(string)
}

func (ph *ProxyHandler) Handle(trans *Transaction) {
	var err error
	buf := trans.Ctx.([]byte)
	conn := trans.Conn

	n, err := conn.Read(buf)
	Success(err)

	trans.Req.ParseBasic(buf[:n])

	ph.lock.RLock()
	// TODO get key, for del
	// key := hex.Decode(trans.Req.Path)
	back := ph.crcle.Get(0).(*BackEnd)

	ph.lock.RUnlock()
	var upstream *net.TCPConn
	if len(back.Free) == 0 {
		upstream, err = net.DialTCP("tcp", nil, back.Addr)
		Success(err)
	} else {
		back.Free <- upstream
	}

	defer func() {
		if err := recover(); err != nil {
			upstream.Close()
			panic(err)
		}
		if len(back.Free) > cap(back.Free) {
			upstream.Close()
		} else {
			back.Free <- upstream
		}
	}()

	Success(upstream.Write(buf[:n]))
	Success(tcp_copy(upstream, conn, buf))
	Success(tcp_copy(conn, upstream, buf))
}

func (h *ProxyHandler) Start() {
	go h.recv()
	go h.update()
}

func (h *ProxyHandler) Close() {
	h.heartBeat.Close()
}

func (h *ProxyHandler) recv() {
	data := make([]byte, 1024)

	for {
		n, _, err := h.heartBeat.ReadFromUDP(data)
		Success(err)

		name := string(data[:n])

		h.lock.Lock()
		if bk, ok := h.backEnds[name]; !ok {
			addr, e := net.ResolveTCPAddr("tcp", name)
			Success(e)
			h.backEnds[name] = &BackEnd{name, addr, time.Now(), make(chan *net.TCPConn, 32)}
		} else {
			bk.Last = time.Now()
		}

		h.lock.Unlock()
	}
}

func (h *ProxyHandler) update() {
	var svrs []Node

	for {
		time.Sleep(h.fault)

		svrs = svrs[:0]
		now := time.Now()

		h.lock.Lock()

		for n, sn := range h.backEnds {
			if now.Sub(sn.Last) > h.fault {
				delete(h.backEnds, n)
			} else {
				svrs = append(svrs, sn)
			}
		}

		h.crcle = NewConstHash(32, svrs)
		h.lock.Unlock()
	}
}

func tcp_copy(des, src *net.TCPConn, buf []byte) error {
	if n, err := src.Read(buf); err != nil {
		return err
	} else {
		if _, err = des.Write(buf[:n]); err != nil {
			return err
		}
	}
	return nil
}
