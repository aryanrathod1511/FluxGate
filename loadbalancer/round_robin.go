package loadbalancer

import (
	"fmt"
	"sync/atomic"
)

type RoundRobin struct {
	servers []string
	indx    uint64
}

func init() {
	RegistrLoadBalancer("round_robin", func(servers []string) LoadBalancer {
		return NewRoundRobin(servers)
	})
}

func NewRoundRobin(servers []string) *RoundRobin {
	return &RoundRobin{servers: servers}
}

func (rr *RoundRobin) NextServer() (string, error) {
	if len(rr.servers) == 0 {
		return "", fmt.Errorf("no servers available")
	}

	i := atomic.AddUint64(&rr.indx, 1)
	return rr.servers[(i-1)%uint64(len(rr.servers))], nil
}
