package main

import (
	"bytes"
	"encoding/binary"
	"sync"
)

type HKey [KEY_HASH_LEN]byte

type Key struct {
	Hash  HKey
	Range [2]uint32
}

type ItemInfo struct {
	ID         Key
	Grp        HKey
	Block      int64
	Off        int64
	Expire     int64
	EtagHash   uint32
	ExpireHash uint32
	BitMap     uint64
	BodyLen    int64
	HeadLen    int64
	RawKeyLen  uint32
	RawKey     [RAW_KEY_LIMIT]byte
	Tags       [TAG_LIMIT]uint16
}

type Item struct {
	Putting bool
	Info    *ItemInfo
}

type bucket struct {
	items map[Key]*Item
	lock  sync.RWMutex
}

type Meta struct {
	buckets [BUCKET_LIMIT]bucket
}

func NewMeta() *Meta {
	m := new(Meta)
	for i := 0; i < BUCKET_LIMIT; i++ {
		m.buckets[i].items = make(map[Key]*Item)
	}
	return m
}

func (m *Meta) AddAll(infos []ItemInfo) {
	for i := 0; i < BUCKET_LIMIT; i++ {
		b := &m.buckets[i]
		b.lock.Lock()
		defer b.lock.Unlock()
	}

	for _, i := range infos {
		b := m.getBucket(i.ID)
		b.items[i.ID] = &Item{false, &i}
	}
}

func (m *Meta) DumpAll(buf *bytes.Buffer) int {
	n := 0
	for i := 0; i < BUCKET_LIMIT; i++ {
		b := &m.buckets[i]
		b.lock.Lock()
		defer b.lock.Unlock()

		n += len(b.items)
		for _, b := range b.items {
			Success(binary.Write(buf, binary.LittleEndian, *b.Info))
		}
	}

	return n
}

func (m *Meta) Get(id Key) *Item {
	b := m.getBucket(id)
	b.lock.RLock()
	defer b.lock.RUnlock()
	if item, ok := b.items[id]; ok {
		return item
	}
	return nil
}

func (m *Meta) Add(item *Item) {
	b := m.getBucket(item.Info.ID)
	b.lock.Lock()
	defer b.lock.Unlock()

	b.items[item.Info.ID] = item
}

func (m *Meta) Delete(id Key) {
	b := m.getBucket(id)
	b.lock.RLock()
	defer b.lock.RUnlock()
	delete(b.items, id)
}

func (m *Meta) DeleteBatch(match func(*Item) bool) {
	for i := 0; i < BUCKET_LIMIT; i++ {
		func() {
			b := &m.buckets[i]
			b.lock.Lock()
			defer b.lock.Unlock()
			for id, item := range b.items {
				if match(item) {
					delete(b.items, id)
				}
			}
		}()
	}
}

func (m *Meta) getBucket(id Key) *bucket {
	idx := int(id.Hash[0]) % BUCKET_LIMIT
	return &m.buckets[idx]
}
