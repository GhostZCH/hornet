package store

import (
	"hornet/common"
	"sync"
)

const BucketCount int = 16
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
	blockLock  sync.RWMutex
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

	for i := 0; i < BucketCount; i++ {
		dev.bucket = append(dev.bucket, NewBucket(i, BucketCount, dev.dir))
	}

	return dev
}

func (d *Device) Get(k *Key) (buf []byte, item *Item, isHot bool) {
	b := d.getBucket(k)
	item, isHot = d.bucket[b].Get(k)
	if item != nil {
		d.blockLock.RLock()
		defer d.blockLock.RUnlock()
		block, ok := d.blocks[item.Block]
		if ok {
			buf = block.data[item.Offset : item.Offset+int64(item.HeaderLen)+int64(item.BodyLen)]
			return
		}
	}

	return nil, nil, false
}

func (d *Device) Put(item *Item, buf []byte) {
	off, block := d.putBuf(buf)

	b := d.getBucket(&item.Key)
	item.Block = block
	item.Offset = off
	d.bucket[b].Add(item)
}

func (d *Device) putBuf(buf []byte) (off int64, block int64) {
	size := int64(len(buf))
	d.blockLock.Lock()
	defer d.blockLock.Unlock()
	if size > d.blockSize {
		// single block for big data
		d.addBlock(size)
	} else if d.curBlock == nil || size+d.curOff > d.blockSize {
		d.addBlock(d.blockSize)
	}

	off = d.curOff
	block = d.curBlock.id
	copy(d.curBlock.data[d.curOff:d.curOff+size], buf)
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
	d.curBlock = NewBlock(d.dir, -1, size)
	d.curOff = 0
}

func (d *Device) getBucket(k *Key) int {
	return int(k.H2) % len(d.bucket)
}

func getBlockInfo(cap int64) (blockSize int64, blockCount int64) {
	if cap/BlockCount < MinBlockSize {
		return MinBlockSize, cap / MinBlockSize
	}
	return cap / BlockCount, BlockCount
}
