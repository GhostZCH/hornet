package http

import (
	"bytes"
	"encoding/gob"
	"errors"
	"fmt"
	"hornet/common"
	"hornet/store"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/valyala/fasthttp"
)

type CacheServer struct {
	name      string
	addr      string
	adminAddr string
	// upstream string
	store  *store.Store
	logger *common.HourlyLogger
}

func NewCacheServer(conf *common.Config, store *store.Store, logger *common.HourlyLogger) *CacheServer {
	return &CacheServer{
		name:      conf.Common.Name,
		addr:      conf.Cache.Addr,
		adminAddr: conf.Cache.Admin,
		store:     store,
		logger:    logger,
	}
}

func (svr *CacheServer) cacheHandler(ctx *fasthttp.RequestCtx) {
	key := append(ctx.URI().Host(), ctx.URI().Path()...)
	k := store.GetKey(key)
	buf, headerSize := svr.store.Get(&k)
	if buf == nil {
		// upstream
		// resp.Header.VisitAll(func(key, value []byte) {
		// 	fmt.Printf("%s: %s\n", key, value)
		// })
	}

	dec := gob.NewDecoder(bytes.NewReader(buf[:headerSize]))
	var headers map[string]string
	common.Success(dec.Decode(&headers))
	for k, v := range headers {
		ctx.Response.Header.Add(k, v)
	}
	ctx.Write(buf[headerSize:])

	svr.logger.WriteLog(&common.LogData{Url: ctx.RequestURI()})
}

func (svr *CacheServer) Start() {
	handler := fasthttp.RequestHandler(svr.cacheHandler)

	// 启动 fasthttp 服务器
	if err := fasthttp.ListenAndServe(svr.addr, handler); err != nil {
		fmt.Printf("Error when starting server: %s\n", err.Error())
	}
}

func getMaxAge(header string) (int, error) {
	parts := strings.Split(header, ",")
	for _, part := range parts {
		kv := strings.SplitN(strings.TrimSpace(part), "=", 2)
		if kv[0] == "max-age" {
			return strconv.Atoi(kv[1])
		}
	}
	return 0, errors.New("max-age not found")
}

func getCacheMaxAge(resp *fasthttp.Response) int64 {
	b := resp.Header.Peek("Cache-Control")
	if b == nil || len(b) == 0 {
		return -1
	}

	cacheControl := string(b)
	re := regexp.MustCompile(`\bmax-age=(\d+)\b`)
	matches := re.FindStringSubmatch(cacheControl)
	if len(matches) < 2 {
		return -1
	}

	maxAgeStr := matches[1]
	maxAge, err := strconv.Atoi(maxAgeStr)
	if err != nil {
		return -1
	}

	return time.Now().UnixMilli() + int64(maxAge)
}
