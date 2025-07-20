package events

import (
	"crypto/sha256"
	"fmt"
	"html/template"
	"log"
	"strings"
	"time"

	"github.com/flopp/freiburg-run/internal/utils"
	"github.com/google/uuid"
)

type EventMeta struct {
	Current  bool
	BaseName utils.Name
	SeoTitle string
	Siblings []*Event
}

type Event struct {
	Type           string
	Name           utils.Name
	NameOld        utils.Name
	Time           utils.TimeRange
	Old            bool
	Status         string
	Cancelled      bool
	Obsolete       bool
	Special        bool
	Location       Location
	Details        template.HTML
	Details2       template.HTML
	Url            string
	RawTags        []string
	Tags           []*Tag
	RawSeries      []string
	Series         []*Serie
	Links          []*utils.Link
	Calendar       string
	CalendarGoogle string
	Added          string
	New            bool
	Prev           *Event
	Next           *Event
	UpcomingNear   []*Event
	Meta           EventMeta
}

func (event Event) GetUUID() (uuid.UUID, error) {
	if event.IsSeparator() {
		return uuid.UUID{}, fmt.Errorf("cannot create UUID for separator")
	}

	hash := sha256.New()
	slug := event.Slug()
	hash.Write([]byte(slug))
	hashId := hash.Sum(nil)
	uid, err := uuid.FromBytes(hashId[:16])
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("create UUID: %w", err)
	}

	return uid, nil
}

func (event Event) GenerateDescription() string {
	min := 110
	max := 160

	var description string

	location := ""
	if event.Location.NameNoFlag() != "" {
		location = fmt.Sprintf(" in '%s'", event.Location.NameNoFlag())
	}

	time := ""
	if event.Time.Original != "" {
		if event.Time.Original == "Verschiedene Termine" {
			time = ", verschiedene Termine"
		} else {
			time = fmt.Sprintf(" am %s", event.Time.Original)
		}
	}

	switch event.Type {
	case "event":
		description = fmt.Sprintf("Informationen zur Laufveranstaltung '%s' %s %s", event.Name.Orig, location, time)
	case "group":
		description = fmt.Sprintf("Informationen zur Laufgruppe / zum Lauftreff '%s' %s %s", event.Name.Orig, location, time)
	case "shop":
		description = fmt.Sprintf("Informationen zum Lauf-Shop '%s' %s", event.Name.Orig, location)
	}

	if len(description) >= min {
		return description
	}

	for i, tag := range event.Tags {
		if len(description) >= max {
			break
		}
		if i == 0 {
			description += "; "
		} else {
			description += ", "
		}
		description += tag.Name.Orig
	}

	return description
}

func (event Event) IsSeparator() bool {
	return event.Type == ""
}

func NonSeparators(events []*Event) int {
	count := 0
	for _, e := range events {
		if !e.IsSeparator() {
			count += 1
		}
	}
	return count
}

func createSeparatorEvent(t time.Time) *Event {
	label := fmt.Sprintf("%s %d", utils.MonthStr(t.Month()), t.Year())

	return &Event{
		"",
		utils.NewName(label),
		utils.NewName(""),
		utils.TimeRange{},
		false,
		"",
		false,
		false,
		false,
		Location{},
		"",
		"",
		"",
		nil,
		nil,
		nil,
		nil,
		nil,
		"",
		"",
		"",
		true,
		nil,
		nil,
		nil,
		EventMeta{},
	}
}

func (event *Event) slug(ext string) string {
	t := event.Type
	sanitized := event.Name.Sanitized

	if !event.Time.IsZero() {
		return fmt.Sprintf("%s/%d-%s.%s", t, event.Time.Year(), sanitized, ext)
	}

	return fmt.Sprintf("%s/%s.%s", t, sanitized, ext)
}

func (event *Event) SlugOld() string {
	if event.NameOld.Orig == "" {
		return ""
	}

	t := event.Type
	if strings.Contains(event.NameOld.Orig, "parkrun") {
		t = "event"
	}

	sanitized := event.NameOld.Sanitized
	if !event.Time.IsZero() {
		return fmt.Sprintf("%s/%d-%s.html", t, event.Time.Year(), sanitized)
	}
	return fmt.Sprintf("%s/%s.html", t, sanitized)
}

