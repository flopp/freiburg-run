package events

import (
	"context"
	"fmt"
	"html/template"
	"log"
	"sort"
	"strings"
	"time"

	"github.com/flopp/freiburg-run/internal/utils"
	"github.com/flopp/go-googlesheetswrapper"
)

type SheetsData struct {
	Events  []*Event
	Groups  []*Event
	Shops   []*Event
	Parkrun []*ParkrunEvent
	Tags    []*Tag
	Series  []*Serie
}

func LoadSheets(config utils.Config, today time.Time, client googlesheetswrapper.Client) (SheetsData, error) {
	ctx := context.Background()
	sheets, err := client.ReadAll(ctx)
	if err != nil {
		return SheetsData{}, fmt.Errorf("fetching all sheets: %w", err)
	}

	eventSheets, groupsSheet, shopsSheet, parkrunSheet, tagsSheet, seriesSheet, err := findSheetNames(config, sheets)
	if err != nil {
		return SheetsData{}, err
	}

	events, err := loadEvents(config, today, eventSheets, sheets)
	if err != nil {
		return SheetsData{}, fmt.Errorf("fetching events: %w", err)
	}
	groups, err := fetchEvents(config, today, "group", groupsSheet, sheets)
	if err != nil {
		return SheetsData{}, fmt.Errorf("fetching groups: %w", err)
	}
	shops, err := fetchEvents(config, today, "shop", shopsSheet, sheets)
	if err != nil {
		return SheetsData{}, fmt.Errorf("fetching shops: %w", err)
	}
	var parkrun []*ParkrunEvent
	if config.Pages.Parkrun {
		parkrun, err = fetchParkrunEvents(config, today, parkrunSheet, sheets)
		if err != nil {
			return SheetsData{}, fmt.Errorf("fetching parkrun events: %w", err)
		}
	}
	tags, err := fetchTags(config, tagsSheet, sheets)
	if err != nil {
		return SheetsData{}, fmt.Errorf("fetching tags: %w", err)
	}
	series, err := fetchSeries(config, seriesSheet, sheets)
	if err != nil {
		return SheetsData{}, fmt.Errorf("fetching series: %w", err)
	}

	return SheetsData{
		Events:  events,
		Groups:  groups,
		Shops:   shops,
		Parkrun: parkrun,
		Tags:    tags,
		Series:  series,
	}, nil
}

func findSheetNames(config utils.Config, sheets map[string][][]string) (eventSheets []string, groupsSheet, shopsSheet, parkrunSheet, tagsSheet, seriesSheet string, err error) {
	for sheetName := range sheets {
		switch {
		case strings.HasPrefix(sheetName, "Events"):
			eventSheets = append(eventSheets, sheetName)
		case sheetName == "Groups":
			groupsSheet = sheetName
		case sheetName == "Shops":
			shopsSheet = sheetName
		case sheetName == "Parkrun":
			parkrunSheet = sheetName
		case sheetName == "Tags":
			tagsSheet = sheetName
		case sheetName == "Series":
			seriesSheet = sheetName
		case strings.Contains(sheetName, "ignore"):
			// ignore
		default:
			log.Printf("ignoring unknown sheet: '%s'", sheetName)
		}
	}

	if len(eventSheets) < 2 {
		return nil, "", "", "", "", "", fmt.Errorf("fetching sheets: unable to find enough 'Events' sheets")
	}
	// sort eventSheets by name, so that they are always in the same order (e.g. for testing)
	sort.Slice(eventSheets, func(i, j int) bool { return eventSheets[i] < eventSheets[j] })
	// validate eventSheets names ("EventsYYYY" format, YYYY=year), order and consecutiveness (no missing years)
	lastYear := -1
	for _, sheetName := range eventSheets {
		if !strings.HasPrefix(sheetName, "Events") {
			return nil, "", "", "", "", "", fmt.Errorf("fetching sheets: unexpected event sheet name '%s' (no 'Events' prefix)", sheetName)
		}
		if len(sheetName) != 10 {
			return nil, "", "", "", "", "", fmt.Errorf("fetching sheets: unexpected event sheet name '%s' (bad length %d, expected 10)", sheetName, len(sheetName))
		}
		yearStr := sheetName[6:]
		if date, err := time.Parse("2006", yearStr); err != nil {
			return nil, "", "", "", "", "", fmt.Errorf("fetching sheets: unexpected event sheet name '%s' (yearStr='%s'): %v", sheetName, yearStr, err)
		} else {
			if lastYear != -1 && date.Year() != lastYear+1 {
				return nil, "", "", "", "", "", fmt.Errorf("fetching sheets: unexpected event sheet name '%s': missing year %d", sheetName, lastYear+1)
			}
			lastYear = date.Year()
		}
	}

	if groupsSheet == "" {
		return nil, "", "", "", "", "", fmt.Errorf("fetching sheets: unable to find 'Groups' sheet")
	}
	if shopsSheet == "" {
		return nil, "", "", "", "", "", fmt.Errorf("fetching sheets: unable to find 'Shops' sheet")
	}
	if config.Pages.Parkrun && parkrunSheet == "" {
		return nil, "", "", "", "", "", fmt.Errorf("fetching sheets: unable to find 'Parkrun' sheet")
	}
	if tagsSheet == "" {
		return nil, "", "", "", "", "", fmt.Errorf("fetching sheets: unable to find 'Tags' sheet")
	}
	if seriesSheet == "" {
		return nil, "", "", "", "", "", fmt.Errorf("fetching sheets: unable to find 'Series' sheet")
	}
	return eventSheets, groupsSheet, shopsSheet, parkrunSheet, tagsSheet, seriesSheet, nil
}

