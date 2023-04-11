package store

import "hornet/common"

type Store struct {
	devices []*Device
}

func NewStore(conf *common.CacheConfig) *Store {
	devices := make([]*Device, 0)
	for _, dc := range conf.Device {
		devices = append(devices, NewDevice(&dc))
	}
	return &Store{devices: devices}
}

func (s *Store) Get(srcKey []byte) []byte {
	k := GetKey(srcKey)
	for i, d := range s.devices {
		buf := d.Get(k)
		if d.Get(k) != nil {
			if i > 0 {
				s.devices[i-1].Put(k, buf)
			}
			return buf
		}
	}
	return nil
}

// TODO : alloc implate in upstream model, simply code

func (s *Store) Put(srcKey []byte, buf []byte) {
	last := len(s.devices) - 1
	k := GetKey(srcKey)
	s.devices[last].Put(k, buf)
}
