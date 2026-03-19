[![Build](https://github.com/fetch-kit/chaos-proxy-go/actions/workflows/ci.yml/badge.svg)](https://github.com/fetch-kit/chaos-proxy-go/actions)
[![GitHub stars](https://img.shields.io/github/stars/fetch-kit/chaos-proxy-go?style=social)](https://github.com/fetch-kit/chaos-proxy-go)

# chaos-proxy-go

**chaos-proxy-go** is a Go port of [fetch-kit/chaos-proxy](https://github.com/fetch-kit/chaos-proxy): a proxy server for injecting configurable network chaos (latency, failures, connection drops, rate-limiting, etc.) into any HTTP or HTTPS traffic. Use it via CLI or programmatically to apply ordered middleware (global and per-route) and forward requests to your target server, preserving method, path, headers, query, and body.

---

## Features

- Simple configuration via a single `chaos.yaml` file
- Programmatic API and CLI usage
- Built-in middleware primitives: latency, latencyRange, fail, failRandomly, failNth, dropConnection, rateLimit, cors, throttle, headerTransform, bodyTransformJSON
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

---

## Usage

### CLI

```sh
./chaos-proxy-go --config chaos.yaml [--verbose]
```
- `--config <path>`: YAML config file (default `./chaos.yaml`)
- `--verbose`: print loaded config, middleware setup, and per-request logs

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

See the [original chaos-proxy README](https://github.com/fetch-kit/chaos-proxy) for detailed config options. This Go port supports a compatible YAML structure.

---

## Middleware Primitives

- `latency(ms)` — delay every request
- `latencyRange(minMs, maxMs)` — random delay
- `fail({ status, body })` — always fail
- `failRandomly({ rate, status, body })` — fail with probability
- `failNth({ n, status, body })` — fail every nth request
- `dropConnection({ prob })` — randomly drop connection (`prob` defaults to `1.0` if omitted)
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

> This is a Go port of [fetch-kit/chaos-proxy](https://github.com/fetch-kit/chaos-proxy).