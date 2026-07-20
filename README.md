[![Build](https://github.com/fetch-kit/chaos-proxy-go/actions/workflows/ci.yml/badge.svg)](https://github.com/fetch-kit/chaos-proxy-go/actions)
[![GitHub stars](https://img.shields.io/github/stars/fetch-kit/chaos-proxy-go?style=social)](https://github.com/fetch-kit/chaos-proxy-go)

# chaos-proxy-go


> Part of the **[fetch-kit ecosystem](https://fetchkit.org)** - production-ready fetch utilities and chaos-testing tools.

**chaos-proxy-go** is a Go port of [fetch-kit/chaos-proxy](https://github.com/fetch-kit/chaos-proxy): a proxy server for injecting configurable network chaos (latency, failures, connection drops, rate-limiting, etc.) into any HTTP or HTTPS traffic. Use it via CLI or programmatically to apply ordered middleware (global and per-route) and forward requests to your target server, preserving method, path, headers, query, and body.

---

## Quick Start

1. Download from [GitHub Releases](https://github.com/fetch-kit/chaos-proxy-go/releases) or build from source:

   ```sh
   go build -o chaos-proxy-go .
   ```

2. Create a minimal `chaos.yaml`:

   ```yaml
   target: "http://localhost:4000"
   port: 5000
   global:
     - failRandomly:
         rate: 0.1
         status: 503
   ```

3. Run the proxy:

   ```sh
   ./chaos-proxy-go --config chaos.yaml --verbose
   ```

All traffic to `http://localhost:5000` is now forwarded to `http://localhost:4000` with 10% random 503 failures injected.

---

## Documentation

- [Middleware reference](docs/middlewares.md) — all 11 built-in primitives with config tables
- [Observability](docs/observability.md) — OpenTelemetry tracing, span attributes, connecting to a collector
- [Hot reload](docs/hot-reload.md) — runtime config reload, endpoint spec, edge cases

---

## Presets

Ready-made scenarios in the [`presets/`](presets/) folder:

| Preset | Simulates |
|--------|-----------|
| [`flaky-backend.yaml`](presets/flaky-backend.yaml) | Unstable upstream: latency jitter, 5% 503s, 2% connection drops |
| [`mobile-3g.yaml`](presets/mobile-3g.yaml) | Mobile 3G: 100–300ms latency, 50 KB/s bandwidth, 1% drops |
| [`burst-errors.yaml`](presets/burst-errors.yaml) | Error bursts: every 5th fails with 500, plus 10% random 503s |
| [`timeout-storm.yaml`](presets/timeout-storm.yaml) | Timeout storm: 1–8s delays, 10% drops, 15% instant 504s |

Run a preset directly:

```sh
./chaos-proxy-go --config presets/flaky-backend.yaml
```

---

## Features

- Simple configuration via a single `chaos.yaml` file
- Programmatic API and CLI usage
- Built-in middleware primitives: latency, latencyRange, fail, failRandomly, failFirstN, failNth, dropConnection, rateLimit, cors, throttle, headerTransform, bodyTransformJSON
- Extensible registry for custom middleware
- Supports both request and response interception/modification
- Method+path route support (e.g., `GET /api/users`)
- Robust short-circuiting: middlewares halt further processing when sending a response or dropping a connection
- Runtime config reload via `POST /reload` without process restart

---

## Installation

Download the latest release from [GitHub Releases](https://github.com/fetch-kit/chaos-proxy-go/releases) or build from source:

```sh
go build -o chaos-proxy-go .
```

Requires Go 1.21 or later.

---

## Usage

### CLI

```sh
./chaos-proxy-go --config chaos.yaml [--verbose]
```
- `--config <path>`: YAML config file (default `./chaos.yaml`)
- `--verbose`: emit structured `key=value` observability logs for startup, each
  request (`verbose.request.begin` / `verbose.request.end`), config reloads,
  proxy errors, and shutdown. Sensitive query-string values (e.g. `token`,
  `secret`, `password`, `api_key`) are redacted and control characters are
  stripped. `INFO`/`DEBUG` lines go to stdout; `WARN`/`ERROR` to stderr.

### Programmatic API

```go
import (
	"log"

	"chaos-proxy-go/internal/config"
	"chaos-proxy-go/internal/proxy"
)

cfg, err := config.Load("chaos.yaml")
if err != nil {
	log.Fatalf("failed to load config: %v", err)
}
server, err := proxy.New(cfg, false)
if err != nil {
	log.Fatalf("failed to create server: %v", err)
}
if err := server.Start(); err != nil {
	log.Fatalf("server error: %v", err)
}
```

---

## Runtime Config Reload

Chaos Proxy supports full runtime reloads without process restart.

- Endpoint: `POST /reload`
- Content-Type: `application/json`
- Payload: full config snapshot (same shape as `chaos.yaml`, but JSON)
- Behavior: build-then-swap — all-or-nothing, the active state is never partially updated
- Body size limit: 1 MB

### Request Example

```sh
curl -X POST http://localhost:5000/reload \
  -H "Content-Type: application/json" \
  -d '{
    "target": "http://localhost:4000",
    "port": 5000,
    "global": [
      { "latency": { "ms": 120 } },
      { "failRandomly": { "rate": 0.05, "status": 503 } }
    ],
    "routes": {
      "GET /users/:id": [
        { "failNth": { "n": 3, "status": 500 } }
      ]
    }
  }'
```

### Success Response

```json
{
  "ok": true,
  "version": 2,
  "reload_ms": 3
}
```

### Failure Responses

| Status | Reason |
|--------|--------|
| `400` | Invalid or unparseable config (active state is unchanged) |
| `409` | Reload already in progress |
| `415` | Wrong `Content-Type` (must be `application/json`) |

```json
{
  "ok": false,
  "error": "target is required",
  "version": 1,
  "reload_ms": 0
}
```

### Programmatic Reload

`proxy.New(...)` returns a `*Server` with a `ReloadConfig` method:

```go
result := server.ReloadConfig(newCfg)
if !result.OK {
    log.Printf("reload failed: %s", result.Error)
} else {
    log.Printf("reloaded to version %d in %dms", result.Version, result.ReloadMs)
}
```

### Edge-Case Semantics

- **In-flight requests** are deterministic: they run on the snapshot captured at the moment the request arrived, immune to concurrent reloads.
- **New requests** after a successful swap immediately use the new snapshot.
- **All-or-nothing**: if parse, validate, or middleware-build fails, the active state is unchanged.
- **Middleware state resets** on reload (e.g., rate-limit and failNth counters start fresh).
- **Concurrent reloads** are rejected with `409`; the second caller must retry.

---

## Configuration (`chaos.yaml`)

```yaml
target: "http://localhost:4000"  # required
port: 5000                       # default: 5000

# Optional: OpenTelemetry tracing
otel:
  serviceName: "my-service"      # required if otel is set
  endpoint: "http://localhost:4318" # required if otel is set
  flushIntervalMs: 5000          # default
  maxBatchSize: 100              # default
  maxQueueSize: 1000             # default
  headers:                       # optional OTLP request headers
    x-api-key: "secret"

# Global middleware — applied to every proxied request
global:
  - latencyRange:
      minMs: 20
      maxMs: 100
  - failRandomly:
      rate: 0.05
      status: 503

# Route-specific middleware — "METHOD /path" format
routes:
  GET /api/users:
    - latency:
        ms: 500
  POST /api/orders:
    - failNth:
        n: 3
        status: 500
```

See [docs/middlewares.md](docs/middlewares.md) for the full middleware reference and [docs/observability.md](docs/observability.md) for `otel` options.

---

## Middleware Primitives

- `latency(ms)` — delay every request
- `latencyRange(minMs, maxMs, seed?)` — random delay (deterministic when `seed` is set)
- `fail({ status, body })` — always fail
- `failRandomly({ rate, status, body, seed? })` — fail with probability (deterministic when `seed` is set)
- `failNth({ n, status, body })` — fail every nth request
- `dropConnection({ prob, seed? })` — randomly drop connection (`prob` defaults to `1.0` if omitted; deterministic when `seed` is set)
- `rateLimit({ limit, windowMs, key })` — rate limiting (by header key if configured, otherwise by client remote address in ip:port format)
- `cors({ origin, methods, headers })` — enable and configure CORS headers
- `throttle({ rate, chunkSize, burst })` — throttles bandwidth per request (`rate` is bytes/second)
- `headerTransform({ request: { set, delete }, response: { set, delete } })` — mutate request/response headers
- `bodyTransformJSON({ request: { set, delete }, response: { set, delete } })` — mutate JSON request/response bodies

---

## Extensibility

Register custom middleware in Go. See the `internal/middleware` package for examples.

---

## Security & Limitations

- Proxy forwards all headers; be careful with sensitive tokens.
- Intended for local/dev/test only.
- HTTPS pass-through requires TLS termination; not supported out-of-the-box.
- Not intended for stress testing; connection limits apply.
- Middleware execution order is nondeterministic when multiple middlewares are in the same YAML map element. For example:
  ```yaml
  global:
    - latency: { ms: 100 }
      fail: { status: 500 }      # Order vs latency is not guaranteed
  ```
  For deterministic order, use separate map elements:
  ```yaml
  global:
    - latency: { ms: 100 }
    - fail: { status: 500 }      # Always runs after latency
  ```

---

## License

MIT

---

Go port of [fetch-kit/chaos-proxy](https://github.com/fetch-kit/chaos-proxy).

---

> This is a Go port of [fetch-kit/chaos-proxy](https://github.com/fetch-kit/chaos-proxy).