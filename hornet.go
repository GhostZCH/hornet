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
	// path, mode := parseArgs()
	// dir, name := filepath.Split(path)
	// LoadConf(path, dir+"local_"+name)
	// InitLog()

	// defer func() {
	// 	if err := recover().(error); err != nil {
	// 		Log.Warn(string(debug.Stack()))
	// 		time.Sleep(time.Duration(1) * time.Second)
	// 	}
	// }()
}
