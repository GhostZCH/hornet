package store

import (
	"hornet/common"
	"runtime"
	"sort"
	"sync"
)

const BucketCount int = 256
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
	removeLock sync.Mutex
}

func NewDevice(conf *common.DeviceCfg) *Device {
	size := common.ParseSize(conf.Size)
	blockSize, blockCount := getBlockInfo(size)

	dev := &Device{
		dir:        conf.Dir,
		name:       conf.Name,
		cap:        common.ParseSize(conf.Size),
		blockCount: blockCount,
		blockSize:  blockSize,
		curOff:     -1, // start a new block after reboot
		curBlock:   nil,
		blocks:     LoadBlocks(conf.Dir),
		bucket:     make([]*Bucket, 0)}

	for i := 0; i < BucketCount; i++ {
		dev.bucket = append(dev.bucket, NewBucket(i, dev.dir))
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
			end := item.Offset + item.HeaderLen + item.BodyLen
			buf = data[item.Offset:end]
			return
		}
	}

	return nil, nil, false
}

func (d *Device) Remove(args []*RemoveArg) {
	// key删除较快，直接删除
	isKey := false
	for _, arg := range args {
		if arg.Cmd == "key" {
			isKey = true
			break
		}
	}
	if isKey {
		if len(args) != 1 {
			panic("按照key删除不支持包含其他的参数")
		}
		k := NewKey([]byte(args[0].Val))
		idx := d.getBucket(k)
		d.bucket[idx].RemoveByKey(k)
		return
	}

	d.removeLock.Lock()
	defer d.removeLock.Unlock()

	var wg sync.WaitGroup
	wg.Add(BucketCount)

	// 限制最大并行不超过cpu数量的一半防止删除造成正常业务抖动
	sem := make(chan struct{}, runtime.NumCPU()/2)

	for i := 0; i < BucketCount; i++ {
		sem <- struct{}{}
		go func(b *Bucket) {
			b.Remove(args)
			<-sem
		}(d.bucket[i])
	}

	wg.Wait()
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
	return int(k.Hash64() % int64(BucketCount))
}

func getBlockInfo(cap int64) (blockSize int64, blockCount int64) {
	if cap/BlockCount < MinBlockSize {
		return MinBlockSize, cap / MinBlockSize
	}
	return cap / BlockCount, BlockCount
}
