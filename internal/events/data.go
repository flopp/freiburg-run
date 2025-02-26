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

func collectEventTags(tags map[string]*Tag, event *Event) {
	if event.Tags != nil {
		panic("expecting event.Tags=nil")
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
			panic(fmt.Errorf("unexpected event.Type: %s", event.Type))
		}
	}
}

func (data *Data) collectTags() {
	tags := make(map[string]*Tag)
	for _, tag := range data.Tags {
		tags[tag.Sanitized] = tag
	}

	for _, e := range data.Events {
		collectEventTags(tags, e)
	}
	for _, e := range data.EventsOld {
		collectEventTags(tags, e)
	}
	for _, e := range data.Groups {
		collectEventTags(tags, e)
	}
	for _, e := range data.Shops {
		collectEventTags(tags, e)
	}

	tagsList := make([]*Tag, 0, len(tags))
	for _, tag := range tags {
		tag.Events = AddMonthSeparators(tag.Events)
		tag.EventsOld = AddMonthSeparatorsDescending(tag.EventsOld)
		tagsList = append(tagsList, tag)
	}
	sort.Slice(tagsList, func(i, j int) bool { return tagsList[i].Sanitized < tagsList[j].Sanitized })
	data.Tags = tagsList
}

func collectEventSeries(seriesMap map[string]*Serie, event *Event) {
	if event.Series != nil {
		panic("expecting event.Series=nil")
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
			panic(fmt.Errorf("unexpected event.Type: %s", event.Type))
		}
	}
}

func (data *Data) collectSeries() {
	seriesMap := make(map[string]*Serie)
	for _, series := range data.Series {
		seriesMap[series.Sanitized] = series
	}

	for _, e := range data.Events {
		collectEventSeries(seriesMap, e)
	}
	for _, e := range data.EventsOld {
		collectEventSeries(seriesMap, e)
	}
	for _, e := range data.Groups {
		collectEventSeries(seriesMap, e)
	}
	for _, e := range data.Shops {
		collectEventSeries(seriesMap, e)
	}

	seriesList := make([]*Serie, 0, len(data.Series))
	seriesListOld := make([]*Serie, 0, len(data.Series))
	for _, s := range data.Series {
		if s.IsOld() {
			seriesListOld = append(seriesListOld, s)
		} else {
			seriesList = append(seriesList, s)
		}
		s.Events = AddMonthSeparators(s.Events)
		s.EventsOld = AddMonthSeparatorsDescending(s.EventsOld)
	}
	sort.Slice(seriesList, func(i, j int) bool { return seriesList[i].Sanitized < seriesList[j].Sanitized })
	sort.Slice(seriesListOld, func(i, j int) bool { return seriesListOld[i].Sanitized < seriesListOld[j].Sanitized })

	data.Series = seriesList
	data.SeriesOld = seriesListOld
}
