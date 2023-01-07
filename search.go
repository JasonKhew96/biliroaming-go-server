package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strconv"

	"github.com/JasonKhew96/biliroaming-go-server/entity/android"
	"github.com/JasonKhew96/biliroaming-go-server/entity/bstar"
	"github.com/JasonKhew96/biliroaming-go-server/entity/web"
	"github.com/mailru/easyjson"
	"github.com/valyala/fasthttp"
	"golang.org/x/net/idna"
)

func (b *BiliroamingGo) addSearchAds(data []byte, clientType ClientType) ([]byte, error) {
	if b.config.CustomSearch.Data == "" {
		return data, nil
	}

	var customData interface{}
	if err := json.Unmarshal([]byte(b.config.CustomSearch.Data), &customData); err != nil {
		return nil, err
	}

	switch clientType {
	case ClientTypeBstarA:
		searchResult := &bstar.SearchResult{}
		if err := easyjson.Unmarshal(data, searchResult); err != nil {
			return nil, err
		}
		searchResult.Data.Items = append([]interface{}{customData}, searchResult.Data.Items...)
		searchResult.Data.Total++
		return easyjson.Marshal(searchResult)
	default:
		searchResult := &android.SearchResult{}
		if err := easyjson.Unmarshal(data, searchResult); err != nil {
			return nil, err
		}
		searchResult.Data.Items = append([]interface{}{customData}, searchResult.Data.Items...)
		searchResult.Data.Total++
		return easyjson.Marshal(searchResult)
	}
}

func (b *BiliroamingGo) addWebSearchAds(data []byte) ([]byte, error) {
	if b.config.CustomSearch.WebData == "" {
		return data, nil
	}

	var customData interface{}
	if err := json.Unmarshal([]byte(b.config.CustomSearch.WebData), &customData); err != nil {
		return nil, err
	}

	searchResult := &web.SearchResult{}
	if err := easyjson.Unmarshal(data, searchResult); err != nil {
		return nil, err
	}
	searchResult.Data.Result = append([]interface{}{customData}, searchResult.Data.Result...)
	searchResult.Data.NumResults++
	return easyjson.Marshal(searchResult)
}

func (b *BiliroamingGo) handleAndroidSearch(ctx *fasthttp.RequestCtx) {
	if !b.checkRoamingVer(ctx) {
		return
	}

	if !b.searchLimiter.Allow() {
		writeErrorJSON(ctx, ERROR_CODE_TOO_MANY_REQUESTS, MSG_ERROR_TOO_MANY_REQUESTS)
		return
	}

	queryArgs := ctx.URI().QueryArgs()
	args := b.processArgs(queryArgs)

	if args.area == "" || args.area == "th" {
		writeErrorJSON(ctx, ERROR_CODE_GEO_RESTRICED, MSG_ERROR_GEO_RESTRICTED)
		return
	}

	client := b.getClientByArea(args.area)

	if args.keyword == "" {
		writeErrorJSON(ctx, ERROR_CODE_MISSING_KEYWORD, MSG_ERROR_MISSING_KEYWORD)
		return
	}

	clientType := getClientPlatform(ctx, args.appkey)

	v := url.Values{}
	v.Set("access_key", args.accessKey)
	v.Set("area", args.area)
	v.Set("build", "6400000")
	v.Set("highlight", "1")
	v.Set("keyword", args.keyword)
	v.Set("type", strconv.Itoa(args.aType))
	v.Set("mobi_app", clientType.String())
	v.Set("platform", "android")
	v.Set("pn", strconv.Itoa(args.pn))

	params, err := SignParams(v, clientType)
	if err != nil {
		b.processError(ctx, err)
		return
	}

	reverseProxy := b.getReverseSearchProxyByArea(args.area)
	if reverseProxy == "" {
		reverseProxy = "app.bilibili.com"
	}
	domain, err := idna.New().ToASCII(reverseProxy)
	if err != nil {
		b.processError(ctx, err)
		return
	}

	url := fmt.Sprintf("https://%s/x/v2/search/type?%s", domain, params)
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
			b.updateHealth(b.getSearchHealth(args.area), ERROR_CODE_INTERNAL_SERVER, MSG_ERROR_INTERNAL_SERVER)
			return
		}
	}

	if isLimited, err := isResponseLimited(data); err != nil {
		b.sugar.Error(err)
	} else if isLimited {
		b.updateHealth(b.getSearchHealth(args.area), ERROR_CODE_TOO_MANY_REQUESTS, MSG_ERROR_TOO_MANY_REQUESTS)
	} else {
		b.updateHealth(b.getSearchHealth(args.area), 0, "0")
	}

	var dataByte []byte
	if args.pn == 1 {
		dataByte, err = b.addSearchAds(data, ClientTypeAndroid)
		if err != nil {
			b.processError(ctx, err)
			dataByte = data
		}
	} else {
		dataByte = data
	}

	setDefaultHeaders(ctx)
	ctx.Write(dataByte)
}

