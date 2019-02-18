package main

import (
	"bytes"
	"errors"
	"io"
	"net"
	"strconv"
	"time"
)

type Server struct {
	run    bool
	store  *Store
	listen *net.TCPListener
}

func (s *Server) Init(store *Store) {
	s.run = true
	s.store = store
}

func (s *Server) Forever() {
	listen, err := net.Listen("tcp", GConfig["listen"].(string))
	AssertSuccess(err)

	s.listen = listen.(*net.TCPListener)

	Lwarn("server start handle requests")
	for s.run {
		conn, err := s.listen.AcceptTCP()
		if err != nil {
			Lerror(err)
			continue
		}
		Linfo(conn)
		go s.handleConn(conn)
	}
}

func (s *Server) Stop() {
	Lwarn("server stoping ...")
	s.run = false
	s.listen.Close()
}

func (s *Server) handlerReq(conn *net.TCPConn, recv []byte, err error) (ok bool) {
	r := NewRequest(conn)

	defer func() {
		var err error
		if e := recover(); e != nil {
			err = e.(error)
		}
		r.Finish(err)
		Laccess(r)
		ok = err == nil
	}()

	AssertSuccess(err)

	recv = r.ParseReqLine(recv)

	switch r.Method {
	case "GET":
		if item, rsp := s.store.Get(r.Dir, r.ID); rsp == nil || item.Putting {
			conn.Write(SPC_RSP_404)
		} else {
			conn.Write(rsp)
		}

	case "DEL":
		s.store.Del(r.Dir, r.ID)
		conn.Write(SPC_RSP_200)

	case "POST":
		if item, _ := s.store.Get(r.Dir, r.ID); item != nil {
			conn.Write(SPC_RSP_200)
			return true
		}

		if recv = r.ParseHeaders(recv); r.Err != nil {
			return false
		}

		for _, h := range GConfig["http.header.discard"].([]interface{}) {
			r.DelHeader([]byte(h.(string)))
		}

		var headerBuf bytes.Buffer
		headerBuf.Write(CACHE_RSP_HEADER)
		r.GenerateHeader(&headerBuf)
		headerBuf.Write(HTTP_SPLITER)

		clh := r.FindHeader([]byte("Content-Length"))
		if clh == nil {
			panic(errors.New("NO_CONTENT_LENGTH"))
		}

		cl, err := strconv.ParseInt(string(clh[2]), 10, 64)
		AssertSuccess(err)

		h := headerBuf.Bytes()
		l := int(cl) + len(h)
		buf, item := s.store.Add(r.Dir, r.ID, l)

		copy(buf, h)
		n := len(h)

		copy(buf[n:], recv)
		n += len(recv)

		recv := make([]byte, GConfig["http.body.bufsize"].(int))
		for n < l {
			if rn, err := conn.Read(recv); err != nil {
				s.store.Del(r.Dir, r.ID)
				panic(err)
			} else {
				copy(buf[n:], recv[:rn])
				n += rn
			}
		}
		item.Putting = false
		conn.Write(SPC_RSP_201)
	}

	return true
}

func (s *Server) handleConn(conn *net.TCPConn) {
	defer func() {
		if err := recover(); err != nil {
			Lerror(err)
		}
		conn.Close()
	}()

	recv := make([]byte, GConfig["http.header.maxlen"].(int))

	for s.run {
		timeout := time.Duration(GConfig["sock.idle.timeout"].(int))
		conn.SetDeadline(time.Now().Add(time.Second * timeout))

		if n, err := conn.Read(recv); err == io.EOF {
			return
		} else {
			timeout := time.Duration(GConfig["sock.req.timeout"].(int))
			conn.SetDeadline(time.Now().Add(time.Second * timeout))
			if !s.handlerReq(conn, recv[:n], err) {
				return
			}
		}
	}
}
