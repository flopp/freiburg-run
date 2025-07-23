package utils

import "testing"

func TestStripUrl(t *testing.T) {
	cases := []struct {
		input    string
		expected string
	}{
		{"https://www.example.com/index.html", "example.com"},
		{"http://example.com/index.htm", "example.com"},
		{"https://example.com/index.php", "example.com"},
		{"https://example.com/?foo=bar", "example.com"},
		{"https://example.com#hash", "example.com"},
		{"https://example.com/", "example.com"},
		{"http://www.example.com/", "example.com"},
		{"https://example.com", "example.com"},
		{"www.example.com", "example.com"},
		{"example.com", "example.com"},
		{"", ""},
	}
	for _, tc := range cases {
		got := stripUrl(tc.input)
		if got != tc.expected {
			t.Errorf("stripUrl(%q) = %q; want %q", tc.input, got, tc.expected)
		}
	}
}

func TestCreateUnnamedLink(t *testing.T) {
	cases := []struct {
		url      string
		expected string
	}{
		{"https://www.example.com/index.html", "example.com"},
		{"http://example.com/index.htm", "example.com"},
		{"https://example.com/index.php", "example.com"},
		{"https://example.com/?foo=bar", "example.com"},
		{"https://example.com#hash", "example.com"},
		{"https://example.com/", "example.com"},
		{"http://www.example.com/", "example.com"},
		{"https://example.com", "example.com"},
		{"www.example.com", "example.com"},
		{"example.com", "example.com"},
		{"", ""},
	}
	for _, tc := range cases {
		link := CreateUnnamedLink(tc.url)
		if link.Name != tc.expected {
			t.Errorf("CreateUnnamedLink(%q).Name = %q; want %q", tc.url, link.Name, tc.expected)
		}
		if link.Url != tc.url {
			t.Errorf("CreateUnnamedLink(%q).Url = %q; want %q", tc.url, link.Url, tc.url)
		}
	}
}
