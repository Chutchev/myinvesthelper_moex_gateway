# MOEX HTTP Client + Parsing + Redis Cache Design

**Date:** 2026-06-29  
**Status:** Approved  
**Author:** Qwen Code (with user approval)

---

## Overview

This spec covers the implementation of MOEX ISS API integration in the Go gateway, including HTTP client, JSON parsing, business logic service layer, and Redis caching.

## Scope

**In scope:**
- HTTP client for MOEX ISS API (4 endpoints)
- JSON parsing for MOEX responses
- Service layer with business logic (liquidity filtering, data merging)
- Redis cache with TTL for bond data

**Out of scope:**
- CBR (Central Bank) integration
- gRPC/protobuf support
- Database persistence
- Python backend changes

---

## Step 1: HTTP Client Implementation

### File: `internal/moex/client.go`

Implement the `Client` interface methods:

```go
type Client interface {
    FetchUniverse(ctx context.Context, limit int) ([]byte, error)
    FetchDescription(ctx context.Context, isin string) ([]byte, error)
    FetchMarketData(ctx context.Context, isin string) ([]byte, error)
    FetchBondization(ctx context.Context, isin string) ([]byte, error)
}
```

**Endpoints:**
| Method | URL Pattern | Timeout |
|---|---|---|
| `FetchUniverse` | `{MOEX_BASE_URL}/engines/stock/markets/bonds/boards/TQCB/securities.json` | 20s |
| `FetchDescription` | `{MOEX_BASE_URL}/securities/{isin}.json` | 10s |
| `FetchMarketData` | `{MOEX_BASE_URL}/engines/stock/markets/bonds/securities/{isin}.json` | 10s |
| `FetchBondization` | `{MOEX_BASE_URL}/statistics/engines/stock/markets/bonds/bondization/{isin}.json` | 10s |

**Configuration:**
- Base URL from `config.MOEXBaseURL`
- HTTP client with configurable timeout (default 10s)
- User-Agent header: `myinvesthelper-gateway/1.0`

**Error handling:**
- HTTP errors → return `apperrors.ErrHTTPError` with status code
- Timeout → return `apperrors.ErrTimeout`
- Network errors → return wrapped error

---

## Step 2: JSON Parsing

### File: `internal/moex/parser.go` (new)

**MOEX response format:**
```json
{
  "securities": {
    "columns": ["SECID", "SHORTNAME", ...],
    "data": [["RU0001...", "OB26001RM...", ...], ...]
  },
  "marketdata": {
    "columns": ["LAST", "YIELD", ...],
    "data": [[102.5, 8.2, ...], ...]
  }
}
```

**Parser functions:**
```go
func ParseUniverseResponse(data []byte) (MarketUniverse, error)
func ParseBondDescription(data []byte) (Bond, error)
func ParseMarketData(data []byte) (*BondMarketData, error)
func ParseBondization(data []byte) ([]Coupon, []Cashflow, error)
```

**Key parsing rules:**
- Map columns by index (not by name)
- Handle null/empty values → use pointers in structs
- Convert numeric strings to `float64`/`int` with error handling
- Normalize coupon types: "фиксированный" → "fixed", "плавающий" → "floating"
- Date format: `YYYY-MM-DD` (preserve as string)

**Helper functions:**
```go
func toInt(value any) *int
func toFloat(value any) *float64
func toBool(value any) *bool
func indexedRow(columns []string, row []any) map[string]any
func pickValue(row map[string]any, names ...string) any
```

---

## Step 3: Service Layer

### File: `internal/moex/service.go`

Implement the `Service` interface:

```go
type Service interface {
    Bond(ctx context.Context, isin string) (Bond, error)
    MarketUniverse(ctx context.Context, limit int) (MarketUniverse, error)
}
```

**`Bond(ctx, isin)` logic:**
1. Check Redis cache for `moex:bond:{isin}`
2. If cache hit → return cached bond
3. If cache miss:
   - Fetch description via `client.FetchDescription(isin)`
   - Fetch market data via `client.FetchMarketData(isin)`
   - Fetch bondization via `client.FetchBondization(isin)`
   - Parse all responses
   - Merge data into single `Bond` struct
   - Build cashflow schedule (coupons + amortizations)
   - Cache result in Redis (TTL 15m)
   - Return bond

**`MarketUniverse(ctx, limit)` logic:**
1. Check Redis cache for `moex:universe:{limit}`
2. If cache hit → return cached universe
3. If cache miss:
   - Fetch universe via `client.FetchUniverse(limit)`
   - Parse response
   - Filter liquid bonds:
     - `FACEUNIT` in ["RUB", "SUR"]
     - `VALTODAY` > 1,000,000
     - `NUMTRADES` > 10
     - Price exists (LAST or LCURRENTPRICE)
   - Sort by `VALTODAY` descending
   - Take top N bonds
   - For each bond, fetch full info via `Bond(ctx, isin)` (concurrent with errgroup, max 8 goroutines)
   - Cache result in Redis (TTL 15m)
   - Return universe

