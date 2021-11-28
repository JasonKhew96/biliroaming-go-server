package entity

type BlackWhitelist struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    struct {
		UID         int    `json:"uid"`
		IsBlacklist bool   `json:"is_blacklist"`
		IsWhitelist bool   `json:"is_whitelist"`
		Reason      string `json:"reason"`
	} `json:"data"`
}
