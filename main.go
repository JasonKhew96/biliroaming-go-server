package main

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/valyala/fasthttp"
	"go.uber.org/zap"
	"golang.org/x/net/idna"
	"golang.org/x/time/rate"
	"gopkg.in/yaml.v2"
)

// ip string
type visitor struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// Config ...
type Config struct {
	Debug                 bool   `yaml:"debug"`                    // 调试模式
	Port                  int    `yaml:"port"`                     // 端口
	GlobalLimit           int    `yaml:"global_limit"`             // 每秒全局限制次数
	GlobalBurst           int    `yaml:"global_burst"`             // 每秒全局突发次数
	IPLimit               int    `yaml:"ip_limit"`                 // 每秒限制次数
	IPBurst               int    `yaml:"ip_burst"`                 // 每秒突发次数
	RedisAddr             string `yaml:"redis_address"`            // redis 地址
	RedisPwd              string `yaml:"redis_password"`           // redis 密码
	AccessKeyMaxCacheTime int    `yaml:"accesskey_max_cache_time"` // accessKey 缓存（天）
	PlayurlCacheTime      int    `yaml:"playurl_cache_time"`       // 播放链接缓存（分钟）
	// 代理(留空禁用)(优先)
	ProxyCN string `yaml:"proxy_cn"`
	ProxyHK string `yaml:"proxy_hk"`
	ProxyTW string `yaml:"proxy_tw"`
	ProxyTH string `yaml:"proxy_th"`
	// 反代(留空禁用)
	ReverseCN string `yaml:"reverse_cn"`
	ReverseHK string `yaml:"reverse_hk"`
	ReverseTW string `yaml:"reverse_tw"`
	ReverseTH string `yaml:"reverse_th"`
}

// BiliroamingGo ...
type BiliroamingGo struct {
	config        *Config
	globalLimiter *rate.Limiter
	visitors      map[string]*visitor
	vMu           sync.RWMutex
	ctx           context.Context
	logger        *zap.Logger
	sugar         *zap.SugaredLogger

	cnClient      *fasthttp.Client
	hkClient      *fasthttp.Client
	twClient      *fasthttp.Client
	thClient      *fasthttp.Client
	defaultClient *fasthttp.Client
}

var reMid = regexp.MustCompile(`(&|\\u0026)mid=\d+`)

// get visitor limiter
func (b *BiliroamingGo) getVisitor(ip string) *rate.Limiter {
	b.vMu.Lock()
	defer b.vMu.Unlock()
	u, exists := b.visitors[ip]
	if !exists {
		uLimiter := rate.NewLimiter(rate.Limit(b.config.IPLimit), b.config.IPBurst)
		b.visitors[ip] = &visitor{
			limiter: uLimiter,
		}
		return uLimiter
	}

	u.lastSeen = time.Now()
	return u.limiter
}

func (b *BiliroamingGo) cleanupVisitors() {
	for {
		time.Sleep(time.Minute)

		b.vMu.Lock()
		for ip, v := range b.visitors {
			if time.Since(v.lastSeen) > 5*time.Minute {
				delete(b.visitors, ip)
			}
		}
		b.vMu.Unlock()
	}
}

func main() {
	// default config
	c := &Config{
		Debug:                 false,
		Port:                  23333,
		GlobalLimit:           4,
		GlobalBurst:           8,
		IPLimit:               2,
		IPBurst:               4,
		AccessKeyMaxCacheTime: 7,
		PlayurlCacheTime:      60,
	}
	data, err := ioutil.ReadFile("config.yaml")
	if err != nil {
		data, err = yaml.Marshal(c)
		if err != nil {
			panic(err)
		}
		err = ioutil.WriteFile("config.yaml", data, os.ModePerm)
		if err != nil {
			panic(err)
		}
	} else {
		err = yaml.Unmarshal(data, c)
		if err != nil {
			panic(err)
		}
	}

	var logger *zap.Logger
	if c.Debug {
		logger, err = zap.NewDevelopment()
		if err != nil {
			panic(err)
		}
	} else {
		logger, err = zap.NewProduction()
		if err != nil {
			panic(err)
		}
	}
	sugar := logger.Sugar()

	b := BiliroamingGo{
		config:        c,
		globalLimiter: rate.NewLimiter(rate.Limit(c.GlobalLimit), c.GlobalBurst),
		visitors:      make(map[string]*visitor),
		ctx:           context.Background(),
		logger:        logger,
		sugar:         sugar,
	}

	cnClient, hkClient, twClient, thClient, defaultClient := b.initProxy(b.config)
	b.cnClient = cnClient
	b.hkClient = hkClient
	b.twClient = twClient
	b.thClient = thClient
	b.defaultClient = defaultClient

	go b.cleanupVisitors()

	mux := func(ctx *fasthttp.RequestCtx) {
		switch string(ctx.Path()) {
		case "/pgc/player/web/playurl": // web
			b.handleWebPlayURL(ctx)
		case "/pgc/player/api/playurl": // android
			b.handleAndroidPlayURL(ctx)
		// case "/intl/gateway/v2/app/search/type": // bstar android
		case "/intl/gateway/v2/ogv/view/app/season": // bstar android
			b.handleBstarAndroidSeason(ctx)
		case "/intl/gateway/v2/app/subtitle": // bstar android
			b.handleBstarAndroidSubtitle(ctx)
		case "/intl/gateway/v2/ogv/playurl": // bstar android
			b.handleBstarAndroidPlayURL(ctx)
		default:
			ctx.Error("Not Found", fasthttp.StatusNotFound)
		}
	}

	sugar.Infof("Listening on :%d ...", c.Port)
	err = fasthttp.ListenAndServe(":"+strconv.Itoa(c.Port), mux)
	if err != nil {
		sugar.Panic(err)
	}
}

