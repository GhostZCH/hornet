package store

import (
	"bytes"
	"encoding/gob"
)

type Item struct {
	Key        Key
	Block      int64
	Offset     int64
	HeaderLen  uint64
	BodyLen    uint64
	UserGroup  uint64
	User       uint64
	RootDomain uint64
	Domain     uint64
	SrcGroup   uint64
	Expires    int64
	Path       []byte
	Tags       int64
}

func ItemEncode(item *Item) []byte {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(item); err != nil {
		panic(err)
	}
	return buf.Bytes()
}

func ItemDecode(data []byte) *Item {
	item := Item{}
	buf := bytes.NewBuffer(data)
	dec := gob.NewDecoder(buf)
	if err := dec.Decode(&item); err != nil {
		panic(err)
	}
	return &item
}
