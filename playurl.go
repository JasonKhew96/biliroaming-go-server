package main

import (
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"time"

	"github.com/JasonKhew96/biliroaming-go-server/database"
	"github.com/valyala/fasthttp"
	"golang.org/x/net/idna"
)

func (b *BiliroamingGo) checkEpisodeAreaCache(episodeId int64, area database.Area) bool {
	if cache, err := b.db.GetEpisodeAreaCache(episodeId); err == nil {
		// shit happened
		if !cache.CN.Bool && !cache.HK.Bool && !cache.TW.Bool && !cache.TH.Bool {
			return true
		}
		switch area {
		case database.AreaCN:
			if cache.CN.Valid && !cache.CN.Bool {
				return false
			}
		case database.AreaHK:
			if cache.HK.Valid && !cache.HK.Bool {
				return false
			}
		case database.AreaTW:
			if cache.TW.Valid && !cache.TW.Bool {
				return false
			}
		case database.AreaTH:
			if cache.TH.Valid && !cache.TH.Bool {
				return false
			}
		default:
			return false
		}
	}
	return true
}

func (b *BiliroamingGo) updateEpisodeCache(data []byte, episodeId int64, area database.Area) error {
	if available, err := isAvailableResponse(data); err != nil {
		return err
	} else if err := b.db.InsertOrUpdateEpisodeAreaCache(episodeId, area, available); err != nil {
		return err
	}
	return nil
}

func (b *BiliroamingGo) handleWebPlayURL(ctx *fasthttp.RequestCtx) {
	queryArgs := ctx.URI().QueryArgs()
	args := b.processArgs(queryArgs)

	if args.area == "" {
		writeErrorJSON(ctx, -10403, []byte("抱歉您所在地区不可观看！"))
		return
	}

	if ok := b.checkEpisodeAreaCache(args.epId, getAreaCode(args.area)); !ok {
		writeErrorJSON(ctx, -10403, []byte("抱歉您所在地区不可观看！"))
		return
	}

	// 验证 sign
	if args.appkey != "" && args.sign != "" && args.ts != 0 {
		if args.ts <= time.Now().Add(-time.Minute).Unix() {
			writeErrorJSON(ctx, -10403, []byte("参数错误！"))
			return
		}

		values, err := url.ParseQuery(queryArgs.String())
		if err != nil {
			b.processError(ctx, err)
			b.updateHealth(b.getPlayUrlHealth(args.area), -500, "服务器错误")
			return
		}

		values.Del("sign")

		sign, err := getSign(values, getClientTypeFromAppkey(args.appkey), args.ts)
		if err != nil {
			b.processError(ctx, err)
			b.updateHealth(b.getPlayUrlHealth(args.area), -500, "服务器错误")
			return
		}

		if sign != args.sign {
			writeErrorJSON(ctx, -10403, []byte("参数错误！"))
			return
		}
	}

	qn := args.qn
	formatType := getFormatType(args.fnval)
	if formatType == database.FormatTypeDash {
		qn = 127
	}

	var isVIP bool
	if b.getAuthByArea(args.area) {
		if ok, status := b.doAuth(ctx, args.accessKey, args.area); !ok {
			return
		} else {
			isVIP = status.isVip
		}

		playurlCache, err := b.db.GetPlayURLCache(database.DeviceTypeWeb, formatType, int16(qn), getAreaCode(args.area), isVIP, args.epId)
		if err == nil && len(playurlCache.Data) > 0 && playurlCache.UpdatedAt.After(time.Now().Add(-b.config.Cache.PlayUrl)) {
			b.sugar.Debug("Replay from cache: ", playurlCache.Data.String())
			setDefaultHeaders(ctx)
			newData, err := replaceQn(playurlCache.Data, args.qn, ClientTypeWeb)
			if err != nil {
				b.processError(ctx, err)
				return
			}
			ctx.Write(newData)
			return
		} else if err != nil && !errors.Is(err, sql.ErrNoRows) {
			b.processError(ctx, err)
			b.updateHealth(b.getPlayUrlHealth(args.area), -500, "服务器错误")
			return
		}
	}

	client := b.getClientByArea(args.area)

	v := url.Values{}
	v.Set("access_key", args.accessKey)
	v.Set("area", args.area)
	v.Set("ep_id", strconv.FormatInt(args.epId, 10))
	v.Set("fnver", "0")

	switch formatType {
	case database.FormatTypeDash:
		v.Set("fnval", "4048")
	case database.FormatTypeMp4:
		v.Set("fnval", "1")
	case database.FormatTypeFlv:
		fallthrough
	default:
		v.Set("fnval", "0")
	}

	v.Set("fourk", "1")
	v.Set("qn", strconv.Itoa(qn))

	params, err := SignParams(v, ClientTypeAndroid)
	if err != nil {
		b.processError(ctx, err)
		return
	}

	reverseProxy := b.getReverseProxyByArea(args.area)
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

	reqParams := &HttpRequestParams{
		Method: []byte(fasthttp.MethodGet),
		Url:    []byte(url),
		UserAgent: ctx.UserAgent(),
	}
	data, err := b.doRequestJsonWithRetry(client, reqParams, 2)
	if err != nil {
		if errors.Is(err, ErrorHttpStatusLimited) {
			data = []byte(`{"code":-412,"message":"请求被拦截","ttl":1}`)
		} else {
			b.processError(ctx, err)
			b.updateHealth(b.getPlayUrlHealth(args.area), -500, "服务器错误")
			return
		}
	}

	newData, err := replaceQn([]byte(data), args.qn, ClientTypeWeb)
	if err != nil {
		b.processError(ctx, err)
		return
	}

	if isNotLogin, err := isResponseNotLogin(data); err != nil {
		b.sugar.Error(err)
	} else if isNotLogin {
		ctx.Write(newData)
		return
	}

	setDefaultHeaders(ctx)
	ctx.Write(newData)

	if err := b.updateEpisodeCache(data, args.epId, getAreaCode(args.area)); err != nil {
		b.sugar.Error(err)
	}

	if b.getAuthByArea(args.area) {
		if err := b.db.InsertOrUpdatePlayURLCache(database.DeviceTypeWeb, formatType, int16(qn), getAreaCode(args.area), isVIP, args.epId, data); err != nil {
			b.sugar.Error(err)
		}
	}
}

