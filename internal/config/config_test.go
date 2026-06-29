package config

import (
	"strings"
	"testing"
	"time"
)

func TestLoadReadsGatewayConfiguration(t *testing.T) {
	t.Setenv("SERVER_HOST", "127.0.0.1")
	t.Setenv("SERVER_PORT", "9090")
	t.Setenv("MOEX_BASE_URL", "https://moex.test")
	t.Setenv("CBR_KEY_RATE_URL", "https://cbr.test/rate")
	t.Setenv("CBR_FORECAST_PAGE_URL", "https://cbr.test/forecast")
	t.Setenv("HTTP_TIMEOUT", "3s")
	t.Setenv("MARKET_CACHE_TTL", "15m")
	t.Setenv("REDIS_ADDR", "redis.test:6380")
	t.Setenv("REDIS_PASSWORD", "test-password")
	t.Setenv("REDIS_DB", "2")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if got, want := cfg.Server.Address(), "127.0.0.1:9090"; got != want {
		t.Errorf("Server.Address() = %q, want %q", got, want)
	}
	if got, want := cfg.MOEXBaseURL, "https://moex.test"; got != want {
		t.Errorf("MOEXBaseURL = %q, want %q", got, want)
	}
	if got, want := cfg.CBRKeyRateURL, "https://cbr.test/rate"; got != want {
		t.Errorf("CBRKeyRateURL = %q, want %q", got, want)
	}
	if got, want := cfg.CBRForecastPageURL, "https://cbr.test/forecast"; got != want {
		t.Errorf("CBRForecastPageURL = %q, want %q", got, want)
	}
	if got, want := cfg.HTTPTimeout, 3*time.Second; got != want {
		t.Errorf("HTTPTimeout = %v, want %v", got, want)
	}
	if got, want := cfg.MarketCacheTTL, 15*time.Minute; got != want {
		t.Errorf("MarketCacheTTL = %v, want %v", got, want)
	}
	if got, want := cfg.RedisAddr, "redis.test:6380"; got != want {
		t.Errorf("RedisAddr = %q, want %q", got, want)
	}
	if got, want := cfg.RedisPassword, "test-password"; got != want {
		t.Errorf("RedisPassword = %q, want %q", got, want)
	}
	if got, want := cfg.RedisDB, 2; got != want {
		t.Errorf("RedisDB = %d, want %d", got, want)
	}
}

func TestLoadRejectsInvalidDurations(t *testing.T) {
	tests := []struct {
		name    string
		key     string
		wantErr string
	}{
		{name: "HTTP timeout", key: "HTTP_TIMEOUT", wantErr: "invalid HTTP_TIMEOUT"},
		{name: "market cache TTL", key: "MARKET_CACHE_TTL", wantErr: "invalid MARKET_CACHE_TTL"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("HTTP_TIMEOUT", "10s")
			t.Setenv("MARKET_CACHE_TTL", "15m")
			t.Setenv(tt.key, "not-a-duration")

			_, err := Load()
			if err == nil {
				t.Fatal("Load() error = nil, want error")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("Load() error = %q, want substring %q", err, tt.wantErr)
			}
		})
	}
}

func TestLoadRejectsInvalidRedisDB(t *testing.T) {
	t.Setenv("HTTP_TIMEOUT", "10s")
	t.Setenv("MARKET_CACHE_TTL", "15m")
	t.Setenv("REDIS_DB", "not-an-int")

	_, err := Load()
	if err == nil {
		t.Fatal("Load() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "invalid REDIS_DB") {
		t.Errorf("Load() error = %q, want substring %q", err, "invalid REDIS_DB")
	}
}
