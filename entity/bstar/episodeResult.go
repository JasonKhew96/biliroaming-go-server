package bstar

type EpisodeResult struct {
	Code    int               `json:"code"`
	Message string            `json:"message"`
	TTL     int               `json:"ttl"`
	Data    EpisodeResultData `json:"data"`
}

type EpisodeResultData struct {
	SubtitleSuggestKey string      `json:"subtitle_suggest_key"`
	Jump               interface{} `json:"jump"`
	Subtitles          []Subtitles `json:"subtitles"`
}
