# Changelog

All notable changes to this project will be documented in this file.

## [0.1.0] - 2026-03-18
### Added
- Graceful shutdown using `http.Server` with OS signal handling.
- `--verbose` per-request logging.
- Throttle middleware registration in the default middleware registry.

### Changed
- Improved project documentation.

### Fixed
- Client abort propagation in `latency` and `latencyRange` middleware.
- Throttle burst allowance reset per request.
- `--help` no longer starts the server (startup logic moved into Cobra `RunE`).
- Rate-limit race condition and negative `X-RateLimit-Remaining` header values.
- Throttle delay precision using nanosecond-accurate duration calculation.
- `bodyTransformJSON` now applies to parameterized `Content-Type` headers.
- Silent `url.Parse` error handling and `GET`/`POST` typo in example config.

## [0.0.1] - 2025-10-09
### Added
- Initial release: Go port of [fetch-kit/chaos-proxy](https://github.com/fetch-kit/chaos-proxy)
- Core proxy server and middleware registry
- Middleware: latency, fail, headerTransform, bodyTransformJSON, rateLimit, cors, throttle, dropConnection, etc.
- CLI and programmatic API
- YAML configuration support
- Full integration and unit test suite
