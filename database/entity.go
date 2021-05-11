package database

import (
	"time"

	"gorm.io/gorm"
)

// Area 地区
type Area int

const (
	// AreaCN ...
	AreaCN Area = iota
	// AreaHK ...
	AreaHK
	// AreaTW ...
	AreaTW
	// AreaTH ...
	AreaTH
)

// DeviceType 装置种类
type DeviceType int

const (
	// DeviceTypeWeb ...
	DeviceTypeWeb DeviceType = iota
	// DeviceTypeAndroid ...
	DeviceTypeAndroid
)

// AccessKeys key 缓存
type AccessKeys struct {
	// gorm.Model
	Key       string         `gorm:"primarykey"` // key
	CreatedAt time.Time      // 创建时间
	UpdatedAt time.Time      // 更新时间
	DeletedAt gorm.DeletedAt `gorm:"index"` // 删除时间
	UID       int            // 用户 ID
	DueDate   time.Time      // key 到期时间
}

// Users 用户资料
type Users struct {
	gorm.Model
	UID        int       // 用户 ID
	VIPDueDate time.Time // VIP 到期时间
	Name       string    // 用户暱称
}

// PlayURLCache 播放链接缓存
type PlayURLCache struct {
	gorm.Model
	Area       Area       // 地区
	DeviceType DeviceType // 装置种类
	CID        int        // ...
	EpisodeID  int        // 剧集 ID
	Data       string     // 内容
}

// type History struct {
// 	gorm.Model
// 	CID       int
// 	EpisodeID int
// }
