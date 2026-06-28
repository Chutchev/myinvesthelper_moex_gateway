package config

import (
	"fmt"
	"net"
	"os"
	"time"
)

type Config struct {
	Server             ServerConfig
	MOEXBaseURL        string
	CBRKeyRateURL      string
	CBRForecastPageURL string
	HTTPTimeout        time.Duration
	MarketCacheTTL     time.Duration
	LogLevel           string
}

type ServerConfig struct{ Host, Port string }

func (c *Config) Address() string {
	return net.JoinHostPort(c.Server.Host, c.Server.Port)
}

func Load() (*Config, error) {
	timeout, err := time.ParseDuration(getEnv("HTTP_TIMEOUT", "10s"))
	if err != nil {
		return nil, fmt.Errorf("invalid HTTP_TIMEOUT: %w", err)
	}
	ttl, err := time.ParseDuration(getEnv("MARKET_CACHE_TTL", "15m"))
	if err != nil {
		return nil, fmt.Errorf("invalid MARKET_CACHE_TTL: %w", err)
	}

	return &Config{
		Server:             ServerConfig{Host: getEnv("SERVER_HOST", "0.0.0.0"), Port: getEnv("SERVER_PORT", "8080")},
		MOEXBaseURL:        getEnv("MOEX_BASE_URL", "https://iss.moex.com/iss"),
		CBRKeyRateURL:      getEnv("CBR_KEY_RATE_URL", "https://www.cbr.ru/hd_base/KeyRate/"),
		CBRForecastPageURL: getEnv("CBR_FORECAST_PAGE_URL", "https://www.cbr.ru/statistics/ddkp/mo_br/"),
		HTTPTimeout:        timeout,
		MarketCacheTTL:     ttl,
		LogLevel:           getEnv("LOG_LEVEL", "info"),
	}, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
