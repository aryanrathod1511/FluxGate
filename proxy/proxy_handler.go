package proxy

import (
	"FluxGate/circuitbreaker"
	"FluxGate/configuration"
	"net/http"
	"time"
)

func ProxyHandler(breakers map[string]*circuitbreaker.CircuitBreaker) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upstream := r.Context().Value(configuration.UpstreamCtxKey).(string)
		timeout := 5 * time.Second
		ReverseProxy(w, r, upstream, timeout)
	})
}
