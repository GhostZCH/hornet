package store

import (
	"fmt"
	"hornet/common"
)

type Key struct {
	H1 int64
	H2 int64
}

func GetKey(key []byte) Key {
	h1, h2 := common.Hash128(key)
	return Key{h1, h2}
}

func (k Key) String() string {
	return fmt.Sprintf("%016x%016x", k.H1, k.H2)
}
