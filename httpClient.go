package main

import (
	"bytes"
	"strings"

	"github.com/JasonKhew96/biliroaming-go-server/response"
	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/fasthttpproxy"
)

func (b *BiliroamingGo) initProxy(c *Config) (*fasthttp.Client, *fasthttp.Client, *fasthttp.Client, *fasthttp.Client, *fasthttp.Client) {
	cnClient := b.newClient(c.ProxyCN)
	hkClient := b.newClient(c.ProxyHK)
	twClient := b.newClient(c.ProxyTW)
	thClient := b.newClient(c.ProxyTH)
	defaultClient := &fasthttp.Client{}
	return cnClient, hkClient, twClient, thClient, defaultClient
}

func (b *BiliroamingGo) newClient(proxy string) *fasthttp.Client {
	if proxy != "" {
		b.sugar.Debug("New socks proxy client: ", proxy)
		return &fasthttp.Client{
			Dial: fasthttpproxy.FasthttpSocksDialer(proxy),
		}
	}
	b.sugar.Debug("New normal client")
	return &fasthttp.Client{}
}

func (b *BiliroamingGo) getClientByArea(area string) *fasthttp.Client {
	switch strings.ToLower(area) {
	case "cn":
		return b.cnClient
	case "hk":
		return b.hkClient
	case "tw":
		return b.twClient
	case "th":
		return b.thClient
	default:
		return b.defaultClient
	}
}

func (b *BiliroamingGo) getReverseProxyByArea(area string) string {
	switch strings.ToLower(area) {
	case "cn":
		return b.config.ReverseCN
	case "hk":
		return b.config.ReverseHK
	case "tw":
		return b.config.ReverseTW
	case "th":
		return b.config.ReverseTH
	default:
		return ""
	}
}

func setDefaultHeaders(ctx *fasthttp.RequestCtx) {
	ctx.Response.Header.SetBytesKV([]byte("Access-Control-Allow-Origin"), []byte("https://www.bilibili.com"))
	ctx.Response.Header.SetBytesKV([]byte("Access-Control-Allow-Credentials"), []byte("true"))
	ctx.Response.Header.SetBytesKV([]byte("Server"), []byte("Potato"))
	ctx.SetContentTypeBytes([]byte("application/json"))
}

func writeErrorJSON(ctx *fasthttp.RequestCtx, code int, msg []byte) {
	setDefaultHeaders(ctx)
	resp := &response.SimpleResponse{
		Code:    code,
		Message: string(msg),
		TTL:     1,
	}
	respData, err := resp.MarshalJSON()
	if err != nil {
		ctx.Write([]byte(`{"code":-500,"message":"服务器错误","ttl":1}`))
		return
	}
	ctx.Write(respData)
	// ctx.Write([]byte(`{"accept_format":"mp4","code":0,"seek_param":"start","is_preview":0,"fnval":1,"video_project":true,"fnver":0,"type":"MP4","bp":0,"result":"suee","seek_type":"offset","qn_extras":[{"attribute":0,"icon":"http://i0.hdslb.com/bfs/app/81dab3a04370aafa93525053c4e760ac834fcc2f.png","icon2":"http://i0.hdslb.com/bfs/app/4e6f14c2806f7cc508d8b6f5f1d8306f94a71ecc.png","need_login":true,"need_vip":true,"qn":112},{"attribute":0,"icon":"","icon2":"","need_login":false,"need_vip":false,"qn":80},{"attribute":0,"icon":"","icon2":"","need_login":false,"need_vip":false,"qn":64},{"attribute":0,"icon":"","icon2":"","need_login":false,"need_vip":false,"qn":32},{"attribute":0,"icon":"","icon2":"","need_login":false,"need_vip":false,"qn":16}],"accept_watermark":[false,false,false,false,false],"from":"local","video_codecid":7,"durl":[{"order":1,"length":16740,"size":172775,"ahead":"","vhead":"","url":"https://s1.hdslb.com/bfs/static/player/media/error.mp4","backup_url":[]}],"no_rexcode":0,"format":"mp4","support_formats":[{"display_desc":"360P","superscript":"","format":"mp4","description":"流畅 360P","quality":16,"new_description":"360P 流畅"}],"message":"","accept_quality":[16],"quality":16,"timelength":16740,"has_paid":false,"accept_description":["流畅 360P"],"status":2}`))
}

func (b *BiliroamingGo) doRequest(ctx *fasthttp.RequestCtx, client *fasthttp.Client, url string) []byte {
	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)
	req.Header.SetUserAgentBytes(ctx.UserAgent())
	req.SetRequestURI(url)

	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(resp)

	err := client.Do(req, resp)
	if err != nil {
		b.processError(ctx, err)
		return nil
	}

	if resp.StatusCode() != fasthttp.StatusOK {
		b.processError(ctx, err)
		return nil
	}

	// Verify the content type
	contentType := resp.Header.Peek("Content-Type")
	if bytes.Index(contentType, []byte("application/json")) != 0 {
		b.processError(ctx, err)
		return nil
	}

	// Do we need to decompress the response?
	contentEncoding := resp.Header.Peek("Content-Encoding")
	var body []byte
	if bytes.EqualFold(contentEncoding, []byte("gzip")) {
		body, _ = resp.BodyGunzip()
	} else {
		body = resp.Body()
	}

	b.sugar.Debug("Content: ", string(body))

	// Remove mid from json content
	s := reMid.FindAllString(string(body), 1)
	if len(s) > 0 {
		body = []byte(strings.ReplaceAll(string(body), s[0], ""))
		b.sugar.Debug("New content: ", string(body))
	}

	setDefaultHeaders(ctx)
	ctx.Write(body)
	return body
}
