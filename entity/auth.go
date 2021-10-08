package entity

// AccInfo account info
//easyjson:json
type AccInfo struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    struct {
		Mid  int    `json:"mid"`
		Name string `json:"name"`
		VIP  struct {
			DueDate int64 `json:"due_date"`
		} `json:"vip"`
	} `json:"data"`
}
