package utils

import (
	"FluxGate/circuitbreaker"
	"FluxGate/loadbalancer"
	"fmt"
	"log"
)

func PickHealthyServer(lb loadbalancer.LoadBalancer, breakers map[string]*circuitbreaker.CircuitBreaker) (string, error) {

	serversSeen := 0

	servers := lb.Servers()
	if len(servers) == 0 {
		return "", fmt.Errorf("no upstream servers configured")
	}

	for {
		server, err := lb.NextServer()
		if err != nil {
			return "", err
		}

		log.Printf("[pick] trying server %s", server)

		cb := breakers[server] // string key = server URL

		if cb == nil || cb.Allow() {
			if cb != nil {
				log.Printf("[pick] server %s allowed by circuit breaker", server)
			}
			// allowed upstream
			return server, nil
		}

		// skip blocked server, try another
		serversSeen++
		if serversSeen >= len(servers) {
			return "", fmt.Errorf("no healthy upstreams (all circuits open)")
		}
	}
}
