package middleware

import "net/http"

func CircuitBreaker(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		//check reaker state

		//return error or call retry
		next.ServeHTTP(w, r)
	})
}
