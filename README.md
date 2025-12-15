# FluxGate - High-Performance API Gateway

FluxGate is a **high-performance, production-inspired API Gateway** built in Go, designed to handle real-world traffic patterns and failure modes—not just happy paths.

It focuses on **core gateway responsibilities**:
- Routing
- Load balancing
- Rate limiting
- Circuit breaking
- Response caching
- Retries
- Latency and cache metrics

The goal isn’t to wrap an existing proxy or ship a full production product.  
The goal is to **understand and demonstrate real system trade-offs**—latency vs throughput, cache warm-up, tail behavior (p99), retries under load, and how gateways actually behave under concurrency.

FluxGate is opinionated, measurable, and intentionally transparent about what works, what breaks, and why.


---

## 🌟 Key Features

### 🔀 Routing
- **Path-based routing** with HTTP method matching
- Supports **parameters** (e.g. `:id`, `{id}`) and **wildcards** (`*`)
- **Route scoring** to select the most specific match
- Per-user / per-tenant route configuration (e.g. `demo` user)

### ⚖️ Load Balancing
- **Round-robin** load balancer
- **Weighted round-robin** implementation available
- Per-route load balancer instances
- Integrates with circuit breakers to avoid unhealthy upstreams

### 🚦 Rate Limiting
- **Token bucket** algorithm
- **Route-level** and **user-level** limits
- Flexible user identification via:
  - Headers
  - Query params
  - Cookies
  - Form fields
  - Basic auth
  - JWT token
  - IP address
- Registry-based design for adding new limiter types

> Note: In the demo setup, configuration is constructed in-memory but the design supports JSON-based configuration.

### 🔌 Circuit Breaker
- Three-state model: **Closed → Open → Half-Open**
- Configurable:
  - Failure threshold
  - Sliding time window
  - Open duration
  - Half‑open trial limit
  - Success threshold for recovery
- Per-upstream breaker instances keyed by upstream URL

### 💾 Response Caching
- In-memory **LRU cache** per route
- Configurable TTL and maximum entries
- Cache key built from **HTTP method + path + query**
- Only **HTTP 200** responses are cached
- Exposes cache warm-up and stampede behavior under load

### 🔁 Resilient Retries
- Middleware-driven **retry handler** with exponential backoff + jitter
- Retries on **5xx** or network failures
- Per-route configuration:
  - Enabled/disabled
  - Max tries
  - Base backoff duration
- Re-picks healthy upstreams on each retry using load balancer + circuit breakers

### 🔄 Reverse Proxy
- HTTP **reverse proxy** to upstream services
- Request/response header forwarding
- `X-Forwarded-*` headers support
- Context-driven per-request timeout

### 📊 Metrics & Observability
- Per-second aggregation of:
  - Total requests
  - Cache hits & misses
  - Latency histogram over fixed buckets
- **p95 latency** and cache hit ratio computed continuously
- Metrics are flushed to a JSONL file (e.g. `bench_metrics.jsonl`) for offline analysis

---

## 📊 Performance Benchmark (Cache vs No Cache)

This benchmark compares gateway behavior **with and without response caching** under identical load.

### 🔧 Test Setup
- Load tool: `wrk`
- Command:

```bash
wrk -t4 -c4 -d30s --latency -s hdr.lua http://localhost:8080/slow
```

- Upstream behavior: artificial delay of ~800 ms
- Environment: localhost
- Gateway restarted before each run

### 📈 Results (Representative)

| Metric                    | Without Cache | With Cache |
|---------------------------|--------------:|-----------:|
| Requests/sec              |          ~10  |   ~17,283 |
| Total Requests (30s)      |         ~300  |  ~518,779 |
| Avg Latency               |      ~399 ms  |    ~4.8 ms |
| p50 Latency               |      ~402 ms  |   ~0.18 ms |
| p75 Latency               |      ~802 ms  |   ~0.33 ms |
| p90 Latency               |      ~803 ms  |   ~0.72 ms |
| p99 Latency               |      ~805 ms  |     ~81 ms |
| Max Latency               |      ~808 ms  |    ~802 ms |
| Dominant Path             |    Upstream   |   Cache    |

**Takeaway:** caching removes the slow upstream from the critical path, yielding **orders-of-magnitude higher throughput** and shifting median latency from hundreds of milliseconds to **sub-millisecond** levels. Tail latency reflects rare cache misses (warm-up / expiry).

---

## 🏗️ Architecture Overview

### High-Level Request Flow

```text
Client
  ↓
Gateway HTTP handler
  ↓
Route matching (user + path + method)
  ↓
Load balancer → Upstream selection
Middleware chain
  ├─ Circuit breaker / upstream health selection
  ├─ Rate limiting (route + user)
  ├─ Cache lookup
  ├─ Retry handler (with backoff)
  ↓
Reverse proxy → Upstream service
  ↓
Response (optionally cached)
```

