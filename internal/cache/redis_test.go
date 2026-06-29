package cache

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"

	"github.com/Chutchev/myinvesthelper_moex_gateway/internal/apperrors"
)

type sample struct {
	ISIN  string  `json:"isin"`
	Price float64 `json:"price"`
}

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

func TestRedisCacheMalformedJSON(t *testing.T) {
	server := miniredis.RunT(t)
	server.Set("bad", "not-json")
	err := NewRedisCache(server.Addr(), "", 0).Get(context.Background(), "bad", &sample{})
	if !errors.Is(err, apperrors.ErrCacheError) {
		t.Fatalf("Get() error = %v, want ErrCacheError", err)
	}
}

func TestRedisCacheDelete(t *testing.T) {
	server := miniredis.RunT(t)
	cache := NewRedisCache(server.Addr(), "", 0)
	cache.Set(context.Background(), "key", sample{ISIN: "X"}, time.Minute)

	if err := cache.Delete(context.Background(), "key"); err != nil {
		t.Fatal(err)
	}
	err := cache.Get(context.Background(), "key", &sample{})
	if !errors.Is(err, apperrors.ErrCacheMiss) {
		t.Fatalf("after Delete, Get() error = %v, want ErrCacheMiss", err)
	}
}

func TestRedisCacheUnreachable(t *testing.T) {
	cache := NewRedisCache("localhost:1", "", 0)
	err := cache.Get(context.Background(), "key", &sample{})
	if !errors.Is(err, apperrors.ErrCacheError) {
		t.Fatalf("Get() error = %v, want ErrCacheError", err)
	}
}
