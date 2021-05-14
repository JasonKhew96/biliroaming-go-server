package database

import (
	"time"
)

// Area 地区
type Area int

// Area
const (
	AreaNone Area = iota
	AreaCN
	AreaHK
	AreaTW
	AreaTH
)

// DeviceType 装置种类
type DeviceType int

// DeviceType
const (
	DeviceTypeWeb DeviceType = iota
	DeviceTypeAndroid
)

// AccessKeys key 缓存
type AccessKeys struct {
	// gorm.Model
	Key       string    `gorm:"primarykey"` // key
	UID       int       // 用户 ID
	CreatedAt time.Time // 创建时间
	UpdatedAt time.Time // 更新时间
}

// Users 用户资料
type Users struct {
	UID        int       `gorm:"primarykey"` // 用户 ID
	VIPDueDate time.Time // VIP 到期时间
	Name       string    // 用户暱称
	CreatedAt  time.Time // 创建时间
	UpdatedAt  time.Time // 更新时间
}

// PlayURLCache 播放链接缓存
type PlayURLCache struct {
	ID         uint       `gorm:"primarykey"` // ...
	IsVip      bool       // 大会员
	CID        int        // ...
	Area       Area       // 地区
	DeviceType DeviceType // 装置种类
	EpisodeID  int        // 剧集 ID
	JSONData   string     // 内容
	CreatedAt  time.Time  // 创建时间
	UpdatedAt  time.Time  // 更新时间
}

// History 历史记录(统计)
// type History struct {
// 	gorm.Model
// 	EpisodeID int
// 	Area      Area
// }
