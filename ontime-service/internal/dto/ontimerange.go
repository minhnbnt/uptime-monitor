package dto

import "time"

type CalculateUptimeRequest struct {
	From       time.Time `json:"from"`
	To         time.Time `json:"to"`
	Resolution *string   `json:"resolution,omitempty"`
}

type UptimeResponse struct {
	ServerID      uint             `json:"server_id"`
	Uptime        float64          `json:"uptime"`
	From          string           `json:"from"`
	To            string           `json:"to"`
	TotalSeconds  float64          `json:"total_seconds"`
	OnlineSeconds float64          `json:"online_seconds"`
	Intervals     []IntervalResult `json:"intervals"`
}

type IntervalResult struct {
	From   string  `json:"from"`
	To     string  `json:"to"`
	Uptime float64 `json:"uptime"`
}
