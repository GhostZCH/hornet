package main

import (
	"hornet/common"
	"hornet/http"
	"hornet/store"
)

func main() {

	confPath := common.ParseArgs()
	conf := common.LoadConf(confPath)
	store := store.NewStore(&conf.Cache)

	logger := common.NewHourlyLogger("test/log/")

	cacheSvr := http.NewCacheServer(&conf, store, logger)
	cacheSvr.Start()
}
