package http

import (
	"hornet/common"
	"net"
	"time"

	lru "github.com/hashicorp/golang-lru/v2"
	"github.com/valyala/fasthttp"
)

type Doer interface {
	Do(req *fasthttp.Request, resp *fasthttp.Response) error
}

type ProxyPool struct {
	pools *lru.ARCCache[string, Doer]
}

func NewProxyPool() *ProxyPool {
	p, e := lru.NewARC[string, Doer](10)
	common.Success(e)
	return &ProxyPool{pools: p}
}

func (pool *ProxyPool) getPool(addr string) Doer {
	p, ok := pool.pools.Get(addr)

	if !ok {
		var p Doer
		if common.IsIPPort(addr) {
			dialFunc := func(addr string) (net.Conn, error) {
				return fasthttp.DialTimeout(addr, 10*time.Second)
			}

			p = &fasthttp.HostClient{
				Addr:                addr,
				Dial:                dialFunc,
				MaxConns:            512,
				MaxIdleConnDuration: 2 * time.Second,
			}
		} else {

			p = &fasthttp.Client{
				MaxConnsPerHost:     512,
				MaxIdleConnDuration: 2 * time.Second,
			}
		}
		pool.pools.Add(addr, p)
	}

	return p
}

func (pool *ProxyPool) Get(proxyAddr string, ctx *fasthttp.RequestCtx) *fasthttp.Response {

	// 创建请求
	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)

	ctx.Request.Header.CopyTo(&req.Header)
	req.SetBody(ctx.Request.Body())

	// 发送请求并接收响应
	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(resp)

	p := pool.getPool(proxyAddr)
	common.Success(p.Do(req, resp))

	// 将响应写回客户端
	resp.Header.CopyTo(&ctx.Response.Header)
	ctx.Write(resp.Body())

	return &ctx.Response
}

// func getMaxAge(header string) (int, error) {
// 	parts := strings.Split(header, ",")
// 	for _, part := range parts {
// 		kv := strings.SplitN(strings.TrimSpace(part), "=", 2)
// 		if kv[0] == "max-age" {
// 			return strconv.Atoi(kv[1])
// 		}
// 	}
// 	return 0, errors.New("max-age not found")
// }

// func getCacheMaxAge(resp *fasthttp.Response) int64 {
// 	b := resp.Header.Peek("Cache-Control")
// 	if b == nil || len(b) == 0 {
// 		return -1
// 	}

// 	cacheControl := string(b)
// 	re := regexp.MustCompile(`\bmax-age=(\d+)\b`)
// 	matches := re.FindStringSubmatch(cacheControl)
// 	if len(matches) < 2 {
// 		return -1
// 	}

// 	maxAgeStr := matches[1]
// 	maxAge, err := strconv.Atoi(maxAgeStr)
// 	if err != nil {
// 		return -1
// 	}

// 	return time.Now().UnixMilli() + int64(maxAge)
// }
