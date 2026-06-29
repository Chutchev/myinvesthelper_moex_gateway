package moex

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseBondization_Success(t *testing.T) {
	// Build response with coupons and principal blocks
	couponsBlock := map[string]any{
		"columns": []string{"COUPONDATE", "VALUE", "TAXABLE"},
		"data": [][]any{
			{"2025-03-15", "1234.56", "true"},
			{"2025-06-15", "1234.56", "true"},
		},
	}
	principalBlock := map[string]any{
		"columns": []string{"AMORTDATE", "VALUE", "TAXABLE"},
		"data": [][]any{
			{"2026-01-15", "10000", "false"},
		},
	}
	resp := map[string]any{
		"coupons":   couponsBlock,
		"principal": principalBlock,
	}
	data, err := json.Marshal(resp)
	require.NoError(t, err)

	coupons, cashflows, err := ParseBondization(data)
	require.NoError(t, err)

	assert.Len(t, coupons, 2)
	assert.Equal(t, "2025-03-15", coupons[0].Date)
	assert.Equal(t, "2025-06-15", coupons[1].Date)
	assert.Equal(t, 1234.56, coupons[0].Value)
	assert.Equal(t, 1234.56, coupons[1].Value)

	assert.Len(t, cashflows, 3) // 2 coupons + 1 principal
	// Coupons are first (sorted by date, then kind)
	assert.Equal(t, "coupon", cashflows[0].Kind)
	assert.Equal(t, "2025-03-15", cashflows[0].Date)
	assert.Equal(t, 1234.56, cashflows[0].Amount)
	assert.True(t, cashflows[0].Taxable)

	assert.Equal(t, "coupon", cashflows[1].Kind)
	assert.Equal(t, "2025-06-15", cashflows[1].Date)
	assert.Equal(t, 1234.56, cashflows[1].Amount)
	assert.True(t, cashflows[1].Taxable)

	// Principal is last
	assert.Equal(t, "principal", cashflows[2].Kind)
	assert.Equal(t, "2026-01-15", cashflows[2].Date)
	assert.Equal(t, 10000.0, cashflows[2].Amount)
	assert.False(t, cashflows[2].Taxable)
}

func TestParseBondization_FieldFallbacks(t *testing.T) {
	// Test DATE fallback for coupons
	couponsBlock := map[string]any{
		"columns": []string{"DATE", "COUPONVALUE"},
		"data": [][]any{
			{"2025-03-15", "500"},
		},
	}
	// Test AMORTDATE/DATE/MATDATE fallback for principal
	principalBlock := map[string]any{
		"columns": []string{"MATDATE", "FACEVALUE"},
		"data": [][]any{
			{"2026-01-15", "10000"},
		},
	}
	resp := map[string]any{
		"coupons":   couponsBlock,
		"principal": principalBlock,
	}
	data, err := json.Marshal(resp)
	require.NoError(t, err)

	coupons, cashflows, err := ParseBondization(data)
	require.NoError(t, err)

	assert.Len(t, coupons, 1)
	assert.Equal(t, "2025-03-15", coupons[0].Date)
	assert.Equal(t, 500.0, coupons[0].Value)

	assert.Len(t, cashflows, 2)
	assert.Equal(t, "2026-01-15", cashflows[1].Date)
	assert.Equal(t, 10000.0, cashflows[1].Amount)
}

func TestParseBondization_ValueFallback(t *testing.T) {
	// Test VALUEPRC fallback for principal
	principalBlock := map[string]any{
		"columns": []string{"DATE", "VALUEPRC"},
		"data": [][]any{
			{"2026-01-15", "100"},
		},
	}
	resp := map[string]any{
		"principal": principalBlock,
	}
	data, err := json.Marshal(resp)
	require.NoError(t, err)

	_, cashflows, err := ParseBondization(data)
	require.NoError(t, err)

	assert.Len(t, cashflows, 1)
	assert.Equal(t, 100.0, cashflows[0].Amount)
}

