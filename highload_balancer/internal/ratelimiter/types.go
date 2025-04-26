package ratelimiter

import "time"

type ClientConfig struct {
	ClientID    string    `json:"client_id"`
	Capacity    int       `json:"capacity"`
	RatePerSec  int       `json:"rate_per_sec"`
	CreatedAt   time.Time `json:"created_at"`
	LastUpdated time.Time `json:"last_updated"`
}

type RateLimitResponse struct {
	Code       int           `json:"code"`
	Message    string        `json:"message"`
	RetryAfter time.Duration `json:"retry_after"`
}
