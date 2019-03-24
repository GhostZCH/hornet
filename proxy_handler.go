package main

import (
	"net"
)

type ProxySvr struct {
}

func proxy(c *net.TCPConn) {
	defer c.Close()
	buf := make([]byte, 32*1024)

	u, e := net.Dial("tcp", "www.badu.com:80")
	if e != nil {
		fmt.Println(e)
		return
	}
	defer u.Close()

	for {
		if err := sock_copy(u.(*net.TCPConn), c, buf); err != nil {
			fmt.Println(err)
			return
		}

		if err := sock_copy(c, u.(*net.TCPConn), buf); err != nil {
			fmt.Println(err)
			return
		}
	}
}

func main() {
	lst, err := net.Listen("tcp", "127.0.0.1:8080")
	if err != nil {
		fmt.Println(err)
		return
	}

	for {
		if conn, ce := lst.(*net.TCPListener).AcceptTCP(); ce != nil {
			fmt.Println(ce)
		} else {
			go proxy(conn)
		}
	}
}
