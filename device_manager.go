package main

import (
	"errors"
	"fmt"
	"go.uber.org/zap"
)

const RangeSize int64 = 4 * 1024 * 1024

const DeviceLimit int = 8

type DeviceManager struct {
	devices [DeviceLimit]*Device
	lowest  int
}

func NewDeviceManager() *DeviceManager {
	dm := &DeviceManager{lowest: DeviceLimit}

	for lv, _ := range dm.devices {
		dir, ok1 := Conf[fmt.Sprintf("cache.%d.dir", lv)]
		cap, ok2 := Conf[fmt.Sprintf("cache.%d.cap", lv)]

		if ok1 && ok2 {
			if dm.devices[lv] = NewDevice(dir.(string), lv, cap.(int)); dm.devices[lv] != nil {
				dm.lowest = lv
			}
			Log.Warn("load device",
				zap.Int("lv", lv),
				zap.String("dir", dir.(string)),
				zap.Int("cap", cap.(int)),
				zap.Bool("re", dm.devices[lv] != nil))
		}
	}

	if dm.lowest == DeviceLimit {
		panic(errors.New("no devices"))
	}

	return dm
}

func (dm *DeviceManager) Close() {
	for _, d := range dm.devices {
		if d != nil {
			d.Close()
		}
	}
}

func (dm *DeviceManager) Alloc(item *Item) ([]byte, int) {
	for i := DeviceLimit - 1; i > 0; i-- {
		if dm.devices[i] != nil {
			return dm.devices[i].Alloc(item), i
		}
	}
	return nil, -1
}

func (dm *DeviceManager) Add(dev int, k Key) {
	dm.devices[dev].Add(k)
}

func (dm *DeviceManager) Get(k Key) (*Item, []byte, int) {
	for i, d := range dm.devices {
		if item, data := d.Get(k); item != nil {
			for j := i - 1; j >= 0; j-- {
				if dm.devices[j] != nil {
					new := *item
					buf := dm.devices[j].Alloc(&new)
					copy(buf, data)
					dm.devices[j].Add(k)
					break
				}
			}

			return item, data, i
		}
	}
	return nil, nil, DeviceLimit
}

func (dm *DeviceManager) Del(match func(*Item) bool) uint {
	n := uint(0)
	for _, d := range dm.devices {
		n += d.DeleteBatch(match)
	}
	return n
}

func (dm *DeviceManager) DelPut(k Key) {
	for _, d := range dm.devices {
		d.DeletePut(k)
	}
}
