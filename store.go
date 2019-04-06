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

type sHeader struct {
	Magic   int64
	Version int64
	Blocks  int64
	Items   int64
}

type Store struct {
	cap      int
	size     int
	bSize    int
	curOff   int
	curBlock int64
	name     string
	path     string
	mpath    string
	meta     *Meta
	lock     sync.RWMutex
	blocks   map[int64][]byte
}

func NewStore(name, mpath, path string, cap, bSize int) *Store {
	s := &Store{name: name, mpath: mpath, path: path,
		cap: cap, bSize: bSize, curOff: bSize,
		blocks: make(map[int64][]byte)}
	return s
}

func (s *Store) Add(item *Item) []byte {
	s.meta.Add(item)

	size := int(item.Info.HeadLen + item.Info.BodyLen)
	if size > s.bSize {
		s.addBlock(size) // single block for big data
	} else if size+s.curOff > s.bSize {
		s.addBlock(s.bSize)
	}

	item.Info.Block = s.curBlock
	item.Info.Off = int64(s.curOff)

	data := s.blocks[s.curBlock][s.curOff : s.curOff+size]
	s.curOff += size

	return data
}

func (s *Store) Get(id Key) (*Item, []byte) {
	item := s.meta.Get(id)
	if item == nil {
		return nil, nil
	}
	info := item.Info
	size := int(info.Off + info.HeadLen + info.BodyLen)
	data := s.blocks[info.Block][int(info.Off):size]
	return item, data
}

func (s *Store) Delete(id Key) {
	s.meta.Delete(id)
}

func (s *Store) DeleteBatch(match func(*Item) bool) {
	s.DeleteBatch(match)
}

func (s *Store) Close() {
	s.lock.Lock()
	defer s.lock.Unlock()

	le := binary.LittleEndian
	h := sHeader{Magic: MAGIC, Version: META_VERSION}
	h.Blocks = int64(len(s.blocks))

	blocks := new(bytes.Buffer)
	for id, _ := range s.blocks {
		Success(binary.Write(blocks, le, id))
	}

	items := new(bytes.Buffer)
	h.Items = int64(s.meta.DumpAll(items))

	tmp := s.mpath + ".tmp"
	f, err := os.OpenFile(tmp, os.O_RDWR|os.O_CREATE, 0600)
	Success(err)
	defer f.Close()

	Success(binary.Write(f, le, h))
	Success(f.Write(blocks.Bytes()))
	Success(f.Write(items.Bytes()))

	Success(os.Rename(tmp, s.mpath))
}

func (s *Store) minBlock() (min int64, data []byte) {
	min = -1
	for i, d := range s.blocks {
		if i < min || min < 0 {
			min, data = i, d
		}
	}
	return min, data
}

func (s *Store) clear(timeout int) {
	for len(s.blocks) >= 0 && s.size > s.cap {
		min, data := s.minBlock()

		Lwarn(s.name, " delete block ", min, "size =", len(data),
			"cur-size =", s.size-len(data))

		s.meta.DeleteBatch(func(i *Item) bool {
			return i.Info.Block == min
		})

		s.size -= len(data)
		delete(s.blocks, min)

		go func() {
			// wait for request which is using buf finish
			if timeout > 0 {
				time.Sleep(time.Second * time.Duration(timeout))
			}
			s.lock.Lock()
			defer s.lock.Unlock()

			Success(os.Remove(s.getFileName(min)))
			Success(syscall.Munmap(data))
		}()
	}
}

func (s *Store) loadBlock(f *os.File, n int) {
	blocks := make([]int64, n)
	binary.Read(f, binary.LittleEndian, blocks)

	for _, b := range blocks {
		name := s.getFileName(b)
		if st, err := os.Stat(name); err != nil {
			Lerror(err)
		} else {
			s.blocks[b] = mmap(name, int(st.Size()))
			s.size += int(st.Size())
		}
		s.curBlock = b
	}

	s.clear(0)
}

func (s *Store) loadItem(f *os.File, n int) {
	infos := make([]ItemInfo, n)
	Success(binary.Read(f, binary.LittleEndian, infos))
	s.meta.AddAll(infos)
}

func (s *Store) load() {
	s.lock.Lock()
	defer s.lock.Unlock()

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

	var h sHeader
	Success(binary.Read(f, binary.LittleEndian, &h))

	if h.Magic != MAGIC || h.Version != META_VERSION {
		Lwarn(s.name, " META_VERSION or MAGIC not fix")
		return
	}

	s.loadBlock(f, int(h.Blocks))
	s.loadItem(f, int(h.Items))
}

func (s *Store) addBlock(size int) {
	s.lock.Lock()
	defer s.lock.Unlock()

	now := time.Now().UnixNano()
	name := s.getFileName(now)
	s.blocks[s.curBlock] = mmap(name, size)

	s.curBlock = now
	s.curOff = 0
	s.size += size

	timeout := GConfig["common.sock.req.timeout"].(int)
	s.clear(timeout + 1)
}

func (s *Store) getFileName(block int64) string {
	return fmt.Sprintf(FILE_NAME_FMT, s.path, s.curBlock)
}

func mmap(path string, size int) []byte {
	f, ferr := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0600)
	Success(ferr)
	defer f.Close()

	f.Truncate(int64(size))

	flag := syscall.PROT_READ | syscall.PROT_WRITE
	d, merr := syscall.Mmap(int(f.Fd()), 0, size, flag, syscall.MAP_SHARED)
	Success(merr)

	return d
}
