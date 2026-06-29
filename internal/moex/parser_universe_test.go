package moex

import (
	"errors"
	"testing"

	"github.com/Chutchev/myinvesthelper_moex_gateway/internal/apperrors"
)

func TestParseUniverseResponse_Success(t *testing.T) {
	payload := []byte(`{
		"securities": {
			"columns": ["SECID", "SHORTNAME", "LOTSIZE", "FACEVALUE", "FACEUNIT", "MATDATE", "COUPONPERCENT", "COUPONVALUE", "ISSUESIZE"],
			"data": [
				["RU000A10ABC1", "Bond1", 10, 1000, "RUB", "2030-01-01", 5.5, 27.5, 5000000000],
				["RU000A10ABC2", "Bond2", 100, 5000, "SUR", "2031-06-01", 6.0, 75.0, 10000000000]
			]
		},
		"marketdata": {
			"columns": ["SECID", "LAST", "YIELD", "DURATION", "ACCRUEDINT", "VALTODAY", "NUMTRADES", "SYSTIME"],
			"data": [
				["RU000A10ABC1", 101.25, 5.8, 3.2, 1.5, 2500000, 42, "2026-06-29T10:00:00"],
				["RU000A10ABC2", 99.5, 6.2, 4.1, 2.0, 3000000, 37, "2026-06-29T10:00:00"]
			]
		}
	}`)

	got, err := ParseUniverseResponse(payload)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2", len(got))
	}

	// First bond
	if got[0].ISIN != "RU000A10ABC1" {
		t.Errorf("ISIN = %s, want RU000A10ABC1", got[0].ISIN)
	}
	if stringValue(t, got[0].SECID) != "RU000A10ABC1" {
		t.Errorf("SECID = %v, want RU000A10ABC1", got[0].SECID)
	}
	if stringValue(t, got[0].Ticker) != "RU000A10ABC1" {
		t.Errorf("Ticker = %v, want RU000A10ABC1", got[0].Ticker)
	}
	if floatValue(t, got[0].Price) != 101.25 {
		t.Errorf("Price = %v, want 101.25", got[0].Price)
	}
	if floatValue(t, got[0].ValueToday) != 2500000 {
		t.Errorf("ValueToday = %v, want 2500000", got[0].ValueToday)
	}
	if intValue(t, got[0].NumTrades) != 42 {
		t.Errorf("NumTrades = %v, want 42", got[0].NumTrades)
	}
	if floatValue(t, got[0].FaceValue) != 1000 {
		t.Errorf("FaceValue = %v, want 1000", got[0].FaceValue)
	}
	if stringValue(t, got[0].FaceUnit) != "RUB" {
		t.Errorf("FaceUnit = %v, want RUB", got[0].FaceUnit)
	}

	// Second bond
	if got[1].ISIN != "RU000A10ABC2" {
		t.Errorf("ISIN = %s, want RU000A10ABC2", got[1].ISIN)
	}
	if floatValue(t, got[1].Price) != 99.5 {
		t.Errorf("Price = %v, want 99.5", got[1].Price)
	}
}

func TestParseUniverseResponse_JoinBySecid(t *testing.T) {
	// Market data order differs from securities order — should still join correctly.
	payload := []byte(`{
		"securities": {
			"columns": ["SECID", "SHORTNAME"],
			"data": [
				["B", "Bond B"],
				["A", "Bond A"]
			]
		},
		"marketdata": {
			"columns": ["SECID", "LAST"],
			"data": [
				["A", 100],
				["B", 200]
			]
		}
	}`)

	got, err := ParseUniverseResponse(payload)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2", len(got))
	}

	// Find bonds by ISIN
	bonds := make(map[string]Bond)
	for _, b := range got {
		bonds[b.ISIN] = b
	}

	if floatValue(t, bonds["A"].Price) != 100 {
		t.Errorf("A.Price = %v, want 100", bonds["A"].Price)
	}
	if floatValue(t, bonds["B"].Price) != 200 {
		t.Errorf("B.Price = %v, want 200", bonds["B"].Price)
	}
}

