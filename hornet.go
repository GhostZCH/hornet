package main

import (
	"errors"
	"flag"
	"path/filepath"
	"runtime/debug"
)

func parseArgs() (string, string) {
	path := flag.String("conf", "hornet.yaml", "conf file path")
	mode := flag.String("mode", "cache", "start mode cache or proxy")

	flag.Parse()
	return *path, *mode
}

func main() {
	path, mode := parseArgs()

	dir, name := filepath.Split(path)
	LoadConf(path, dir+"local_"+name)
	InitLog()

	Lwarn(GConfig)

	defer func() {
		if err := recover(); err != nil {
			Lerror(err)
			Lwarn(string(debug.Stack()))
		}
	}()

	var h Handler

	switch mode {
	case "cache":
		h = NewCacheHandler()
	case "proxy":
		h = NewProxyHandler()
	default:
		panic(errors.New("unknown mode:" + mode))
	}

	svr := NewServer(h)

	go HandleSignal(svr)

	svr.Start()
}
