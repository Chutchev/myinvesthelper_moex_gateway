# Moex Gateway Scaffold Design

## Goal

Create a compilable Go HTTP gateway scaffold that provides clear boundaries for
moving the MOEX and CBR external-request code described in
`myinvesthelper_backend/docs/go-migration.md`. The scaffold must not implement
external HTTP requests, parsing, caching, or migration business logic yet.

## Architecture

Code is grouped by domain so MOEX and CBR can be implemented and tested
independently. HTTP transport is isolated from domain services, and application
wiring is isolated from both. The Python backend remains responsible for
database access and bond-sync orchestration.

```text
moex-gateway/
├── cmd/gateway/main.go
├── internal/
│   ├── app/app.go
│   ├── config/config.go
│   ├── httpserver/
│   │   ├── router.go
│   │   ├── health_handler.go
│   │   ├── moex_handler.go
│   │   └── cbr_handler.go
│   ├── moex/
│   │   ├── client.go
│   │   ├── service.go
│   │   └── types.go
│   ├── cbr/
│   │   ├── client.go
│   │   ├── service.go
│   │   └── types.go
│   └── cache/cache.go
├── .env.example
├── Makefile
├── README.md
├── go.mod
└── go.sum
```

## Components

- `cmd/gateway` loads configuration, constructs the application, and starts it.
- `internal/app` owns dependency wiring and HTTP server lifecycle.
- `internal/httpserver` defines routes, JSON responses, validation boundaries,
  and handler-to-service calls.
- `internal/moex` contains the MOEX client boundary, service API, and response
  types corresponding to bond details and the market universe.
- `internal/cbr` contains the CBR client boundary, service API, and response
  types corresponding to rate history, direction, and forecast data.
- `internal/cache` defines the cache interface needed later by MOEX. It does not
  select or implement Redis or another external cache.
- `internal/config` contains environment-backed server and upstream settings.

Existing uncommitted client, config, and repository work must be preserved.
Useful definitions may be moved into the new packages, but unrelated behavior
must not be discarded or overwritten.

## HTTP Contract

The router exposes these initial endpoints:

- `GET /health`
- `GET /v1/bonds/{isin}`
- `GET /v1/bonds?limit=40`
- `GET /v1/cbr/rates`

Health returns a successful JSON response. Domain endpoints call typed service
interfaces. Until domain behavior is implemented, service methods return a
shared `ErrNotImplemented`, and handlers map it to HTTP 501 with a stable JSON
error shape.

## Data Flow and Errors

Handlers parse path and query input, call a domain service interface, and encode
the result. Domain services will eventually coordinate clients and caches, but
the scaffold contains no outbound requests. Invalid input maps to HTTP 400,
`ErrNotImplemented` maps to HTTP 501, and unexpected errors map to HTTP 500.

## Dependencies

Use only the Go standard library in the scaffold. Add `x/net/html`, `excelize`,
and `errgroup` when their corresponding implementation is introduced. This
keeps `go.mod` aligned with code that actually exists.

## Verification

- Format all Go sources with `gofmt`.
- Run `go test ./...`; lightweight tests cover route registration, health, and
  the not-implemented response boundary.
- Run `go vet ./...`.
- Run `go build ./...`.

## Out of Scope

- MOEX or CBR network requests.
- HTML, JSON, or Excel parsing.
- In-memory or Redis cache implementations.
- Concurrent MOEX fan-out.
- Python integration changes.
- Database access or bond-sync orchestration.