func (event *Event) SlugFile() string {
	if event.Type == "event" && event.Meta.BaseName.Sanitized != "" && event.Meta.Current {
		return fmt.Sprintf("%s/%s/index.html", event.Type, event.Meta.BaseName.Sanitized)
	}
	return event.slug("html")
}

func (event *Event) Slug() string {
	if event.Type == "event" && event.Meta.BaseName.Sanitized != "" && event.Meta.Current {
		return fmt.Sprintf("%s/%s/", event.Type, event.Meta.BaseName.Sanitized)
	}
	return event.slug("html")
}

func (event *Event) SlugNoBase() string {
	return event.slug("html")
}

func (event *Event) CalendarSlug() string {
	return event.slug("ics")
}

func (event *Event) LinkTitle() string {
	switch event.Type {
	case "event":
		if strings.HasPrefix(event.Url, "mailto:") {
			return "Mail an Veranstalter"
		}
		return "Zur Veranstaltung"
	case "group":
		if strings.HasPrefix(event.Url, "mailto:") {
			return "Mail an Organisator"
		}
		return "Zum Lauftreff"
	case "shop":
		return "Zum Lauf-Shop"
	default:
		return "Zur Veranstaltung"
	}
}

func (event *Event) NiceType() string {
	if event.Old {
		return "vergangene Veranstaltung"
	}
	switch event.Type {
	case "event":
		return "Veranstaltung"
	case "group":
		return "Lauftreff"
	case "shop":
		return "Lauf-Shop"
	default:
		return "Veranstaltung"
	}
}

func SplitEvents(eventList []*Event) ([]*Event, []*Event) {
	futureEvents := make([]*Event, 0)
	pastEvents := make([]*Event, 0)

	for _, event := range eventList {
		if event.Old {
			pastEvents = append(pastEvents, event)
		} else {
			futureEvents = append(futureEvents, event)
		}
	}
	return futureEvents, pastEvents
}

func SplitObsolete(eventList []*Event) ([]*Event, []*Event) {
	currentEvents := make([]*Event, 0)
	obsoleteEvents := make([]*Event, 0)

	for _, event := range eventList {
		if event.Obsolete {
			obsoleteEvents = append(obsoleteEvents, event)
		} else {
			currentEvents = append(currentEvents, event)
		}
	}
	return currentEvents, obsoleteEvents
}

func AddMonthSeparators(eventList []*Event) []*Event {
	result := make([]*Event, 0, len(eventList))
	var last time.Time

	for _, event := range eventList {
		d := event.Time.From
		if event.Time.From.IsZero() {
			// no label
		} else if last.IsZero() {
			// initial label
			last = d
			result = append(result, createSeparatorEvent(last))
		} else if d.After(last) {
			if last.Year() == d.Year() && last.Month() == d.Month() {
				// no new month label
			} else {
				for last.Year() != d.Year() || last.Month() != d.Month() {
					if last.Month() == time.December {
						last = time.Date(last.Year()+1, time.January, 1, 0, 0, 0, 0, last.Location())
					} else {
						last = time.Date(last.Year(), last.Month()+1, 1, 0, 0, 0, 0, last.Location())
					}
					result = append(result, createSeparatorEvent(last))
				}
			}
		}

		result = append(result, event)
	}
	return result
}

func AddMonthSeparatorsDescending(eventList []*Event) []*Event {
	result := make([]*Event, 0, len(eventList))
	var last time.Time

	for _, event := range eventList {
		d := event.Time.From
		if event.Time.From.IsZero() {
			// no label
		} else if last.IsZero() {
			// initial label
			last = d
			result = append(result, createSeparatorEvent(last))
		} else if d.Before(last) {
			if last.Year() == d.Year() && last.Month() == d.Month() {
				// no new month label
			} else {
				for last.Year() != d.Year() || last.Month() != d.Month() {
					if last.Month() == time.January {
						last = time.Date(last.Year()-1, time.December, 1, 0, 0, 0, 0, last.Location())
					} else {
						last = time.Date(last.Year(), last.Month()-1, 1, 0, 0, 0, 0, last.Location())
					}
					result = append(result, createSeparatorEvent(last))
				}
			}
		}

		result = append(result, event)
	}
	return result
}

