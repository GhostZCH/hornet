package http

import (
	"errors"
	"hornet/common"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/valyala/fasthttp"
)

func GoSource(ctx *fasthttp.RequestCtx) *fasthttp.Response {
	client := &fasthttp.Client{}

	// 创建请求
	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)

	ctx.Request.Header.CopyTo(&req.Header)
	req.SetBody(ctx.Request.Body())

	// 发送请求并接收响应
	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(resp)
	common.Success(client.Do(req, resp))

	// 将响应写回客户端
	resp.Header.CopyTo(&ctx.Response.Header)
	ctx.Write(resp.Body())

	return &ctx.Response
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
