package main

import (
	"bytes"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/JasonKhew96/biliroaming-go-server/entity"
	"github.com/mailru/easyjson"
	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/fasthttpproxy"
)

type HttpCookiesParams struct {
	Key   []byte
	Value []byte
}

type HttpRequestParams struct {
	Method    []byte
	Url       []byte
	UserAgent []byte
	Cookie    []HttpCookiesParams
}

type ErrorHttpStatus struct {
	Code    int
	Message string
}

func (e *ErrorHttpStatus) Error() string {
	return fmt.Sprintf("status code error %d with message %s", e.Code, e.Message)
}

func (e *ErrorHttpStatus) Is(tgt error) bool {
	target, ok := tgt.(*ErrorHttpStatus)
	if !ok {
		return false
	}
	return e.Code == target.Code
}

func NewErrorHttpLimited(code int) *ErrorHttpStatus {
	return &ErrorHttpStatus{
		Code: code,
	}
}

var (
	ErrorHttpStatusLimited = NewErrorHttpLimited(-412)
)

func (b *BiliroamingGo) initProxy(c *Config) {
	b.cnClient = b.newClient(c.Proxy.CN)
	b.hkClient = b.newClient(c.Proxy.HK)
	b.twClient = b.newClient(c.Proxy.TW)
	b.thClient = b.newClient(c.Proxy.TH)
	b.defaultClient = b.newClient(c.Proxy.Default)
}

func (b *BiliroamingGo) newClient(proxy string) *fasthttp.Client {
	var dialFunc fasthttp.DialFunc
	switch {
	case strings.HasPrefix(proxy, "socks5://"), strings.HasPrefix(proxy, "socks5h://"):
		b.sugar.Debug("New socks proxy client: ", proxy)
		dialFunc = fasthttpproxy.FasthttpSocksDialer(proxy)
	case proxy != "":
		b.sugar.Debug("New http proxy client: ", proxy)
		dialFunc = fasthttpproxy.FasthttpHTTPDialer(proxy)
	case proxy == "":
		b.sugar.Debug("New normal client")
		dialFunc = nil
	}

	return &fasthttp.Client{
		ReadTimeout:   10 * time.Second,
		WriteTimeout:  10 * time.Second,
		Dial:          dialFunc,
		DialDualStack: b.config.IPV6,
	}
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
		return b.config.Reverse.CN
	case "hk":
		return b.config.Reverse.HK
	case "tw":
		return b.config.Reverse.TW
	case "th":
		return b.config.Reverse.TH
	default:
		return ""
	}
}

func (b *BiliroamingGo) getReverseSearchProxyByArea(area string) string {
	switch strings.ToLower(area) {
	case "cn":
		return b.config.ReverseSearch.CN
	case "hk":
		return b.config.ReverseSearch.HK
	case "tw":
		return b.config.ReverseSearch.TW
	case "th":
		return b.config.ReverseSearch.TH
	default:
		return ""
	}
}

func (b *BiliroamingGo) getReverseWebSearchProxyByArea(area string) string {
	switch strings.ToLower(area) {
	case "cn":
		return b.config.ReverseWebSearch.CN
	case "hk":
		return b.config.ReverseWebSearch.HK
	case "tw":
		return b.config.ReverseWebSearch.TW
	default:
		return ""
	}
}

func setDefaultHeaders(ctx *fasthttp.RequestCtx) {
	ctx.Response.Header.SetBytesKV([]byte("Access-Control-Allow-Origin"), []byte("https://www.bilibili.com"))
	ctx.Response.Header.SetBytesKV([]byte("Access-Control-Allow-Credentials"), []byte("true"))
	ctx.SetContentTypeBytes([]byte("application/json"))
}

