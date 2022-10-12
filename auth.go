package main

import (
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/JasonKhew96/biliroaming-go-server/entity"
	"github.com/mailru/easyjson"
	"github.com/valyala/fasthttp"
)

// BlockTypeEnum block type
type BlockTypeEnum int

// BlockType
const (
	BlockTypeDisabled BlockTypeEnum = iota
	BlockTypeEnabled
	BlockTypeWhitelist
)

type userStatus struct {
	isLogin     bool
	isVip       bool
	isBlacklist bool
	isWhitelist bool
	uid         int64
	banUntil    time.Time
}

func (b *BiliroamingGo) getAuthByArea(area string) bool {
	switch strings.ToLower(area) {
	case "cn":
		return b.config.Auth.CN
	case "hk":
		return b.config.Auth.HK
	case "tw":
		return b.config.Auth.TW
	case "th":
		return b.config.Auth.TH
	default:
		return true
	}
}

func (b *BiliroamingGo) checkBWlist(ctx *fasthttp.RequestCtx, uid int64) (*entity.BlackWhitelist, error) {
	apiUrl := fmt.Sprintf(b.config.BlacklistApiUrl, uid)
	reqParams := &HttpRequestParams{
		Method:    []byte(fasthttp.MethodGet),
		Url:       []byte(apiUrl),
		UserAgent: []byte(DEFAULT_NAME),
	}
	data, err := b.doRequestJsonWithRetry(b.defaultClient, reqParams, 2)
	if err != nil {
		return nil, err
	}
	blackwhitelist := &entity.BlackWhitelist{}
	if err := easyjson.Unmarshal(data, blackwhitelist); err != nil {
		return nil, err
	}

	return blackwhitelist, nil
}

func (b *BiliroamingGo) isAuth(ctx *fasthttp.RequestCtx, accessKey string) (*userStatus, error) {
	userStatus := &userStatus{
		uid: -1,
	}

	keyData, err := b.db.GetUserFromKey(accessKey)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		// unknown error
		b.sugar.Error("GetUserFromKey error ", err)
		return userStatus, err
	} else if err == nil && keyData.UpdatedAt.After(time.Now().Add(-b.config.Cache.User)) {
		// cached
		b.sugar.Debug("Get vip status from cache: ", keyData)

		userStatus.uid = keyData.UID
		userStatus.isLogin = true

		if b.config.BlockType != BlockTypeDisabled {
			b.sugar.Debugf("isAuth %d %s", keyData.UID, accessKey)
			bwlist, err := b.checkBWlist(ctx, keyData.UID)
			if err != nil {
				return userStatus, err
			}

			if bwlist.Code == 0 {
				userStatus.isBlacklist = bwlist.Data.Status == 1
				userStatus.isWhitelist = bwlist.Data.Status == 2

				userStatus.banUntil = time.Unix(bwlist.Data.BanUntil, 0)
			}
		}

		if keyData.VipDueDate.After(time.Now()) {
			userStatus.isVip = true
		}

		return userStatus, nil
	}

	body, err := b.getMyInfo(ctx, accessKey)
	if err != nil {
		return userStatus, err
	}
	data := &entity.AccInfo{}
	err = easyjson.Unmarshal(body, data)
	if err != nil {
		return userStatus, err
	}
	if data.Code != 0 {
		return userStatus, errors.New(data.Message)
	}
	b.sugar.Debugf("mid: %d, name: %s, due_date: %s", data.Data.Mid, data.Data.Name, time.Unix(data.Data.VIP.DueDate/1000, 0).String())

	userStatus.uid = data.Data.Mid
	userStatus.isLogin = true

	vipDue := time.Unix(data.Data.VIP.DueDate/1000, 0)
	err = b.db.InsertOrUpdateUser(data.Data.Mid, data.Data.Name, vipDue)
	if err != nil {
		return userStatus, err
	}

	err = b.db.InsertOrUpdateKey(accessKey, data.Data.Mid)
	if err != nil {
		return userStatus, err
	}

	if vipDue.After(time.Now()) {
		userStatus.isVip = true
	}

	if b.config.BlockType != BlockTypeDisabled {
		bwlist, err := b.checkBWlist(ctx, data.Data.Mid)
		if err != nil {
			return userStatus, err
		}
		if bwlist.Code == 0 {
			userStatus.isBlacklist = bwlist.Data.Status == 1
			userStatus.isWhitelist = bwlist.Data.Status == 2

			userStatus.banUntil = time.Unix(bwlist.Data.BanUntil, 0)
		}
	}

	return userStatus, nil
}

