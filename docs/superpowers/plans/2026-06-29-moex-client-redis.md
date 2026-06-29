# MOEX Client, Parsing, and Redis Cache Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the MOEX stubs with a production HTTP client, response parsers, liquidity-filtering service, and Redis-backed 15-minute cache.

**Architecture:** Keep transport, parsing, orchestration, and cache concerns separate. `moex.HTTPClient` returns raw ISS payloads, parser functions map MOEX column/data blocks into the existing domain types, and `moex.CachedService` composes the client and `cache.Cache`; `app.New` wires production dependencies while handlers continue depending only on `moex.Service`.

**Tech Stack:** Go 1.25, standard `net/http` and `encoding/json`, `golang.org/x/sync/errgroup`, `github.com/redis/go-redis/v9`, `github.com/alicebob/miniredis/v2` for Redis tests, Fiber v3.

## Progress

- [x] Task 1: Error and configuration contracts — committed `637faba`
- [x] Task 2: Redis JSON cache — committed `5adec51`
- [x] Task 3: MOEX HTTP client — committed `dcafe81`
- [ ] Task 4: MOEX JSON parsers (parser.go, parser_test.go) — **pending**
- [x] Task 5: Cached single-bond service — committed `02e0d29`
- [x] Task 6: Liquid market universe — committed `4c71a2b`

## Global Constraints

- Keep Go at version `1.25.0` or newer.
- Use `MOEX_BASE_URL`, default `https://iss.moex.com/iss` (the approved spec says `/2`, but every documented endpoint is an ISS path and the repository/Python source already use `/iss`; preserve the working repository default).
- Use `HTTP_TIMEOUT`, default `10s`, for description, market-data, and bondization requests; use `20s` for universe requests.
- Send `User-Agent: myinvesthelper-gateway/1.0` on every MOEX request.
- Retry an HTTP 5xx response once; never retry 4xx, context cancellation, timeout, or other network errors.
- Use `REDIS_ADDR=localhost:6379`, empty `REDIS_PASSWORD`, and `REDIS_DB=0` as defaults.
- Configure Redis dial timeout to `5s` and read/write timeouts to `3s`.
- Cache both `moex:bond:{isin}` and `moex:universe:{limit}` for `MARKET_CACHE_TTL`, default `15m`.
- A liquid universe row must have `FACEUNIT` in `RUB` or `SUR`, `VALTODAY > 1_000_000`, `NUMTRADES > 10`, and a non-null `LAST` or `LCURRENTPRICE`.
- Sort liquid candidates by `VALTODAY` descending, then preserve that order in the concurrent result.
- Limit full-bond fan-out to eight simultaneous calls with `errgroup.WithContext` and a buffered semaphore.
- Preserve MOEX dates as `YYYY-MM-DD` strings; do not introduce holiday or workday adjustment in this scope.
- Keep CBR as `cbr.StubService`; do not add handlers, protobuf, persistence, or Python changes.
- New code must pass `go test ./...`, `go vet ./...`, and reach at least 80% statement coverage in `internal/moex` and `internal/cache`.

---

## File Map

- Modify `go.mod` / `go.sum`: add Redis and test-only miniredis dependencies; make `x/sync` direct.
- Modify `.env.example`: document Redis settings already consumed by the gateway.
- Modify `internal/apperrors/errors.go`: define stable sentinel errors used with `errors.Is`.
- Modify `internal/config/config.go` and `internal/config/config_test.go`: load and validate Redis configuration.
- Modify `internal/cache/cache.go`: adopt the typed JSON cache contract from the approved design and add `Delete`.
- Create `internal/cache/redis.go` and `internal/cache/redis_test.go`: implement and test Redis JSON storage.
- Modify `internal/moex/client.go` and `internal/moex/client_test.go`: implement URL construction, timeouts, retries, headers, and error classification.
- Create `internal/moex/parser.go` and `internal/moex/parser_test.go`: decode all four MOEX response shapes.
- Modify `internal/moex/service.go` and `internal/moex/service_test.go`: implement cached bond assembly and universe fan-out.
- Modify `internal/app/app.go` and `internal/app/app_test.go`: wire the real MOEX service and retain injectable construction for tests.
- Create `internal/moex/integration_test.go`: verify HTTP -> parse -> service -> Redis -> cached response.

### Task 1: Error and configuration contracts

