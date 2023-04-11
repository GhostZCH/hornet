package store

import (
	"hornet/common"
	"sync"
)

const BucketCount int = 128
const BlockCount int64 = 512
const MinBlockSize int64 = 1 * 1024 * 1024

type Device struct {
	dir        string
	name       string
	cap        int64
	blockSize  int64
	blockCount int64
	curOff     int64
	curBlock   *Block
	bucket     []*Bucket
	blocks     map[int64]*Block
	blockLock  sync.Mutex
}

func NewDevice(conf *common.DeviceCfg) *Device {
	cap := common.ParseSize(conf.Size)
	blockSize, blockCount := getBlockInfo(cap)

	dev := &Device{
		dir:        conf.Dir,
		name:       conf.Name,
		cap:        common.ParseSize(conf.Size),
		blockCount: blockCount,
		blockSize:  blockSize,
		curOff:     -1, // start a new block affter roboot
		curBlock:   nil,
		blocks:     LoadBlocks(conf.Dir),
		bucket:     make([]*Bucket, 0)}

	return dev
}

func (d *Device) Get(k Key) []byte {
	return nil
}

func (d *Device) Put(k Key, buf []byte) (block int64, off int64) {
	size := int64(len(buf))
	if size > d.blockSize {
		d.addBlock(size) // single block for big data
	} else if d.curBlock == nil || size+d.curOff > d.blockSize {
		d.addBlock(d.blockSize)
	}

	off = d.curOff
	block = d.curBlock.id
	copy(buf, d.curBlock.data[d.curOff:d.curOff+size])
	d.curOff += size
	return
}

func (d *Device) Size() int64 {
	size := int64(0)
	for _, b := range d.blocks {
		size += int64(len(b.data))
	}
	return size
}

func (d *Device) Clear() (blocks []int64) {
	d.blockLock.Lock()
	defer d.blockLock.Unlock()

	for len(d.blocks) >= 0 && d.Size() > d.cap {
		b := d.earlistBlock()
		delete(d.blocks, b.id)
		blocks = append(blocks, b.id)
	}

	return blocks
}

func (d *Device) Close() {
	d.blockLock.Lock()
	defer d.blockLock.Unlock()

	for _, b := range d.blocks {
		b.Close()
	}
}

func (d *Device) earlistBlock() *Block {
	min := int64(0)
	for i := range d.blocks {
		if i < min || min == 0 {
			min = i
		}
	}
	return d.blocks[min]
}

func (d *Device) addBlock(size int64) {
	d.blockLock.Lock()
	defer d.blockLock.Unlock()
	d.curBlock = NewBlock(d.dir, -1, size)
	d.curOff = 0
}

func getBlockInfo(cap int64) (blockSize int64, blockCount int64) {
	if cap/BlockCount < MinBlockSize {
		return MinBlockSize, cap / MinBlockSize
	}
	return cap / BlockCount, BlockCount
}