func (b *BiliroamingGo) handleBstarAndroidSearch(ctx *fasthttp.RequestCtx) {
	if !b.checkRoamingVer(ctx) {
		return
	}

	if !b.searchLimiter.Allow() {
		writeErrorJSON(ctx, ERROR_CODE_TOO_MANY_REQUESTS, MSG_ERROR_TOO_MANY_REQUESTS)
		return
	}

	queryArgs := ctx.URI().QueryArgs()
	args := b.processArgs(queryArgs)

	args.area = "th"

	client := b.getClientByArea(args.area)

	if args.keyword == "" {
		writeErrorJSON(ctx, ERROR_CODE_MISSING_KEYWORD, MSG_ERROR_MISSING_KEYWORD)
		return
	}

	v := url.Values{}
	v.Set("access_key", args.accessKey)
	v.Set("build", "1080003")
	v.Set("keyword", args.keyword)
	v.Set("platform", "android")
	v.Set("s_locale", "zh_SG")
	v.Set("type", strconv.Itoa(args.aType))
	v.Set("mobi_app", "bstar_a")
	v.Set("pn", strconv.Itoa(args.pn))

	params, err := SignParams(v, ClientTypeBstarA)
	if err != nil {
		b.processError(ctx, err)
		return
	}

	reverseProxy := b.getReverseSearchProxyByArea(args.area)
	if reverseProxy == "" {
		reverseProxy = "app.biliintl.com"
	}
	domain, err := idna.New().ToASCII(reverseProxy)
	if err != nil {
		b.processError(ctx, err)
		return
	}

	url := fmt.Sprintf("https://%s/intl/gateway/v2/app/search/type?%s", domain, params)
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
			b.updateHealth(b.HealthSearchTH, ERROR_CODE_INTERNAL_SERVER, MSG_ERROR_INTERNAL_SERVER)
			return
		}
	}

	if isLimited, err := isResponseLimited(data); err != nil {
		b.sugar.Error(err)
	} else if isLimited {
		b.updateHealth(b.HealthSearchTH, ERROR_CODE_TOO_MANY_REQUESTS, MSG_ERROR_TOO_MANY_REQUESTS)
	} else {
		b.updateHealth(b.HealthSearchTH, 0, "0")
	}

	var dataByte []byte
	if args.pn == 1 {
		dataByte, err = b.addSearchAds(data, ClientTypeBstarA)
		if err != nil {
			b.processError(ctx, err)
			dataByte = data
		}
	} else {
		dataByte = data
	}

	setDefaultHeaders(ctx)
	ctx.Write(dataByte)
}

func (b *BiliroamingGo) handleWebSearch(ctx *fasthttp.RequestCtx) {
	if !b.checkRoamingVer(ctx) {
		return
	}

	if !b.searchLimiter.Allow() {
		writeErrorJSON(ctx, ERROR_CODE_TOO_MANY_REQUESTS, MSG_ERROR_TOO_MANY_REQUESTS)
		return
	}

	queryArgs := ctx.URI().QueryArgs()
	args := b.processArgs(queryArgs)

	if args.area == "" || args.area == "th" {
		writeErrorJSON(ctx, ERROR_CODE_GEO_RESTRICED, MSG_ERROR_GEO_RESTRICTED)
		return
	}

	client := b.getClientByArea(args.area)

	if args.keyword == "" {
		writeErrorJSON(ctx, ERROR_CODE_GEO_RESTRICED, MSG_ERROR_GEO_RESTRICTED)
		return
	}

	v := url.Values{}
	v.Set("area", args.area)
	v.Set("search_type", "media_bangumi")
	v.Set("keyword", args.keyword)
	v.Set("page", strconv.Itoa(args.page))

	params, err := SignParams(v, ClientTypeAndroid)
	if err != nil {
		b.processError(ctx, err)
		return
	}

	reverseProxy := b.getReverseWebSearchProxyByArea(args.area)
	if reverseProxy == "" {
		reverseProxy = "api.bilibili.com"
	}
	domain, err := idna.New().ToASCII(reverseProxy)
	if err != nil {
		b.processError(ctx, err)
		return
	}

	url := fmt.Sprintf("https://%s/x/web-interface/search/type?%s", domain, params)
	b.sugar.Debug("New url: ", url)

	reqParams := &HttpRequestParams{
		Method:    []byte(fasthttp.MethodGet),
		Url:       []byte(url),
		UserAgent: ctx.UserAgent(),
	}
	buvid3Key := []byte("buvid3")
	buvid3Value := ctx.Request.Header.CookieBytes(buvid3Key)
	if buvid3Value == nil || (buvid3Value != nil && len(buvid3Value) == 0) {
		writeErrorJSON(ctx, ERROR_CODE_MISSING_COOKIE, MSG_ERROR_MISSING_COOKIE)
		return
	}
	reqParams.Cookie = append(reqParams.Cookie, HttpCookiesParams{
		Key:   buvid3Key,
		Value: buvid3Value,
	})
	data, err := b.doRequestJson(client, reqParams)
	if err != nil {
		if errors.Is(err, ErrorHttpStatusLimited) {
			data = []byte(`{"code":-412,"message":"请求被拦截"}`)
		} else {
			b.processError(ctx, err)
			b.updateHealth(b.getSearchHealth(args.area), ERROR_CODE_INTERNAL_SERVER, MSG_ERROR_INTERNAL_SERVER)
			return
		}
	}

	if isLimited, err := isResponseLimited(data); err != nil {
		b.sugar.Error(err)
	} else if isLimited {
		b.updateHealth(b.getSearchHealth(args.area), ERROR_CODE_TOO_MANY_REQUESTS, MSG_ERROR_TOO_MANY_REQUESTS)
	} else {
		b.updateHealth(b.getSearchHealth(args.area), 0, "0")
	}

	dataByte, err := b.addWebSearchAds([]byte(data))
	if err != nil {
		b.processError(ctx, err)
		return
	}

	setDefaultHeaders(ctx)
	ctx.Write(dataByte)
}
