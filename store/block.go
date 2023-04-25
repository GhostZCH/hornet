package store

import (
	"fmt"
	"hornet/common"
	"io/ioutil"
	"os"
	"strings"
	"sync"
	"syscall"
	"time"
)

type Block struct {
	id   int64
	data []byte
	path string
}

func NewBlock(dir string, id int64, size int64) *Block {
	if id < 0 {
		id = time.Now().UnixNano()
	}
	p := getPath(dir, id)
	d, err := mmap(p, int(size))
	if err != nil {
		panic(err)
	}

	return &Block{id: id, path: p, data: d}
}

func (b *Block) Remove() {
	common.Success(os.Remove(b.path))
	common.Success(syscall.Munmap(b.data))
}

func (b *Block) Close() {
	syscall.Munmap(b.data)
}

func getPath(dir string, block int64) string {
	return fmt.Sprintf("%s/%016x.dat", dir, block)
}

func LoadBlocks(dir string) *sync.Map {
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		panic(err)
	}

	blocks := &sync.Map{}
	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".dat") {
			var id int64
			_, err := fmt.Sscanf(file.Name(), "%016x.dat", &id)
			if err != nil {
				panic(err)
			}
			fileInfo, err := os.Stat(dir + file.Name())
			if err != nil {
				panic(err)
			}
			blocks.Store(id, NewBlock(dir, id, fileInfo.Size()))
		}
	}

	return blocks
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
