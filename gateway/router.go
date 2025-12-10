package gateway

import (
	"FluxGate/configuration"
	"FluxGate/loadbalancer"
	"context"
	"net/http"
)

type Gateway struct {
	Store *configuration.GatewayConfigStore
	LBs   map[string]loadbalancer.LoadBalancer
}

func NewGateway(store *configuration.GatewayConfigStore, lbs map[string]loadbalancer.LoadBalancer) *Gateway {
	return &Gateway{
		Store: store,
		LBs:   lbs,
	}
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

	upstream, err := route.LoadBalancer.NextServer()
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}

	// Store route in context
	r = r.WithContext(
		context.WithValue(r.Context(), RouteCtxKey, route),
	)

	// Continue to middleware chain
	//g.Proxy.Forward(upstream, w, r)
}
