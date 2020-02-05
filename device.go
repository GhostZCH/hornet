package main

import (
	"go.uber.org/zap"
)

const BucketLimit int = 256

type Device struct {
	level int
	meta  *Meta
	store *Store
}

func NewDevice(dir string, level int, cap int) *Device {
	if cap == 0 || dir == "" {
		return nil
	}

	m := NewMeta(dir)
	s := NewStore(dir, uint64(cap), m.GetBlocks())
	d := &Device{store: s, meta: m, level: level}
	d.clear()

	return d
}

func (d *Device) Close() {
	d.store.Close()
	d.meta.Dump(d.store.GetBlocks())
}

func (d *Device) Get(k Key) (*Item, []byte) {
	if i := d.meta.Get(k); i != nil {
		if d := d.store.Get(i.Block, i.Off, i.BodyLen+i.HeadLen); d != nil {
			return i, d
		}
	}
	return nil, nil
}

func (d *Device) clear() {
	for _, b := range d.store.Clear() {
		d.DeleteBatch(func(item *Item) bool { return item.Block == b })
	}

	// delete items in unloaded blocks if any
	// should not run those code in normal conditions
	for _, b := range d.meta.GetBlocks() {
		found := false
		for _, i := range d.store.GetBlocks() {
			if i == b {
				found = true
				break
			}
		}
		if !found {
			Log.Warn("delete items ", zap.Int64("block", b))
			d.meta.DeleteBatch(func(item *Item) bool { return item.Block == b })
		}
	}
}

func (d *Device) Alloc(item *Item) (data []byte) {
	if !d.meta.Alloc(item) {
		return nil
	}

	item.Block, item.Off, data = d.store.Alloc(item.HeadLen + item.BodyLen)
	d.clear()

	return data
}

func (d *Device) Add(k Key) {
	d.meta.Add(k)
}

func (d *Device) DeleteBatch(match func(*Item) bool) uint {
	return d.meta.DeleteBatch(match)
}

func (d *Device) DeletePut(k Key) {
	d.meta.DeletePut(k)
}
