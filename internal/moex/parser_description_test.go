package moex

import (
	"errors"
	"testing"

	"github.com/Chutchev/myinvesthelper_moex_gateway/internal/apperrors"
)

func TestParseBondDescription_Success(t *testing.T) {
	payload := []byte(`{
		"description": {
			"columns": ["key", "type", "value"],
			"data": [
				["NAME", "string", "Test Bond"],
				["SHORTNAME", "string", "TST"],
				["MATDATE", "date", "2030-01-01"],
				["FACEVALUE", "number", 1000],
				["FACEUNIT", "string", "RUB"],
				["CURRENCY", "string", "RUB"],
				["ISSUESIZE", "number", 5000000000],
				["COUPONPERCENT", "number", 5.5],
				["COUPONVALUE", "number", 27.5],
				["COUPONTYPE", "string", "фиксированная"],
				["COUPONFREQUENCY", "number", 4],
				["ISSUEDATE", "date", "2025-01-01"],
				["STARTDATEMOEX", "date", "2025-01-05"],
				["REGISTRYDATE", "date", "2025-01-01"],
				["REGNUMBER", "string", "AI-001"],
				["ISSUERID", "number", 123],
				["BONDTYPE", "string", "corporate"],
				["BONDSUBTYPE", "string", "senior"],
				["LATINNAME", "string", "Test Bond LLC"],
				["TYPE", "string", "bond"],
				["TYPENAME", "string", "Corporate Bond"],
				["LISTLEVEL", "number", 1],
				["ISQUALIFIEDINVESTORS", "boolean", false],
				["HASPROSPECTUS", "boolean", true],
				["HASDEFAULT", "boolean", false],
				["HASTECHNICALDEFAULT", "boolean", false],
				["NEXTCOUPONDATE", "date", "2026-07-01"],
				["DAYS TO REDEMPTION", "number", 520],
				["GROUPNAME", "string", "bonds"]
			]
		}
	}`)

	bond, err := ParseBondDescription(payload)
	if err != nil {
		t.Fatal(err)
	}

	if stringValue(t, bond.Name) != "Test Bond" {
		t.Errorf("Name = %v, want Test Bond", bond.Name)
	}
	if stringValue(t, bond.ShortName) != "TST" {
		t.Errorf("ShortName = %v, want TST", bond.ShortName)
	}
	if stringValue(t, bond.MaturityDate) != "2030-01-01" {
		t.Errorf("MaturityDate = %v, want 2030-01-01", bond.MaturityDate)
	}
	if floatValue(t, bond.FaceValue) != 1000 {
		t.Errorf("FaceValue = %v, want 1000", bond.FaceValue)
	}
	if floatValue(t, bond.InitialFaceValue) != 1000 {
		t.Errorf("InitialFaceValue = %v, want 1000", bond.InitialFaceValue)
	}
	if stringValue(t, bond.FaceUnit) != "RUB" {
		t.Errorf("FaceUnit = %v, want RUB", bond.FaceUnit)
	}
	if stringValue(t, bond.Currency) != "RUB" {
		t.Errorf("Currency = %v, want RUB", bond.Currency)
	}
	if floatValue(t, bond.IssueSize) != 5000000000 {
		t.Errorf("IssueSize = %v, want 5000000000", bond.IssueSize)
	}
	if floatValue(t, bond.CouponPercent) != 5.5 {
		t.Errorf("CouponPercent = %v, want 5.5", bond.CouponPercent)
	}
	if floatValue(t, bond.CouponValue) != 27.5 {
		t.Errorf("CouponValue = %v, want 27.5", bond.CouponValue)
	}
	if stringValue(t, bond.CouponType) != "fixed" {
		t.Errorf("CouponType = %v, want fixed", bond.CouponType)
	}
	if intValue(t, bond.CouponFrequency) != 4 {
		t.Errorf("CouponFrequency = %v, want 4", bond.CouponFrequency)
	}
	if stringValue(t, bond.IssueDate) != "2025-01-01" {
		t.Errorf("IssueDate = %v, want 2025-01-01", bond.IssueDate)
	}
	if stringValue(t, bond.StartDateMOEX) != "2025-01-05" {
		t.Errorf("StartDateMOEX = %v, want 2025-01-05", bond.StartDateMOEX)
	}
	if stringValue(t, bond.RegistryDate) != "2025-01-01" {
		t.Errorf("RegistryDate = %v, want 2025-01-01", bond.RegistryDate)
	}
	if stringValue(t, bond.RegNumber) != "AI-001" {
		t.Errorf("RegNumber = %v, want AI-001", bond.RegNumber)
	}
	if intValue(t, bond.IssuerID) != 123 {
		t.Errorf("IssuerID = %v, want 123", bond.IssuerID)
	}
	if stringValue(t, bond.BondType) != "corporate" {
		t.Errorf("BondType = %v, want corporate", bond.BondType)
	}
	if stringValue(t, bond.BondSubtype) != "senior" {
		t.Errorf("BondSubtype = %v, want senior", bond.BondSubtype)
	}
	if stringValue(t, bond.LatinName) != "Test Bond LLC" {
		t.Errorf("LatinName = %v, want Test Bond LLC", bond.LatinName)
	}
	if stringValue(t, bond.Type) != "bond" {
		t.Errorf("Type = %v, want bond", bond.Type)
	}
	if stringValue(t, bond.TypeName) != "Corporate Bond" {
		t.Errorf("TypeName = %v, want Corporate Bond", bond.TypeName)
	}
	if intValue(t, bond.ListLevel) != 1 {
		t.Errorf("ListLevel = %v, want 1", bond.ListLevel)
	}
	if boolValue(t, bond.IsQualifiedInvestors) != false {
		t.Errorf("IsQualifiedInvestors = %v, want false", bond.IsQualifiedInvestors)
	}
	if boolValue(t, bond.HasProspectus) != true {
		t.Errorf("HasProspectus = %v, want true", bond.HasProspectus)
	}
	if boolValue(t, bond.HasDefault) != false {
		t.Errorf("HasDefault = %v, want false", bond.HasDefault)
	}
	if boolValue(t, bond.HasTechnicalDefault) != false {
		t.Errorf("HasTechnicalDefault = %v, want false", bond.HasTechnicalDefault)
	}
	if stringValue(t, bond.NextCouponDate) != "2026-07-01" {
		t.Errorf("NextCouponDate = %v, want 2026-07-01", bond.NextCouponDate)
	}
	if intValue(t, bond.DaysToRedemption) != 520 {
		t.Errorf("DaysToRedemption = %v, want 520", bond.DaysToRedemption)
	}
	if stringValue(t, bond.InstrumentGroup) != "bonds" {
		t.Errorf("InstrumentGroup = %v, want bonds", bond.InstrumentGroup)
	}
}

