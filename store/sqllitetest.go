package store

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type ItemInfo struct {
	ID       []byte // 16-byte ID字段
	Filename uint64 // 文件名的64位整数表示
	Offset   int64  // 文件中的偏移量
	Size     int64  // 数据的长度
}

func NewSQLLite(fileName string) {
	// 打开数据库
	db, err := sql.Open("sqlite3", "file:info.db?cache=shared&mode=rwc")
	if err != nil {
		panic(err)
	}
	defer db.Close()

	// 创建表
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS item_info (
		id BLOB PRIMARY KEY,
		filename INTEGER,
		offset INTEGER,
		size INTEGER
	);
	
	CREATE INDEX IF NOT EXISTS item_info_id_index ON item_info(id);
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
	stmt, err := tx.Prepare("INSERT INTO item_info(id, filename, offset, size) VALUES (?, ?, ?, ?)")
	if err != nil {
		log.Fatal(err)
	}
	defer stmt.Close()

	n := 100000
	for i := 0; i < n; i++ {
		id := fmt.Sprintf("%016x", i)
		filename := int64(i)
		offset := int64(i * 10)
		size := int64(1024)

		_, err := stmt.Exec(id, filename, offset, size)
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
		id := fmt.Sprintf("%016x", i)
		err := db.QueryRow("SELECT * FROM item_info WHERE id = ?", id).Scan(&info.ID, &info.Filename, &info.Offset, &info.Size)
		if err != nil {
			panic(err)
		}
	}
	fmt.Printf("Query by id %d times takes %s\n", n, time.Since(start))

}
