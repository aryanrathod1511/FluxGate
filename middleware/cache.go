package middleware

import (
	"FluxGate/configuration"
	"FluxGate/gateway"
	"FluxGate/storage"
	"bytes"
	"net/http"
	"time"
)

func CacheMiddleware(store *configuration.GatewayConfigStore) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {

		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			// extract route from context
			route := r.Context().Value(gateway.RouteCtxKey).(*configuration.RouteConfig)

			cache := route.CacheInstance
			if cache == nil {
				next.ServeHTTP(w, r)
				return
			}

			key := r.Method + ":" + r.URL.Path + "?" + r.URL.RawQuery

			// check cache
			if entry, ok := cache.Get(key); ok {
				for hk, vals := range entry.Header {
					for _, v := range vals {
						w.Header().Add(hk, v)
					}
				}
				w.Write(entry.Body)
				return
			}

			// capture response
			rec := &responseRecorder{
				ResponseWriter: w,
				header:         http.Header{},
			}

			next.ServeHTTP(rec, r)

			// only cache 200 responses
			if rec.status == http.StatusOK {
				cache.Set(key, storage.CacheEntry{
					Body:       rec.body.Bytes(),
					Header:     rec.header.Clone(),
					ExpiryTime: time.Now().Add(route.Cache.TTL),
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
