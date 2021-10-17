package main

import (
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
	BlockTypeWhitelist
	BlockTypeBlacklist
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

func (b *BiliroamingGo) isBlacklist(ctx *fasthttp.RequestCtx, accessKey string) (bool, error) {
	apiUrl := fmt.Sprintf("https://black.qimo.ink/?access_key=%s", accessKey)
	data, err := b.doRequest(ctx, b.defaultClient, apiUrl)
	if err != nil {
		return false, err
	}
	if string(data) == "ban" {
		return true, nil
	}
	return false, nil
}

func (b *BiliroamingGo) isAuth(ctx *fasthttp.RequestCtx, accessKey string) (*userStatus, error) {
	keyData, err := b.db.GetKey(accessKey)
	if err == nil {
		b.sugar.Debug("Get vip status from cache: ", keyData)
		userData, err := b.db.GetUser(keyData.UID)
		if err != nil {
			return nil, err
		}
		isBlacklisted := false
		if b.config.BlockType == BlockTypeBlacklist {
			b.sugar.Debugf("isBlacklist %d %s", keyData.UID, accessKey)
			isBlacklisted, err = b.isBlacklist(ctx, accessKey)
			if err != nil {
				return nil, err
			}
		}
		if userData.VIPDueDate.After(time.Now()) {
			return &userStatus{
				isAuth:      true,
				isVip:       true,
				isBlacklist: isBlacklisted,
				isWhitelist: false,
			}, nil
		}
		return &userStatus{
			isAuth:      true,
			isVip:       false,
			isBlacklist: isBlacklisted,
			isWhitelist: false,
		}, nil
	}

	body, err := b.getMyInfo(ctx, accessKey)
	if err != nil {
		return nil, err
	}
	data := &entity.AccInfo{}
	err = easyjson.Unmarshal([]byte(body), data)
	if err != nil {
		return nil, err
	}
	if data.Code != 0 {
		return nil, errors.New(data.Message)
	}
	b.sugar.Debugf("mid: %d, name: %s, due_date: %s", data.Data.Mid, data.Data.Name, time.Unix(data.Data.VIP.DueDate/1000, 0).String())

	_, err = b.db.InsertOrUpdateKey(accessKey, data.Data.Mid)
	if err != nil {
		return nil, err
	}
	vipDue := time.Unix(data.Data.VIP.DueDate/1000, 0)
	_, err = b.db.InsertOrUpdateUser(data.Data.Mid, data.Data.Name, vipDue)
	if err != nil {
		return nil, err
	}

	isVip := vipDue.After(time.Now())

	isBlacklisted := false
	if b.config.BlockType == BlockTypeBlacklist {
		isBlacklisted, err = b.isBlacklist(ctx, accessKey)
		if err != nil {
			return nil, err
		}
	}

	return &userStatus{
		isAuth:      true,
		isVip:       isVip,
		isBlacklist: isBlacklisted,
		isWhitelist: false,
	}, nil
}

func (b *BiliroamingGo) getMyInfo(ctx *fasthttp.RequestCtx, accessKey string) (string, error) {
	apiURL := "https://app.bilibili.com/x/v2/account/myinfo"

	v := url.Values{}

	v.Set("access_key", accessKey)

	params, err := SignParams(v, ClientTypeAndroid)
	if err != nil {
		return "", err
	}
	apiURL += "?" + params

	b.sugar.Debug(apiURL)

	body, err := b.doRequestJson(ctx, b.defaultClient, apiURL)
	if err != nil {
		return "", err
	}

	b.sugar.Debug("Content: ", body)

	return body, nil
}
