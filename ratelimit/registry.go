package ratelimit

var Registry = make(map[string]func(float64, float64) RateLimiter)

func RegisterRateLimiter(name string, constructor func(float64, float64) RateLimiter) {
	Registry[name] = constructor
}