func TestParseUniverseResponse_NoMarketData(t *testing.T) {
	payload := []byte(`{
		"securities": {
			"columns": ["SECID", "SHORTNAME"],
			"data": [
				["RU000A10ABC1", "Bond1"]
			]
		}
	}`)

	got, err := ParseUniverseResponse(payload)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("len = %d, want 1", len(got))
	}
	if got[0].ISIN != "RU000A10ABC1" {
		t.Errorf("ISIN = %s, want RU000A10ABC1", got[0].ISIN)
	}
	if got[0].Price != nil {
		t.Errorf("Price should be nil, got %v", got[0].Price)
	}
}

func TestParseUniverseResponse_MarketDataOnlySecid(t *testing.T) {
	// Market data has a SECID not in securities — should be skipped.
	payload := []byte(`{
		"securities": {
			"columns": ["SECID"],
			"data": [["A"]]
		},
		"marketdata": {
			"columns": ["SECID", "LAST"],
			"data": [
				["A", 100],
				["B", 200]
			]
		}
	}`)

	got, err := ParseUniverseResponse(payload)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("len = %d, want 1", len(got))
	}
	if got[0].ISIN != "A" {
		t.Errorf("ISIN = %s, want A", got[0].ISIN)
	}
}

func TestParseUniverseResponse_MalformedJSON(t *testing.T) {
	_, err := ParseUniverseResponse([]byte(`not json`))
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, apperrors.ErrParseError) {
		t.Fatalf("error = %v, want ErrParseError", err)
	}
}

func TestParseUniverseResponse_MissingSecurities(t *testing.T) {
	_, err := ParseUniverseResponse([]byte(`{"marketdata": {"columns": [], "data": []}}`))
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, apperrors.ErrParseError) {
		t.Fatalf("error = %v, want ErrParseError", err)
	}
}

func TestParseUniverseResponse_FallbackColumns(t *testing.T) {
	payload := []byte(`{
		"securities": {
			"columns": ["SECID"],
			"data": [["RU000A10ABC1"]]
		},
		"marketdata": {
			"columns": ["SECID", "LCURRENTPRICE", "ACCINT", "UPDATETIME"],
			"data": [["RU000A10ABC1", 99.5, 0.8, "2026-06-29"]]
		}
	}`)

	got, err := ParseUniverseResponse(payload)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("len = %d, want 1", len(got))
	}
	if floatValue(t, got[0].Price) != 99.5 {
		t.Errorf("Price (LCURRENTPRICE) = %v, want 99.5", got[0].Price)
	}
	if floatValue(t, got[0].AccruedInterest) != 0.8 {
		t.Errorf("AccruedInterest (ACCINT) = %v, want 0.8", got[0].AccruedInterest)
	}
	if stringValue(t, got[0].MarketDataAsOf) != "2026-06-29" {
		t.Errorf("MarketDataAsOf (UPDATETIME) = %v, want 2026-06-29", got[0].MarketDataAsOf)
	}
}

func TestParseUniverseResponse_InvalidNumeric(t *testing.T) {
	payload := []byte(`{
		"securities": {
			"columns": ["SECID", "FACEVALUE"],
			"data": [["RU000A10ABC1", "not-a-number"]]
		},
		"marketdata": {
			"columns": ["SECID"],
			"data": [["RU000A10ABC1"]]
		}
	}`)

	_, err := ParseUniverseResponse(payload)
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, apperrors.ErrParseError) {
		t.Fatalf("error = %v, want ErrParseError", err)
	}
}

func TestParseUniverseResponse_EmptySecurities(t *testing.T) {
	payload := []byte(`{
		"securities": {
			"columns": ["SECID"],
			"data": []
		},
		"marketdata": {
			"columns": ["SECID", "LAST"],
			"data": [["RU000A10ABC1", 100]]
		}
	}`)

	got, err := ParseUniverseResponse(payload)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("len = %d, want 0", len(got))
	}
}
