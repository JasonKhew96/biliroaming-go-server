package main

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/valyala/fasthttp"
	"golang.org/x/net/idna"
)

func (b *BiliroamingGo) addSearchAds(data string) string {
	if b.config.CustomSearch.Data == "" {
		return data
	}

	old := "\"items\":["
	new := old + b.config.CustomSearch.Data + ","
	newData := strings.Replace(data, old, new, 1)

	if isValidJson(newData) {
		return newData
	}

	b.sugar.Debugf("isValidJson: %t\n%s", isValidJson, newData)
	return data
}

func (b *BiliroamingGo) handleAndroidSearch(ctx *fasthttp.RequestCtx) {
	queryArgs := ctx.URI().QueryArgs()
	args := b.processArgs(queryArgs)

	if args.area == "" || args.area == "th" {
		writeErrorJSON(ctx, -688, []byte("地理区域限制"))
		return
	}

	client := b.getClientByArea(args.area)

	if args.keyword == "" {
		writeErrorJSON(ctx, -688, []byte("地理区域限制"))
		return
	}

	v := url.Values{}
	v.Set("access_key", args.accessKey)
	v.Set("area", args.area)
	v.Set("build", "6400000")
	v.Set("highlight", "1")
	v.Set("keyword", args.keyword)
	v.Set("type", "7")
	v.Set("mobi_app", "android")
	v.Set("platform", "android")
	v.Set("pn", args.pn)

	params, err := SignParams(v, ClientTypeAndroid)
	if err != nil {
		b.sugar.Error(err)
		ctx.Error(
			fasthttp.StatusMessage(fasthttp.StatusInternalServerError),
			fasthttp.StatusInternalServerError,
		)
		return
	}

	reverseProxy := b.getReverseSearchProxyByArea(args.area)
	if reverseProxy == "" {
		reverseProxy = "app.bilibili.com"
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

	url := fmt.Sprintf("https://%s/x/v2/search/type?%s", domain, params)
	b.sugar.Debug("New url: ", url)

	data, err := b.doRequestJson(ctx, client, url)
	if err != nil {
		b.processError(ctx, err)
		return
	}

	if isLimited, err := isResponseLimited(data); err != nil {
		b.sugar.Error(err)
	} else {
		b.updateHealth(b.getSearchHealth(args.area), isLimited)
	}

	data = b.addSearchAds(data)

	setDefaultHeaders(ctx)
	ctx.WriteString(data)
}

func (b *BiliroamingGo) handleBstarAndroidSearch(ctx *fasthttp.RequestCtx) {
	queryArgs := ctx.URI().QueryArgs()
	args := b.processArgs(queryArgs)

	if args.area == "" {
		args.area = "th"
		// writeErrorJSON(ctx, -688, []byte("地理区域限制"))
		// return
	}

	client := b.getClientByArea(args.area)

	if args.keyword == "" {
		writeErrorJSON(ctx, -400, []byte("请求错误"))
		return
	}

	if b.getAuthByArea(args.area) {
		if ok, _ := b.doAuth(ctx, args.accessKey, args.area); !ok {
			return
		}
	}

	v := url.Values{}
	v.Set("access_key", args.accessKey)
	v.Set("area", args.area)
	v.Set("build", "1080003")
	v.Set("keyword", args.keyword)
	v.Set("s_locale", "zh_SG")
	v.Set("type", "7")
	v.Set("mobi_app", "bstar_a")
	v.Set("pn", args.pn)

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
		reverseProxy = "app.biliintl.com"
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

	url := fmt.Sprintf("https://%s/intl/gateway/v2/app/search/type?%s", domain, params)
	b.sugar.Debug("New url: ", url)

	data, err := b.doRequestJson(ctx, client, url)
	if err != nil {
		b.processError(ctx, err)
		return
	}

	if isLimited, err := isResponseLimited(data); err != nil {
		b.sugar.Error(err)
	} else {
		b.updateHealth(b.HealthSearchTH, isLimited)
	}

	data = b.addSearchAds(data)

	setDefaultHeaders(ctx)
	ctx.WriteString(data)
}
