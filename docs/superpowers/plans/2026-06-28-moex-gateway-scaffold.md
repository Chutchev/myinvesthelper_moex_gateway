# Moex Gateway Scaffold Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a compilable standard-library Go HTTP gateway scaffold for the MOEX and CBR migrations without implementing outbound requests, parsing, caching, or business logic.

**Architecture:** Organize code by MOEX and CBR domains, expose each through typed service interfaces, and keep HTTP routing and application lifecycle in separate packages. Preserve the useful configuration and storage-interface work already present in the working tree while replacing the temporary layer-based package locations with the approved domain layout.

**Tech Stack:** Go 1.22, `net/http`, `encoding/json`, `context`, Go testing package

---

## File Map

- Create `cmd/gateway/main.go`: process entry point and dependency construction.
- Delete `cmd/app/main.go`: superseded temporary entry point.
- Replace `internal/app/App.go` with `internal/app/app.go`: HTTP server lifecycle.
- Modify `internal/config/config.go`: retain environment loading, remove premature Redis configuration, and expose a server address helper.
- Create `internal/config/config_test.go`: configuration parsing coverage.
- Create `internal/apperrors/errors.go`: shared `ErrNotImplemented` sentinel.
- Create `internal/cache/cache.go`: preserved cache contract from `internal/repository/storage.go`.
- Delete `internal/repository/storage.go`: superseded package location.
- Create `internal/moex/client.go`: MOEX upstream boundary and no-op client constructor.
- Create `internal/moex/service.go`: MOEX handler-facing service contract and stub.
- Create `internal/moex/types.go`: bond response contracts.
- Delete `internal/clients/moex.go`: superseded package location.
- Create `internal/cbr/client.go`: CBR upstream boundary and no-op client constructor.
- Create `internal/cbr/service.go`: CBR handler-facing service contract and stub.
- Create `internal/cbr/types.go`: rate response contracts.
- Delete `internal/clients/cbr.go`: superseded package location.
- Create `internal/httpserver/router.go`: route assembly and common JSON errors.
- Create `internal/httpserver/health_handler.go`: liveness endpoint.
- Create `internal/httpserver/moex_handler.go`: MOEX endpoint adapters.
- Create `internal/httpserver/cbr_handler.go`: CBR endpoint adapter.
- Create `internal/httpserver/router_test.go`: route, validation, and 501 behavior tests.
- Create `.env.example`: documented runtime variables.
- Create `Makefile`: standard local commands.
- Create `README.md`: project purpose, structure, and commands.
- Create `go.sum`: intentionally empty until external dependencies are introduced.

### Task 1: Configuration boundary

**Files:**
- Modify: `internal/config/config.go`
- Create: `internal/config/config_test.go`

- [ ] **Step 1: Write failing configuration tests**

Create table-driven tests that set `SERVER_HOST`, `SERVER_PORT`, `MOEX_BASE_URL`,
`CBR_KEY_RATE_URL`, `CBR_FORECAST_PAGE_URL`, `HTTP_TIMEOUT`, and
`MARKET_CACHE_TTL`, call `Load`, and assert exact parsed values. Add invalid
duration cases for both duration variables.

```go
func TestLoad(t *testing.T) {
	t.Setenv("SERVER_HOST", "127.0.0.1")
	t.Setenv("SERVER_PORT", "9090")
	t.Setenv("MOEX_BASE_URL", "https://moex.test")
	t.Setenv("CBR_KEY_RATE_URL", "https://cbr.test/rate")
	t.Setenv("CBR_FORECAST_PAGE_URL", "https://cbr.test/forecast")
	t.Setenv("HTTP_TIMEOUT", "3s")
	t.Setenv("MARKET_CACHE_TTL", "15m")

	cfg, err := Load()
	if err != nil { t.Fatal(err) }
	if cfg.Server.Address() != "127.0.0.1:9090" { t.Fatalf("address = %q", cfg.Server.Address()) }
	if cfg.MOEXBaseURL != "https://moex.test" { t.Fatalf("MOEXBaseURL = %q", cfg.MOEXBaseURL) }
	if cfg.CBRKeyRateURL != "https://cbr.test/rate" { t.Fatalf("CBRKeyRateURL = %q", cfg.CBRKeyRateURL) }
	if cfg.CBRForecastPageURL != "https://cbr.test/forecast" { t.Fatalf("CBRForecastPageURL = %q", cfg.CBRForecastPageURL) }
	if cfg.HTTPTimeout != 3*time.Second { t.Fatalf("HTTPTimeout = %s", cfg.HTTPTimeout) }
	if cfg.MarketCacheTTL != 15*time.Minute { t.Fatalf("MarketCacheTTL = %s", cfg.MarketCacheTTL) }
}
```

