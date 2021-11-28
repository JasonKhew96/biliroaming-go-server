package main

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
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
)

type userStatus struct {
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

func (b *BiliroamingGo) checkBWlist(ctx *fasthttp.RequestCtx, uid int64) (*entity.BlackWhitelist, error) {
	apiUrl := fmt.Sprintf("https://black.qimo.ink/status.php?uid=%d", uid)
	data, err := b.doRequestJson(ctx, b.defaultClient, apiUrl, []byte(http.MethodGet))
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
		isVip:       false,
		isBlacklist: false,
		isWhitelist: false,
	}

	keyData, err := b.db.GetUserFromKey(accessKey)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		// unknown error
		b.sugar.Error("GetUserFromKey error ", err)
		return nil, err
	} else if err == nil {
		// cached
		b.sugar.Debug("Get vip status from cache: ", keyData)

		if b.config.BlockType == BlockTypeEnabled {
			b.sugar.Debugf("isBlacklist %d %s", keyData.UID, accessKey)
			bwlist, err := b.checkBWlist(ctx, keyData.UID)
			if err != nil {
				return nil, err
			}

			if bwlist.Code == 0 {
				userStatus.isBlacklist = bwlist.Data.IsBlacklist
				userStatus.isWhitelist = bwlist.Data.IsWhitelist
			}
		}

		if keyData.VipDueDate.After(time.Now()) {
			userStatus.isVip = true
		}

		return userStatus, nil
	}

	body, err := b.getMyInfo(ctx, accessKey)
	if err != nil {
		return nil, err
	}
	data := &entity.AccInfo{}
	err = easyjson.Unmarshal(body, data)
	if err != nil {
		return nil, err
	}
	if data.Code != 0 {
		return nil, errors.New(data.Message)
	}
	b.sugar.Debugf("mid: %d, name: %s, due_date: %s", data.Data.Mid, data.Data.Name, time.Unix(data.Data.VIP.DueDate/1000, 0).String())

	vipDue := time.Unix(data.Data.VIP.DueDate/1000, 0)
	err = b.db.InsertOrUpdateUser(data.Data.Mid, data.Data.Name, vipDue)
	if err != nil {
		return nil, err
	}

	err = b.db.InsertOrUpdateKey(accessKey, data.Data.Mid)
	if err != nil {
		return nil, err
	}

	if vipDue.After(time.Now()) {
		userStatus.isVip = true
	}

	if b.config.BlockType == BlockTypeEnabled {
		bwlist, err := b.checkBWlist(ctx, data.Data.Mid)
		if err != nil {
			return nil, err
		}
		if bwlist.Code == 0 {
			userStatus.isBlacklist = bwlist.Data.IsBlacklist
			userStatus.isWhitelist = bwlist.Data.IsWhitelist
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

	body, err := b.doRequestJson(ctx, b.defaultClient, apiURL, []byte(http.MethodGet))
	if err != nil {
		return nil, err
	}

	b.sugar.Debug("Content: ", string(body))

	return body, nil
}

func (b *BiliroamingGo) doAuth(ctx *fasthttp.RequestCtx, accessKey, area string) (bool, *userStatus) {
	if len(accessKey) != 32 {
		writeErrorJSON(ctx, -2, []byte("Access Key错误"))
		return false, nil
	}

	key, ok := b.getKey(accessKey)
	if ok {
		if key.isBlacklist {
			writeErrorJSON(ctx, -101, []byte("黑名单"))
			return false, nil
		}
		if !key.isLogin {
			writeErrorJSON(ctx, -101, []byte("账号未登录"))
			return false, nil
		}
		return key.isLogin, &userStatus{
			isVip:       key.isVip,
			isBlacklist: key.isBlacklist,
			isWhitelist: key.isWhitelist,
		}
	}

	status, err := b.isAuth(ctx, accessKey)
	if err != nil {
		b.setKey(accessKey, false, &userStatus{})
		writeErrorJSON(ctx, -101, []byte("账号未登录"))
		return false, nil
	}

	b.setKey(accessKey, true, status)

	if status.isBlacklist {
		writeErrorJSON(ctx, -101, []byte("黑名单"))
		return false, nil
	}

	return true, status
}
