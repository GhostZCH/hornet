package main

import (
	"fmt"
	"net"
	"time"
)

type Transaction struct {
	Conn      *net.TCPConn
	Ctx       interface{}
	Req       InRequest
	Rsp       OutRespose
	Time      time.Time
	Err       error
	ClientMsg string
	SvrMsg    string
}

func NewTrans(c *net.TCPConn, ctx interface{}) *Transaction {
	t := new(Transaction)
	t.Conn = c
	t.Ctx = ctx
	t.Time = time.Now()
	return t
}

func (t *Transaction) String() string {
	return fmt.Sprintf("%d\t%d\t%d\t%s\t%s\t%s\t%d\t%s\t%s\t%v\n",
		VERSION, t.Time.Unix(), time.Since(t.Time),
		t.Req.Method, t.Req.Path, t.Conn.RemoteAddr(),
		t.Rsp.Status, t.ClientMsg, t.SvrMsg, t.Err)
}
