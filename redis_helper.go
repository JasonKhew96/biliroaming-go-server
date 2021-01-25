package main

import (
	"fmt"
	"time"
)

func (b *biliroamingGo) getMid(accessKey string) (string, error) {
	return b.rdb.Get(b.ctx, "access_key_mid:"+accessKey).Result()
}

func (b *biliroamingGo) setAccessKey(accessKey, mid string) error {
	// 1 week expired
	return b.rdb.Set(b.ctx, "access_key_mid:"+accessKey, mid, time.Duration(b.config.AccessKeyMaxCacheTime)*24*time.Hour).Err()
}

func (b *biliroamingGo) getName(mid string) (string, error) {
	return b.rdb.Get(b.ctx, "mid_name:"+mid).Result()
}

func (b *biliroamingGo) setName(mid, name string) error {
	return b.rdb.Set(b.ctx, "mid_name:"+mid, name, 0).Err()
}

func (b *biliroamingGo) getVIP(mid string) (string, error) {
	return b.rdb.Get(b.ctx, "vip_due:"+mid).Result()
}

func (b *biliroamingGo) setVIP(mid string, due time.Time) error {
	if due.Before(time.Now()) {
		return nil
	}
	diff := due.Sub(time.Now())
	return b.rdb.Set(b.ctx, "vip_due:"+mid, due.Format(time.RFC3339), diff).Err()
}

func (b *biliroamingGo) getBanListKeys() ([]string, error) {
	return b.rdb.Keys(b.ctx, "mid_banned:*").Result()
}

func (b *biliroamingGo) getBan(mid string) (map[string]string, error) {
	return b.rdb.HGetAll(b.ctx, "mid_banned:"+mid).Result()
}

func (b *biliroamingGo) setBan(mid, reason string) error {
	banTime := time.Now().Format(time.RFC3339)
	return b.rdb.HSet(b.ctx, "mid_banned:"+mid, "time", banTime, "reason", reason).Err()
}

func (b *biliroamingGo) getBangumiReqCountKeys() ([]string, error) {
	return b.rdb.Keys(b.ctx, "bangumi_req_count:*").Result()
}

func (b *biliroamingGo) incrBangumiReqCount(cid string) error {
	return b.rdb.Incr(b.ctx, "bangumi_req_count:"+cid).Err()
}

func (b *biliroamingGo) getPlayURLCacheFrom(cid, fnval, qn, isVip string) (string, error) {
	return b.rdb.Get(b.ctx, fmt.Sprintf("play_url_cache:%s:%s:%s:%s", cid, fnval, qn, isVip)).Result()
}

// novip sample
// stream / download
// fnval=16 qn=32 / fnval=0 qn=0
func (b *biliroamingGo) setPlayURLCache(cid, fnval, qn, isVip, resp string) error {
	// maximum 2 hours cache
	return b.rdb.Set(b.ctx, fmt.Sprintf("play_url_cache:%s:%s:%s:%s", cid, fnval, qn, isVip), resp, time.Duration(b.config.PlayurlCacheTime)*time.Minute).Err()
}

func (b *biliroamingGo) getPlayURLWebCacheFrom(cid, fnval, qn, isVip string) (string, error) {
	return b.rdb.Get(b.ctx, fmt.Sprintf("play_url_web_cache:%s:%s:%s:%s", cid, fnval, qn, isVip)).Result()
}

// novip sample
// stream / download
// fnval=16 qn=32 / fnval=0 qn=0
func (b *biliroamingGo) setPlayURLWebCache(cid, fnval, qn, isVip, resp string) error {
	// maximum 2 hours cache
	return b.rdb.Set(b.ctx, fmt.Sprintf("play_url_web_cache:%s:%s:%s:%s", cid, fnval, qn, isVip), resp, time.Duration(b.config.PlayurlCacheTime)*time.Minute).Err()
}
