package http

import (
	"bytes"
	"encoding/gob"
	"hornet/common"
	"hornet/store"
	"time"

	"github.com/valyala/fasthttp"
)

type CacheServer struct {
	name     string
	addr     string
	admin    string
	store    *store.Store
	logger   *common.HourlyLogger
	upstream *ProxyPool
}

func NewCacheServer(conf *common.Config, store *store.Store, logger *common.HourlyLogger) *CacheServer {
	return &CacheServer{
		name:     conf.Common.Name,
		addr:     conf.Cache.Addr,
		admin:    conf.Cache.Admin,
		store:    store,
		logger:   logger,
		upstream: NewProxyPool(),
	}
}

func (svr *CacheServer) adminHandler(ctx *fasthttp.RequestCtx) {
	if "DELETE" == string(ctx.Method()) {
		args := make([]*store.RemoveArg, 0)
		ctx.QueryArgs().VisitAll(func(key, value []byte) {
			args = append(args, &store.RemoveArg{Cmd: string(key), Val: string(value)})
		})
		svr.store.Del()
	}
}

func (svr *CacheServer) cacheHandler(ctx *fasthttp.RequestCtx) {
	key := append(ctx.URI().Host(), ctx.URI().Path()...)
	k := store.NewKey(key)

	buf, headerSize, level := svr.store.Get(k)
	if buf == nil {
		// TODO 从req中读取特定参数覆盖原有参数
		resp := svr.upstream.Get(string(ctx.Host()), ctx)
		item, buffer := toItem(k, ctx.Host(), ctx.URI().Path(), resp)
		svr.store.Put(item, buffer)
	} else {
		dec := gob.NewDecoder(bytes.NewReader(buf[:headerSize]))
		headers := make([]Pair, 0)
		common.Success(dec.Decode(&headers))
		for _, p := range headers {
			ctx.Response.Header.AddBytesKV(p.Key, p.Val)
		}
		common.Success(ctx.Write(buf[headerSize:]))
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
	common.Success(enc.Encode(headers))
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
	common.Success(fasthttp.ListenAndServe(svr.addr, svr.cacheHandler))
	common.Success(fasthttp.ListenAndServe(svr.admin, svr.adminHandler))
}
