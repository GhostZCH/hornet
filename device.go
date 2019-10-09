package main

import (
	"os"
)

type Device struct {
	cap      int
	size     int
	bSize    int
	curOff   int
	curBlock int64
	name     string
	dir      string
	meta     *Meta
	store    *Store
}

func NewDevice(name, metaDir, dir string, cap, bSize int) *Store {
	d := &Device{name: name, dir: dir,
		cap: cap, bSize: bSize, curOff: bSize,
		blocks: make(map[int64][]byte), meta: NewMeta()}

	s.lock.Lock()
	defer s.lock.Unlock()

	defer f.Close()
	defer os.Remove(meta)

	s.meta.Load(f)

	return s
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
