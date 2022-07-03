package main

import (
	"encoding/binary"
	"log"

	"github.com/boltdb/bolt"
)

const path string = "/tmp/ldb"

func BytesToInt(bys []byte) uint64 {
	return binary.LittleEndian.Uint64(bys)
}

func IntToByte(i uint64) []byte {
	buf := make([]byte, 8)
	binary.LittleEndian.PutUint64(buf, i)
	return buf
}

func main() {
	// x := make(map[int64][]byte)
	db, err := bolt.Open("/tmp/my.db", 0600, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
}