func (b *BiliroamingGo) processError(ctx *fasthttp.RequestCtx, err error) {
	b.sugar.Error(err)
	ctx.Error(
		fasthttp.StatusMessage(fasthttp.StatusInternalServerError),
		fasthttp.StatusInternalServerError,
	)
}

func (b *BiliroamingGo) processArgs(args *fasthttp.Args) (string, string, string, string, string) {
	bAccessKey := args.Peek("access_key")
	bArea := args.Peek("area")
	bCID := args.Peek("cid")
	bEpID := args.Peek("ep_id")
	bSeasonID := args.Peek("season_id")
	b.sugar.Debug("Request args", args.String())
	b.sugar.Debugf(
		"Parsed request args: access_key: %s, area: %s, cid: %s, ep_id: %s, season_id: %s",
		string(bAccessKey), string(bArea), string(bCID), string(bEpID), string(bSeasonID),
	)
	return string(bAccessKey), string(bArea), string(bCID), string(bEpID), string(bSeasonID)
}

func (b *BiliroamingGo) doRequest(ctx *fasthttp.RequestCtx, client *fasthttp.Client, url string) {
	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)
	req.Header.SetUserAgentBytes(ctx.UserAgent())
	req.SetRequestURI(url)

	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(resp)

	err := client.Do(req, resp)
	if err != nil {
		b.processError(ctx, err)
		return
	}

	if resp.StatusCode() != fasthttp.StatusOK {
		b.processError(ctx, err)
		return
	}

	// Verify the content type
	contentType := resp.Header.Peek("Content-Type")
	if bytes.Index(contentType, []byte("application/json")) != 0 {
		b.processError(ctx, err)
		return
	}

	// Do we need to decompress the response?
	contentEncoding := resp.Header.Peek("Content-Encoding")
	var body []byte
	if bytes.EqualFold(contentEncoding, []byte("gzip")) {
		fmt.Println("Unzipping...")
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
}

func (b *BiliroamingGo) handleWebPlayURL(ctx *fasthttp.RequestCtx) {
	queryArgs := ctx.URI().QueryArgs()
	accessKey, area, cid, epid, _ := b.processArgs(queryArgs)
	client := b.getClientByArea(area)

	v := url.Values{}
	v.Set("access_key", accessKey)
	v.Set("area", area)
	v.Set("cid", cid)
	v.Set("ep_id", epid)
	v.Set("fnver", "0")
	v.Set("fnval", "80")
	v.Set("fourk", "1")
	// v.Set("qn", "120")

	params, err := SignParams(v, ClientTypeAndroid)
	if err != nil {
		b.sugar.Error(err)
		ctx.Error(
			fasthttp.StatusMessage(fasthttp.StatusInternalServerError),
			fasthttp.StatusInternalServerError,
		)
		return
	}

	reverseProxy := b.getReverseProxyByArea(area)
	if reverseProxy == "" {
		reverseProxy = "api.bilibili.com"
	}
	domain, err := idna.New().ToASCII(reverseProxy)
	if err != nil {
		b.sugar.Error(err)
		ctx.Error(
			fasthttp.StatusMessage(fasthttp.StatusInternalServerError),
			fasthttp.StatusInternalServerError,
		)
		return
	}

	url := fmt.Sprintf("https://%s/pgc/player/web/playurl?%s", domain, params)
	b.sugar.Debug("New url: ", url)

	b.doRequest(ctx, client, url)
}