func TestParseBondDescription_CouponTypeFloating(t *testing.T) {
	payload := []byte(`{
		"description": {
			"columns": ["key", "type", "value"],
			"data": [
				["COUPONTYPE", "string", "плавающая"]
			]
		}
	}`)
	bond, err := ParseBondDescription(payload)
	if err != nil {
		t.Fatal(err)
	}
	if stringValue(t, bond.CouponType) != "floating" {
		t.Errorf("CouponType = %v, want floating", bond.CouponType)
	}
}

func TestParseBondDescription_CouponTypeFallback(t *testing.T) {
	payload := []byte(`{
		"description": {
			"columns": ["key", "type", "value"],
			"data": [
				["COUPON_TYPE", "string", "fixed"]
			]
		}
	}`)
	bond, err := ParseBondDescription(payload)
	if err != nil {
		t.Fatal(err)
	}
	if stringValue(t, bond.CouponType) != "fixed" {
		t.Errorf("CouponType = %v, want fixed", bond.CouponType)
	}
}

func TestParseBondDescription_MaturityDateFallback(t *testing.T) {
	payload := []byte(`{
		"description": {
			"columns": ["key", "type", "value"],
			"data": [
				["MATURITYDATE", "date", "2031-06-01"]
			]
		}
	}`)
	bond, err := ParseBondDescription(payload)
	if err != nil {
		t.Fatal(err)
	}
	if stringValue(t, bond.MaturityDate) != "2031-06-01" {
		t.Errorf("MaturityDate = %v, want 2031-06-01", bond.MaturityDate)
	}
}

func TestParseBondDescription_FaceValueFallback(t *testing.T) {
	payload := []byte(`{
		"description": {
			"columns": ["key", "type", "value"],
			"data": [
				["INITIALFACEVALUE", "number", 5000]
			]
		}
	}`)
	bond, err := ParseBondDescription(payload)
	if err != nil {
		t.Fatal(err)
	}
	if floatValue(t, bond.FaceValue) != 5000 {
		t.Errorf("FaceValue = %v, want 5000", bond.FaceValue)
	}
}

