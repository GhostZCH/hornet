package main

import (
	"io"
	"net"
	"sync"
	"time"
)

type CacheHandler struct {
	run       bool
	store     *Store
	listener  *net.TCPListener
	heartbeat *net.UDPConn
}

func NewCacheSvr(store *Store) (s *CacheSvr) {
	s = new(Server)
	s.run = true
	s.store = store
	return s
}

func (s *CacheSvr) Start() {
	var err error
	var laddr *net.TCPAddr

	laddr, err = net.ResolveTCPAddr("tcp", GConfig["server.listen"].(string))
	Success(err)

	s.listener, err = net.ListenTCP("tcp")
	Success(err)

	Lwarn("server start handle requests")
	wg := new(sync.WaitGroup)
	for s.run {
		conn, err := s.listen.AcceptTCP()
		if err != nil {
			Lerror(err)
			continue
		}
		go s.handleConn(conn, wg)
	}
	wg.Wait()
}

func (s *CacheSvr) Stop() {
	Lwarn("server stoping ...")
	s.run = false
	s.listen.Close()
}

func (s *CacheSvr) handlerReq(conn *net.TCPConn, recv []byte) (err error) {
	t := NewTrans(conn, s.store)

	defer func() {
		if e := recover(); e != nil {
			err = e.(error)
		}
		if err != io.EOF {
			t.Finish(err)
			Laccess(t)
		}
	}()

	setTimeOut(conn, GConfig["sock.req.timeout"].(int))

	t.Handle(recv)

	setTimeOut(conn, GConfig["sock.idle.timeout"].(int))
	return nil
}

func (ch *CacheHandler) GetConnCtx(conn *net.TCPConn) interface{} {
	return make([]byte, GConfig["http.header.maxlen"].(int))
}

func (ch *CacheHandler) handlerReq(conn *net.TCPConn, ctx interface{}) (err error) {
	t := NewTrans(conn, s.store)

	recv := ctx.([]byte)

	defer func() {
		if e := recover(); e != nil {
			err = e.(error)
		}
		if err != io.EOF {
			t.Finish(err)
			Laccess(t)
			SetTimeOut(conn, GConfig["sock.idle.timeout"].(int))
		}
	}()

	SetTimeOut(conn, GConfig["sock.req.timeout"].(int))

	t.Handle(recv)

	return nil
}
