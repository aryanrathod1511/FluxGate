package loadbalancer

import (
	"fmt"
	"sync"
)

type WeightedRoundRobin struct {
	Servers       []string
	Weights       []int
	CurrentWeight []int
	Indx          int
	mu            sync.Mutex
}

func NewWeightedRoundRobin(servers []string, weights []int) *WeightedRoundRobin {
	return &WeightedRoundRobin{
		Servers:       servers,
		Weights:       weights,
		CurrentWeight: make([]int, len(servers)),
		Indx:          0,
		mu:            sync.Mutex{},
	}
}

func (wrr *WeightedRoundRobin) NextServer() (string, error) {
	if len(wrr.Servers) == 0 || len(wrr.Weights) == 0 || len(wrr.Servers) != len(wrr.Weights) {
		return "", fmt.Errorf("no servers available or weights mismatch")
	}
	wrr.mu.Lock()
	defer wrr.mu.Unlock()

	totalWeight := 0
	for _, weight := range wrr.Weights {
		totalWeight += int(weight)
	}

	//select next server
	maxWtIndx := -1
	maxWt := -1
	for i := 0; i < len(wrr.Servers); i++ {
		wrr.CurrentWeight[i] += wrr.Weights[i]
		if wrr.CurrentWeight[i] >= maxWt {
			maxWt = wrr.CurrentWeight[i]
			maxWtIndx = i
		}
	}
	wrr.CurrentWeight[maxWtIndx] -= totalWeight
	return wrr.Servers[maxWtIndx], nil
}
