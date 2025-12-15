package metrics

func RecordLatency(ms int64) {
	mu.Lock()
	defer mu.Unlock()

	current.TotalRequests++

	for i, bound := range LatencyBuckets {
		if ms <= bound {
			current.LatencyCounts[i]++
			return
		}
	}

	// overflow
	current.LatencyCounts[len(current.LatencyCounts)-1]++
}

func RecordCacheHit() {
	mu.Lock()
	current.CacheHits++
	mu.Unlock()
}

func RecordCacheMiss() {
	mu.Lock()
	current.CacheMisses++
	mu.Unlock()
}