**Files:**
- Modify: `internal/apperrors/errors.go`
- Modify: `internal/config/config.go`
- Modify: `internal/config/config_test.go`
- Modify: `.env.example`

**Interfaces:**
- Produces: `apperrors.ErrHTTPError`, `ErrTimeout`, `ErrParseError`, `ErrCacheMiss`, and `ErrCacheError` sentinels.
- Produces: `Config.RedisAddr string`, `Config.RedisPassword string`, and `Config.RedisDB int`.

- [ ] **Step 1: Extend failing configuration tests**

Add Redis variables to `TestLoadReadsGatewayConfiguration`, assert their values, and add a table case proving that `REDIS_DB=not-an-int` returns an error containing `invalid REDIS_DB`:

```go
t.Setenv("REDIS_ADDR", "redis.test:6380")
t.Setenv("REDIS_PASSWORD", "test-password")
t.Setenv("REDIS_DB", "2")

if got, want := cfg.RedisAddr, "redis.test:6380"; got != want {
	t.Errorf("RedisAddr = %q, want %q", got, want)
}
if got, want := cfg.RedisPassword, "test-password"; got != want {
	t.Errorf("RedisPassword = %q, want %q", got, want)
}
if got, want := cfg.RedisDB, 2; got != want {
	t.Errorf("RedisDB = %d, want %d", got, want)
}
```

- [ ] **Step 2: Run the focused test and confirm failure**

Run: `go test ./internal/config -run 'TestLoad' -v`

Expected: FAIL because the Redis fields do not exist and invalid `REDIS_DB` is not parsed.

- [ ] **Step 3: Add error sentinels and Redis configuration**

Define errors as package-level sentinels so callers can wrap context while retaining `errors.Is` behavior:

```go
var (
	ErrNotImplemented = errors.New("not implemented")
	ErrHTTPError      = errors.New("HTTP request failed")
	ErrTimeout        = errors.New("request timeout")
	ErrParseError     = errors.New("failed to parse response")
	ErrCacheMiss      = errors.New("cache miss")
	ErrCacheError     = errors.New("cache operation failed")
)
```

In `config.Load`, parse the database before constructing `Config`:

```go
redisDB, err := strconv.Atoi(getEnv("REDIS_DB", "0"))
if err != nil || redisDB < 0 {
	return nil, fmt.Errorf("invalid REDIS_DB: must be a non-negative integer")
}
```

Add `RedisAddr`, `RedisPassword`, and `RedisDB` to `Config`, populate them with the specified defaults, and append these exact lines to `.env.example`:

```dotenv
REDIS_ADDR=localhost:6379
REDIS_PASSWORD=
REDIS_DB=0
```

- [ ] **Step 4: Verify and commit**

Run: `gofmt -w internal/apperrors/errors.go internal/config/config.go internal/config/config_test.go && go test ./internal/config ./internal/apperrors`

Expected: PASS.

```bash
git add .env.example internal/apperrors/errors.go internal/config/config.go internal/config/config_test.go
git commit -m "feat: add MOEX cache configuration"
```

### Task 2: Redis JSON cache

**Files:**
- Modify: `go.mod`
- Modify: `go.sum`
- Modify: `internal/cache/cache.go`
- Create: `internal/cache/redis.go`
- Create: `internal/cache/redis_test.go`

**Interfaces:**
- Produces: `type Cache interface { Get(context.Context, string, any) error; Set(context.Context, string, any, time.Duration) error; Delete(context.Context, string) error }`.
- Produces: `NewRedisCache(addr, password string, db int) *RedisCache`.
- Produces: cache misses wrapping `apperrors.ErrCacheMiss`; Redis and JSON failures wrapping `apperrors.ErrCacheError`.

- [ ] **Step 1: Add dependencies and write failing Redis tests**

Run:

```bash
go get github.com/redis/go-redis/v9@v9.0.5
go get -t github.com/alicebob/miniredis/v2
```

Create tests using one `miniredis.RunT(t)` instance per test. Cover round-trip JSON, TTL, missing key classification, malformed cached JSON, delete, and unreachable Redis:

