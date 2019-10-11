package main

const RANGE_SIZE int = 4 * 1024 * 1024

const (
	IDX_MEM = iota
	IDX_SSD
	IDX_HDD
)

type DeviceManager struct {
	devices [3]*Device
}

func NewDeviceManager() (sm *StoreManager) {
	dm = new(DeviceManager)

	dm.devices[IDX_MEM] = NewStore(
		"mem",
		GConfig["cache.mem.dir"].(string),
		GConfig["cache.mem.cap"].(int),
	)

	dm.devices[IDX_SSD] = NewStore(
		"ssd",
		GConfig["cache.ssd.dir"].(string),
		GConfig["cache.ssd.cap"].(int),
	)

	dm.devices[IDX_HDD] = NewStore(
		"hdd",
		GConfig["cache.hdd.dir"].(string),
		GConfig["cache.hdd.cap"].(int),
	)

	return dm
}

func (dm *DeviceManager) Close() {
	for _, d := range dm.stores {
		if d != nil {
			d.Close()
		}
	}
}

func (dm *DeviceManager) Add(k Key) []byte {
	for i := IDX_HDD; i > 0; i-- {
		if dm.stores[i] != nil {
			return sm.stores[i].Add(item)
		}
	}
	return nil
}

func (sm *StoreManager) Get(id Key) (*Item, []byte, *string) {
	for i, s := range sm.stores {
		if item, data, name := s.Get(id); item != nil {
			for j := i - 1; j >= 0; j-- {
				if sm.stores[j] != nil {
					new := *item
					buf := sm.stores[j].Add(&new)
					copy(buf, data)
					break
				}
			}

			return item, data, name
		}
	}
	return nil, nil, nil
}

func (sm *DeviceManager) DeleteBatch(match func(*Item) bool) {
	for _, s := range sm.stores {
		s.DeleteBatch(match)
	}
}

//TODO delete by mask