func TestParseBondization_EmptyBlocks(t *testing.T) {
	resp := map[string]any{
		"coupons":   map[string]any{},
		"principal": map[string]any{},
	}
	data, err := json.Marshal(resp)
	require.NoError(t, err)

	coupons, cashflows, err := ParseBondization(data)
	require.NoError(t, err)

	assert.Empty(t, coupons)
	assert.Empty(t, cashflows)
}

func TestParseBondization_MissingBondization(t *testing.T) {
	resp := map[string]any{}
	data, err := json.Marshal(resp)
	require.NoError(t, err)

	coupons, cashflows, err := ParseBondization(data)
	require.NoError(t, err)

	assert.Empty(t, coupons)
	assert.Empty(t, cashflows)
}

func TestParseBondization_MalformedJSON(t *testing.T) {
	_, _, err := ParseBondization([]byte("not json"))
	assert.Error(t, err)
}

func TestParseBondization_InvalidNumericValue(t *testing.T) {
	couponsBlock := map[string]any{
		"columns": []string{"COUPONDATE", "VALUE"},
		"data": [][]any{
			{"2025-03-15", "not-a-number"},
		},
	}
	resp := map[string]any{
		"coupons": couponsBlock,
	}
	data, err := json.Marshal(resp)
	require.NoError(t, err)

	_, _, err = ParseBondization(data)
	assert.Error(t, err)
}

func TestParseBondization_NullValues(t *testing.T) {
	couponsBlock := map[string]any{
		"columns": []string{"COUPONDATE", "VALUE"},
		"data": [][]any{
			{nil, nil},
		},
	}
	resp := map[string]any{
		"coupons": couponsBlock,
	}
	data, err := json.Marshal(resp)
	require.NoError(t, err)

	coupons, cashflows, err := ParseBondization(data)
	require.NoError(t, err)

	// Should skip rows with null date
	assert.Empty(t, coupons)
	assert.Empty(t, cashflows)
}

func TestParseBondization_Sorting(t *testing.T) {
	// Create cashflows that should be sorted by (date, kind)
	couponsBlock := map[string]any{
		"columns": []string{"COUPONDATE", "VALUE"},
		"data": [][]any{
			{"2025-06-15", "200"},
			{"2025-03-15", "100"},
		},
	}
	principalBlock := map[string]any{
		"columns": []string{"DATE", "VALUE"},
		"data": [][]any{
			{"2025-03-15", "10000"}, // Same date as second coupon, should come after
		},
	}
	resp := map[string]any{
		"coupons":   couponsBlock,
		"principal": principalBlock,
	}
	data, err := json.Marshal(resp)
	require.NoError(t, err)

	_, cashflows, err := ParseBondization(data)
	require.NoError(t, err)

	// Verify sorting: 2025-03-15 coupon, 2025-03-15 principal, 2025-06-15 coupon
	assert.Equal(t, "2025-03-15", cashflows[0].Date)
	assert.Equal(t, "coupon", cashflows[0].Kind)

	assert.Equal(t, "2025-03-15", cashflows[1].Date)
	assert.Equal(t, "principal", cashflows[1].Kind)

	assert.Equal(t, "2025-06-15", cashflows[2].Date)
	assert.Equal(t, "coupon", cashflows[2].Kind)
}

func TestParseBondization_LegalClosePriceFallback(t *testing.T) {
	// Test FACEVALUE fallback for principal value (LEGALCLOSEPRICE is not in the fallback chain)
	principalBlock := map[string]any{
		"columns": []string{"DATE", "FACEVALUE"},
		"data": [][]any{
			{"2026-01-15", "9500"},
		},
	}
	resp := map[string]any{
		"principal": principalBlock,
	}
	data, err := json.Marshal(resp)
	require.NoError(t, err)

	_, cashflows, err := ParseBondization(data)
	require.NoError(t, err)

	assert.Len(t, cashflows, 1)
	assert.Equal(t, 9500.0, cashflows[0].Amount)
}
