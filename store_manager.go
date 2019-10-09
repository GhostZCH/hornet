package main

const (
	IDX_MEM = iota
	IDX_SSD
	IDX_HDD
)

type StoreManager struct {
	stores [3]*Store
}

func NewStoreManager() (sm *StoreManager) {
	sm = new(StoreManager)

	sm.stores[IDX_MEM] = NewStore(
		"mem",
		GConfig["cache.mem.meta"].(string),
		GConfig["cache.mem.path"].(string),
		GConfig["cache.mem.cap"].(int),
		GConfig["cache.mem.blocksize"].(int),
	)

	sm.stores[IDX_SSD] = NewStore(
		"ssd",
		GConfig["cache.ssd.meta"].(string),
		GConfig["cache.ssd.path"].(string),
		GConfig["cache.ssd.cap"].(int),
		GConfig["cache.ssd.blocksize"].(int),
	)

	sm.stores[IDX_HDD] = NewStore(
		"hdd",
		GConfig["cache.hdd.meta"].(string),
		GConfig["cache.hdd.path"].(string),
		GConfig["cache.hdd.cap"].(int),
		GConfig["cache.hdd.blocksize"].(int),
	)

	return sm
}

func (sm *StoreManager) Close() {
	for _, s := range sm.stores {
		if s != nil {
			s.Close()
		}
	}
}

func (sm *StoreManager) Add(item *Item) []byte {
	for i := IDX_HDD; i > 0; i-- {
		if sm.stores[i] != nil {
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

func (sm *StoreManager) Delete(id Key) {
	for _, s := range sm.stores {
		s.Delete(id)
	}
}

func (sm *StoreManager) DeleteBatch(match func(*Item) bool) {
	for _, s := range sm.stores {
		s.DeleteBatch(match)
	}
}

//TODO delete by mask
