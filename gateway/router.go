package gateway

import (
	"FluxGate/circuitbreaker"
	"FluxGate/configuration"
	"FluxGate/middleware"
	"FluxGate/proxy"
	"FluxGate/utils"
	"context"
	"log"
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

	// pick upstream
	log.Printf("[gateway] incoming: %s %s for user=%s; matched route=%s", r.Method, r.URL.Path, userId, route.Path)
	upstream, err := utils.PickHealthyServer(route.LoadBalancer, g.Breaker)
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	log.Printf("[gateway] selected upstream %s for route %s", upstream, route.Path)

	// put route + upstream into context
	r = r.WithContext(context.WithValue(r.Context(), configuration.RouteCtxKey, route))
	r = r.WithContext(context.WithValue(r.Context(), middleware.UpstreamCtxKey, upstream))

	// build middleware chain
	chain := g.wrapWithMiddlewares(proxyHandler)

	// run the chain
	chain.ServeHTTP(w, r)
}

func (g *Gateway) wrapWithMiddlewares(final http.Handler) http.Handler {
	h := final
	h = middleware.CacheMiddleware(g.Store)(h)
	h = middleware.RateLimiter(h)
	h = middleware.CircuitBreakerMiddleware(g.Breaker)(h)

	return h
}

var proxyHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	upstream := r.Context().Value(middleware.UpstreamCtxKey).(string)

	timeout := 5 * time.Second
	proxy.ReverseProxy(w, r, upstream, timeout,g.Breaker[upstream])
})
