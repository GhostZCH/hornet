package main

import (
	"io"
	"net"
	"time"
)

type Server struct {
	run    bool
	store  *Store
	listen *net.TCPListener
}

func NewServer(store *Store) (s *Server) {
	s = new(Server)
	s.run = true
	s.store = store
	return s
}

func (s *Server) Forever() {
	listen, err := net.Listen("tcp", GConfig["server.listen"].(string))
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
	// TODO wait for all request finished
}

func (s *Server) Stop() {
	Lwarn("server stoping ...")
	s.run = false
	s.listen.Close()
}

func (s *Server) handlerReq(conn *net.TCPConn, recv []byte) (err error) {
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

func (s *Server) handleConn(conn *net.TCPConn) {
	defer conn.Close()
	recv := make([]byte, GConfig["http.header.maxlen"].(int))

	for s.run {
		if err := s.handlerReq(conn, recv); err != nil {
			return
		}
	}
}

func setTimeOut(conn *net.TCPConn, seconds int) {
	timeout := time.Duration(seconds) * time.Second
	deadline := time.Now().Add(timeout)
	Success(conn.SetDeadline(deadline))
}
