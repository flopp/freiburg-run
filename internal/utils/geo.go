package utils

import (
	"fmt"
	"regexp"
)

var geoRe1 = regexp.MustCompile(`^\s*(\d*\.?\d*)\s*,\s*(\d*\.?\d*)\s*$`)
var geoRe2 = regexp.MustCompile(`^\s*N\s*(\d*\.?\d*)\s*E\s*(\d*\.?\d*)\s*$`)

func NormalizeGeo(s string) string {
	m := geoRe1.FindStringSubmatch(s)
	if m != nil {
		return fmt.Sprintf("%s,%s", m[1], m[2])
	}
	m = geoRe2.FindStringSubmatch(s)
	if m != nil {
		return fmt.Sprintf("%s,%s", m[1], m[2])
	}
	return ""
}
