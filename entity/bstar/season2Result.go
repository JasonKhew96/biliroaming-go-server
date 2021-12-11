package bstar

type Season2Result struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	TTL     int         `json:"ttl"`
	Data    Season2Data `json:"data"`
}

type Season2EpDetails struct {
	HorizontalCover  string      `json:"horizontal_cover"`
	Badge            interface{} `json:"badge"`
	EpisodeID        int64       `json:"episode_id"`
	Title            string      `json:"title"`
	ShortTitle       string      `json:"short_title"`
	LongTitle        string      `json:"long_title"`
	LongTitleDisplay string      `json:"long_title_display"`
	Status           int         `json:"status"`
	Jump             interface{} `json:"jump"`
	Dialog           interface{} `json:"dialog"`
	Subtitles        []Subtitles `json:"subtitles"`
	Dimension        interface{} `json:"dimension"`
}
type Season2Section struct {
	Title     string             `json:"title"`
	Style     string             `json:"style"`
	EpDetails []Season2EpDetails `json:"ep_details"`
}
type Season2Sections struct {
	Title       string           `json:"title"`
	EpListTitle string           `json:"ep_list_title"`
	Section     []Season2Section `json:"section"`
}

type Season2Data struct {
	Experiments        interface{}     `json:"experiments"`
	CommentRestriction string          `json:"comment_restriction"`
	NoComment          string          `json:"no_comment"`
	Status             int             `json:"status"`
	Title              string          `json:"title"`
	Limit              string          `json:"limit"`
	UpdateDesc         string          `json:"update_desc"`
	SubtitleSuggestKey string          `json:"subtitle_suggest_key"`
	SeasonID           int64           `json:"season_id"`
	OpenSkipSwitch     bool            `json:"open_skip_switch"`
	AllowDownload      bool            `json:"allow_download"`
	HorizonCover       string          `json:"horizon_cover"`
	EpisodeCardStyle   int             `json:"episode_card_style"`
	InteractiveIcons   []string        `json:"interactive_icons"`
	Remind             interface{}     `json:"remind"`
	UserStatus         interface{}     `json:"user_status"`
	SubscribeGuide     interface{}     `json:"subscribe_guide"`
	Sections           Season2Sections `json:"sections"`
	Info               interface{}     `json:"info"`
	Details            interface{}     `json:"details"`
	Stat               interface{}     `json:"stat"`
	SeasonSeries       []interface{}   `json:"season_series"`
	Related            interface{}     `json:"related"`
	ForYou             interface{}     `json:"for_you"`
}
