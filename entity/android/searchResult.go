package android

type SearchResult struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	TTL     int    `json:"ttl"`
	Data    struct {
		Pages   int           `json:"pages"`
		Total   int           `json:"total"`
		ExpStr  string        `json:"exp_str"`
		Keyword string        `json:"keyword"`
		Items   []interface{} `json:"items"`
	} `json:"data"`
}
