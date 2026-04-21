# Hot Reload

chaos-proxy-go supports full runtime config reload without a process restart.

## How It Works

When a reload is triggered:

1. The new payload is parsed and validated (same rules as `chaos.yaml`).
2. Middleware chains are compiled against the new config.
3. The compiled snapshot is swapped atomically.
4. In-flight requests that started before the swap continue running on the previous snapshot.
5. All new requests after the swap use the new snapshot immediately.

If parsing, validation, or middleware compilation fails, the swap is aborted and the running config is unchanged.

## Reload Endpoint

```
POST /reload
Content-Type: application/json
```

The payload is the full config object in JSON — the same shape as `chaos.yaml`.

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

`version` increments by 1 on every successful reload.

### Error Responses

| Status | Meaning |
|--------|---------|
| `400` | Invalid config or unparseable payload — running config is unchanged |
| `409` | Another reload is already in progress |
| `415` | Wrong `Content-Type` — must be `application/json` |

```json
{
  "ok": false,
  "error": "target is required",
  "version": 1,
  "reload_ms": 0
}
```

## Programmatic Reload

`proxy.New(...)` returns a `*Server` with a `ReloadConfig` method:

```go
newCfg, err := config.Load("chaos-new.yaml")
if err != nil {
    log.Fatalf("load: %v", err)
}

result := server.ReloadConfig(newCfg)
if !result.OK {
    log.Printf("reload failed: %s", result.Error)
} else {
    log.Printf("reloaded to version %d in %dms", result.Version, result.ReloadMs)
}
```

## Edge Cases

- **In-flight requests** are deterministic: they complete on the snapshot captured when the request arrived, immune to any concurrent reload.
- **Removed routes** in the new config do not affect in-flight requests that already matched those routes on the old snapshot.
- **Middleware state resets** on reload — rate-limit counters, `failNth` counters, and similar stateful middleware start fresh with every successful reload.
- **All-or-nothing** — partial config application never occurs.
- **Concurrent reloads** are rejected with `409`; the second caller should retry.
- **OTEL exporter** is reused if the `otel` config block is unchanged, replaced if it changes, and shut down if removed. See [observability.md](observability.md).
