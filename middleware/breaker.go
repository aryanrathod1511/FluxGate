package middleware

import (
	"FluxGate/circuitbreaker"
	"FluxGate/configuration"
	"net/http"
)

const UpstreamCtxKey = configuration.UpstreamCtxKey

func CircuitBreakerMiddleware(breakers map[string]*circuitbreaker.CircuitBreaker) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// extract upstream from context
			val := r.Context().Value(UpstreamCtxKey)
			if val == nil {
				// no upstream selected yet — let the request proceed
				next.ServeHTTP(w, r)
				return
			}

			upstream, ok := val.(string)
			if !ok || upstream == "" {
				next.ServeHTTP(w, r)
				return
			}

			cb := breakers[upstream]

			// capture response status
			rec := &statusRecorder{ResponseWriter: w, status: 0}
			next.ServeHTTP(rec, r)

			if cb != nil {
				status := rec.status
				if status == 0 {
					// no explicit status set, assume 200
					status = http.StatusOK
				}
				if status >= 500 {
					cb.OnFailure()
				} else {
					cb.OnSuccess()
				}
			}
		})
	}
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}
