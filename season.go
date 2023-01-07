package main

import (
	"database/sql"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/JasonKhew96/biliroaming-go-server/database"
	"github.com/JasonKhew96/biliroaming-go-server/entity"
	"github.com/JasonKhew96/biliroaming-go-server/entity/bstar"
	"github.com/mailru/easyjson"
	"github.com/pkg/errors"
	"github.com/valyala/fasthttp"
	"golang.org/x/net/idna"
)

func (b *BiliroamingGo) checkSeasonAreaCache(seasonId int64, area database.Area) bool {
	if cache, err := b.db.GetSeasonAreaCache(seasonId); err == nil {
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

func (b *BiliroamingGo) updateSeasonCache(data []byte, seasonId int64, area database.Area) error {
	if available, err := isAvailableResponse(data); err != nil {
		return err
	} else if err := b.db.InsertOrUpdateSeasonAreaCache(seasonId, area, available); err != nil {
		return err
	}
	return nil
}

func (b *BiliroamingGo) insertSeasonCache(data []byte, isVIP bool) error {
	seasonResult := &bstar.SeasonResult{}
	err := easyjson.Unmarshal(data, seasonResult)
	if err != nil {
		return errors.Wrap(err, "season response unmarshal")
	}

	if err := b.db.InsertOrUpdateTHSeasonCache(seasonResult.Result.SeasonID, isVIP, data); err != nil {
		b.sugar.Error(err)
	}

	if len(seasonResult.Result.Modules) <= 0 {
		return nil
	}

	if err := b.db.InsertOrUpdateSeasonAreaCache(seasonResult.Result.SeasonID, database.AreaTH, true); err != nil {
		b.sugar.Error(err)
	}

	for _, mdl := range seasonResult.Result.Modules {
		for _, ep := range mdl.Data.Episodes {
			if err := b.db.InsertOrUpdateEpisodeAreaCache(ep.ID, database.AreaTH, true); err != nil {
				b.sugar.Error()
			}
			if err := b.db.InsertOrUpdateTHSeasonEpisodeCache(ep.ID, seasonResult.Result.SeasonID); err != nil {
				b.sugar.Error(err)
			}
			if len(ep.Subtitles) > 0 {
				subtitles := bstar.SubtitleResult{
					Code:    0,
					Message: "0",
					TTL:     1,
					Data: bstar.SubtitleResultData{
						SuggestKey: ep.Subtitles[0].Key,
						Subtitles:  ep.Subtitles,
					},
				}
				newSubtitle, err := easyjson.Marshal(&subtitles)
				if err != nil {
					b.sugar.Error(err)
					continue
				}
				if err := b.db.InsertOrUpdateTHSubtitleCache(ep.ID, newSubtitle); err != nil {
					b.sugar.Error(err)
				}
			}
		}
	}

	return nil
}

func (b *BiliroamingGo) replaceSeason(ctx *fasthttp.RequestCtx, seasonResult []byte) ([]byte, error) {
	b.sugar.Debugf("Replace season")
	seasonJson := &bstar.SeasonResult{}
	err := easyjson.Unmarshal(seasonResult, seasonJson)
	if err != nil {
		return nil, errors.Wrap(err, "season response unmarshal")
	}

	if len(seasonJson.Result.Modules) <= 0 || len(seasonJson.Result.Modules[0].Data.Episodes) <= 0 {
		return nil, errors.Wrap(err, "custom subtitle api cannnot append to weird season api response")
	}

	seasonId := seasonJson.Result.SeasonID
	b.sugar.Debugf("Replace season from season id %d", seasonId)

	requestUrl := fmt.Sprintf(b.config.CustomSubtitle.ApiUrl, seasonId)
	reqParams := &HttpRequestParams{
		Method:    []byte(fasthttp.MethodGet),
		Url:       []byte(requestUrl),
		UserAgent: []byte(DEFAULT_NAME),
	}
	customSubData, err := b.doRequestJson(b.defaultClient, reqParams)
	if err != nil {
		return nil, errors.Wrap(err, "custom subtitle api")
	}

	customSubJson := &entity.CustomSubResponse{}
	err = easyjson.Unmarshal(customSubData, customSubJson)
	if err != nil {
		return nil, errors.Wrap(err, "custom subtitle response unmarshal")
	}

	if customSubJson.Code != 0 {
		return nil, errors.Wrap(err, fmt.Sprintf("custom subtitle api return code %d", customSubJson.Code))
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

	for i, m := range seasonJson.Result.Modules {
		for j := range m.Data.Episodes {
			if b.config.ThRedirect.Aid != 0 {
				seasonJson.Result.Modules[i].Data.Episodes[j].Aid = b.config.ThRedirect.Aid
			}
		}
	}

	newSeasonBytes, err := easyjson.Marshal(seasonJson)
	if err != nil {
		return nil, errors.Wrap(err, "new season response marshal")
	}

	b.sugar.Debugf("New season response: %s", string(newSeasonBytes))

	return newSeasonBytes, nil
}

func (b *BiliroamingGo) handleBstarAndroidSeason(ctx *fasthttp.RequestCtx) {
	if !b.checkRoamingVer(ctx) {
		return
	}

	queryArgs := ctx.URI().QueryArgs()
	args := b.processArgs(queryArgs)

	args.area = "th"

	client := b.getClientByArea(args.area)

	if args.seasonId == 0 && args.epId == 0 {
		writeErrorJSON(ctx, ERROR_CODE_MISSING_SS_OR_EP, MSG_ERROR_MISSING_SS_OR_EP)
		return
	}

	if args.seasonId != 0 {
		if ok := b.checkSeasonAreaCache(args.seasonId, getAreaCode(args.area)); !ok {
			writeErrorJSON(ctx, ERROR_CODE_GEO_RESTRICED, MSG_ERROR_GEO_RESTRICTED)
			return
		}
	}

	if b.getAuthByArea(args.area) {
		if ok, _ := b.doAuth(ctx, args.accessKey, getClientPlatform(ctx, args.appkey), args.area, false); !ok {
			return
		}
		if args.seasonId != 0 {
			seasonCache, err := b.db.GetTHSeasonCache(args.seasonId, false)
			if err == nil && len(seasonCache.Data) > 0 && seasonCache.UpdatedAt.After(time.Now().Add(-b.config.Cache.THSeason)) {
				b.sugar.Debug("Replay from cache: ", seasonCache.Data.String())
				setDefaultHeaders(ctx)
				ctx.Write(seasonCache.Data)
				return
			} else if err != nil && !errors.Is(err, sql.ErrNoRows) {
				b.processError(ctx, err)
				b.updateHealth(b.HealthSeasonTH, ERROR_CODE_INTERNAL_SERVER, MSG_ERROR_INTERNAL_SERVER)
				return
			}
		}
		if args.epId != 0 {
			seasonCache, err := b.db.GetTHSeasonEpisodeCache(args.epId, false)
			if err == nil && len(seasonCache.Data) > 0 && seasonCache.UpdatedAt.After(time.Now().Add(-b.config.Cache.THSeason)) {
				b.sugar.Debug("Replay from cache: ", seasonCache.Data)
				setDefaultHeaders(ctx)
				ctx.Write(seasonCache.Data)
				return
			} else if err != nil && !errors.Is(err, sql.ErrNoRows) {
				b.processError(ctx, err)
				b.updateHealth(b.HealthSeasonTH, ERROR_CODE_INTERNAL_SERVER, MSG_ERROR_INTERNAL_SERVER)
				return
			}
		}
	}

	v := url.Values{}
	v.Set("access_key", args.accessKey)
	v.Set("build", "1080003")
	v.Set("s_locale", "zh_SG")
	if args.seasonId != 0 {
		v.Set("season_id", strconv.FormatInt(args.seasonId, 10))
	}
	if args.epId != 0 {
		v.Set("ep_id", strconv.FormatInt(args.epId, 10))
	}
	v.Set("mobi_app", "bstar_a")

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

	url := fmt.Sprintf("https://%s/intl/gateway/v2/ogv/view/app/season?%s", domain, params)
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
			b.updateHealth(b.HealthSeasonTH, ERROR_CODE_INTERNAL_SERVER, MSG_ERROR_INTERNAL_SERVER)
			return
		}
	}

	if isLimited, err := isResponseLimited(data); err != nil {
		b.sugar.Error(err)
	} else if isLimited {
		b.updateHealth(b.HealthSeasonTH, ERROR_CODE_TOO_MANY_REQUESTS, MSG_ERROR_TOO_MANY_REQUESTS)
	} else {
		b.updateHealth(b.HealthSeasonTH, 0, "0")
	}

	if b.config.CustomSubtitle.ApiUrl != "" {
		newData, err := b.replaceSeason(ctx, data)
		if err != nil {
			b.sugar.Error(err)
		}
		if len(newData) > 0 {
			data = newData
		}
	}

	setDefaultHeaders(ctx)
	ctx.Write(data)

	if b.getAuthByArea(args.area) {
		b.insertSeasonCache(data, false)
	}
}
