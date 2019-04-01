package main

import (
	"fmt"
	"net"
	"time"
)

type Transaction struct {
	Conn      *net.TCPConn
	Ctx       interface{}
	Req       Request
	Rsp       Respose
	Time      time.Time
	Err       error
	ClientMsg string
	SvrMsg    string
}

func NewTrans(c *net.TCPConn, ctx interface{}) *Transaction {
	t := new(Transaction)
	t.Conn = c
	t.Time = time.Now()
	t.Req.Init()
	t.Rsp.Init()
	t.Ctx = ctx
	return t
}

func (t *Transaction) String() string {
	return fmt.Sprintf("%d %d %s %s %s %s %s %d [%s] [%s] [%v]\n",
		VERSION, t.Time.Unix(), time.Since(t.Time),
		t.Req.Method, t.Req.Path, t.Req.Arg, t.Conn.RemoteAddr(),
		t.Rsp.Status, t.ClientMsg, t.SvrMsg, t.Err)
}
