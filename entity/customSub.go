package entity

type CustomSubResponse struct {
	Code int             `json:"code"`
	Data []CustomSubData `json:"data"`
}

type CustomSubData struct {
	Ep   int    `json:"ep"`
	Key  string `json:"key"`
	Lang string `json:"lang"`
	URL  string `json:"url"`
}
