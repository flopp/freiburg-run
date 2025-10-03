package events

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"time"

	ical "github.com/arran4/golang-ical"
	"github.com/flopp/freiburg-run/internal/config"
	"github.com/flopp/freiburg-run/internal/utils"
)

const (
	dateFormatUtc = "20060102"

	propertyDtStart ical.Property = "DTSTART;VALUE=DATE"
	propertyDtEnd   ical.Property = "DTEND;VALUE=DATE"

	componentPropertyDtStart = ical.ComponentProperty(propertyDtStart)
	componentPropertyDtEnd   = ical.ComponentProperty(propertyDtEnd)
)

func CreateEventCalendar(config config.Config, event *Event, now time.Time, baseUrl utils.Url, calendarUrl string, path string) error {
	infoUrl := baseUrl.Join(event.Slug())
	endPlusOneDay := event.Time.To.AddDate(0, 0, 1)

	// ical/ics data
	cal := ical.NewCalendar()
	cal.SetProductId(fmt.Sprintf("Laufevents - %s", config.Website.Name))
	cal.SetMethod(ical.MethodPublish)
	cal.SetDescription(fmt.Sprintf("Liste aller Laufevents im Raum %s (50km Umkreis)", config.City.Name))
	///cal.SetUrl(calendarUrl)
	uid, err := event.GetUUID()
	if err != nil {
		return fmt.Errorf("create UUID for '%s': %w", event.Name.Orig, err)
	}
	calEvent := cal.AddEvent(uid.String())
	calEvent.SetDtStampTime(now)
	calEvent.SetSummary(event.Name.Orig)
	calEvent.SetLocation(event.Location.NameNoFlag())
	calEvent.SetDescription(string(event.Details))
	calEvent.SetProperty(componentPropertyDtStart, event.Time.From.Format(dateFormatUtc))
	calEvent.SetProperty(componentPropertyDtEnd, endPlusOneDay.Format(dateFormatUtc))
	calEvent.SetURL(infoUrl)
	serialized := cal.Serialize()
	// Encode as data URL for download
	encoded := url.QueryEscape(serialized)
	event.CalendarDataICS = "data:text/calendar;charset=utf-8," + encoded

	// Google Calendar link
	event.CalendarGoogle = fmt.Sprintf("https://calendar.google.com/calendar/u/0/r/eventedit?text=%s&dates=%s/%s&details=%s&location=%s",
		url.QueryEscape(event.Name.Orig),
		event.Time.From.Format(dateFormatUtc),
		endPlusOneDay.Format(dateFormatUtc),
		url.QueryEscape(fmt.Sprintf(`%s<br>Infos: <a href="%s">%s</a>`, event.Details, infoUrl, config.Website.Name)),
		url.QueryEscape(event.Location.NameNoFlag()),
	)

	return nil
}

func CreateCalendar(config config.Config, eventsList []*Event, now time.Time, baseUrl utils.Url, calendarUrl string, path string) error {
	cal := ical.NewCalendar()
	cal.SetProductId(fmt.Sprintf("Laufevents - %s", config.Website.Name))
	cal.SetMethod(ical.MethodPublish)
	cal.SetDescription(fmt.Sprintf("Liste aller Laufevents im Raum %s (50km Umkreis)", config.City.Name))
	cal.SetUrl(calendarUrl)

	for _, e := range eventsList {
		if e.IsSeparator() {
			continue
		}

		uid, err := e.GetUUID()
		if err != nil {
			return fmt.Errorf("create UUID for '%s': %w", e.Name.Orig, err)
		}

		infoUrl := baseUrl.Join(e.Slug())

		calEvent := cal.AddEvent(uid.String())
		calEvent.SetDtStampTime(now)
		calEvent.SetSummary(e.Name.Orig)
		calEvent.SetLocation(e.Location.NameNoFlag())
		calEvent.SetDescription(string(e.Details))
		calEvent.SetProperty(componentPropertyDtStart, e.Time.From.Format(dateFormatUtc))
		// end + 1 day; Outlook seems to like it this way
		endPlusOneDay := e.Time.To.AddDate(0, 0, 1)
		calEvent.SetProperty(componentPropertyDtEnd, endPlusOneDay.Format(dateFormatUtc))
		calEvent.SetURL(infoUrl)
	}

	serialized := cal.Serialize()
	if err := os.MkdirAll(filepath.Dir(path), 0770); err != nil {
		return fmt.Errorf("serializing calender to %s: %w", path, err)
	}
	if err := os.WriteFile(path, []byte(serialized), 0o777); err != nil {
		return fmt.Errorf("serializing calender to %s: %w", path, err)
	}

	return nil
}
