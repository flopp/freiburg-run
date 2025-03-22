package events

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"time"

	ical "github.com/arran4/golang-ical"
)

const (
	dateFormatUtc = "20060102"

	propertyDtStart ical.Property = "DTSTART;VALUE=DATE"
	propertyDtEnd   ical.Property = "DTEND;VALUE=DATE"

	componentPropertyDtStart = ical.ComponentProperty(propertyDtStart)
	componentPropertyDtEnd   = ical.ComponentProperty(propertyDtEnd)
)

func CreateEventCalendar(event *Event, now time.Time, baseUrl string, calendarUrl string, path string) error {
	eventsList := make([]*Event, 1)
	eventsList[0] = event
	return CreateCalendar(eventsList, now, baseUrl, calendarUrl, path)
}

func CreateCalendar(eventsList []*Event, now time.Time, baseUrl string, calendarUrl string, path string) error {
	cal := ical.NewCalendar()
	cal.SetProductId("Laufevents - freiburg.run")
	cal.SetMethod(ical.MethodPublish)
	cal.SetDescription("Liste aller Laufevents im Raum Freiburg (50km Umkreis)")
	cal.SetUrl(calendarUrl)

	for _, e := range eventsList {
		if e.IsSeparator() {
			continue
		}

		uid, err := e.GetUUID()
		if err != nil {
			return fmt.Errorf("create UUID for '%s': %w", e.Name, err)
		}

		infoUrl := fmt.Sprintf("%s/%s", baseUrl, e.Slug())

		calEvent := cal.AddEvent(uid.String())
		calEvent.SetDtStampTime(now)
		calEvent.SetSummary(e.Name)
		calEvent.SetLocation(e.Location.NameNoFlag())
		calEvent.SetDescription(e.Details)
		calEvent.SetProperty(componentPropertyDtStart, e.Time.From.Format(dateFormatUtc))
		// end + 1 day; Outlook seems to like it this way
		endPlusOneDay := e.Time.To.AddDate(0, 0, 1)
		calEvent.SetProperty(componentPropertyDtEnd, endPlusOneDay.Format(dateFormatUtc))
		calEvent.SetURL(infoUrl)

		// Google Calendar link
		e.CalendarGoogle = fmt.Sprintf("https://calendar.google.com/calendar/u/0/r/eventedit?text=%s&dates=%s/%s&details=%s&location=%s",
			url.QueryEscape(e.Name),
			e.Time.From.Format(dateFormatUtc),
			endPlusOneDay.Format(dateFormatUtc),
			url.QueryEscape(fmt.Sprintf(`%s<br>Infos: <a href="%s">freiburg.run</a>`, e.Details, infoUrl)),
			url.QueryEscape(e.Location.NameNoFlag()),
		)
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