func (b *BiliroamingGo) handleAndroidPlayURL(ctx *fasthttp.RequestCtx) {
	queryArgs := ctx.URI().QueryArgs()
	accessKey, area, cid, epid, _ := b.processArgs(queryArgs)
	client := b.getClientByArea(area)

	v := url.Values{}
	v.Set("access_key", accessKey)
	v.Set("area", area)
	v.Set("cid", cid)
	v.Set("ep_id", epid)
	v.Set("fnver", "0")
	v.Set("fnval", "80")
	v.Set("fourk", "1")
	v.Set("platform", "android")
	// v.Set("qn", "120")

	params, err := SignParams(v, ClientTypeAndroid)
	if err != nil {
		b.sugar.Error(err)
		ctx.Error(
			fasthttp.StatusMessage(fasthttp.StatusInternalServerError),
			fasthttp.StatusInternalServerError,
		)
		return
	}

	reverseProxy := b.getReverseProxyByArea(area)
	if reverseProxy == "" {
		reverseProxy = "api.bilibili.com"
	}
	domain, err := idna.New().ToASCII(reverseProxy)
	if err != nil {
		b.sugar.Error(err)
		ctx.Error(
			fasthttp.StatusMessage(fasthttp.StatusInternalServerError),
			fasthttp.StatusInternalServerError,
		)
		return
	}

	url := fmt.Sprintf("https://%s/pgc/player/api/playurl?%s", domain, params)
	b.sugar.Debug("New url: ", url)

	b.doRequest(ctx, client, url)
}

func (b *BiliroamingGo) handleBstarAndroidSeason(ctx *fasthttp.RequestCtx) {
	queryArgs := ctx.URI().QueryArgs()
	accessKey, area, _, epID, seasonID := b.processArgs(queryArgs)
	client := b.getClientByArea(area)

	v := url.Values{}
	v.Set("access_key", accessKey)
	v.Set("area", area)
	v.Set("s_locale", "zh_SG")
	if epID != "" {
		v.Set("ep_id", string(epID))
	}
	if seasonID != "" {
		v.Set("season_id", string(seasonID))
	}
	v.Set("mobi_app", "bstar_a")

	params, err := SignParams(v, ClientTypeBstarA)
	if err != nil {
		b.sugar.Error(err)
		ctx.Error(
			fasthttp.StatusMessage(fasthttp.StatusInternalServerError),
			fasthttp.StatusInternalServerError,
		)
		return
	}

	reverseProxy := b.getReverseProxyByArea(area)
	if reverseProxy == "" {
		reverseProxy = "api.global.bilibili.com"
	}
	domain, err := idna.New().ToASCII(reverseProxy)
	if err != nil {
		b.sugar.Error(err)
		ctx.Error(
			fasthttp.StatusMessage(fasthttp.StatusInternalServerError),
			fasthttp.StatusInternalServerError,
		)
		return
	}

	url := fmt.Sprintf("https://%s/intl/gateway/v2/ogv/view/app/season?%s", domain, params)
	b.sugar.Debug("New url: ", url)

	b.doRequest(ctx, client, url)
}

func (b *BiliroamingGo) handleBstarAndroidSubtitle(ctx *fasthttp.RequestCtx) {
	queryArgs := ctx.URI().QueryArgs()
	accessKey, area, _, epID, _ := b.processArgs(queryArgs)
	client := b.getClientByArea(area)

	v := url.Values{}
	v.Set("access_key", accessKey)
	v.Set("area", area)
	v.Set("s_locale", "zh_SG")
	v.Set("ep_id", string(epID))
	v.Set("mobi_app", "bstar_a")

	params, err := SignParams(v, ClientTypeBstarA)
	if err != nil {
		b.sugar.Error(err)
		ctx.Error(
			fasthttp.StatusMessage(fasthttp.StatusInternalServerError),
			fasthttp.StatusInternalServerError,
		)
		return
	}

	reverseProxy := b.getReverseProxyByArea(area)
	if reverseProxy == "" {
		reverseProxy = "app.global.bilibili.com"
	}
	domain, err := idna.New().ToASCII(reverseProxy)
	if err != nil {
		b.sugar.Error(err)
		ctx.Error(
			fasthttp.StatusMessage(fasthttp.StatusInternalServerError),
			fasthttp.StatusInternalServerError,
		)
		return
	}

	url := fmt.Sprintf("https://%s/intl/gateway/v2/app/subtitle?%s", domain, params)
	b.sugar.Debug("New url: ", url)

	b.doRequest(ctx, client, url)
}

