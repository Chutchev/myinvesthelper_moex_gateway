package moex

import (
	"encoding/json"
	"testing"
)

func TestCouponJSONShape(t *testing.T) {
	data, err := json.Marshal(Coupon{Date: "2026-07-01", Value: 42.5})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := string(data), `{"date":"2026-07-01","value":42.5}`; got != want {
		t.Fatalf("coupon JSON = %s, want %s", got, want)
	}
}

func TestCashflowJSONShape(t *testing.T) {
	data, err := json.Marshal(Cashflow{
		Date:    "2026-07-01",
		Amount:  1000,
		Kind:    "principal",
		Taxable: false,
	})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := string(data), `{"date":"2026-07-01","amount":1000,"kind":"principal","taxable":false}`; got != want {
		t.Fatalf("cashflow JSON = %s, want %s", got, want)
	}
}

func TestMarketUniverseJSONShape(t *testing.T) {
	data, err := json.Marshal(MarketUniverse{{ISIN: "RU000A000001"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(data) == 0 || data[0] != '[' {
		t.Fatalf("market universe must serialize as array, got %s", data)
	}
}