```go
func TestRedisCacheRoundTripAndTTL(t *testing.T) {
	server := miniredis.RunT(t)
	cache := NewRedisCache(server.Addr(), "", 0)
	want := sample{ISIN: "RU000A10ABC1", Price: 101.5}

	if err := cache.Set(context.Background(), "bond", want, 15*time.Minute); err != nil {
		t.Fatal(err)
	}
	var got sample
	if err := cache.Get(context.Background(), "bond", &got); err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("Get() = %#v, want %#v", got, want)
	}
	if gotTTL := server.TTL("bond"); gotTTL != 15*time.Minute {
		t.Fatalf("TTL = %v, want 15m", gotTTL)
	}
}

func TestRedisCacheMiss(t *testing.T) {
	server := miniredis.RunT(t)
	err := NewRedisCache(server.Addr(), "", 0).Get(context.Background(), "missing", &sample{})
	if !errors.Is(err, apperrors.ErrCacheMiss) {
		t.Fatalf("Get() error = %v, want ErrCacheMiss", err)
	}
}
```

- [ ] **Step 2: Run tests and confirm failure**

Run: `go test ./internal/cache -v`

Expected: FAIL because `RedisCache` and the typed cache methods do not exist.

- [ ] **Step 3: Replace the cache contract and implement RedisCache**

Use JSON at the adapter boundary, not in the MOEX service:

```go
type Cache interface {
	Get(ctx context.Context, key string, value any) error
	Set(ctx context.Context, key string, value any, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
}

type RedisCache struct {
	client *redis.Client
}

func NewRedisCache(addr, password string, db int) *RedisCache {
	return &RedisCache{client: redis.NewClient(&redis.Options{
		Addr: addr, Password: password, DB: db,
		DialTimeout: 5 * time.Second,
		ReadTimeout: 3 * time.Second,
		WriteTimeout: 3 * time.Second,
	})}
}

func (c *RedisCache) Get(ctx context.Context, key string, value any) error {
	payload, err := c.client.Get(ctx, key).Bytes()
	if errors.Is(err, redis.Nil) {
		return fmt.Errorf("%w: %s", apperrors.ErrCacheMiss, key)
	}
	if err != nil {
		return fmt.Errorf("%w: get %s: %v", apperrors.ErrCacheError, key, err)
	}
	if err := json.Unmarshal(payload, value); err != nil {
		return fmt.Errorf("%w: decode %s: %v", apperrors.ErrCacheError, key, err)
	}
	return nil
}

func (c *RedisCache) Set(ctx context.Context, key string, value any, ttl time.Duration) error {
	payload, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("%w: encode %s: %v", apperrors.ErrCacheError, key, err)
	}
	if err := c.client.Set(ctx, key, payload, ttl).Err(); err != nil {
		return fmt.Errorf("%w: set %s: %v", apperrors.ErrCacheError, key, err)
	}
	return nil
}

func (c *RedisCache) Delete(ctx context.Context, key string) error {
	if err := c.client.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("%w: delete %s: %v", apperrors.ErrCacheError, key, err)
	}
	return nil
}
```

- [ ] **Step 4: Verify and commit**

Run: `gofmt -w internal/cache && go test ./internal/cache -cover`

Expected: PASS with at least 80% statement coverage.

```bash
git add go.mod go.sum internal/cache
git commit -m "feat: implement Redis JSON cache"
```

### Task 3: MOEX HTTP client

**Files:**
- Modify: `internal/moex/client.go`
- Modify: `internal/moex/client_test.go`

**Interfaces:**
- Preserves: the four existing `Client` methods.
- Produces: `NewHTTPClient(baseURL string, timeout time.Duration) *HTTPClient`.
- Internal test seam: `newHTTPClient(baseURL string, client *http.Client, timeout time.Duration) *HTTPClient`.

- [ ] **Step 1: Replace the constructor-only test with table-driven failing transport tests**

Use `httptest.Server` and cover each endpoint path, `User-Agent`, successful body return, and the universe `limit` query. The universe request should ask MOEX for the columns needed by parsing/filtering and pass `limit` as a query parameter:

```go
tests := []struct {
	name string
	call func(context.Context, Client) ([]byte, error)
	path string
}{
	{"universe", func(ctx context.Context, c Client) ([]byte, error) { return c.FetchUniverse(ctx, 40) }, "/engines/stock/markets/bonds/boards/TQCB/securities.json"},
	{"description", func(ctx context.Context, c Client) ([]byte, error) { return c.FetchDescription(ctx, "RU000A10ABC1") }, "/securities/RU000A10ABC1.json"},
	{"market data", func(ctx context.Context, c Client) ([]byte, error) { return c.FetchMarketData(ctx, "RU000A10ABC1") }, "/engines/stock/markets/bonds/securities/RU000A10ABC1.json"},
	{"bondization", func(ctx context.Context, c Client) ([]byte, error) { return c.FetchBondization(ctx, "RU000A10ABC1") }, "/statistics/engines/stock/markets/bonds/bondization/RU000A10ABC1.json"},
}
```

