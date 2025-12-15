package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"FluxGate/configuration"
	"FluxGate/gateway"
	metrics "FluxGate/matrics"
	"FluxGate/testservers"
)

func mergeMaps(a, b map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	for k, v := range a {
		result[k] = v
	}
	for k, v := range b {
		result[k] = v
	}
	return result
}

func main() {
	// start test upstreams
	ups := testservers.StartAll()
	for k, v := range ups {
		log.Printf("started upstream %s -> %s", k, v)
	}

	// common rate limiting configuration for all routes
	commonRateLimit := map[string]interface{}{
		"route_rate_limit": map[string]interface{}{
			"type":        "token_bucket",
			"capacity":    100.0,
			"refill_rate": 10.0,
		},
		"user_rate_limit": map[string]interface{}{
			"type":        "token_bucket",
			"capacity":    20.0,
			"refill_rate": 2.0,
		},
		"user_id_key": []string{"header:X-User-ID", "ip"},
	}

	routes := []map[string]interface{}{
		// fast route has two upstreams to demonstrate LB behavior
		mergeMaps(map[string]interface{}{
			"path":           "/fast",
			"method":         "GET",
			"load_balancing": "round_robin",
			"upstreams": []map[string]interface{}{
				{"url": ups["fast"], "weight": 2},
				{"url": ups["slow"], "weight": 1},
			},
			"cache": map[string]interface{}{"enabled": true, "ttl_ms": 60000, "max_entry": 100},
			"retry": map[string]interface{}{
				"enabled":      true,
				"max_tries":    3,
				"base_time_ms": 100,
			},
		}, commonRateLimit),
		// slow-only
		mergeMaps(map[string]interface{}{
			"path":           "/slow",
			"method":         "GET",
			"load_balancing": "round_robin",
			"upstreams": []map[string]interface{}{
				{"url": ups["slow"], "weight": 1},
			},
			"cache": map[string]interface{}{"enabled": true, "ttl_ms": 60000, "max_entry": 50},
			"retry": map[string]interface{}{
				"enabled":      false,
				"max_tries":    0,
				"base_time_ms": 0,
			},
		}, commonRateLimit),
		// faulty has two upstreams
		mergeMaps(map[string]interface{}{
			"path":           "/faulty",
			"method":         "GET",
			"load_balancing": "round_robin",
			"upstreams": []map[string]interface{}{
				{"url": ups["faulty30"], "weight": 1},
				{"url": ups["faulty20"], "weight": 1},
			},
			"cache": map[string]interface{}{"enabled": false},
			"retry": map[string]interface{}{
				"enabled":      true,
				"max_tries":    5,
				"base_time_ms": 200,
			},
		}, commonRateLimit),
		// echo can use echo and fast upstreams (multi-method)
		mergeMaps(map[string]interface{}{
			"path":           "/echo",
			"method":         "GET",
			"load_balancing": "round_robin",
			"upstreams": []map[string]interface{}{
				{"url": ups["echo"], "weight": 2},
				{"url": ups["fast"], "weight": 1},
			},
			"cache": map[string]interface{}{"enabled": true, "ttl_ms": 30000, "max_entry": 200},
			"retry": map[string]interface{}{
				"enabled":      true,
				"max_tries":    2,
				"base_time_ms": 50,
			},
		}, commonRateLimit),
		mergeMaps(map[string]interface{}{
			"path":           "/echo",
			"method":         "POST",
			"load_balancing": "round_robin",
			"upstreams": []map[string]interface{}{
				{"url": ups["echo"], "weight": 1},
			},
			"cache": map[string]interface{}{"enabled": true, "ttl_ms": 30000, "max_entry": 200},
			"retry": map[string]interface{}{
				"enabled":      true,
				"max_tries":    3,
				"base_time_ms": 100,
			},
		}, commonRateLimit),
	}

	data, _ := json.Marshal(routes)

	store := configuration.NewGatewayConfigStore()
	if err := store.LoadConfig("demo", data); err != nil {
		log.Fatalf("failed to load demo config: %v", err)
	}

	gw := gateway.NewGateway(store)
	metrics.StartFlusher("bench_metrics.jsonl")

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("ok")) })
	http.HandleFunc("/", gw.Handler)

	srv := &http.Server{Addr: ":8080", ReadTimeout: 10 * time.Second, WriteTimeout: 10 * time.Second}
	fmt.Println("Gateway demo running on :8080 — use X-User-ID: demo header")
	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("gateway failed: %v", err)
	}
}
