package loadbalancer

import (
	"fmt"
	"sync"
)

type WeightedRoundRobin struct {
	servers       []string
	weights       []int
	currentWeight []int
	Indx          int
	mu            sync.Mutex
}

func NewWeightedRoundRobin(servers []string, weights []int) *WeightedRoundRobin {
	return &WeightedRoundRobin{
		servers:       servers,
		weights:       weights,
		currentWeight: make([]int, len(servers)),
		Indx:          0,
		mu:            sync.Mutex{},
	}
}

func (wrr *WeightedRoundRobin) NextServer() (string, error) {
	if len(wrr.servers) == 0 || len(wrr.weights) == 0 || len(wrr.servers) != len(wrr.weights) {
		return "", fmt.Errorf("no servers available or weights mismatch")
	}
	wrr.mu.Lock()
	defer wrr.mu.Unlock()

	totalWeight := 0
	for _, weight := range wrr.weights {
		totalWeight += int(weight)
	}

	//select next server
	maxWtIndx := -1
	maxWt := -1
	for i := 0; i < len(wrr.servers); i++ {
		wrr.currentWeight[i] += wrr.weights[i]
		if wrr.currentWeight[i] >= maxWt {
			maxWt = wrr.currentWeight[i]
			maxWtIndx = i
		}
	}
	wrr.currentWeight[maxWtIndx] -= totalWeight
	return wrr.servers[maxWtIndx], nil
}

func (wrr *WeightedRoundRobin) Servers() []string {
	return wrr.servers
}
