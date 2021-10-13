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

func (b *BiliroamingGo) updateHealth(health *entity.Health, isLimited bool) {
	if health == nil {
		return
	}
	health.Data.LastCheck = time.Now()
	if isLimited {
		health.Code = -412
		health.Message = "请求被拦截"
	} else {
		health.Code = 0
		health.Message = "0"
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
		writeErrorJSON(ctx, -400, []byte("area 参数缺失"))
		return
	}

	if argType == "" {
		writeErrorJSON(ctx, -400, []byte("type 参数缺失"))
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
		writeErrorJSON(ctx, -400, []byte("参数错误"))
	}
}
