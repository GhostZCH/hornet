package main

import (
	"encoding/gob"
	"io"
	"sync"
)

type Key [KEY_HASH_LEN]byte

type Item struct {
	ID         Key
	Grp        Key
	Putting    bool
	CacheAll   bool
	RBSize     uint32 // range block size
	RBMap      []byte // range index bitmap
	Block      int64
	Off        int64
	Expire     int64
	BodyLen    int64
	HeadLen    int64
	EtagHash   uint32
	ExpireHash uint32
	BitMap     uint32
	RawKey     []byte
}

type Bucket struct {
	Items map[Key]*Item
	lock  sync.RWMutex
}

type Meta struct {
	Buckets [BUCKET_LIMIT]Bucket
}

func NewMeta() *Meta {
	m := new(Meta)
	for i := 0; i < BUCKET_LIMIT; i++ {
		m.Buckets[i].Items = make(map[Key]*Item)
	}
	return m
}

func (m *Meta) Load(r io.Reader) {
	// TODO: use binary when not fast enough
	d := gob.NewDecoder(r)
	Success(d.Decode(m))
}

func (m *Meta) Dump(w io.Writer) {
	// TODO: remove puting and expires
	e := gob.NewEncoder(w)
	Success(e.Encode(m))
}

func (m *Meta) Get(id Key) *Item {
	b := m.getBucket(id)
	b.lock.RLock()
	defer b.lock.RUnlock()
	if item, ok := b.Items[id]; ok {
		return item
	}
	return nil
}

func (m *Meta) Add(item *Item) {
	b := m.getBucket(item.ID)
	b.lock.Lock()
	defer b.lock.Unlock()

	b.Items[item.ID] = item
}

func (m *Meta) Delete(id Key) {
	b := m.getBucket(id)
	b.lock.RLock()
	defer b.lock.RUnlock()
	delete(b.Items, id)
}

func (m *Meta) DeleteBatch(match func(*Item) bool) {
	for i := 0; i < BUCKET_LIMIT; i++ {
		func() {
			b := &m.buckets[i]
			b.lock.Lock()
			defer b.lock.Unlock()
			for id, item := range b.Items {
				if match(item) {
					delete(b.items, id)
				}
			}
		}()
	}
}

func (m *Meta) getBucket(id Key) *Bucket {
	return &m.Buckets[int(id[0])%BUCKET_LIMIT]
}
