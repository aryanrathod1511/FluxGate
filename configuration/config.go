package configuration

import (
	"FluxGate/loadbalancer"
	"FluxGate/ratelimit"
	"FluxGate/storage"
	"sync"
	"time"
)

type GatewayConfigStore struct {
	mu    sync.RWMutex
	Users map[string][]*RouteConfig
}

// shared context key type and keys used across packages
type CtxKey string

const RouteCtxKey CtxKey = "route"
const UpstreamCtxKey CtxKey = "upstream"

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

	// Retry configuration (route-level)
	Retry RetryConfig `json:"retry"`

	Cache         CacheConfig       `json:"cache"`
	CacheInstance *storage.LRUCache `json:"-"`

	UserIdentityKey []string `json:"user_id_key"`
	Plugins         []string `json:"plugins"`
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

type RetryConfig struct {
	Enabled    bool  `json:"enabled"`
	MaxTries   int   `json:"max_tries"`
	BaseTimeMs int64 `json:"base_time_ms"`
}

type UpstreamConfig struct {
	URL            string               `json:"url"`
	Weight         int                  `json:"weight"`
	RetryEnabled   bool                 `json:"retry_enabled"`
	Retries        int                  `json:"retries"`
	BaseTimeMs     int64                `json:"base_time_ms"`
	CircuitBreaker CircuitBreakerConfig `json:"circuit_breaker"`
}

type CircuitBreakerConfig struct {
	Enabled bool `json:"enabled"`

	FailureThreshold int `json:"failure_threshold"`
	WindowSeconds    int `json:"window_seconds"`

	OpenSeconds      int `json:"open_seconds"`
	HalfOpenRequests int `json:"half_open_requests"`
	SuccessThreshold int `json:"success_threshold"`
}

type CacheConfig struct {
	Enabled  bool          `json:"enabled"`
	TTL      time.Duration `json:"ttl_ms"`
	MaxEntry int           `json:"max_entry"`
}
