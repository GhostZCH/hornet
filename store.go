package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
	"sync"
	"syscall"
	"time"
)

const MAGIC int64 = 6000576210161258312 //HORNETFS
const META_VERSION int64 = VERSION - VERSION%1000000
const DATA_FMT string = "%s/%016x.dat"
const META_FMT string = "%s/meta"

type diskItem struct {
	Dir   uint64
	ID    uint64
	Block int64
	Off   int64
	Size  int64
}

type metaHead struct {
	Magic   int64
	Version int64
	Blocks  int64
	Items   int64
}

type Item struct {
	Block   int64
	Off     int
	Size    int
	Putting bool
}

type Store struct {
	cap       int
	size      int
	blockSize int
	curOff    int
	curBlock  int64
	watermark float64
	path      string
	lock      sync.RWMutex
	blocks    map[int64][]byte
	meta      map[uint64]map[uint64]*Item
}

func openMmap(path string, size int) []byte {
	var f, fe = os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0600)
	if fe != nil {
		panic(fe)
	}

	f.Truncate(int64(size))
	defer f.Close()

	var flag = syscall.PROT_READ | syscall.PROT_WRITE
	var data, me = syscall.Mmap(int(f.Fd()), 0, size, flag, syscall.MAP_SHARED)
	if me != nil {
		panic(me)
	}

	Lwarn("mmap ", path, size)
	return data
}

func (s *Store) clear() {
	for float64(s.size) > s.watermark*float64(s.cap) {
		var min int64 = 0
		for id, _ := range s.blocks {
			if id < min || min == 0 {
				min = id
			}
		}

		if min == 0 {
			continue
		}

		path := fmt.Sprintf(DATA_FMT, s.path, min)
		size := len(s.blocks[min])
		s.size -= size

		Lwarn("delete block ", path, size, s.size)

		if err := os.Remove(path); err != nil {
			panic(err)
		}

		go func(b []byte) {
			// wait for request which use buf finish
			timeout := time.Duration(GConfig["sock.req.timeout"].(int))
			time.Sleep(time.Second*timeout + 1)
			syscall.Munmap(b)
		}(s.blocks[min])

		for _, dmap := range s.meta {
			for id, item := range dmap {
				if item.Block == min {
					delete(dmap, id)
				}
			}
		}
		delete(s.blocks, min)
	}
}

func (s *Store) addBlock(size int) {
	s.curBlock = time.Now().UnixNano()
	s.curOff = 0
	var name = fmt.Sprintf(DATA_FMT, s.path, s.curBlock)
	s.blocks[s.curBlock] = openMmap(name, size)
	s.size += size
	s.clear()
}

func (s *Store) Init() {
	s.path = GConfig["store.path"].(string)
	s.watermark = GConfig["store.watermark"].(float64)
	s.cap = GConfig["store.cap"].(int)
	s.blockSize = GConfig["store.blocksize"].(int)

	s.size = 0
	s.curOff = s.blockSize
	s.blocks = map[int64][]byte{}
	s.meta = make(map[uint64]map[uint64]*Item)

	var mpath = fmt.Sprintf(META_FMT, s.path)
	var mfile, err = os.Open(mpath)
	if err != nil {
		if os.IsNotExist(err) {
			return
		}
		panic(err)
	}

	var h metaHead
	if err := binary.Read(mfile, binary.LittleEndian, &h); err != nil {
		panic(err)
	}

	defer mfile.Close()
	defer os.Remove(mpath)

	if h.Magic != MAGIC || h.Version != META_VERSION {
		return
	}

	blocks := make([]int64, h.Blocks)
	binary.Read(mfile, binary.LittleEndian, blocks)

	for _, b := range blocks {
		dpath := fmt.Sprintf(DATA_FMT, s.path, b)
		if st, err := os.Stat(dpath); err != nil {
			Lerror(err)
		} else {
			s.blocks[b] = openMmap(dpath, int(st.Size()))
			s.size += int(st.Size())
		}
	}
	s.clear()

	var items = make([]diskItem, h.Items)
	binary.Read(mfile, binary.LittleEndian, items)
	for _, ditem := range items {
		var ok bool
		if _, ok = s.meta[ditem.Dir]; !ok {
			s.meta[ditem.Dir] = make(map[uint64]*Item)
		}

		if _, ok = s.blocks[ditem.Block]; ok {
			item := &Item{ditem.Block, int(ditem.Off), int(ditem.Size), false}
			s.meta[ditem.Dir][ditem.ID] = item
		}
	}
}

func (s *Store) Close() {
	var buf = new(bytes.Buffer)
	var h = metaHead{Magic: MAGIC, Version: META_VERSION}

	h.Blocks = int64(len(s.blocks))
	for _, item := range s.meta {
		h.Items += int64(len(item))
	}
	AssertSuccess(binary.Write(buf, binary.LittleEndian, h))

	for id, _ := range s.blocks {
		AssertSuccess(binary.Write(buf, binary.LittleEndian, id))
	}

	for d, items := range s.meta {
		for id, item := range items {
			ditem := diskItem{d, id, item.Block, int64(item.Off), int64(item.Size)}
			AssertSuccess(binary.Write(buf, binary.LittleEndian, ditem))
		}
	}

	mpath := fmt.Sprintf(META_FMT, s.path)
	mpath_tmp := mpath + ".tmp"

	mfile, err := os.OpenFile(mpath_tmp, os.O_RDWR|os.O_CREATE, 0600)
	AssertSuccess(err)

	defer mfile.Close()

	d := buf.Bytes()
	if n, err := mfile.Write(d); n == len(d) && err == nil {
		if os.Rename(mpath_tmp, mpath) == nil {
			Lwarn("store closed successed")
		}
	}
}

func (s *Store) Add(dir, id uint64, size int) (data []byte, item *Item) {
	s.lock.Lock()
	defer s.lock.Unlock()

	if size > s.blockSize {
		s.addBlock(size)
	}

	if size+s.curOff > s.blockSize {
		s.addBlock(s.blockSize)
	}

	data = s.blocks[s.curBlock][s.curOff : s.curOff+size]

	if _, ok := s.meta[dir]; !ok {
		s.meta[dir] = make(map[uint64]*Item)
	}

	s.meta[dir][id] = &Item{s.curBlock, s.curOff, size, true}
	s.curOff += size

	return data, s.meta[dir][id]
}

func (s *Store) Get(dir, id uint64) (*Item, []byte) {
	s.lock.RLock()
	defer s.lock.RUnlock()

	if dir, ok := s.meta[dir]; ok {
		if item, ok := dir[id]; ok && !item.Putting {
			return item, s.blocks[item.Block][item.Off : item.Off+item.Size]
		}
	}

	return nil, nil
}

func (s *Store) Del(dir, id uint64) {
	s.lock.Lock()
	defer s.lock.Unlock()

	if dir == 0 {
		s.meta = make(map[uint64]map[uint64]*Item)
		return
	}

	if id == 0 {
		if _, ok := s.meta[dir]; ok {
			delete(s.meta, dir)
		}
		return
	}

	if dirMap, ok := s.meta[dir]; ok {
		if _, ok := dirMap[id]; ok {
			delete(dirMap, id)
		}
	}
}
