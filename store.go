package main

import (
	"fmt"
	"os"
	"sync"
	"syscall"
	"time"
)

const BLOCK_COUNT uint64 = 512

type Store struct {
	dir      string
	cap      uint64
	size     uint64
	bSize    uint64
	curOff   uint64
	curBlock int64
	lock     sync.RWMutex
	blocks   map[int64][]byte
}

func NewStore(dir string, cap uint64, blocks []int64) *Store {
	s := &Store{
		dir:      dir,
		cap:      cap,
		size:     0,
		bSize:    cap / BLOCK_COUNT,
		curOff:   cap / BLOCK_COUNT,
		curBlock: 0,
		blocks:   make(map[int64][]byte)}

	for _, b := range blocks {
		path := getPath(dir, b)
		if info, err := os.Stat(path); err != nil {
			Lwarn(fmt.Sprintf("stat %s error [%v]", path, err))
		} else {
			if data, err := mmap(path, int(info.Size())); err != nil {
				Lwarn(fmt.Sprintf("mmap %s error [%v]", path, err))
			} else {
				s.blocks[b] = data
			}
		}
	}

	return s
}

func (s *Store) GetBlocks() (blocks []int64) {
	s.lock.Lock()
	defer s.lock.Unlock()
	for b, _ := range s.blocks {
		blocks = append(blocks, b)
	}
	return blocks
}

func (s *Store) Get(block, off int64, len uint64) []byte {
	s.lock.RLock()
	defer s.lock.RUnlock()

	if b, ok := s.blocks[block]; ok {
		return b[off : off+int64(len)]
	}

	return nil
}

func (s *Store) Alloc(size uint64) (block int64, off int64, data []byte) {
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

func (s *Store) Clear() (blocks []int64) {
	s.lock.Lock()
	defer s.lock.Unlock()

	for len(s.blocks) >= 0 && s.size > s.cap {
		min, data := s.minBlock()
		path := getPath(s.dir, min)

		Lwarn(fmt.Sprintf("Store.Clear: rm %s release %d]", path, len(data)))

		s.size -= uint64(len(data))
		delete(s.blocks, min)
		blocks = append(blocks, min)

		Success(os.Remove(path))
		Success(syscall.Munmap(data))
	}

	return blocks
}

func (s *Store) Close() {
	s.lock.Lock()
	defer s.lock.Unlock()

	for _, data := range s.blocks {
		syscall.Munmap(data)
	}
}

func (s *Store) minBlock() (min int64, data []byte) {
	min = 0
	for i, d := range s.blocks {
		if i < min || min == 0 {
			min, data = i, d
		}
	}
	return min, data
}

func (s *Store) addBlock(size uint64) {
	s.lock.Lock()
	defer s.lock.Unlock()

	now := time.Now().UnixNano()
	name := getPath(s.dir, now)
	if data, err := mmap(name, int(size)); err != nil {
		s.blocks[now] = data
		s.curBlock = now
		s.curOff = 0
		s.size += size
	} else {
		Lwarn("add block failed ", name)
	}
}

func getPath(dir string, block int64) string {
	return fmt.Sprintf("%s/%016x.dat", dir, block)
}

func mmap(path string, size int) (data []byte, err error) {
	var f *os.File
	f, err = os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0600)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	err = f.Truncate(int64(size))
	if err != nil {
		return nil, err
	}

	flag := syscall.PROT_READ | syscall.PROT_WRITE
	return syscall.Mmap(int(f.Fd()), 0, size, flag, syscall.MAP_SHARED)
}