func ChangeRegistrationLinks(events []*Event) {
	regSitesWithResults := []string{"raceresult.com", "sporkrono.fr", "racepedia.de"}
	for _, event := range events {
		for _, link := range event.Links {
			if link.IsRegistration() {
				for _, regSite := range regSitesWithResults {
					if strings.Contains(link.Url, regSite) {
						link.Name = "Anmeldung / Ergebnisse"
						break
					}
				}
			}
		}
	}
}

func Reverse(s []*Event) []*Event {
	a := make([]*Event, len(s))
	copy(a, s)

	for i := len(a)/2 - 1; i >= 0; i-- {
		opp := len(a) - 1 - i
		a[i], a[opp] = a[opp], a[i]
	}

	return a
}

func ValidateDateOrder(events []*Event) {
	var lastDate utils.TimeRange
	for _, event := range events {
		if !lastDate.IsZero() {
			if event.Time.From.IsZero() {
				log.Printf("event '%s' has no date", event.Name.Orig)
				return
			}
			if event.Time.From.Before(lastDate.From) {
				log.Printf("event '%s' has date '%s' before date of previous event '%s'", event.Name.Orig, event.Time.Formatted, lastDate.Formatted)
				return
			}
		}

		lastDate = event.Time
	}
}

func ValidateNameOrder(eventList []*Event) {
	var last *Event = nil

	for _, event := range eventList {
		if last == nil {
			last = event
			continue
		}

		if !(last.Name.Sanitized < event.Name.Sanitized) {
			log.Printf("bad order: %s ... %s\n", last.Name.Sanitized, event.Name.Sanitized)
		}

		last = event
	}
}

func FindPrevNextEvents(eventList []*Event) {
	for _, event := range eventList {
		var prev *Event = nil
		for _, event2 := range eventList {
			if event2 == event {
				break
			}

			if utils.IsSimilarName(event2.Name.Sanitized, event.Name.Sanitized) {
				prev = event2
			}
		}

		if prev != nil {
			prev.Next = event
			event.Prev = prev
		}
	}
}

func FindSiblings(eventList []*Event, today time.Time) {
	collected := make(map[*Event]struct{})

	for i, startEvent := range eventList {
		if _, found := collected[startEvent]; found {
			continue
		}
		if startEvent.Meta.BaseName.Sanitized == "" {
			continue
		}

		siblings := make([]*Event, 0)
		for j := i; j < len(eventList); j++ {
			event := eventList[j]
			if _, found := collected[event]; found {
				continue
			}
			if event.Meta.BaseName.Sanitized != startEvent.Meta.BaseName.Sanitized {
				continue
			}
			collected[event] = struct{}{}
			siblings = append(siblings, event)
		}
		for _, event := range siblings {
			event.Meta.Siblings = siblings
			event.Meta.Current = false
		}

		limitBefore := today.AddDate(0, 0, -7)
		var current *Event
		for i := len(siblings) - 1; i >= 0; i -= 1 {
			event := siblings[i]
			if current == nil {
				current = event
			} else if event.Time.Before(limitBefore) {
				break
			} else {
				current = event
			}
		}
		current.Meta.Current = true
	}
}

func FindUpcomingNearEvents(eventList []*Event, upcomingEvents []*Event, maxDistanceKM float64, count int) {
	for _, event := range eventList {
		if !event.Location.HasGeo() {
			continue
		}
		event.UpcomingNear = make([]*Event, 0, count)
		for _, candidate := range upcomingEvents {
			if candidate == event || candidate.Cancelled || !candidate.Location.HasGeo() {
				continue
			}
			if distanceKM, _ := utils.DistanceBearing(event.Location.Lat, event.Location.Lon, candidate.Location.Lat, candidate.Location.Lon); distanceKM > maxDistanceKM {
				continue
			}
			event.UpcomingNear = append(event.UpcomingNear, candidate)
			if len(event.UpcomingNear) >= count {
				break
			}
		}
	}
}
