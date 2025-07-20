package utils

import (
	"path"
	"strings"
)

type Url string

func (u Url) Join(s string) string {
	if len(s) == 0 {
		return string(u)
	}
	return path.Join(string(u), s)
}

func ExtractDomain(url string) string {
	if len(url) == 0 {
		return ""
	}

	// Remove the scheme (http:// or https://) if present
	if strings.HasPrefix(url, "http://") {
		url = url[7:]
	} else if strings.HasPrefix(url, "https://") {
		url = url[8:]
	}

	// Find the first slash, question mark, or hash and truncate
	if idx := strings.IndexAny(url, "/?#"); idx != -1 {
		url = url[:idx]
	}

	return url
}
