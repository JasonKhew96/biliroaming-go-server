package main

import (
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"time"

	"github.com/valyala/fasthttp"
	"golang.org/x/net/idna"
)

func (b *BiliroamingGo) handleBstarAndroidSubtitle(ctx *fasthttp.RequestCtx) {
	if !b.checkRoamingVer(ctx) {
		return
	}

	queryArgs := ctx.URI().QueryArgs()
	args := b.processArgs(queryArgs)

	args.area = "th"

	client := b.getClientByArea(args.area)

	if b.getAuthByArea(args.area) {
		// if ok, _ := b.doAuth(ctx, accessKey, area, false); !ok {
		// 	return
		// }
		subtitleCache, err := b.db.GetTHSubtitleCache(args.epId)
		if err == nil && len(subtitleCache.Data) > 0 && subtitleCache.UpdatedAt.After(time.Now().Add(-b.config.Cache.THSubtitle)) {
			b.sugar.Debug("Replay from cache: ", subtitleCache.Data.String())
			setDefaultHeaders(ctx)
			ctx.Write(subtitleCache.Data)
			return
		}
	}

	v := url.Values{}
	v.Set("access_key", args.accessKey)
	v.Set("s_locale", "zh_SG")
	v.Set("ep_id", strconv.FormatInt(args.epId, 10))
	v.Set("mobi_app", "bstar_a")

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

	url := fmt.Sprintf("https://%s/intl/gateway/v2/app/subtitle?%s", domain, params)
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
			return
		}
	}

	setDefaultHeaders(ctx)
	ctx.Write(data)

	if b.getAuthByArea(args.area) {
		if err := b.db.InsertOrUpdateTHSubtitleCache(args.epId, data); err != nil {
			b.sugar.Error(err)
		}
	}
}
