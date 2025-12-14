package proxy

import (
	"bytes"
	"context"
	"io"
	"log"
	"net"
	"net/http"
	"time"
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

	ctx, cancel := context.WithTimeout(r.Context(), timeout)

	// recreate request
	req, err := http.NewRequestWithContext(
		ctx,
		r.Method,
		upstreamURL,
		io.NopCloser(bytes.NewReader(bodyBytes)),
	)
	if err != nil {
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

	log.Printf("[proxy] forwarding %s %s -> %s", r.Method, r.URL.RequestURI(), upstreamURL)
	resp, err := httpClient.Do(req)
	cancel()
	if err != nil {
		// Network/connection error - will be retried by RetryHandler if appropriate
		log.Printf("[proxy] request to upstream %s failed: %v", upstreamURL, err)
		http.Error(w, "Bad Gateway", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	log.Printf("[proxy] upstream %s responded %d", upstreamURL, resp.StatusCode)

	// Write response regardless of status code
	// RetryHandler will decide whether to retry based on status code
	writeResponse(w, resp)
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
