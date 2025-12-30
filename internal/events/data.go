package events

import (
	"fmt"
	"log"
	"sort"
	"time"

	"github.com/flopp/freiburg-run/internal/utils"
)

type OldEvents struct {
	Year   string
	Events []*Event
}

type Data struct {
	Events         []*Event
	EventsOld      []*Event
	OldEvents      []OldEvents // Old events grouped by year
	EventsObsolete []*Event
	Groups         []*Event
	GroupsObsolete []*Event
	Shops          []*Event
	ShopsObsolete  []*Event
	Tags           []*Tag
	Series         []*Serie
	SeriesOld      []*Serie
	ParkrunEvents  []*ParkrunEvent
}

type CheckUrl struct {
	Url   string
	Event *Event
	Name  string
}

func (data *Data) CheckLinks() {
	// collect urls to check from current events
	urls := make([]CheckUrl, 0)
	for _, event := range data.Events {
		if event.IsSeparator() {
			continue
		}
		urls = append(urls, CheckUrl{Url: event.MainLink.Url, Event: event, Name: "main"})
		for _, link := range event.Links {
			if link.IsExternal() {
				urls = append(urls, CheckUrl{Url: link.Url, Event: event, Name: "link"})
			}
		}
	}

	// do actual checks
	// Group URLs by domain to implement per-domain rate limiting
	domainMap := make(map[string][]CheckUrl)
	for _, url := range urls {
		domain := utils.ExtractDomain(url.Url)
		domainMap[domain] = append(domainMap[domain], url)
	}

	lc := utils.NewLinkChecker()
	results := make(chan struct {
		url CheckUrl
		err error
	}, len(urls))

	// Limit concurrent requests per domain
	const perDomainLimit = 2
	for domain, urlList := range domainMap {
		fmt.Printf("Checking %d links for domain %s\n", len(urlList), domain)
		sem := make(chan struct{}, perDomainLimit)
		for _, u := range urlList {
			sem <- struct{}{}
			go func(u CheckUrl, sem chan struct{}) {
				defer func() { <-sem }()
				err := lc.Check(u.Url)
				results <- struct {
					url CheckUrl
					err error
				}{u, err}
			}(u, sem)
		}
	}

	for i := 0; i < len(urls); i++ {
		res := <-results
		if res.err != nil {
			fmt.Printf("Invalid %s link in event '%s': %s -> %v\n", res.url.Name, res.url.Event.Name.Orig, res.url.Url, res.err)
		}
	}
}

func FetchData(config utils.Config, today time.Time) (Data, error) {
	var data Data

	sheetsData, err := LoadSheets(config, today)
	if err != nil {
		return data, err
	}

	ValidateDateOrder(sheetsData.Events)
	ValidateNameOrder(sheetsData.Groups)
	ValidateNameOrder(sheetsData.Shops)

	data.Events, data.EventsObsolete = SplitObsolete(sheetsData.Events)
	data.Groups, data.GroupsObsolete = SplitObsolete(sheetsData.Groups)
	data.Shops, data.ShopsObsolete = SplitObsolete(sheetsData.Shops)
	data.Tags = sheetsData.Tags
	data.Series = sheetsData.Series
	data.ParkrunEvents = sheetsData.Parkrun

	FindPrevNextEvents(data.Events)
	FindSiblings(data.Events, today)
	data.Events, data.EventsOld = SplitEvents(data.Events)
	data.Events = AddMonthSeparators(data.Events, today)
	FindUpcomingNearEvents(data.Events, data.Events, 5.0, 3)
	FindUpcomingNearEvents(data.EventsOld, data.Events, 5.0, 3)
	data.EventsOld = Reverse(data.EventsOld)
	data.EventsOld = AddMonthSeparatorsDescending(data.EventsOld)
	ChangeRegistrationLinks(data.EventsOld)
	data.collectTags(today)
	data.collectSeries(today)

	// Collect old events by year
	maxYear := 0
	minYear := 0
	oldEventsMap := make(map[int][]*Event)
	for _, e := range data.EventsOld {
		if e.IsSeparator() {
			continue
		}
		year := e.Time.From.Year()
		oldEvents, ok := oldEventsMap[year]
		if !ok {
			oldEvents = make([]*Event, 0)
			oldEvents = append(oldEvents, e)
			oldEventsMap[year] = oldEvents
			if maxYear == 0 || year > maxYear {
				maxYear = year
			}
			if minYear == 0 || year < minYear {
				minYear = year
			}
		} else {
			oldEvents = append(oldEvents, e)
			oldEventsMap[year] = oldEvents
		}
	}
	data.OldEvents = make([]OldEvents, 0, maxYear-minYear+1)
	for year := maxYear; year >= minYear; year-- {
		if oldEvents, ok := oldEventsMap[year]; ok {
			oldEvents = AddMonthSeparatorsDescending(oldEvents)
			data.OldEvents = append(data.OldEvents, OldEvents{Year: fmt.Sprintf("%d", year), Events: oldEvents})
		}
	}
	return data, nil
}

