package main

type Device struct {
	meta  *Meta
	store *Store
}

func NewDevice(dir string, cap int) *Device {
	if cap == 0 || dir == "" {
		return nil
	}

	m := NewMeta(dir)
	s := NewStore(dir, cap, m.GetBlocks())
	d := &Device{store: s, meta: m}
	d.clear()
}

func (d *Device) Close() {
	d.store.Close()
	d.meta.Dump(d.store.GetBlocks())
}

func (d *Device) Get(k Key) (*Item, []byte) {
	if i := d.meta.Get(k); item != nil {
		if d := d.store.Get(i.Block, i.Off, i.BodyLen+i.HeadLen); d != nil {
			return i, d
		}
	}
	return nil, nil
}

func (d *Device) clear() {
	for b := range d.store.Clear() {
		d.DeleteBatch(func(item *Item) { return item.Block == b })
	}

	// delete items in unloaded blocks if any
	// should not run those code in normal conditions
	for b := range m.GetBlocks() {
		found := false
		for i := range s.GetBlocks() {
			if i == b {
				found = true
				break
			}
		}
		if !found {
			Lwarn("delete items in unload block ", b)
			m.DeleteBatch(func(item *Item) { return item.Block == b })
		}
	}
}

func (d *Device) Alloc(k Key, head, body int64) (item *Item, data []byte) {
	item = d.meta.Alloc(k)
	if item == nil {
		return nil, nil
	}

	item.HeadLen = head
	item.BodyLen = body
	item.Block, item.Off, data = d.store.Alloc(head + body)
	d.clear()

	return item, data
}

func (d *Device) Add(k Key) {
	d.meta.Add(k)
}

func (d *Device) DeleteBatch(match func(*Item) bool) {
	d.meta.DeleteBatch(match)
}
