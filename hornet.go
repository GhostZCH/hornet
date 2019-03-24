package main

import (
	"flag"
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

	LoadConf(path, "local_"+path)
	InitLog()

	Lwarn(GConfig)

	defer func() {
		if err := recover(); err != nil {
			Lerror(err)
		}
	}()

	var svr Server

	switch mode {
	case:"cache"
		svr = NewServer(NewStore())
	case: "proxy"
		svr = NewProxy()
	} 
	go handleSignal(svr)

	svr.Start()
}
