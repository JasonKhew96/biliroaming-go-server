package main

import (
	"crypto/md5"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/JasonKhew96/biliroaming-go-server/database"
	"github.com/JasonKhew96/biliroaming-go-server/entity"
	"github.com/mailru/easyjson"
	"github.com/valyala/fasthttp"

	"github.com/JasonKhew96/biliroaming-go-server/entity/android"
	"github.com/JasonKhew96/biliroaming-go-server/entity/bstar"
	"github.com/JasonKhew96/biliroaming-go-server/entity/web"
)

var reMid = regexp.MustCompile(`(&|\\u0026)mid=\d+`)

// ClientType ...
type ClientType int

// ClientType
const (
	ClientTypeUnknown ClientType = iota
	ClientTypeAndroid
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

// biliArgs query arguments struct
type biliArgs struct {
	accessKey string
	area      string
	cid       int64
	epId      int64
	seasonId  int64
	keyword   string
	pn        int
	page      int
	qn        int
	aType     int
	fnval     int
	appkey    string
	ts        int64
	sign      string
}

// SignParams sign params according to client type
func SignParams(values url.Values, clientType ClientType) (string, error) {
	return signParams(values, clientType, time.Now().Unix())
}

func signParams(values url.Values, clientType ClientType, timestamp int64) (string, error) {
	appkey, appsec := getSecrets(clientType)

	values.Set("ts", strconv.FormatInt(timestamp, 10))
	values.Set("appkey", appkey)

	encoded := values.Encode() + appsec
	data := []byte(encoded)
	values.Set("sign", fmt.Sprintf("%x", md5.Sum(data)))
	return values.Encode(), nil
}

func getClientTypeFromAppkey(appkey string) ClientType {
	if appkey == appkeyAndroid {
		return ClientTypeAndroid
	} else if appkey == appkeyBstarA {
		return ClientTypeBstarA
	}
	return ClientTypeUnknown
}

func getSecrets(clientType ClientType) (appkey, appsec string) {
	switch clientType {
	case ClientTypeAndroid:
		return appkeyAndroid, appsecAndroid
	case ClientTypeBstarA:
		return appkeyBstarA, appsecBstarA
	default:
		return "", ""
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

// getFormatType ...
//    [   0] - FLV
//    [   1] - MP4
//    [   2] - ?
//    [   4] - ?
//    [   8] - ?
//    [  16] - DASH
//    [  32] - ?
//    [  64] - [DASH |QN 125] HDR
//    [ 128] - [FOURK|QN 120] 4K
//    [ 256] - [DASH |      ] DOLBY AUDIO
//    [ 512] - [DASH |      ] DOLBY VISION
//    [1024] - [DASH |QN 127] 8K
//    [2048] - [DASH |      ] AV1
//
//    FLV     0
//    MP4     1
//    DASH 4048
func getFormatType(fnval int) database.FormatType {
	if fnval == 0 {
		return database.FormatTypeFlv
	} else if fnval&1 == 1 {
		return database.FormatTypeMp4
	} else if fnval&16 == 16 {
		return database.FormatTypeDash
	} else {
		return database.FormatTypeUnknown
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
		if playUrl.Code != 0 {
			return data, nil
		}
		playUrl.Quality = qn
		return easyjson.Marshal(playUrl)
	case ClientTypeBstarA:
		var playUrl bstar.PlayUrlResult
		if err := easyjson.Unmarshal([]byte(data), &playUrl); err != nil {
			return nil, err
		}
		if playUrl.Code != 0 {
			return data, nil
		}
		playUrl.Data.VideoInfo.Quality = qn
		return easyjson.Marshal(playUrl)
	default:
		var playUrl web.PlayUrlResult
		if err := easyjson.Unmarshal([]byte(data), &playUrl); err != nil {
			return nil, err
		}
		if playUrl.Code != 0 {
			return data, nil
		}
		playUrl.Result.Quality = qn
		return easyjson.Marshal(playUrl)
	}
}

func (b *BiliroamingGo) processArgs(args *fasthttp.Args) *biliArgs {
	cid, err := strconv.ParseInt(string(args.Peek("cid")), 10, 64)
	if err != nil {
		cid = 0
	}
	epId, err := strconv.ParseInt(string(args.Peek("ep_id")), 10, 64)
	if err != nil {
		epId = 0
	}
	seasonId, err := strconv.ParseInt(string(args.Peek("season_id")), 10, 64)
	if err != nil {
		seasonId = 0
	}
	pn, err := strconv.Atoi(string(args.Peek("pn")))
	if err != nil {
		pn = 0
	}
	page, err := strconv.Atoi(string(args.Peek("page")))
	if err != nil {
		page = 0
	}
	qn, err := strconv.Atoi(string(args.Peek("qn")))
	if err != nil || qn == 0 {
		qn = 16
	}
	aType, err := strconv.Atoi(string(args.Peek("type")))
	if err != nil || aType == 0 {
		aType = 7
	}
	fnval, err := strconv.Atoi(string(args.Peek("fnval")))
	if err != nil || fnval == 0 {
		fnval = 4048
	}
	ts, err := strconv.ParseInt(string(args.Peek("ts")), 10, 64)
	if err != nil {
		ts = 0
	}

	queryArgs := &biliArgs{
		accessKey: string(args.Peek("access_key")),
		area:      strings.ToLower(string(args.Peek("area"))),
		cid:       cid,
		epId:      epId,
		seasonId:  seasonId,
		keyword:   string(args.Peek("keyword")),
		pn:        pn,
		page:      page,
		qn:        qn,
		aType:     aType,
		fnval:     fnval,
		appkey:    string(args.Peek("appkey")),
		ts:        ts,
		sign:      string(args.Peek("sign")),
	}

	b.sugar.Debug("Request args ", args.String())
	b.sugar.Debugf(
		"Parsed request args: %v",
		queryArgs,
	)
	return queryArgs
}
