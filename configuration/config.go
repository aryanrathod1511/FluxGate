package configuration

import (
	"FluxGate/loadbalancer"
	"FluxGate/ratelimit"
	"sync"
)

type GatewayConfigStore struct {
	mu    sync.RWMutex
	users map[string][]*RouteConfig
}

type RouteConfig struct {
	Path        string           `json:"path"`
	Method      string           `json:"method"`
	Upstreams   []UpstreamConfig `json:"upstreams"`
	LoadBalance string           `json:"load_balancing"`

	// LB instance
	LoadBalancer loadbalancer.LoadBalancer `json:"-"`

	// Rate limit
	RouteRateLimit RouteRateLimitConfig `json:"route_rate_limit"`
	UserRateLimit  UserRateLimitConfig  `json:"user_rate_limit"`

	// Instances
	RouteRateLimiter ratelimit.RateLimiter `json:"-"` // single instance
	UserRateLimiter  sync.Map              `json:"-"` // multiple instances

	CacheTTL     int64 `json:"cache_ttl"`
	CacheEnabled bool  `json:"cache_enabled"`

	CircuitBreaker  CircuitBreakerConfig `json:"circuit_breaker"`
	UserIdentityKey []string             `json:"user_id_key"`
	Plugins         []string             `json:"plugins"`
}

// "user_id_keys": [
//     "jwt:sub",
//     "header:X-API-Key",
//     "jwt:user_id",
//     "query:uid",
//     "ip"
//   ],

type RouteRateLimitConfig struct {
	Capacity   float64 `json:"capacity"`    // max tokens
	RefillRate float64 `json:"refill_rate"` // tokens/sec
	Type       string  `json:"type"`        // "token_bucket" / "none"
}

type UserRateLimitConfig struct {
	Capacity   float64 `json:"capacity"`
	RefillRate float64 `json:"refill_rate"`
	Type       string  `json:"type"`
}

type UpstreamConfig struct {
	URL          string `json:"url"`
	Weight       int    `json:"weight"`
	RetryEnabled bool   `json:"retry_enabled"`
	Retries      int    `json:"retries"`
	BaseTimeMs   int64  `json:"base_time_ms"`
}

type CircuitBreakerConfig struct {
	Enabled         bool    `json:"enabled"`
	ErrorThreshold  float64 `json:"error_threshold"`
	HalfOpenRequest int64   `json:"half_open_request"`
}
