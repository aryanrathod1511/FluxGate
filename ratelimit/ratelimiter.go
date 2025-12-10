package ratelimit

type RateLimiter interface {
	Allow() bool
}
