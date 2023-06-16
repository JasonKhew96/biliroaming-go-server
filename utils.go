package main

import (
	"bytes"
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
type ClientType string

// ClientType
// taken from https://github.com/yujincheng08/BiliRoaming/wiki/%E8%87%AA%E5%BB%BA%E8%A7%A3%E6%9E%90%E6%9C%8D%E5%8A%A1%E5%99%A8#api-%E8%AF%B7%E6%B1%82%E7%AD%BE%E5%90%8D
const (
	ClientTypeUnknown            ClientType = "unknown"
	ClientTypeAi4cCreatorAndroid ClientType = "ai4c_creator_android"
	ClientTypeAndroid            ClientType = "android"
	ClientTypeAndroidB           ClientType = "android_b"
	ClientTypeAndroidBiliThings  ClientType = "android_bilithings"
	ClientTypeAndroidHD          ClientType = "android_hd"
	ClientTypeAndroidI           ClientType = "android_i"
	ClientTypeAndroidMallTicket  ClientType = "android_mall_ticket"
	ClientTypeAndroidOttSdk      ClientType = "android_ott_sdk"
	ClientTypeAndroidTV          ClientType = "android_tv"
	ClientTypeAnguAndroid        ClientType = "angu_android"
	ClientTypeBiliLink           ClientType = "biliLink"
	ClientTypeBiliScan           ClientType = "biliScan"
	ClientTypeBstarA             ClientType = "bstar_a"
	ClientTypeWeb                ClientType = "web"
	ClientTypeIphone             ClientType = "iphone"
)

func (c *ClientType) IsValid() bool {
	switch *c {
	case ClientTypeAi4cCreatorAndroid,
		ClientTypeAndroid,
		ClientTypeAndroidB,
		ClientTypeAndroidBiliThings,
		ClientTypeAndroidHD,
		ClientTypeAndroidI,
		ClientTypeAndroidMallTicket,
		ClientTypeAndroidOttSdk,
		ClientTypeAndroidTV,
		ClientTypeAnguAndroid,
		ClientTypeBiliLink,
		ClientTypeBiliScan,
		ClientTypeBstarA,
		ClientTypeWeb,
		ClientTypeIphone:
		return true
	default:
		return false
	}
}

func (c ClientType) String() string {
	return string(c)
}

// appkey
const (
	appkeyAi4cCreatorAndroid = "9d5889cf67e615cd"
	appkeyAndroid            = "1d8b6e7d45233436"
	appkeyAndroidB           = "07da50c9a0bf829f"
	appkeyAndroidBiliThings  = "8d23902c1688a798"
	appkeyAndroidHD          = "dfca71928277209b"
	appkeyAndroidI           = "bb3101000e232e27"
	appkeyAndroidMallTicket  = "4c6e1021617d40d9"
	appkeyAndroidOttSdk      = "c034e8b74130a886"
	appkeyAndroidTV          = "4409e2ce8ffd12b8"
	appkeyAnguAndroid        = "50e1328c6a1075a1"
	appkeyBiliLink           = "37207f2beaebf8d7"
	appkeyBiliScan           = "9a75abf7de2d8947"
	appkeyBstarA             = "7d089525d3611b1c"
	appkeyIphone             = "27eb53fc9058f8c3"
)

// appsec
const (
	appsecAi4cCreatorAndroid = "8fd9bb32efea8cef801fd895bef2713d"
	appsecAndroid            = "560c52ccd288fed045859ed18bffd973"
	appsecAndroidB           = "25bdede4e1581c836cab73a48790ca6e"
	appsecAndroidBiliThings  = "710f0212e62bd499b8d3ac6e1db9302a"
	appsecAndroidHD          = "b5475a8825547a4fc26c7d518eaaa02e"
	appsecAndroidI           = "36efcfed79309338ced0380abd824ac1"
	appsecAndroidMallTicket  = "e559a59044eb2701b7a8628c86aa12ae"
	appsecAndroidOttSdk      = "e4e8966b1e71847dc4a3830f2d078523"
	appsecAndroidTV          = "59b43e04ad6965f34319062b478f83dd"
	appsecAnguAndroid        = "4d35e3dea073433cd24dd14b503d242e"
	appsecBiliLink           = "e988e794d4d4b6dd43bc0e89d6e90c43"
	appsecBiliScan           = "35ca1c82be6c2c242ecc04d88c735f31"
	appsecBstarA             = "acd495b248ec528c2eed1e862d393126"
	appsecIphone             = "c2ed53a74eeefe3cf99fbd01d8c9c375"
)

// biliArgs query arguments struct
type biliArgs struct {
	accessKey      string
	area           string
	cid            int64
	epId           int64
	seasonId       int64
	keyword        string
	pn             int
	page           int
	qn             int
	aType          int
	fnval          int
	appkey         string
	ts             int64
	sign           string
	preferCodeType bool
}

// SignParams sign params according to client type
func SignParams(values url.Values, clientType ClientType) (string, error) {
	return signParams(values, clientType, time.Now().Unix())
}

func signParams(values url.Values, clientType ClientType, timestamp int64) (string, error) {
	sign, err := getSign(values, clientType, timestamp)
	if err != nil {
		return "", err
	}
	values.Set("sign", sign)
	return values.Encode(), nil
}

func getSign(values url.Values, clientType ClientType, timestamp int64) (string, error) {
	appkey, appsec := getSecrets(clientType)

	values.Set("ts", strconv.FormatInt(timestamp, 10))
	values.Set("appkey", appkey)

	encoded := values.Encode() + appsec
	data := []byte(encoded)
	return fmt.Sprintf("%x", md5.Sum(data)), nil
}

func getClientTypeFromAppkey(appkey string) ClientType {
	switch appkey {
	case appkeyAi4cCreatorAndroid:
		return ClientTypeAi4cCreatorAndroid
	case appkeyAndroid:
		return ClientTypeAndroid
	case appkeyAndroidB:
		return ClientTypeAndroidB
	case appkeyAndroidBiliThings:
		return ClientTypeAndroidBiliThings
	case appkeyAndroidHD:
		return ClientTypeAndroidHD
	case appkeyAndroidI:
		return ClientTypeAndroidI
	case appkeyAndroidMallTicket:
		return ClientTypeAndroidMallTicket
	case appkeyAndroidOttSdk:
		return ClientTypeAndroidOttSdk
	case appkeyAndroidTV:
		return ClientTypeAndroidTV
	case appkeyAnguAndroid:
		return ClientTypeAnguAndroid
	case appkeyBiliLink:
		return ClientTypeBiliLink
	case appkeyBiliScan:
		return ClientTypeBiliScan
	case appkeyBstarA:
		return ClientTypeBstarA
	case appkeyIphone:
		fallthrough
	default:
		return ClientTypeIphone
	}
}

func getSecrets(clientType ClientType) (appkey, appsec string) {
	switch clientType {
	case ClientTypeAi4cCreatorAndroid:
		return appkeyAi4cCreatorAndroid, appsecAi4cCreatorAndroid
	case ClientTypeAndroid:
		return appkeyAndroid, appsecAndroid
	case ClientTypeAndroidB:
		return appkeyAndroidB, appsecAndroidB
	case ClientTypeAndroidBiliThings:
		return appkeyAndroidBiliThings, appsecAndroidBiliThings
	case ClientTypeAndroidHD:
		return appkeyAndroidHD, appsecAndroidHD
	case ClientTypeAndroidI:
		return appkeyAndroidI, appsecAndroidI
	case ClientTypeAndroidMallTicket:
		return appkeyAndroidMallTicket, appsecAndroidMallTicket
	case ClientTypeAndroidOttSdk:
		return appkeyAndroidOttSdk, appsecAndroidOttSdk
	case ClientTypeAndroidTV:
		return appkeyAndroidTV, appsecAndroidTV
	case ClientTypeAnguAndroid:
		return appkeyAnguAndroid, appsecAnguAndroid
	case ClientTypeBiliLink:
		return appkeyBiliLink, appsecBiliLink
	case ClientTypeBiliScan:
		return appkeyBiliScan, appsecBiliScan
	case ClientTypeBstarA:
		return appkeyBstarA, appsecBstarA
	case ClientTypeIphone:
		return appkeyIphone, appsecIphone
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
//
//	[   0] - FLV (或许是 自动)
//	[   1] - MP4
//	[   2] - ? (可能是 FLV)
//	[   4] - ?
//	[   8] - ?
//	[  16] - DASH
//	[  32] - ?
//	[  64] - [DASH |QN 125] HDR
//	[ 128] - [FOURK|QN 120] 4K
//	[ 256] - [DASH |      ] DOLBY AUDIO
//	[ 512] - [DASH |      ] DOLBY VISION
//	[1024] - [DASH |QN 127] 8K
//	[2048] - [DASH |      ] AV1
//
//	FLV     0
//	MP4     1
//	FLV     2
//	DASH 4048
func getFormatType(fnval int) database.FormatType {
	if fnval&1 == 1 {
		return database.FormatTypeMp4
	} else if fnval&2 == 2 {
		return database.FormatTypeFlv
	} else if fnval == 0 || fnval&16 == 16 {
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

func isAvailableResponse(data []byte) (bool, error) {
	resp := &entity.SimpleResponse{}
	if err := easyjson.Unmarshal(data, resp); err != nil {
		return false, err
	}

	/*
		{"code":10015002,"message":"访问权限不足","ttl":1}
		{"code":-10403,"message":"大会员专享限制"}
		{"code":-10403,"message":"抱歉您所使用的平台不可观看！"}
		{"code":-10403,"message":"抱歉您所在地区不可观看！"}
		{"code":-400,"message":"请求错误"}
		{"code":-404,"message":"啥都木有"}
		{"code":-404,"message":"啥都木有","ttl":1}
	*/

	if resp.Code == 0 {
		return true, nil
	}
	if resp.Code == 10015002 && resp.Message == "访问权限不足" {
		return true, nil
	}
	if resp.Code == -10403 && resp.Message == "大会员专享限制" {
		return true, nil
	}
	if resp.Code == -10403 && resp.Message == "抱歉您所使用的平台不可观看！" {
		return true, nil
	}
	if resp.Code == -10403 && resp.Message == "抱歉您所在地区不可观看！" {
		return false, nil
	}
	if resp.Code == -404 && resp.Message == "啥都木有" {
		return false, nil
	}

	return false, fmt.Errorf("code: %d, message: %s", resp.Code, resp.Message)
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

func playUrlVipStatus(data []byte, clientType ClientType) (bool, bool, error) {
	switch clientType {
	case ClientTypeAndroid:
		var playUrl android.PlayUrlResult
		if err := easyjson.Unmarshal([]byte(data), &playUrl); err != nil {
			return false, false, err
		}
		return playUrl.Code == 0, playUrl.VipStatus != 0, nil
	case ClientTypeWeb:
		var playUrl web.PlayUrlResult
		if err := easyjson.Unmarshal([]byte(data), &playUrl); err != nil {
			return false, false, err
		}
		return playUrl.Code == 0, playUrl.Result.VipStatus != 0, nil
	case ClientTypeBstarA:
		var playUrl bstar.PlayUrlResult
		if err := easyjson.Unmarshal([]byte(data), &playUrl); err != nil {
			return false, false, err
		}
		return playUrl.Code == 0, false, nil
	}
	return false, false, nil
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
	area := string(args.Peek("area"))
	if area == "" && b.config.DefaultArea != "" {
		area = b.config.DefaultArea
	}
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
	if err != nil {
		qn = 0
	}
	aType, err := strconv.Atoi(string(args.Peek("type")))
	if err != nil {
		aType = 0
	}
	fnval, err := strconv.Atoi(string(args.Peek("fnval")))
	if err != nil {
		fnval = 0
	}
	ts, err := strconv.ParseInt(string(args.Peek("ts")), 10, 64)
	if err != nil {
		ts = 0
	}
	preferCodeType := bytes.EqualFold(args.Peek("prefer_code_type"), []byte("1"))

	accessKey := string(args.Peek("access_key"))
	if len(accessKey) > 32 {
		accessKey = accessKey[:32]
	}

	queryArgs := &biliArgs{
		accessKey:      accessKey,
		area:           strings.ToLower(area),
		cid:            cid,
		epId:           epId,
		seasonId:       seasonId,
		keyword:        string(args.Peek("keyword")),
		pn:             pn,
		page:           page,
		qn:             qn,
		aType:          aType,
		fnval:          fnval,
		appkey:         string(args.Peek("appkey")),
		ts:             ts,
		sign:           string(args.Peek("sign")),
		preferCodeType: preferCodeType,
	}

	b.sugar.Debug("Request args ", args.String())
	b.sugar.Debugf(
		"Parsed request args: %v",
		queryArgs,
	)
	return queryArgs
}
