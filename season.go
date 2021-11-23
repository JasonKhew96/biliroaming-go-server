package main

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/JasonKhew96/biliroaming-go-server/entity"
	"github.com/JasonKhew96/biliroaming-go-server/entity/bstar"
	"github.com/mailru/easyjson"
	"github.com/pkg/errors"
	"github.com/valyala/fasthttp"
	"golang.org/x/net/idna"
)

func (b *BiliroamingGo) insertSeasonCache(data string, isVIP bool) error {
	var seasonJson *bstar.SeasonResult
	err := easyjson.Unmarshal([]byte(data), seasonJson)
	if err != nil {
		return errors.Wrap(err, "season response unmarshal")
	}

	b.db.InsertOrUpdateTHSeasonCache(seasonJson.Result.SeasonID, isVIP, data)

	if len(seasonJson.Result.Modules) <= 0 {
		return nil
	}

	for _, ep := range seasonJson.Result.Modules[0].Data.Episodes {
		b.db.InsertOrUpdateTHSeasonEpisodeCache(ep.ID, seasonJson.Result.SeasonID, isVIP)
	}

	return nil
}

func (b *BiliroamingGo) addCustomSubSeason(ctx *fasthttp.RequestCtx, seasonId int, oldSeason string) (string, error) {
	b.sugar.Debugf("Getting custom subtitle from season id %s", seasonId)
	seasonJson := &bstar.SeasonResult{}
	err := easyjson.Unmarshal([]byte(oldSeason), seasonJson)
	if err != nil {
		return "", errors.Wrap(err, "season response unmarshal")
	}

	requestUrl := fmt.Sprintf(b.config.CustomSubtitle.ApiUrl, seasonId)
	customSubData, err := b.doRequestJson(ctx, b.defaultClient, requestUrl, []byte(http.MethodGet))
	if err != nil {
		return "", errors.Wrap(err, "custom subtitle api")
	}

	customSubJson := &entity.CustomSubResponse{}
	err = easyjson.Unmarshal([]byte(customSubData), customSubJson)
	if err != nil {
		return "", errors.Wrap(err, "custom subtitle response unmarshal")
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
				title := fmt.Sprintf("%s[%s][非官方]", customSubEp.Lang, b.config.CustomSubtitle.TeamName)
				subtitles = append([]bstar.Subtitles{
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

	newSeasonBytes, err := easyjson.Marshal(seasonJson)
	if err != nil {
		return "", errors.Wrap(err, "new season response marshal")
	}

	newSeason := string(newSeasonBytes)

	b.sugar.Debugf("New season response: %s", newSeason)

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

	if args.seasonId == 0 && args.epId == 0 {
		writeErrorJSON(ctx, -400, []byte("请求错误"))
		return
	}

	if b.getAuthByArea(args.area) {
		if ok, _ := b.doAuth(ctx, args.accessKey, args.area); !ok {
			return
		}
		if args.seasonId != 0 {
			seasonCache, err := b.db.GetTHSeasonCache(args.seasonId, false)
			if err == nil && seasonCache.JSONData != "" && seasonCache.UpdatedAt.After(time.Now().Add(-b.config.Cache.THSeason)) {
				b.sugar.Debug("Replay from cache: ", seasonCache.JSONData)
				setDefaultHeaders(ctx)
				ctx.Write([]byte(seasonCache.JSONData))
				return
			}
		}
		if args.epId != 0 {
			seasonCache, err := b.db.GetTHSeasonEpisodeCache(args.epId, false)
			if err == nil && seasonCache.JSONData != "" && seasonCache.UpdatedAt.After(time.Now().Add(-b.config.Cache.THSeason)) {
				b.sugar.Debug("Replay from cache: ", seasonCache.JSONData)
				setDefaultHeaders(ctx)
				ctx.Write([]byte(seasonCache.JSONData))
				return
			}
		}
	}

	v := url.Values{}
	v.Set("access_key", args.accessKey)
	v.Set("area", args.area)
	v.Set("build", "1080003")
	v.Set("s_locale", "zh_SG")
	if args.seasonId != 0 {
		v.Set("season_id", strconv.Itoa(args.seasonId))
	}
	if args.epId != 0 {
		v.Set("ep_id", strconv.Itoa(args.epId))
	}
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

	data, err := b.doRequestJson(ctx, client, url, []byte(http.MethodGet))
	if err != nil {
		b.processError(ctx, err)
		b.updateHealth(b.HealthSeasonTH, -500, "服务器错误")
		return
	}

	if isLimited, err := isResponseLimited(data); err != nil {
		b.sugar.Error(err)
	} else if isLimited {
		b.updateHealth(b.HealthSeasonTH, -412, "请求被拦截")
	} else {
		b.updateHealth(b.HealthSeasonTH, 0, "0")
	}

	if b.config.CustomSubtitle.ApiUrl != "" {
		newData, err := b.addCustomSubSeason(ctx, args.seasonId, data)
		if err != nil {
			b.processError(ctx, err)
			return
		}
		if isValidJson(newData) {
			data = newData
		} else {
			b.sugar.Error("addCustomSubSeason: ", newData)
		}
	}

	setDefaultHeaders(ctx)
	ctx.WriteString(data)

	if b.getAuthByArea(args.area) {
		b.insertSeasonCache(data, false)
	}
}
