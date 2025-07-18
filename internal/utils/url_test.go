package utils

import (
	"testing"
)

func TestExtractDomain(t *testing.T) {
	testCases := []struct {
		input    string
		expected string
	}{
		{"", ""},
		{"foo", "foo"},
		{"https://example.com", "example.com"},
		{"http://example.com", "example.com"},
		{"https://example.com/sub/xxx", "example.com"},
		{"https://example.com#hash", "example.com"},
		{"https://example.com?param", "example.com"},
	}

	for _, tc := range testCases {
		result := ExtractDomain(tc.input)
		if result != tc.expected {
			t.Errorf("ExtractDomain(%q) = %q; want %q", tc.input, result, tc.expected)
		}
	}
}
