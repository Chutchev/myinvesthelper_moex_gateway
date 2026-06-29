package httpserver_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"math"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/Chutchev/myinvesthelper_moex_gateway/internal/apperrors"
	"github.com/Chutchev/myinvesthelper_moex_gateway/internal/cbr"
	"github.com/Chutchev/myinvesthelper_moex_gateway/internal/httpserver"
	"github.com/Chutchev/myinvesthelper_moex_gateway/internal/moex"
	"github.com/gofiber/fiber/v3"
)

type fakeMOEXService struct {
	bond          moex.Bond
	bondErr       error
	universe      moex.MarketUniverse
	universeErr   error
	receivedISIN  string
	receivedLimit int
	bondContext   context.Context
	universeCtx   context.Context
	bondCalls     int
	universeCalls int
}

func (f *fakeMOEXService) Bond(ctx context.Context, isin string) (moex.Bond, error) {
	f.bondCalls++
	f.bondContext = ctx
	f.receivedISIN = isin
	return f.bond, f.bondErr
}

func (f *fakeMOEXService) MarketUniverse(ctx context.Context, limit int) (moex.MarketUniverse, error) {
	f.universeCalls++
	f.universeCtx = ctx
	f.receivedLimit = limit
	return f.universe, f.universeErr
}

type fakeCBRService struct {
	snapshot cbr.RateSnapshot
	err      error
	context  context.Context
	calls    int
}

func (f *fakeCBRService) Snapshot(ctx context.Context) (cbr.RateSnapshot, error) {
	f.calls++
	f.context = ctx
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

func TestHealthRejectsPOSTWithAllowedMethod(t *testing.T) {
	response := requestMethod(t, httpserver.NewRouter(&fakeMOEXService{}, &fakeCBRService{}), http.MethodPost, "/health")

	assertStatus(t, response, http.StatusMethodNotAllowed)
	if allowed := response.Header().Get("Allow"); !strings.Contains(allowed, http.MethodGet) {
		t.Fatalf("Allow = %q, want it to include GET", allowed)
	}
}

func TestSwaggerUI(t *testing.T) {
	response := request(t, httpserver.NewRouter(&fakeMOEXService{}, &fakeCBRService{}), "/swagger/index.html")

	assertStatus(t, response, http.StatusOK)
	if !strings.Contains(response.Body.String(), "Swagger UI") {
		t.Fatalf("body does not contain Swagger UI: %s", response.Body.String())
	}
}

func TestSwaggerDocumentContainsPublicPaths(t *testing.T) {
	response := request(t, httpserver.NewRouter(&fakeMOEXService{}, &fakeCBRService{}), "/swagger/doc.json")

	assertStatus(t, response, http.StatusOK)
	var document struct {
		Paths       map[string]any `json:"paths"`
		Definitions map[string]any `json:"definitions"`
	}
	decodeJSON(t, response, &document)
	for _, path := range []string{"/health", "/v1/bonds", "/v1/bonds/{isin}", "/v1/cbr/rates"} {
		if _, ok := document.Paths[path]; !ok {
			t.Errorf("Swagger path %q is missing", path)
		}
	}
	for _, schema := range []string{"cbr.RateSnapshot", "httpserver.ErrorResponse", "moex.Bond"} {
		if _, ok := document.Definitions[schema]; !ok {
			t.Errorf("Swagger schema %q is missing", schema)
		}
	}
}

func TestBondRejectsInvalidISIN(t *testing.T) {
	for _, isin := range []string{"bad", "RU000A10abc1", "RU000A10ABC!", "RU000A10ABC12"} {
		t.Run(isin, func(t *testing.T) {
			service := &fakeMOEXService{}
			response := request(t, httpserver.NewRouter(service, &fakeCBRService{}), "/v1/bonds/"+isin)

			assertStatus(t, response, http.StatusBadRequest)
			assertJSON(t, response, map[string]any{"error": "invalid ISIN"})
			if service.bondCalls != 0 {
				t.Fatalf("Bond calls = %d, want 0", service.bondCalls)
			}
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
			if strings.Contains(response.Body.String(), "database password leaked") {
				t.Fatal("response leaked underlying service error")
			}
			assertJSON(t, response, map[string]any{"error": tt.wantError})
			if service.receivedISIN != "RU000A10ABC1" {
				t.Fatalf("received ISIN = %q", service.receivedISIN)
			}
			if service.bondCalls != 1 {
				t.Fatalf("Bond calls = %d, want 1", service.bondCalls)
			}
		})
	}
}

func TestBondEncodesResult(t *testing.T) {
	service := &fakeMOEXService{bond: moex.Bond{ISIN: "RU000A10ABC1"}}
	response := request(t, httpserver.NewRouter(service, &fakeCBRService{}), "/v1/bonds/RU000A10ABC1")

	assertStatus(t, response, http.StatusOK)
	assertContentType(t, response)
	var got moex.Bond
	decodeJSON(t, response, &got)
	if !reflect.DeepEqual(got, service.bond) {
		t.Fatalf("bond = %#v, want %#v", got, service.bond)
	}
	if service.bondContext == nil {
		t.Fatal("Bond received nil context")
	}
	if service.bondCalls != 1 {
		t.Fatalf("Bond calls = %d, want 1", service.bondCalls)
	}
}

func TestBondEncodingFailureReturnsInternalServerError(t *testing.T) {
	invalidNumber := math.NaN()
	service := &fakeMOEXService{bond: moex.Bond{ISIN: "RU000A10ABC1", Price: &invalidNumber}}
	response := request(t, httpserver.NewRouter(service, &fakeCBRService{}), "/v1/bonds/RU000A10ABC1")

	assertStatus(t, response, http.StatusInternalServerError)
	if got := strings.TrimSpace(response.Body.String()); got != `{"error":"internal server error"}` {
		t.Fatalf("body = %q, want stable internal error JSON", got)
	}
	assertJSON(t, response, map[string]any{"error": "internal server error"})
}

func TestMarketUniverseRejectsInvalidLimit(t *testing.T) {
	for _, limit := range []string{"0", "x", "201"} {
		t.Run(limit, func(t *testing.T) {
			service := &fakeMOEXService{}
			response := request(t, httpserver.NewRouter(service, &fakeCBRService{}), "/v1/bonds?limit="+limit)

			assertStatus(t, response, http.StatusBadRequest)
			assertJSON(t, response, map[string]any{"error": "invalid limit"})
			if service.universeCalls != 0 {
				t.Fatalf("MarketUniverse calls = %d, want 0", service.universeCalls)
			}
		})
	}
}

func TestMarketUniverseMapsServiceErrors(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantError  string
	}{
		{name: "not implemented", err: apperrors.ErrNotImplemented, wantStatus: http.StatusNotImplemented, wantError: "not implemented"},
		{name: "unexpected", err: errors.New("secret universe error"), wantStatus: http.StatusInternalServerError, wantError: "internal server error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &fakeMOEXService{universeErr: tt.err}
			response := request(t, httpserver.NewRouter(service, &fakeCBRService{}), "/v1/bonds")

			assertStatus(t, response, tt.wantStatus)
			if strings.Contains(response.Body.String(), "secret universe error") {
				t.Fatal("response leaked underlying service error")
			}
			assertJSON(t, response, map[string]any{"error": tt.wantError})
			if service.universeCalls != 1 {
				t.Fatalf("MarketUniverse calls = %d, want 1", service.universeCalls)
			}
		})
	}
}