Add separate tests for: 404 performs one attempt and wraps `ErrHTTPError` with status; 500 performs exactly two attempts; second-attempt success returns the body; a slow request wraps `ErrTimeout`; canceled context remains `context.Canceled`; an unreachable address returns a wrapped network error that is neither `ErrTimeout` nor `ErrHTTPError`; a response larger than the chosen 16 MiB limit fails safely.

- [ ] **Step 2: Run the focused suite and confirm failure**

Run: `go test ./internal/moex -run 'TestHTTPClient' -v`

Expected: FAIL because all fetch methods still return `ErrNotImplemented`.

- [ ] **Step 3: Implement the shared request path**

Store a trimmed base URL, transport, and configured per-bond timeout. Keep `http.Client.Timeout` at zero so the 20-second universe context is not cut off by the 10-second default:

```go
const (
	userAgent = "myinvesthelper-gateway/1.0"
	universeTimeout = 20 * time.Second
	maxResponseBytes = 16 << 20
)

type HTTPClient struct {
	baseURL string
	client *http.Client
	timeout time.Duration
}

func NewHTTPClient(baseURL string, timeout time.Duration) *HTTPClient {
	return newHTTPClient(baseURL, &http.Client{}, timeout)
}

func newHTTPClient(baseURL string, client *http.Client, timeout time.Duration) *HTTPClient {
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	return &HTTPClient{baseURL: strings.TrimRight(baseURL, "/"), client: client, timeout: timeout}
}
```

Implement `get(ctx, path string, query url.Values, timeout time.Duration)` to create a fresh request for each of at most two attempts, set the header, classify context deadline as `ErrTimeout`, preserve `context.Canceled`, retry only 5xx, close every response body, and use `io.LimitReader(response.Body, maxResponseBytes+1)` before rejecting an oversized payload. Wrap HTTP status as `fmt.Errorf("%w: status %d", apperrors.ErrHTTPError, response.StatusCode)`.

For `FetchUniverse`, send these exact query values so the parser receives all required columns:

```go
query.Set("limit", strconv.Itoa(limit))
query.Set("securities.columns", "SECID,SHORTNAME,LOTSIZE,FACEVALUE,FACEUNIT,MATDATE,COUPONPERCENT,COUPONVALUE,ISSUESIZE")
query.Set("marketdata.columns", "SECID,LAST,LCURRENTPRICE,YIELD,DURATION,ACCRUEDINT,VALTODAY,NUMTRADES,SYSTIME")
```

- [ ] **Step 4: Verify and commit**

Run: `gofmt -w internal/moex/client.go internal/moex/client_test.go && go test ./internal/moex -run 'TestHTTPClient' -race`

Expected: PASS.

```bash
git add internal/moex/client.go internal/moex/client_test.go
git commit -m "feat: implement MOEX HTTP client"
```

### Task 4: MOEX JSON parsers

**Files:**
- Create: `internal/moex/parser.go`
- Create: `internal/moex/parser_test.go`

**Interfaces:**
- Produces: `ParseUniverseResponse([]byte) (MarketUniverse, error)`.
- Produces: `ParseBondDescription([]byte) (Bond, error)`.
- Produces: `ParseMarketData([]byte) (*BondMarketData, error)` and a new internal `BondMarketData` struct containing market and session fields.
- Produces: `ParseBondization([]byte) ([]Coupon, []Cashflow, error)`.

- [ ] **Step 1: Write parser fixtures and failing tests**

Use small inline JSON fixtures whose `columns` order differs from the logical field order. Assert universe joins `securities` and `marketdata` by `SECID`, not row position; description maps every field currently present on `Bond`; market data uses fallbacks `LAST/LCURRENTPRICE`, `ACCRUEDINT/ACCINT`, `SYSTIME/UPDATETIME`, and `CURRENCYID/FACEUNIT`; an empty market-data block returns `(nil, nil)`; bondization produces taxable coupon and non-taxable principal cashflows sorted by `(date, kind)`.

