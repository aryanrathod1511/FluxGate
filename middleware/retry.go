package middleware

import (
	"FluxGate/circuitbreaker"
	"FluxGate/configuration"
	"FluxGate/utils"
	"bytes"
	"context"
	"io"
	"math/rand"
	"net/http"
	"time"
)

type responseCapture struct {
	status int
	body   *bytes.Buffer
	header http.Header
}

type responseWriter struct {
	http.ResponseWriter
	capture *responseCapture
}

func RetryHandler(breakers map[string]*circuitbreaker.CircuitBreaker) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			routeVal := r.Context().Value(configuration.RouteCtxKey)
			if routeVal == nil {
				next.ServeHTTP(w, r)
				return
			}
			route := routeVal.(*configuration.RouteConfig)

			retryConfig := route.Retry
			if !retryConfig.Enabled || retryConfig.MaxTries <= 0 {
				upstream, err := utils.PickHealthyServer(route.LoadBalancer, breakers)
				if err != nil {
					http.Error(w, err.Error(), http.StatusServiceUnavailable)
					return
				}
				r = r.WithContext(context.WithValue(r.Context(), configuration.UpstreamCtxKey, upstream))
				next.ServeHTTP(w, r)
				return
			}

			bodyBytes, err := io.ReadAll(r.Body)
			if err != nil {
				http.Error(w, "Bad Request", http.StatusBadRequest)
				return
			}
			r.Body.Close()

			maxTries := retryConfig.MaxTries
			baseDelay := time.Duration(retryConfig.BaseTimeMs) * time.Millisecond

			for attempt := 0; attempt < maxTries; attempt++ {
				upstream, err := utils.PickHealthyServer(route.LoadBalancer, breakers)
				if err != nil {
					http.Error(w, err.Error(), http.StatusServiceUnavailable)
					return
				}

				r = r.WithContext(context.WithValue(r.Context(), configuration.UpstreamCtxKey, upstream))
				r.Body = io.NopCloser(bytes.NewReader(bodyBytes))

				capture := &responseCapture{
					status: 0,
					body:   &bytes.Buffer{},
					header: http.Header{},
				}

				rw := &responseWriter{
					ResponseWriter: w,
					capture:        capture,
				}

				// retry attempt logged removed
				next.ServeHTTP(rw, r)

				status := capture.status
				if status == 0 {
					status = http.StatusOK
				}

				utils.UpdateCircuitBreaker(breakers[upstream], status)

				if status < 500 {
					for k := range w.Header() {
						w.Header().Del(k)
					}
					for k, vals := range capture.header {
						for _, v := range vals {
							w.Header().Add(k, v)
						}
					}
					w.WriteHeader(status)
					w.Write(capture.body.Bytes())
					return
				}

				if attempt < maxTries-1 {
					jitter := time.Duration(rand.Int63n(int64(25 * time.Millisecond)))
					time.Sleep(baseDelay*(1<<attempt) + jitter)
				}
			}

			http.Error(w, "Bad Gateway", http.StatusBadGateway)
		})
	}
}

func (rw *responseWriter) Header() http.Header {
	return rw.capture.header
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	return rw.capture.body.Write(b)
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.capture.status = code
}
