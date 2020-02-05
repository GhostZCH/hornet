package main

import (
	"go.uber.org/zap"
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

	ls, err := net.ListenTCP("tcp", addr)
	Success(err)

	svr.listener = ls

	Log.Warn("server start handle requests")
	svr.handler.Start()

	wg := new(sync.WaitGroup)
	for svr.run {
		conn, err := ls.AcceptTCP()
		if err != nil {
			Log.Error("AcceptTCP", zap.NamedError("err", err))
			break
		}

		go svr.handleConn(conn, wg)
	}
	wg.Wait()
	svr.handler.Close()
}

func (svr *Server) Stop() {
	Log.Warn("server stoping ...")
	svr.run = false
	svr.listener.Close()
}

func (svr *Server) handleTrans(trans *Transaction) (err error) {
	defer func() {
		if e, ok := recover().(error); ok {
			err = e
		}

		if err != io.EOF {
			trans.Err = err
			Access.Info(trans.String())
			if err == nil {
				SetTimeOut(trans.Conn, Conf["common.sock.idle.timeout"].(int))
			}
		}
	}()

	SetTimeOut(trans.Conn, Conf["common.sock.req.timeout"].(int))
	svr.handler.Handle(trans)
	return err
}

func (svr *Server) handleConn(conn *net.TCPConn, wg *sync.WaitGroup) {
	wg.Add(1)

	defer func() {
		if err := recover().(error); err != nil {
			Log.Warn("handleConn", zap.NamedError("err", err))
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
