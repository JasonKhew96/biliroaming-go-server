package main

import (
	"context"
	"log"
	"os"
	"regexp"
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
	MAJOR    = "1"
	MINOR    = "1"
	REVISION = "0"

	VERSION = "v" + MAJOR + "." + MINOR + "." + REVISION
)

// biliArgs query arguments struct
type biliArgs struct {
	accessKey string
	area      string
	cid       string
	epId      string
	seasonId  string
	keyword   string
	pn        string
}

// ip string
type visitor struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// BiliroamingGo ...
type BiliroamingGo struct {
	config   *Config
	visitors map[string]*visitor
	vMu      sync.RWMutex
	ctx      context.Context
	logger   *zap.Logger
	sugar    *zap.SugaredLogger

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

	db *database.Database
}

var reMid = regexp.MustCompile(`(&|\\u0026)mid=\d+`)

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
		config:   c,
		visitors: make(map[string]*visitor),
		ctx:      context.Background(),
		logger:   logger,
		sugar:    sugar,

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
	queryArgs := &biliArgs{
		accessKey: string(args.Peek("access_key")),
		area:      string(args.Peek("area")),
		cid:       string(args.Peek("cid")),
		epId:      string(args.Peek("ep_id")),
		seasonId:  string(args.Peek("season_id")),
		keyword:   string(args.Peek("keyword")),
		pn:        string(args.Peek("pn")),
	}
	b.sugar.Debug("Request args ", args.String())
	b.sugar.Debugf(
		"Parsed request args: %v",
		queryArgs,
	)
	return queryArgs
}

func (b *BiliroamingGo) doAuth(ctx *fasthttp.RequestCtx, accessKey, area string) (bool, bool) {
	if len(accessKey) != 32 {
		writeErrorJSON(ctx, -2, []byte("Access Key错误"))
		return false, false
	}

	status, err := b.isAuth(ctx, accessKey)
	if err != nil {
		b.sugar.Error(err)
		writeErrorJSON(ctx, -500, []byte("服务器错误"))
		return false, false
	}
	if !status.isAuth {
		writeErrorJSON(ctx, -101, []byte("账号未登录"))
		return false, false
	}
	if status.isBlacklist {
		writeErrorJSON(ctx, -101, []byte("黑名单"))
		return false, false
	}
	return true, status.isVip
}
