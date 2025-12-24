package gateway

import (
	"FluxGate/configuration"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

func TestGatewayCacheHit(t *testing.T) {
	var upstreamHits atomic.Int32
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upstreamHits.Add(1)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"ok":true}`))
	}))
	t.Cleanup(upstream.Close)

	store := configuration.NewGatewayConfigStore()
	routes := []map[string]interface{}{
		{
			"path":           "/cache",
			"method":         "GET",
			"load_balancing": "round_robin",
			"upstreams": []map[string]interface{}{
				{"url": upstream.URL, "weight": 1, "circuit_breaker": map[string]interface{}{
					"enabled":            true,
					"failure_threshold":  5,
					"window_seconds":     60,
					"open_seconds":       2,
					"half_open_requests": 1,
					"success_threshold":  1,
				}},
			},
			"user_rate_limit":  map[string]interface{}{},
			"route_rate_limit": map[string]interface{}{},
			"user_id_key":      []string{"header:X-User-ID"},
			"cache": map[string]interface{}{
				"enabled":   true,
				"ttl_ms":    500,
				"max_entry": 10,
			},
			"retry": map[string]interface{}{
				"enabled":      false,
				"max_tries":    0,
				"base_time_ms": 0,
			},
		},
	}

	data, _ := json.Marshal(routes)
	if err := store.LoadConfig("demo", data); err != nil {
		t.Fatalf("load config: %v", err)
	}

	gw := NewGateway(store)

	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodGet, "/cache", nil)
		req.Header.Set("X-User-ID", "demo")
		rr := httptest.NewRecorder()
		gw.Handler(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("request %d expected 200 got %d", i+1, rr.Code)
		}
	}

	if hits := upstreamHits.Load(); hits != 1 {
		t.Fatalf("expected 1 upstream hit due to caching, got %d", hits)
	}
}

func TestGatewayRetrySucceeds(t *testing.T) {
	var calls atomic.Int32
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cur := calls.Add(1)
		if cur == 1 {
			http.Error(w, "fail once", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"ok":true}`))
	}))
	t.Cleanup(upstream.Close)

	store := configuration.NewGatewayConfigStore()
	routes := []map[string]interface{}{
		{
			"path":           "/retry",
			"method":         "GET",
			"load_balancing": "round_robin",
			"upstreams": []map[string]interface{}{
				{"url": upstream.URL, "weight": 1, "circuit_breaker": map[string]interface{}{
					"enabled":            true,
					"failure_threshold":  5,
					"window_seconds":     60,
					"open_seconds":       2,
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
				"enabled":      true,
				"max_tries":    3,
				"base_time_ms": 1,
			},
		},
	}

	data, _ := json.Marshal(routes)
	if err := store.LoadConfig("demo", data); err != nil {
		t.Fatalf("load config: %v", err)
	}

	gw := NewGateway(store)

	req := httptest.NewRequest(http.MethodGet, "/retry", nil)
	req.Header.Set("X-User-ID", "demo")
	rr := httptest.NewRecorder()
	gw.Handler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 after retry, got %d", rr.Code)
	}
	if hits := calls.Load(); hits != 2 {
		t.Fatalf("expected 2 upstream calls (fail once then succeed), got %d", hits)
	}
}

func TestGatewayRetryExhaustsAndFails(t *testing.T) {
	var calls atomic.Int32
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		http.Error(w, "always fail", http.StatusInternalServerError)
	}))
	t.Cleanup(upstream.Close)

	store := configuration.NewGatewayConfigStore()
	routes := []map[string]interface{}{
		{
			"path":           "/retry-fail",
			"method":         "GET",
			"load_balancing": "round_robin",
			"upstreams": []map[string]interface{}{
				{"url": upstream.URL, "weight": 1, "circuit_breaker": map[string]interface{}{
					"enabled":            true,
					"failure_threshold":  5,
					"window_seconds":     60,
					"open_seconds":       2,
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
				"enabled":      true,
				"max_tries":    2,
				"base_time_ms": 1,
			},
		},
	}

	data, _ := json.Marshal(routes)
	if err := store.LoadConfig("demo", data); err != nil {
		t.Fatalf("load config: %v", err)
	}

	gw := NewGateway(store)

	req := httptest.NewRequest(http.MethodGet, "/retry-fail", nil)
	req.Header.Set("X-User-ID", "demo")
	rr := httptest.NewRecorder()
	gw.Handler(rr, req)

	if rr.Code != http.StatusBadGateway {
		t.Fatalf("expected 502 after exhausting retries, got %d", rr.Code)
	}
	if hits := calls.Load(); hits != 2 {
		t.Fatalf("expected exactly 2 upstream attempts, got %d", hits)
	}
}
