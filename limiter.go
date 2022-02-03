package main

import (
	"time"

	"github.com/valyala/fasthttp"
	"golang.org/x/time/rate"
)

func (b *BiliroamingGo) getVisitor(uid int64) *rate.Limiter {
	b.vMu.Lock()
	defer b.vMu.Unlock()
	u, exists := b.visitors[uid]
	if !exists {
		rt := rate.Every(time.Second / time.Duration(b.config.Limiter.Limit))
		uLimiter := rate.NewLimiter(rt, b.config.Limiter.Burst)
		b.visitors[uid] = &visitor{
			limiter: uLimiter,
		}
		return uLimiter
	}

	u.lastSeen = time.Now()
	return u.limiter
}

type visitor struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

func (b *BiliroamingGo) doCheckUidLimiter(ctx *fasthttp.RequestCtx, uid int64) bool {
	limiter := b.getVisitor(uid)
	return limiter.Allow()
}
