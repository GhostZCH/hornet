package http

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"hornet/common"
	"hornet/store"
	"time"

	"github.com/valyala/fasthttp"
)

type CacheServer struct {
	name      string
	addr      string
	adminAddr string
	store     *store.Store
	logger    *common.HourlyLogger
	upstream  *ProxyPool
}

func NewCacheServer(conf *common.Config, store *store.Store, logger *common.HourlyLogger) *CacheServer {
	return &CacheServer{
		name:      conf.Common.Name,
		addr:      conf.Cache.Addr,
		adminAddr: conf.Cache.Admin,
		store:     store,
		logger:    logger,
		upstream:  NewProxyPool(),
	}
}

func (svr *CacheServer) cacheHandler(ctx *fasthttp.RequestCtx) {
	key := append(ctx.URI().Host(), ctx.URI().Path()...)
	k := store.GetKey(key)

	buf, headerSize, level := svr.store.Get(&k)
	if buf == nil {
		// TODO 从req中读取特定参数覆盖原有参数
		resp := svr.upstream.Get(string(ctx.Host()), ctx)
		item, buffer := toItem(&k, ctx.Host(), ctx.URI().Path(), resp)
		svr.store.Put(item, buffer)
	} else {
		dec := gob.NewDecoder(bytes.NewReader(buf[:headerSize]))
		headers := make([]Pair, 0)
		common.Success(dec.Decode(&headers))
		for _, p := range headers {
			ctx.Response.Header.AddBytesKV(p.Key, p.Val)
		}
		ctx.Write(buf[headerSize:])
	}

	svr.logger.WriteLog(&common.LogData{Url: ctx.RequestURI(), Hit: buf != nil, Level: level})
}

type Pair struct {
	Key []byte
	Val []byte
}

func toItem(k *store.Key, host []byte, path []byte, resp *fasthttp.Response) (item *store.Item, buf []byte) {
	headers := make([]Pair, 0)
	resp.Header.VisitAll(func(k []byte, v []byte) {
		headers = append(headers, Pair{Key: k, Val: v})
	})

	tmp := bytes.Buffer{}
	enc := gob.NewEncoder(&tmp)
	enc.Encode(headers)
	tmpByte := tmp.Bytes()
	buf = append(tmpByte, resp.Body()...)

	item = &store.Item{
		Key:        *k,
		Expires:    time.Now().Unix() + 3600,
		HeaderLen:  int64(len(tmpByte)),
		BodyLen:    int64(len(resp.Body())),
		UserGroup:  0,
		User:       0,
		RootDomain: 0,
		Domain:     common.Hash64(host),
		SrcGroup:   0,
		Path:       path,
		Tags:       0,
	}

	return item, buf
}

func (svr *CacheServer) Start() {
	handler := fasthttp.RequestHandler(svr.cacheHandler)

	// 启动 fasthttp 服务器
	if err := fasthttp.ListenAndServe(svr.addr, handler); err != nil {
		fmt.Printf("Error when starting server: %s\n", err.Error())
	}
}