Include conversion cases for JSON numbers, numeric strings, booleans (`true`, `1`, `0`, `"yes"`, `"no"`), null, and empty strings. Include malformed JSON, missing required blocks, short rows, and a non-convertible populated numeric field; each malformed structural/conversion case must wrap `apperrors.ErrParseError` rather than panic.

Define typed test helpers locally (`stringValue`, `floatValue`, `intValue`, and `boolValue`) that fail the test when a required pointer is nil. Representative expectations:

```go
if got[0].ISIN != "RU000A10ABC1" || floatValue(t, got[0].Price) != 101.25 || floatValue(t, got[0].ValueToday) != 2_500_000 {
	t.Fatalf("unexpected universe bond: %#v", got[0])
}
if stringValue(t, bond.CouponType) != "floating" {
	t.Fatalf("CouponType = %v, want floating", bond.CouponType)
}
if cashflows[0].Kind != "coupon" || !cashflows[0].Taxable || cashflows[1].Kind != "principal" || cashflows[1].Taxable {
	t.Fatalf("cashflows = %#v", cashflows)
}
```

- [ ] **Step 2: Run parser tests and confirm failure**

Run: `go test ./internal/moex -run 'TestParse|TestTo|TestIndexed|TestPick' -v`

Expected: FAIL because `parser.go` does not exist.

- [ ] **Step 3: Implement generic block decoding and strict helpers**

Decode blocks through a reusable shape:

```go
type responseBlock struct {
	Columns []string `json:"columns"`
	Data [][]any `json:"data"`
}

func indexedRow(columns []string, row []any) map[string]any {
	result := make(map[string]any, len(columns))
	for index, column := range columns {
		if index < len(row) {
			result[column] = row[index]
		}
	}
	return result
}

func pickValue(row map[string]any, names ...string) any {
	for _, name := range names {
		if value, ok := row[name]; ok && value != nil && value != "" {
			return value
		}
	}
	return nil
}
```

Implement `toInt`, `toFloat`, and `toBool` with the specified pointer results. Add internal strict wrappers (`parseOptionalInt`, `parseOptionalFloat`, `parseOptionalBool`) that distinguish null/empty from invalid populated input and return `ErrParseError`; use those wrappers in exported parsers so corrupt MOEX data is not silently dropped.

- [ ] **Step 4: Map all domain fields and normalize values**

Description data is a property table where column 0 is the key and column 2 is the value. Map exact keys and fallbacks from the Python source: `GROUPNAME/GROUP`, `MATDATE/MATURITYDATE`, `FACEVALUE/INITIALFACEVALUE`, and `COUPONTYPE/COUPON_TYPE/COUPON_TYPE_NAME`. Normalize coupon type by recognizing substrings `перемен`, `плава`, `float`, `variable` as `floating`, and `фикс`, `постоян`, `fixed`, `constant` as `fixed`.

Define the parser-only merge shape:

```go
type BondMarketData struct {
	Price *float64
	YieldToMaturity *float64
	Duration *float64
	AccruedInterest *float64
	ValueToday *float64
	NumTrades *int
	MarketDataAsOf *string
	LotSize *int
	Currency *string
	FaceUnit *string
	MorningSession *bool
	EveningSession *bool
	WeekendSession *bool
}
```

For universe parsing, build `map[string]Bond` from the `securities` block and iterate market rows, emitting only rows that match a security. Set `ISIN`, `SECID`, and `Ticker` from `SECID`; do not filter liquidity in the parser.

For bondization, read coupon date from `COUPONDATE/DATE` and value from `VALUE/COUPONVALUE/LEGALCLOSEPRICE`; read principal date from `AMORTDATE/DATE/MATDATE` and value from `VALUE/AMORTVALUE/FACEVALUE/VALUEPRC`. Skip rows only when date or amount is absent; return a parse error for populated invalid amounts.

- [ ] **Step 5: Verify and commit**

Run: `gofmt -w internal/moex/parser.go internal/moex/parser_test.go && go test ./internal/moex -run 'TestParse|TestTo|TestIndexed|TestPick' -cover`

Expected: PASS.

```bash
git add internal/moex/parser.go internal/moex/parser_test.go
git commit -m "feat: parse MOEX ISS responses"
```

