package httpserver_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	"github.com/Chutchev/myinvesthelper_moex_gateway/internal/apperrors"
	"github.com/Chutchev/myinvesthelper_moex_gateway/internal/cbr"
	"github.com/Chutchev/myinvesthelper_moex_gateway/internal/httpserver"
	"github.com/Chutchev/myinvesthelper_moex_gateway/internal/moex"
)

type fakeMOEXService struct {
	bond          moex.Bond
	bondErr       error
	universe      moex.MarketUniverse
	universeErr   error
	receivedISIN  string
	receivedLimit int
}

func (f *fakeMOEXService) Bond(_ context.Context, isin string) (moex.Bond, error) {
	f.receivedISIN = isin
	return f.bond, f.bondErr
}

func (f *fakeMOEXService) MarketUniverse(_ context.Context, limit int) (moex.MarketUniverse, error) {
	f.receivedLimit = limit
	return f.universe, f.universeErr
}

type fakeCBRService struct {
	snapshot cbr.RateSnapshot
	err      error
}

func (f *fakeCBRService) Snapshot(context.Context) (cbr.RateSnapshot, error) {
	return f.snapshot, f.err
}

func TestHealth(t *testing.T) {
	response := request(t, httpserver.NewRouter(&fakeMOEXService{}, &fakeCBRService{}), "/health")

	assertStatus(t, response, http.StatusOK)
	if got := response.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("Content-Type = %q, want application/json", got)
	}
	assertJSON(t, response, map[string]any{"status": "ok"})
}

func TestBondRejectsInvalidISIN(t *testing.T) {
	for _, isin := range []string{"bad", "RU000A10abc1", "RU000A10ABC!", "RU000A10ABC12"} {
		t.Run(isin, func(t *testing.T) {
			response := request(t, httpserver.NewRouter(&fakeMOEXService{}, &fakeCBRService{}), "/v1/bonds/"+isin)

			assertStatus(t, response, http.StatusBadRequest)
			assertJSON(t, response, map[string]any{"error": "invalid ISIN"})
		})
	}
}

func TestBondMapsServiceErrors(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantError  string
	}{
		{name: "not implemented", err: apperrors.ErrNotImplemented, wantStatus: http.StatusNotImplemented, wantError: "not implemented"},
		{name: "unexpected", err: errors.New("database password leaked"), wantStatus: http.StatusInternalServerError, wantError: "internal server error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &fakeMOEXService{bondErr: tt.err}
			response := request(t, httpserver.NewRouter(service, &fakeCBRService{}), "/v1/bonds/RU000A10ABC1")

			assertStatus(t, response, tt.wantStatus)
			assertJSON(t, response, map[string]any{"error": tt.wantError})
			if service.receivedISIN != "RU000A10ABC1" {
				t.Fatalf("received ISIN = %q", service.receivedISIN)
			}
		})
	}
}

func TestBondEncodesResult(t *testing.T) {
	service := &fakeMOEXService{bond: moex.Bond{ISIN: "RU000A10ABC1"}}
	response := request(t, httpserver.NewRouter(service, &fakeCBRService{}), "/v1/bonds/RU000A10ABC1")

	assertStatus(t, response, http.StatusOK)
	var got moex.Bond
	decodeJSON(t, response, &got)
	if !reflect.DeepEqual(got, service.bond) {
		t.Fatalf("bond = %#v, want %#v", got, service.bond)
	}
}

func TestMarketUniverseRejectsInvalidLimit(t *testing.T) {
	for _, limit := range []string{"0", "x", "201"} {
		t.Run(limit, func(t *testing.T) {
			response := request(t, httpserver.NewRouter(&fakeMOEXService{}, &fakeCBRService{}), "/v1/bonds?limit="+limit)

			assertStatus(t, response, http.StatusBadRequest)
			assertJSON(t, response, map[string]any{"error": "invalid limit"})
		})
	}
}

func TestMarketUniverseDefaultsLimitAndEncodesEmptyArray(t *testing.T) {
	service := &fakeMOEXService{}
	response := request(t, httpserver.NewRouter(service, &fakeCBRService{}), "/v1/bonds")

	assertStatus(t, response, http.StatusOK)
	if service.receivedLimit != 40 {
		t.Fatalf("received limit = %d, want 40", service.receivedLimit)
	}
	assertJSON(t, response, []any{})
}

func TestCBRRatesMapsNotImplemented(t *testing.T) {
	service := &fakeCBRService{err: apperrors.ErrNotImplemented}
	response := request(t, httpserver.NewRouter(&fakeMOEXService{}, service), "/v1/cbr/rates")

	assertStatus(t, response, http.StatusNotImplemented)
	assertJSON(t, response, map[string]any{"error": "not implemented"})
}

func TestCBRRatesEncodesResult(t *testing.T) {
	rate := 16.5
	service := &fakeCBRService{snapshot: cbr.RateSnapshot{
		CurrentRate: &rate,
		Direction:   cbr.DirectionDown,
		FetchedAt:   time.Date(2026, time.June, 28, 10, 0, 0, 0, time.UTC),
	}}
	response := request(t, httpserver.NewRouter(&fakeMOEXService{}, service), "/v1/cbr/rates")

	assertStatus(t, response, http.StatusOK)
	var got cbr.RateSnapshot
	decodeJSON(t, response, &got)
	if !reflect.DeepEqual(got, service.snapshot) {
		t.Fatalf("snapshot = %#v, want %#v", got, service.snapshot)
	}
}

func request(t *testing.T, handler http.Handler, target string) *httptest.ResponseRecorder {
	t.Helper()
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, target, nil))
	return recorder
}

func assertStatus(t *testing.T, response *httptest.ResponseRecorder, want int) {
	t.Helper()
	if response.Code != want {
		t.Fatalf("status = %d, want %d; body = %s", response.Code, want, response.Body.String())
	}
}

func assertJSON(t *testing.T, response *httptest.ResponseRecorder, want any) {
	t.Helper()
	var got any
	decodeJSON(t, response, &got)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("JSON = %#v, want %#v", got, want)
	}
}

func decodeJSON(t *testing.T, response *httptest.ResponseRecorder, target any) {
	t.Helper()
	if err := json.NewDecoder(response.Body).Decode(target); err != nil {
		t.Fatalf("decode JSON: %v", err)
	}
}
