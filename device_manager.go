package main

import (
	"errors"
	"hash/crc64"
	"regexp"
)

const RANGE_SIZE uint32 = uint32(4 * 1024 * 1024)

var DEVICES [3]string = [3]string{"mem", "ssd", "hdd"}

type DeviceManager struct {
	devices [len(DEVICES)]*Device
}

func NewDeviceManager() *DeviceManager {
	dm := new(DeviceManager)

	err := errors.New("no devices")
	for i, name := range DEVICES {
		dir := GConfig["cache."+name+".dir"].(string)
		cap := GConfig["cache."+name+".cap"].(int)
		if dm.devices[i] = NewDevice(dir, cap); dm.devices[i] != nil {
			err = nil
		}
	}

	Success(err)

	return dm
}

func (dm *DeviceManager) Close() {
	for _, d := range dm.devices {
		if d != nil {
			d.Close()
		}
	}
}

func (dm *DeviceManager) Alloc(k Key, head, body int64) (*Item, []byte, int) {
	for i := len(DEVICES) - 1; i > 0; i-- {
		if dm.devices[i] != nil {
			return dm.devices[i].Alloc(k, head, body)
		}
	}
	panic(errors.New("NO_DEVICE_FOR_ALLOC"))
}

func (dm *DeviceManager) Add(dev int, k Key) {
	dm.devices[dev].Add(k)
}

func (dm *DeviceManager) Get(k Key) (*Item, [][]byte, *string) {
	for i, d := range dm.devices {
		// TODO range
		// bytes 0- 怎么处理？
		if item, data := d.Get(k); item != nil {
			for j := i - 1; j >= 0; j-- {
				if sm.stores[j] != nil {
					new, buf := dm.devices[j].Alloc(k, item.HeadLen, item.BodyLen)
					copy(buf, data)
					dm.devices[j].Add(k)
					break
				}
			}

			return item, data, &DEVICES[i]
		}
	}
	return nil, nil, nil
}

func (dm *DeviceManager) Del(match func(*Item) bool) {
	for _, d := range dm.devices {
		s.DeleteBatch(match)
	}
}

func (dm *DeviceManager) DelByID(id Hash) {
	for _, s := range dm.devices {
		s.DeleteBatch(func(item *Item) {
			return item.Key.ID == id
		})
	}
}

func (dm *DeviceManager) DelByGroup(group []byte) {
	g := crc64.Checksum(group, nil)
	for _, s := range dm.devices {
		s.DeleteBatch(func(item *Item) {
			return item.GroupCRC == g
		})
	}
}

func (dm *DeviceManager) DelByType(group, itemType []byte) {
	g := crc64.Checksum(group, nil)
	t := crc64.Checksum(itemType, nil)
	for _, s := range dm.devices {
		s.DeleteBatch(func(item *Item) {
			return item.GroupCRC == g && item.TypeCRC == t
		})
	}
}

func (dm *DeviceManager) DelByTag(group []byte, mask int64) {
	g := crc64.Checksum(group, nil)
	for _, s := range dm.devices {
		s.DeleteBatch(func(item *Item) {
			return item.GroupCRC == g && item.Tag&mask != 0
		})
	}
}

func (dm *DeviceManager) DelByRegex(group []byte, regex []byte) {
	g := crc64.Checksum(group, nil)
	r := regexp.MustCompile(string(reg))
	for _, s := range dm.devices {
		s.DeleteBatch(func(item *Item) {
			return item.GroupCRC == g && r.Match(item.RawKey)
		})
	}
}

//TODO delete by mask
