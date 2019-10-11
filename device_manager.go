package main

import (
	"errors"
	"hash/crc64"
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

func (dm *DeviceManager) Get(id Hash, start, end int64) (*Item, [][]byte, *string) {
	for i, d := range dm.devices {
		// TODO range
		// bytes 0- 怎么处理？
		if item, data := d.Get(k); item != nil {
			for j := i - 1; j >= 0; j-- {
				if sm.stores[j] != nil {
					new := *item
					buf := sm.stores[j].Add(&new)
					copy(buf, data)
					break
				}
			}

			return item, data, &DEVICES[i]
		}
	}
	return nil, nil, nil
}

func (dm *DeviceManager) Delete(match func(*Item) bool) {
	for _, d := range dm.devices {
		s.DeleteBatch(match)
	}
}

func (dm *DeviceManager) DeleteByID(id Hash) {
	for _, s := range dm.devices {
		s.DeleteBatch(func(item *Item) {
			return item.Key.ID == id
		})
	}
}

func (dm *DeviceManager) DeleteByID(id Hash) {
	for _, s := range dm.devices {
		s.DeleteBatch(func(item *Item) {
			return item.Key.ID == id
		})
	}
}

func (dm *DeviceManager) DeleteByGroup(group []byte) {
	g := crc64.Checksum(group, nil)
	for _, s := range dm.devices {
		s.DeleteBatch(func(item *Item) {
			return item.GroupCRC == g
		})
	}
}

//TODO delete by mask
