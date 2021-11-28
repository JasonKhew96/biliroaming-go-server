package main

import (
	"context"
	"log"
	"os"
	"strconv"
	"sync"
	"time"

	realip "github.com/Ferluci/fast-realip"
	"github.com/JasonKhew96/biliroaming-go-server/database"
	"github.com/JasonKhew96/biliroaming-go-server/entity"
	"github.com/valyala/fasthttp"
	"go.uber.org/zap"
	"golang.org/x/time/rate"
)

const (
	MAJOR    = "2"
	MINOR    = "2"
	REVISION = "0"

	VERSION = "v" + MAJOR + "." + MINOR + "." + REVISION
)

// biliArgs query arguments struct
type biliArgs struct {
	accessKey string
	area      string
	cid       int64
	epId      int64
	seasonId  int64
	keyword   string
	pn        int
	qn        int
}

// ip string
type visitor struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

type accessKey struct {
	isLogin     bool
	isVip       bool
	isBlacklist bool
	isWhitelist bool
	timestamp   time.Time
}

// BiliroamingGo ...
type BiliroamingGo struct {
	configPath string
	config     *Config
	visitors   map[string]*visitor
	accessKeys map[string]*accessKey
	vMu        sync.RWMutex
	aMu        sync.RWMutex
	ctx        context.Context
	logger     *zap.Logger
	sugar      *zap.SugaredLogger

	cnClient      *fasthttp.Client
	hkClient      *fasthttp.Client
	twClient      *fasthttp.Client
	thClient      *fasthttp.Client
	defaultClient *fasthttp.Client

	HealthPlayUrlCN *entity.Health
	HealthPlayUrlHK *entity.Health
	HealthPlayUrlTW *entity.Health
	HealthPlayUrlTH *entity.Health

	HealthSeasonTH *entity.Health

	HealthSearchCN *entity.Health
	HealthSearchHK *entity.Health
	HealthSearchTW *entity.Health
	HealthSearchTH *entity.Health

	db *database.DbHelper
}

// get visitor limiter
func (b *BiliroamingGo) getVisitor(ip string) *rate.Limiter {
	b.vMu.Lock()
	defer b.vMu.Unlock()
	u, exists := b.visitors[ip]
	if !exists {
		rt := rate.Every(time.Second / time.Duration(b.config.Limiter.IpLimit))
		uLimiter := rate.NewLimiter(rt, b.config.Limiter.IpBurst)
		b.visitors[ip] = &visitor{
			limiter: uLimiter,
		}
		return uLimiter
	}

	u.lastSeen = time.Now()
	return u.limiter
}

func (b *BiliroamingGo) getKey(key string) (*accessKey, bool) {
	b.aMu.Lock()
	defer b.aMu.Unlock()
	k, exists := b.accessKeys[key]
	return k, exists
}

func (b *BiliroamingGo) setKey(key string, isLogin bool, status *userStatus) {
	b.aMu.Lock()
	defer b.aMu.Unlock()
	b.accessKeys[key] = &accessKey{
		isLogin:     isLogin,
		isVip:       status.isVip,
		isBlacklist: status.isBlacklist,
		isWhitelist: status.isWhitelist,
		timestamp:   time.Now(),
	}
}

func (b *BiliroamingGo) loop() {
	for {
		b.sugar.Debug("Cleaning database...")
		if aff, err := b.db.CleanupAccessKeys(b.config.Cache.AccessKey); err != nil {
			b.sugar.Error(err)
		} else {
			b.sugar.Debugf("Cleanup %d access keys cache", aff)
		}
		if aff, err := b.db.CleanupUsers(b.config.Cache.User); err != nil {
			b.sugar.Error(err)
		} else {
			b.sugar.Debugf("Cleanup %d users cache", aff)
		}
		// if aff, err := b.db.CleanupPlayURLCache(time.Duration(b.config.CachePlayURL) * time.Minute); err != nil {
		// 	b.sugar.Error(err)
		// } else {
		// 	b.sugar.Debugf("Cleanup %d playURL cache", aff)
		// }
		// if aff, err := b.db.CleanupTHSeasonCache(time.Duration(b.config.CacheTHSeason) * time.Minute); err != nil {
		// 	b.sugar.Error(err)
		// } else {
		// 	b.sugar.Debugf("Cleanup %d TH season cache", aff)
		// }
		// if aff, err := b.db.CleanupTHSubtitleCache(time.Duration(b.config.CacheTHSubtitle) * time.Minute); err != nil {
		// 	b.sugar.Error(err)
		// } else {
		// 	b.sugar.Debugf("Cleanup %d TH subtitle cache", aff)
		// }

		// cleanup ip cache
		b.vMu.Lock()
		for ip, v := range b.visitors {
			if time.Since(v.lastSeen) > 15*time.Minute {
				delete(b.visitors, ip)
			}
		}
		b.vMu.Unlock()

		// cleanup key cache
		b.aMu.Lock()
		for k, v := range b.accessKeys {
			if time.Since(v.timestamp) > 15*time.Minute {
				delete(b.accessKeys, k)
			}
		}
		b.aMu.Unlock()

		time.Sleep(15 * time.Minute)
	}
}

func getDbPassword(c *Config) (string, error) {
	pgPassword := c.PostgreSQL.Password
	if c.PostgreSQL.PasswordFile != "" {
		data, err := os.ReadFile(c.PostgreSQL.PasswordFile)
		if err != nil {
			return "", err
		}
		if len(data) > 0 {
			pgPassword = string(data)
		}
	}
	return pgPassword, nil
}

