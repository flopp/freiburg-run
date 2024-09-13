package utils

import (
	"testing"
)

func TestName(t *testing.T) {
	testCases := []struct {
		input    string
		expected string
	}{
		{"", ""},
		{" Test 123 !?", "test-123"},
		{"ÄÖÜäöüß", "aeoeueaeoeuess"},
	}

	for _, tc := range testCases {
		result := SanitizeName(tc.input)
		if result != tc.expected {
			t.Errorf("SanitizeName(%q) = %q; want %q", tc.input, result, tc.expected)
		}
	}
}
