package main

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"regexp"
	"sync"
	"syscall"
	"time"
)

const (
	MAGIC         int64  = 6000576210161258312 //HORNETFS
	META_VERSION  int64  = VERSION - VERSION%1000000
	DATA_FMT      string = "%s/%08x-%016x.dat"
	META_FMT      string = "%s/meta"
	KEY_HASH_LEN  int    = 16
	RAW_KEY_LIMIT int    = 128
	RANGE_BLOCK   int    = 256 * 1024
	BUCKET        int    = 256
	TAG_LIMIT     int    = 4
)

type HKey [KEY_HASH_LEN]byte

type Key struct {
	Hash  HKey
	Range uint32
}

var EMPTY_KEY = Key{}

type CoreItem struct {
	ID        Key
	Grp       HKey
	Block     int64
	Off       int64
	Expire    uint32
	BodyLen   uint32
	HeadLen   uint32
	RawKeyLen uint16
	RawKey    [RAW_KEY_LIMIT]byte
	Tags      [TAG_LIMIT]uint16
}

type Item struct {
	Putting  bool
	Mem      []byte
	MemBlock int64
	Item     CoreItem
}

type bucket struct {
	items map[Key]*Item
	lock  sync.RWMutex
}

type sHeader struct {
	Magic   int64
	Version int64
	Blocks  int64
	Items   int64
}

type StoreGroup struct {
	cap       int
	bSize     int
	size      int
	curOff    int
	curBlock  int64
	waterMark float64
	blocks    map[int64][]byte
	lock      sync.RWMutex
	path      *string
}

type Store struct {
	disk    StoreGroup
	mem     StoreGroup
	lock    sync.RWMutex
	buckets [BUCKET]bucket
}

func NewStore() (s *Store) {
	s = new(Store)
	path := GConfig["store.path"].(string)

	s.mem.init(
		GConfig["store.mem.cap"].(int),
		GConfig["store.mem.blocksize"].(int),
		GConfig["store.mem.watermark"].(float32),
		nil)

	s.disk.init(
		GConfig["store.disk.cap"].(int),
		GConfig["store.disk.blocksize"].(int),
		GConfig["store.disk.watermark"].(float32),
		&path)

	for i := 0; i < BUCKET; i++ {
		s.buckets[i].items = make(map[Key]*Item)
	}

	le := binary.LittleEndian
	mpath := fmt.Sprintf(META_FMT, path)
	mfile, err := os.Open(mpath)
	if err != nil && !os.IsNotExist(err) {
		panic(err)
	}

	defer mfile.Close()
	defer os.Remove(mpath)

	var h sHeader
	Success(binary.Read(mfile, le, &h))

	if h.Magic != MAGIC || h.Version != META_VERSION {
		return
	}

	blocks := make([]int64, h.Blocks)
	binary.Read(mfile, le, blocks)

	for _, b := range blocks {
		dpath := fmt.Sprintf(DATA_FMT, s.disk.path, b)
		if st, err := os.Stat(dpath); err != nil {
			Lerror(err)
		} else {
			s.disk.blocks[b] = OpenMmap(dpath, int(st.Size()))
			s.disk.size += int(st.Size())
		}
	}
	s.clear(true)

	items := make([]CoreItem, h.Items)
	Success(binary.Read(mfile, le, items))

	for _, ci := range items {
		b := s.getBucket(ci.ID)
		b.items[ci.ID] = &Item{false, nil, 0, ci}
	}

	return s
}

func (s *Store) Close() {
	le := binary.LittleEndian

	buf := new(bytes.Buffer)
	h := sHeader{Magic: MAGIC, Version: META_VERSION}

	h.Blocks = int64(len(s.disk.blocks))
	for _, bucket := range s.buckets {
		h.Items += int64(len(bucket.items))
	}
	Success(binary.Write(buf, le, h))

	for id, _ := range s.disk.blocks {
		Success(binary.Write(buf, le, id))
	}

	for _, bucket := range s.buckets {
		for _, i := range bucket.items {
			Success(binary.Write(buf, le, i.Item))
		}
	}

	mpath := fmt.Sprintf(META_FMT, s.disk.path)
	mpath_tmp := mpath + ".tmp"

	mfile, err := os.OpenFile(mpath_tmp, os.O_RDWR|os.O_CREATE, 0600)
	Success(err)

	defer mfile.Close()

	d := buf.Bytes()
	if n, err := mfile.Write(d); n == len(d) && err == nil {
		if os.Rename(mpath_tmp, mpath) == nil {
			Lwarn("store closed successed")
		}
	}
}

