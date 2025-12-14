# FluxGate - High-Performance API Gateway

A high-performance API Gateway built in Go with routing, load balancing, rate limiting, circuit breaking, caching, and extensible plugin architecture.

## 🚀 Features

### ✅ Implemented Features

- **🔀 Intelligent Routing**
  - Path-based routing with HTTP method matching
  - Wildcard and parameter support (`:param`, `{param}`, `*`)
  - Route scoring for best match selection
  - Per-user route configuration

- **⚖️ Load Balancing**
  - Round Robin algorithm
  - Weighted Round Robin algorithm
  - Extensible load balancer registry
  - Automatic unhealthy server detection via circuit breakers

- **🚦 Rate Limiting**
  - Token Bucket algorithm implementation
  - Route-level rate limiting
  - User-level rate limiting
  - Flexible user identification (headers, query params, cookies, JWT, IP, etc.)
  - Extensible rate limiter registry

- **🔌 Circuit Breaking**
  - Three-state circuit breaker (Closed, Open, Half-Open)
  - Configurable failure thresholds
  - Time-based window for failure tracking
  - Automatic recovery with half-open state
  - Per-upstream circuit breaker instances

- **💾 Response Caching**
  - LRU (Least Recently Used) cache implementation
  - Configurable TTL (Time To Live)
  - Per-route cache configuration
  - Automatic cache eviction

- **🔄 Reverse Proxy**
  - HTTP/HTTPS reverse proxy
  - Request/response header forwarding
  - X-Forwarded-For, X-Forwarded-Host, X-Forwarded-Proto headers
  - Configurable request timeouts

### 🚧 Planned Features

- **🔄 Retries**
  - Automatic retry with exponential backoff
  - Configurable retry attempts and base time
  - Retry on specific HTTP status codes

- **🔌 Plugin System**
  - Extensible middleware plugin architecture
  - Custom plugin registration
  - Plugin execution pipeline

## 📋 Table of Contents

