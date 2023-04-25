package main

import (
	"hornet/common"
	"hornet/http"
	"hornet/store"
)

func main() {

	confPath := common.ParseArgs()
	c := common.LoadConf(confPath)
	s := store.NewStore(&c.Cache)

	access := common.NewHourlyLogger("test/log/")

	cacheSvr := http.NewCacheServer(&c, s, access)
	cacheSvr.Start()
}
