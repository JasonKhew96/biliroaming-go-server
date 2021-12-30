package web

type SearchResult struct {
	Code    int        `json:"code"`
	Message string     `json:"message"`
	TTL     int        `json:"ttl"`
	Data    SearchData `json:"data"`
}

type SearchData struct {
	Seid           string        `json:"seid"`
	Page           int           `json:"page"`
	Pagesize       int           `json:"pagesize"`
	NumResults     int           `json:"numResults"`
	NumPages       int           `json:"numPages"`
	SuggestKeyword string        `json:"suggest_keyword"`
	RqtType        string        `json:"rqt_type"`
	CostTime       interface{}   `json:"cost_time"`
	ExpList        interface{}   `json:"exp_list"`
	EggHit         int           `json:"egg_hit"`
	Result         []interface{} `json:"result"`
	ShowColumn     int           `json:"show_column"`
}
