package main

import (
	"github.com/kelseyhightower/envconfig"
)

// Config ...
type Config struct {
	Debug   bool `envconfig:"debug" default:"false"` // 调试模式
	Port    int  `envconfig:"port" default:"23333"`  // 端口
	IPLimit int  `envconfig:"ip_limit" default:"1"`  // 每秒限制次数
	IPBurst int  `envconfig:"ip_burst" default:"2"`  // 每秒突发次数
	// 黑白名单
	BlockType BlockType `envconfig:"block_type" default:"0"` // 0 - 关闭 / 1 - 白名单 / 2 - 黑名单
	// 字幕(泰区)
	CustomSubAPI  string `envconfig:"custom_sub_api"`  // 自定义字幕API
	CustomSubTeam string `envconfig:"custom_sub_team"` // 自定义字幕组名字
	// 缓存时间
	CacheAccessKey  int `envconfig:"cache_accesskey" default:"7"`    // accessKey 缓存（天）
	CacheUser       int `envconfig:"cache_user" default:"7"`         // 用户资料缓存（天）
	CachePlayURL    int `envconfig:"cache_playurl" default:"15"`     // 播放链接缓存（分钟）
	CacheTHSeason   int `envconfig:"cache_th_season" default:"15"`   // 东南亚 season api 缓存（分钟）
	CacheTHSubtitle int `envconfig:"cache_th_subtitle" default:"15"` // 东南亚 字幕 api 缓存（分钟）
	// 代理(留空禁用)
	ProxyCN string `envconfig:"proxy_cn"`
	ProxyHK string `envconfig:"proxy_hk"`
	ProxyTW string `envconfig:"proxy_tw"`
	ProxyTH string `envconfig:"proxy_th"`
	// 反代(留空禁用)
	ReverseCN string `envconfig:"reverse_cn"`
	ReverseHK string `envconfig:"reverse_hk"`
	ReverseTW string `envconfig:"reverse_tw"`
	ReverseTH string `envconfig:"reverse_th"`
	// 鉴权+缓存
	AuthCN bool `envconfig:"auth_th" default:"false"`
	AuthHK bool `envconfig:"auth_th" default:"false"`
	AuthTW bool `envconfig:"auth_th" default:"false"`
	AuthTH bool `envconfig:"auth_th" default:"false"`
	// PostgreSQL
	PGHost         string `envconfig:"pg_host" default:"localhost"`
	PGUser         string `envconfig:"pg_user"  default:"postgres"`
	PGPassword     string `envconfig:"pg_password"`
	PGPasswordFile string `envconfig:"pg_password_file"`
	PGDBName       string `envconfig:"pg_dbname" default:"postgres"`
	PGPort         int    `envconfig:"pg_port" default:"5432"`
}

func initConfig() (*Config, error) {
	var c Config
	err := envconfig.Process("", &c)
	if err != nil {
		return nil, err
	}
	return &c, nil
}
