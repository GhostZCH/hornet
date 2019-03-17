package main

import (
	"flag"
	"os"
	"os/signal"
	"syscall"
)

const VERSION int64 = 1000000

func parseArgs() string {
	path := flag.String("conf", "hornet.yaml", "conf file path")
	flag.Parse()
	return *path
}

func handleSignal(svr *Server) {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGUSR2, syscall.SIGINT, syscall.SIGTERM)

	for {
		sig := <-sigs
		Lwarn("get signal ", sig)

		switch sig {
		case syscall.SIGTERM:
			fallthrough
		case syscall.SIGINT:
			svr.Stop()
			break
		case syscall.SIGUSR2:
			InitLog()
		}
	}
}

func main() {
	path := parseArgs()

	LoadConf(path, "local_"+path)
	InitLog()

	Lwarn(GConfig)

	defer func() {
		if err := recover(); err != nil {
			Lerror(err)
		}
	}()

	s := NewStore()
	svr := NewServer(s)
	go handleSignal(svr)

	svr.Start()
	s.Close()
}
