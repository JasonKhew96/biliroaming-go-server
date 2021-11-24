package database

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
