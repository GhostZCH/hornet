package main

import (
	"fmt"
	"hornet/common"
	"math/rand"
	"os"
	"strings"
	"syscall"
	"time"

	"github.com/valyala/fasthttp"
)

func mmap(path string, size int) (data []byte, err error) {
	var f *os.File
	f, err = os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0600)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	err = f.Truncate(int64(size))
	if err != nil {
		return nil, err
	}

	flag := syscall.PROT_READ | syscall.PROT_WRITE
	return syscall.Mmap(int(f.Fd()), 0, size, flag, syscall.MAP_SHARED)
}

func main() {
	rand.Seed(time.Now().UnixNano())

	// store.NewSQLLite("test/meta")
	path := "/dev/shm/random_file"
	info, err := os.Stat(path)
	if err != nil {
		panic(err)
	}

	data, err := mmap(path, int(info.Size()))
	if err != nil {
		panic(err)
	}

	logger := common.NewHourlyLogger("test/log/")

	// 创建 fasthttp 请求处理函数
	handler := func(ctx *fasthttp.RequestCtx) {
		switch string(ctx.Path()) {
		case "/":
			fmt.Fprintf(ctx, "Welcome to my website!")
		case "/ping":
			fmt.Fprintf(ctx, strings.Repeat("x", 4096))
		case "/file":
			ctx.SetContentType("text/plain")
			ctx.Response.Header.Set("Content-Length", "4096")
			off := 2000
			if rand.Intn(10) < 100 {
				// 90% in mem
				off = rand.Intn(int(info.Size() - 4096))
			}
			ctx.Write(data[off : off+4096])
		default:
			ctx.Error("Unsupported path", fasthttp.StatusBadRequest)
		}
		logger.WriteLog(&common.LogData{Url: ctx.Path()})

	}

	// 启动 fasthttp 服务器
	if err := fasthttp.ListenAndServe(":8080", handler); err != nil {
		fmt.Printf("Error when starting server: %s\n", err.Error())
	}
}