func (s *Store) Add(id Key, size int) (data []byte, item *Item) {
	item = &Item{true, nil, 0, CoreItem{}}
	item.Item.ID = id
	disk := &s.disk

	b := s.getBucket(id)
	disk.lock.Lock()

	// todo funcion for sg return data, blk, off
	if size > disk.bSize {
		disk.addBlock(size) // single block for big data
	} else if size+disk.curOff > disk.bSize {
		disk.addBlock(disk.bSize)
	}

	item.Item.Block = disk.curBlock
	item.Item.Off = int64(disk.curOff)

	data = disk.blocks[disk.curBlock][disk.curOff : disk.curOff+size]
	disk.curOff += size
	disk.lock.Unlock()

	b.lock.Lock()
	b.items[id] = item
	b.lock.Unlock()

	return data, item
}

func (s *Store) Get(id Key) (*Item, []byte) {
	b := s.getBucket(id)
	b.lock.RLock()
	defer b.lock.RUnlock()

	if i, ok := b.items[id]; ok && !i.Putting {
		if _, ok = s.mem.blocks[i.MemBlock]; ok && i.Mem != nil {
			return i, i.Mem
		}
		end := i.Item.Off + int64(i.Item.HeadLen+i.Item.BodyLen)
		return i, s.disk.blocks[i.Item.Block][i.Item.Off:end]
	}

	return nil, nil
}

func (s *Store) Del(id Key) {
	b := s.getBucket(id)
	b.lock.Lock()
	defer b.lock.Unlock()
	delete(b.items, id)
}

func (s *Store) DelByGroup(g HKey) {
	for i := 0; i < BUCKET; i++ {
		b := &s.buckets[i]
		b.lock.Lock()
		for id, item := range b.items {
			if item.Item.Grp == g {
				delete(b.items, id)
			}
		}
		b.lock.Unlock()
	}
}

func (s *Store) DelByRawKey(reg *regexp.Regexp) {
	for i := 0; i < BUCKET; i++ {
		b := &s.buckets[i]
		b.lock.Lock()
		for id, item := range b.items {
			if reg.Match(item.Item.RawKey[:item.Item.RawKeyLen]) {
				delete(b.items, id)
			}
		}
		b.lock.Unlock()
	}
}

func (s *Store) clear(disk bool) {
	sg := &s.mem
	if disk {
		sg = &s.disk
	}

	for float64(sg.size) > s.conf.WaterMark*float64(sg.cap) {
		var min int64 = -1
		for id, _ := range sg.blocks {
			if id < min || min < 0 {
				min = id
			}
		}

		if min < 0 {
			panic(errors.New("Can not find block to remove"))
		}

		path := fmt.Sprintf(DATA_FMT, s.conf.Path, min)
		size := len(sg.blocks[min])
		sg.size -= size

		Lwarn("delete block ", path, size, sg.size)
		Success(os.Remove(path))

		go func(b []byte) {
			// wait for request which is using buf finish
			timeout := time.Duration(GConfig["sock.req.timeout"].(int))
			time.Sleep(time.Second*timeout + 1)
			syscall.Munmap(b)
		}(sg.blocks[min])

		for i := 0; i < BUCKET; i++ {
			b := &s.buckets[i]
			b.lock.Lock()
			for id, item := range b.items {
				if item.Item.Block == min {
					delete(b.items, id)
				}
			}
			b.lock.Unlock()
		}

		s.lock.Lock()
		delete(s.disk.blocks, min)
		s.lock.Unlock()
	}
}

func (s *Store) getBucket(id Key) *bucket {
	idx := int(id.Hash[0]) % BUCKET
	return &s.buckets[idx]
}

func (sg *StoreGroup) addBlock(size int) {
	sg.curBlock = time.Now().UnixNano()
	sg.curOff = 0
	var name = fmt.Sprintf(DATA_FMT, s.conf.Path, sg.curBlock)
	sg.blocks[sg.curBlock] = OpenMmap(name, size)
	sg.size += size
	sg.clear()
}

func (sg *StoreGroup) init(bSize, cap int, waterMark float32, path *string) {
	sg.cap = cap
	sg.size = 0
	sg.bSize = bSize
	sg.curOff = bSize
	sg.curBlock = 0
	sg.waterMark = waterMark
	sg.blocks = map[int64][]byte{}
	sg.path = path
}
