package main

import (
	"io"
	"net"
	"sync"
	"time"
)

type Handler interface {
	Start()
	GetListener() string
	GetConnCtx(conn *net.TCPConn) interface{}
	Handle(conn *net.TCPConn)
	Stop()
}

type Server struct {
	run      bool
	listener *net.TCPListener
	handler  *Handler
}

func NewServer(h *Handler) (svr *Server) {
	svr = new(Server)
	svr.run = true
	svr.handler = h
	return svr
}

func (svr *Server) Start() {
	laddr, rerr = net.ResolveTCPAddr("tcp", svr.GetListener())
	Success(err)

	listener, lerr := net.ListenTCP("tcp")
	Success(err)

	Lwarn("server start handle requests")
	wg := new(sync.WaitGroup)
	for s.run {
		conn, err := listener.AcceptTCP()
		if err != nil {
			Lerror(err)
			break
		}
		go s.handleConn(conn, wg)
	}
	wg.Wait()
}

func (svr *Server) Stop() {
	Lwarn("server stoping ...")
	svr.run = false
	svr.listener.Close()
}

func (svr *Server) handleConn(conn *net.TCPConn, wg *sync.WaitGroup) {
	wg.Add(1)

	defer conn.Close()
	defer wg.Done()

	ctx := svr.handler.GetConnCtx(conn)

	for ch.run {
		if err := svr.handler.Handle(conn, ctx); err != nil {
			return
		}
	}
}
