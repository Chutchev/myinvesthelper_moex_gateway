package moex

import (
	"errors"
	"testing"

	"github.com/Chutchev/myinvesthelper_moex_gateway/internal/apperrors"
)

func TestIndexedRow(t *testing.T) {
	cols := []string{"A", "B", "C"}
	row := []any{"x", "y", "z"}
	m := indexedRow(cols, row)
	if len(m) != 3 || m["A"] != "x" || m["B"] != "y" || m["C"] != "z" {
		t.Fatalf("indexedRow = %v", m)
	}
}

func TestIndexedRowShortRow(t *testing.T) {
	cols := []string{"A", "B", "C"}
	row := []any{"x"}
	m := indexedRow(cols, row)
	if m["A"] != "x" || m["B"] != nil || m["C"] != nil {
		t.Fatalf("indexedRow with short row = %v", m)
	}
}

func TestPickValue(t *testing.T) {
	row := map[string]any{"A": nil, "B": "", "C": "found", "D": "also"}
	if v := pickValue(row, "A", "B", "C"); v != "found" {
		t.Fatalf("pickValue = %v, want found", v)
	}
	if v := pickValue(row, "A", "B", "Z"); v != nil {
		t.Fatalf("pickValue missing = %v, want nil", v)
	}
}

func TestParseOptionalInt(t *testing.T) {
	tests := []struct {
		name  string
		raw   any
		want  *int
		isErr bool
	}{
		{"nil", nil, nil, false},
		{"empty_string", "", nil, false},
		{"float64", float64(42), ptrInt(42), false},
		{"string_number", "17", ptrInt(17), false},
		{"string_invalid", "abc", nil, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseOptionalInt(tc.raw)
			if tc.isErr && err == nil {
				t.Fatal("expected error")
			}
			if !tc.isErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tc.want == nil {
				if got != nil {
					t.Fatalf("got %v, want nil", got)
				}
			} else {
				if got == nil {
					t.Fatal("got nil, want non-nil")
				} else if *got != *tc.want {
					t.Fatalf("got %d, want %d", *got, *tc.want)
				}
			}
		})
	}
}

func TestParseOptionalFloat(t *testing.T) {
	tests := []struct {
		name  string
		raw   any
		want  *float64
		isErr bool
	}{
		{"nil", nil, nil, false},
		{"empty_string", "", nil, false},
		{"float64", float64(3.14), ptrFloat(3.14), false},
		{"string_number", "2.5", ptrFloat(2.5), false},
		{"string_invalid", "abc", nil, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseOptionalFloat(tc.raw)
			if tc.isErr && err == nil {
				t.Fatal("expected error")
			}
			if !tc.isErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tc.want == nil {
				if got != nil {
					t.Fatalf("got %v, want nil", got)
				}
			} else {
				if got == nil {
					t.Fatal("got nil, want non-nil")
				} else if *got != *tc.want {
					t.Fatalf("got %f, want %f", *got, *tc.want)
				}
			}
		})
	}
}

func TestParseOptionalBool(t *testing.T) {
	tests := []struct {
		name  string
		raw   any
		want  *bool
		isErr bool
	}{
		{"nil", nil, nil, false},
		{"empty_string", "", nil, false},
		{"bool_true", true, ptrBool(true), false},
		{"bool_false", false, ptrBool(false), false},
		{"float64_one", float64(1), ptrBool(true), false},
		{"float64_zero", float64(0), ptrBool(false), false},
		{"float64_other", float64(2), nil, true},
		{"string_yes", "yes", ptrBool(true), false},
		{"string_no", "no", ptrBool(false), false},
		{"string_one", "1", ptrBool(true), false},
		{"string_zero", "0", ptrBool(false), false},
		{"string_true", "true", ptrBool(true), false},
		{"string_false", "false", ptrBool(false), false},
		{"string_invalid", "abc", nil, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseOptionalBool(tc.raw)
			if tc.isErr && err == nil {
				t.Fatal("expected error")
			}
			if !tc.isErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tc.want == nil {
				if got != nil {
					t.Fatalf("got %v, want nil", got)
				}
			} else {
				if got == nil {
					t.Fatal("got nil, want non-nil")
				} else if *got != *tc.want {
					t.Fatalf("got %v, want %v", *got, *tc.want)
				}
			}
		})
	}
}

func TestNormalizeCouponType(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"переменная", "floating"},
		{"плавающая", "floating"},
		{"floating", "floating"},
		{"variable", "floating"},
		{"фиксированная", "fixed"},
		{"постоянная", "fixed"},
		{"fixed", "fixed"},
		{"constant", "fixed"},
		{"unknown", "unknown"},
	}
	for _, tc := range tests {
		if got := normalizeCouponType(tc.input); got != tc.want {
			t.Errorf("normalizeCouponType(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestParseOptionalIntErrIsParseError(t *testing.T) {
	_, err := parseOptionalInt("not-a-number")
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, apperrors.ErrParseError) {
		t.Fatalf("error = %v, want ErrParseError", err)
	}
}

// helpers
func ptrInt(n int) *int       { return &n }
func ptrFloat(f float64) *float64 { return &f }
func ptrBool(b bool) *bool   { return &b }
