package main

import (
	"fmt"
	"net/url"
	"strconv"
	"time"

	"github.com/valyala/fasthttp"
	"golang.org/x/net/idna"
)

func (b *BiliroamingGo) handleBstarAndroidSubtitle(ctx *fasthttp.RequestCtx) {
	queryArgs := ctx.URI().QueryArgs()
	args := b.processArgs(queryArgs)

	if args.area == "" {
		args.area = "th"
		// writeErrorJSON(ctx, -688, []byte("地理区域限制"))
		// return
	}

	client := b.getClientByArea(args.area)

	episodeIdInt, err := strconv.Atoi(args.epId)
	if err != nil {
		b.processError(ctx, err)
		return
	}

	if b.getAuthByArea(args.area) {
		// if ok, _ := b.doAuth(ctx, accessKey, area); !ok {
		// 	return
		// }
		subtitleCache, err := b.db.GetTHSubtitleCache(episodeIdInt)
		if err == nil && subtitleCache.JSONData != "" && subtitleCache.UpdatedAt.After(time.Now().Add(-time.Duration(b.config.CacheTHSubtitle)*time.Minute)) {
			b.sugar.Debug("Replay from cache: ", subtitleCache.JSONData)
			setDefaultHeaders(ctx)
			ctx.Write([]byte(subtitleCache.JSONData))
			return
		}
	}

	v := url.Values{}
	v.Set("access_key", args.accessKey)
	v.Set("area", args.area)
	v.Set("s_locale", "zh_SG")
	v.Set("ep_id", args.epId)
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

	url := fmt.Sprintf("https://%s/intl/gateway/v2/app/subtitle?%s", domain, params)
	b.sugar.Debug("New url: ", url)

	data, err := b.doRequestJson(ctx, client, url)
	if err != nil {
		b.processError(ctx, err)
		return
	}

	setDefaultHeaders(ctx)
	ctx.WriteString(data)

	if b.getAuthByArea(args.area) {
		b.db.InsertOrUpdateTHSubtitleCache(episodeIdInt, string(data))
	}
}