**Concurrency:**
- Use `golang.org/x/sync/errgroup` with `errgroup.WithContext`
- Limit concurrency with buffered channel (size 8) as semaphore

---

## Step 4: Redis Cache

### File: `internal/cache/redis.go` (new)

**Interface:**
```go
type Cache interface {
    Get(ctx context.Context, key string, value any) error
    Set(ctx context.Context, key string, value any, ttl time.Duration) error
    Delete(ctx context.Context, key string) error
}
```

**Redis implementation:**
```go
type RedisCache struct {
    client *redis.Client
}

func NewRedisCache(addr string, password string, db int) *RedisCache
func (c *RedisCache) Get(ctx context.Context, key string, value any) error
func (c *RedisCache) Set(ctx context.Context, key string, value any, ttl time.Duration) error
func (c *RedisCache) Delete(ctx context.Context, key string) error
```

**Cache keys:**
- `moex:universe:{limit}` — MarketUniverse result (TTL: 15m)
- `moex:bond:{isin}` — Individual bond data (TTL: 15m)

**Serialization:**
- Use `encoding/json` for Go structs → JSON → Redis
- Handle cache misses (redis.Nil) → return error to trigger refetch

**Configuration:**
- Redis address from `REDIS_ADDR` env var (default: `localhost:6379`)
- Password from `REDIS_PASSWORD` env var (default: empty)
- DB from `REDIS_DB` env var (default: 0)
- Connection timeout: 5s
- Read/write timeout: 3s

---

## Dependencies

Add to `go.mod`:
```go
require (
    github.com/redis/go-redis/v9 v9.0.5
    golang.org/x/sync v0.20.0  // already present
)
```

---

## Testing Strategy

**Unit tests:**
- `client_test.go` — mock HTTP responses, test all 4 Fetch methods
- `parser_test.go` — test JSON parsing with sample MOEX responses
- `service_test.go` — test business logic (filtering, merging, caching)
- `redis_test.go` — test cache operations with testcontainers or mock

**Integration tests:**
- Test full flow: HTTP → parse → service → cache
- Test cache invalidation
- Test error scenarios (timeout, network failure, malformed JSON)

---

## Error Handling

**Error types:**
```go
var (
    ErrHTTPError    = errors.New("HTTP request failed")
    ErrTimeout      = errors.New("request timeout")
    ErrParseError   = errors.New("failed to parse response")
    ErrCacheMiss    = errors.New("cache miss")
    ErrCacheError   = errors.New("cache operation failed")
)
```

**HTTP error mapping:**
- 4xx → return error to caller (bad request, not found)
- 5xx → retry once, then return error
- Timeout → return `ErrTimeout`
- Network error → wrap and return

---

## Configuration

Add to `internal/config/config.go`:
```go
type Config struct {
    // ... existing fields
    
    // Redis
    RedisAddr    string `env:"REDIS_ADDR,default=localhost:6379"`
    RedisPassword string `env:"REDIS_PASSWORD,default="`
    RedisDB      int    `env:"REDIS_DB,default=0"`
    
    // MOEX
    MOEXBaseURL string `env:"MOEX_BASE_URL,default=https://iss.moex.com/2"`
    HTTPTimeout time.Duration `env:"HTTP_TIMEOUT,default=10s"`
    MarketCacheTTL time.Duration `env:"MARKET_CACHE_TTL,default=15m"`
}
```

---

## Success Criteria

1. ✅ HTTP client successfully fetches data from all 4 MOEX endpoints
2. ✅ JSON parser correctly maps MOEX responses to Go structs
3. ✅ Service layer filters and sorts bonds by liquidity
4. ✅ Redis cache stores and retrieves bond data with TTL
5. ✅ All unit tests pass (≥80% coverage for new code)
6. ✅ Integration test verifies full flow: HTTP → parse → cache → return

---

## Rollout Plan

1. Implement HTTP client (1-2 hours)
2. Implement JSON parser (1-2 hours)
3. Implement service layer (2-3 hours)
4. Implement Redis cache (1 hour)
5. Write tests (2-3 hours)
6. Integration testing and debugging (1-2 hours)

**Total estimated time:** 8-12 hours

---

## Notes

- Follow existing Go conventions in the project (see QWEN.md)
- Use Swagger annotations for new endpoints (if adding HTTP handlers)
- Keep stub services for CBR until next iteration
- Commit incrementally after each step