func TestMarketUniverseDefaultsLimitAndEncodesEmptyArray(t *testing.T) {
	for _, universe := range []moex.MarketUniverse{nil, {}} {
		service := &fakeMOEXService{universe: universe}
		response := request(t, httpserver.NewRouter(service, &fakeCBRService{}), "/v1/bonds")

		assertStatus(t, response, http.StatusOK)
		if service.receivedLimit != 40 {
			t.Fatalf("received limit = %d, want 40", service.receivedLimit)
		}
		if service.universeCtx == nil {
			t.Fatal("MarketUniverse received nil context")
		}
		assertJSON(t, response, []any{})
	}
}

func TestCBRRatesMapsNotImplemented(t *testing.T) {
	service := &fakeCBRService{err: apperrors.ErrNotImplemented}
	response := request(t, httpserver.NewRouter(&fakeMOEXService{}, service), "/v1/cbr/rates")

	assertStatus(t, response, http.StatusNotImplemented)
	assertJSON(t, response, map[string]any{"error": "not implemented"})
	if service.calls != 1 {
		t.Fatalf("Snapshot calls = %d, want 1", service.calls)
	}
}

func TestCBRRatesMapsUnexpectedErrorWithoutLeak(t *testing.T) {
	service := &fakeCBRService{err: errors.New("secret CBR error")}
	response := request(t, httpserver.NewRouter(&fakeMOEXService{}, service), "/v1/cbr/rates")

	assertStatus(t, response, http.StatusInternalServerError)
	if strings.Contains(response.Body.String(), "secret CBR error") {
		t.Fatal("response leaked underlying service error")
	}
	assertJSON(t, response, map[string]any{"error": "internal server error"})
	if service.calls != 1 {
		t.Fatalf("Snapshot calls = %d, want 1", service.calls)
	}
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
	assertContentType(t, response)
	var got cbr.RateSnapshot
	decodeJSON(t, response, &got)
	if !reflect.DeepEqual(got, service.snapshot) {
		t.Fatalf("snapshot = %#v, want %#v", got, service.snapshot)
	}
	if service.context == nil {
		t.Fatal("Snapshot received nil context")
	}
	if service.calls != 1 {
		t.Fatalf("Snapshot calls = %d, want 1", service.calls)
	}
}

func request(t *testing.T, app *fiber.App, target string) *httptest.ResponseRecorder {
	t.Helper()
	return requestMethodWithContext(t, app, http.MethodGet, target, context.Background())
}

func requestMethod(t *testing.T, app *fiber.App, method, target string) *httptest.ResponseRecorder {
	t.Helper()
	return requestMethodWithContext(t, app, method, target, context.Background())
}

func requestMethodWithContext(t *testing.T, app *fiber.App, method, target string, ctx context.Context) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(method, target, nil).WithContext(ctx)
	response, err := app.Test(request, fiber.TestConfig{Timeout: 0})
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read response: %v", err)
	}

	recorder := httptest.NewRecorder()
	for key, values := range response.Header {
		for _, value := range values {
			recorder.Header().Add(key, value)
		}
	}
	recorder.WriteHeader(response.StatusCode)
	_, _ = recorder.Write(body)
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
	assertContentType(t, response)
	var got any
	decodeJSON(t, response, &got)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("JSON = %#v, want %#v", got, want)
	}
}

func assertContentType(t *testing.T, response *httptest.ResponseRecorder) {
	t.Helper()
	if got := response.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("Content-Type = %q, want application/json", got)
	}
}

func decodeJSON(t *testing.T, response *httptest.ResponseRecorder, target any) {
	t.Helper()
	if err := json.NewDecoder(response.Body).Decode(target); err != nil {
		t.Fatalf("decode JSON: %v", err)
	}
}
