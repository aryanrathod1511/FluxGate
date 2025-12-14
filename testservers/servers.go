package testservers

import (
	"encoding/json"
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"time"
)

// StartAll starts several test servers and returns a map[name]url
func StartAll() map[string]string {
	rand.Seed(time.Now().UnixNano())

	results := map[string]string{}
	results["fast"] = startServer(9001, fastHandler("fast"))
	results["slow"] = startServer(9002, slowHandler("slow", 800*time.Millisecond))
	results["faulty30"] = startServer(9003, faultyHandler("faulty30", 0.3))
	results["faulty20"] = startServer(9004, faultyHandler("faulty20", 0.2))
	results["echo"] = startServer(9005, echoHandler("echo"))

	return results
}

func startServer(port int, handler http.Handler) string {
	addr := ":" + strconv.Itoa(port)
	srv := &http.Server{
		Addr:    addr,
		Handler: handler,
	}
	go func() {
		log.Printf("testserver listening on %s", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("testserver %s failed: %v", addr, err)
		}
	}()
	return "http://localhost:" + strconv.Itoa(port)
}

type srvResp struct {
	Server  string              `json:"server"`
	Time    time.Time           `json:"time"`
	Method  string              `json:"method"`
	Path    string              `json:"path"`
	Query   map[string][]string `json:"query"`
	Headers map[string][]string `json:"headers,omitempty"`
	Body    interface{}         `json:"body,omitempty"`
	Note    string              `json:"note,omitempty"`
}

func fastHandler(name string) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		resp := srvResp{Server: name, Time: time.Now().UTC(), Method: r.Method, Path: r.URL.Path, Query: r.URL.Query(), Headers: r.Header}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "public, max-age=60")
		_ = json.NewEncoder(w).Encode(resp)
	})
	return mux
}

func slowHandler(name string, delay time.Duration) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(delay)
		resp := srvResp{Server: name, Time: time.Now().UTC(), Method: r.Method, Path: r.URL.Path, Query: r.URL.Query(), Headers: r.Header, Note: "delayed"}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "public, max-age=30")
		_ = json.NewEncoder(w).Encode(resp)
	})
	return mux
}

func faultyHandler(name string, failRatio float64) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if rand.Float64() < failRatio {
			http.Error(w, "internal server error (simulated)", http.StatusInternalServerError)
			return
		}
		resp := srvResp{Server: name, Time: time.Now().UTC(), Method: r.Method, Path: r.URL.Path, Query: r.URL.Query(), Headers: r.Header}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "public, max-age=10")
		_ = json.NewEncoder(w).Encode(resp)
	})
	return mux
}

func echoHandler(name string) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		var body interface{}
		if r.Method == http.MethodPost || r.Method == http.MethodPut {
			dec := json.NewDecoder(r.Body)
			if err := dec.Decode(&body); err != nil {
				body = nil
			}
		}
		resp := srvResp{Server: name, Time: time.Now().UTC(), Method: r.Method, Path: r.URL.Path, Query: r.URL.Query(), Headers: r.Header, Body: body}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "public, max-age=20")
		_ = json.NewEncoder(w).Encode(resp)
	})
	return mux
}