- [ ] **Step 2: Run the tests and verify failure**

Run: `go test ./internal/config`

Expected: compilation fails because the new fields and `Address` do not exist.

- [ ] **Step 3: Implement the minimal configuration API**

Keep `getEnv` and duration error wrapping. Define:

```go
type Config struct {
	Server             ServerConfig
	MOEXBaseURL        string
	CBRKeyRateURL      string
	CBRForecastPageURL string
	HTTPTimeout        time.Duration
	MarketCacheTTL     time.Duration
	LogLevel           string
}

type ServerConfig struct { Host, Port string }

func (c ServerConfig) Address() string { return net.JoinHostPort(c.Host, c.Port) }
```

Use production defaults from the Python migration source and a `15m` market
cache TTL. Do not keep the unused Redis fields.

- [ ] **Step 4: Run tests and format**

Run: `gofmt -w internal/config && go test ./internal/config`

Expected: PASS.

### Task 2: Domain contracts and stubs

**Files:**
- Create: `internal/apperrors/errors.go`
- Create: `internal/cache/cache.go`
- Create: `internal/moex/client.go`
- Create: `internal/moex/service.go`
- Create: `internal/moex/types.go`
- Create: `internal/cbr/client.go`
- Create: `internal/cbr/service.go`
- Create: `internal/cbr/types.go`
- Delete: `internal/clients/moex.go`
- Delete: `internal/clients/cbr.go`
- Delete: `internal/repository/storage.go`

- [ ] **Step 1: Add a compile-time contract test for service stubs**

Create `internal/httpserver/contracts_test.go` temporarily with compile-time
assignments. It will remain as protection against wiring drift:

```go
var _ moex.Service = moex.NewStubService()
var _ cbr.Service = cbr.NewStubService()
```

- [ ] **Step 2: Run the test and verify failure**

Run: `go test ./internal/httpserver`

Expected: compilation fails because the domain packages do not exist.

- [ ] **Step 3: Add shared error and cache boundary**

```go
// internal/apperrors/errors.go
var ErrNotImplemented = errors.New("not implemented")

// internal/cache/cache.go
type Cache interface {
	Get(context.Context, string) ([]byte, error)
	Set(context.Context, string, []byte, time.Duration) error
}
```

This preserves the existing `repository.Storage` intent under the approved
package name.

- [ ] **Step 4: Add typed MOEX contracts**

Define `Cashflow`, `Bond`, and `MarketUniverse` JSON types in `types.go`. `Bond`
must include the Python consumer fields: ISIN, SECID, ticker, names, type data,
issuer and registration data, issue/maturity dates, face values, currency,
issue size, coupon data, listing/default flags, price/yield/duration, accrued
interest, liquidity data, market timestamp, coupon calendar, and cashflow
schedule. Use pointer scalar fields where the upstream value can be absent.

Define in `service.go`:

```go
type Service interface {
	Bond(context.Context, string) (Bond, error)
	MarketUniverse(context.Context, int) ([]Bond, error)
}

type StubService struct{}
func NewStubService() *StubService { return &StubService{} }
```

Both methods return zero values and `apperrors.ErrNotImplemented`.

Define in `client.go` an upstream `Client` interface with `FetchUniverse`,
`FetchDescription`, `FetchMarketData`, and `FetchBondization`, each returning
raw bytes. Add an `HTTPClient` constructor that retains base URL and
`*http.Client`; its methods return `ErrNotImplemented` without issuing requests.

- [ ] **Step 5: Add typed CBR contracts**

Define `Direction`, its `up`, `flat`, `down`, and `unknown` constants,
`RatePoint`, `RateForecast`, and `RateSnapshot` with JSON tags matching the
Python dataclasses.

Define in `service.go`:

```go
type Service interface { Snapshot(context.Context) (RateSnapshot, error) }
type StubService struct{}
func NewStubService() *StubService { return &StubService{} }
```

`Snapshot` returns a zero value and `apperrors.ErrNotImplemented`.

Define in `client.go` an upstream `Client` interface with `FetchKeyRatePage`,
`FetchForecastPage`, and `FetchForecastWorkbook`. Add an `HTTPClient`
constructor and non-networking methods that return `ErrNotImplemented`.

- [ ] **Step 6: Remove superseded temporary packages and verify contracts**

Remove only the three migrated files under `internal/clients` and
`internal/repository`; remove empty directories afterward. Preserve their
configuration/cache/client intent in the new packages.

