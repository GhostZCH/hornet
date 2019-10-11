package main

import (
	"crypto/md5"
	"encoding/binary"
	"encoding/gob"
	"errors"
	"os"
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
	path    string
	Buckets [BUCKET_LIMIT]Bucket
	Blocks  []int64
}

func NewMeta(dir string) *Meta {
	m := &Meta{path: dir + "/meta"}
	for i := 0; i < BUCKET_LIMIT; i++ {
		m.Buckets[i].Items = make(map[Key]*Item)
		m.Buckets[i].putting = make(map[Key]*Item)
	}

	f, err := os.Open(s.mpath)
	if err != nil {
		if !os.IsNotExist(err) {
			panic(err)
		}
		Lwarn(s.name, " no meta file found")
		return
	}

	defer f.Close()
	defer os.Remove(s.mpath)

	Success(gob.NewDecoder(r).Decode(m))

	return m
}

func (m *Meta) GetBlocks() (blocks []int64) {
	return m.Blocks
}

func (m *Meta) Dump(blocks []int64) {
	m.Blocks = blocks

	tmp := m.path + ".tmp"
	f, err := os.OpenFile(tmp, os.O_RDWR|os.O_CREATE, 0600)
	Success(err)

	Success(gob.NewEncoder(w).Encode(m))
	f.Close()

	Success(os.Rename(tmp, m.path))
}

func (m *Meta) Get(k Key) (item *Item) {
	b := m.getBucket(k.ID)
	b.lock.RLock()
	defer b.lock.RUnlock()

	if i, ok := b.Items[k]; ok {
		return i
	}

	return nil
}

func (m *Meta) Alloc(k Key) (item *Item) {
	b := m.getBucket(k.ID)
	b.lock.Lock()
	defer b.lock.Unlock()

	if i, ok := b.Items[k]; ok {
		return nil
	}

	if i, ok := b.putting[k]; ok {
		return nil
	}

	b.putting[k] = new(Item)
	return b.putting[k]
}

func (m *Meta) Add(k Key) {
	b := m.getBucket(k.ID)
	b.lock.Lock()
	defer b.lock.Unlock()

	if _, ok := b.putting[k]; !ok {
		panic(errors.New("NO_PUTTING_ITEMS"))
	}

	if _, ok := b.Items[k]; !ok {
		b.Items[k] = b.putting[k]
	}
	delete(b.putting, k)
}

func (m *Meta) DeleteBatch(match func(*Item) bool) uint {
	n := uint(0)
	for i := 0; i < BUCKET_LIMIT; i++ {
		func() {
			b := &m.Buckets[i]
			b.lock.Lock()
			defer b.lock.Unlock()
			for id, item := range b.Items {
				if match(item) {
					delete(b.Items, id)
					n++
				}
			}
		}()
	}
	return n
}

func (m *Meta) getBucket(id Hash) *Bucket {
	k := binary.BigEndian.Uint32(id[:4]) % uint32(BUCKET_LIMIT)
	return &m.Buckets[k]
}
