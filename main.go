package main

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/md5"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/pkg/errors"
	"github.com/tidwall/gjson"
	"go.uber.org/zap"
	"golang.org/x/time/rate"
	"gopkg.in/yaml.v2"
)

// ip string
type visitor struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

type configuration struct {
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

type biliroamingGo struct {
	config        *configuration
	globalLimiter *rate.Limiter
	visitors      map[string]*visitor
	vMu           sync.RWMutex
	ctx           context.Context
	rdb           *redis.Client
	logger        *zap.Logger
	sugar         *zap.SugaredLogger
}

const (
	// local url
	localTopBangumiURL = "/public/bangumi"
	localBanlistURL    = "/public/banlist"

	// api url
	// bstar
	apiBstarPlayURL  = "/intl/gateway/v2/ogv/playurl"
	apiBstarSubtitle = "/intl/gateway/v2/app/subtitle"
	apiBstarSearch   = "/intl/gateway/v2/app/search/type"
	// pink
	apiPinkPlayURL = "/pgc/player/api/playurl"
	// web
	apiWebPlayURL = "/pgc/player/web/playurl"

	// host
	hostPinkURL    = "api.bilibili.com"
	hostBlueAPIURL = "api.global.bilibili.com"
	hostBlueAppURL = "app.global.bilibili.com"
)

var validReqPaths = []string{
	// blue
	apiBstarPlayURL,
	apiBstarSubtitle,
	apiBstarSearch,
	// pink
	apiPinkPlayURL,
	// web
	apiWebPlayURL,
}

// get visitor limiter
func (b *biliroamingGo) getVisitor(ip string) *rate.Limiter {
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

func (b *biliroamingGo) cleanupVisitors() {
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

func isProxyPath(path string) bool {
	for _, validPath := range validReqPaths {
		if strings.HasPrefix(path, validPath) {
			return true
		}
	}
	return false
}

func main() {
	// default config
	c := &configuration{
		Debug:                 false,
		Port:                  23333,
		GlobalLimit:           4,
		GlobalBurst:           8,
		IPLimit:               2,
		IPBurst:               4,
		RedisAddr:             "localhost:6379",
		RedisPwd:              "",
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

	rdb := redis.NewClient(&redis.Options{
		Addr:     c.RedisAddr,
		Password: c.RedisPwd,
		DB:       0,
	})

	b := biliroamingGo{
		config:        c,
		globalLimiter: rate.NewLimiter(rate.Limit(c.GlobalLimit), c.GlobalBurst),
		visitors:      make(map[string]*visitor),
		ctx:           context.Background(),
		rdb:           rdb,
		logger:        logger,
		sugar:         sugar,
	}

	go b.cleanupVisitors()

	mux := http.NewServeMux()

	mux.HandleFunc(localTopBangumiURL, b.handleTopBangumi)
	mux.HandleFunc(localBanlistURL, b.handleBanList)
	mux.HandleFunc("/", b.handleReverseProxy)
	sugar.Infof("Listening on :%d ...", c.Port)
	err = http.ListenAndServe(":"+strconv.Itoa(c.Port), mux)
	if err != nil {
		sugar.Panic(err)
	}
}

func (b *biliroamingGo) handleTopBangumi(w http.ResponseWriter, r *http.Request) {
	simpleList := `
	<head>
	<style>
	  table, th, td {
		border: 1px solid black;
	  }
	</style>
	</head>
	<body>
	`
	simpleList += "<table><tr><th>cid</th><th>Counter</th></tr>"
	keys, err := b.getBangumiReqCountKeys()
	if err != nil {
		b.sugar.Error(err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	for _, key := range keys {
		count, err := b.rdb.Get(b.ctx, key).Result()
		if err != nil {
			b.sugar.Error(err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		datas := strings.Split(key, ":")
		epID := datas[1]
		simpleList += fmt.Sprintf("<tr><td>%s</td><td>%s</td></tr>", epID, count)
	}
	simpleList += "</table></body>"
	fmt.Fprintf(w, simpleList)
}

func (b *biliroamingGo) handleBanList(w http.ResponseWriter, r *http.Request) {
	simpleList := `
	<head>
	<style>
	  table, th, td {
		border: 1px solid black;
	  }
	</style>
	</head>
	<body>
	`
	simpleList += "<table><tr><th>Banned time</th><th>User ID</th><th>Name</th><th>Reason</th></tr>"
	keys, err := b.getBanListKeys()
	if err != nil {
		b.sugar.Error(err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	for _, key := range keys {
		data, err := b.rdb.HGetAll(b.ctx, key).Result()
		if err != nil {
			b.sugar.Error(err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		mid := strings.Split(key, ":")[1]
		name, err := b.getName(mid)
		if err != nil {
			b.sugar.Error(err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		simpleList += fmt.Sprintf("<tr><td>%s</td><td>%s</td><td>%s</td><td>%s</td></tr>", data["time"], mid, name, data["reason"])
	}
	simpleList += "</table></body>"
	fmt.Fprintf(w, simpleList)
}

// swap host
func (b *biliroamingGo) directorFunc(req *http.Request) {
	req.URL.Scheme = "https"
	if strings.HasPrefix(req.URL.Path, apiBstarPlayURL) {
		req.URL.Host = hostBlueAPIURL
		req.Host = hostBlueAPIURL
	} else if strings.HasPrefix(req.URL.Path, apiBstarSubtitle) || strings.HasPrefix(req.URL.Path, apiBstarSearch) {
		req.URL.Host = hostBlueAppURL
		req.Host = hostBlueAppURL
	} else if strings.HasPrefix(req.URL.Path, apiPinkPlayURL) || strings.HasPrefix(req.URL.Path, apiWebPlayURL) {
		req.URL.Host = hostPinkURL
		req.Host = hostPinkURL
	} else {
		b.sugar.Debug("Unknown path:", req.URL.Path)
	}

	b.sugar.Debug("Proxy URL: " + req.URL.String())
}

func (b *biliroamingGo) modifyResponse(res *http.Response) error {
	if res.StatusCode != http.StatusOK {
		return nil
	}
	if !strings.HasPrefix(res.Request.URL.Path, apiBstarSubtitle) && !strings.HasPrefix(res.Request.URL.Path, apiBstarSearch) {
		// statistics and cache
		cid := res.Request.URL.Query().Get("cid")
		fnval := res.Request.URL.Query().Get("fnval")
		qn := res.Request.URL.Query().Get("qn")
		if cid == "" || fnval == "" || qn == "" {
			return nil
		}
		err := b.incrBangumiReqCount(cid)
		if err != nil {
			b.sugar.Error(errors.Wrap(err, "redis increment bangumi"))
		}
		accessKey := res.Request.URL.Query().Get("access_key")
		if accessKey == "" {
			return nil
		}
		mid, err := b.getMid(accessKey)
		if err != nil {
			return nil
		}

		var reader io.ReadCloser
		switch res.Header.Get("Content-Encoding") {
		case "gzip":
			reader, err = gzip.NewReader(res.Body)
			if err != nil {
				b.sugar.Error(errors.Wrap(err, "Read response failed"))
			}
			defer reader.Close()
		default:
			reader = res.Body
		}

		body, err := ioutil.ReadAll(reader)
		if err != nil {
			return err
		}
		res.Body.Close()

		isVip := ""
		_, err = b.getVIP(mid)
		if err == redis.Nil {
			isVip = "0"
		} else if err == nil {
			isVip = "1"
		} else {
			b.sugar.Error(errors.Wrap(err, "redis getVIP unknown error"))
			return nil
		}
		b.sugar.Debug("Response:", string(body))

		code := gjson.Get(string(body), "code").Int()
		// status ok || status area restricted
		if code == 0 || code == -10403 {
			data := string(body)
			m1 := regexp.MustCompile(`\&mid=\d+`)
			newBody := m1.ReplaceAllString(data, "")
			body = []byte(newBody)
			if strings.HasPrefix(res.Request.URL.Path, apiWebPlayURL) {
				err = b.setPlayURLWebCache(cid, fnval, qn, isVip, newBody)
			} else if strings.HasPrefix(res.Request.URL.Path, apiBstarPlayURL) {
				err = b.setPlayURLBstarCache(cid, fnval, qn, isVip, newBody)
			} else {
				err = b.setPlayURLCache(cid, fnval, qn, isVip, newBody)
			}
			if err != nil {
				b.sugar.Error(errors.Wrap(err, "redis insertPlayURLCache"))
				return nil
			}
		}

		res.Header.Del("Content-Encoding")
		res.Body = ioutil.NopCloser(bytes.NewReader(body))
	}

	// CORS
	if strings.HasPrefix(res.Request.URL.Path, apiWebPlayURL) || strings.HasPrefix(res.Request.URL.Path, apiBstarPlayURL) {
		res.Header.Set("Access-Control-Allow-Origin", "https://www.bilibili.com")
		res.Header.Set("Access-Control-Allow-Credentials", "true")
	}

	return nil
}

func (b *biliroamingGo) handleReverseProxy(w http.ResponseWriter, r *http.Request) {
	if !isProxyPath(r.URL.Path) {
		http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
		return
	}

	// check area
	area := r.Header.Get("area")
	if area == "" {
		http.Error(w, `{"code":-10403,"message":"抱歉您所在地区不可观看！"}`, http.StatusForbidden)
		return
	}

	// get ip
	var err error
	ip := r.Header.Get("X-Forwarded-For")
	if ip == "" {
		ip = r.Header.Get("X-Real-IP")
	}
	if ip == "" {
		ip, _, err = net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			b.sugar.Error(errors.Wrap(err, "SplitHostPort"))
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
	}

	accessKey := r.URL.Query().Get("access_key")
	if accessKey == "" {
		http.Error(w, `{"code":-10403,"message":"抱歉您所在地区不可观看！"}`, http.StatusForbidden)
		return
	}
	if len(accessKey) == 32 {
		// with access_key
		b.sugar.Debugf("%s %s", ip, accessKey)
		var name string
		// get mid from access key
		mid, err := b.getMid(accessKey)
		// access key not found
		if err != nil {
			// no cache, fetching...

			// check global limit
			if b.globalLimiter.Allow() == false {
				// allow to retry
				b.sugar.Debug("Blocked %s due to global limit", ip)
				http.Error(w, `{"code":-412,"message":"请求被拦截"}`, http.StatusTooManyRequests)
				return
			}

			// fetching new user info
			data, err := b.getMyInfo(accessKey)
			if err != nil {
				b.sugar.Error(ip, r.URL.String())
				b.sugar.Error(errors.Wrap(err, "getMyInfo"))
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				return
			}
			b.sugar.Debug("myInfo:", data)

			if gjson.Get(data, "code").String() != "0" {
				b.sugar.Error(ip, r.URL.String())
				b.sugar.Error("getMyInfo: " + data)
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				return
			}
			name = gjson.Get(data, "data.name").String()
			mid = gjson.Get(data, "data.mid").String()
			vipDueUnix := gjson.Get(data, "data.vip.due_date").Int() / 1000
			if mid == "" {
				b.sugar.Error(ip, r.URL.String())
				b.sugar.Error("getMyInfo malformed json: " + data)
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				return
			}

			b.sugar.Debugf("access_key %s %s %s %s", accessKey, mid, name, vipDueUnix)
			err = b.setAccessKey(accessKey, mid)
			if err != nil {
				b.sugar.Error(ip, r.URL.String())
				b.sugar.Error(errors.Wrap(err, "redis insertAccessKey"))
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				return
			}

			err = b.setName(mid, name)
			if err != nil {
				b.sugar.Error(ip, r.URL.String())
				b.sugar.Error(err)
				b.sugar.Error(errors.Wrap(err, "redis insertName"))
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				return
			}

			// save vip data
			if vipDueUnix != 0 {
				err = b.setVIP(mid, time.Unix(vipDueUnix, 0))
				if err != nil {
					b.sugar.Error(ip, r.URL.String())
					b.sugar.Error(errors.Wrap(err, "redis insertVIP"))
					http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
					return
				}
			}
		} else {
			// cached
			name, err = b.getName(mid)
			if err != nil {
				b.sugar.Error(ip, r.URL.String())
				b.sugar.Error(errors.Wrap(err, "redis getName"))
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				return
			}

			bans, err := b.getBan(mid)
			if err != nil {
				b.sugar.Error(ip, r.URL.String())
				b.sugar.Error(errors.Wrap(err, "redis getBan"))
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				return
			}

			// is banned
			if len(bans) > 0 {
				b.sugar.Debugf("Blocked %s with mid %s and name %s (time: %s, reason: %s)", ip, mid, name, bans["time"], bans["reason"])
				writeErrorJSON(w)
				return
			}
		}
		// ban if request too many
		uLimiter := b.getVisitor(ip)
		if uLimiter.Allow() == false {
			b.sugar.Debugf("Banned %s with mid %s and name %s (autoban)", ip, mid, name)
			err = b.setBan(mid, "autoban")
			if err != nil {
				b.sugar.Error(ip, r.URL.String())
				b.sugar.Error(errors.Wrap(err, "redis insertBan"))
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				return
			}
			writeErrorJSON(w)
			return
		}

		// check playurl cache
		if !strings.HasPrefix(r.URL.Path, apiBstarSubtitle) && !strings.HasPrefix(r.URL.Path, apiBstarSearch) {
			cid := r.URL.Query().Get("cid")
			fnval := r.URL.Query().Get("fnval")
			qn := r.URL.Query().Get("qn")
			if cid != "" || fnval != "" || qn != "" {
				isVip := ""
				_, err = b.getVIP(mid)
				if err == redis.Nil {
					isVip = "0"
				} else if err == nil {
					isVip = "1"
				} else {
					b.sugar.Error(ip, r.URL.String())
					b.sugar.Error(errors.Wrap(err, "redis getVIP unknown error"))
					http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
					return
				}

				if strings.HasPrefix(r.URL.Path, apiWebPlayURL) {
					data, err := b.getPlayURLWebCacheFrom(cid, fnval, qn, isVip)
					if err != redis.Nil {
						// playurl cached
						b.sugar.Debug("Replay cache response:", data)

						// CORS
						w.Header().Set("Access-Control-Allow-Origin", "https://www.bilibili.com")
						w.Header().Set("Access-Control-Allow-Credentials", "true")

						fmt.Fprintf(w, "%s", data)
						return
					}
				} else if strings.HasPrefix(r.URL.Path, apiBstarPlayURL) {
					data, err := b.getPlayURLBstarCacheFrom(cid, fnval, qn, isVip)
					if err != redis.Nil {
						// playurl cached
						b.sugar.Debug("Replay cache response:", data)

						// CORS
						w.Header().Set("Access-Control-Allow-Origin", "https://www.bilibili.com")
						w.Header().Set("Access-Control-Allow-Credentials", "true")

						fmt.Fprintf(w, "%s", data)
						return
					}
				} else {
					data, err := b.getPlayURLCacheFrom(cid, fnval, qn, isVip)
					if err != redis.Nil {
						// playurl cached
						b.sugar.Debug("Replay cache response:", data)

						fmt.Fprintf(w, "%s", data)
						return
					}
				}

			}
		}
	}
	// } else {
	// 	// without access_key
	// 	uLimiter := b.getVisitor(ip)
	// 	if uLimiter.Allow() == false {
	// 		b.sugar.Debug("Blocked %s due to ip rate limit", ip)
	// 		writeErrorJSON(w)
	// 		return
	// 	}
	// }

	// finally
	proxy := &httputil.ReverseProxy{
		Director:       b.directorFunc,
		ModifyResponse: b.modifyResponse,
	}
	proxy.ServeHTTP(w, r)
	// fmt.Fprintf(w, "OK")
}

func writeErrorJSON(w http.ResponseWriter) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"accept_format":"mp4","code":0,"seek_param":"start","is_preview":0,"fnval":1,"video_project":true,"fnver":0,"type":"MP4","bp":0,"result":"suee","seek_type":"offset","qn_extras":[{"attribute":0,"icon":"http://i0.hdslb.com/bfs/app/81dab3a04370aafa93525053c4e760ac834fcc2f.png","icon2":"http://i0.hdslb.com/bfs/app/4e6f14c2806f7cc508d8b6f5f1d8306f94a71ecc.png","need_login":true,"need_vip":true,"qn":112},{"attribute":0,"icon":"","icon2":"","need_login":false,"need_vip":false,"qn":80},{"attribute":0,"icon":"","icon2":"","need_login":false,"need_vip":false,"qn":64},{"attribute":0,"icon":"","icon2":"","need_login":false,"need_vip":false,"qn":32},{"attribute":0,"icon":"","icon2":"","need_login":false,"need_vip":false,"qn":16}],"accept_watermark":[false,false,false,false,false],"from":"local","video_codecid":7,"durl":[{"order":1,"length":16740,"size":172775,"ahead":"","vhead":"","url":"https://s1.hdslb.com/bfs/static/player/media/error.mp4","backup_url":[]}],"no_rexcode":0,"format":"mp4","support_formats":[{"display_desc":"360P","superscript":"","format":"mp4","description":"流畅 360P","quality":16,"new_description":"360P 流畅"}],"message":"","accept_quality":[16],"quality":16,"timelength":16740,"has_paid":false,"accept_description":["流畅 360P"],"status":2}`))
}

func (b *biliroamingGo) getMyInfo(accessKey string) (string, error) {
	apiURL := "https://app.bilibili.com/x/v2/account/myinfo"

	v := url.Values{}

	v.Add("access_key", accessKey)
	v.Add("appkey", "1d8b6e7d45233436")
	v.Add("ts", strconv.FormatInt(time.Now().Unix(), 10))
	v.Add("sign", getSign(v.Encode()))

	apiURL += "?" + v.Encode()

	b.sugar.Debug(apiURL)

	res, err := http.Get(apiURL)
	if err != nil {
		return "", err
	}
	if res.StatusCode != 200 {
		return "", fmt.Errorf("Get info failed with status code %d", res.StatusCode)
	}
	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return "", err
	}
	return string(body), nil
}

func getSign(params string) string {
	toEncode := params + "560c52ccd288fed045859ed18bffd973"
	data := []byte(toEncode)
	return fmt.Sprintf("%x", md5.Sum(data))
}
