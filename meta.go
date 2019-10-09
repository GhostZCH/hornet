package main

import (
	"crypto/md5"
	"encoding/binary"
	"encoding/gob"
	"errors"
	"io"
	"sync"
)

type Hash [md5.Size]byte

type Key struct {
	ID    Hash
	Range uint32 // index of range block
}

type Item struct {
	Key       Key
	Group     Hash
	Block     int64
	Off       int64
	RangeSize uint32 // RangeSize = 0 means cache all
	Expire    int64
	BodyLen   int64
	HeadLen   int64
	EtagCRC   uint32
	Tag       int64
	RawKey    []byte
}

type Bucket struct {
	lock    sync.RWMutex
	putting map[Key]*Item
	Items   map[Key]*Item
	// Range   map[Hash]uint32 // TODO: delete ranges faster
	// DirTree // TODO: suport delete by dir
}

type Meta struct {
	Buckets [BUCKET_LIMIT]Bucket
}

func NewMeta() *Meta {
	m := new(Meta)
	for i := 0; i < BUCKET_LIMIT; i++ {
		m.Buckets[i].Items = make(map[Key]*Item)
		m.Buckets[i].putting = make(map[Key]*Item)
	}
	return m
}

func (m *Meta) Load(r io.Reader) {
	// TODO: use other serialize method when gob not fast enough
	Success(gob.NewDecoder(r).Decode(m))
}

func (m *Meta) Dump(w io.Writer) {
	Success(gob.NewEncoder(w).Encode(m))
}

func (m *Meta) Get(k Key) (item *Item, puttingItem *Item) {
	b := m.getBucket(k.ID)
	b.lock.RLock()
	defer b.lock.RUnlock()

	if i, ok := b.Items[k]; ok {
		return i, nil
	}

	if ｐ, ok := b.putting[k]; ok {
		return nil, nil
	}

	b.putting[k] = new(Item)
	return nil, b.putting[k]
}

func (m *Meta) Add(k Key) {
	b := m.getBucket(k.ID)
	b.lock.Lock()
	defer b.lock.Unlock()

	if _, ok := b.putting[k]; !ok {
		panic(errors.New("NO_PUTTING_ITEMS"))
	}

	defer delete(b.putting, k)

	if _, ok := b.Items[k]; !ok {
		b.Items[k] = b.putting[k]
	}
}

// func (m *Meta) Delete(id Hash) {
// 	b := m.getBucket(id)
// 	b.lock.RLock()
// 	defer b.lock.RUnlock()
// 	delete(b.Items, id)
// }

func (m *Meta) DeleteBatch(match func(*Item) bool) {
	for i := 0; i < BUCKET_LIMIT; i++ {
		func() {
			b := &m.Buckets[i]
			b.lock.Lock()
			defer b.lock.Unlock()
			for id, item := range b.Items {
				if match(item) {
					delete(b.Items, id)
				}
			}
		}()
	}
}

func (m *Meta) getBucket(id Hash) *Bucket {
	k := binary.BigEndian.Uint32(id[:4]) % uint32(BUCKET_LIMIT)
	return &m.Buckets[k]
}
