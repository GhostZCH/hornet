package main

import (
	"fmt"
	"os"
	"sync"
	"syscall"
	"time"
)

type Store struct {
	lock   sync.RWMutex
	blocks map[int64][]byte
}

func (s *Store) Alloc(size int64) (block int64, off int64, data []byte) {
	if size > s.bSize {
		s.addBlock(size) // single block for big data
	} else if size+s.curOff > s.bSize {
		s.addBlock(s.bSize)
	}

	block = s.curBlock
	off = int64(s.curOff)

	data = s.blocks[s.curBlock][s.curOff : s.curOff+size]
	s.curOff += size

	return block, off, data
}

func (s *Store) Get(id Key) (*Item, []byte, *string) {
	item, putItem := s.meta.Get(id)
	if item == nil {
		return nil, nil, nil
	}
	size := int(item.Off + item.HeadLen + item.BodyLen)
	data := s.blocks[item.Block][int(item.Off):size]

	return item, data, &s.name
}

// func (s *Store) Delete(id Key) {
// 	s.meta.Delete(id)
// }

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