func (b *BiliroamingGo) handleAndroidPlayURL(ctx *fasthttp.RequestCtx) {
	if !b.checkRoamingVer(ctx) {
		return
	}

	queryArgs := ctx.URI().QueryArgs()
	args := b.processArgs(queryArgs)

	if args.area == "" {
		writeErrorJSON(ctx, -10403, []byte("抱歉您所在地区不可观看！"))
		return
	}

	if ok := b.checkEpisodeAreaCache(args.epId, getAreaCode(args.area)); !ok {
		writeErrorJSON(ctx, -10403, []byte("抱歉您所在地区不可观看！"))
		return
	}

	// 验证 sign
	if args.appkey == "" && args.sign == "" && args.ts == 0 {
		writeErrorJSON(ctx, -10403, []byte("参数错误！"))
		return
	} else {
		if args.ts <= time.Now().Add(-time.Minute).Unix() {
			writeErrorJSON(ctx, -10403, []byte("参数错误！"))
			return
		}

		values, err := url.ParseQuery(queryArgs.String())
		if err != nil {
			b.processError(ctx, err)
			b.updateHealth(b.getPlayUrlHealth(args.area), -500, "服务器错误")
			return
		}

		values.Del("sign")

		sign, err := getSign(values, getClientTypeFromAppkey(args.appkey), args.ts)
		if err != nil {
			b.processError(ctx, err)
			b.updateHealth(b.getPlayUrlHealth(args.area), -500, "服务器错误")
			return
		}

		if sign != args.sign {
			writeErrorJSON(ctx, -10403, []byte("参数错误！"))
			return
		}
	}

	client := b.getClientByArea(args.area)

	qn := args.qn
	formatType := getFormatType(args.fnval)
	if formatType == database.FormatTypeDash {
		qn = 127
	}

	var isVIP bool
	if b.getAuthByArea(args.area) {
		if ok, status := b.doAuth(ctx, args.accessKey, args.area); !ok {
			return
		} else {
			isVIP = status.isVip
		}

		playurlCache, err := b.db.GetPlayURLCache(database.DeviceTypeAndroid, formatType, int16(qn), getAreaCode(args.area), isVIP, args.epId)
		if err == nil && len(playurlCache.Data) > 0 && playurlCache.UpdatedAt.After(time.Now().Add(-b.config.Cache.PlayUrl)) {
			b.sugar.Debug("Replay from cache: ", playurlCache.Data.String())
			setDefaultHeaders(ctx)
			newData, err := replaceQn(playurlCache.Data, args.qn, ClientTypeAndroid)
			if err != nil {
				b.processError(ctx, err)
				return
			}
			ctx.Write(newData)
			return
		} else if err != nil && !errors.Is(err, sql.ErrNoRows) {
			b.processError(ctx, err)
			b.updateHealth(b.getPlayUrlHealth(args.area), -500, "服务器错误")
			return
		}
	}

	v := url.Values{}
	v.Set("access_key", args.accessKey)
	v.Set("area", args.area)
	v.Set("ep_id", strconv.FormatInt(args.epId, 10))
	v.Set("fnver", "0")

	switch formatType {
	case database.FormatTypeDash:
		v.Set("fnval", "4048")
	case database.FormatTypeMp4:
		v.Set("fnval", "1")
	case database.FormatTypeFlv:
		fallthrough
	default:
		v.Set("fnval", "0")
	}

	v.Set("fourk", "1")
	v.Set("platform", "android")
	v.Set("qn", strconv.Itoa(qn))

	params, err := SignParams(v, ClientTypeAndroid)
	if err != nil {
		b.sugar.Error(err)
		ctx.Error(
			fasthttp.StatusMessage(fasthttp.StatusInternalServerError),
			fasthttp.StatusInternalServerError,
		)
		return
	}

	reverseProxy := b.getReverseProxyByArea(args.area)
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

	reqParams := &HttpRequestParams{
		Method: []byte(fasthttp.MethodGet),
		Url:    []byte(url),
		UserAgent: ctx.UserAgent(),
	}
	data, err := b.doRequestJsonWithRetry(client, reqParams, 2)
	if err != nil {
		if errors.Is(err, ErrorHttpStatusLimited) {
			data = []byte(`{"code":-412,"message":"请求被拦截","ttl":1}`)
		} else {
			b.processError(ctx, err)
			b.updateHealth(b.getPlayUrlHealth(args.area), -500, "服务器错误")
			return
		}
	}

	newData, err := replaceQn([]byte(data), args.qn, ClientTypeAndroid)
	if err != nil {
		b.processError(ctx, err)
		return
	}

	if isNotLogin, err := isResponseNotLogin(data); err != nil {
		b.sugar.Error(err)
	} else if isNotLogin {
		ctx.Write(newData)
		return
	}

	if isLimited, err := isResponseLimited(data); err != nil {
		b.sugar.Error(err)
	} else if isLimited {
		b.updateHealth(b.getPlayUrlHealth(args.area), -412, "请求被拦截")
	} else {
		b.updateHealth(b.getPlayUrlHealth(args.area), 0, "0")
	}

	setDefaultHeaders(ctx)
	ctx.Write(newData)

	if err := b.updateEpisodeCache(data, args.epId, getAreaCode(args.area)); err != nil {
		b.sugar.Error(err)
	}

	if b.getAuthByArea(args.area) {
		if err := b.db.InsertOrUpdatePlayURLCache(database.DeviceTypeAndroid, formatType, int16(qn), getAreaCode(args.area), isVIP, args.epId, data); err != nil {
			b.sugar.Error(err)
		}
	}
}

