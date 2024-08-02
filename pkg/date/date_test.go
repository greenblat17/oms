package date

import (
	"testing"
	"time"
)

func TestParseDateToUTC(t *testing.T) {
	tests := []struct {
		input       string
		expected    time.Time
		expectError bool
	}{
		{"15.04.2023", time.Date(2023, 4, 15, 0, 0, 0, 0, time.UTC), false},
		{"01.01.2000", time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC), false},
		{"31.12.1999", time.Date(1999, 12, 31, 0, 0, 0, 0, time.UTC), false},
		{"invalid_date", time.Time{}, true},
	}

	for _, tt := range tests {
		result, err := ParseDateToUTC(tt.input)
		if tt.expectError {
			if err == nil {
				t.Errorf("expected error for input %s, got nil", tt.input)
			}
		} else {
			if err != nil {
				t.Errorf("unexpected error for input %s: %v", tt.input, err)
			}
			if !result.Equal(tt.expected) {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		}
	}
}

func TestConvertUTCToStr(t *testing.T) {
	tests := []struct {
		input    time.Time
		expected string
	}{
		{time.Date(2023, 4, 15, 0, 0, 0, 0, time.UTC), "15.04.2023"},
		{time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC), "01.01.2000"},
		{time.Date(1999, 12, 31, 0, 0, 0, 0, time.UTC), "31.12.1999"},
	}

	for _, tt := range tests {
		result := ConvertUTCToStr(tt.input)
		if result != tt.expected {
			t.Errorf("expected %s, got %s", tt.expected, result)
		}
	}
}
