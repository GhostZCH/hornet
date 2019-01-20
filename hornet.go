package main

import (
	"net"
    "log"
    "runtime"
)

var cache = Store{}

func Handle(conn net.Conn) {
    defer conn.Close()

    for {
        recv := make([]byte, 4096)
		n, err := conn.Read(recv)
        if err != nil {
			log.Println(err)
            return
        }

        r := Req{}
        r.Parse(recv[:n])
		conn.Write(cache.Get(r.Dir, r.ID))
    }
}


func main() {
    runtime.GOMAXPROCS(8)
    cache.Init()

    listen, err := net.Listen("tcp", ":8080")
    if err != nil {
        log.Println("listen error: ", err)
        return
    }

    for {
        conn, err := listen.Accept()
        if err != nil {
            log.Println("accept error: ", err)
            break
        }
        go Handle(conn)
    }
}
