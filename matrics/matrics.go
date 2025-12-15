package metrics

import (
	"sync"
	"time"
)

var LatencyBuckets = []int64{
	1, 2, 3, 4, 5, 10, 25, 50, 75, 100, 200, 300, 400, 800, 1000, 1200, 1600,
}

type SecondMetrics struct {
	Second        int64
	TotalRequests int64
	CacheHits     int64
	CacheMisses   int64
	LatencyCounts []int64
}

func newSecondMetrics(sec int64) *SecondMetrics {
	return &SecondMetrics{
		Second:        sec,
		LatencyCounts: make([]int64, len(LatencyBuckets)),
	}
}

type FlushedMetrics struct {
	Second        int64   `json:"second"`
	P95LatencyMs  int64   `json:"p95_latency_ms"`
	CacheHitRatio float64 `json:"cache_hit_ratio"`
	TotalRequests int64   `json:"total_requests"`
}

var (
	mu      sync.Mutex
	current = newSecondMetrics(time.Now().Unix())
)
