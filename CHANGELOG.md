# Changelog

All notable changes to this project will be documented in this file.

## [0.6.0] - 2026-07-20
### Added
- Structured `--verbose` observability logging: `key=value` events for startup,
  request begin/end, config reloads, proxy errors, and shutdown, with
  sensitive query-string redaction and control-character sanitization. Warnings
  and errors are written to stderr, info/debug to stdout.

### Changed
- Bumped GitHub Actions Go runtime to `1.25.12` in CI and release workflows to
  clear the `GO-2026-5856` standard-library advisory.
- Release workflow now mints a short-lived GitHub App token and announces
  releases to Discord.

## [0.5.2] - 2026-06-12
### Changed
- Bumped GitHub Actions Go runtime to `1.25.11` in CI and release workflows.

### Fixed
- Restored passing `govulncheck` in CI/release after upstream Go standard library vulnerability updates.

## [0.5.0] - 2026-05-30
### Added
- `failFirstN` middleware: fails the first N requests, then passes through all subsequent requests. Config: `{ n, status?, body? }`.

## [0.4.1] - 2026-05-30
### Changed
- Hardened CI and release workflows:
	- pinned GitHub Actions to commit SHAs
	- added `govulncheck` to CI and release checks
	- pinned GoReleaser version in the release workflow
	- tightened workflow permissions

### Fixed
- Bumped GitHub Actions Go runtime to `1.25.10` so vulnerability checks pass against patched standard library versions.

### Dependencies
- Updated Go modules via Dependabot (`go.mod` and `go.sum`).

## [0.4.0] - 2026-04-21
### Added
- OpenTelemetry endpoint implementation.

## [0.3.0] - 2026-03-19
### Added
- Optional seed support for failRandomly, dropConnection, and latencyRange for reproducible randomness

### Fixed
- `bodyTransformJSON` now skips response mutation for streamed responses and preserves pass-through behavior

## [0.2.1] - 2026-03-19
### Added
- Test added for in-flight request snapshot isolation during config reload.

## [0.2.0] - 2026-03-19
### Added
- Runtime config reload via `POST /reload` endpoint without process restart.
- `ReloadConfig` programmatic API for dynamic configuration updates.
- Full atomic snapshot semantics: in-flight requests are deterministic, new requests use updated config immediately.
- Integration tests for reload success, failure/rollback, HTTP endpoint, and concurrent rejection.

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
