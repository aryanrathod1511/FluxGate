package ratelimit

func New(configType string, capacity float64, refill float64) RateLimiter {
	f, ok := Registry[configType]
	if ok {
		return f(capacity, refill)
	}
	return nil
}