func (b *BiliroamingGo) handleBstarAndroidPlayURL(ctx *fasthttp.RequestCtx) {
	if !b.checkRoamingVer(ctx) {
		return
	}

	queryArgs := ctx.URI().QueryArgs()
	args := b.processArgs(queryArgs)

	if args.area == "" {
		args.area = "th"
		// writeErrorJSON(ctx, -10403, []byte("抱歉您所在地区不可观看！"))
		// return
	}

	if ok := b.checkEpisodeAreaCache(args.epId, getAreaCode(args.area)); !ok {
		writeErrorJSON(ctx, -10403, []byte("抱歉您所在地区不可观看！"))
		return
	}

	// 验证 sign
	if args.appkey == "" && args.sign == "" && args.ts == 0 {
		writeErrorJSON(ctx, -10403, []byte("参数错误！"))
		return
	} else {
		if args.ts <= time.Now().Add(-time.Minute).Unix() {
			writeErrorJSON(ctx, -10403, []byte("参数错误！"))
			return
		}

		values, err := url.ParseQuery(queryArgs.String())
		if err != nil {
			b.processError(ctx, err)
			b.updateHealth(b.getPlayUrlHealth(args.area), -500, "服务器错误")
			return
		}

		values.Del("sign")

		sign, err := getSign(values, ClientTypeBstarA, args.ts)
		if err != nil {
			b.processError(ctx, err)
			b.updateHealth(b.getPlayUrlHealth(args.area), -500, "服务器错误")
			return
		}

		if sign != args.sign {
			writeErrorJSON(ctx, -10403, []byte("参数错误！"))
			return
		}
	}

	client := b.getClientByArea(args.area)

	qn := args.qn
	formatType := getFormatType(args.fnval)
	if formatType == database.FormatTypeDash {
		qn = 127
	}

	var isVIP bool
	if b.getAuthByArea(args.area) {
		if ok, status := b.doAuth(ctx, args.accessKey, args.area); !ok {
			return
		} else {
			isVIP = status.isVip
		}

		playurlCache, err := b.db.GetPlayURLCache(database.DeviceTypeAndroid, formatType, int16(qn), getAreaCode(args.area), isVIP, args.epId)
		if err == nil && len(playurlCache.Data) > 0 && playurlCache.UpdatedAt.After(time.Now().Add(-b.config.Cache.PlayUrl)) {
			b.sugar.Debug("Replay from cache: ", playurlCache.Data.String())
			setDefaultHeaders(ctx)
			newData, err := replaceQn(playurlCache.Data, args.qn, ClientTypeBstarA)
			if err != nil {
				b.processError(ctx, err)
				return
			}
			ctx.Write(newData)
			return
		} else if err != nil && !errors.Is(err, sql.ErrNoRows) {
			b.processError(ctx, err)
			b.updateHealth(b.getPlayUrlHealth(args.area), -500, "服务器错误")
			return
		}
	}

	v := url.Values{}
	v.Set("access_key", args.accessKey)
	v.Set("ep_id", strconv.FormatInt(args.epId, 10))
	v.Set("fnver", "0")

	switch formatType {
	case database.FormatTypeDash:
		v.Set("fnval", "4048")
	case database.FormatTypeMp4:
		v.Set("fnval", "1")
	case database.FormatTypeFlv:
		fallthrough
	default:
		v.Set("fnval", "0")
	}

	v.Set("fourk", "1")
	v.Set("platform", "android")
	v.Set("s_locale", "zh_SG")
	v.Set("qn", strconv.Itoa(qn))

	params, err := SignParams(v, ClientTypeBstarA)
	if err != nil {
		b.sugar.Error(err)
		ctx.Error(
			fasthttp.StatusMessage(fasthttp.StatusInternalServerError),
			fasthttp.StatusInternalServerError,
		)
		return
	}

	reverseProxy := b.getReverseProxyByArea(args.area)
	if reverseProxy == "" {
		reverseProxy = "api.biliintl.com"
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

	reqParams := &HttpRequestParams{
		Method: []byte(fasthttp.MethodGet),
		Url:    []byte(url),
		UserAgent: ctx.UserAgent(),
	}
	data, err := b.doRequestJsonWithRetry(client, reqParams, 2)
	if err != nil {
		if errors.Is(err, ErrorHttpStatusLimited) {
			data = []byte(`{"code":-412,"message":"请求被拦截","ttl":1}`)
		} else {
			b.processError(ctx, err)
			b.updateHealth(b.HealthPlayUrlTH, -500, "服务器错误")
			return
		}
	}

	newData, err := replaceQn([]byte(data), args.qn, ClientTypeBstarA)
	if err != nil {
		b.processError(ctx, err)
		return
	}

	if isLimited, err := isResponseLimited(data); err != nil {
		b.sugar.Error(err)
	} else if isLimited {
		b.updateHealth(b.HealthPlayUrlTH, -412, "请求被拦截")
	} else {
		b.updateHealth(b.HealthPlayUrlTH, 0, "0")
	}

	setDefaultHeaders(ctx)
	ctx.Write(newData)

	if err := b.updateEpisodeCache(data, args.epId, getAreaCode(args.area)); err != nil {
		b.sugar.Error(err)
	}

	if b.getAuthByArea(args.area) {
		if err := b.db.InsertOrUpdatePlayURLCache(database.DeviceTypeAndroid, formatType, int16(qn), getAreaCode(args.area), isVIP, args.epId, data); err != nil {
			b.sugar.Error(err)
		}
	}
}