### Task 5: Cached single-bond service

**Files:**
- Modify: `internal/moex/service.go`
- Replace: `internal/moex/service_test.go`

**Interfaces:**
- Produces: `type CachedService struct { client Client; cache cache.Cache; ttl time.Duration }`.
- Produces: `NewService(client Client, cache cache.Cache, ttl time.Duration) *CachedService`.
- Preserves: `Service.Bond` and `Service.MarketUniverse`.

- [ ] **Step 1: Build reusable fakes and failing Bond tests**

Define concurrency-safe fake client/cache implementations in `service_test.go`; fake cache stores JSON bytes so tests also catch accidental pointer aliasing. Test:

- cache hit makes zero client calls;
- cache miss calls description, market data, and bondization exactly once each;
- parsed fields, session data, coupons, and cashflows merge into one `Bond`;
- cache key is `moex:bond:RU000A10ABC1` and TTL is the constructor TTL;
- description, market-data, bondization, parse, cache read, and cache write failures propagate;
- canceled context reaches the client and prevents subsequent fetches.

The successful assertion should verify that description owns static identity fields, market data owns prices/sessions, and bondization owns calendars:

```go
if got.ISIN != isin || stringValue(t, got.Name) != "Test Bond" || floatValue(t, got.Price) != 101.25 {
	t.Fatalf("Bond() = %#v", got)
}
if len(got.CouponCalendar) != 1 || len(got.CashflowSchedule) != 2 {
	t.Fatalf("calendars = %#v / %#v", got.CouponCalendar, got.CashflowSchedule)
}
```

- [ ] **Step 2: Run Bond tests and confirm failure**

Run: `go test ./internal/moex -run 'TestCachedServiceBond' -v`

Expected: FAIL because `CachedService` and `NewService` do not exist.

- [ ] **Step 3: Implement cache-first Bond assembly**

Use a default TTL only when a non-positive value is passed and classify only `ErrCacheMiss` as a refetch condition:

```go
const defaultCacheTTL = 15 * time.Minute

func (s *CachedService) Bond(ctx context.Context, isin string) (Bond, error) {
	key := "moex:bond:" + isin
	var cached Bond
	if err := s.cache.Get(ctx, key, &cached); err == nil {
		return cached, nil
	} else if !errors.Is(err, apperrors.ErrCacheMiss) {
		return Bond{}, fmt.Errorf("read bond cache: %w", err)
	}

	descriptionPayload, err := s.client.FetchDescription(ctx, isin)
	if err != nil { return Bond{}, fmt.Errorf("fetch description: %w", err) }
	bond, err := ParseBondDescription(descriptionPayload)
	if err != nil { return Bond{}, err }
	bond.ISIN = isin

	marketPayload, err := s.client.FetchMarketData(ctx, isin)
	if err != nil { return Bond{}, fmt.Errorf("fetch market data: %w", err) }
	market, err := ParseMarketData(marketPayload)
	if err != nil { return Bond{}, err }
	mergeMarketData(&bond, market)

	bondizationPayload, err := s.client.FetchBondization(ctx, isin)
	if err != nil { return Bond{}, fmt.Errorf("fetch bondization: %w", err) }
	bond.CouponCalendar, bond.CashflowSchedule, err = ParseBondization(bondizationPayload)
	if err != nil { return Bond{}, err }

	if err := s.cache.Set(ctx, key, bond, s.ttl); err != nil {
		return Bond{}, fmt.Errorf("write bond cache: %w", err)
	}
	return bond, nil
}
```

Implement `mergeMarketData` as explicit assignments for every `BondMarketData` field and return immediately when its argument is nil. Set `FaceUnit` only when description did not provide it; set `Currency` from market `CURRENCYID`, falling back to market/description `FaceUnit`. Do not use reflection or JSON round-trips for merging.

- [ ] **Step 4: Verify and commit**

Run: `gofmt -w internal/moex/service.go internal/moex/service_test.go && go test ./internal/moex -run 'TestCachedServiceBond' -race`

Expected: PASS.

```bash
git add internal/moex/service.go internal/moex/service_test.go
git commit -m "feat: assemble and cache MOEX bonds"
```

### Task 6: Liquid market universe with bounded concurrency

**Files:**
- Modify: `go.mod`
- Modify: `go.sum`
- Modify: `internal/moex/service.go`
- Modify: `internal/moex/service_test.go`