- [Installation](#installation)
- [Quick Start](#quick-start)
- [Configuration](#configuration)
- [Architecture](#architecture)
- [Roadmap](#-planned-features)


## 🛠️ Installation

### Prerequisites

- Go 1.23.3 or higher
- Git

### Build from Source

```bash
# Clone the repository
git clone <repository-url>
cd API_gateway

# Build the gateway
go build -o fluxgate ./cmd/demo

# Run the gateway
./fluxgate
```

## 🚀 Quick Start

1. **Start the gateway demo:**

```bash
go run ./cmd/demo/main.go
```

The gateway will start on `http://localhost:8080` and automatically start test upstream servers.

2. **Test the gateway:**

```bash
# Make a request with user identification header
curl -H "X-User-ID: demo" http://localhost:8080/fast

# Check health endpoint
curl http://localhost:8080/health
```

## ⚙️ Configuration

The gateway uses JSON-based configuration. Each user can have their own set of routes.

### Route Configuration Structure

```json
{
  "path": "/api/users",
  "method": "GET",
  "load_balancing": "round_robin",
  "upstreams": [
    {
      "url": "http://localhost:9001",
      "weight": 2,
      "retry_enabled": false,
      "retries": 3,
      "base_time_ms": 100,
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
    "capacity": 100.0,
    "refill_rate": 10.0
  },
  "user_rate_limit": {
    "type": "token_bucket",
    "capacity": 20.0,
    "refill_rate": 2.0
  },
  "user_id_key": ["header:X-User-ID", "ip"],
  "cache": {
    "enabled": true,
    "ttl_ms": 60000,
    "max_entry": 100
  },
  "plugins": []
}
```

### Configuration Fields

#### Route Configuration

| Field | Type | Description | Required |
|-------|------|-------------|----------|
| `path` | string | Route path pattern (supports wildcards and parameters) | Yes |
| `method` | string | HTTP method (GET, POST, PUT, DELETE, etc.) | Yes |
| `load_balancing` | string | Load balancing algorithm (`round_robin`, `weighted_rr`) | Yes |
| `upstreams` | array | List of upstream server configurations | Yes |
| `route_rate_limit` | object | Route-level rate limiting configuration | No |
| `user_rate_limit` | object | User-level rate limiting configuration | No |
| `user_id_key` | array | User identification methods (see below) | No |
| `cache` | object | Caching configuration | No |
| `plugins` | array | Plugin names (planned feature) | No |

#### Upstream Configuration

| Field | Type | Description | Default |
|-------|------|-------------|---------|
| `url` | string | Upstream server URL | - |
| `weight` | int | Weight for weighted round-robin | 1 |
| `retry_enabled` | bool | Enable retries (planned) | false |
| `retries` | int | Number of retry attempts (planned) | 0 |
| `base_time_ms` | int64 | Base retry delay in milliseconds (planned) | 0 |
| `circuit_breaker` | object | Circuit breaker configuration | - |

#### Circuit Breaker Configuration

| Field | Type | Description | Default |
|-------|------|-------------|---------|
| `enabled` | bool | Enable circuit breaker | false |
| `failure_threshold` | int | Number of failures to open circuit | 5 |
| `window_seconds` | int | Time window for counting failures | 60 |
| `open_seconds` | int | Duration to keep circuit open | 30 |
| `half_open_requests` | int | Max concurrent requests in half-open state | 3 |
| `success_threshold` | int | Successes needed to close from half-open | 2 |

#### Rate Limit Configuration

| Field | Type | Description | Default |
|-------|------|-------------|---------|
| `type` | string | Rate limiter type (`token_bucket`, `none`) | `none` |
| `capacity` | float64 | Maximum tokens (burst capacity) | 0 |
| `refill_rate` | float64 | Tokens refilled per second | 0 |

#### Cache Configuration

| Field | Type | Description | Default |
|-------|------|-------------|---------|
| `enabled` | bool | Enable caching | false |
| `ttl_ms` | int | Time-to-live in milliseconds | 0 |
| `max_entry` | int | Maximum cache entries (LRU eviction) | 100 |

### User Identification

The `user_id_key` field supports multiple identification methods:

- `header:X-User-ID` - Extract from HTTP header
- `query:uid` - Extract from query parameter
- `cookie:session_id` - Extract from cookie
- `form:user_id` - Extract from form data
- `jwt:sub` - Extract from JWT token (Authorization header)
- `basic:username` - Extract from Basic Auth username
- `ip` - Use client IP address

The gateway tries each method in order until it finds a value.

### Path Matching

Routes support flexible path matching:

- **Exact match**: `/api/users`
- **Parameters**: `/api/users/:id` or `/api/users/{id}`
- **Wildcards**: `/api/*` (matches any path starting with `/api/`)

Routes are scored, and the most specific match is selected.

## 🏗️ Architecture

### Project Structure

```
API_gateway/
├── cmd/
│   └── demo/
│       └── main.go              # Demo application entry point
├── gateway/
│   ├── server.go                # Gateway server implementation
│   └── router.go                # Routing logic
├── configuration/
│   ├── config.go                # Configuration structures
│   └── store.go                 # Configuration store and matching
├── loadbalancer/
│   ├── loadBalancer.go          # Load balancer interface
│   ├── factory.go               # Load balancer factory
│   ├── registry.go              # Load balancer registry
│   ├── round_robin.go           # Round robin implementation
│   └── weighted_rr.go           # Weighted round robin implementation
├── ratelimit/
│   ├── ratelimiter.go           # Rate limiter interface
│   ├── factory.go               # Rate limiter factory
│   ├── registry.go              # Rate limiter registry
│   └── token_bucket.go          # Token bucket implementation
├── circuitbreaker/
│   └── circuitbreaker.go        # Circuit breaker implementation
├── middleware/
│   ├── rate_limiter.go          # Rate limiting middleware
│   ├── cache.go                 # Caching middleware
│   └── breaker.go                # Circuit breaker middleware
├── proxy/
│   └── reverse_proxy.go         # Reverse proxy implementation
├── storage/
│   └── cache.go                 # LRU cache implementation
├── utils/
│   └── pick_healthy_server.go   # Healthy server selection
└── testservers/
    └── servers.go               # Test upstream servers
```

### Request Flow

```
Client Request
    ↓
Gateway Handler
    ↓
Route Matching (by user, path, method)
    ↓
Load Balancer (select healthy upstream)
    ↓
Middleware Chain:
    ├─ Circuit Breaker Middleware
    ├─ Rate Limiter Middleware (route + user)
    └─ Cache Middleware
    ↓
Reverse Proxy (forward to upstream)
    ↓
Response (with caching if applicable)
```

### Extensibility

The gateway uses a registry pattern for extensibility:

- **Load Balancers**: Register custom algorithms via `loadbalancer.RegistrLoadBalancer()`
- **Rate Limiters**: Register custom algorithms via `ratelimit.RegisterRateLimiter()`
- **Plugins**: Plugin system infrastructure (planned)



## 🎯 Planned Features

- [ ] **Retry Mechanism**
  - Exponential backoff
  - Configurable retry attempts
  - Retry on specific status codes

- [ ] **Plugin System**
  - Plugin interface definition
  - Plugin registration and execution
  - Built-in plugins (auth, logging, transformation)

- [ ] **Additional Load Balancers**
  - Least connections
 

- [ ] **Additional Rate Limiters**
  - Sliding window
  - Fixed window
  - Leaky bucket

- [ ] **Observability**
  - Metrics export (Prometheus)
  - Structured logging

- [ ] **Security**
  - JWT validation
  - API key authentication
  - CORS support

- [ ] **Configuration Management**
  - File-based configuration
  - Hot reload
  - Configuration validation

---

**Note**: This is an active development project. Some features (retries, plugin system and Observability) are planned but not yet implemented.