func TestParseBondDescription_MalformedJSON(t *testing.T) {
	_, err := ParseBondDescription([]byte(`not json`))
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, apperrors.ErrParseError) {
		t.Fatalf("error = %v, want ErrParseError", err)
	}
}

func TestParseBondDescription_MissingBlock(t *testing.T) {
	_, err := ParseBondDescription([]byte(`{}`))
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, apperrors.ErrParseError) {
		t.Fatalf("error = %v, want ErrParseError", err)
	}
}

func TestParseBondDescription_EmptyData(t *testing.T) {
	_, err := ParseBondDescription([]byte(`{"description": {"columns": [], "data": []}}`))
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, apperrors.ErrParseError) {
		t.Fatalf("error = %v, want ErrParseError", err)
	}
}

func TestParseBondDescription_InvalidNumeric(t *testing.T) {
	payload := []byte(`{
		"description": {
			"columns": ["key", "type", "value"],
			"data": [
				["FACEVALUE", "string", "not-a-number"]
			]
		}
	}`)
	_, err := ParseBondDescription(payload)
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, apperrors.ErrParseError) {
		t.Fatalf("error = %v, want ErrParseError", err)
	}
}

func TestParseBondDescription_NullValues(t *testing.T) {
	payload := []byte(`{
		"description": {
			"columns": ["key", "type", "value"],
			"data": [
				["NAME", "string", null],
				["FACEVALUE", "number", null]
			]
		}
	}`)
	bond, err := ParseBondDescription(payload)
	if err != nil {
		t.Fatal(err)
	}
	if bond.Name != nil {
		t.Errorf("Name should be nil for null, got %v", bond.Name)
	}
	if bond.FaceValue != nil {
		t.Errorf("FaceValue should be nil for null, got %v", bond.FaceValue)
	}
}

func TestParseBondDescription_StringNumbers(t *testing.T) {
	payload := []byte(`{
		"description": {
			"columns": ["key", "type", "value"],
			"data": [
				["FACEVALUE", "string", "1000.5"],
				["COUPONFREQUENCY", "string", "2"]
			]
		}
	}`)
	bond, err := ParseBondDescription(payload)
	if err != nil {
		t.Fatal(err)
	}
	if floatValue(t, bond.FaceValue) != 1000.5 {
		t.Errorf("FaceValue = %v, want 1000.5", bond.FaceValue)
	}
	if intValue(t, bond.CouponFrequency) != 2 {
		t.Errorf("CouponFrequency = %v, want 2", bond.CouponFrequency)
	}
}

func TestParseBondDescription_BoolVariants(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  bool
	}{
		{"true", "true", true},
		{"false", "false", false},
		{"1", "1", true},
		{"0", "0", false},
		{"yes", "yes", true},
		{"no", "no", false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			payload := []byte(`{
				"description": {
					"columns": ["key", "type", "value"],
					"data": [
						["HASPROSPECTUS", "boolean", "` + tc.value + `"]
					]
				}
			}`)
			bond, err := ParseBondDescription(payload)
			if err != nil {
				t.Fatal(err)
			}
			if boolValue(t, bond.HasProspectus) != tc.want {
				t.Errorf("HasProspectus = %v, want %v", bond.HasProspectus, tc.want)
			}
		})
	}
}

func TestParseBondDescription_ShortRow(t *testing.T) {
	payload := []byte(`{
		"description": {
			"columns": ["key", "type", "value"],
			"data": [
				["NAME"],
				["FACEVALUE", "number", 1000]
			]
		}
	}`)
	bond, err := ParseBondDescription(payload)
	if err != nil {
		t.Fatal(err)
	}
	if bond.Name != nil {
		t.Errorf("Name should be nil for short row, got %v", bond.Name)
	}
	if floatValue(t, bond.FaceValue) != 1000 {
		t.Errorf("FaceValue = %v, want 1000", bond.FaceValue)
	}
}

// helper for bool pointers
func boolValue(t *testing.T, b *bool) bool {
	t.Helper()
	if b == nil {
		t.Fatal("expected non-nil bool pointer")
	}
	return *b
}
