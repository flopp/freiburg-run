package utils

import (
	"testing"
	"time"
)

func TestParseDate(t *testing.T) {
	testCases := []struct {
		input         string
		expectedY     int
		expectedM     time.Month
		expectedD     int
		expectedError bool
	}{
		{"", 0, 0, 0, true},
		{"something", 0, 0, 0, true},
		{"31. 12. 2024", 0, 0, 0, true},
		{"2024-12-31", 2024, time.December, 31, false},
		{"2024-01-02", 2024, time.January, 2, false},
		{"2024-1-2", 0, 0, 0, true},
		{"31.12.2024", 2024, time.December, 31, false},
		{"02.01.2024", 2024, time.January, 2, false},
		{"2.1.2024", 0, 0, 0, true},
	}

	for _, tc := range testCases {
		result, err := ParseDate(tc.input)
		if err != nil {
			if !tc.expectedError {
				t.Errorf("ParseDate(%q); unexpected error: %q", tc.input, err)
			}
		} else if tc.expectedError {
			t.Errorf("ParseDate(%q) = %q; but expected an error", tc.input, result)
		} else if result.Year() != tc.expectedY || result.Month() != tc.expectedM || result.Day() != tc.expectedD {
			t.Errorf("ParseDate(%q) = %q; but expected Y=%q M=%q D=%q", tc.input, result, tc.expectedY, tc.expectedM.String(), tc.expectedD)
		}
	}
}
