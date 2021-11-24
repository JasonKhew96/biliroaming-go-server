package main

import (
	"crypto/md5"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/JasonKhew96/biliroaming-go-server/database"
	"github.com/JasonKhew96/biliroaming-go-server/entity"
	"github.com/mailru/easyjson"

	"github.com/JasonKhew96/biliroaming-go-server/entity/android"
	"github.com/JasonKhew96/biliroaming-go-server/entity/bstar"
	"github.com/JasonKhew96/biliroaming-go-server/entity/web"
)

var reMid = regexp.MustCompile(`(&|\\u0026)mid=\d+`)

// ClientType ...
type ClientType int

// ClientType
const (
	ClientTypeAndroid ClientType = iota
	ClientTypeBstarA
	ClientTypeWeb
)

// appkey
const (
	appkeyAndroid = "1d8b6e7d45233436"
	appkeyBstarA  = "7d089525d3611b1c"
)

// appsec
const (
	appsecAndroid = "560c52ccd288fed045859ed18bffd973"
	appsecBstarA  = "acd495b248ec528c2eed1e862d393126"
)

// SignParams sign params according to client type
func SignParams(values url.Values, clientType ClientType) (string, error) {
	return signParams(values, clientType, time.Now().Unix())
}

func signParams(values url.Values, clientType ClientType, timestamp int64) (string, error) {
	appkey, appsec, err := getSecrets(clientType)
	if err != nil {
		return "", err
	}

	values.Set("ts", strconv.FormatInt(timestamp, 10))
	values.Set("appkey", appkey)

	encoded := values.Encode() + appsec
	data := []byte(encoded)
	values.Set("sign", fmt.Sprintf("%x", md5.Sum(data)))
	return values.Encode(), nil
}

func getSecrets(clientType ClientType) (appkey, appsec string, err error) {
	switch clientType {
	case ClientTypeAndroid:
		return appkeyAndroid, appsecAndroid, nil
	case ClientTypeBstarA:
		return appkeyBstarA, appsecBstarA, nil
	default:
		return "", "", errors.New("unknown client type")
	}
}

func getAreaCode(area string) database.Area {
	switch strings.ToLower(area) {
	case "cn":
		return database.AreaCN
	case "hk":
		return database.AreaHK
	case "tw":
		return database.AreaTW
	case "th":
		return database.AreaTH
	default:
		return database.AreaNone
	}
}

func isResponseLimited(data []byte) (bool, error) {
	resp := &entity.SimpleResponse{}
	if err := easyjson.Unmarshal(data, resp); err != nil {
		return false, err
	}
	if resp.Code == -412 {
		return true, nil
	}
	return false, nil
}

func isResponseNotLogin(data []byte) (bool, error) {
	resp := &entity.SimpleResponse{}
	if err := easyjson.Unmarshal(data, resp); err != nil {
		return false, err
	}
	if resp.Code == -101 {
		return true, nil
	}
	return false, nil
}

func removeMid(data string) string {
	s := reMid.FindAllString(data, 1)
	if len(s) > 0 {
		data = strings.ReplaceAll(data, s[0], "")
	}
	return data
}

func replaceQn(data []byte, qn int, clientType ClientType) ([]byte, error) {
	switch clientType {
	case ClientTypeAndroid:
		var playUrl android.PlayUrlResult
		if err := easyjson.Unmarshal([]byte(data), &playUrl); err != nil {
			return nil, err
		}
		playUrl.Quality = qn
		return easyjson.Marshal(playUrl)
	case ClientTypeBstarA:
		var playUrl bstar.PlayUrlResult
		if err := easyjson.Unmarshal([]byte(data), &playUrl); err != nil {
			return nil, err
		}
		playUrl.Data.VideoInfo.Quality = qn
		return easyjson.Marshal(playUrl)
	default:
		var playUrl web.PlayUrlResult
		if err := easyjson.Unmarshal([]byte(data), &playUrl); err != nil {
			return nil, err
		}
		playUrl.Result.Quality = qn
		return easyjson.Marshal(playUrl)
	}
}