func writeErrorJSON(ctx *fasthttp.RequestCtx, code int, msg string) {
	setDefaultHeaders(ctx)
	resp := &entity.SimpleResponse{
		Code:    code,
		Message: fmt.Sprintf("解析服务器: %s", msg),
	}
	respData, err := easyjson.Marshal(resp)
	if err != nil {
		ctx.Write([]byte(`{"code":500,"message":"解析服务器发送错误"}`))
		return
	}
	ctx.Write(respData)
	// ctx.Write([]byte(`{"accept_format":"mp4","code":0,"seek_param":"start","is_preview":0,"fnval":1,"video_project":true,"fnver":0,"type":"MP4","bp":0,"result":"suee","seek_type":"offset","qn_extras":[{"attribute":0,"icon":"http://i0.hdslb.com/bfs/app/81dab3a04370aafa93525053c4e760ac834fcc2f.png","icon2":"http://i0.hdslb.com/bfs/app/4e6f14c2806f7cc508d8b6f5f1d8306f94a71ecc.png","need_login":true,"need_vip":true,"qn":112},{"attribute":0,"icon":"","icon2":"","need_login":false,"need_vip":false,"qn":80},{"attribute":0,"icon":"","icon2":"","need_login":false,"need_vip":false,"qn":64},{"attribute":0,"icon":"","icon2":"","need_login":false,"need_vip":false,"qn":32},{"attribute":0,"icon":"","icon2":"","need_login":false,"need_vip":false,"qn":16}],"accept_watermark":[false,false,false,false,false],"from":"local","video_codecid":7,"durl":[{"order":1,"length":16740,"size":172775,"ahead":"","vhead":"","url":"https://s1.hdslb.com/bfs/static/player/media/error.mp4","backup_url":[]}],"no_rexcode":0,"format":"mp4","support_formats":[{"display_desc":"360P","superscript":"","format":"mp4","description":"流畅 360P","quality":16,"new_description":"360P 流畅"}],"message":"","accept_quality":[16],"quality":16,"timelength":16740,"has_paid":false,"accept_description":["流畅 360P"],"status":2}`))
}

func writeHealthJSON(ctx *fasthttp.RequestCtx, health *entity.Health) {
	setDefaultHeaders(ctx)
	if health == nil {
		ctx.Write([]byte(`{"code":500,"message":"解析服务器发送错误"}`))
		return
	}
	respData, err := easyjson.Marshal(health)
	if err != nil {
		ctx.Write([]byte(`{"code":500,"message":"解析服务器发送错误"}`))
		return
	}
	ctx.Write(respData)
}

func getClientPlatform(ctx *fasthttp.RequestCtx, appkey string) ClientType {
	platform := string(ctx.Request.Header.PeekBytes([]byte("platform-from-biliroaming")))
	if platform == "" && appkey == "" {
		return ClientTypeIphone
	}
	clientType := ClientType(platform)
	if clientType.IsValid() {
		return clientType
	}
	if appkey != "" {
		return getClientTypeFromAppkey(appkey)
	}
	return ClientTypeUnknown
}

func (b *BiliroamingGo) checkRoamingVer(ctx *fasthttp.RequestCtx) bool {
	versionCode := ctx.Request.Header.PeekBytes([]byte("build"))
	versionName := ctx.Request.Header.PeekBytes([]byte("x-from-biliroaming"))

	if len(versionCode) == 0 && len(versionName) == 0 {
		return true
	}

	if len(versionCode) > 0 && len(versionName) > 0 {
		build, err := strconv.Atoi(string(versionCode))
		if err != nil {
			writeErrorJSON(ctx, ERROR_CODE_HEADER_WRONG, MSG_ERROR_HEADER_WRONG)
			return false
		}
		if build < b.config.RoamingMinVer {
			writeErrorJSON(ctx, ERROR_CODE_HEADER_MIN_VERSION, MSG_ERROR_HEADER_MIN_VERSION)
			return false
		}
		return true
	}

	writeErrorJSON(ctx, ERROR_CODE_HEADER_WRONG, MSG_ERROR_HEADER_WRONG)
	return false
}

