package ratelimit

import (
	"sync"
	"time"
)

type TokenBucket struct {
	Capacity   float64
	Tokens     float64
	RefillRate float64
	LastRefill time.Time
	mu         sync.Mutex
}

func init() {
	RegisterRateLimiter("token_bucket", func(capacity float64, refillrate float64) RateLimiter {
		return NewTokenBucket(capacity, refillrate)
	})
}

func NewTokenBucket(capacity float64, refillrate float64) *TokenBucket {
	return &TokenBucket{
		Capacity:   capacity,
		Tokens:     capacity,
		RefillRate: refillrate,
		LastRefill: time.Now(),
	}
}

func (tb *TokenBucket) Allow() bool {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	tb.Refill()
	if tb.Tokens >= 1 {
		tb.Tokens -= 1
		return true
	}
	return false

}

func (tb *TokenBucket) Refill() {
	now := time.Now()
	elapsed := now.Sub(tb.LastRefill).Seconds()
	tb.Tokens += elapsed * tb.RefillRate
	if tb.Tokens > tb.Capacity {
		tb.Tokens = tb.Capacity
	}
	tb.LastRefill = now
}
