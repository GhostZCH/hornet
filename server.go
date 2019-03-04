package main

import (
	"io"
	"net"
	"time"
)

type Handler interface {
	Handle(t *Transaction) bool
}

type Server struct {
	run      bool
	listen   *net.TCPListener
	handlers []Handler
}

func NewServer(handlers []Handler) (s *Server) {
	s = new(Server)
	s.run = true
	s.handlers = handlers
	return s
}

func (s *Server) Forever() {
	listen, err := net.Listen("tcp", GConfig["listen"].(string))
	Success(err)

	s.listen = listen.(*net.TCPListener)

	Lwarn("server start handle requests")
	for s.run {
		conn, err := s.listen.AcceptTCP()
		if err != nil {
			Lerror(err)
			continue
		}
		go s.handleConn(conn)
	}
}

func (s *Server) Stop() {
	Lwarn("server stoping ...")
	s.run = false
	s.listen.Close()
}

func (s *Server) handlerReq(conn *net.TCPConn) (err error) {
	t := NewTrans(conn)

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

	for _, h := range s.handlers {
		// return false if not continue next handler
		// raise panic if close conn
		if !h.Handle(t) {
			break
		}
	}

	setTimeOut(conn, GConfig["sock.idle.timeout"].(int))
	return nil
}

func (s *Server) handleConn(conn *net.TCPConn) {
	defer conn.Close()

	for s.run {
		if err := s.handlerReq(conn); err != nil {
			return
		}
	}
}

func setTimeOut(conn *net.TCPConn, seconds int) {
	timeout := time.Duration(seconds) * time.Second
	deadline := time.Now().Add(timeout)
	Success(conn.SetDeadline(deadline))
}
