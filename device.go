package main

type Device struct {
	name  string
	meta  *Meta
	store *Store
}

func NewDevice(name, dir string, cap int) *Store {
	m := NewMeta(dir)
	s := NewStore(dir, cap, m.getBucket())
	return &Device{name: name, store: s, meta: m}
}

func (d *Device) Close() {
	d.store.Close()
	d.meta.Dump(d.store.GetBlocks())
}

func (d *Device) Get(k Key) (*Item, []byte) {
	return nil, nil
}

func (d *Device) Alloc(k Key, head int64, body int64) (item *Item, data []byte) {
	item = d.meta.Alloc(k)
	item.HeadLen = header
	item.BodyLen = body
	item.Block, item.Off, data = d.store.Alloc(header + body)
	delBlocks := d.store.Clear()

	for b := range delBlocks {
		d.DeleteBatch(func(item *Item) {
			return item.Block == b
		})
	}

	return item, data
}

func (d *Device) Add(k Key) {
	d.meta.Add(k)
}

func (d *Device) DeleteBatch(match func(*Item) bool) {
	d.meta.DeleteBatch(match)
}
