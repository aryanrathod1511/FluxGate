package storage

import (
	"net/http"
	"testing"
	"time"
)

func TestLRUCacheTTLAndEviction(t *testing.T) {
	cache := NewLRUCache(1, 50*time.Millisecond)

	entry := CacheEntry{
		Body:       []byte("ok"),
		Header:     http.Header{"X-Test": []string{"v"}},
		ExpiryTime: time.Now().Add(50 * time.Millisecond),
	}
	cache.Set("k1", entry)

	if _, ok := cache.Get("k1"); !ok {
		t.Fatalf("expected cache hit before expiry")
	}

	time.Sleep(60 * time.Millisecond)
	if _, ok := cache.Get("k1"); ok {
		t.Fatalf("expected cache miss after expiry")
	}

	// Test eviction with capacity 1
	entry.ExpiryTime = time.Now().Add(time.Second)
	cache.Set("k1", entry)
	cache.Set("k2", entry)
	if _, ok := cache.Get("k1"); ok {
		t.Fatalf("expected k1 to be evicted when capacity exceeded")
	}
	if _, ok := cache.Get("k2"); !ok {
		t.Fatalf("expected k2 to remain present")
	}
}
