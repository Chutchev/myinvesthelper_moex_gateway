package moex

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Chutchev/myinvesthelper_moex_gateway/internal/apperrors"
)

func TestHTTPClient_FetchMethods(t *testing.T) {
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

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != tt.path {
					t.Errorf("path = %q, want %q", r.URL.Path, tt.path)
				}
				if got := r.Header.Get("User-Agent"); got != userAgent {
					t.Errorf("User-Agent = %q, want %q", got, userAgent)
				}
				w.Write([]byte(`{}`))
			}))
			defer server.Close()

			client := NewHTTPClient(server.URL, 5*time.Second)
			body, err := tt.call(context.Background(), client)
			if err != nil {
				t.Fatalf("call error = %v", err)
			}
			if string(body) != `{}` {
				t.Errorf("body = %s, want {}", string(body))
			}
		})
	}
}

func TestHTTPClient_UniverseQuery(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("limit") != "40" {
			t.Errorf("limit = %q, want 40", r.URL.Query().Get("limit"))
		}
		if r.URL.Query().Get("securities.columns") == "" {
			t.Error("missing securities.columns")
		}
		if r.URL.Query().Get("marketdata.columns") == "" {
			t.Error("missing marketdata.columns")
		}
		w.Write([]byte(`{}`))
	}))
	defer server.Close()

	client := NewHTTPClient(server.URL, 5*time.Second)
	_, err := client.FetchUniverse(context.Background(), 40)
	if err != nil {
		t.Fatal(err)
	}
}

func TestHTTPClient_HTTP404_NoRetry(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := NewHTTPClient(server.URL, 5*time.Second)
	_, err := client.FetchDescription(context.Background(), "RU000A10ABC1")
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, apperrors.ErrHTTPError) {
		t.Fatalf("error = %v, want ErrHTTPError", err)
	}
	if !strings.Contains(err.Error(), "status 404") {
		t.Errorf("error = %v, want status 404", err)
	}
	if attempts != 1 {
		t.Errorf("attempts = %d, want 1", attempts)
	}
}

func TestHTTPClient_HTTP500_RetriesOnce(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts == 1 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Write([]byte(`{}`))
	}))
	defer server.Close()

	client := NewHTTPClient(server.URL, 5*time.Second)
	body, err := client.FetchDescription(context.Background(), "RU000A10ABC1")
	if err != nil {
		t.Fatalf("unexpected error = %v", err)
	}
	if string(body) != `{}` {
		t.Errorf("body = %s, want {}", string(body))
	}
	if attempts != 2 {
		t.Errorf("attempts = %d, want 2", attempts)
	}
}

func TestHTTPClient_Timeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
	}))
	defer server.Close()

	client := NewHTTPClient(server.URL, 100*time.Millisecond)
	_, err := client.FetchDescription(context.Background(), "RU000A10ABC1")
	if !errors.Is(err, apperrors.ErrTimeout) {
		t.Fatalf("error = %v, want ErrTimeout", err)
	}
}

func TestHTTPClient_CanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(500 * time.Millisecond)
	}))
	defer server.Close()

	client := NewHTTPClient(server.URL, 5*time.Second)
	_, err := client.FetchDescription(ctx, "RU000A10ABC1")
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want context.Canceled", err)
	}
}

func TestHTTPClient_OversizedResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Write more than 16 MiB
		for i := 0; i < 20<<20; i++ {
			w.Write([]byte{0})
		}
	}))
	defer server.Close()

	client := NewHTTPClient(server.URL, 5*time.Second)
	_, err := client.FetchDescription(context.Background(), "RU000A10ABC1")
	if err == nil {
		t.Fatal("expected error for oversized response")
	}
	if !errors.Is(err, apperrors.ErrHTTPError) {
		t.Fatalf("error = %v, want ErrHTTPError", err)
	}
}

func TestNewHTTPClient_UsesDefaultTimeout(t *testing.T) {
	client := NewHTTPClient("https://moex.test", 0)
	if client.timeout != 10*time.Second {
		t.Errorf("timeout = %v, want 10s", client.timeout)
	}
}
