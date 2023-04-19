package store

import (
	"hornet/common"
	"sync"

	lru "github.com/hashicorp/golang-lru/v2"
)

type HotItems struct {
	lock  sync.RWMutex
	cache *lru.ARCCache[int64, *Item]
}

func (hi *HotItems) Init(itemSize int) {
	hi.lock.Lock()
	defer hi.lock.Unlock()
	if itemSize/6 < 1000 {
		itemSize = 1000
	}
	c, err := lru.NewARC[int64, *Item](int(itemSize))
	common.Success(err)
	hi.cache = c
}

func (hi *HotItems) Get(k *Key) (item *Item, ok bool) {
	hi.lock.RLock()
	defer hi.lock.RUnlock()
	item, ok = hi.cache.Get(k.H1)
	if item != nil && item.Key.H2 != k.H2 {
		//应对少量的碰撞
		return nil, false
	}
	return
}

func (hi *HotItems) Add(k *Key, item *Item) {
	hi.lock.RLock()
	defer hi.lock.RUnlock()
	hi.cache.Add(k.H1, item)
}

func (hi *HotItems) Remove(k *Key) {
	hi.lock.RLock()
	defer hi.lock.RUnlock()
	hi.cache.Remove(k.H1)
}

func (hi *HotItems) Purge() {
	hi.lock.RLock()
	defer hi.lock.RUnlock()
	hi.cache.Purge()
}