func initHttpServer(c *Config, b *BiliroamingGo) {
	fs := &fasthttp.FS{
		Root:               "html",
		IndexNames:         []string{"index.html"},
		GenerateIndexPages: true,
		Compress:           true,
		AcceptByteRange:    false,
		PathNotFound:       processNotFound,
		// PathRewrite:        fasthttp.NewVHostPathRewriter(0),
	}
	fsHandler := fs.NewRequestHandler()

	mux := func(ctx *fasthttp.RequestCtx) {
		clientIP := realip.FromRequest(ctx)
		limiter := b.getVisitor(clientIP)
		if !limiter.Allow() {
			ctx.Error(fasthttp.StatusMessage(fasthttp.StatusTooManyRequests), fasthttp.StatusTooManyRequests)
			return
		}

		switch string(ctx.Path()) {
		case "/pgc/player/web/playurl": // web
			b.handleWebPlayURL(ctx)
		case "/x/v2/search/type": // android
			b.handleAndroidSearch(ctx)
		case "/pgc/player/api/playurl": // android
			b.handleAndroidPlayURL(ctx)
		case "/intl/gateway/v2/app/search/type": // bstar android
			b.handleBstarAndroidSearch(ctx)
		case "/intl/gateway/v2/ogv/view/app/season": // bstar android
			b.handleBstarAndroidSeason(ctx)
		case "/intl/gateway/v2/app/subtitle": // bstar android
			b.handleBstarAndroidSubtitle(ctx)
		case "/intl/gateway/v2/ogv/playurl": // bstar android
			b.handleBstarAndroidPlayURL(ctx)

		case "/api/health": // custom health
			b.handleApiHealth(ctx)

		default:
			fsHandler(ctx)
			// ctx.Error(fasthttp.StatusMessage(fasthttp.StatusNotFound), fasthttp.StatusNotFound)
		}
	}

	b.sugar.Infof("Listening on :%d ...", c.Port)
	err := fasthttp.ListenAndServe(":"+strconv.Itoa(c.Port), mux)
	if err != nil {
		b.sugar.Panic(err)
	}
}

func main() {
	configPath, err := parseFlags()
	if err != nil {
		log.Fatal(err)
	}

	c, err := initConfig(configPath)
	if err != nil {
		log.Fatal(err)
	}

	logger, err := initLogger(c.Debug)
	if err != nil {
		log.Fatal(err)
	}
	sugar := logger.Sugar()

	sugar.Infof("Version: %s", VERSION)
	sugar.Debug(c)

	b := &BiliroamingGo{
		configPath: configPath,
		config:     c,
		visitors:   make(map[string]*visitor),
		accessKeys: make(map[string]*accessKey),
		ctx:        context.Background(),
		logger:     logger,
		sugar:      sugar,

		HealthPlayUrlCN: newHealth(),
		HealthPlayUrlHK: newHealth(),
		HealthPlayUrlTW: newHealth(),
		HealthPlayUrlTH: newHealth(),

		HealthSeasonTH: newHealth(),

		HealthSearchCN: newHealth(),
		HealthSearchHK: newHealth(),
		HealthSearchTW: newHealth(),
		HealthSearchTH: newHealth(),
	}

	b.initProxy(b.config)

	pgPassword, err := getDbPassword(c)
	if err != nil {
		b.sugar.Fatal(err)
	}

	b.db, err = database.NewDBConnection(&database.Config{
		Host:     c.PostgreSQL.Host,
		User:     c.PostgreSQL.User,
		Password: pgPassword,
		DBName:   c.PostgreSQL.DBName,
		Port:     c.PostgreSQL.Port,
		Debug:    c.Debug,
	})
	if err != nil {
		b.sugar.Fatal(err)
	}

	go b.loop()

	initHttpServer(c, b)
}

func processNotFound(ctx *fasthttp.RequestCtx) {
	ctx.Error(fasthttp.StatusMessage(fasthttp.StatusNotFound), fasthttp.StatusNotFound)
}

func (b *BiliroamingGo) processError(ctx *fasthttp.RequestCtx, err error) {
	b.sugar.Error(err)
	ctx.Error(
		fasthttp.StatusMessage(fasthttp.StatusInternalServerError),
		fasthttp.StatusInternalServerError,
	)
}

func (b *BiliroamingGo) processArgs(args *fasthttp.Args) *biliArgs {
	cid, err := strconv.ParseInt(string(args.Peek("cid")), 10, 64)
	if err != nil {
		cid = 0
	}
	epId, err := strconv.ParseInt(string(args.Peek("ep_id")), 10, 64)
	if err != nil {
		epId = 0
	}
	seasonId, err := strconv.ParseInt(string(args.Peek("season_id")), 10, 64)
	if err != nil {
		seasonId = 0
	}
	pn, err := strconv.Atoi(string(args.Peek("pn")))
	if err != nil {
		pn = 0
	}
	qn, err := strconv.Atoi(string(args.Peek("qn")))
	if err != nil || qn == 0 {
		qn = 16
	}

	queryArgs := &biliArgs{
		accessKey: string(args.Peek("access_key")),
		area:      string(args.Peek("area")),
		cid:       cid,
		epId:      epId,
		seasonId:  seasonId,
		keyword:   string(args.Peek("keyword")),
		pn:        pn,
		qn:        qn,
	}

	b.sugar.Debug("Request args ", args.String())
	b.sugar.Debugf(
		"Parsed request args: %v",
		queryArgs,
	)
	return queryArgs
}
