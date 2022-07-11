package entity

// SimpleResponse simple response
//easyjson:json
type SimpleResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}
