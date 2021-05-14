package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"sync"
	"time"

	"github.com/JasonKhew96/biliroaming-go-server/database"
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
	// Authentications
	AuthCN bool `yaml:"auth_cn"`
	AuthHK bool `yaml:"auth_hk"`
	AuthTW bool `yaml:"auth_tw"`
	AuthTH bool `yaml:"auth_th"`
	// Postgres
	PGHost     string `yaml:"pg_host"`
	PGUser     string `yaml:"pg_user"`
	PGPassword string `yaml:"pg_password"`
	PGDBName   string `yaml:"pg_dbname"`
	PGPort     int    `yaml:"pg_port"`
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

	db *database.Database
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

	b.db, err = database.NewDBConnection(&database.Config{
		Host:     c.PGHost,
		User:     c.PGUser,
		Password: c.PGPassword,
		DBName:   c.PGDBName,
		Port:     c.PGPort,
	})
	if err != nil {
		b.sugar.Fatal(err)
	}

	// go b.cleanupVisitors()

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
			ctx.Error(fasthttp.StatusMessage(fasthttp.StatusNotFound), fasthttp.StatusNotFound)
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

func (b *BiliroamingGo) doAuth(ctx *fasthttp.RequestCtx, accessKey, area string) (bool, bool) {
	if len(accessKey) != 32 {
		writeErrorJSON(ctx, -2, []byte("Access Key错误"))
		return false, false
	}

	isAuth, isVIP, err := b.isAuth(ctx.Request.Header.UserAgent(), accessKey)
	if err != nil {
		b.sugar.Error(err)
		writeErrorJSON(ctx, -500, []byte("服务器错误"))
		return false, false
	}
	if !isAuth {
		writeErrorJSON(ctx, -101, []byte("账号未登录"))
		return false, false
	}
	return true, isVIP
}

func (b *BiliroamingGo) handleWebPlayURL(ctx *fasthttp.RequestCtx) {
	queryArgs := ctx.URI().QueryArgs()
	accessKey, area, cid, epid, _ := b.processArgs(queryArgs)

	if area == "" {
		writeErrorJSON(ctx, -688, []byte("地理区域限制"))
		return
	}

	cidInt, err := strconv.Atoi(cid)
	if err != nil {
		b.processError(ctx, err)
		return
	}
	epidInt, err := strconv.Atoi(epid)
	if err != nil {
		b.processError(ctx, err)
		return
	}

	var isVIP bool
	if b.getAuthByArea(area) {
		var ok bool
		if ok, isVIP = b.doAuth(ctx, accessKey, area); !ok {
			return
		}
		playurlCache, err := b.db.GetPlayURLCache(database.DeviceTypeWeb, getAreaCode(area), isVIP, cidInt, epidInt)
		if err == nil {
			if playurlCache.JSONData != "" {
				b.sugar.Debug("Replay from cache: ", playurlCache.JSONData)
				setDefaultHeaders(ctx)
				ctx.Write([]byte(playurlCache.JSONData))
				return
			}
		}
	}

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
		b.processError(ctx, err)
		return
	}

	reverseProxy := b.getReverseProxyByArea(area)
	if reverseProxy == "" {
		reverseProxy = "api.bilibili.com"
	}
	domain, err := idna.New().ToASCII(reverseProxy)
	if err != nil {
		b.processError(ctx, err)
		return
	}

	url := fmt.Sprintf("https://%s/pgc/player/web/playurl?%s", domain, params)
	b.sugar.Debug("New url: ", url)

	data := b.doRequest(ctx, client, url)
	if data != nil && b.getAuthByArea(area) {
		b.db.InsertOrUpdatePlayURLCache(database.DeviceTypeWeb, getAreaCode(area), isVIP, cidInt, epidInt, string(data))
	}
}

func (b *BiliroamingGo) handleAndroidPlayURL(ctx *fasthttp.RequestCtx) {
	queryArgs := ctx.URI().QueryArgs()
	accessKey, area, cid, epid, _ := b.processArgs(queryArgs)
	client := b.getClientByArea(area)

	if area == "" {
		writeErrorJSON(ctx, -688, []byte("地理区域限制"))
		return
	}

	cidInt, err := strconv.Atoi(cid)
	if err != nil {
		b.processError(ctx, err)
		return
	}
	epidInt, err := strconv.Atoi(epid)
	if err != nil {
		b.processError(ctx, err)
		return
	}

	var isVIP bool
	if b.getAuthByArea(area) {
		var ok bool
		if ok, isVIP = b.doAuth(ctx, accessKey, area); !ok {
			return
		}

		playurlCache, err := b.db.GetPlayURLCache(database.DeviceTypeAndroid, getAreaCode(area), isVIP, cidInt, epidInt)
		if err == nil {
			if playurlCache.JSONData != "" {
				b.sugar.Debug("Replay from cache: ", playurlCache.JSONData)
				setDefaultHeaders(ctx)
				ctx.Write([]byte(playurlCache.JSONData))
				return
			}
		}
	}

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

	data := b.doRequest(ctx, client, url)
	if data != nil && b.getAuthByArea(area) {
		b.db.InsertOrUpdatePlayURLCache(database.DeviceTypeAndroid, getAreaCode(area), isVIP, cidInt, epidInt, string(data))
	}
}

func (b *BiliroamingGo) handleBstarAndroidSeason(ctx *fasthttp.RequestCtx) {
	queryArgs := ctx.URI().QueryArgs()
	accessKey, area, _, epID, seasonID := b.processArgs(queryArgs)
	client := b.getClientByArea(area)

	if area == "" {
		writeErrorJSON(ctx, -688, []byte("地理区域限制"))
		return
	}

	if b.getAuthByArea(area) {
		if ok, _ := b.doAuth(ctx, accessKey, area); !ok {
			return
		}
	}

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

	if area == "" {
		writeErrorJSON(ctx, -688, []byte("地理区域限制"))
		return
	}

	if b.getAuthByArea(area) {
		if ok, _ := b.doAuth(ctx, accessKey, area); !ok {
			return
		}
	}

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

	if area == "" {
		writeErrorJSON(ctx, -688, []byte("地理区域限制"))
		return
	}

	cidInt, err := strconv.Atoi(cid)
	if err != nil {
		b.processError(ctx, err)
		return
	}
	epidInt, err := strconv.Atoi(epid)
	if err != nil {
		b.processError(ctx, err)
		return
	}

	var isVIP bool
	if b.getAuthByArea(area) {
		var ok bool
		if ok, isVIP = b.doAuth(ctx, accessKey, area); !ok {
			return
		}

		playurlCache, err := b.db.GetPlayURLCache(database.DeviceTypeAndroid, getAreaCode(area), isVIP, cidInt, epidInt)
		if err == nil {
			if playurlCache.JSONData != "" {
				b.sugar.Debug("Replay from cache: ", playurlCache.JSONData)
				setDefaultHeaders(ctx)
				ctx.Write([]byte(playurlCache.JSONData))
				return
			}
		}
	}

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

	data := b.doRequest(ctx, client, url)
	if data != nil && b.getAuthByArea(area) {
		b.db.InsertOrUpdatePlayURLCache(database.DeviceTypeAndroid, getAreaCode(area), isVIP, cidInt, epidInt, string(data))
	}
}
