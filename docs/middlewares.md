# Chaos Middlewares

This guide covers the built-in middleware primitives, ordering, and behavior details.

## Middleware Order

chaos-proxy-go runs middleware in this order:

1. Optional `otel` middleware (when configured)
2. `global` middleware chain
3. Matching route middleware chain

The first middleware to send a response or terminate the request short-circuits later middleware.

## Built-in Primitives

### `latency`

Delays every request by a fixed number of milliseconds.

```yaml
global:
  - latency:
      ms: 200
```

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `ms` | int | yes | — | Fixed delay in milliseconds |

---

### `latencyRange`

Delays every request by a random value within a range.

```yaml
global:
  - latencyRange:
      minMs: 50
      maxMs: 300
```

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `minMs` | int | yes | — | Lower bound in milliseconds |
| `maxMs` | int | yes | — | Upper bound in milliseconds |
| `seed` | int | no | — | Seed for deterministic output |

---

### `fail`

Always responds with an error status, short-circuiting the request.

```yaml
global:
  - fail:
      status: 503
      body: "service unavailable"
```

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `status` | int | no | `503` | HTTP status code |
| `body` | string | no | — | Response body text |

---

### `failRandomly`

Fails with the given probability on each request.

```yaml
global:
  - failRandomly:
      rate: 0.1
      status: 503
```

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `rate` | float | yes | — | Failure probability (0.0–1.0) |
| `status` | int | no | `503` | HTTP status code |
| `body` | string | no | — | Response body text |
| `seed` | int | no | — | Seed for deterministic output |

---

### `failNth`

Fails every nth request, then resets the counter.

```yaml
global:
  - failNth:
      n: 5
      status: 500
```

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `n` | int | yes | — | Fail on every nth request |
| `status` | int | no | `503` | HTTP status code |
| `body` | string | no | — | Response body text |

---

### `dropConnection`

Randomly closes the connection without sending a response.

```yaml
global:
  - dropConnection:
      prob: 0.05
```

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `prob` | float | no | `1.0` | Drop probability (0.0–1.0) |
| `seed` | int | no | — | Seed for deterministic output |

---

### `rateLimit`

Enforces a fixed-window request rate limit. Returns `429` when exceeded, with `X-RateLimit-Limit`, `X-RateLimit-Remaining`, and `X-RateLimit-Reset` response headers.

```yaml
global:
  - rateLimit:
      limit: 100
      windowMs: 60000
      key: "Authorization"
```

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `limit` | int | yes | — | Max requests per window |
| `windowMs` | int | yes | — | Window duration in milliseconds |
| `key` | string | no | — | Header name to bucket by; falls back to client remote address |

---

### `cors`

Sets CORS response headers and handles `OPTIONS` preflight.

```yaml
global:
  - cors:
      origin: "https://app.example.com"
      methods: "GET,POST"
      headers: "Content-Type,Authorization"
```

| Field | Type | Required | Default |
|-------|------|----------|---------|
| `origin` | string | no | `*` |
| `methods` | string | no | `GET,POST,PUT,DELETE,OPTIONS` |
| `headers` | string | no | `Content-Type,Authorization` |

---

### `throttle`

Limits response bandwidth by streaming the body in chunks at a controlled rate.

```yaml
global:
  - throttle:
      rate: 51200
      chunkSize: 1024
      burst: 10240
```

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `rate` | int | yes | — | Bytes per second |
| `chunkSize` | int | no | `16384` | Chunk size in bytes |
| `burst` | int | no | — | Extra bytes allowed before steady-state throttling |

---

### `headerTransform`

Sets or deletes request and/or response headers.

```yaml
global:
  - headerTransform:
      request:
        set:
          x-chaos: "injected"
        delete:
          - "Authorization"
      response:
        set:
          x-powered-by: "chaos-proxy-go"
```

Both `request` and `response` blocks are optional.

---

### `bodyTransformJSON`

Sets or deletes fields in JSON request and/or response bodies. Skipped for streamed responses (missing `Content-Length` with `Transfer-Encoding: chunked` or `Content-Type: text/event-stream`).

```yaml
routes:
  POST /api/orders:
    - bodyTransformJSON:
        request:
          set:
            injected: true
          delete:
            - "sensitiveField"
        response:
          set:
            chaos: true
```

Both `request` and `response` blocks are optional.

---

## Middleware YAML Shape

Each entry is a single-key map. One middleware per list item for deterministic ordering:

```yaml
global:
  - latency:
      ms: 100
  - failRandomly:
      rate: 0.05
      status: 503
  - failNth:
      n: 10
      status: 500
```

> **Note:** If you put multiple middleware keys in the same map element, execution order between them is non-deterministic. Use separate list items.

## Determinism

For randomness-based primitives (`latencyRange`, `failRandomly`, `dropConnection`), set an integer `seed` to make behavior fully reproducible across restarts and reloads.

```yaml
global:
  - failRandomly:
      rate: 0.2
      seed: 42
```