func loadEvents(config utils.Config, today time.Time, eventSheets []string, sheets map[string][][]string) ([]*Event, error) {
	eventList := make([]*Event, 0)
	for _, sheet := range eventSheets {
		yearList, err := fetchEvents(config, today, "event", sheet, sheets)
		if err != nil {
			return nil, err
		}
		eventList = append(eventList, yearList...)
	}
	return eventList, nil
}

func getVal(cols map[string]int, col string, row []string) (string, error) {
	colIndex, ok := cols[col]
	if !ok {
		return "", fmt.Errorf("missing column '%s'", col)
	}
	if colIndex >= len(row) {
		return "", nil
	}
	return row[colIndex], nil
}

func getLinks(cols map[string]int, row []string) []string {
	links := make([]string, 0)

	for i := 1; true; i += 1 {
		link, err := getVal(cols, fmt.Sprintf("LINK%d", i), row)
		if err != nil {
			break
		}
		links = append(links, link)
	}

	return links
}

type EventData struct {
	Date         string
	Added        string
	Name         string
	Name2        string
	Status       string
	Url          string
	Description  string
	Location     string
	Coordinates  string
	Registration string
	Tags         string
	Links        []string
}

func getEventData(cols map[string]int, row []string) (EventData, error) {
	var data EventData
	var err error
	fields := []struct {
		name string
		dest *string
	}{
		{"DATE", &data.Date},
		{"ADDED", &data.Added},
		{"NAME", &data.Name},
		{"NAME2", &data.Name2},
		{"STATUS", &data.Status},
		{"URL", &data.Url},
		{"DESCRIPTION", &data.Description},
		{"LOCATION", &data.Location},
		{"COORDINATES", &data.Coordinates},
		{"REGISTRATION", &data.Registration},
		{"TAGS", &data.Tags},
	}
	for _, f := range fields {
		*f.dest, err = getVal(cols, f.name, row)
		if err != nil {
			return EventData{}, err
		}
	}
	data.Links = getLinks(cols, row)
	return data, nil
}

