package middleware

import (
	"FluxGate/configuration"
	metrics "FluxGate/matrics"
	"FluxGate/storage"
	"bytes"
	"net/http"
	"time"
)

func CacheMiddleware(store *configuration.GatewayConfigStore) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {

		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			route := r.Context().Value(configuration.RouteCtxKey).(*configuration.RouteConfig)

			cache := route.CacheInstance
			if cache == nil {
				next.ServeHTTP(w, r)
				return
			}

			key := r.Method + ":" + r.URL.Path
			if r.URL.RawQuery != "" {
				key += "?" + r.URL.RawQuery
			}

			// cache hit
			if entry, ok := cache.Get(key); ok {
				metrics.RecordCacheHit()
				for hk, vals := range entry.Header {
					for _, v := range vals {
						w.Header().Add(hk, v)
					}
				}
				w.WriteHeader(http.StatusOK)
				w.Write(entry.Body)
				return
			}

			metrics.RecordCacheMiss()

			// capture response
			rec := &responseRecorder{
				ResponseWriter: w,
				header:         http.Header{},
			}

			next.ServeHTTP(rec, r)

			// cache only 200 OK
			if rec.status == http.StatusOK {
				ttlDur := time.Duration(route.Cache.TTL) * time.Millisecond
				cache.Set(key, storage.CacheEntry{
					Body:       rec.body.Bytes(),
					Header:     rec.header.Clone(),
					ExpiryTime: time.Now().Add(ttlDur),
				})
			}
		})
	}
}

type responseRecorder struct {
	http.ResponseWriter
	status int
	body   bytes.Buffer
	header http.Header
}

func (r *responseRecorder) Header() http.Header {
	return r.header
}

func (r *responseRecorder) Write(b []byte) (int, error) {
	r.body.Write(b)
	return len(b), nil
}

func (r *responseRecorder) WriteHeader(code int) {
	r.status = code
}
