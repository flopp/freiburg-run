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

	d, err = time.ParseInLocation("02.01.2006", s, loc)
	if err == nil {
		return d, nil
	}
	return d, fmt.Errorf("Cannot parse date '%s' using formats '2006-01-02' or '02.01.2006'", s)
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
	dates := dateRe.FindAllStringSubmatch(original, -1)
	if dates == nil {
		// no dates found, just return as is
		return TimeRange{original, original, time.Time{}, time.Time{}}, nil
	}

	replacements := make(map[string]string)
	var from, to time.Time

	for _, d := range dates {
		dateStr := d[1]
		date, err := ParseDate(dateStr)
		if err != nil {
			return TimeRange{}, fmt.Errorf("cannot parse date '%s' from '%s'", dateStr, original)
		}

		replacements[dateStr] = fmt.Sprintf("%s, %s", WeekdayStr(date.Weekday()), dateStr)

		// update range
		if from.IsZero() {
			from = date
			to = date
		} else {
			if date.Before(from) {
				from = date
			} else if date.After(to) {
				to = date
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

var germanWeekdays = map[time.Weekday]string{
	time.Monday:    "Montag",
	time.Tuesday:   "Dienstag",
	time.Wednesday: "Mittwoch",
	time.Thursday:  "Donnerstag",
	time.Friday:    "Freitag",
	time.Saturday:  "Samstag",
	time.Sunday:    "Sonntag",
}

func WeekdayStr(d time.Weekday) string {
	if name, ok := germanWeekdays[d]; ok {
		return name
	}
	return "Sonntag"
}

var germanMonths = map[time.Month]string{
	time.January:   "Januar",
	time.February:  "Februar",
	time.March:     "März",
	time.April:     "April",
	time.May:       "Mai",
	time.June:      "Juni",
	time.July:      "Juli",
	time.August:    "August",
	time.September: "September",
	time.October:   "Oktober",
	time.November:  "November",
	time.December:  "Dezember",
}

func MonthStr(m time.Month) string {
	if name, ok := germanMonths[m]; ok {
		return name
	}
	return "Dezember"
}
