package configuration

import (
	"FluxGate/loadbalancer"
	"sync"
)

type GatewayConfigStore struct {
	mu    sync.RWMutex
	users map[string][]*RouteConfig
}

type RouteConfig struct {
	Path          string                    `json:"path"`
	Method        string                    `json:"method"`
	Upstreams     []UpstreamConfig          `json:"upstreams"`
	LoadBalancing string                    `json:"load_balancing"`
	LoadBalancer  loadbalancer.LoadBalancer `json:"-"`
	RateLimit     RateLimitConfig           `json:"rate_limit"`
	CacheTTL      int64                     `json:"cache_ttl"`
	CacheEnabled  bool                      `json:"cache_enabled"`

	CircuitBreaker CircuitBreakerConfig `json:"circuit_breaker"`

	Plugins []string `json:"plugins"`
}

type UpstreamConfig struct {
	URL          string `json:"url"`
	Weight       int    `json:"weight"`
	RetryEnabled bool   `json:"retry_enabled"`
	Retries      int    `json:"retries"`
	BaseTimeMs   int64  `json:"base_time_ms"`
}

type RateLimitConfig struct {
	Enabled bool  `json:"enabled"`
	Rate    int64 `json:"rate"`
}

type CircuitBreakerConfig struct {
	Enabled         bool    `json:"enabled"`
	ErrorThreshold  float64 `json:"error_threshold"`
	HalfOpenRequest int64   `json:"half_open_request"`
}
