package store

import (
	"context"
	"database/sql"
	"fmt"
	"hornet/common"
	"sync"
	"time"

	"github.com/allegro/bigcache/v3"
	_ "github.com/mattn/go-sqlite3"
)

type Bucket struct {
	index int
	db    *sql.DB
	cache *bigcache.BigCache
	lock  sync.RWMutex
}

func NewBucket(index int, bucket int, dir string) *Bucket {
	path := fmt.Sprintf("%s/meta_%05d_%05d.db", dir, bucket, index)
	db, err := sql.Open("sqlite3", "file:"+path+"?cache=shared&mode=rwc")
	if err != nil {
		panic(err)
	}
	createTable(db)

	cacheConf := bigcache.DefaultConfig(time.Hour)
	cacheConf.MaxEntrySize = 10240
	cacheConf.CleanWindow = 1 * time.Minute

	c, e := bigcache.New(context.Background(), cacheConf)
	common.Success(e)

	return &Bucket{
		index: index,
		db:    db,
		cache: c}
}

func (b *Bucket) RemoveByKey(k Key) {
	b.lock.Lock()
	defer b.lock.Unlock()
	b.cache.Delete(k.String())
	common.Success(b.db.Exec("DELETE FROM items WHERE H1=? AND H2=?", k.H1, k.H2))
}

func (b *Bucket) RemoveByPrefix(patten string) {
	b.lock.Lock()
	defer b.lock.Unlock()
	b.cache.Reset()
	common.Success(b.db.Exec(`DELETE FROM item_info WHERE url LIKE '?%'`, patten))
}

func (b *Bucket) RemoveBySurfix(patten string) {
	b.lock.Lock()
	defer b.lock.Unlock()
	b.cache.Reset()
	common.Success(b.db.Exec(`DELETE FROM item_info WHERE url LIKE '%?'`, patten))
}

func (b *Bucket) RemoveByBlock(block int64) {
	b.lock.Lock()
	defer b.lock.Unlock()
	common.Success(b.db.Exec(`DELETE FROM items WHERE block = ?`, block))
}

func (b *Bucket) DeleteByUserGroup(userGroup uint64) {
	b.lock.Lock()
	defer b.lock.Unlock()
	b.cache.Reset()
	common.Success(b.db.Exec(`DELETE FROM items WHERE user_group = ?`, userGroup))
}

func (b *Bucket) RemoveByDomain(domain uint64) {
	b.lock.Lock()
	defer b.lock.Unlock()
	b.cache.Reset()
	common.Success(b.db.Exec(`DELETE FROM items WHERE domain = ?`, domain))
}

func (b *Bucket) RemoveByRootDomain(rootDomain string) {
	b.lock.Lock()
	defer b.lock.Unlock()
	common.Success(b.db.Exec(`DELETE FROM items WHERE root_domain = ?`, rootDomain))
}

func (b *Bucket) RemoveByUser(user uint64) {
	b.lock.Lock()
	defer b.lock.Unlock()
	b.cache.Reset()
	common.Success(b.db.Exec(`DELETE FROM items WHERE user = ?`, user))
}

func (b *Bucket) RemoveByUserGroup(userGroup uint64) {
	b.lock.Lock()
	defer b.lock.Unlock()
	b.cache.Reset()
	common.Success(b.db.Exec(`DELETE FROM items WHERE user_group = ?`, userGroup))
}

func (b *Bucket) RemoveBySrcGroup(srcGroup uint64) {
	b.lock.Lock()
	defer b.lock.Unlock()
	b.cache.Reset()
	common.Success(b.db.Exec(`DELETE FROM items WHERE src_group = ?`, srcGroup))
}

func (b *Bucket) RemoveByExpires() {
	b.lock.Lock()
	defer b.lock.Unlock()
	b.cache.Reset()
	common.Success(b.db.Exec(`DELETE FROM items WHERE expires < ?`, time.Now().UnixMilli()))
}

// func (b *Bucket) removeFromCache(match func(*Item) bool) {
// 	// TODO 可以优化缓存性能 b.cache.Reset()
// }

func (b *Bucket) Add(item *Item) {
	_, err := b.db.Exec(`
	INSERT INTO item (
		h1, h2, block, offset, header_len, body_len, user_group, 
		user, root_domain, domain, src_group, expires, path, tags
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		item.Key.H1, item.Key.H2, item.Block, item.Offset, item.HeaderLen,
		item.BodyLen, item.UserGroup, item.User, item.RootDomain, item.Domain,
		item.SrcGroup, item.Expires, item.Path, item.Tags)
	if err != nil {
		panic(err)
	}

}

func (b *Bucket) Get(k Key) *Item {
	b.lock.RLock()
	defer b.lock.RUnlock()

	buf, err := b.cache.Get(k.String())
	if err == nil && buf != nil {
		return ItemDecode(buf)
	}

	query := "SELECT block, offset, header_len, body_len, user_group, user, root_domain, domain, src_group, expires, path, tags FROM items WHERE h1 = ? AND h2 = ? LIMIT 1"
	row := b.db.QueryRow(query, k.H1, k.H2)

	var path []byte
	item := &Item{}
	common.Success(row.Scan(&item.Block, &item.Offset, &item.HeaderLen,
		&item.BodyLen, &item.UserGroup, &item.User, &item.RootDomain,
		&item.Domain, &item.SrcGroup, &item.Expires, &path, &item.Tags))
	item.Path = make([]byte, len(path))
	copy(item.Path, path)

	item.Key = k
	b.cache.Set(k.String(), ItemEncode(item))

	return item

}

func createTable(db *sql.DB) {
	_, err := db.Exec(`
        CREATE TABLE IF NOT EXISTS items (
            h1 UNSIGNED BIG INT NOT NULL,
            h2 UNSIGNED BIG INT NOT NULL,
            block UNSIGNED BIG INT NOT NULL,
            offset UNSIGNED BIG INT NOT NULL,
            header_len UNSIGNED BIG INT NOT NULL,
            body_len UNSIGNED BIG INT NOT NULL,
            user_group UNSIGNED BIG INT NOT NULL,
            user UNSIGNED BIG INT NOT NULL,
            root_domain UNSIGNED BIG INT NOT NULL,
            domain UNSIGNED BIG INT NOT NULL,
            src_group UNSIGNED BIG INT NOT NULL,
            expires BIG INT NOT NULL,
            path TEXT NOT NULL,
            tags BIG INT NOT NULL,
            PRIMARY KEY (h1, h2)
        );
    `)
	if err != nil {
		panic(err)
	}
}
