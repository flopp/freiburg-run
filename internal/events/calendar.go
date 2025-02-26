package events

import (
	"fmt"
	"os"
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

func CreateEventCalendar(event *Event, now time.Time, calendarUrl string, path string) error {
	eventsList := make([]*Event, 1)
	eventsList[0] = event
	return CreateCalendar(eventsList, now, calendarUrl, path)
}

func CreateCalendar(eventsList []*Event, now time.Time, calendarUrl string, path string) error {
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

		url := fmt.Sprintf("https://freiburg.run/%s", e.Slug())

		calEvent := cal.AddEvent(uid.String())
		calEvent.SetDtStampTime(now)
		calEvent.SetSummary(e.Name)
		calEvent.SetLocation(e.Location.NameNoFlag())
		calEvent.SetDescription(e.Details)
		calEvent.SetProperty(componentPropertyDtStart, e.Time.From.Format(dateFormatUtc))
		// end + 1 day; Outlook seems to like it this way
		endPlusOneDay := e.Time.To.AddDate(0, 0, 1)
		calEvent.SetProperty(componentPropertyDtEnd, endPlusOneDay.Format(dateFormatUtc))

		calEvent.SetURL(url)
	}

	serialized := cal.Serialize()
	if err := os.WriteFile(path, []byte(serialized), 0o777); err != nil {
		return fmt.Errorf("serializing calender to %s: %w", path, err)
	}

	return nil
}