**Interfaces:**
- Consumes: `CachedService.Bond`, including its per-ISIN cache.
- Produces: `CachedService.MarketUniverse(ctx, limit)` and `moex:universe:{limit}` cache entries.

- [ ] **Step 1: Make `x/sync` direct and write failing universe tests**

Run: `go get golang.org/x/sync@v0.20.0`.

Add tests for cache hit, strict filter boundaries, descending turnover sort, top-N truncation, stable ordering despite out-of-order goroutine completion, at most eight concurrent `Bond` loads, cache TTL/key, empty result as a non-nil slice, fetch/parse/fan-out/cache errors, and context cancellation. Include candidates at exactly `VALTODAY=1_000_000` and `NUMTRADES=10` and assert they are excluded because the approved spec uses strict `>`.

Instrument the fake client with an atomic in-flight counter and block bond detail calls on a channel; assert `maxInFlight <= 8` before releasing them.

- [ ] **Step 2: Run universe tests and confirm failure**

Run: `go test ./internal/moex -run 'TestCachedServiceMarketUniverse' -v`

Expected: FAIL because `MarketUniverse` still returns `ErrNotImplemented`.

- [ ] **Step 3: Implement filtering, ordering, fan-out, and snapshot merge**

Use small named helpers so filtering and merge rules are directly testable:

```go
func liquidBond(bond Bond) bool {
	return bond.FaceUnit != nil && (*bond.FaceUnit == "RUB" || *bond.FaceUnit == "SUR") &&
		bond.ValueToday != nil && *bond.ValueToday > 1_000_000 &&
		bond.NumTrades != nil && *bond.NumTrades > 10 && bond.Price != nil
}

func mergeUniverseSnapshot(full Bond, snapshot Bond) Bond {
	full.Ticker = snapshot.Ticker
	if full.Name == nil { full.Name = snapshot.ShortName }
	if full.FaceValue == nil { full.FaceValue = snapshot.FaceValue }
	if full.FaceUnit == nil { full.FaceUnit = snapshot.FaceUnit }
	if full.Currency == nil { full.Currency = snapshot.FaceUnit }
	if full.LotSize == nil { full.LotSize = snapshot.LotSize }
	if full.MaturityDate == nil { full.MaturityDate = snapshot.MaturityDate }
	full.Price = snapshot.Price
	full.YieldToMaturity = snapshot.YieldToMaturity
	full.Duration = snapshot.Duration
	full.AccruedInterest = snapshot.AccruedInterest
	full.ValueToday = snapshot.ValueToday
	full.NumTrades = snapshot.NumTrades
	full.MarketDataAsOf = snapshot.MarketDataAsOf
	return full
}
```

After cache miss, fetch and parse, filter, stable-sort by turnover descending, and slice to `limit`. Allocate the result by index before starting goroutines so completion order cannot reorder output:

```go
group, groupCtx := errgroup.WithContext(ctx)
semaphore := make(chan struct{}, 8)
result := make(MarketUniverse, len(candidates))
for index, candidate := range candidates {
	index, candidate := index, candidate
	group.Go(func() error {
		select {
		case semaphore <- struct{}{}:
		case <-groupCtx.Done():
			return groupCtx.Err()
		}
		defer func() { <-semaphore }()
		full, err := s.Bond(groupCtx, candidate.ISIN)
		if err != nil { return fmt.Errorf("load bond %s: %w", candidate.ISIN, err) }
		result[index] = mergeUniverseSnapshot(full, candidate)
		return nil
	})
}
if err := group.Wait(); err != nil {
	return nil, err
}
```

Cache `result` under `fmt.Sprintf("moex:universe:%d", limit)` using `s.ttl`. Treat non-miss cache read errors and cache write errors consistently with `Bond`.

- [ ] **Step 4: Verify and commit**

Run: `gofmt -w internal/moex/service.go internal/moex/service_test.go && go test ./internal/moex -run 'TestCachedService' -race -cover`

Expected: PASS with at least 80% statement coverage for `internal/moex`.

```bash
git add go.mod go.sum internal/moex/service.go internal/moex/service_test.go
git commit -m "feat: build liquid MOEX universe"
```

### Task 7: Production wiring and end-to-end cache flow

**Files:**
- Modify: `internal/app/app.go`
- Modify: `internal/app/app_test.go`
- Create: `internal/moex/integration_test.go`

