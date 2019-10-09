package main

import (
	"fmt"
	"os"
	"sync"
	"syscall"
	"time"
)

type Store struct {
	cap      int
	size     int
	bSize    int
	curOff   int
	curBlock int64
	name     string
	dir      string
	meta     *Meta
	lock     sync.RWMutex
	blocks   map[int64][]byte
}

func NewStore(name, metaDir, dir string, cap, bSize int) *Store {
	s := &Store{name: name, dir: dir,
		cap: cap, bSize: bSize, curOff: bSize,
		blocks: make(map[int64][]byte), meta: NewMeta()}

	s.lock.Lock()
	defer s.lock.Unlock()

	meta := s.getPath("meta", META_VERSION)
	f, err := os.Open(meta)
	if err != nil {
		if !os.IsNotExist(err) {
			panic(err)
		}
		Lwarn(s.name, " no meta file found")
		return s
	}

	defer f.Close()
	defer os.Remove(meta)

	s.meta.Load(f)

	return s
}

func (s *Store) Close() {
	s.lock.Lock()
	defer s.lock.Unlock()

	meta := s.getPath("meta", META_VERSION)
	tmp := meta + ".tmp"
	f, err := os.OpenFile(tmp, os.O_RDWR|os.O_CREATE, 0600)
	Success(err)
	s.meta.Dump(f)
	Success(f.Close())
	Success(os.Rename(tmp, meta))
}

func (s *Store) Add(k Key) {
	s.meta.Add(k)
}

func (s *Store) Alloc(item *Item) []byte {
	size := int(item.HeadLen + item.BodyLen)
	if size > s.bSize {
		s.addBlock(size) // single block for big data
	} else if size+s.curOff > s.bSize {
		s.addBlock(s.bSize)
	}

	item.Block = s.curBlock
	item.Off = int64(s.curOff)

	data := s.blocks[s.curBlock][s.curOff : s.curOff+size]
	s.curOff += size

	return data
}

func (s *Store) Get(id Key) (*Item, []byte, *string) {
	item := s.meta.Get(id)
	if item == nil {
		return nil, nil, nil
	}
	size := int(item.Off + item.HeadLen + item.BodyLen)
	data := s.blocks[item.Block][int(item.Off):size]

	return item, data, &s.name
}

func (s *Store) Delete(id Key) {
	s.meta.Delete(id)
}

func (s *Store) DeleteBatch(match func(*Item) bool) {
	s.DeleteBatch(match)
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
			return i.Block == min
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

			Success(os.Remove(s.getPath("dat", min)))
			Success(syscall.Munmap(data))
		}()
	}
}

func (s *Store) addBlock(size int) {
	s.lock.Lock()
	defer s.lock.Unlock()

	now := time.Now().UnixNano()
	name := s.getPath("dat", now)
	s.blocks[now] = mmap(name, size)

	s.curBlock = now
	s.curOff = 0
	s.size += size

	timeout := GConfig["common.sock.req.timeout"].(int)
	s.clear(timeout + 1)
}

func (s *Store) getPath(ext string, data int64) string {
	return fmt.Sprintf("%s/%s-%016x.%s", s.dir, s.name, data, ext)
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