func fetchEvents(config utils.Config, today time.Time, eventType string, tableName string, tables map[string][][]string) ([]*Event, error) {
	table, ok := tables[tableName]
	if !ok {
		return nil, fmt.Errorf("table '%s' not found", tableName)
	}

	requiredHeaders := []string{"DATE", "ADDED", "NAME", "NAME2", "STATUS", "URL", "DESCRIPTION", "LOCATION", "COORDINATES", "REGISTRATION", "TAGS"}
	cols, err := googlesheetswrapper.ExtractHeader(table, requiredHeaders, true)
	if err != nil {
		return nil, fmt.Errorf("fetching table '%s': %v", tableName, err)
	}

	eventsList := make([]*Event, 0)
	for line, row := range table[1:] {
		data, err := getEventData(cols, row)
		if err != nil {
			return nil, fmt.Errorf("table '%s', line '%d': %v", tableName, line+1, err)
		}
		cancelled := strings.Contains(data.Status, "abgesagt") || strings.Contains(data.Status, "geschlossen")
		if cancelled && (data.Status == "abgesagt" || data.Status == "geschlossen") {
			// clear simple cancelled/closed status (other info might be present)
			data.Status = ""
		}
		obsolete := data.Status == "obsolete"
		if obsolete {
			data.Status = ""
		}
		if data.Status == "temp" {
			log.Printf("table '%s', line '%d': skipping row with temp status", table, line)
			continue
		}
		if eventType == "event" {
			if data.Date == "" {
				log.Printf("table '%s', line '%d': skipping row with empty date", table, line)
				continue
			}
		}
		if data.Name == "" {
			log.Printf("table '%s', line '%d': skipping row with empty name", table, line)
			continue
		}
		if !strings.Contains(data.Name, data.Name2) {
			log.Printf("table '%s', line '%d': name '%s' does not contain name2 '%s'", table, line, data.Name, data.Name2)
		}
		if data.Url == "" {
			log.Printf("table '%s', line '%d': skipping row with empty url", table, line)
			continue
		}

		name, nameOld := utils.SplitPair(data.Name)
		url := data.Url
		description1, description2 := utils.SplitPair(data.Description)
		tags := make([]string, 0)
		series := make([]string, 0)
		for _, t := range utils.SplitList(data.Tags) {
			if strings.HasPrefix(t, "serie") {
				series = append(series, t[6:])
			} else {
				tags = append(tags, utils.SanitizeName(t))
			}
		}
		location := CreateLocation(config, data.Location, data.Coordinates)
		tags = append(tags, location.Tags()...)
		timeRange, err := utils.CreateTimeRange(data.Date)
		if err != nil {
			log.Printf("event '%s': %v", name, err)
		}
		isOld := timeRange.Before(today)
		links, err := parseLinks(data.Links, data.Registration)
		if err != nil {
			return nil, fmt.Errorf("parsing links of event '%s': %w", name, err)
		}

		eventsList = append(eventsList, &Event{
			eventType,
			utils.NewName(name),
			utils.NewName(nameOld),
			timeRange,
			isOld,
			data.Added,
			data.Status,
			cancelled,
			obsolete,
			location,
			template.HTML(description1),
			template.HTML(description2),
			utils.CreateUnnamedLink(url),
			utils.SortAndUniquify(tags),
			nil,
			nil,
			series,
			nil,
			links,
			"",
			"",
			"",
			EventMeta{
				false,
				utils.NewName(data.Name2),
				nil,
				nil,
				nil,
				nil,
			},
		})
	}

	return eventsList, nil
}

type ParkrunEventData struct {
	Index   string
	Date    string
	Runners string
	Temp    string
	Special string
	Cafe    string
	Results string
	Report  string
	Author  string
	Photos  string
}

func getParkrunEventData(cols map[string]int, row []string) (ParkrunEventData, error) {
	var data ParkrunEventData
	var err error
	fields := []struct {
		name string
		dest *string
	}{
		{"DATE", &data.Date},
		{"INDEX", &data.Index},
		{"RUNNERS", &data.Runners},
		{"TEMP", &data.Temp},
		{"SPECIAL", &data.Special},
		{"CAFE", &data.Cafe},
		{"RESULTS", &data.Results},
		{"REPORT", &data.Report},
		{"AUTHOR", &data.Author},
		{"PHOTOS", &data.Photos},
	}
	for _, f := range fields {
		*f.dest, err = getVal(cols, f.name, row)
		if err != nil {
			return ParkrunEventData{}, err
		}
	}
	return data, nil
}

func fetchParkrunEvents(config utils.Config, today time.Time, tableName string, tables map[string][][]string) ([]*ParkrunEvent, error) {
	table, ok := tables[tableName]
	if !ok {
		return nil, fmt.Errorf("table '%s' not found", tableName)
	}
	requiredHeaders := []string{"DATE", "INDEX", "RUNNERS", "TEMP", "SPECIAL", "CAFE", "RESULTS", "REPORT", "AUTHOR", "PHOTOS"}
	cols, err := googlesheetswrapper.ExtractHeader(table, requiredHeaders, true)
	if err != nil {
		return nil, fmt.Errorf("fetching table '%s': %v", tableName, err)
	}
	rows := table[1:]

	eventsList := make([]*ParkrunEvent, 0)
	for _, row := range rows {
		data, err := getParkrunEventData(cols, row)
		if err != nil {
			return nil, fmt.Errorf("table '%s': %v", table, err)
		}

		if data.Temp != "" {
			data.Temp = fmt.Sprintf("%s°C", data.Temp)
		}

		if data.Results != "" {
			data.Results = fmt.Sprintf("https://www.parkrun.com.de/dietenbach/results/%s", data.Results)
		}

		// determine is this is for the current week (but only for "real" parkrun events with index)
		currentWeek := false
		if data.Index != "" {
			d, err := utils.ParseDate(data.Date)
			if err == nil {
				today_y, today_m, today_d := today.Date()
				d_y, d_m, d_d := d.Date()
				currentWeek = (today_y == d_y && today_m == d_m && today_d == d_d) || (today.After(d) && today.Before(d.AddDate(0, 0, 7)))
			}
		}

		eventsList = append(eventsList, &ParkrunEvent{
			currentWeek,
			data.Index,
			data.Date,
			data.Runners,
			data.Temp,
			data.Special,
			data.Cafe,
			data.Results,
			data.Report,
			data.Author,
			data.Photos,
		})
	}

	return eventsList, nil
}

