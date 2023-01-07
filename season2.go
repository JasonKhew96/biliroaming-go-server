package main

import (
	"database/sql"
	"fmt"
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

func (b *BiliroamingGo) insertSeason2Cache(data []byte, isVIP bool) error {
	season2Result := &bstar.Season2Result{}
	err := easyjson.Unmarshal(data, season2Result)
	if err != nil {
		return errors.Wrap(err, "season response unmarshal")
	}

	if err := b.db.InsertOrUpdateTHSeason2Cache(season2Result.Data.SeasonID, isVIP, data); err != nil {
		b.sugar.Error(err)
	}

	if len(season2Result.Data.Sections.Section) <= 0 {
		return nil
	}

	for _, mdl := range season2Result.Data.Sections.Section {
		for _, ep := range mdl.EpDetails {
			if err := b.db.InsertOrUpdateTHSeason2EpisodeCache(ep.EpisodeID, season2Result.Data.SeasonID); err != nil {
				b.sugar.Error(err)
			}
			if len(ep.Subtitles) > 0 {
				episode := bstar.EpisodeResult{
					Code:    0,
					Message: "0",
					TTL:     1,
					Data: bstar.EpisodeResultData{
						SubtitleSuggestKey: ep.Subtitles[0].Key,
						Jump:               ep.Jump,
						Subtitles:          ep.Subtitles,
					},
				}
				newSubtitle, err := easyjson.Marshal(&episode)
				if err != nil {
					b.sugar.Error(err)
					continue
				}
				if err := b.db.InsertOrUpdateTHEpisodeCache(ep.EpisodeID, newSubtitle); err != nil {
					b.sugar.Error(err)
				}
			}
		}
	}

	return nil
}

func (b *BiliroamingGo) addCustomSubSeason2(ctx *fasthttp.RequestCtx, seasonResult []byte) ([]byte, error) {
	b.sugar.Debugf("Getting custom subtitle")
	season2Json := &bstar.Season2Result{}
	err := easyjson.Unmarshal(seasonResult, season2Json)
	if err != nil {
		return nil, errors.Wrap(err, "season response unmarshal")
	}

	if len(season2Json.Data.Sections.Section) <= 0 || len(season2Json.Data.Sections.Section[0].EpDetails) <= 0 {
		return nil, errors.Wrap(err, "custom subtitle api cannnot append to weird season api response")
	}

	seasonId := season2Json.Data.SeasonID
	b.sugar.Debugf("Getting custom subtitle from season id %d", seasonId)

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

	episodeIndex := 0
	for sectionIndex, sec := range season2Json.Data.Sections.Section {
		for epIndex, ep := range sec.EpDetails {
			subtitles := ep.Subtitles
			for j, customSubEp := range customSubJson.Data {
				if episodeIndex == customSubEp.Ep {
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
			episodeIndex++
			season2Json.Data.Sections.Section[sectionIndex].EpDetails[epIndex].Subtitles = subtitles
		}
	}

	newSeason2Bytes, err := easyjson.Marshal(season2Json)
	if err != nil {
		return nil, errors.Wrap(err, "new season response marshal")
	}

	b.sugar.Debugf("New season response: %s", string(newSeason2Bytes))

	return newSeason2Bytes, nil
}

func (b *BiliroamingGo) handleBstarAndroidSeason2(ctx *fasthttp.RequestCtx) {
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

	if b.getAuthByArea(args.area) {
		if ok, _ := b.doAuth(ctx, args.accessKey, getClientPlatform(ctx, args.appkey), args.area, false); !ok {
			return
		}
		if args.seasonId != 0 {
			season2Cache, err := b.db.GetTHSeason2Cache(args.seasonId, false)
			if err == nil && len(season2Cache.Data) > 0 && season2Cache.UpdatedAt.After(time.Now().Add(-b.config.Cache.THSeason)) {
				b.sugar.Debug("Replay from cache: ", season2Cache.Data.String())
				setDefaultHeaders(ctx)
				ctx.Write(season2Cache.Data)
				return
			} else if err != nil && !errors.Is(err, sql.ErrNoRows) {
				b.processError(ctx, err)
				b.updateHealth(b.HealthSeasonTH, ERROR_CODE_INTERNAL_SERVER, MSG_ERROR_INTERNAL_SERVER)
				return
			}
		}
		if args.epId != 0 {
			season2Cache, err := b.db.GetTHSeason2EpisodeCache(args.epId, false)
			if err == nil && len(season2Cache.Data) > 0 && season2Cache.UpdatedAt.After(time.Now().Add(-b.config.Cache.THSeason)) {
				b.sugar.Debug("Replay from cache: ", season2Cache.Data)
				setDefaultHeaders(ctx)
				ctx.Write(season2Cache.Data)
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
	v.Set("platform", "android")
	v.Set("s_locale", "zh_SG")
	if args.seasonId != 0 {
		v.Set("season_id", strconv.FormatInt(args.seasonId, 10))
	}
	if args.epId != 0 {
		v.Set("ep_id", strconv.FormatInt(args.epId, 10))
	}

	params, err := SignParams(v, ClientTypeBstarA)
	if err != nil {
		b.processError(ctx, err)
		return
	}

	reverseProxy := b.getReverseProxyByArea(args.area)
	if reverseProxy == "" {
		reverseProxy = "app.biliintl.com"
	}
	domain, err := idna.New().ToASCII(reverseProxy)
	if err != nil {
		b.processError(ctx, err)
		return
	}

	url := fmt.Sprintf("https://%s/intl/gateway/v2/ogv/view/app/season2?%s", domain, params)
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
		newData, err := b.addCustomSubSeason2(ctx, data)
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
		b.insertSeason2Cache(data, false)
	}
}
