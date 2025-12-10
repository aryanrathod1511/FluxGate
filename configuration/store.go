package configuration

import (
	"FluxGate/loadbalancer"
	"encoding/json"
	"fmt"
	"strings"
)

// constructor
func NewGatewayConfigStore() *GatewayConfigStore {
	return &GatewayConfigStore{
		users: make(map[string][]*RouteConfig),
	}
}

// methods
func (store *GatewayConfigStore) GetConfig(userId string) ([]byte, error) {
	store.mu.RLock()
	routes, ok := store.users[userId]
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

	// spin up loadbalancer instance for each route

	assignLoadBalancer(routes)

	store.users[userId] = routes
	return nil
}

func (store *GatewayConfigStore) DeleteConfig(userId string) {
	store.mu.Lock()
	defer store.mu.Unlock()

	delete(store.users, userId)
}

func (store *GatewayConfigStore) UpdateConfig(userId string, configData []byte) error {

	var routes []*RouteConfig
	err := json.Unmarshal(configData, &routes)

	if err != nil {
		return err
	}

	assignLoadBalancer(routes)

	store.mu.Lock()
	store.users[userId] = routes
	store.mu.Unlock()
	return nil
}

func (store *GatewayConfigStore) MatchPath(userId string, path string, method string) (*RouteConfig, error) {
	store.mu.RLock()
	defer store.mu.RUnlock()

	routes, ok := store.users[userId]
	if !ok {
		return nil, fmt.Errorf("no user found")
	}

	//implement tighter prefix matching later
	for _, route := range routes {
		if strings.HasPrefix(path, route.Path) && route.Method == method {
			return route, nil
		}
	}
	return nil, fmt.Errorf("no matching route found")
}

// utils
func assignLoadBalancer(routes []*RouteConfig) {
	for _, route := range routes {
		lbType := route.LoadBalancing
		switch lbType {
		case "round_robin":
			route.LoadBalancer = loadbalancer.NewRoundRobin(getUpstreamURLs(route.Upstreams))
		case "weighted_round_robin":
			route.LoadBalancer = loadbalancer.NewWeightedRoundRobin(
				getUpstreamURLs(route.Upstreams),
				getUpstreamWeights(route.Upstreams),
			)
		default:
			route.LoadBalancer = loadbalancer.NewRoundRobin(getUpstreamURLs(route.Upstreams))
		}
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