**Interfaces:**
- `app.New(config.Config)` must construct Redis cache, MOEX HTTP client, and `moex.NewService`.
- Produces: package-private `newWithServices(config.Config, moex.Service, cbr.Service) *App` test seam so app tests never require external services.

- [ ] **Step 1: Write failing wiring and integration tests**

Update `TestRunStopsAfterContextCancellation` to call `newWithServices` with local stubs. Add an app-level test with `httptest.Server` plus miniredis: create `app.New(cfg)`, request `/v1/bonds/RU000A10ABC1`, assert `200` and mapped JSON, request it again, and assert each upstream endpoint was hit exactly once.

Create `internal/moex/integration_test.go` for the universe path. The fake ISS server returns two liquid rows and the three per-bond responses; the first `MarketUniverse(ctx, 2)` must traverse HTTP -> parser -> service -> Redis, and the second call must return the same value after the server is closed, proving a Redis hit rather than another HTTP request. Then call `cache.Delete(ctx, "moex:universe:2")` and assert a third call fails because the upstream is closed, proving invalidation.

- [ ] **Step 2: Run integration tests and confirm failure**

Run: `go test ./internal/app ./internal/moex -run 'TestAppMOEXFlow|TestMOEXRedisIntegration' -v`

Expected: FAIL because `app.New` still wires `moex.StubService`.

- [ ] **Step 3: Wire production dependencies**

Construct the real MOEX graph in `app.New` and route all construction through the test seam:

```go
func New(cfg config.Config) *App {
	redisCache := cache.NewRedisCache(cfg.RedisAddr, cfg.RedisPassword, cfg.RedisDB)
	moexClient := moex.NewHTTPClient(cfg.MOEXBaseURL, cfg.HTTPTimeout)
	moexService := moex.NewService(moexClient, redisCache, cfg.MarketCacheTTL)
	return newWithServices(cfg, moexService, cbr.NewStubService())
}

func newWithServices(cfg config.Config, moexService moex.Service, cbrService cbr.Service) *App {
	return &App{
		server: httpserver.NewRouter(moexService, cbrService),
		address: cfg.Server.Address(),
	}
}
```

Keep the existing handler contracts and error redaction unchanged; transport/cache details must not leak into JSON error messages.

- [ ] **Step 4: Run complete verification**

Run:

```bash
gofmt -w internal/app internal/moex
go test ./... -race -coverprofile=coverage.out
go tool cover -func=coverage.out
go vet ./...
go build ./...
```

Expected: all commands exit 0; `internal/moex` and `internal/cache` each report at least 80% statement coverage. Remove the generated `coverage.out` after reading the report.

- [ ] **Step 5: Commit the wiring and integration coverage**

```bash
git add internal/app/app.go internal/app/app_test.go internal/moex/integration_test.go
git commit -m "feat: wire cached MOEX service"
```

### Task 8: Final contract and documentation check

**Files:**
- Modify only if generated output changes: `docs/docs.go`, `docs/swagger.json`, `docs/swagger.yaml`
- Modify: `README.md`

**Interfaces:**
- Confirms public HTTP routes and JSON schemas remain backward compatible.

- [ ] **Step 1: Add operational documentation**

In `README.md`, replace statements that MOEX routes return `501` with their implemented behavior, list the three Redis environment variables, state that Redis is required for MOEX endpoints, and give a local start example using `redis://localhost:6379` settings expressed through the existing discrete environment variables.

- [ ] **Step 2: Regenerate and verify Swagger**

Run: `make swagger && git diff --exit-code -- docs/docs.go docs/swagger.json docs/swagger.yaml`

Expected: no generated Swagger diff because route contracts and schemas are unchanged. If the generator normalizes existing output, inspect the diff and commit only semantically correct generated changes.

- [ ] **Step 3: Run the final clean-room checks**

Run:

```bash
go test ./... -count=1 -race
go vet ./...
go build ./...
git status --short
```

Expected: tests, vet, and build pass; status contains only intended implementation/documentation changes and pre-existing untracked `QWEN.md` and `go-migration.md` if they remain untracked.

- [ ] **Step 4: Commit documentation**

```bash
git add README.md docs/docs.go docs/swagger.json docs/swagger.yaml
git commit -m "docs: describe cached MOEX integration"
```
