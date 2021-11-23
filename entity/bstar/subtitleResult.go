package bstar

type SubtitleResult struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	TTL     int    `json:"ttl"`
	Data    struct {
		SuggestKey string `json:"suggest_key"`
		Subtitles  []struct {
			Key       string `json:"key"`
			ID        int64  `json:"id"`
			URL       string `json:"url"`
			Title     string `json:"title"`
			IsMachine bool   `json:"is_machine"`
		} `json:"subtitles"`
	} `json:"data"`
}
