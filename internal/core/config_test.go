package core

import (
	"testing"
)

func TestIsValidLayout(t *testing.T) {
	tests := []struct {
		name   string
		layout string
		want   bool
	}{
		{"timeline is valid", "timeline", true},
		{"grid is valid", "grid", true},
		{"minimal is valid", "minimal", true},
		{"unknown layout is invalid", "unknown", false},
		{"empty string is invalid", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isValidLayout(tt.layout); got != tt.want {
				t.Errorf("isValidLayout(%q) = %v, want %v", tt.layout, got, tt.want)
			}
		})
	}
}

func TestIsValidSort(t *testing.T) {
	tests := []struct {
		name string
		sort string
		want bool
	}{
		{"newest_first is valid", "newest_first", true},
		{"oldest_first is valid", "oldest_first", true},
		{"alphabetical is valid", "alphabetical", true},
		{"random is invalid", "random", false},
		{"empty string is invalid", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isValidSort(tt.sort); got != tt.want {
				t.Errorf("isValidSort(%q) = %v, want %v", tt.sort, got, tt.want)
			}
		})
	}
}

func TestToInt(t *testing.T) {
	tests := []struct {
		name    string
		input   any
		wantVal int
		wantOk  bool
	}{
		{"int value", 42, 42, true},
		{"float64 truncates", float64(3.7), 3, true},
		{"int64 value", int64(100), 100, true},
		{"string is not convertible", "not a number", 0, false},
		{"nil is not convertible", nil, 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := toInt(tt.input)
			if ok != tt.wantOk || got != tt.wantVal {
				t.Errorf("toInt(%v) = (%d, %v), want (%d, %v)", tt.input, got, ok, tt.wantVal, tt.wantOk)
			}
		})
	}
}
