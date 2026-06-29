package moex

import (
	"errors"
	"testing"

	"github.com/Chutchev/myinvesthelper_moex_gateway/internal/apperrors"
)

func TestParseMarketData_Success(t *testing.T) {
	payload := []byte(`{
		"marketdata": {
			"columns": ["LAST", "YIELD", "DURATION", "ACCRUEDINT", "VALTODAY", "NUMTRADES", "SYSTIME", "LOTSIZE", "CURRENCYID", "FACEUNIT"],
			"data": [[101.25, 5.8, 3.2, 1.5, 2500000, 42, "2026-06-29T10:00:00", 10, "RUB", "RUB"]]
		}
	}`)
	md, err := ParseMarketData(payload)
	if err != nil {
		t.Fatal(err)
	}
	if floatValue(t, md.Price) != 101.25 {
		t.Errorf("Price = %v, want 101.25", md.Price)
	}
	if floatValue(t, md.YieldToMaturity) != 5.8 {
		t.Errorf("YieldToMaturity = %v, want 5.8", md.YieldToMaturity)
	}
	if floatValue(t, md.Duration) != 3.2 {
		t.Errorf("Duration = %v, want 3.2", md.Duration)
	}
	if floatValue(t, md.AccruedInterest) != 1.5 {
		t.Errorf("AccruedInterest = %v, want 1.5", md.AccruedInterest)
	}
	if floatValue(t, md.ValueToday) != 2500000 {
		t.Errorf("ValueToday = %v, want 2500000", md.ValueToday)
	}
	if intValue(t, md.NumTrades) != 42 {
		t.Errorf("NumTrades = %v, want 42", md.NumTrades)
	}
	if stringValue(t, md.MarketDataAsOf) != "2026-06-29T10:00:00" {
		t.Errorf("MarketDataAsOf = %v, want 2026-06-29T10:00:00", md.MarketDataAsOf)
	}
	if intValue(t, md.LotSize) != 10 {
		t.Errorf("LotSize = %v, want 10", md.LotSize)
	}
	if stringValue(t, md.Currency) != "RUB" {
		t.Errorf("Currency = %v, want RUB", md.Currency)
	}
	if stringValue(t, md.FaceUnit) != "RUB" {
		t.Errorf("FaceUnit = %v, want RUB", md.FaceUnit)
	}
}

func TestParseMarketData_FallbackColumns(t *testing.T) {
	payload := []byte(`{
		"marketdata": {
			"columns": ["LCURRENTPRICE", "ACCINT", "UPDATETIME"],
			"data": [[99.5, 0.8, "2026-06-29"]]
		}
	}`)
	md, err := ParseMarketData(payload)
	if err != nil {
		t.Fatal(err)
	}
	if floatValue(t, md.Price) != 99.5 {
		t.Errorf("Price (LCURRENTPRICE) = %v, want 99.5", md.Price)
	}
	if floatValue(t, md.AccruedInterest) != 0.8 {
		t.Errorf("AccruedInterest (ACCINT) = %v, want 0.8", md.AccruedInterest)
	}
	if stringValue(t, md.MarketDataAsOf) != "2026-06-29" {
		t.Errorf("MarketDataAsOf (UPDATETIME) = %v, want 2026-06-29", md.MarketDataAsOf)
	}
}

func TestParseMarketData_EmptyBlock(t *testing.T) {
	payload := []byte(`{"marketdata": {"columns": [], "data": []}}`)
	md, err := ParseMarketData(payload)
	if err != nil {
		t.Fatal(err)
	}
	if md != nil {
		t.Fatalf("expected nil, got %#v", md)
	}
}

func TestParseMarketData_MissingBlock(t *testing.T) {
	payload := []byte(`{}`)
	md, err := ParseMarketData(payload)
	if err != nil {
		t.Fatal(err)
	}
	if md != nil {
		t.Fatalf("expected nil, got %#v", md)
	}
}

func TestParseMarketData_MalformedJSON(t *testing.T) {
	_, err := ParseMarketData([]byte(`not json`))
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, apperrors.ErrParseError) {
		t.Fatalf("error = %v, want ErrParseError", err)
	}
}

func TestParseMarketData_InvalidNumeric(t *testing.T) {
	payload := []byte(`{
		"marketdata": {
			"columns": ["LAST"],
			"data": [["not-a-number"]]
		}
	}`)
	_, err := ParseMarketData(payload)
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, apperrors.ErrParseError) {
		t.Fatalf("error = %v, want ErrParseError", err)
	}
}

func TestParseMarketData_NullValues(t *testing.T) {
	payload := []byte(`{
		"marketdata": {
			"columns": ["LAST", "YIELD"],
			"data": [[null, null]]
		}
	}`)
	md, err := ParseMarketData(payload)
	if err != nil {
		t.Fatal(err)
	}
	if md.Price != nil {
		t.Errorf("Price should be nil for null, got %v", md.Price)
	}
	if md.YieldToMaturity != nil {
		t.Errorf("YieldToMaturity should be nil for null, got %v", md.YieldToMaturity)
	}
}

func TestParseMarketData_StringNumbers(t *testing.T) {
	payload := []byte(`{
		"marketdata": {
			"columns": ["LAST", "NUMTRADES"],
			"data": [["102.5", "37"]]
		}
	}`)
	md, err := ParseMarketData(payload)
	if err != nil {
		t.Fatal(err)
	}
	if floatValue(t, md.Price) != 102.5 {
		t.Errorf("Price = %v, want 102.5", md.Price)
	}
	if intValue(t, md.NumTrades) != 37 {
		t.Errorf("NumTrades = %v, want 37", md.NumTrades)
	}
}
