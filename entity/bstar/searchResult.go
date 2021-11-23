package bstar

type SearchResult struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	TTL     int    `json:"ttl"`
	Data    struct {
		Pages int           `json:"pages"`
		Total int           `json:"total"`
		Items []interface{} `json:"items"`
	} `json:"data"`
}
