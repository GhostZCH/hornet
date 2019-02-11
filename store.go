package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
	"sync"
	"syscall"
)

const MAGIC int64 = 6000576210161258312 //HORNETFS
const META_VERSION int64 = VERSION - VERSION%1000000
const DATA_FMT string = "%s/%032x.dat"
const META_FMT string = "%s/meta"

type diskItem struct {
	Dir   uint64
	ID    uint64
	Block uint64
	Off   int64
	Size  int64
}

type diskBlock struct {
	ID   uint64
	Size int64
}

type metaHead struct {
	Magic   int64
	Version int64
	Blocks  int64
	Items   int64
}

type Item struct {
	Block   uint64
	Off     int
	Size    int
	Putting bool
}

type Store struct {
	blockSize  int
	blockCount int
	curOff     int
	curBlock   uint64
	path       string
	lock       sync.RWMutex
	blocks     map[uint64][]byte
	meta       map[uint64]map[uint64]*Item
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

	return data
}

func (s *Store) clear() {
	for len(s.blocks) > s.blockCount {
		var min uint64 = 0
		for id, _ := range s.blocks {
			if id < min || min == 0 {
				min = id
			}
		}

		if min == 0 {
			continue
		}

		if err := os.Remove(fmt.Sprintf(DATA_FMT, s.path, min)); err != nil {
			panic(err)
		}
		syscall.Munmap(s.blocks[min])
		delete(s.blocks, min)

		for _, dmap := range s.meta {
			for id, item := range dmap {
				if item.Block == min {
					delete(dmap, id)
				}
			}
		}
	}
}

func (s *Store) Init() {
	s.path = GConfig["store.path"].(string)
	s.curOff = GConfig["store.block.size"].(int)
	s.curBlock = 0
	s.blockCount = GConfig["store.block.count"].(int)
	s.blockSize = GConfig["store.block.size"].(int)
	s.blocks = map[uint64][]byte{}
	s.meta = make(map[uint64]map[uint64]*Item)

	var meta = fmt.Sprintf(META_FMT, s.path)
	var f, e = os.Open(meta)
	if e != nil {
		return
	}

	var h metaHead
	err := binary.Read(f, binary.LittleEndian, &h)
	if err != nil {
		panic(err)
	}

	if h.Magic != MAGIC || h.Version != META_VERSION {
		return
	}

	var blocks = make([]diskBlock, h.Blocks)
	binary.Read(f, binary.LittleEndian, blocks)
	for _, b := range blocks {
		if b.ID > s.curBlock {
			s.curBlock = b.ID
		}

		var name = fmt.Sprintf(DATA_FMT, s.path, b.ID)
		s.blocks[b.ID] = openMmap(name, int(b.Size))
	}
	s.clear()
	s.curBlock += 1

	var items = make([]diskItem, h.Items)
	binary.Read(f, binary.LittleEndian, items)
	for _, ditem := range items {
		var ok bool
		if _, ok = s.meta[ditem.Dir]; !ok {
			s.meta[ditem.Dir] = make(map[uint64]*Item)
		}

		if _, ok = s.blocks[ditem.Block]; ok {
			item := Item{ditem.Block, int(ditem.Off), int(ditem.Size), false}
			s.meta[ditem.Dir][ditem.ID] = &item
		}
	}

	f.Close()
	os.Remove(meta)
}

func (s *Store) Close() {
	var buf = new(bytes.Buffer)

	var h = metaHead{Magic: MAGIC, Version: META_VERSION}
	h.Blocks = int64(len(s.blocks))
	for _, item := range s.meta {
		h.Items += int64(len(item))
	}
	if e := binary.Write(buf, binary.LittleEndian, h); e != nil {
		panic(e)
	}

	for id, blk := range s.blocks {
		var dBlock = diskBlock{id, int64(len(blk))}
		if e := binary.Write(buf, binary.LittleEndian, dBlock); e != nil {
			panic(e)
		}
	}

	for d, items := range s.meta {
		for id, item := range items {
			var ditem = diskItem{id, d, item.Block, int64(item.Off), int64(item.Size)}
			if e := binary.Write(buf, binary.LittleEndian, ditem); e != nil {
				panic(e)
			}
		}
	}

	var meta = fmt.Sprintf(META_FMT, s.path)
	var f, e = os.OpenFile(meta, os.O_RDWR|os.O_CREATE, 0600)
	if e != nil {
		panic(e)
	}
	defer f.Close()
	var t = buf.Bytes()
	f.Write(t)
}

func (s *Store) Add(dir, id uint64, size int, force bool) (data []byte, item *Item) {
	if data := s.Get(dir, id); data != nil {
		if !force {
			s.Del(dir, id)
		} else {
			return nil, nil
		}
	}

	s.lock.Lock()
	defer s.lock.Unlock()

	if size+s.curOff > s.blockSize {
		s.curBlock += 1
		s.curOff = 0
		var name = fmt.Sprintf(DATA_FMT, s.path, s.curBlock)
		s.blocks[s.curBlock] = openMmap(name, s.blockSize)
		s.clear()
	}

	data = s.blocks[s.curBlock][s.curOff : s.curOff+size]

	if _, ok := s.meta[dir]; !ok {
		s.meta[dir] = make(map[uint64]*Item)
	}

	s.meta[dir][id] = &Item{s.curBlock, s.curOff, size, true}
	s.curOff += size

	return data, s.meta[dir][id]
}

func (s *Store) Get(dir, id uint64) []byte {
	s.lock.RLock()
	defer s.lock.RUnlock()

	if dir, ok := s.meta[dir]; ok {
		if item, ok := dir[id]; ok && !item.Putting {
			if block, ok := s.blocks[item.Block]; ok {
				return block[item.Off : item.Off+item.Size]
			}
		}
	}

	return nil
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