func collectEventTags(tags map[string]*Tag, eventList []*Event) error {
	for _, event := range eventList {
		if event.Tags != nil {
			return fmt.Errorf("expecting event.Tags=nil for '%s'", event.Name.Orig)
		}

		event.Tags = make([]*Tag, 0, len(event.RawTags))
		for _, t := range event.RawTags {
			tag := GetTag(tags, t)
			event.Tags = append(event.Tags, tag)
			switch event.Type {
			case "event":
				if event.Old {
					tag.EventsOld = append(tag.EventsOld, event)
				} else {
					tag.Events = append(tag.Events, event)
				}
			case "group":
				tag.Groups = append(tag.Groups, event)
			case "shop":
				tag.Shops = append(tag.Shops, event)
			default:
				return fmt.Errorf("unexpected event.Type for '%s': %s", event.Name.Orig, event.Type)
			}
		}
	}
	return nil
}

func (data *Data) collectTags(today time.Time) error {
	tags := make(map[string]*Tag)
	for _, tag := range data.Tags {
		tags[tag.Name.Sanitized] = tag
	}

	lists := []struct {
		name string
		list []*Event
	}{
		{"Events", data.Events},
		{"EventsOld", data.EventsOld},
		{"Groups", data.Groups},
		{"Shops", data.Shops},
	}
	for _, l := range lists {
		if err := collectEventTags(tags, l.list); err != nil {
			return fmt.Errorf("collectEventTags for %s: %w", l.name, err)
		}
	}

	tagsList := make([]*Tag, 0, len(tags))
	for _, tag := range tags {
		tag.Events = AddMonthSeparators(tag.Events, today)
		tag.EventsOld = AddMonthSeparatorsDescending(tag.EventsOld)
		tagsList = append(tagsList, tag)
	}
	sort.Slice(tagsList, func(i, j int) bool { return tagsList[i].Name.Sanitized < tagsList[j].Name.Sanitized })
	data.Tags = tagsList
	return nil
}

func collectEventSeries(seriesMap map[string]*Serie, eventList []*Event) error {
	for _, event := range eventList {
		if event.Series != nil {
			return fmt.Errorf("expecting event.Series=nil for '%s'", event.Name.Orig)
		}

		event.Series = make([]*Serie, 0, len(event.RawSeries))
		for _, s := range event.RawSeries {
			serie, already_existed := GetSerie(seriesMap, s)
			if !already_existed {
				log.Printf("Event: %s has unknown series tag: %s", event.Name.Orig, s)
			}
			event.Series = append(event.Series, serie)
			switch event.Type {
			case "event":
				if event.Old {
					serie.EventsOld = append(serie.EventsOld, event)
				} else {
					serie.Events = append(serie.Events, event)
				}
			case "group":
				serie.Groups = append(serie.Groups, event)
			case "shop":
				serie.Shops = append(serie.Shops, event)
			default:
				return fmt.Errorf("unexpected event.Type for '%s': %s", event.Name.Orig, event.Type)
			}
		}
	}
	return nil
}

func (data *Data) collectSeries(today time.Time) error {
	seriesMap := make(map[string]*Serie)
	for _, series := range data.Series {
		seriesMap[series.Name.Sanitized] = series
	}

	lists := []struct {
		name string
		list []*Event
	}{
		{"Events", data.Events},
		{"EventsOld", data.EventsOld},
		{"Groups", data.Groups},
		{"Shops", data.Shops},
	}
	for _, l := range lists {
		if err := collectEventSeries(seriesMap, l.list); err != nil {
			return fmt.Errorf("collectEventSeries for %s: %w", l.name, err)
		}
	}

	var seriesList, seriesListOld []*Serie
	for _, s := range data.Series {
		s.Events = AddMonthSeparators(s.Events, today)
		s.EventsOld = AddMonthSeparatorsDescending(s.EventsOld)

		if s.IsOld() {
			seriesListOld = append(seriesListOld, s)
		} else {
			seriesList = append(seriesList, s)
		}
	}

	sortSeries := func(sl []*Serie) {
		sort.Slice(sl, func(i, j int) bool {
			return sl[i].Name.Sanitized < sl[j].Name.Sanitized
		})
	}
	sortSeries(seriesList)
	sortSeries(seriesListOld)

	data.Series = seriesList
	data.SeriesOld = seriesListOld

	return nil
}
