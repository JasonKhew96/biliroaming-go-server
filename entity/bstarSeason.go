package entity

type SeasonResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Result  Result `json:"result"`
	Success bool   `json:"success"`
}
type Rights struct {
	Copyright       string `json:"copyright"`
	AllowBp         int    `json:"allow_bp"`
	AllowDownload   int    `json:"allow_download"`
	AreaLimit       int    `json:"area_limit"`
	AllowReview     int    `json:"allow_review"`
	IsPreview       int    `json:"is_preview"`
	BanAreaShow     int    `json:"ban_area_show"`
	AllowBpRank     int    `json:"allow_bp_rank"`
	CanWatch        int    `json:"can_watch"`
	Forbidpre       int    `json:"forbidPre"`
	Onlyvipdownload int    `json:"onlyVipDownload"`
}
type Publish struct {
	PubTime         string `json:"pub_time"`
	PubTimeShow     string `json:"pub_time_show"`
	IsStarted       int    `json:"is_started"`
	IsFinish        int    `json:"is_finish"`
	Weekday         int    `json:"weekday"`
	ReleaseDateShow string `json:"release_date_show"`
	TimeLengthShow  string `json:"time_length_show"`
	UnknowPubDate   int    `json:"unknow_pub_date"`
}
type Actor struct {
	Title string `json:"title"`
	Info  string `json:"info"`
}
type Styles struct {
	ID   int    `json:"id"`
	URL  string `json:"url"`
	Name string `json:"name"`
}
type Dimension struct {
	Width  int `json:"width"`
	Height int `json:"height"`
	Rotate int `json:"rotate"`
}
type Subtitles struct {
	ID        int64  `json:"id"`
	Key       string `json:"key"`
	Title     string `json:"title"`
	URL       string `json:"url"`
	IsMachine bool   `json:"is_machine"`
}
type Episodes struct {
	Aid              int         `json:"aid"`
	Cid              int         `json:"cid"`
	Cover            string      `json:"cover"`
	ID               int         `json:"id"`
	Title            string      `json:"title"`
	LongTitle        string      `json:"long_title"`
	Status           int         `json:"status"`
	From             string      `json:"from"`
	ShareURL         string      `json:"share_url"`
	Dimension        Dimension   `json:"dimension"`
	Jump             interface{} `json:"jump"`
	TitleDisplay     string      `json:"title_display"`
	LongTitleDisplay string      `json:"long_title_display"`
	Subtitles        []Subtitles `json:"subtitles"`
}
type Data struct {
	Episodes []Episodes `json:"episodes"`
}
type ModuleStyle struct {
	Line      int         `json:"line"`
	Hidden    int         `json:"hidden"`
	ShowPages interface{} `json:"show_pages"`
}
type Modules struct {
	ID          int         `json:"id"`
	Style       string      `json:"style"`
	Title       string      `json:"title"`
	More        string      `json:"more"`
	CanOrdDesc  int         `json:"can_ord_desc"`
	Data        Data        `json:"data"`
	ModuleStyle ModuleStyle `json:"module_style"`
	Partition   int         `json:"partition"`
}
type UserStatus struct {
	Follow           int         `json:"follow"`
	Vip              int         `json:"vip"`
	LikeState        int         `json:"like_state"`
	DemandNoPayEpids interface{} `json:"demand_no_pay_epids"`
}
type NewEp struct {
	ID           int    `json:"id"`
	Title        string `json:"title"`
	NewEpDisplay string `json:"new_ep_display"`
}
type Stat struct {
	Favorites  int    `json:"favorites"`
	Views      int    `json:"views"`
	Danmakus   int    `json:"danmakus"`
	Coins      int    `json:"coins"`
	Reply      int    `json:"reply"`
	Share      int    `json:"share"`
	Hot        int    `json:"hot"`
	Play       string `json:"play"`
	Followers  string `json:"followers"`
	SeriesPlay string `json:"series_play"`
	Likes      int    `json:"likes"`
}
type StatFormat struct {
	Play  string `json:"play"`
	Likes string `json:"likes"`
	Share string `json:"share"`
	Reply string `json:"reply"`
}
type Areas struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}
type Result struct {
	SeasonID           int         `json:"season_id"`
	Alias              string      `json:"alias"`
	CommentRestriction string      `json:"comment_restriction"`
	NoComment          string      `json:"no_comment"`
	Title              string      `json:"title"`
	Subtitle           string      `json:"subtitle"`
	DynamicSubtitle    string      `json:"dynamic_subtitle"`
	SeasonTitle        string      `json:"season_title"`
	SquareCover        string      `json:"square_cover"`
	RefineCover        string      `json:"refine_cover"`
	ShareURL           string      `json:"share_url"`
	ShareCopy          string      `json:"share_copy"`
	ShortLink          string      `json:"short_link"`
	Evaluate           string      `json:"evaluate"`
	Link               string      `json:"link"`
	Type               int         `json:"type"`
	TypeName           string      `json:"type_name"`
	Mode               int         `json:"mode"`
	Status             int         `json:"status"`
	Total              int         `json:"total"`
	Rights             Rights      `json:"rights"`
	Publish            Publish     `json:"publish"`
	Detail             string      `json:"detail"`
	Staff              interface{} `json:"staff"`
	Actor              Actor       `json:"actor"`
	OriginName         string      `json:"origin_name"`
	Styles             []Styles    `json:"styles"`
	Modules            []Modules   `json:"modules"`
	UpInfo             interface{} `json:"up_info"`
	UserStatus         UserStatus  `json:"user_status"`
	NewEp              NewEp       `json:"new_ep"`
	Rating             interface{} `json:"rating"`
	Stat               Stat        `json:"stat"`
	StatFormat         StatFormat  `json:"stat_format"`
	Cover              string      `json:"cover"`
	HorizonCover       string      `json:"horizon_cover"`
	Areas              []Areas     `json:"areas"`
	Limit              interface{} `json:"limit"`
	Payment            interface{} `json:"payment"`
	ActivityDialog     interface{} `json:"activity_dialog"`
	LoginDialog        interface{} `json:"login_dialog"`
	UpdatePartten      string      `json:"update_partten"`
	SubtitleSuggestKey string      `json:"subtitle_suggest_key"`
	OpenSkipSwitch     bool        `json:"open_skip_switch"`
}
