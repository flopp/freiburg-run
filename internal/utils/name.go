package utils

import (
	"sort"
	"strings"
	"unicode"

	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

// specialCharReplacer is used to replace special characters in names.
var specialCharReplacer = strings.NewReplacer(
	"ä", "ae",
	"ö", "oe",
	"ü", "ue",
	"ß", "ss",
	" ", "-",
	".", "-",
	"'", "-",
	"\"", "-",
	"(", "-",
	")", "-",
)

func SanitizeName(s string) string {
	// lowercase
	sanitized := strings.ToLower(s)

	// replace special characters
	sanitized = specialCharReplacer.Replace(sanitized)

	// remove all other special characters
	result, _, err := transform.String(transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn))), sanitized)
	if err != nil {
		result = sanitized
	}

	// remove all non-alphanumeric characters & replace with '-'; avoid leading, trailing and consecutive '-'
	var builder strings.Builder
	builder.Grow(len(result)) // Pre-allocate capacity
	needSep := false
	for _, char := range result {
		if ('a' <= char && char <= 'z') || ('0' <= char && char <= '9') {
			if needSep && builder.Len() > 0 {
				builder.WriteByte('-')
			}
			needSep = false
			builder.WriteByte(byte(char))
		} else {
			needSep = true
		}
	}

	return builder.String()
}

// SplitList splits a string by commas and trims whitespace from each part (ignoring empty parts).
func SplitList(s string) []string {
	if s == "" {
		return nil
	}

	res := make([]string, 0)
	start := 0
	for i := 0; i <= len(s); i++ {
		if i == len(s) || s[i] == ',' {
			// Extract substring and trim whitespace
			part := strings.TrimSpace(s[start:i])
			if len(part) != 0 {
				res = append(res, part)
			}
			start = i + 1
		}
	}

	return res
}

func SplitPair(s string) (string, string) {
	i := strings.Index(s, "|")
	if i > -1 {
		return s[:i], s[i+1:]
	}
	return s, ""
}

// SortAndUniquify sorts a slice of strings and removes duplicates.
func SortAndUniquify(items []string) []string {
	uniqueMap := make(map[string]struct{}, len(items))
	for _, item := range items {
		uniqueMap[item] = struct{}{} // Empty struct uses 0 bytes
	}

	uniqueItems := make([]string, 0, len(uniqueMap))
	for item := range uniqueMap {
		uniqueItems = append(uniqueItems, item)
	}

	sort.Strings(uniqueItems)

	return uniqueItems
}

func IsSimilarName(s1, s2 string) bool {
	var builder1 strings.Builder
	for _, r := range s1 {
		if unicode.IsLetter(r) {
			builder1.WriteRune(unicode.ToLower(r))
		}
	}
	var builder2 strings.Builder
	for _, r := range s2 {
		if unicode.IsLetter(r) {
			builder2.WriteRune(unicode.ToLower(r))
		}
	}
	return builder1.String() == builder2.String()
}
