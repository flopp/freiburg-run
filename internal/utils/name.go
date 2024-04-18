package utils

import (
	"sort"
	"strings"
	"unicode"

	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

func SanitizeName(s string) string {
	sanitized := strings.ToLower(s)
	sanitized = strings.ReplaceAll(sanitized, "ä", "ae")
	sanitized = strings.ReplaceAll(sanitized, "ö", "oe")
	sanitized = strings.ReplaceAll(sanitized, "ü", "ue")
	sanitized = strings.ReplaceAll(sanitized, "ß", "ss")
	sanitized = strings.ReplaceAll(sanitized, " ", "-")
	sanitized = strings.ReplaceAll(sanitized, ".", "-")
	sanitized = strings.ReplaceAll(sanitized, "'", "-")
	sanitized = strings.ReplaceAll(sanitized, "\"", "-")
	sanitized = strings.ReplaceAll(sanitized, "(", "-")
	sanitized = strings.ReplaceAll(sanitized, ")", "-")
	result, _, err := transform.String(transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn))), sanitized)
	if err != nil {
		result = sanitized
	}
	s = ""
	lastSp := true
	for _, char := range result {
		if char >= 'a' && char <= 'z' {
			s += string(char)
			lastSp = false
		} else if char >= '0' && char <= '9' {
			s += string(char)
			lastSp = false
		} else {
			if !lastSp {
				s += "-"
				lastSp = true
			}
		}
	}

	if lastSp && len(s) > 0 {
		return s[:len(s)-1]
	}

	return s
}

func Split(s string) []string {
	res := make([]string, 0)
	for _, ss := range strings.Split(s, ",") {
		ss = strings.TrimSpace(ss)
		if len(ss) > 0 {
			res = append(res, ss)
		}
	}
	return res
}

func SortAndUniquify(a []string) []string {
	m := make(map[string]bool)
	for _, s := range a {
		m[s] = true
	}

	tags := make([]string, 0, len(m))
	for tag := range m {
		tags = append(tags, tag)
	}
	sort.Slice(tags, func(i, j int) bool { return tags[i] < tags[j] })
	return tags
}
