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

func CreateCalendar(data Data, now time.Time, path string) error {
	_ = data

	cal := ical.NewCalendar()
	cal.SetProductId("Laufevents - freiburg.run")
	cal.SetMethod(ical.MethodPublish)
	cal.SetDescription("Liste aller Laufevents im Raum Freiburg (50km Umkreis)")
	cal.SetUrl("https://freiburg.run/events.ics")

	for _, e := range data.Events {
		if e.IsSeparator() {
			continue
		}

		url := fmt.Sprintf("https://freiburg.run/%s", e.Slug())

		calEvent := cal.AddEvent(url)
		calEvent.SetDtStampTime(now)
		calEvent.SetSummary(e.Name)
		calEvent.SetLocation(e.Location.NameNoFlag())
		calEvent.SetDescription(e.Details)
		calEvent.SetProperty(componentPropertyDtStart, e.Time.From.UTC().Format(dateFormatUtc))
		calEvent.SetProperty(componentPropertyDtEnd, e.Time.To.UTC().Format(dateFormatUtc))
		calEvent.SetURL(url)
	}

	serialized := cal.Serialize()
	if err := os.WriteFile(path, []byte(serialized), 0o777); err != nil {
		return fmt.Errorf("serializing calender to %s: %w", path, err)
	}

	return nil
}
