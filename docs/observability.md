# Observability

chaos-proxy-go supports optional OpenTelemetry tracing via the `otel` top-level config block.

If `otel` is not configured, the proxy runs without any telemetry export.

The OTLP endpoint is an operator-controlled outbound destination. Configuration
files and access to `POST /reload` must therefore be restricted to trusted
administrators; see [Hot reload](hot-reload.md).

## otel Configuration

Add an `otel` block to `chaos.yaml`:

```yaml
target: "http://localhost:4000"
port: 5000

otel:
  serviceName: "checkout-api"
  endpoint: "http://localhost:4318"
  flushIntervalMs: 1000
  maxBatchSize: 20
  maxQueueSize: 1000
  headers:
    x-tenant-id: "local-dev"

global:
  - latencyRange:
      minMs: 20
      maxMs: 120
```

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `serviceName` | string | yes | — | Service name label attached to all spans |
| `endpoint` | string | yes | — | Absolute HTTP(S) OTLP base URL without credentials, query, or fragment |
| `flushIntervalMs` | int | no | `5000` | Span queue flush interval in milliseconds (`1`–`86400000`) |
| `maxBatchSize` | int | no | `100` | Max spans per export request (`1`–`1000`) |
| `maxQueueSize` | int | no | `1000` | Max spans held in memory (`1`–`10000`, at least `maxBatchSize`) |
| `headers` | map | no | — | Extra HTTP headers added to every OTLP export request |

## What Gets Traced

Every proxied request produces one span with these attributes:

| Attribute | Value |
|-----------|-------|
| `http.method` | Request method (`GET`, `POST`, …) |
| `http.url` | Full request URL |
| `http.target` | Path + query string |
| `http.status_code` | Response status code |
| `service.name` | Value of `otel.serviceName` |

Span status is set to `STATUS_CODE_ERROR` when the response status is ≥ 400.

## Trace Context Propagation

- Incoming requests with a valid W3C `traceparent` header have the trace ID continued.
- Requests without `traceparent` start a fresh trace.
- The outgoing request to the upstream target carries the updated `traceparent` header.

## Middleware Position

`otel` is always applied before the `global` middleware chain and route-specific middleware. This means chaos effects (latency, failures) are included in the span duration and error status.

## Connecting to a Collector

Point `endpoint` at any OTLP-compatible HTTP collector:

```yaml
otel:
  serviceName: "my-service"
  endpoint: "http://localhost:4318"
```

Or with authentication headers (e.g. Grafana Cloud, Honeycomb):

```yaml
otel:
  serviceName: "my-service"
  endpoint: "https://otlp.example.com"
  headers:
    x-honeycomb-team: "your-api-key"
```

## Using with chaos-proxy's Observability Stack

If you have [chaos-proxy](https://github.com/fetch-kit/chaos-proxy) installed, its built-in observability stack (OTel Collector → Prometheus → Jaeger → Grafana) works directly with chaos-proxy-go.

Start the stack from the chaos-proxy directory:

```sh
npm run obs:up
```

Then configure chaos-proxy-go to export to its collector:

```yaml
otel:
  serviceName: "chaos-proxy-go-dev"
  endpoint: "http://localhost:4318"
```

Endpoints:

| Service | URL |
|---------|-----|
| Grafana | `http://localhost:3000` |
| Prometheus | `http://localhost:9090` |
| Jaeger | `http://localhost:16686` |
| OTLP HTTP | `http://localhost:4318` |

## Exporter Lifecycle

- The exporter is started once when the proxy starts (if `otel` is configured).
- On config reload via `POST /reload`:
  - If `otel` config is unchanged, the existing exporter is reused (no restart, no queue loss).
  - If `otel` config changes, a new exporter is started and the old one is flushed and stopped.
  - If `otel` is removed, the exporter is flushed and stopped.
- On proxy shutdown, the exporter flushes any remaining spans before exiting.

## Troubleshooting

If spans are not appearing:

1. Confirm the collector is reachable: `curl http://localhost:4318/v1/traces` should return something (even an error body means the port is open).
2. Check proxy logs for `[otel] export failed` lines — these indicate network or auth issues.
3. Verify `serviceName` matches what you are filtering by in Jaeger or Grafana.
4. If using `maxBatchSize` and traffic is low, spans may be held until `flushIntervalMs` expires — lower `flushIntervalMs` to see data sooner.
