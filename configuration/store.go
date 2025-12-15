package configuration

import (
	"FluxGate/loadbalancer"
	"FluxGate/ratelimit"
	"FluxGate/storage"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"
)

// constructor
func NewGatewayConfigStore() *GatewayConfigStore {
	return &GatewayConfigStore{
		Users: make(map[string][]*RouteConfig),
	}
}

// methods
func (store *GatewayConfigStore) GetConfig(userId string) ([]byte, error) {
	store.mu.RLock()
	routes, ok := store.Users[userId]
	store.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("no config found for user: %s", userId)
	}

	data, err := json.Marshal(routes)
	if err != nil {
		return nil, err
	}

	return data, nil
}

func (store *GatewayConfigStore) LoadConfig(userId string, configData []byte) error {
	store.mu.Lock()
	defer store.mu.Unlock()

	var routes []*RouteConfig

	err := json.Unmarshal(configData, &routes)
	if err != nil {
		return err
	}

	// spin up loadbalancer, ratelimiter and LRU cache instances for each route

	assignLoadBalancer(routes)
	assignRateLimiter(routes)
	assignCacheInstances(routes)

	store.Users[userId] = routes
	log.Printf("Loaded %d routes for user: %s", len(routes), userId)
	return nil
}

func (store *GatewayConfigStore) DeleteConfig(userId string) {
	store.mu.Lock()
	defer store.mu.Unlock()

	delete(store.Users, userId)
}

func (store *GatewayConfigStore) UpdateConfig(userId string, configData []byte) error {

	var routes []*RouteConfig
	err := json.Unmarshal(configData, &routes)

	if err != nil {
		return err
	}

	assignLoadBalancer(routes)
	assignRateLimiter(routes)
	assignCacheInstances(routes)

	store.mu.Lock()
	store.Users[userId] = routes
	store.mu.Unlock()
	return nil
}

func (store *GatewayConfigStore) MatchPath(userId string, path string, method string) (*RouteConfig, error) {
	store.mu.RLock()
	defer store.mu.RUnlock()

	routes, ok := store.Users[userId]
	if !ok {
		return nil, fmt.Errorf("no user found")
	}

	// Normalize incoming path: strip query and fragment (if any), ensure leading '/'
	reqPath := path
	if idx := strings.IndexAny(reqPath, "?#"); idx >= 0 {
		reqPath = reqPath[:idx]
	}
	if reqPath == "" {
		reqPath = "/"
	}
	// remove duplicate slashes and trailing slash (except root)
	reqPath = "/" + strings.Trim(strings.ReplaceAll(reqPath, "//", "/"), "/")

	// Pick the most specific matching route. Score higher for exact segment matches.
	var best *RouteConfig
	bestScore := -1

	for _, route := range routes {
		if route.Method != method {
			continue
		}

		// Normalize pattern from config similarly
		pattern := route.Path
		if idx := strings.IndexAny(pattern, "?#"); idx >= 0 {
			pattern = pattern[:idx]
		}
		if pattern == "" {
			pattern = "/"
		}
		pattern = "/" + strings.Trim(strings.ReplaceAll(pattern, "//", "/"), "/")

		okMatch, score := matchAndScore(pattern, reqPath)
		if okMatch {
			if score > bestScore {
				best = route
				bestScore = score
			}
		}
	}

	if best != nil {
		return best, nil
	}
	return nil, fmt.Errorf("no matching route found")
}

func matchAndScore(pattern, req string) (bool, int) {
	// trim leading slash for splitting
	p := strings.Trim(pattern, "/")
	r := strings.Trim(req, "/")

	if p == "" && r == "" {
		return true, 100 // root exact
	}

	pSegs := []string{}
	if p != "" {
		pSegs = strings.Split(p, "/")
	}
	rSegs := []string{}
	if r != "" {
		rSegs = strings.Split(r, "/")
	}

	score := 0
	i := 0
	for i < len(pSegs) {
		ps := pSegs[i]
		// wildcard at end
		if ps == "*" && i == len(pSegs)-1 {
			// matches any remainder
			score += 0
			return true, score
		}

		if i >= len(rSegs) {
			// pattern longer than request and not a trailing wildcard
			return false, 0
		}

		rs := rSegs[i]

		// parameter segments
		if strings.HasPrefix(ps, ":") || (strings.HasPrefix(ps, "{") && strings.HasSuffix(ps, "}")) {
			// matches any single segment, small score
			score += 1
		} else if ps == rs {
			// exact match - higher score
			score += 3
		} else {
			// no match
			return false, 0
		}

		i++
	}

	// if pattern consumed but request has extra segments, no match unless pattern ended with wildcard
	if len(rSegs) > len(pSegs) {
		return false, 0
	}

	return true, score
}

// utils
func assignLoadBalancer(routes []*RouteConfig) {
	for _, route := range routes {
		route.LoadBalancer = loadbalancer.New(route.LoadBalance, getUpstreamURLs(route.Upstreams), getUpstreamWeights(route.Upstreams))
	}
}

func getUpstreamURLs(upstreams []UpstreamConfig) []string {
	var urls []string
	for _, upstream := range upstreams {
		urls = append(urls, upstream.URL)
	}
	return urls
}

func getUpstreamWeights(upstreams []UpstreamConfig) []int {
	var weights []int
	for _, upstream := range upstreams {
		weights = append(weights, upstream.Weight)
	}
	return weights
}

func assignRateLimiter(routes []*RouteConfig) {
	for _, route := range routes {
		// ROUTE-LEVEL rate limiter
		if route.RouteRateLimit.Type != "" && route.RouteRateLimit.Type != "none" {
			route.RouteRateLimiter = ratelimit.New(
				route.RouteRateLimit.Type,
				route.RouteRateLimit.Capacity,
				route.RouteRateLimit.RefillRate,
			)
			if route.RouteRateLimiter == nil {
				panic(fmt.Sprintf("failed to create route rate limiter for type '%s' on route %s %s",
					route.RouteRateLimit.Type, route.Method, route.Path))
			}
		}

		// USER-LEVEL rate limiters - initialize map for per-user instances
		// Individual limiters are created on-demand in middleware
		route.UserRateLimiter = sync.Map{}
	}
}

func assignCacheInstances(routes []*RouteConfig) {
	for _, route := range routes {
		if route.Cache.Enabled {
			route.CacheInstance = storage.NewLRUCache(
				route.Cache.MaxEntry,
				time.Duration(route.Cache.TTL)*time.Millisecond,
			)
		}
	}
}
