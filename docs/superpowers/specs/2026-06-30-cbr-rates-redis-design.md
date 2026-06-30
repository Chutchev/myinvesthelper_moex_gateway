# CBR Rates and Forecast with Redis Cache Design

**Date:** 2026-06-30
**Status:** Approved

## Overview

This specification covers completion of the partially implemented CBR integration in the Go gateway. It preserves the behavior and JSON contract of the Python `CbrRateService`, while separating HTTP transport, parsing, orchestration, and caching in the same style as the existing MOEX integration.

## Scope

### In scope

- HTTP client for the CBR key-rate page, forecast page, and discovered XLSX workbook.
- HTML parsing for key-rate history and forecast workbook discovery.
- XLSX parsing for both workbook formats supported by the Python service.
- Service orchestration with independent degradation of rate and forecast sources.
- Redis caching of complete snapshots for 24 hours.
- Production dependency wiring for `GET /v1/cbr/rates`.
- Compatible response fields and date formats.
- Unit and HTTP contract tests.

### Out of scope

- Changes to the Python backend or `bond_sync.py`.
- Database persistence.
- Background refresh, retries, stale-cache fallback, or distributed locking.
- Changes to the public `RateSnapshot` JSON schema.
- Caching partial or warning-bearing snapshots.

## Existing Contract

The endpoint remains `GET /v1/cbr/rates` and returns the existing `cbr.RateSnapshot`:

```go
type RateSnapshot struct {
	CurrentRate *float64      `json:"current_rate"`
	History     []RatePoint   `json:"history"`
	Direction   Direction     `json:"direction"`
	Forecast    *RateForecast `json:"forecast"`
	FetchedAt   time.Time     `json:"fetched_at"`
	Warnings    []string      `json:"warnings"`
}
```

Field names and meanings remain compatible. Effective and publication dates remain date strings; `fetched_at` remains a JSON timestamp.

## Architecture

The implementation has four responsibilities:

1. `cbr.HTTPClient` downloads raw HTML and XLSX bytes.
2. Parser functions convert raw payloads into CBR domain types.
3. A real `cbr.Service` independently orchestrates current-rate and forecast retrieval, freshness checks, warnings, and caching.
4. `app.New` wires the HTTP client, service, and existing Redis cache into the existing Fiber handler.

Handlers continue to depend only on the `cbr.Service` interface. The existing `StubService` may remain for isolated tests, but production wiring must not use it.

## HTTP Client

Implement the existing client methods:

```go
type Client interface {
	FetchKeyRatePage(ctx context.Context) ([]byte, error)
	FetchForecastPage(ctx context.Context) ([]byte, error)
	FetchForecastWorkbook(ctx context.Context, url string) ([]byte, error)
}
```

Requirements:

- Use `CBR_KEY_RATE_URL` and `CBR_FORECAST_PAGE_URL` from configuration.
- Use the configured `HTTP_TIMEOUT` for every request.
- Propagate the caller's `context.Context`.
- Close response bodies on every path.
- Follow standard `net/http` redirect behavior.
- Send `User-Agent: myinvesthelper-gateway/1.0`.
- Limit HTML responses to 4 MiB and workbook responses to 32 MiB.
- Reject non-2xx responses.
- Preserve `context.Canceled`; classify deadline expiration, HTTP status failures, network failures, and oversized responses without exposing response bodies.
- Resolve the discovered workbook URL relative to `CBR_FORECAST_PAGE_URL` before downloading it.

## Key-rate HTML Parsing

Use `golang.org/x/net/html` and preserve the Python algorithm:

- Walk table rows and collect text from `td` and `th` cells, including nested elements.
- Treat the first cell as a date in `DD.MM.YYYY` and the second as a numeric rate.
- Skip headers and malformed rows.
- Accept decimal comma or point, regular and non-breaking spaces, and Unicode minus.
- Return a parsing error if no valid rate points are found.
- Sort points by effective date descending.
- Set `current_rate` from the newest point.
- Calculate direction from the newest two rates:
  - newest greater than previous: `up`;
  - newest lower than previous: `down`;
  - equal: `flat`;
  - fewer than two points: `unknown`.

## Forecast Discovery and XLSX Parsing

### Workbook discovery

- Parse all anchor elements on the forecast page.
- Normalize anchor text by trimming, lowercasing, replacing non-breaking spaces, and collapsing whitespace.
- Select the first link whose text contains `агрегированные результаты опроса`.
- Resolve relative links against the configured forecast page URL.
- Return a parsing error if no matching link exists.

### Workbook parsing

Use `github.com/xuri/excelize/v2`. Preserve both parsing paths from Python, trying the matrix format first and the fallback table format second.

For the current matrix format:

- Identify a sheet mentioning `ключевая ставка` in its first five rows.
- Locate the row containing `прогнозный период` and its dated survey columns.
- Locate the `медиана`, `10-й процентиль`/`10 перцентиль`, and `90-й процентиль`/`90 перцентиль` sections.
- Target the year after `fetched_at.Year()`.
- Inspect survey columns from newest to oldest and return the first column containing all three values for the target year.
- Set `published_at` to the survey column date.

For the fallback table format:

