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
	data, err := b.doRequestJson(b.defaultClient, reqParams)
	if err != nil {
		return nil, err
	}
	blackwhitelist := &entity.BlackWhitelist{}
	if err := easyjson.Unmarshal(data, blackwhitelist); err != nil {
		return nil, err
	}

	return blackwhitelist, nil
}

func (b *BiliroamingGo) isAuth(ctx *fasthttp.RequestCtx, accessKey string, clientType ClientType, isForced bool) (*userStatus, error) {
	userStatus := &userStatus{
		uid: -1,
	}

	keyData, err := b.db.GetUserFromKey(accessKey)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		// unknown error
		b.sugar.Error("GetUserFromKey error ", err)
		return userStatus, err
	} else if err == nil && !isForced && keyData.UpdatedAt.After(time.Now().Add(-b.config.Cache.User)) {
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

	body, err := b.getMyInfo(ctx, accessKey, clientType)
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

	err = b.db.InsertOrUpdateKey(accessKey, data.Data.Mid, clientType.String())
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

func (b *BiliroamingGo) getMyInfo(ctx *fasthttp.RequestCtx, accessKey string, clientType ClientType) ([]byte, error) {
	apiURL := "https://app.bilibili.com/x/v2/account/myinfo"

	v := url.Values{}

	v.Set("access_key", accessKey)

	if clientType == ClientTypeBstarA || clientType == ClientTypeUnknown {
		clientType = ClientTypeIphone
	}

	params, err := SignParams(v, clientType)
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
	body, err := b.doRequestJson(b.defaultClient, reqParams)
	if err != nil {
		return nil, err
	}

	b.sugar.Debug("Content: ", string(body))

	return body, nil
}

func (b *BiliroamingGo) doAuth(ctx *fasthttp.RequestCtx, accessKey string, clientType ClientType, area string, isForced bool) (bool, *userStatus) {
	if len(accessKey) == 0 {
		writeErrorJSON(ctx, ERROR_CODE_AUTH_NOT_LOGIN, MSG_ERROR_AUTH_NOT_LOGIN)
		return false, nil
	}

	if len(accessKey) != 32 {
		writeErrorJSON(ctx, ERROR_CODE_AUTH_ACCESS_KEY, MSG_ERROR_AUTH_ACCESS_KEY)
		return false, nil
	}

	key, ok := b.getKey(accessKey)
	if ok {
		if !b.doCheckUidLimiter(ctx, key.uid) {
			writeErrorJSON(ctx, ERROR_CODE_TOO_MANY_REQUESTS, MSG_ERROR_TOO_MANY_REQUESTS)
			return false, nil
		}
		switch b.config.BlockType {
		case BlockTypeEnabled:
			if key.isBlacklist {
				writeErrorJSON(ctx, ERROR_CODE_AUTH_BLACKLIST, fmt.Sprintf(MSG_ERROR_AUTH_BLACKLIST, key.uid, key.banUntil.In(LOCATION_SHANGHAI).Format(TIME_FORMAT)))
				return false, nil
			}
		case BlockTypeWhitelist:
			if !key.isWhitelist {
				writeErrorJSON(ctx, ERROR_CODE_AUTH_WHITELIST, MSG_ERROR_AUTH_WHITELIST)
				return false, nil
			}
		}
		if !key.isLogin {
			writeErrorJSON(ctx, ERROR_CODE_AUTH_NOT_LOGIN, MSG_ERROR_AUTH_NOT_LOGIN)
			return false, nil
		}
		return key.isLogin, &userStatus{
			isLogin:     key.isLogin,
			isVip:       key.isVip,
			isBlacklist: key.isBlacklist,
			isWhitelist: key.isWhitelist,
			uid:         key.uid,
			banUntil:    key.banUntil,
		}
	}

	status, err := b.isAuth(ctx, accessKey, clientType, isForced)
	if err != nil {
		b.setKey(accessKey, status)
		if status.isLogin {
			return true, status
		}
		writeErrorJSON(ctx, ERROR_CODE_AUTH_NOT_LOGIN, MSG_ERROR_AUTH_NOT_LOGIN)
		return false, nil
	}

	b.setKey(accessKey, status)

	switch b.config.BlockType {
	case BlockTypeEnabled:
		if status.isBlacklist {
			writeErrorJSON(ctx, ERROR_CODE_AUTH_BLACKLIST, fmt.Sprintf(MSG_ERROR_AUTH_BLACKLIST, status.uid, status.banUntil.In(LOCATION_SHANGHAI).Format(TIME_FORMAT)))
			return false, nil
		}
	case BlockTypeWhitelist:
		if !status.isWhitelist {
			writeErrorJSON(ctx, ERROR_CODE_AUTH_WHITELIST, MSG_ERROR_AUTH_WHITELIST)
			return false, nil
		}
	}

	if !b.doCheckUidLimiter(ctx, status.uid) {
		writeErrorJSON(ctx, ERROR_CODE_TOO_MANY_REQUESTS, MSG_ERROR_TOO_MANY_REQUESTS)
		return false, nil
	}

	return true, status
}
