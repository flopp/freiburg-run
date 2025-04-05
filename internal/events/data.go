package events

import (
	"fmt"
	"sort"
	"time"
)

type Data struct {
	Events         []*Event
	EventsOld      []*Event
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

func FetchData(config SheetsConfigData, today time.Time) (Data, error) {
	var data Data

	eventList, groupList, shopList, parkrunList, tagList, seriesList, err := LoadSheets(config, today)
	if err != nil {
		return data, err
	}

	ValidateDateOrder(eventList)

	data.Events, data.EventsObsolete = SplitObsolete(eventList)
	data.Groups, data.GroupsObsolete = SplitObsolete(groupList)
	data.Shops, data.ShopsObsolete = SplitObsolete(shopList)
	data.Tags = tagList
	data.Series = seriesList
	data.ParkrunEvents = parkrunList

	FindPrevNextEvents(data.Events)
	data.Events, data.EventsOld = SplitEvents(data.Events)
	data.Events = AddMonthSeparators(data.Events)
	FindUpcomingNearEvents(data.Events, data.Events, 5.0, 3)
	FindUpcomingNearEvents(data.EventsOld, data.Events, 5.0, 3)
	data.EventsOld = Reverse(data.EventsOld)
	data.EventsOld = AddMonthSeparatorsDescending(data.EventsOld)
	ChangeRegistrationLinks(data.EventsOld)
	data.collectTags()
	data.collectSeries()

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
			if event.Type == "event" {
				if event.Old {
					tag.EventsOld = append(tag.EventsOld, event)
				} else {
					tag.Events = append(tag.Events, event)
				}
			} else if event.Type == "group" {
				tag.Groups = append(tag.Groups, event)
			} else if event.Type == "shop" {
				tag.Shops = append(tag.Shops, event)
			} else {
				return fmt.Errorf("unexpected event.Type for '%s': %s", event.Name.Orig, event.Type)
			}
		}
	}
	return nil
}

func (data *Data) collectTags() {
	tags := make(map[string]*Tag)
	for _, tag := range data.Tags {
		tags[tag.Name.Sanitized] = tag
	}

	if err := collectEventTags(tags, data.Events); err != nil {
		panic(fmt.Errorf("collectEventTags for events: %w", err))
	}
	if err := collectEventTags(tags, data.EventsOld); err != nil {
		panic(fmt.Errorf("collectEventTags for eventsOld: %w", err))
	}
	if err := collectEventTags(tags, data.Groups); err != nil {
		panic(fmt.Errorf("collectEventTags for groups: %w", err))
	}
	if err := collectEventTags(tags, data.Shops); err != nil {
		panic(fmt.Errorf("collectEventTags for shops: %w", err))
	}

	tagsList := make([]*Tag, 0, len(tags))
	for _, tag := range tags {
		tag.Events = AddMonthSeparators(tag.Events)
		tag.EventsOld = AddMonthSeparatorsDescending(tag.EventsOld)
		tagsList = append(tagsList, tag)
	}
	sort.Slice(tagsList, func(i, j int) bool { return tagsList[i].Name.Sanitized < tagsList[j].Name.Sanitized })
	data.Tags = tagsList
}

func collectEventSeries(seriesMap map[string]*Serie, eventList []*Event) error {
	for _, event := range eventList {
		if event.Series != nil {
			return fmt.Errorf("expecting event.Series=nil for '%s'", event.Name.Orig)
		}

		event.Series = make([]*Serie, 0, len(event.RawSeries))
		for _, s := range event.RawSeries {
			serie := GetSerie(seriesMap, s)
			event.Series = append(event.Series, serie)
			if event.Type == "event" {
				if event.Old {
					serie.EventsOld = append(serie.EventsOld, event)
				} else {
					serie.Events = append(serie.Events, event)
				}
			} else if event.Type == "group" {
				serie.Groups = append(serie.Groups, event)
			} else if event.Type == "shop" {
				serie.Shops = append(serie.Shops, event)
			} else {
				return fmt.Errorf("unexpected event.Type for '%s': %s", event.Name.Orig, event.Type)
			}
		}
	}
	return nil
}

func (data *Data) collectSeries() error {
	seriesMap := make(map[string]*Serie)
	for _, series := range data.Series {
		seriesMap[series.Name.Sanitized] = series
	}

	if err := collectEventSeries(seriesMap, data.Events); err != nil {
		return err
	}
	if err := collectEventSeries(seriesMap, data.EventsOld); err != nil {
		return err
	}
	if err := collectEventSeries(seriesMap, data.Groups); err != nil {
		return err
	}
	if err := collectEventSeries(seriesMap, data.Shops); err != nil {
		return err
	}

	var seriesList, seriesListOld []*Serie
	for _, s := range data.Series {
		s.Events = AddMonthSeparators(s.Events)
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
