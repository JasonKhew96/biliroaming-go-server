package bstar

type Subtitles struct {
	ID        int64  `json:"id"`
	Key       string `json:"key"`
	Title     string `json:"title"`
	URL       string `json:"url"`
	IsMachine bool   `json:"is_machine"`
}

type SeasonResult struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Result  struct {
		SeasonID           int           `json:"season_id"`
		Alias              string        `json:"alias"`
		CommentRestriction string        `json:"comment_restriction"`
		NoComment          string        `json:"no_comment"`
		Title              string        `json:"title"`
		Subtitle           string        `json:"subtitle"`
		DynamicSubtitle    string        `json:"dynamic_subtitle"`
		SeasonTitle        string        `json:"season_title"`
		SquareCover        string        `json:"square_cover"`
		RefineCover        string        `json:"refine_cover"`
		ShareURL           string        `json:"share_url"`
		ShareCopy          string        `json:"share_copy"`
		ShortLink          string        `json:"short_link"`
		Evaluate           string        `json:"evaluate"`
		Link               string        `json:"link"`
		Type               int           `json:"type"`
		TypeName           string        `json:"type_name"`
		Mode               int           `json:"mode"`
		Status             int           `json:"status"`
		Total              int           `json:"total"`
		Rights             interface{}   `json:"rights"`
		Publish            interface{}   `json:"publish"`
		Detail             string        `json:"detail"`
		Staff              interface{}   `json:"staff"`
		Actor              interface{}   `json:"actor"`
		OriginName         string        `json:"origin_name"`
		Styles             []interface{} `json:"styles"`
		Modules            []struct {
			ID         int    `json:"id"`
			Style      string `json:"style"`
			Title      string `json:"title"`
			More       string `json:"more"`
			CanOrdDesc int    `json:"can_ord_desc"`
			Data       struct {
				Episodes []struct {
					Aid              int         `json:"aid"`
					Cid              int         `json:"cid"`
					Cover            string      `json:"cover"`
					ID               int         `json:"id"`
					Title            string      `json:"title"`
					LongTitle        string      `json:"long_title"`
					Status           int         `json:"status"`
					From             string      `json:"from"`
					ShareURL         string      `json:"share_url"`
					Dimension        interface{} `json:"dimension"`
					Jump             interface{} `json:"jump"`
					TitleDisplay     string      `json:"title_display"`
					LongTitleDisplay string      `json:"long_title_display"`
					Subtitles        []Subtitles `json:"subtitles"`
				} `json:"episodes"`
			} `json:"data"`
			ModuleStyle interface{} `json:"module_style"`
			Partition   int         `json:"partition"`
		} `json:"modules"`
		UpInfo             interface{}   `json:"up_info"`
		UserStatus         interface{}   `json:"user_status"`
		NewEp              interface{}   `json:"new_ep"`
		Rating             interface{}   `json:"rating"`
		Stat               interface{}   `json:"stat"`
		StatFormat         interface{}   `json:"stat_format"`
		Cover              string        `json:"cover"`
		HorizonCover       string        `json:"horizon_cover"`
		Areas              []interface{} `json:"areas"`
		Limit              interface{}   `json:"limit"`
		Payment            interface{}   `json:"payment"`
		ActivityDialog     interface{}   `json:"activity_dialog"`
		LoginDialog        interface{}   `json:"login_dialog"`
		UpdatePartten      string        `json:"update_partten"`
		Series             interface{}   `json:"series"`
		SubtitleSuggestKey string        `json:"subtitle_suggest_key"`
		OpenSkipSwitch     bool          `json:"open_skip_switch"`
	} `json:"result"`
	Success bool `json:"success"`
}
