package main

import (
	"fmt"
	"net"
	"sync"
	"time"
)

type Balancer struct {
	addr    *net.UDPAddr
	cluster string
	key     []byte
	local   []byte
	span    time.Duration
	svrs    map[string]int
	sender  *net.UDPConn
	recver  *net.UDPConn
	lock    sync.Mutex
}

func NewBalancer() *Balancer {
	var err error

	b := new(Balancer)

	b.svrs = make(map[string]int)
	b.local = []byte(GConfig["server.listen"].(string))
	b.span = time.Duration(GConfig["balancer.span_ms"].(int)) * time.Millisecond

	b.addr, err = net.ResolveUDPAddr("udp", GConfig["balancer.addr"].(string))
	Success(err)

	b.recver, err = net.ListenMulticastUDP("udp", nil, b.addr)
	Success(err)

	b.sender, err = net.DialUDP("udp", nil, b.addr)
	Success(err)

	return b
}

func (b *Balancer) send() {
	conn, _ := net.DialUDP("udp", nil, b.addr)

	for {
		conn.Write(b.local)
		time.Sleep(b.span)
	}
}

func (b *Balancer) recv() {
	data := make([]byte, 1024)
	for {
		n, _, err := b.recver.ReadFromUDP(data)
		Success(err)

		b.lock.Lock()
		b.svrs[string(data[:n])] = time.Now().Nanosecond() / 1e6
		b.lock.Unlock()
		time.Sleep(b.span)
	}
}

func (b *Balancer) GetServers() (svrs []string) {
	now := time.Now().Nanosecond() / 1e6

	b.lock.Lock()
	b.lock.Unlock()

	for k, v := range b.svrs {
		if v-now > GConfig["balancer.fault_ms"].(int) {
			delete(b.svrs, k)
		} else {
			svrs = append(svrs, k)
		}
	}

	return svrs
}

func (b *Balancer) Start() {
	go b.send()
	go b.recv()
}
