package main

import (
	"fmt"
	"os"
	"sync"
	"syscall"
	"time"
)

type Store struct {
	dir      string
	cap      int
	size     int
	bSize    int
	curOff   int
	curBlock int64
	lock     sync.RWMutex
	blocks   map[int64][]byte
}

func NewStore() {
	// todo
}

func (s *Store) Get(block, off, len int64) []byte {
	s.lock.Lock()
	defer s.lock.Unlock()

	if b, ok := s.lock[block]; ok {
		return b[off : off+len]
	}

	return nil
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

func (s *Store) minBlock() (min int64, data []byte) {
	min = -1
	for i, d := range s.blocks {
		if i < min || min < 0 {
			min, data = i, d
		}
	}
	return min, data
}

func (s *Store) Clear() {
	if len(s.blocks) >= 0 && s.size > s.cap {
		s.lock.Lock()
		defer s.lock.Unlock()

		min, data := s.minBlock()

		s.size -= len(data)
		delete(s.blocks, min)

		s.lock.Lock()
		defer s.lock.Unlock()

		Success(os.Remove(s.getPath("dat", min)))
		Success(syscall.Munmap(data))
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

func (s *Store) getPath(ext string, bid int64) string {
	return fmt.Sprintf("%s/%016x.dat", s.dir, bid)
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
