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

func (s *Store) Get(k *Key) (buf []byte, headerLen int, level int) {
	for i, d := range s.devices {
		buffer, item, isHot := d.Get(k)
		if isHot && i > 0 {
			newItem := &Item{
				Key:        item.Key,
				HeaderLen:  item.HeaderLen,
				BodyLen:    item.BodyLen,
				UserGroup:  item.UserGroup,
				User:       item.User,
				RootDomain: item.RootDomain,
				Domain:     item.Domain,
				SrcGroup:   item.SrcGroup,
				Expires:    item.Expires,
				Tags:       item.Tags}
			newItem.Path = make([]byte, len(item.Path))
			copy(newItem.Path, item.Path)

			s.devices[i-1].Put(newItem, buffer)
		}
		if buffer != nil {
			return buffer, int(item.HeaderLen), i
		}
	}
	return nil, 0, -1
}

func (s *Store) Put(item *Item, buf []byte) {
	dev := s.devices[len(s.devices)-1]
	dev.Put(item, buf)
}
