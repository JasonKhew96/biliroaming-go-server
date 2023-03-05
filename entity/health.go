package entity

import "time"

type Health struct {
	Code    int        `json:"code"`
	Message string     `json:"message"`
	Data    HealthData `json:"data"`
}

type HealthData struct {
	LastCheck time.Time `json:"last_check"`
	Counter   int64     `json:"counter"`
}