### Design Principles

- **Short-circuit early**: rate limiting, circuit breaking, and caching try to fail fast or serve from memory
- **Tail latency first**: p95 and higher percentiles are emphasized, not just averages
- **In-memory state**: makes concurrency effects and bottlenecks explicit & observable
- **Composable pieces**: routing, LB, rate limiting, caching, and retry are pluggable concepts

---

## ⚙️ Configuration Model

Although the demo builds configuration in Go, the design centers on **JSON-based per-user configuration**.

### Example Route Configuration (Conceptual)

```json
{
  "path": "/api/users",
  "method": "GET",
  "load_balancing": "round_robin",
  "upstreams": [
    {
      "url": "http://localhost:9001",
      "weight": 2,
      "circuit_breaker": {
        "enabled": true,
        "failure_threshold": 5,
        "window_seconds": 60,
        "open_seconds": 30,
        "half_open_requests": 3,
        "success_threshold": 2
      }
    }
  ],
  "route_rate_limit": {
    "type": "token_bucket",
    "capacity": 100,
    "refill_rate": 10
  },
  "user_rate_limit": {
    "type": "token_bucket",
    "capacity": 20,
    "refill_rate": 2
  },
  "user_id_key": ["header:X-User-ID", "ip"],
  "cache": {
    "enabled": true,
    "ttl_ms": 60000,
    "max_entry": 100
  },
  "retry": {
    "enabled": true,
    "max_tries": 3,
    "base_time_ms": 100
  }
}
```

In the demo (`cmd/demo/main.go`), similar configs are built programmatically for routes like `/fast`, `/slow`, `/faulty`, and `/echo`.

---

## 🛠️ Running the Demo Locally

### Prerequisites

- **Go** 1.23+
- **Git**

### 1. Clone the repository

```bash
git clone https://github.com/aryanrathod1511/FluxGate
cd FluxGate
```

### 2. Run the gateway demo

```bash
go run ./cmd/demo/main.go
```

- The gateway starts on **`http://localhost:8080`**.
- Test upstream services are started automatically on ports `9001–9005`.

### 3. Try a few requests

Use the `X-User-ID` header (the demo config expects `demo`):

```bash
curl -H "X-User-ID: demo" http://localhost:8080/fast
curl -H "X-User-ID: demo" http://localhost:8080/slow
curl -H "X-User-ID: demo" http://localhost:8080/faulty
curl -H "X-User-ID: demo" http://localhost:8080/echo
curl -H "X-User-ID: demo" -X POST \
  -H "Content-Type: application/json" \
  -d '{"msg":"hello"}' \
  http://localhost:8080/echo
```

### 4. Run the latency benchmark (optional)

Make sure the gateway is running, then:

```bash
wrk -t4 -c4 -d30s --latency -s hdr.lua http://localhost:8080/slow
```

Metrics will be flushed to `bench_metrics.jsonl` in the project root.

---

## 📁 Project Structure

- `cmd/demo/` — Demo entry point; wires configs and starts gateway + test servers
- `gateway/` — Core gateway HTTP handler and middleware composition
- `configuration/` — Route configuration models, JSON loading, and route matching
- `loadbalancer/` — Load balancer interfaces and implementations (round-robin, weighted RR)
- `ratelimit/` — Rate limiter registry and token bucket implementation
- `circuitbreaker/` — Circuit breaker implementation and state machine
- `middleware/` — Cache, rate limiting, and retry middleware
- `proxy/` — Reverse proxy and HTTP transport logic
- `storage/` — In-memory LRU cache implementation
- `matrics/` — (metrics) aggregation, p95 calculation, periodic flushing
- `testservers/` — Local upstream servers (fast, slow, faulty, echo) for experiments
- `hdr.lua` — `wrk` script for latency histogram / headers-based benchmarking

---

## 🚧 Status, Limitations & Next Steps

### Current Status

FluxGate is **experimental / educational** and designed for local testing and benchmarking.

### Known Limitations

- **In-memory only**:
  - No distributed cache
  - No shared rate limiting across processes
- **No TLS termination** (expects to sit behind a TLS-terminating proxy/load balancer)
- **No persistent configuration store** (configs are built in-code or would be loaded from local JSON)
- **Non-Prometheus metrics**: metrics are exported as JSONL, not Prometheus out of the box

### Potential Future Work 💡

- Prometheus / OpenTelemetry exporters for metrics
- Pluggable **middleware/plugin system** for custom behaviors
- Hot-reloadable configuration from file or remote store
- Additional load-balancing and rate-limiting strategies (e.g. least connections, leaky bucket)
- More advanced cache policies (e.g. request coalescing, stale-while-revalidate)

---
