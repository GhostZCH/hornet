package store

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type ItemInfo struct {
	ID1      int64
	ID2      int64
	ID       []byte // 16-byte ID字段
	Filename uint64 // 文件名的64位整数表示
	Offset   int64  // 文件中的偏移量
	Size     int64  // 数据的长度
}

func NewSQLLite(path string) {
	// 打开数据库
	db, err := sql.Open("sqlite3", "file:"+path+"cache=shared&mode=rwc")
	if err != nil {
		panic(err)
	}
	defer db.Close()

	// 创建表
	db.Exec(`DROP TABLE item_info`)
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS item_info (
		id1 BIGINT,
		id2 BIGINT,
		filename BIGINT,
		offset BIGINT,
		size BIGINT,
		url TEXT(256),
		PRIMARY KEY (id1, id2)
	);
	
	CREATE INDEX IF NOT EXISTS item_info_id_index ON item_info(id1,id2);
	DELETE FROM item_info;`)
	if err != nil {
		panic(err)
	}

	startTime := time.Now()

	// 开始事务
	tx, err := db.Begin()
	if err != nil {
		panic(err)
	}

	// 执行插入操作
	stmt, err := tx.Prepare("INSERT INTO item_info(id1, id2, filename, offset, size, url) VALUES (?, ?, ?, ?, ?, ?)")
	if err != nil {
		log.Fatal(err)
	}
	defer stmt.Close()

	n := 100000
	for i := 0; i < n; i++ {
		filename := int64(i)
		offset := int64(i * 10)
		size := int64(1024)
		url := fmt.Sprintf("%d_https//www.example.com/products/item1www.example.com/products/item1www.example.com/products/item1?id=%020d", i/10, i)

		_, err := stmt.Exec(int64(i), int64(99999999-i), filename, offset, size, url)
		if err != nil {
			log.Fatal(err)
		}
	}

	// 提交事务
	err = tx.Commit()
	if err != nil {
		log.Fatal(err)
	}

	// 输出耗时
	duration := time.Since(startTime)
	fmt.Printf("Insert %d rows took %s\n", n, duration)

	// 根据id查询
	start := time.Now()
	info := ItemInfo{}
	for i := 0; i < n; i++ {
		err := db.QueryRow("SELECT id1,id2,filename,offset,size FROM item_info WHERE id1 = ? AND id2 = ?", i, 99999999-i).Scan(&info.ID1, &info.ID2, &info.Filename, &info.Offset, &info.Size)
		if err != nil {
			panic(err)
		}
	}
	fmt.Printf("Query by id %d times takes %s\n", n, time.Since(start))

	// start = time.Now()
	// db.Exec(`DELETE FROM item_info WHERE url LIKE  '3_%'`)
	// fmt.Printf("delte 10%% of %d items takes %s\n", n, time.Since(start))

	start = time.Now()
	db.Exec(`DELETE FROM item_info WHERE url LIKE  '3_%'`)
	fmt.Printf("delte 10%% of %d items takes %s\n", n, time.Since(start))

	count := 0
	db.QueryRow("SELECT COUNT(*) FROM item_info ").Scan(&count)
	fmt.Printf("count %d \n", count)
}
