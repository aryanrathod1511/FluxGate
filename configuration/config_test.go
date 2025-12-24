package configuration

import (
	"encoding/json"
	"testing"
)

func TestMatchAndScore(t *testing.T) {
	tests := []struct {
		name      string
		pattern   string
		request   string
		wantMatch bool
		minScore  int
	}{
		{"exact match", "/api/users", "/api/users", true, 6},
		{"param segment", "/api/:id", "/api/123", true, 4},
		{"wildcard tail", "/api/*", "/api/users/123", true, 0},
		{"no match shorter request", "/api/users", "/api", false, 0},
		{"no match different segment", "/api/orders", "/api/users", false, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ok, score := matchAndScore(tt.pattern, tt.request)
			if ok != tt.wantMatch {
				t.Fatalf("matchAndScore(%q,%q) match=%v want %v", tt.pattern, tt.request, ok, tt.wantMatch)
			}
			if tt.wantMatch && score < tt.minScore {
				t.Fatalf("score=%d want >=%d", score, tt.minScore)
			}
		})
	}
}

func TestMatchPathPicksMostSpecific(t *testing.T) {
	store := NewGatewayConfigStore()
	cfg := []map[string]interface{}{
		{
			"path":           "/api/users",
			"method":         "GET",
			"load_balancing": "round_robin",
			"upstreams": []map[string]interface{}{
				{"url": "http://localhost:9001", "weight": 1, "circuit_breaker": map[string]interface{}{
					"enabled":            true,
					"failure_threshold":  5,
					"window_seconds":     60,
					"open_seconds":       5,
					"half_open_requests": 1,
					"success_threshold":  1,
				}},
			},
			"user_rate_limit":  map[string]interface{}{},
			"route_rate_limit": map[string]interface{}{},
			"user_id_key":      []string{"header:X-User-ID"},
			"cache": map[string]interface{}{
				"enabled":   false,
				"ttl_ms":    0,
				"max_entry": 0,
			},
			"retry": map[string]interface{}{
				"enabled":      false,
				"max_tries":    0,
				"base_time_ms": 0,
			},
		},
		{
			"path":           "/api/users/:id",
			"method":         "GET",
			"load_balancing": "round_robin",
			"upstreams": []map[string]interface{}{
				{"url": "http://localhost:9002", "weight": 1, "circuit_breaker": map[string]interface{}{
					"enabled":            true,
					"failure_threshold":  5,
					"window_seconds":     60,
					"open_seconds":       5,
					"half_open_requests": 1,
					"success_threshold":  1,
				}},
			},
			"user_rate_limit":  map[string]interface{}{},
			"route_rate_limit": map[string]interface{}{},
			"user_id_key":      []string{"header:X-User-ID"},
			"cache": map[string]interface{}{
				"enabled":   false,
				"ttl_ms":    0,
				"max_entry": 0,
			},
			"retry": map[string]interface{}{
				"enabled":      false,
				"max_tries":    0,
				"base_time_ms": 0,
			},
		},
	}

	data, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("marshal config: %v", err)
	}
	if err := store.LoadConfig("demo", data); err != nil {
		t.Fatalf("load config: %v", err)
	}

	route, err := store.MatchPath("demo", "/api/users/123", "GET")
	if err != nil {
		t.Fatalf("MatchPath returned error: %v", err)
	}
	if route.Path != "/api/users/:id" {
		t.Fatalf("expected param route, got %s", route.Path)
	}
}
