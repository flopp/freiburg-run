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
	From time.Time
	To   time.Time
}

var dateRe = regexp.MustCompile(`\b(\d\d\.\d\d\.\d\d\d\d)\b`)

func ParseTimeRange(s string) (TimeRange, error) {
	var from, to time.Time
	for _, mm := range dateRe.FindAllStringSubmatch(s, -1) {
		d, err := ParseDate(mm[1])
		if err != nil {
			return TimeRange{}, fmt.Errorf("cannot parse date '%s' from '%s'", mm[1], s)
		}
		if from.IsZero() {
			from = d
		} else {
			if d.Before(to) {
				return TimeRange{}, fmt.Errorf("invalid time range '%s' (wrongly ordered components)", s)
			}
		}
		to = d
	}

	return TimeRange{from, to}, nil
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

func InsertWeekdays(s string) (string, error) {
	replacements := make(map[string]string)

	for _, mm := range dateRe.FindAllStringSubmatch(s, -1) {
		d, err := ParseDate(mm[1])
		if err != nil {
			return s, fmt.Errorf("cannot parse date '%s' from '%s'", mm[1], s)
		}
		replacements[mm[1]] = fmt.Sprintf("%s, %s", WeekdayStr(d.Weekday()), mm[1])
	}

	// insert weekdays
	for s1, s2 := range replacements {
		s = strings.ReplaceAll(s, s1, s2)
	}

	return s, nil
}
