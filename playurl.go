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
		writeErrorJSON(ctx, ERROR_CODE_GEO_RESTRICED, MSG_ERROR_GEO_RESTRICTED)
		return
	}

	// 验证 epId
	if args.epId == 0 {
		writeErrorJSON(ctx, ERROR_CODE_PARAMETERS, MSG_ERROR_PARAMETERS)
		return
	}

	if ok := b.checkEpisodeAreaCache(args.epId, getAreaCode(args.area)); !ok {
		writeErrorJSON(ctx, ERROR_CODE_GEO_RESTRICED, MSG_ERROR_GEO_RESTRICTED)
		return
	}

	// 验证 sign
	if args.appkey != "" && args.sign != "" && args.ts != 0 {
		if args.ts <= time.Now().Add(-time.Minute).Unix() {
			writeErrorJSON(ctx, ERROR_CODE_PARAMETERS, MSG_ERROR_PARAMETERS)
			return
		}

		values, err := url.ParseQuery(queryArgs.String())
		if err != nil {
			b.processError(ctx, err)
			b.updateHealth(b.getPlayUrlHealth(args.area), ERROR_CODE_INTERNAL_SERVER, MSG_ERROR_INTERNAL_SERVER)
			return
		}

		values.Del("sign")

		sign, err := getSign(values, getClientTypeFromAppkey(args.appkey), args.ts)
		if err != nil {
			b.processError(ctx, err)
			b.updateHealth(b.getPlayUrlHealth(args.area), ERROR_CODE_INTERNAL_SERVER, MSG_ERROR_INTERNAL_SERVER)
			return
		}

		if sign != args.sign {
			writeErrorJSON(ctx, ERROR_CODE_PARAMETERS, MSG_ERROR_PARAMETERS)
			return
		}
	}

	qn := args.qn
	formatType := getFormatType(args.fnval)
	if formatType == database.FormatTypeDash {
		qn = 127
	}

	clientType := getClientPlatform(ctx, args.appkey)

	var status *userStatus
	if b.getAuthByArea(args.area) {
		var ok bool
		ok, status = b.doAuth(ctx, args.accessKey, clientType, args.area, false)
		if !ok {
			return
		}

		playurlCache, err := b.db.GetPlayURLCache(database.DeviceTypeWeb, formatType, int16(qn), getAreaCode(args.area), status.isVip, false, args.epId)
		if err == nil && len(playurlCache.Data) > 0 && playurlCache.UpdatedAt.After(time.Now().Add(-b.config.Cache.PlayUrl)) {
			if b.config.VipOnly && !status.isVip {
				writeErrorJSON(ctx, ERROR_CODE_VIP_ONLY, MSG_ERROR_VIP_ONLY)
				return
			}

			b.sugar.Debug("Replay from cache: ", playurlCache.Data.String())
			setDefaultHeaders(ctx)
			data, err := replaceQn(playurlCache.Data, args.qn, ClientTypeWeb)
			if err != nil {
				b.processError(ctx, err)
				return
			}
			ctx.Write(data)
			return
		} else if err != nil && !errors.Is(err, sql.ErrNoRows) {
			b.processError(ctx, err)
			b.updateHealth(b.getPlayUrlHealth(args.area), ERROR_CODE_INTERNAL_SERVER, MSG_ERROR_INTERNAL_SERVER)
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

	params, err := SignParams(v, clientType)
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
		Method:    []byte(fasthttp.MethodGet),
		Url:       []byte(url),
		UserAgent: ctx.UserAgent(),
	}
	data, err := b.doRequestJson(client, reqParams)
	if err != nil {
		if errors.Is(err, ErrorHttpStatusLimited) {
			data = []byte(`{"code":-412,"message":"请求被拦截"}`)
		} else {
			b.processError(ctx, err)
			b.updateHealth(b.getPlayUrlHealth(args.area), ERROR_CODE_INTERNAL_SERVER, MSG_ERROR_INTERNAL_SERVER)
			return
		}
	}

	setDefaultHeaders(ctx)

	if isNotLogin, err := isResponseNotLogin(data); err != nil {
		b.sugar.Error(err)
	} else if isNotLogin {
		ctx.Write(data)
		return
	}

	if isLimited, err := isResponseLimited(data); err != nil {
		b.sugar.Error(err)
	} else if isLimited {
		b.updateHealth(b.getPlayUrlHealth(args.area), ERROR_CODE_TOO_MANY_REQUESTS, MSG_ERROR_TOO_MANY_REQUESTS)
	} else {
		b.updateHealth(b.getPlayUrlHealth(args.area), 0, "0")
	}

	data, err = replaceQn(data, args.qn, ClientTypeWeb)
	if err != nil {
		b.processError(ctx, err)
		return
	}

	if err := b.updateEpisodeCache(data, args.epId, getAreaCode(args.area)); err != nil {
		b.sugar.Error(err)
	}

	if b.getAuthByArea(args.area) {
		if err := b.db.InsertOrUpdatePlayURLCache(database.DeviceTypeWeb, formatType, int16(qn), getAreaCode(args.area), status.isVip, false, args.epId, data); err != nil {
			b.sugar.Error(err)
		}
	}

	if ok, isStatusVip, err := playUrlVipStatus(data, ClientTypeWeb); err != nil {
		b.sugar.Error(err)
	} else if ok && isStatusVip != status.isVip {
		delete(b.accessKeys, args.accessKey)
		if ok, _ := b.doAuth(ctx, args.accessKey, clientType, args.area, true); !ok {
			return
		}
		writeErrorJSON(ctx, ERROR_CODE_VIP_STATUS, MSG_ERROR_VIP_STATUS)
		return
	}

	if b.config.VipOnly && !status.isVip {
		writeErrorJSON(ctx, ERROR_CODE_VIP_ONLY, MSG_ERROR_VIP_ONLY)
		return
	}

	ctx.Write(data)
}

