package proxy

import (
	"bytes"            
	"context"
	"io"
	"math/rand"        
	"net"
	"net/http"
	"time"
	"FluxGate/configuration"
)

var transport = &http.Transport{
	Proxy:                 http.ProxyFromEnvironment,
	MaxIdleConns:          100,
	IdleConnTimeout:       90 * time.Second,
	TLSHandshakeTimeout:   10 * time.Second,
	ExpectContinueTimeout: 1 * time.Second,
}

var httpClient = &http.Client{
	Transport: transport,
	Timeout:   0, // context controls timeout
}

// ReverseProxy forwards the request to the chosen upstream URL
func ReverseProxy(
	w http.ResponseWriter,
	r *http.Request,
	upstreamURL string,
	timeout time.Duration,
) {

	// read body
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}
	r.Body.Close()

	
	route := r.Context().Value(configuration.RouteCtxKey).(*configuration.RouteConfig)

	// find upstream index
	indx := -1
	for i, ups := range route.Upstreams {
		if ups.URL == upstreamURL {
			indx = i
			break
		}
	}
	if indx == -1 {
		http.Error(w, "Bad Gateway", http.StatusBadGateway)
		return
	}

	maxTries := route.Upstreams[indx].Retries
	baseDelay := time.Duration(route.Upstreams[indx].BaseTimeMs) * time.Millisecond

	var resp *http.Response

	for i := 0; i < maxTries; i++ {

		ctx, cancel := context.WithTimeout(r.Context(), timeout)

		// recreate request
		req, err := http.NewRequestWithContext(
			ctx,
			r.Method,
			upstreamURL,
			io.NopCloser(bytes.NewReader(bodyBytes)),
		)
		if err != nil {
			cancel()
			http.Error(w, "Bad Gateway", http.StatusBadGateway)
			return
		}

		// copy headers
		for k, v := range r.Header {
			for _, vv := range v {
				req.Header.Add(k, vv)
			}
		}

		// forwarding headers
		host, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			host = r.RemoteAddr
		}
		req.Header.Set("X-Forwarded-For", host)
		req.Header.Set("X-Forwarded-Host", r.Host)
		req.Header.Set("X-Forwarded-Proto", r.URL.Scheme)

		resp, err = httpClient.Do(req)
		cancel()

		// success
		if err == nil && resp.StatusCode < 500 {
			defer resp.Body.Close()
			writeResponse(w, resp)
			return
		}

		
		if resp != nil {
			resp.Body.Close()
		}

		if i == maxTries-1 {
			break
		}

		// exponential backoff + jitter
		jitter := time.Duration(rand.Int63n(int64(25 * time.Millisecond)))
		time.Sleep(baseDelay*(1<<i) + jitter)
	}

	http.Error(w, "Bad Gateway", http.StatusBadGateway)
}

func writeResponse(w http.ResponseWriter, resp *http.Response) {
	for k, vals := range resp.Header {
		for _, v := range vals {
			w.Header().Add(k, v)
		}
	}
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}
