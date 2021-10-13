package main

import (
	"fmt"
	"net/url"
	"strconv"
	"time"

	"github.com/JasonKhew96/biliroaming-go-server/database"
	"github.com/valyala/fasthttp"
	"golang.org/x/net/idna"
)

func (b *BiliroamingGo) handleWebPlayURL(ctx *fasthttp.RequestCtx) {
	queryArgs := ctx.URI().QueryArgs()
	args := b.processArgs(queryArgs)

	if args.area == "" {
		writeErrorJSON(ctx, -688, []byte("地理区域限制"))
		return
	}

	cidInt, err := strconv.Atoi(args.cid)
	if err != nil {
		b.processError(ctx, err)
		return
	}
	epidInt, err := strconv.Atoi(args.epId)
	if err != nil {
		b.processError(ctx, err)
		return
	}

	var isVIP bool
	if b.getAuthByArea(args.area) {
		var ok bool
		if ok, isVIP = b.doAuth(ctx, args.accessKey, args.area); !ok {
			return
		}
		playurlCache, err := b.db.GetPlayURLCache(database.DeviceTypeWeb, getAreaCode(args.area), isVIP, cidInt, epidInt)
		if err == nil && playurlCache.JSONData != "" && playurlCache.UpdatedAt.After(time.Now().Add(-time.Duration(b.config.CachePlayURL)*time.Minute)) {
			b.sugar.Debug("Replay from cache: ", playurlCache.JSONData)
			setDefaultHeaders(ctx)
			ctx.Write([]byte(playurlCache.JSONData))
			return
		}
	}

	client := b.getClientByArea(args.area)

	v := url.Values{}
	v.Set("access_key", args.accessKey)
	v.Set("area", args.area)
	v.Set("cid", args.cid)
	v.Set("ep_id", args.epId)
	v.Set("fnver", "0")
	v.Set("fnval", "80")
	v.Set("fourk", "1")
	// v.Set("qn", "120")

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

	data, err := b.doRequestJson(ctx, client, url)
	if err != nil {
		b.processError(ctx, err)
		return
	}

	setDefaultHeaders(ctx)
	ctx.Write(data)

	if b.getAuthByArea(args.area) {
		b.db.InsertOrUpdatePlayURLCache(database.DeviceTypeWeb, getAreaCode(args.area), isVIP, cidInt, epidInt, string(data))
	}
}

func (b *BiliroamingGo) handleAndroidPlayURL(ctx *fasthttp.RequestCtx) {
	queryArgs := ctx.URI().QueryArgs()
	args := b.processArgs(queryArgs)

	if args.area == "" {
		writeErrorJSON(ctx, -688, []byte("地理区域限制"))
		return
	}

	client := b.getClientByArea(args.area)

	cidInt, err := strconv.Atoi(args.cid)
	if err != nil {
		b.processError(ctx, err)
		return
	}
	epidInt, err := strconv.Atoi(args.epId)
	if err != nil {
		b.processError(ctx, err)
		return
	}

	var isVIP bool
	if b.getAuthByArea(args.area) {
		var ok bool
		if ok, isVIP = b.doAuth(ctx, args.accessKey, args.area); !ok {
			return
		}

		playurlCache, err := b.db.GetPlayURLCache(database.DeviceTypeAndroid, getAreaCode(args.area), isVIP, cidInt, epidInt)
		if err == nil && playurlCache.JSONData != "" && playurlCache.UpdatedAt.After(time.Now().Add(-time.Duration(b.config.CachePlayURL)*time.Minute)) {
			b.sugar.Debug("Replay from cache: ", playurlCache.JSONData)
			setDefaultHeaders(ctx)
			ctx.Write([]byte(playurlCache.JSONData))
			return
		}
	}

	v := url.Values{}
	v.Set("access_key", args.accessKey)
	v.Set("area", args.area)
	v.Set("cid", args.cid)
	v.Set("ep_id", args.epId)
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

	data, err := b.doRequestJson(ctx, client, url)
	if err != nil {
		b.processError(ctx, err)
		return
	}

	if isLimited, err := isResponseLimited(data); err != nil {
		b.sugar.Error(err)
	} else {
		b.updateHealth(b.getPlayUrlHealth(args.area), isLimited)
	}

	setDefaultHeaders(ctx)
	ctx.Write(data)

	if b.getAuthByArea(args.area) {
		b.db.InsertOrUpdatePlayURLCache(database.DeviceTypeAndroid, getAreaCode(args.area), isVIP, cidInt, epidInt, string(data))
	}
}

func (b *BiliroamingGo) handleBstarAndroidPlayURL(ctx *fasthttp.RequestCtx) {
	queryArgs := ctx.URI().QueryArgs()
	args := b.processArgs(queryArgs)

	if args.area == "" {
		args.area = "th"
		// writeErrorJSON(ctx, -688, []byte("地理区域限制"))
		// return
	}

	client := b.getClientByArea(args.area)

	cidInt, err := strconv.Atoi(args.cid)
	if err != nil {
		b.processError(ctx, err)
		return
	}
	epidInt, err := strconv.Atoi(args.epId)
	if err != nil {
		b.processError(ctx, err)
		return
	}

	var isVIP bool
	if b.getAuthByArea(args.area) {
		var ok bool
		if ok, isVIP = b.doAuth(ctx, args.accessKey, args.area); !ok {
			return
		}

		playurlCache, err := b.db.GetPlayURLCache(database.DeviceTypeAndroid, getAreaCode(args.area), isVIP, cidInt, epidInt)
		if err == nil && playurlCache.JSONData != "" && playurlCache.UpdatedAt.After(time.Now().Add(-time.Duration(b.config.CachePlayURL)*time.Minute)) {
			b.sugar.Debug("Replay from cache: ", playurlCache.JSONData)
			setDefaultHeaders(ctx)
			ctx.Write([]byte(playurlCache.JSONData))
			return
		}
	}

	v := url.Values{}
	v.Set("access_key", args.accessKey)
	v.Set("area", args.area)
	v.Set("cid", args.cid)
	v.Set("ep_id", args.epId)
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

	data, err := b.doRequestJson(ctx, client, url)
	if err != nil {
		b.processError(ctx, err)
		return
	}

	if isLimited, err := isResponseLimited(data); err != nil {
		b.sugar.Error(err)
	} else {
		b.updateHealth(b.HealthPlayUrlTH, isLimited)
	}

	setDefaultHeaders(ctx)
	ctx.Write(data)

	if b.getAuthByArea(args.area) {
		b.db.InsertOrUpdatePlayURLCache(database.DeviceTypeAndroid, getAreaCode(args.area), isVIP, cidInt, epidInt, string(data))
	}
}
