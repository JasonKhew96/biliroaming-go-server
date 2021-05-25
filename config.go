package main

import (
	"github.com/kelseyhightower/envconfig"
)

// Config ...
type Config struct {
	Debug   bool `default:"false"` // 调试模式
	Port    int  `default:"23333"` // 端口
	IPLimit int  `default:"1"`     // 每秒限制次数
	IPBurst int  `default:"2"`     // 每秒突发次数
	// 缓存时间
	AccesskeyCache  int `default:"7"`  // accessKey 缓存（天）
	UserCache       int `default:"7"`  // 用户资料缓存（天）
	PlayURLCache    int `default:"15"` // 播放链接缓存（分钟）
	THSeasonCache   int `default:"15"` // 东南亚 season api 缓存（分钟）
	THSubtitleCache int `default:"15"` // 东南亚 字幕 api 缓存（分钟）
	// 代理(留空禁用)(优先)
	ProxyCN string
	ProxyHK string
	ProxyTW string
	ProxyTH string
	// 反代(留空禁用)
	ReverseCN string
	ReverseHK string
	ReverseTW string
	ReverseTH string
	// 鉴权+缓存
	AuthCN bool `default:"false"`
	AuthHK bool `default:"false"`
	AuthTW bool `default:"false"`
	AuthTH bool `default:"false"`
	// PostgreSQL
	PGHost     string
	PGUser     string
	PGPassword string
	PGDBName   string
	PGPort     int
}

func initConfig() (*Config, error) {
	var c Config
	err := envconfig.Process("server", &c)
	if err != nil {
		return nil, err
	}
	return &c, nil
}
