package main

import (
	"errors"
	"net/url"
	"strings"
	"time"

	"github.com/JasonKhew96/biliroaming-go-server/entity"
	"github.com/mailru/easyjson"
	"github.com/valyala/fasthttp"
)

type userStatus struct {
	isAuth      bool
	isVip       bool
	isBlacklist bool
	isWhitelist bool
}

func (b *BiliroamingGo) getAuthByArea(area string) bool {
	switch strings.ToLower(area) {
	case "cn":
		return b.config.AuthCN
	case "hk":
		return b.config.AuthHK
	case "tw":
		return b.config.AuthTW
	case "th":
		return b.config.AuthTH
	default:
		return true
	}
}

func (b *BiliroamingGo) isAuth(ctx *fasthttp.RequestCtx, accessKey string) (*userStatus, error) {
	// isAuth, isVIP, error
	keyData, err := b.db.GetKey(accessKey)
	if err == nil {
		b.sugar.Debug("Get vip status from cache: ", keyData)
		userData, err := b.db.GetUser(keyData.UID)
		if err != nil {
			return &userStatus{
				isAuth:      false,
				isVip:       false,
				isBlacklist: false,
				isWhitelist: false,
			}, err
		}
		if userData.VIPDueDate.After(time.Now()) {
			return &userStatus{
				isAuth:      true,
				isVip:       true,
				isBlacklist: false,
				isWhitelist: false,
			}, nil
		}
		return &userStatus{
			isAuth:      true,
			isVip:       false,
			isBlacklist: false,
			isWhitelist: false,
		}, nil
	}

	body, err := b.getMyInfo(ctx, accessKey)
	if err != nil {
		return &userStatus{
			isAuth:      false,
			isVip:       false,
			isBlacklist: false,
			isWhitelist: false,
		}, err
	}
	data := &entity.AccInfo{}
	err = easyjson.Unmarshal(body, data)
	if err != nil {
		return &userStatus{
			isAuth:      false,
			isVip:       false,
			isBlacklist: false,
			isWhitelist: false,
		}, err
	}
	if data.Code != 0 {
		return &userStatus{
			isAuth:      false,
			isVip:       false,
			isBlacklist: false,
			isWhitelist: false,
		}, errors.New(data.Message)
	}
	b.sugar.Debugf("mid: %d, name: %s, due_date: %s", data.Data.Mid, data.Data.Name, time.Unix(data.Data.VIP.DueDate/1000, 0).String())

	_, err = b.db.InsertOrUpdateKey(accessKey, data.Data.Mid)
	if err != nil {
		return &userStatus{
			isAuth:      false,
			isVip:       false,
			isBlacklist: false,
			isWhitelist: false,
		}, err
	}
	_, err = b.db.InsertOrUpdateUser(data.Data.Mid, data.Data.Name, time.Unix(data.Data.VIP.DueDate/1000, 0))
	if err != nil {
		return &userStatus{
			isAuth:      false,
			isVip:       false,
			isBlacklist: false,
			isWhitelist: false,
		}, err
	}

	return &userStatus{
		isAuth:      true,
		isVip:       false,
		isBlacklist: false,
		isWhitelist: false,
	}, nil
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

	body, err := b.doRequestJson(ctx, b.defaultClient, apiURL)
	if err != nil {
		return nil, err
	}

	b.sugar.Debug("Content: ", string(body))

	return body, nil
}