func (b *BiliroamingGo) doRequest(client *fasthttp.Client, params *HttpRequestParams) ([]byte, error) {
	if params == nil {
		return nil, errors.New("params is nil")
	}
	if params.Url == nil {
		return nil, errors.New("url is empty")
	}
	if params.UserAgent == nil {
		return nil, errors.New("user agent is empty")
	}
	if params.Method == nil {
		params.Method = []byte(fasthttp.MethodGet)
	}
	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)
	req.SetRequestURIBytes(params.Url)
	req.Header.SetBytesKV([]byte("Accept-Encoding"), []byte("br, gzip, deflate"))
	req.Header.SetUserAgentBytes(params.UserAgent)
	req.Header.SetMethodBytes(params.Method)
	if params.Cookie != nil {
		for _, cookie := range params.Cookie {
			req.Header.SetCookieBytesKV(cookie.Key, cookie.Value)
		}
	}

	b.sugar.Debugf("doRequest: %s", req.RequestURI())

	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(resp)

	err := client.DoRedirects(req, resp, 1)
	if err != nil {
		return nil, err
	}

	b.sugar.Debugf("doRedirects: %d", resp.StatusCode())

	if resp.StatusCode() != fasthttp.StatusOK {
		return nil, NewErrorHttpLimited(resp.StatusCode())
	}

	contentEncoding := resp.Header.Peek("Content-Encoding")
	var bodyBytes []byte
	if bytes.EqualFold(contentEncoding, []byte("gzip")) {
		bodyBytes, err = resp.BodyGunzip()
	} else if bytes.EqualFold(contentEncoding, []byte("br")) {
		bodyBytes, err = resp.BodyUnbrotli()
	} else if bytes.EqualFold(contentEncoding, []byte("deflate")) {
		bodyBytes, err = resp.BodyInflate()
	} else {
		bodyBytes = resp.Body()
	}

	if err != nil {
		return nil, err
	}

	if isLimited, err := isResponseLimited(bodyBytes); err != nil {
		return nil, err
	} else if isLimited {
		return nil, ErrorHttpStatusLimited
	}

	b.sugar.Debug("Content: ", string(bodyBytes))

	return bodyBytes, nil
}

func (b *BiliroamingGo) doRequestJson(client *fasthttp.Client, params *HttpRequestParams) ([]byte, error) {
	if params == nil {
		return nil, errors.New("params is nil")
	}
	if params.Url == nil {
		return nil, errors.New("url is empty")
	}
	if params.UserAgent == nil {
		return nil, errors.New("user agent is empty")
	}
	if params.Method == nil {
		params.Method = []byte(fasthttp.MethodGet)
	}
	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)
	req.SetRequestURIBytes(params.Url)
	req.Header.SetBytesKV([]byte("Accept-Encoding"), []byte("br, gzip, deflate"))
	req.Header.SetUserAgentBytes(params.UserAgent)
	req.Header.SetMethodBytes(params.Method)
	if params.Cookie != nil {
		for _, cookie := range params.Cookie {
			req.Header.SetCookieBytesKV(cookie.Key, cookie.Value)
		}
	}

	b.sugar.Debugf("doRequest: %s", req.RequestURI())

	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(resp)

	err := client.DoRedirects(req, resp, 3)
	if err != nil {
		return nil, err
	}

	b.sugar.Debugf("doRedirects: %d", resp.StatusCode())

	if resp.StatusCode() != fasthttp.StatusOK {
		return nil, NewErrorHttpLimited(resp.StatusCode())
	}

	// Verify the content type
	contentType := resp.Header.Peek("Content-Type")
	if bytes.Index(contentType, []byte("application/json")) != 0 {
		return nil, fmt.Errorf("expected content-type json but %s", string(contentType))
	}

	contentEncoding := resp.Header.Peek("Content-Encoding")
	var bodyBytes []byte
	if bytes.EqualFold(contentEncoding, []byte("gzip")) {
		bodyBytes, err = resp.BodyGunzip()
	} else if bytes.EqualFold(contentEncoding, []byte("br")) {
		bodyBytes, err = resp.BodyUnbrotli()
	} else if bytes.EqualFold(contentEncoding, []byte("deflate")) {
		bodyBytes, err = resp.BodyInflate()
	} else {
		bodyBytes = resp.Body()
	}

	if err != nil {
		return nil, err
	}

	if isLimited, err := isResponseLimited(bodyBytes); err != nil {
		return nil, err
	} else if isLimited {
		return nil, ErrorHttpStatusLimited
	}

	body := string(bodyBytes)

	b.sugar.Debug("Content: ", body)

	// Remove mid from json content
	// if strings.Contains(url, "/playurl?") {
	// 	body = removeMid(body)
	// 	b.sugar.Debug("New content: ", body)
	// }

	return []byte(body), nil
}

func processNotFound(ctx *fasthttp.RequestCtx) {
	ctx.Error(fasthttp.StatusMessage(fasthttp.StatusNotFound), fasthttp.StatusNotFound)
}

func (b *BiliroamingGo) processError(ctx *fasthttp.RequestCtx, err error) {
	if !errors.Is(err, fasthttp.ErrTimeout) && !errors.Is(err, fasthttp.ErrTLSHandshakeTimeout) && !errors.Is(err, fasthttp.ErrConnectionClosed) {
		b.sugar.Error(err)
	}
	writeErrorJSON(ctx, ERROR_CODE_INTERNAL_SERVER, MSG_ERROR_INTERNAL_SERVER)
}