func (b *BiliroamingGo) getMyInfo(ctx *fasthttp.RequestCtx, accessKey string) ([]byte, error) {
	apiURL := "https://app.bilibili.com/x/v2/account/myinfo"

	v := url.Values{}

	v.Set("access_key", accessKey)

	params, err := SignParams(v, ClientTypeAndroid)
	if err != nil {
		return nil, err
	}
	apiURL += "?" + params

	b.sugar.Debug(apiURL)

	reqParams := &HttpRequestParams{
		Method:    []byte(fasthttp.MethodGet),
		Url:       []byte(apiURL),
		UserAgent: ctx.UserAgent(),
	}
	body, err := b.doRequestJsonWithRetry(b.defaultClient, reqParams, 2)
	if err != nil {
		return nil, err
	}

	b.sugar.Debug("Content: ", string(body))

	return body, nil
}

func (b *BiliroamingGo) doAuth(ctx *fasthttp.RequestCtx, accessKey, area string) (bool, *userStatus) {
	if len(accessKey) == 0 {
		writeErrorJSON(ctx, -401, []byte("帐号未登录"))
		return false, nil
	}

	if len(accessKey) != 32 {
		writeErrorJSON(ctx, -403, []byte("Access Key错误"))
		return false, nil
	}

	key, ok := b.getKey(accessKey)
	if ok {
		if !b.doCheckUidLimiter(ctx, key.uid) {
			writeErrorJSON(ctx, -429, []byte("请求过快"))
			return false, nil
		}
		switch b.config.BlockType {
		case BlockTypeEnabled:
			if key.isBlacklist {
				writeErrorJSON(ctx, -403, []byte(fmt.Sprintf("黑名单\nUID: %d\n解除时间: %s", key.uid, key.banUntil.In(time.FixedZone("Asia/Shanghai", 8)).Format("2006-01-02 15:04:05"))))
				return false, nil
			}
		case BlockTypeWhitelist:
			if !key.isWhitelist {
				writeErrorJSON(ctx, -403, []byte("非白名单"))
				return false, nil
			}
		}
		if !key.isLogin {
			writeErrorJSON(ctx, -401, []byte("账号未登录"))
			return false, nil
		}
		return key.isLogin, &userStatus{
			isVip:       key.isVip,
			isBlacklist: key.isBlacklist,
			isWhitelist: key.isWhitelist,
			banUntil:    key.banUntil,
		}
	}

	status, err := b.isAuth(ctx, accessKey)
	if err != nil {
		b.setKey(accessKey, status)
		if status.isLogin {
			return true, status
		}
		writeErrorJSON(ctx, -401, []byte("账号未登录"))
		return false, nil
	}

	b.setKey(accessKey, status)

	switch b.config.BlockType {
	case BlockTypeEnabled:
		if status.isBlacklist {
			writeErrorJSON(ctx, -403, []byte(fmt.Sprintf("黑名单\nUID: %d\n解除时间: %s", status.uid, status.banUntil.In(time.FixedZone("Asia/Shanghai", 8)).Format("2006-01-02 15:04:05"))))
			return false, nil
		}
	case BlockTypeWhitelist:
		if !status.isWhitelist {
			writeErrorJSON(ctx, -403, []byte("非白名单"))
			return false, nil
		}
	}

	if !b.doCheckUidLimiter(ctx, status.uid) {
		writeErrorJSON(ctx, -429, []byte("请求过快"))
		return false, nil
	}

	return true, status
}
