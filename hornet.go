package main

import (
	"errors"
	"flag"
	"go.uber.org/zap"
	"path/filepath"
	"runtime/debug"
	"time"
)

const VERSION int64 = 1000000

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
	Log.Warn("Conf", zap.Any("conf", Conf))

	defer func() {
		if err := recover().(error); err != nil {
			Log.Warn(string(debug.Stack()))
			Log.Error("main", zap.NamedError("err", err))
			time.Sleep(time.Duration(1) * time.Second)
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
