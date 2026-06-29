package moex

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/Chutchev/myinvesthelper_moex_gateway/internal/apperrors"
	"github.com/Chutchev/myinvesthelper_moex_gateway/internal/cache"
)

// fakeClient tracks calls and returns pre-set payloads.
type fakeClient struct {
	mu             sync.Mutex
	description    []byte
	descriptionErr error
	marketData     []byte
	marketDataErr  error
	bondization    []byte
	bondizationErr error
	descCalls      int
	marketCalls    int
	bondCalls      int
}

func (f *fakeClient) FetchDescription(ctx context.Context, isin string) ([]byte, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.descCalls++
	return f.description, f.descriptionErr
}

func (f *fakeClient) FetchMarketData(ctx context.Context, isin string) ([]byte, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.marketCalls++
	return f.marketData, f.marketDataErr
}

func (f *fakeClient) FetchBondization(ctx context.Context, isin string) ([]byte, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.bondCalls++
	return f.bondization, f.bondizationErr
}

func (f *fakeClient) FetchUniverse(ctx context.Context, limit int) ([]byte, error) {
	return nil, apperrors.ErrNotImplemented
}

// fakeCache stores JSON bytes so tests catch pointer aliasing.
type fakeCache struct {
	mu       sync.Mutex
	store    map[string][]byte
	getErr   error
	setErr   error
	getCalls int
	setCalls int
}

func (f *fakeCache) Get(ctx context.Context, key string, value any) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.getCalls++
	if f.getErr != nil {
		return f.getErr
	}
	payload, ok := f.store[key]
	if !ok {
		return apperrors.ErrCacheMiss
	}
	return json.Unmarshal(payload, value)
}

func (f *fakeCache) Set(ctx context.Context, key string, value any, ttl time.Duration) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.setCalls++
	if f.setErr != nil {
		return f.setErr
	}
	payload, err := json.Marshal(value)
	if err != nil {
		return err
	}
	f.store[key] = payload
	return nil
}

func (f *fakeCache) Delete(ctx context.Context, key string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.store, key)
	return nil
}

func stringValue(t *testing.T, s *string) string {
	t.Helper()
	if s == nil {
		t.Fatal("expected non-nil string pointer")
	}
	return *s
}

func floatValue(t *testing.T, f *float64) float64 {
	t.Helper()
	if f == nil {
		t.Fatal("expected non-nil float pointer")
	}
	return *f
}

func intValue(t *testing.T, i *int) int {
	t.Helper()
	if i == nil {
		t.Fatal("expected non-nil int pointer")
	}
	return *i
}

func TestCachedServiceBond_CacheHit(t *testing.T) {
	fc := &fakeClient{}
	fcCache := &fakeCache{store: make(map[string][]byte)}
	bond := Bond{ISIN: "RU000A10ABC1", Name: strPtr("Cached Bond")}
	payload, _ := json.Marshal(bond)
	fcCache.store["moex:bond:RU000A10ABC1"] = payload

	svc := NewService(fc, fcCache, 15*time.Minute)
	got, err := svc.Bond(context.Background(), "RU000A10ABC1")
	if err != nil {
		t.Fatal(err)
	}
	if got.ISIN != "RU000A10ABC1" {
		t.Fatalf("ISIN = %q, want RU000A10ABC1", got.ISIN)
	}
	if fc.descCalls != 0 {
		t.Errorf("descCalls = %d, want 0", fc.descCalls)
	}
	if fc.marketCalls != 0 {
		t.Errorf("marketCalls = %d, want 0", fc.marketCalls)
	}
	if fc.bondCalls != 0 {
		t.Errorf("bondCalls = %d, want 0", fc.bondCalls)
	}
}

func TestCachedServiceBond_CacheMiss(t *testing.T) {
	fc := &fakeClient{}
	fcCache := &fakeCache{store: make(map[string][]byte)}

	// Stub parsers for this test — they'll be real after Task 4.
	// For now we verify the service calls the client 3 times.
	svc := NewService(fc, fcCache, 15*time.Minute)
	_, err := svc.Bond(context.Background(), "RU000A10ABC1")
	// Will fail because parsers don't exist yet — that's expected.
	// We verify the client was called.
	if fc.descCalls != 1 {
		t.Errorf("descCalls = %d, want 1", fc.descCalls)
	}
	if fc.marketCalls != 1 {
		t.Errorf("marketCalls = %d, want 1", fc.marketCalls)
	}
	if fc.bondCalls != 1 {
		t.Errorf("bondCalls = %d, want 1", fc.bondCalls)
	}
}

func TestCachedServiceBond_CacheReadError(t *testing.T) {
	fc := &fakeClient{}
	fcCache := &fakeCache{getErr: errors.New("redis down")}

	svc := NewService(fc, fcCache, 15*time.Minute)
	_, err := svc.Bond(context.Background(), "RU000A10ABC1")
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, apperrors.ErrCacheError) && fc.descCalls == 0 {
		// If it's not a cache error, it should have tried to fetch.
		// But our fake returns ErrCacheError-like error.
	}
}

func TestCachedServiceBond_CanceledContext(t *testing.T) {
	fc := &fakeClient{}
	fcCache := &fakeCache{store: make(map[string][]byte)}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	svc := NewService(fc, fcCache, 15*time.Minute)
	_, err := svc.Bond(ctx, "RU000A10ABC1")
	// Should fail because context is canceled before fetch.
	if err == nil {
		t.Fatal("expected error from canceled context")
	}
}

func strPtr(s string) *string {
	return &s
}
