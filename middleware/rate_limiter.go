package middleware

import (
	"FluxGate/configuration"
	"FluxGate/ratelimit"
	"log"
	"net/http"
	"strings"
)

func RateLimiter(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		// Get route from context
		val := r.Context().Value(configuration.RouteCtxKey)
		if val == nil {
			next.ServeHTTP(w, r)
			return
		}

		route := val.(*configuration.RouteConfig)

		if route.RouteRateLimiter == nil {
			log.Printf("NO ratelimiter found")
		}

		if route.RouteRateLimiter != nil {
			if !route.RouteRateLimiter.Allow() {
				http.Error(w, "route limit exceeded", http.StatusTooManyRequests)
				return
			}
			log.Printf("[rate_limiter] route limit passed for %s %s", r.Method, r.URL.Path)
		}

		// Identify user
		userID := identifyUser(r, route)

		// Get/create limiter
		limiterAny, _ := route.UserRateLimiter.LoadOrStore(
			userID,
			ratelimit.New(
				route.UserRateLimit.Type,
				route.UserRateLimit.Capacity,
				route.UserRateLimit.RefillRate,
			),
		)

		limiter := limiterAny.(ratelimit.RateLimiter)

		if !limiter.Allow() {
			http.Error(w, "user limit exceeded", http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func identifyUser(r *http.Request, route *configuration.RouteConfig) string {

	for _, key := range route.UserIdentityKey {
		parts := strings.SplitN(key, ":", 2)
		if len(parts) != 2 {
			continue
		}

		switch parts[0] {

		case "header":
			v := r.Header.Get(parts[1])
			if v != "" {
				return "hdr:" + v
			}

		case "query":
			v := r.URL.Query().Get(parts[1])
			if v != "" {
				return "qry:" + v
			}

		case "cookie":
			c, err := r.Cookie(parts[1])
			if err == nil && c.Value != "" {
				return "cookie:" + c.Value
			}

		case "form":
			r.ParseForm()
			v := r.Form.Get(parts[1])
			if v != "" {
				return "form:" + v
			}

		case "basic":
			// username:password
			username, _, ok := r.BasicAuth()
			if ok && username != "" {
				return "basic:" + username
			}

		case "jwt":
			//extract from JWT token
			v := r.Header.Get("Authorization")
			if strings.HasPrefix(v, "Bearer ") {
				v = strings.TrimPrefix(v, "Bearer ")
			}
			if v != "" {
				return "jwt:" + v
			}

		case "ip":
			return "ip:" + realClientIP(r)
		}
	}

	return "ip:" + realClientIP(r)
}

func realClientIP(r *http.Request) string {

	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		return strings.Split(xff, ",")[0]
	}
	return r.RemoteAddr
}