type TagData struct {
	Tag         string
	Name        string
	Description string
}

func getTagData(cols map[string]int, row []string) (TagData, error) {
	var data TagData
	var err error
	fields := []struct {
		name string
		dest *string
	}{
		{"TAG", &data.Tag},
		{"NAME", &data.Name},
		{"DESCRIPTION", &data.Description},
	}
	for _, f := range fields {
		*f.dest, err = getVal(cols, f.name, row)
		if err != nil {
			return TagData{}, err
		}
	}
	return data, nil
}

func fetchTags(config utils.Config, tableName string, tables map[string][][]string) ([]*Tag, error) {
	table, ok := tables[tableName]
	if !ok {
		return nil, fmt.Errorf("table '%s' not found", tableName)
	}

	requiredHeaders := []string{"TAG", "NAME", "DESCRIPTION"}
	cols, err := googlesheetswrapper.ExtractHeader(table, requiredHeaders, true)
	if err != nil {
		return nil, fmt.Errorf("fetching table '%s': %v", tableName, err)
	}
	rows := table[1:]

	tags := make([]*Tag, 0)
	for _, row := range rows {
		data, err := getTagData(cols, row)
		if err != nil {
			return nil, fmt.Errorf("table '%s': %v", tableName, err)
		}

		tag := utils.SanitizeName(data.Tag)
		if tag != "" && (data.Name != "" || data.Description != "") {
			t := CreateTag(tag)
			t.Name.Orig = data.Name
			t.Description = template.HTML(data.Description)
			tags = append(tags, t)
		}
	}

	return tags, nil
}

type SerieData struct {
	Tag         string
	Name        string
	Description string
	Links       []string
}

func getSerieData(cols map[string]int, row []string) (SerieData, error) {
	var data SerieData
	var err error
	fields := []struct {
		name string
		dest *string
	}{
		{"NAME", &data.Name},
		{"DESCRIPTION", &data.Description},
	}
	for _, f := range fields {
		*f.dest, err = getVal(cols, f.name, row)
		if err != nil {
			return SerieData{}, err
		}
	}
	data.Links = getLinks(cols, row)
	return data, nil
}

func fetchSeries(config utils.Config, tableName string, tables map[string][][]string) ([]*Serie, error) {
	table, ok := tables[tableName]
	if !ok {
		return nil, fmt.Errorf("table '%s' not found", tableName)
	}
	requiredHeaders := []string{"NAME", "DESCRIPTION"}
	cols, err := googlesheetswrapper.ExtractHeader(table, requiredHeaders, true)
	if err != nil {
		return nil, fmt.Errorf("fetching table '%s': %v", tableName, err)
	}
	rows := table[1:]

	series := make([]*Serie, 0)
	for _, row := range rows {
		data, err := getSerieData(cols, row)
		if err != nil {
			return nil, fmt.Errorf("table '%s': %v", table, err)
		}
		links, err := parseLinks(data.Links, "")
		if err != nil {
			return nil, fmt.Errorf("parsing links of series '%s': %w", data.Name, err)
		}
		series = append(series, &Serie{utils.NewName(data.Name), template.HTML(data.Description), links, make([]*Event, 0), make([]*Event, 0), make([]*Event, 0), make([]*Event, 0)})
	}

	return series, nil
}

func parseLinks(ss []string, registration string) ([]*utils.Link, error) {
	links := make([]*utils.Link, 0, len(ss))
	hasRegistration := registration != ""
	if hasRegistration {
		links = append(links, utils.CreateLink("Anmeldung", registration))
	}
	for _, s := range ss {
		if s == "" {
			continue
		}
		a := strings.Split(s, "|")
		if len(a) != 2 {
			return nil, fmt.Errorf("bad link: <%s>", s)
		}
		if !hasRegistration || a[0] != "Anmeldung" {
			links = append(links, utils.CreateLink(a[0], a[1]))
		}
	}
	return links, nil
}