Run: `gofmt -w internal && go test ./internal/httpserver`

Expected: PASS.

### Task 3: HTTP API scaffold

**Files:**
- Create: `internal/httpserver/router.go`
- Create: `internal/httpserver/health_handler.go`
- Create: `internal/httpserver/moex_handler.go`
- Create: `internal/httpserver/cbr_handler.go`
- Create: `internal/httpserver/router_test.go`

- [ ] **Step 1: Write route behavior tests**

Use `httptest` and fake services. Cover:

```go
func TestHealth(t *testing.T)                         // 200, {"status":"ok"}
func TestBondRejectsInvalidISIN(t *testing.T)        // 400
func TestBondNotImplemented(t *testing.T)            // 501
func TestUniverseRejectsInvalidLimit(t *testing.T)   // 400 for zero/non-number
func TestUniverseUsesDefaultLimit(t *testing.T)      // service receives 40
func TestCBRNotImplemented(t *testing.T)             // 501
```

The fake MOEX service records the limit and otherwise returns
`apperrors.ErrNotImplemented`; the fake CBR service returns the same sentinel.

- [ ] **Step 2: Run the route tests and verify failure**

Run: `go test ./internal/httpserver -run 'Test(Health|Bond|Universe|CBR)'`

Expected: compilation fails because `NewRouter` does not exist.

- [ ] **Step 3: Implement router and response helpers**

Use Go 1.22 method-aware patterns:

```go
mux.HandleFunc("GET /health", healthHandler)
mux.HandleFunc("GET /v1/bonds/{isin}", handlers.bond)
mux.HandleFunc("GET /v1/bonds", handlers.universe)
mux.HandleFunc("GET /v1/cbr/rates", handlers.rates)
```

Return JSON with `Content-Type: application/json`. Use a stable error body:

```go
type errorResponse struct { Error string `json:"error"` }
```

Map invalid input to 400, `ErrNotImplemented` to 501, and other service errors
to 500. Validate ISIN with exactly 12 uppercase ASCII letters or digits. Accept
limits from 1 through 200 and default to 40.

- [ ] **Step 4: Run route tests and format**

Run: `gofmt -w internal/httpserver && go test ./internal/httpserver`

Expected: PASS.

### Task 4: Application lifecycle and executable

> **User override from Task 4 onward:** Do not create or run tests. Verify Tasks
> 4 and 5 only with formatting, vet, build, and diff checks.

**Files:**
- Replace: `internal/app/App.go` with `internal/app/app.go`
- Create: `cmd/gateway/main.go`
- Delete: `cmd/app/main.go`

- [ ] **Step 1: Implement lifecycle wiring**

`app.New` constructs the stub MOEX and CBR services, router, and `http.Server`.
Expose `Handler()` as the HTTP boundary. Implement `Run(ctx)` with
`ListenAndServe`, a goroutine waiting on context cancellation, and a five-second
graceful shutdown timeout. Treat `http.ErrServerClosed` as success.

`cmd/gateway/main.go` loads config, creates a signal-aware context for SIGINT and
SIGTERM, runs the app, and logs fatal startup/runtime errors. It contains no
domain logic.

- [ ] **Step 2: Remove the obsolete entry point and verify**

Run:

```bash
gofmt -w cmd internal/app
go vet ./...
go build ./...
git diff --check
```

Expected: formatting completes and vet, build, and diff checks exit 0.

### Task 5: Developer-facing project scaffold

**Files:**
- Create: `.env.example`
- Create: `Makefile`
- Create: `README.md`
- Create: `go.sum`

- [ ] **Step 1: Add environment template**

Document only variables consumed by `config.Load`: server host/port, MOEX base
URL, both CBR URLs, HTTP timeout, market cache TTL, and log level. Do not include
secrets or Redis settings.

- [ ] **Step 2: Add standard commands**

Make targets must be `run`, `test`, `vet`, `build`, and `fmt`, each invoking the
corresponding Go command. Mark all targets phony and set `build` as the default
goal. The `test` command is documented but is not run in this task.

- [ ] **Step 3: Add concise README**

Document purpose, current non-implemented status, route table, package tree,
environment setup, and the Make targets. Explicitly state that Python keeps DB
and orchestration responsibilities.

- [ ] **Step 4: Run complete verification**

Run:

```bash
gofmt -w cmd internal
go vet ./...
go build ./...
git diff --check
```

Expected: every command exits 0 without running tests. Confirm with
`git status --short` that pre-existing user changes were migrated into the
approved locations and no unrelated files changed.
