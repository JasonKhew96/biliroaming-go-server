package bstar

type PlayUrlResult struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	TTL     int    `json:"ttl"`
	Data    struct {
		VideoInfo struct {
			Quality    int           `json:"quality"`
			Timelength int           `json:"timelength"`
			StreamList []interface{} `json:"stream_list"`
			DashAudio  []interface{} `json:"dash_audio"`
		} `json:"video_info"`
		Dimension interface{} `json:"dimension"`
	} `json:"data"`
}
