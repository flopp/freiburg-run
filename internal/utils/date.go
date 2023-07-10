package utils

import (
	"fmt"
	"regexp"
	"time"
)

var dateRe1 = regexp.MustCompile(`^\s*(\d+)\.(\d+)\.(\d\d\d\d)\s*$`)
var dateRe2 = regexp.MustCompile(`^\s*(\d\d\d\d)-(\d+)-(\d+)\s*$`)

func DateYMS(s string) string {
	m := dateRe1.FindStringSubmatch(s)
	if m != nil {
		return fmt.Sprintf("%s-%s-%s", m[3], m[2], m[1])
	}
	m = dateRe2.FindStringSubmatch(s)
	if m != nil {
		return fmt.Sprintf("%s-%s-%s", m[1], m[2], m[3])
	}
	return ""
}

func ParseDate(s string) (time.Time, error) {
	d, err := time.Parse("2006-01-02", s)
	if err == nil {
		return d, nil
	}

	return time.Parse("02.01.2006", s)
}
