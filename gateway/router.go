package gateway

import (
	"FluxGate/circuitbreaker"
	"FluxGate/configuration"
	metrics "FluxGate/matrics"
	"FluxGate/middleware"
	"FluxGate/proxy"
	"context"
	"net/http"
	"time"
)

type Gateway struct {
	Store   *configuration.GatewayConfigStore
	Breaker map[string]*circuitbreaker.CircuitBreaker
}

func NewGateway(store *configuration.GatewayConfigStore) *Gateway {
	Breaker := make(map[string]*circuitbreaker.CircuitBreaker)

	// for each user and their routes build per-upstream circuit breakers
	for _, routes := range store.Users {
		for _, route := range routes {
			servers := route.LoadBalancer.Servers()
			for i, server := range servers {
				if _, exists := Breaker[server]; !exists {
					// find corresponding upstream config
					var cfg configuration.CircuitBreakerConfig
					if i < len(route.Upstreams) {
						cfg = route.Upstreams[i].CircuitBreaker
					}
					// create circuit breaker using config
					Breaker[server] = circuitbreaker.New(cfg)
				}
			}
		}
	}

	return &Gateway{Store: store, Breaker: Breaker}
}

func (g *Gateway) Handler(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	userId := r.Header.Get("X-User-ID")
	if userId == "" {
		http.Error(w, "Missing X-User-ID header", http.StatusBadRequest)
		return
	}

	// match route
	route, err := g.Store.MatchPath(userId, r.URL.Path, r.Method)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// put route into context
	r = r.WithContext(context.WithValue(r.Context(), configuration.RouteCtxKey, route))

	// build middleware chain
	// Cache -> RateLimiter -> RetryHandler -> ProxyHandler
	// Cache and RateLimiter execute once per client request.
	// RetryHandler handles retries, re-picks upstream on each attempt, checks circuit breaker and calls ProxyHandler
	chain := g.wrapWithMiddlewares(proxy.ProxyHandler(g.Breaker))

	// run the chain
	chain.ServeHTTP(w, r)
	latencyMs := time.Since(startTime).Milliseconds()
	metrics.RecordLatency(latencyMs)
}

func (g *Gateway) wrapWithMiddlewares(final http.Handler) http.Handler {
	h := final // final is ProxyHandler

	h = middleware.RetryHandler(g.Breaker)(h)
	//h = middleware.RateLimiter(h)
	h = middleware.CacheMiddleware(g.Store)(h)

	return h
}
