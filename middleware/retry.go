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
			// Extract route from context
			routeVal := r.Context().Value(configuration.RouteCtxKey)
			if routeVal == nil {
				next.ServeHTTP(w, r)
				return
			}
			route := routeVal.(*configuration.RouteConfig)

			var retryEnabled bool
			var maxTries int
			var baseDelay time.Duration
			if len(route.Upstreams) > 0 {
				firstUpstreamConfig := route.Upstreams[0]
				retryEnabled = firstUpstreamConfig.RetryEnabled
				maxTries = firstUpstreamConfig.Retries
				baseDelay = time.Duration(firstUpstreamConfig.BaseTimeMs) * time.Millisecond
			}

			// If no retry config or retries disabled, just call next handler once
			if !retryEnabled || maxTries <= 0 {
				next.ServeHTTP(w, r)
				return
			}

			// Read request body once
			bodyBytes, err := io.ReadAll(r.Body)
			if err != nil {
				http.Error(w, "Bad Request", http.StatusBadRequest)
				return
			}
			r.Body.Close()

			// Retry loop
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

				next.ServeHTTP(&responseWriter{ResponseWriter: w, capture: capture}, r)

				lastStatus := capture.status
				if lastStatus == 0 {
					lastStatus = http.StatusOK
				}

				utils.UpdateCircuitBreaker(breakers[upstream], lastStatus)

				if lastStatus < 500 {
					// Copy headers and body to actual response writer
					for k, vals := range capture.header {
						for _, v := range vals {
							w.Header().Add(k, v)
						}
					}
					w.WriteHeader(lastStatus)
					w.Write(capture.body.Bytes())
					return
				}

				if attempt < maxTries-1 {
					jitter := time.Duration(rand.Int63n(int64(25 * time.Millisecond)))
					delay := baseDelay*(1<<attempt) + jitter
					time.Sleep(delay)
				}
			}

			http.Error(w, "Bad Gateway", http.StatusBadGateway)
		})
	}
}

func (rw *responseWriter) Header() http.Header         { return rw.capture.header }
func (rw *responseWriter) Write(b []byte) (int, error) { return rw.capture.body.Write(b) }
func (rw *responseWriter) WriteHeader(code int)        { rw.capture.status = code }
