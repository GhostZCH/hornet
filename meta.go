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
	ID         Hash
	RangeIndex uint32
}

// 加一个辅助数据记录range和目录结构
type Item struct {
	ID        Hash
	Grp       Hash
	RangeSize uint32 // RangeSize = 0 means cache all
	Expire    int64
	BodyLen   int64
	HeadLen   int64
	EtagCRC   uint32
	Tag       int64
	Type      int64
	RawKey    []byte
}

type Bucket struct {
	lock    sync.RWMutex
	Items   map[Hash]*Item
	Range   map[Hash]uint32
	putting map[Key]*Item
}

type Meta struct {
	Buckets [BUCKET_LIMIT]Bucket
}

func GetHash(data []byte) Hash {
	return md5.Sum(data)
}

func NewMeta() *Meta {
	m := new(Meta)
	for i := 0; i < BUCKET_LIMIT; i++ {
		m.Buckets[i].Items = make(map[Hash]*Item)
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
	// b := m.getBucket(k.ID)
	// b.lock.RLock()
	// defer b.lock.RUnlock()

	// if i, ok := b.Items[k.ID]; ok && i.Key.Ranges Ranges&k.Ranges != 0) {
	//     return i, nil
	// }

	// if ｐ, ok := b.putting[k]; ok {
	//     return nil, nil
	// }

	// b.putting[k] = new(Item)
	// return nil, b.putting[k]
}

func (m *Meta) Add(k Key) {
	b := m.getBucket(k.ID)
	b.lock.Lock()
	defer b.lock.Unlock()

	if _, ok := b.putting[k]; !ok {
		panic(errors.New("NO_PUTTING_ITEMS"))
	}

	defer delete(b.putting, k)

	if _, ok := b.Items[k.ID]; !ok {
		b.Items[k.ID] = b.putting[k]
	} else {
		b.Items[k.ID].Ranges &= k.Ranges
	}
}

func (m *Meta) Delete(id Hash) {
	b := m.getBucket(id)
	b.lock.RLock()
	defer b.lock.RUnlock()
	delete(b.Items, id)
}

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
