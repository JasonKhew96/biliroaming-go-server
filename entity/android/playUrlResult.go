package android

type PlayUrlResult struct {
	AcceptFormat      string        `json:"accept_format"`
	Code              int           `json:"code"`
	SeekParam         string        `json:"seek_param"`
	IsPreview         int           `json:"is_preview"`
	Fnval             int           `json:"fnval"`
	VideoProject      bool          `json:"video_project"`
	Fnver             int           `json:"fnver"`
	Type              string        `json:"type"`
	Bp                int           `json:"bp"`
	Result            string        `json:"result"`
	SeekType          string        `json:"seek_type"`
	VipType           int           `json:"vip_type"`
	From              string        `json:"from"`
	VideoCodecid      int           `json:"video_codecid"`
	NoRexcode         int           `json:"no_rexcode"`
	Format            string        `json:"format"`
	SupportFormats    []interface{} `json:"support_formats"`
	Message           string        `json:"message"`
	AcceptQuality     []int         `json:"accept_quality"`
	Quality           int           `json:"quality"`
	Timelength        int           `json:"timelength"`
	HasPaid           bool          `json:"has_paid"`
	VipStatus         int           `json:"vip_status"`
	Dash              interface{}   `json:"dash,omitempty"`
	DUrl              []interface{} `json:"durl,omitempty"`
	ClipInfoList      []interface{} `json:"clip_info_list"`
	AcceptDescription []string      `json:"accept_description"`
	Status            int           `json:"status"`
}
