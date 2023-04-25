package store

import (
	"hornet/common"
	"sort"
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
	blocks     *sync.Map
	addLock    sync.Mutex
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
		block, ok := d.blocks.Load(item.Block)
		if ok {
			data := block.(*Block).data
			end := item.Offset + int64(item.HeaderLen) + int64(item.BodyLen)
			buf = data[item.Offset:end]
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
	// 添加过程需要加锁
	d.addLock.Lock()
	defer d.addLock.Unlock()

	size := int64(len(buf))
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
	d.blocks.Range(func(key, value any) bool {
		size += int64(len(value.(*Block).data))
		return true
	})
	return size
}

func (d *Device) clear(appendSize int64) {
	ids := make([]int64, 0)
	d.blocks.Range(func(key, value any) bool {
		ids = append(ids, key.(int64))
		return true
	})

	sort.Slice(ids, func(i, j int) bool {
		return ids[i] < ids[j]
	})

	size := d.Size() + appendSize
	for i := 0; size > d.cap && i < len(ids)-1; i++ {
		id := ids[i]
		b, _ := d.blocks.Load(id)
		size -= int64(len(b.(*Block).data))
		b.(*Block).Remove()
		// TODO LOG id
	}
}

func (d *Device) Close() {
	d.blocks.Range(func(key, value any) bool {
		value.(*Block).Close()
		return true
	})
}

func (d *Device) addBlock(size int64) {
	d.clear(size)
	d.curBlock = NewBlock(d.dir, -1, size)
	d.curOff = 0
	d.blocks.Store(d.curBlock.id, d.curBlock)
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