func (b *BiliroamingGo) handleBstarAndroidPlayURL(ctx *fasthttp.RequestCtx) {
	queryArgs := ctx.URI().QueryArgs()
	accessKey, area, cid, epid, _ := b.processArgs(queryArgs)
	client := b.getClientByArea(area)

	v := url.Values{}
	v.Set("access_key", accessKey)
	v.Set("area", area)
	v.Set("cid", cid)
	v.Set("ep_id", epid)
	v.Set("fnver", "0")
	v.Set("fnval", "80")
	v.Set("fourk", "1")
	v.Set("platform", "android")
	v.Set("s_locale", "zh_SG")
	// v.Set("qn", "120")

	params, err := SignParams(v, ClientTypeBstarA)
	if err != nil {
		b.sugar.Error(err)
		ctx.Error(
			fasthttp.StatusMessage(fasthttp.StatusInternalServerError),
			fasthttp.StatusInternalServerError,
		)
		return
	}

	reverseProxy := b.getReverseProxyByArea(area)
	if reverseProxy == "" {
		reverseProxy = "api.global.bilibili.com"
	}
	domain, err := idna.New().ToASCII(reverseProxy)
	if err != nil {
		b.sugar.Error(err)
		ctx.Error(
			fasthttp.StatusMessage(fasthttp.StatusInternalServerError),
			fasthttp.StatusInternalServerError,
		)
		return
	}

	url := fmt.Sprintf("https://%s/intl/gateway/v2/ogv/playurl?%s", domain, params)
	b.sugar.Debug("New url: ", url)

	b.doRequest(ctx, client, url)
}

func setDefaultHeaders(ctx *fasthttp.RequestCtx) {
	ctx.Response.Header.Set("Access-Control-Allow-Origin", "https://www.bilibili.com")
	ctx.Response.Header.Set("Access-Control-Allow-Credentials", "true")
	ctx.Response.Header.Set("Server", "Potato")
	ctx.SetContentType("application/json")
}

func writeErrorJSON(ctx *fasthttp.RequestCtx) {
	setDefaultHeaders(ctx)
	ctx.SetContentType("application/json")
	ctx.Write([]byte(`{"accept_format":"mp4","code":0,"seek_param":"start","is_preview":0,"fnval":1,"video_project":true,"fnver":0,"type":"MP4","bp":0,"result":"suee","seek_type":"offset","qn_extras":[{"attribute":0,"icon":"http://i0.hdslb.com/bfs/app/81dab3a04370aafa93525053c4e760ac834fcc2f.png","icon2":"http://i0.hdslb.com/bfs/app/4e6f14c2806f7cc508d8b6f5f1d8306f94a71ecc.png","need_login":true,"need_vip":true,"qn":112},{"attribute":0,"icon":"","icon2":"","need_login":false,"need_vip":false,"qn":80},{"attribute":0,"icon":"","icon2":"","need_login":false,"need_vip":false,"qn":64},{"attribute":0,"icon":"","icon2":"","need_login":false,"need_vip":false,"qn":32},{"attribute":0,"icon":"","icon2":"","need_login":false,"need_vip":false,"qn":16}],"accept_watermark":[false,false,false,false,false],"from":"local","video_codecid":7,"durl":[{"order":1,"length":16740,"size":172775,"ahead":"","vhead":"","url":"https://s1.hdslb.com/bfs/static/player/media/error.mp4","backup_url":[]}],"no_rexcode":0,"format":"mp4","support_formats":[{"display_desc":"360P","superscript":"","format":"mp4","description":"流畅 360P","quality":16,"new_description":"360P 流畅"}],"message":"","accept_quality":[16],"quality":16,"timelength":16740,"has_paid":false,"accept_description":["流畅 360P"],"status":2}`))
}

func (b *BiliroamingGo) getMyInfo(accessKey string) (string, error) {
	apiURL := "https://app.bilibili.com/x/v2/account/myinfo"

	v := url.Values{}

	v.Add("access_key", accessKey)

	params, err := SignParams(v, ClientTypeAndroid)
	if err != nil {
		return "", err
	}
	apiURL += "?" + params

	b.sugar.Debug(apiURL)

	statusCode, body, err := fasthttp.Get(nil, apiURL)
	if err != nil {
		return "", err
	}
	if statusCode != 200 {
		return "", fmt.Errorf("Get info failed with status code %d", statusCode)
	}
	return string(body), nil
}
