package events

import (
	"fmt"
	"os"

	ical "github.com/arran4/golang-ical"
)

const (
	dateFormatUtc = "20060102"

	propertyDtStart ical.Property = "DTSTART;VALUE=DATE"
	propertyDtEnd   ical.Property = "DTEND;VALUE=DATE"

	componentPropertyDtStart = ical.ComponentProperty(propertyDtStart)
	componentPropertyDtEnd   = ical.ComponentProperty(propertyDtEnd)
)

func CreateCalendar(data Data, path string) error {
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

		id, err := e.GetUUID()
		if err != nil {
			return fmt.Errorf("creating calendar entry for '%s': %w", e.Name, err)
		}

		calEvent := cal.AddEvent(id.String())
		calEvent.SetLocation(e.Location.NameNoFlag())
		calEvent.SetProperty(componentPropertyDtStart, e.Time.From.UTC().Format(dateFormatUtc))
		calEvent.SetProperty(componentPropertyDtEnd, e.Time.To.UTC().Format(dateFormatUtc))
		calEvent.SetSummary(e.Name)
		calEvent.SetURL(fmt.Sprintf("https://freiburg.run/%s", e.Slug()))
		calEvent.SetDescription(e.Details)
	}

	serialized := cal.Serialize()
	if err := os.WriteFile(path, []byte(serialized), 0o777); err != nil {
		return fmt.Errorf("serializing calender to %s: %w", path, err)
	}

	return nil
}
