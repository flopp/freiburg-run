package utils

import (
	"fmt"
	"regexp"
	"strings"
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
	loc, _ := time.LoadLocation("Europe/Berlin")

	d, err := time.ParseInLocation("2006-01-02", s, loc)
	if err == nil {
		return d, nil
	}

	return time.ParseInLocation("02.01.2006", s, loc)
}

type TimeRange struct {
	Original  string
	Formatted string
	From      time.Time
	To        time.Time
}

func (tr TimeRange) IsZero() bool {
	return tr.From.IsZero()
}

func (tr TimeRange) Year() int {
	if tr.IsZero() {
		return 0
	}
	return tr.From.Year()
}

func (tr TimeRange) Before(t time.Time) bool {
	if tr.From.IsZero() {
		return false
	}
	return tr.To.Before(t)
}

var dateRe = regexp.MustCompile(`\b(\d\d\.\d\d\.\d\d\d\d)\b`)

func CreateTimeRange(original string) (TimeRange, error) {
	replacements := make(map[string]string)
	var from, to time.Time

	for _, mm := range dateRe.FindAllStringSubmatch(original, -1) {
		d, err := ParseDate(mm[1])
		if err != nil {
			return TimeRange{original, original, from, to}, fmt.Errorf("cannot parse date '%s' from '%s'", mm[1], original)
		}

		replacements[mm[1]] = fmt.Sprintf("%s, %s", WeekdayStr(d.Weekday()), mm[1])

		if from.IsZero() {
			from = d
			to = d
		} else {
			if d.Before(from) {
				from = d
			} else if d.After(to) {
				to = d
			}
		}
	}

	// insert weekdays
	formatted := original
	for key, value := range replacements {
		formatted = strings.ReplaceAll(formatted, key, value)
	}

	return TimeRange{original, formatted, from, to}, nil
}

func WeekdayStr(d time.Weekday) string {
	switch d {
	case time.Monday:
		return "Montag"
	case time.Tuesday:
		return "Dienstag"
	case time.Wednesday:
		return "Mittwoch"
	case time.Thursday:
		return "Donnerstag"
	case time.Friday:
		return "Freitag"
	case time.Saturday:
		return "Samstag"
	case time.Sunday:
		return "Sonntag"
	}
	return "Sonntag"
}

func MonthStr(m time.Month) string {
	switch m {
	case time.January:
		return "Januar"
	case time.February:
		return "Februar"
	case time.March:
		return "März"
	case time.April:
		return "April"
	case time.May:
		return "Mai"
	case time.June:
		return "Juni"
	case time.July:
		return "Juli"
	case time.August:
		return "August"
	case time.September:
		return "September"
	case time.October:
		return "Oktober"
	case time.November:
		return "November"
	case time.December:
		return "Dezember"
	}
	return "Dezember"
}
