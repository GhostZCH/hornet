package main

const BLOCK_COUNT int = 128

type Device struct {
	meta  *Meta
	store *Store
}

func NewDevice(name, dir string, cap int) *Store {
	d := &Device{name: name, dir: dir,
		cap:   cap,
		store: NewStore(), meta: NewMeta()}

	bSize := cap / BLOCK_COUNT

	return s
}

func Close() {

}

func (d *Device) Get(k Key) (*Item, []byte) {
	return nil, nil
}

func (d *Device) Alloc(k Key, size int64) (*Item, []byte) {
	d.meta.Alloc(k)
	block, off, data := d.store.Alloc(size)
	return nil, nil
}

func (d *Device) Add(k Key) {
	d.meta.Add(k)
}
