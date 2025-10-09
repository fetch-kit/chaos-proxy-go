[![Build](https://github.com/your-org/chaos-proxy-go/actions/workflows/ci.yml/badge.svg)](https://github.com/your-org/chaos-proxy-go/actions)
[![GitHub stars](https://img.shields.io/github/stars/your-org/chaos-proxy-go?style=social)](https://github.com/your-org/chaos-proxy-go)

# chaos-proxy-go

**chaos-proxy-go** is a Go port of [fetch-kit/chaos-proxy](https://github.com/fetch-kit/chaos-proxy): a proxy server for injecting configurable network chaos (latency, failures, connection drops, rate-limiting, etc.) into any HTTP or HTTPS traffic. Use it via CLI or programmatically to apply ordered middleware (global and per-route) and forward requests to your target server, preserving method, path, headers, query, and body.

---

## Features

- Simple configuration via a single `chaos.yaml` file
- Programmatic API and CLI usage
- Built-in middleware primitives: latency, latencyRange, fail, failRandomly, failNth, dropConnection, rateLimit, cors, throttle, bodyTransform
- Extensible registry for custom middleware
- Supports both request and response interception/modification
- Method+path route support (e.g., `GET /api/users`)
- Robust short-circuiting: middlewares halt further processing when sending a response or dropping a connection

---

## Installation

Download the latest release from [GitHub Releases](https://github.com/your-org/chaos-proxy-go/releases) or build from source:

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
- `--verbose`: print loaded middlewares and request logs

### Programmatic API

```go
import "your-org/chaos-proxy-go/internal/proxy"

// Load config and start server
cfg := config.Load("chaos.yaml")
server := proxy.New(cfg, false)
server.Start()
// ...
```

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
- `dropConnection({ prob })` — randomly drop connection
- `rateLimit({ limit, windowMs, key })` — rate limiting (by IP, header, or custom)
- `cors({ origin, methods, headers })` — enable and configure CORS headers
- `throttle({ rate, chunkSize, burst, key })` — throttles bandwidth per request
- `bodyTransform({ transform })` — parse and mutate request body with a custom function

---

## Extensibility

Register custom middleware in Go. See the `internal/middleware` package for examples.

---

## Security & Limitations

- Proxy forwards all headers; be careful with sensitive tokens.
- Intended for local/dev/test only.
- HTTPS pass-through requires TLS termination; not supported out-of-the-box.
- Not intended for stress testing; connection limits apply.

---

## License

MIT

---

> This is a Go port of [fetch-kit/chaos-proxy](https://github.com/fetch-kit/chaos-proxy).