func (b *BiliroamingGo) handleAndroidPlayURL(ctx *fasthttp.RequestCtx) {
	if !b.checkRoamingVer(ctx) {
		return
	}

	queryArgs := ctx.URI().QueryArgs()
	args := b.processArgs(queryArgs)

	if args.area == "" {
		writeErrorJSON(ctx, ERROR_CODE_GEO_RESTRICED, MSG_ERROR_GEO_RESTRICTED)
		return
	}

	// 验证 epId
	if args.epId == 0 {
		writeErrorJSON(ctx, ERROR_CODE_PARAMETERS, MSG_ERROR_PARAMETERS)
		return
	}

	if ok := b.checkEpisodeAreaCache(args.epId, getAreaCode(args.area)); !ok {
		writeErrorJSON(ctx, ERROR_CODE_GEO_RESTRICED, MSG_ERROR_GEO_RESTRICTED)
		return
	}

	// 验证 sign
	if args.appkey == "" && args.sign == "" && args.ts == 0 {
		writeErrorJSON(ctx, ERROR_CODE_PARAMETERS, MSG_ERROR_PARAMETERS)
		return
	} else {
		if args.ts <= time.Now().Add(-time.Minute).Unix() {
			writeErrorJSON(ctx, ERROR_CODE_PARAMETERS, MSG_ERROR_PARAMETERS)
			return
		}

		values, err := url.ParseQuery(queryArgs.String())
		if err != nil {
			b.processError(ctx, err)
			b.updateHealth(b.getPlayUrlHealth(args.area), ERROR_CODE_INTERNAL_SERVER, MSG_ERROR_INTERNAL_SERVER)
			return
		}

		values.Del("sign")

		sign, err := getSign(values, getClientTypeFromAppkey(args.appkey), args.ts)
		if err != nil {
			b.processError(ctx, err)
			b.updateHealth(b.getPlayUrlHealth(args.area), ERROR_CODE_INTERNAL_SERVER, MSG_ERROR_INTERNAL_SERVER)
			return
		}

		if sign != args.sign {
			writeErrorJSON(ctx, ERROR_CODE_PARAMETERS, MSG_ERROR_PARAMETERS)
			return
		}
	}

	client := b.getClientByArea(args.area)

	qn := args.qn
	formatType := getFormatType(args.fnval)
	if formatType == database.FormatTypeDash {
		qn = 127
	}

	clientType := getClientPlatform(ctx, args.appkey)

	var status *userStatus
	if b.getAuthByArea(args.area) {
		var ok bool
		ok, status = b.doAuth(ctx, args.accessKey, clientType, args.area, false)
		if !ok {
			return
		}

		playurlCache, err := b.db.GetPlayURLCache(database.DeviceTypeAndroid, formatType, int16(qn), getAreaCode(args.area), status.isVip, false, args.epId)
		if err == nil && len(playurlCache.Data) > 0 && playurlCache.UpdatedAt.After(time.Now().Add(-b.config.Cache.PlayUrl)) {
			if b.config.VipOnly && !status.isVip {
				writeErrorJSON(ctx, ERROR_CODE_VIP_ONLY, MSG_ERROR_VIP_ONLY)
				return
			}

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
			b.updateHealth(b.getPlayUrlHealth(args.area), ERROR_CODE_INTERNAL_SERVER, MSG_ERROR_INTERNAL_SERVER)
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

	params, err := SignParams(v, clientType)
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

	url := fmt.Sprintf("https://%s/pgc/player/api/playurl?%s", domain, params)
	b.sugar.Debug("New url: ", url)

	reqParams := &HttpRequestParams{
		Method:    []byte(fasthttp.MethodGet),
		Url:       []byte(url),
		UserAgent: ctx.UserAgent(),
	}
	data, err := b.doRequestJson(client, reqParams)
	if err != nil {
		if errors.Is(err, ErrorHttpStatusLimited) {
			data = []byte(`{"code":-412,"message":"请求被拦截"}`)
		} else {
			b.processError(ctx, err)
			b.updateHealth(b.getPlayUrlHealth(args.area), ERROR_CODE_INTERNAL_SERVER, MSG_ERROR_INTERNAL_SERVER)
			return
		}
	}

	setDefaultHeaders(ctx)

	if isNotLogin, err := isResponseNotLogin(data); err != nil {
		b.sugar.Error(err)
	} else if isNotLogin {
		ctx.Write(data)
		return
	}

	if isLimited, err := isResponseLimited(data); err != nil {
		b.sugar.Error(err)
	} else if isLimited {
		b.updateHealth(b.getPlayUrlHealth(args.area), ERROR_CODE_TOO_MANY_REQUESTS, MSG_ERROR_TOO_MANY_REQUESTS)
	} else {
		b.updateHealth(b.getPlayUrlHealth(args.area), 0, "0")
	}

	data, err = replaceQn(data, args.qn, ClientTypeAndroid)
	if err != nil {
		b.processError(ctx, err)
		return
	}

	if err := b.updateEpisodeCache(data, args.epId, getAreaCode(args.area)); err != nil {
		b.sugar.Error(err)
	}

	if b.getAuthByArea(args.area) {
		if err := b.db.InsertOrUpdatePlayURLCache(database.DeviceTypeAndroid, formatType, int16(qn), getAreaCode(args.area), status.isVip, false, args.epId, data); err != nil {
			b.sugar.Error(err)
		}
	}

	if ok, isStatusVip, err := playUrlVipStatus(data, ClientTypeAndroid); err != nil {
		b.sugar.Error(err)
	} else if ok && isStatusVip != status.isVip {
		delete(b.accessKeys, args.accessKey)
		if ok, _ := b.doAuth(ctx, args.accessKey, clientType, args.area, true); !ok {
			return
		}
		writeErrorJSON(ctx, ERROR_CODE_VIP_STATUS, MSG_ERROR_VIP_STATUS)
		return
	}

	if b.config.VipOnly && !status.isVip {
		writeErrorJSON(ctx, ERROR_CODE_VIP_ONLY, MSG_ERROR_VIP_ONLY)
		return
	}

	ctx.Write(data)
}

func (b *BiliroamingGo) handleBstarAndroidPlayURL(ctx *fasthttp.RequestCtx) {
	if !b.checkRoamingVer(ctx) {
		return
	}

	queryArgs := ctx.URI().QueryArgs()
	args := b.processArgs(queryArgs)

	args.area = "th"

	// 验证 epId
	if args.epId == 0 {
		writeErrorJSON(ctx, ERROR_CODE_PARAMETERS, MSG_ERROR_PARAMETERS)
		return
	}

	if ok := b.checkEpisodeAreaCache(args.epId, getAreaCode(args.area)); !ok {
		writeErrorJSON(ctx, ERROR_CODE_GEO_RESTRICED, MSG_ERROR_GEO_RESTRICTED)
		return
	}

	// 验证 sign
	if args.appkey == "" && args.sign == "" && args.ts == 0 {
		writeErrorJSON(ctx, ERROR_CODE_PARAMETERS, MSG_ERROR_PARAMETERS)
		return
	} else {
		if args.ts <= time.Now().Add(-time.Minute).Unix() {
			writeErrorJSON(ctx, ERROR_CODE_PARAMETERS, MSG_ERROR_PARAMETERS)
			return
		}

		values, err := url.ParseQuery(queryArgs.String())
		if err != nil {
			b.processError(ctx, err)
			b.updateHealth(b.getPlayUrlHealth(args.area), ERROR_CODE_INTERNAL_SERVER, MSG_ERROR_INTERNAL_SERVER)
			return
		}

		values.Del("sign")

		sign, err := getSign(values, ClientTypeBstarA, args.ts)
		if err != nil {
			b.processError(ctx, err)
			b.updateHealth(b.getPlayUrlHealth(args.area), ERROR_CODE_INTERNAL_SERVER, MSG_ERROR_INTERNAL_SERVER)
			return
		}

		if sign != args.sign {
			writeErrorJSON(ctx, ERROR_CODE_PARAMETERS, MSG_ERROR_PARAMETERS)
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
	var status *userStatus
	if b.getAuthByArea(args.area) {
		var ok bool
		ok, status = b.doAuth(ctx, args.accessKey, getClientPlatform(ctx, args.appkey), args.area, false)
		if !ok {
			return
		} else {
			isVIP = status.isVip
		}

		playurlCache, err := b.db.GetPlayURLCache(database.DeviceTypeAndroid, formatType, int16(qn), getAreaCode(args.area), isVIP, args.preferCodeType, args.epId)
		if err == nil && len(playurlCache.Data) > 0 && playurlCache.UpdatedAt.After(time.Now().Add(-b.config.Cache.PlayUrl)) {
			if b.config.VipOnly && !status.isVip {
				writeErrorJSON(ctx, ERROR_CODE_VIP_ONLY, MSG_ERROR_VIP_ONLY)
				return
			}

			b.sugar.Debug("Replay from cache: ", playurlCache.Data.String())
			setDefaultHeaders(ctx)
			data, err := replaceQn(playurlCache.Data, args.qn, ClientTypeBstarA)
			if err != nil {
				b.processError(ctx, err)
				return
			}
			ctx.Write(data)
			return
		} else if err != nil && !errors.Is(err, sql.ErrNoRows) {
			b.processError(ctx, err)
			b.updateHealth(b.getPlayUrlHealth(args.area), ERROR_CODE_INTERNAL_SERVER, MSG_ERROR_INTERNAL_SERVER)
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
	if args.preferCodeType {
		v.Set("prefer_code_type", "1")
	}

	params, err := SignParams(v, ClientTypeBstarA)
	if err != nil {
		b.processError(ctx, err)
		return
	}

	reverseProxy := b.getReverseProxyByArea(args.area)
	if reverseProxy == "" {
		reverseProxy = "api.biliintl.com"
	}
	domain, err := idna.New().ToASCII(reverseProxy)
	if err != nil {
		b.processError(ctx, err)
		return
	}

	url := fmt.Sprintf("https://%s/intl/gateway/v2/ogv/playurl?%s", domain, params)
	b.sugar.Debug("New url: ", url)

	reqParams := &HttpRequestParams{
		Method:    []byte(fasthttp.MethodGet),
		Url:       []byte(url),
		UserAgent: ctx.UserAgent(),
	}
	data, err := b.doRequestJson(client, reqParams)
	if err != nil {
		if errors.Is(err, ErrorHttpStatusLimited) {
			data = []byte(`{"code":-412,"message":"请求被拦截"}`)
		} else {
			b.processError(ctx, err)
			b.updateHealth(b.HealthPlayUrlTH, ERROR_CODE_INTERNAL_SERVER, MSG_ERROR_INTERNAL_SERVER)
			return
		}
	}

	setDefaultHeaders(ctx)

	if isNotLogin, err := isResponseNotLogin(data); err != nil {
		b.sugar.Error(err)
	} else if isNotLogin {
		ctx.Write(data)
		return
	}

	if isLimited, err := isResponseLimited(data); err != nil {
		b.sugar.Error(err)
	} else if isLimited {
		b.updateHealth(b.HealthPlayUrlTH, ERROR_CODE_TOO_MANY_REQUESTS, MSG_ERROR_TOO_MANY_REQUESTS)
	} else {
		b.updateHealth(b.HealthPlayUrlTH, 0, "0")
	}

	data, err = replaceQn(data, args.qn, ClientTypeBstarA)
	if err != nil {
		b.processError(ctx, err)
		return
	}

	if err := b.updateEpisodeCache(data, args.epId, getAreaCode(args.area)); err != nil {
		b.sugar.Error(err)
	}

	if b.getAuthByArea(args.area) {
		if err := b.db.InsertOrUpdatePlayURLCache(database.DeviceTypeAndroid, formatType, int16(qn), getAreaCode(args.area), isVIP, args.preferCodeType, args.epId, data); err != nil {
			b.sugar.Error(err)
		}
	}

	if b.config.VipOnly && !status.isVip {
		writeErrorJSON(ctx, ERROR_CODE_VIP_ONLY, MSG_ERROR_VIP_ONLY)
		return
	}

	ctx.Write(data)
}