- Find a header containing `показатель` and future-year columns.
- Select the nearest future year.
- Find the row containing `ключевая ставка`.
- Use the annual value as midpoint.
- Use `10 перцентиль`, falling back to `минимум` and then midpoint, for the low value.
- Use `90 перцентиль`, falling back to `максимум` and then midpoint, for the high value.
- Set `published_at` to the fetch date, matching Python behavior.

Return a parsing error when neither format yields a forecast. The forecast `source_url` is the resolved workbook URL.

## Service Behavior

On `Snapshot(ctx)`:

1. Attempt to read `cbr:rates:snapshot` from Redis.
2. On cache hit, return the stored snapshot unchanged. Its `fetched_at` remains the time of the original fetch.
3. On cache miss or cache read failure, fetch current-rate data and forecast data independently.
4. Use one injected UTC clock value for the entire operation.
5. Build warnings and partial results with the same semantics as Python.
6. Cache the snapshot only when it has no warnings.
7. Return the snapshot even if one or both upstream branches failed.

Caller cancellation is the exception to partial degradation: if `ctx` is canceled, stop processing and return `context.Canceled` instead of converting it into a warning or caching a result.

Current-rate branch:

- On success, return history, current rate, and calculated direction.
- If the newest rate is older than `CBR_RATE_MAX_AGE_DAYS`, set `current_rate` to `null`, set direction to `unknown`, preserve history, and add `Текущая ставка Банка России устарела`.
- On HTTP or parsing failure, return an empty history, `null` current rate, direction `unknown`, and a warning beginning `Текущая ставка Банка России недоступна:`.

Forecast branch:

- Download the forecast page, discover and download the workbook, then parse it.
- If `published_at` is older than `CBR_FORECAST_MAX_AGE_DAYS`, set forecast to `null` and add `Прогноз Банка России устарел`.
- On HTTP, workbook, or parsing failure, set forecast to `null` and add a warning beginning `Прогноз Банка России недоступен:`.

Redis failures are non-fatal. A failed read behaves as a miss; a failed write does not change the HTTP result. The service must not use expired cache data as fallback.

## Redis Cache

- Reuse the existing `cache.Cache` abstraction and Redis implementation.
- Cache key: `cbr:rates:snapshot`.
- TTL: `CBR_CACHE_TTL`, default `24h`.
- Cache only complete snapshots whose `warnings` list is empty.
- Do not cache partial, unavailable, or stale results.
- No stale-cache fallback is allowed after TTL expiry.

## Configuration

The gateway must support:

| Variable | Default | Purpose |
|---|---|---|
| `CBR_KEY_RATE_URL` | `https://www.cbr.ru/hd_base/KeyRate/` | Key-rate history page |
| `CBR_FORECAST_PAGE_URL` | `https://www.cbr.ru/statistics/ddkp/mo_br/` | Forecast landing page |
| `HTTP_TIMEOUT` | `10s` | Per-request timeout |
| `CBR_RATE_MAX_AGE_DAYS` | `7` | Maximum age of current-rate data |
| `CBR_FORECAST_MAX_AGE_DAYS` | `120` | Maximum age of forecast data |
| `CBR_CACHE_TTL` | `24h` | Complete-snapshot cache TTL |

Configuration loading must reject non-positive durations and negative maximum-age values.

## Error Handling

- Parser and transport errors must retain enough wrapped context for logs and tests.
- Expected upstream failures are converted by the service into compatible warnings rather than endpoint-level errors.
- Context cancellation must stop outstanding work and remain distinguishable from upstream unavailability.
- Redis errors must not prevent a fresh CBR response.
- Internal errors and upstream response bodies must not leak through the HTTP error response.

## Testing Strategy

### Required

- HTTP client unit tests for all three requests, paths, headers, success, non-2xx responses, timeout, context cancellation, and response-size limits.
- Parser unit tests for key-rate extraction, sorting, direction, numeric normalization, workbook-link discovery, both XLSX formats, and malformed payloads.
- Service unit tests for full success, independent source failures, stale rate, stale forecast, compatible warnings, injected clock, Redis hit/miss, 24-hour TTL, non-fatal Redis errors, and exclusion of partial snapshots from cache.
- Application/handler contract test proving production-compatible `GET /v1/cbr/rates` JSON.
- `go test ./...` passes.
- `go vet ./...` passes.

### Recommended, not required for acceptance

- End-to-end integration test covering HTTP, parsing, service, Redis, and Fiber handler.
- `go test -race ./...`.
- At least 80% statement coverage in `internal/cbr`.

## Documentation

- Update README to show `GET /v1/cbr/rates` as implemented instead of returning `501 Not Implemented`.
- Update Swagger artifacts if generated schemas or documented responses differ from the current artifacts.
- Update `go-migration.md` after implementation and verification, not as part of this design-only change.

## Acceptance Criteria

1. Production wiring uses the real CBR service instead of `StubService`.
2. With both sources available, the endpoint returns a compatible complete snapshot and caches it for 24 hours.
3. With one source unavailable or stale, the endpoint returns usable data from the other source with Python-compatible warning semantics and does not cache the result.
4. Both XLSX formats handled by the Python service are supported.
5. Redis unavailability does not make the CBR endpoint unavailable.
6. No stale value is served after cache expiry.
7. All required tests, `go test ./...`, and `go vet ./...` pass.
