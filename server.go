package main

import (
	"io"
	"net"
	"sync"
)

type Handler interface {
	Start()
	GetListener() string
	GetCtx() interface{}
	Handle(trans *Transaction)
	Close()
}

type Server struct {
	run      bool
	listener *net.TCPListener
	handler  Handler
}

func NewServer(h Handler) (svr *Server) {
	svr = new(Server)
	svr.run = true
	svr.handler = h
	return svr
}

func (svr *Server) Start() {
	addr, err := net.ResolveTCPAddr("tcp", svr.handler.GetListener())
	Success(err)

	listener, err := net.ListenTCP("tcp", addr)
	Success(err)

	Lwarn("server start handle requests")
	svr.handler.Start()

	wg := new(sync.WaitGroup)
	for svr.run {
		conn, err := listener.AcceptTCP()
		if err != nil {
			Lerror(err)
			break
		}
		go svr.handleConn(conn, wg)
	}
	wg.Wait()
	svr.handler.Close()
}

func (svr *Server) Stop() {
	Lwarn("server stoping ...")
	svr.run = false
	svr.listener.Close()
}

func (svr *Server) handleTrans(trans *Transaction) (err error) {
	defer func() {
		err = recover().(error)
		if err != io.EOF {
			Laccess(trans)
			if err == nil {
				SetTimeOut(trans.Conn, GConfig["common.sock.idle.timeout"].(int))
			}
		}
	}()

	SetTimeOut(trans.Conn, GConfig["common.sock.req.timeout"].(int))
	svr.handler.Handle(trans)
	return err
}

func (svr *Server) handleConn(conn *net.TCPConn, wg *sync.WaitGroup) {
	wg.Add(1)

	defer func() {
		if err := recover(); err != nil {
			Lwarn(err)
		}
	}()

	defer conn.Close()
	defer wg.Done()

	var err error = nil
	ctx := svr.handler.GetCtx()
	for svr.run && err == nil {
		err = svr.handleTrans(NewTrans(conn, ctx))
	}
}
