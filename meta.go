package main

import (
	"crypto/md5"
	"encoding/gob"
	"errors"
	"fmt"
	"go.uber.org/zap"
	"os"
	"sync"
)

const META_VERSION int64 = VERSION - VERSION%1000000
const RAW_LIMIT int = 256

type Hash [md5.Size]byte

type Key struct {
	ID    Hash
	Range uint32 // index of range block
}

type Item struct {
	Key      Key
	Block    int64
	Off      int64
	Expire   int64
	HeadLen  uint64
	BodyLen  uint64
	TotalLen uint64
	Tag      int64
	TypeCRC  uint64 //crc64
	EtagCRC  uint64 //crc64
	GroupCRC uint64 //crc64
	RawKey   []byte
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
	Buckets [BucketLimit]Bucket
	Blocks  []int64
}

func NewMeta(dir string) *Meta {
	m := &Meta{path: fmt.Sprintf("%s/%d-%d.meta", dir, META_VERSION, RangeSize)}
	for i := 0; i < BucketLimit; i++ {
		m.Buckets[i].Items = make(map[Key]*Item)
		m.Buckets[i].putting = make(map[Key]*Item)
	}

	f, err := os.Open(m.path)
	if err != nil {
		if !os.IsNotExist(err) {
			panic(err)
		}
		Log.Warn("meta file found", zap.String("path", m.path))
		return m
	}
	defer f.Close()

	Success(gob.NewDecoder(f).Decode(m))
	os.Remove(m.path)

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

	Success(gob.NewEncoder(f).Encode(m))
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

func (m *Meta) Alloc(item *Item) bool {
	b := m.getBucket(item.Key.ID)
	b.lock.Lock()
	defer b.lock.Unlock()

	if _, ok := b.Items[item.Key]; ok {
		return false
	}

	if _, ok := b.putting[item.Key]; ok {
		return false
	}

	b.putting[item.Key] = item
	return true
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

	if len(b.Items[k].RawKey) > RAW_LIMIT {
		b.Items[k].RawKey = b.Items[k].RawKey[:RAW_LIMIT]
	}
}

func (m *Meta) DeleteBatch(match func(*Item) bool) uint {
	n := uint(0)
	for i := 0; i < BucketLimit; i++ {
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

func (m *Meta) DeletePut(k Key) {
	b := m.getBucket(k.ID)
	b.lock.Lock()
	defer b.lock.Unlock()
	delete(b.putting, k)
}

func (m *Meta) getBucket(id Hash) *Bucket {
	return &m.Buckets[id[0]]
}
