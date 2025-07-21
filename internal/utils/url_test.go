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

func TestUrlJoin(t *testing.T) {
	cases := []struct {
		base     string
		part     string
		expected string
	}{
		{"https://freiburg.run", "tags.html", "https://freiburg.run/tags.html"},
		{"https://freiburg.run/", "tags.html", "https://freiburg.run//tags.html"},
		{"https://freiburg.run", "/tags.html", "https://freiburg.run//tags.html"},
		{"https://freiburg.run", "", "https://freiburg.run"},
		{"https://freiburg.run", "/", "https://freiburg.run//"},
		{"http://example.com", "foo/bar", "http://example.com/foo/bar"},
		{"http://example.com/", "foo/bar", "http://example.com//foo/bar"},
	}
	for _, tc := range cases {
		got := Url(tc.base).Join(tc.part)
		if got != tc.expected {
			t.Errorf("Url(%q).Join(%q) = %q; want %q", tc.base, tc.part, got, tc.expected)
		}
	}
}
