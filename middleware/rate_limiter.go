package middleware

import "net/http"

func RateLimiter(next http.Handler) http.Handler {
	function := func(w http.ResponseWriter, r *http.Request) {
		//logic
		next.ServeHTTP(w, r)
	}
	return http.HandlerFunc(function)
}
