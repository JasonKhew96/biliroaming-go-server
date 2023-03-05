package main

import (
	"time"

	"github.com/JasonKhew96/biliroaming-go-server/database"
	"github.com/JasonKhew96/biliroaming-go-server/entity"
	"github.com/valyala/fasthttp"
)

// newHealth assign new health json
func newHealth() *entity.Health {
	return &entity.Health{
		Code:    0,
		Message: "0",
		Data: entity.HealthData{
			LastCheck: time.Now(),
		},
	}
}

func (b *BiliroamingGo) updateHealth(health *entity.Health, code int, message string) {
	if health == nil {
		return
	}
	health.Data.LastCheck = time.Now()
	health.Message = message
	if code == 0 {
		health.Data.Counter = 0
		health.Code = 0
		return
	}
	health.Data.Counter++
	if health.Data.Counter > 3 {
		health.Code = code
	}
}

func (b *BiliroamingGo) getPlayUrlHealth(area string) *entity.Health {
	areaCode := getAreaCode(area)
	switch areaCode {
	case database.AreaCN:
		return b.HealthPlayUrlCN
	case database.AreaHK:
		return b.HealthPlayUrlHK
	case database.AreaTW:
		return b.HealthPlayUrlTW
	case database.AreaTH:
		return b.HealthPlayUrlTH
	default:
		return nil
	}
}

func (b *BiliroamingGo) getSearchHealth(area string) *entity.Health {
	areaCode := getAreaCode(area)
	switch areaCode {
	case database.AreaCN:
		return b.HealthSearchCN
	case database.AreaHK:
		return b.HealthSearchHK
	case database.AreaTW:
		return b.HealthSearchTW
	case database.AreaTH:
		return b.HealthSearchTH
	default:
		return nil
	}
}

func (b *BiliroamingGo) handleApiHealth(ctx *fasthttp.RequestCtx) {
	queryArgs := ctx.URI().QueryArgs()
	argArea := string(queryArgs.PeekBytes([]byte("area")))
	argType := string(queryArgs.PeekBytes([]byte("type")))

	if argArea == "" {
		writeErrorJSON(ctx, ERROR_CODE_MISSING_AREA, MSG_ERROR_MISSING_AREA)
		return
	}

	if argType == "" {
		writeErrorJSON(ctx, ERROR_CODE_MISSING_TYPE, MSG_ERROR_MISSING_TYPE)
		return
	}

	switch argType {
	case "playurl":
		writeHealthJSON(ctx, b.getPlayUrlHealth(argArea))
	case "search":
		writeHealthJSON(ctx, b.getSearchHealth(argArea))
	case "season":
		writeHealthJSON(ctx, b.HealthSeasonTH)
	default:
		writeErrorJSON(ctx, ERROR_CODE_PARAMETERS, MSG_ERROR_PARAMETERS)
	}
}
