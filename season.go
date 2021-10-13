package main

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/JasonKhew96/biliroaming-go-server/entity"
	"github.com/mailru/easyjson"
	"github.com/pkg/errors"
	"github.com/valyala/fasthttp"
	"golang.org/x/net/idna"
)

func (b *BiliroamingGo) addCustomSubSeason(ctx *fasthttp.RequestCtx, seasonId string, oldSeason []byte) ([]byte, error) {
	b.sugar.Debugf("Getting custom subtitle from season id %s", seasonId)
	seasonJson := &entity.SeasonResponse{}
	err := easyjson.Unmarshal(oldSeason, seasonJson)
	if err != nil {
		return nil, errors.Wrap(err, "season response unmarshal")
	}

	requestUrl := fmt.Sprintf(b.config.CustomSubAPI, seasonId)
	customSubData, err := b.doRequestJson(ctx, b.defaultClient, requestUrl)
	if err != nil {
		return nil, errors.Wrap(err, "custom subtitle api")
	}

	customSubJson := &entity.CustomSubResponse{}
	err = easyjson.Unmarshal(customSubData, customSubJson)
	if err != nil {
		return nil, errors.Wrap(err, "custom subtitle response unmarshal")
	}

	if customSubJson.Code != 0 {
		return oldSeason, nil
	}

	if len(seasonJson.Result.Modules) <= 0 || len(seasonJson.Result.Modules[0].Data.Episodes) <= 0 {
		return oldSeason, nil
	}

	for i, ep := range seasonJson.Result.Modules[0].Data.Episodes {
		subtitles := ep.Subtitles
		for j, customSubEp := range customSubJson.Data {
			if i == customSubEp.Ep {
				newUrl := customSubEp.URL
				if !strings.HasPrefix(newUrl, "https://") {
					newUrl = fmt.Sprintf("https://%s", customSubEp.URL)
				}
				title := fmt.Sprintf("%s[%s][非官方]", customSubEp.Lang, b.config.CustomSubTeam)
				subtitles = append([]entity.Subtitles{
					{
						ID:        int64(j),
						Key:       customSubEp.Key,
						Title:     title,
						URL:       newUrl,
						IsMachine: false,
					},
				}, subtitles...)
			}
		}
		seasonJson.Result.Modules[0].Data.Episodes[i].Subtitles = subtitles
	}

	newSeason, err := easyjson.Marshal(seasonJson)
	if err != nil {
		return nil, errors.Wrap(err, "new season response marshal")
	}

	b.sugar.Debugf("New season response: %s", string(newSeason))

	return newSeason, nil
}

func (b *BiliroamingGo) handleBstarAndroidSeason(ctx *fasthttp.RequestCtx) {
	queryArgs := ctx.URI().QueryArgs()
	args := b.processArgs(queryArgs)

	if args.area == "" {
		args.area = "th"
		// writeErrorJSON(ctx, -688, []byte("地理区域限制"))
		// return
	}

	client := b.getClientByArea(args.area)

	if args.seasonId == "" {
		writeErrorJSON(ctx, -400, []byte("请求错误"))
		return
	}

	seasonIdInt, err := strconv.Atoi(args.seasonId)
	if err != nil {
		b.processError(ctx, err)
		return
	}

	if b.getAuthByArea(args.area) {
		if ok, _ := b.doAuth(ctx, args.accessKey, args.area); !ok {
			return
		}
		seasonCache, err := b.db.GetTHSeasonCache(seasonIdInt)
		if err == nil && seasonCache.JSONData != "" && seasonCache.UpdatedAt.After(time.Now().Add(-time.Duration(b.config.CacheTHSeason)*time.Minute)) {
			b.sugar.Debug("Replay from cache: ", seasonCache.JSONData)
			setDefaultHeaders(ctx)
			ctx.Write([]byte(seasonCache.JSONData))
			return
		}
	}

	v := url.Values{}
	v.Set("access_key", args.accessKey)
	v.Set("area", args.area)
	v.Set("build", "1080003")
	v.Set("s_locale", "zh_SG")
	v.Set("season_id", args.seasonId)
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

	url := fmt.Sprintf("https://%s/intl/gateway/v2/ogv/view/app/season?%s", domain, params)
	b.sugar.Debug("New url: ", url)

	data, err := b.doRequestJson(ctx, client, url)
	if err != nil {
		b.processError(ctx, err)
		return
	}

	if isLimited, err := isResponseLimited(data); err != nil {
		b.sugar.Error(err)
	} else {
		b.updateHealth(b.HealthSeasonTH, isLimited)
	}

	if b.config.CustomSubAPI != "" {
		data, err = b.addCustomSubSeason(ctx, args.seasonId, data)
		if err != nil {
			b.processError(ctx, err)
			return
		}
	}

	setDefaultHeaders(ctx)
	ctx.Write(data)

	if b.getAuthByArea(args.area) {
		b.db.InsertOrUpdateTHSeasonCache(seasonIdInt, string(data))
	}
}
