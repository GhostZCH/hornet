package proxy

import (
	"net"
	"sync"
	"time"
)

type BackEnd struct {
	name string
	addr *net.TCPAddr
	last int
	free chan *net.TCPConn
}

func (sn *BackEnd) Hash(i int) uint32 {
	return 0
}

type Balancer struct {
	msg    []byte
	span   time.Duration
	fault  time.Duration
	svrs   map[string]*SvrNode
	sender *net.UDPConn
	recver *net.UDPConn
	crcle  *ConstHash
	lock   sync.RWMutex
}

func NewBalancer() *Balancer {
	b := new(Balancer)

	b.msg = []byte(GConfig["server.listen"].(string))
	b.span = time.Duration(GConfig["balancer.span_ms"].(int)) * time.Millisecond
	b.fault = time.Duration(GConfig["balancer.fault_ms"].(int)) * time.Millisecond

	addr, e := net.ResolveUDPAddr("udp", GConfig["balancer.addr"].(string))
	Success(e)

	var err error
	b.recver, err = net.ListenMulticastUDP("udp", nil, addr)
	Success(err)

	b.sender, err = net.DialUDP("udp", nil, addr)
	Success(err)

	return b
}

func (b *Balancer) send() {
	for {
		b.sender.Write(b.msg)
		time.Sleep(b.span)
	}
}

func (b *Balancer) recv() {
	data := make([]byte, 1024)

	for {
		n, _, err := b.recver.ReadFromUDP(data)
		Success(err)

		now := time.Now().Nanosecond() / 1e6
		name := string(data[:n])

		b.lock.Lock()
		if s, ok := b.svrs[name]; !ok {
			addr, e := net.ResolveTCPAddr("tcp", name)
			Success(e)
			b.svrs[name] = &SvrNode{name, addr, 0, make(chan *net.TCPConn, 32)}
		}

		b.svrs[string(data[:n])].last = time.Now().Nanosecond()
		b.lock.Unlock()
	}
}

func (b *Balancer) update() {
	var svrs []Node

	fault := GConfig["balancer.fault_ms"].(int)

	for {
		time.Sleep(b.fault)

		svrs = svrs[:0]
		now := time.Now().Nanosecond() / 1e6

		b.lock.Lock()

		for n, sn := range b.svrs {
			if now-sn.last > fault {
				delete(b.svrs, n)
			} else {
				svrs = append(svrs, sn)
			}
		}

		b.crcle = NewConstHash(32, svrs)
		b.lock.Unlock()
	}
}

func (b *Balancer) proxy_conn(c *net.TCPConn) {
	defer c.Close()

	for {
		//TODO get id
		if b.svrs == nil || len(b.svrs) == 0 {
			return
		}

		sn := b.crcle.Get(0).(*SvrNode)

		var err error
		var uc *net.TCPConn
		if len(sn.free) == 0 {
			uc, err = net.DialTCP("tcp", nil, sn.addr)
			Success(err)
		} else {
			uc := <-sn.free
		}

		//copy
	}
}

func (b *Balancer) proxy() {
	time.Sleep(b.fault)
	b.update()

	//TODO

	// listen

	// for{
	// 	   go proxy_conn
	// }
}

func (b *Balancer) Start() {
	mode := GConfig["balancer.mode"].(string)

	if mode == "client" || mode == "proxy" {
		go b.send()

		if mode == "proxy" {
			go b.recv()
			go b.update()
			go b.proxy()
		}
	}
}
