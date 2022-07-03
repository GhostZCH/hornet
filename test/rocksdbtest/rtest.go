package main

import "github.com/leeyazhou/gorocksdb"

func main() {
	options := gorocksdb.NewDefaultOptions()
	options.SetCreateIfMissing(true)
}
