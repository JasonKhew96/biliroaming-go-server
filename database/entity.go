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
	Key       string    `gorm:"column:key; primarykey"` // key
	UID       int       `gorm:"column:uid"`             // 用户 ID
	CreatedAt time.Time `gorm:"column:created_at"`      // 创建时间
	UpdatedAt time.Time `gorm:"column:updated_at"`      // 更新时间
}

// Users 用户资料
type Users struct {
	UID        int       `gorm:"column:uid; primarykey"` // 用户 ID
	VIPDueDate time.Time `gorm:"column:vip_due_date"`    // VIP 到期时间
	Name       string    `gorm:"column:name"`            // 用户暱称
	CreatedAt  time.Time `gorm:"column:created_at"`      // 创建时间
	UpdatedAt  time.Time `gorm:"column:updated_at"`      // 更新时间
}

// PlayURLCache 播放链接缓存
type PlayURLCache struct {
	ID         uint       `gorm:"column:id; primarykey"`        // ...
	IsVip      *bool      `gorm:"column:is_vip; default:false"` // 大会员
	CID        int        `gorm:"column:cid"`                   // cid
	Area       Area       `gorm:"column:area"`                  // 地区
	DeviceType DeviceType `gorm:"column:device_type"`           // 装置种类
	EpisodeID  int        `gorm:"column:episode_id"`            // 剧集 ID
	JSONData   string     `gorm:"column:json_data"`             // 内容
	CreatedAt  time.Time  `gorm:"column:created_at"`            // 创建时间
	UpdatedAt  time.Time  `gorm:"column:updated_at"`            // 更新时间
}

// THSeasonCache season 缓存
type THSeasonCache struct {
	SeasonID  int       `gorm:"column:season_id; primarykey"` // ...
	IsVip     *bool     `gorm:"column:is_vip; default:false"` // 大会员
	JSONData  string    `gorm:"column:json_data"`             // 内容
	CreatedAt time.Time `gorm:"column:created_at"`            // 创建时间
	UpdatedAt time.Time `gorm:"column:updated_at"`            // 更新时间
}

// THSeasonEpisodeCache season episode ID 缓存
type THSeasonEpisodeCache struct {
	EpisodeID int       `gorm:"column:episode_id; primarykey"` // ...
	IsVip     *bool     `gorm:"column:is_vip; default:false"`  // 大会员
	JSONData  string    `gorm:"column:json_data"`              // 内容
	CreatedAt time.Time `gorm:"column:created_at"`             // 创建时间
	UpdatedAt time.Time `gorm:"column:updated_at"`             // 更新时间
}

// THSubtitleCache 字幕缓存
type THSubtitleCache struct {
	EpisodeID int       `gorm:"column:episode_id; primarykey"` // ...
	JSONData  string    `gorm:"column:json_data"`              // 内容
	CreatedAt time.Time `gorm:"column:created_at"`             // 创建时间
	UpdatedAt time.Time `gorm:"column:updated_at"`             // 更新时间
}

// History 历史记录(统计)
// type History struct {
// 	gorm.Model
// 	EpisodeID int
// 	Area      Area
// }
