package proxy

import (
	"context"
	"io"
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
	Timeout:   0,
}

// ReverseProxy forwards the request to the chosen upstream URL
func ReverseProxy(w http.ResponseWriter, r *http.Request, upstreamURL string, timeout time.Duration) {

	// set timeout
	ctx, cancel := context.WithTimeout(r.Context(), timeout)
	defer cancel()

	// clone request
	req, err := http.NewRequestWithContext(ctx, r.Method, upstreamURL, r.Body)
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

	// add forwarding headers
	host, _, _ := net.SplitHostPort(r.RemoteAddr)
	req.Header.Set("X-Forwarded-For", host)
	req.Header.Set("X-Forwarded-Host", r.Host)
	req.Header.Set("X-Forwarded-Proto", r.URL.Scheme)

	// send upstream request
	resp, err := httpClient.Do(req)
	if err != nil {

		// timeout
		if ctx.Err() == context.DeadlineExceeded {
			http.Error(w, "Gateway Timeout", http.StatusGatewayTimeout)
			return
		}

		http.Error(w, "Bad Gateway", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	// copy headers
	for k, vals := range resp.Header {
		for _, v := range vals {
			w.Header().Add(k, v)
		}
	}

	// copy status
	w.WriteHeader(resp.StatusCode)

	// stream body
	io.Copy(w, resp.Body)
}
