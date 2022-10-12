package entity

type BlackWhitelist struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    struct {
		UID         int   `json:"uid"`
		Status      int8  `json:"status"`
		IsWhitelist bool  `json:"is_whitelist"`
		BanUntil    int64 `json:"ban_until"`
	} `json:"data"`
}
