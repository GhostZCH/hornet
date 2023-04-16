package main

import (
	"fmt"
	"hornet/store"
	"time"

	lru "github.com/hashicorp/golang-lru/v2"
)

func LRUtest() {
	cache, err := lru.NewARC[*store.Key, *store.Item](100000)
	if err != nil {
		fmt.Println(err)
		return
	}

	n := 1000000
	startTime := time.Now()
	for i := 0; i < n; i++ {
		k := &store.Key{H1: uint64(i), H2: uint64(n*10 - i)}
		v := &store.Item{}
		cache.Add(k, v)
	}
	// 输出耗时
	duration := time.Since(startTime)
	fmt.Printf("Insert %d rows took %s\n", n, duration)

}

func main() {
	// confPath := common.ParseArgs()
	// conf := common.LoadConf(confPath)
	// store := store.NewStore(&conf.Cache)

	// logger := common.NewHourlyLogger("test/log/")

	// cacheSvr := http.NewCacheServer(&conf, store, logger)
	// cacheSvr.Start()
	LRUtest()
}
