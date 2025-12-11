package gateway

import (
	"FluxGate/circuitbreaker"
	"FluxGate/configuration"
	"FluxGate/middleware"
	"FluxGate/utils"
	"context"
	"net/http"
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

// Define a typed key
type ctxKey string

const RouteCtxKey ctxKey = "route"

func (g *Gateway) Handler(w http.ResponseWriter, r *http.Request) {
	userId := r.Header.Get("X-User-ID")
	if userId == "" {
		http.Error(w, "Missing X-User-ID header", http.StatusBadRequest)
		return
	}

	route, err := g.Store.MatchPath(userId, r.URL.Path, r.Method)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	upstream, err := utils.PickHealthyServer(route.LoadBalancer, g.Breaker)
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}

	// Store route in context
	r = r.WithContext(
		context.WithValue(r.Context(), RouteCtxKey, route),
	)

	// attach selected upstream into context so middlewares can act on it
	r = r.WithContext(
		context.WithValue(r.Context(), middleware.UpstreamCtxKey, upstream))

	// Continue to middleware chain and then proxy

}
