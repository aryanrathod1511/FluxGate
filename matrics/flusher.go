package metrics

import (
	"encoding/json"
	"math"
	"os"
	"time"
)

func StartFlusher(path string) {
	file, _ := os.Create(path)

	go func() {
		ticker := time.NewTicker(100 * time.Millisecond)
		for range ticker.C {
			m := flush()
			if m != nil {
				_ = json.NewEncoder(file).Encode(m)
			}
		}
	}()
}

func flush() *FlushedMetrics {
	mu.Lock()
	old := current
	current = newSecondMetrics(time.Now().Unix())
	mu.Unlock()

	if old.TotalRequests == 0 {
		return nil
	}

	// p95 calculation
	target := int64(math.Ceil(0.95 * float64(old.TotalRequests)))
	var cum int64
	var p95 int64

	for i, c := range old.LatencyCounts {
		cum += c
		if cum >= target {
			p95 = LatencyBuckets[i]
			break
		}
	}

	hitRatio := 0.0
	totalCache := old.CacheHits + old.CacheMisses
	if totalCache > 0 {
		hitRatio = float64(old.CacheHits) / float64(totalCache)
	}

	return &FlushedMetrics{
		Second:        old.Second,
		P95LatencyMs:  p95,
		CacheHitRatio: hitRatio,
		TotalRequests: old.TotalRequests,
	}
}
