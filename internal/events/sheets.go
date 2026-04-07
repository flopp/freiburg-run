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

// LoadSheets loads all data from the given sheets client and returns it structured in a SheetsData struct.
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

// getYearFromEventSheetName extracts the year from an event sheet name in the format "EventsYYYY" and validates the format.
func getYearFromEventSheetName(sheetName string) (int, error) {
	if !strings.HasPrefix(sheetName, "Events") {
		return 0, fmt.Errorf("unexpected event sheet name '%s' (no 'Events' prefix)", sheetName)
	}
	if len(sheetName) != 10 {
		return 0, fmt.Errorf("unexpected event sheet name '%s' (bad length %d, expected 10)", sheetName, len(sheetName))
	}
	yearStr := sheetName[6:]
	date, err := time.Parse("2006", yearStr)
	if err != nil {
		return 0, fmt.Errorf("unexpected event sheet name '%s' (yearStr='%s'): %v", sheetName, yearStr, err)
	}
	return date.Year(), nil
}

// findSheetNames identifies the relevant sheet names for events, groups, shops, parkrun, tags and series based on their names and validates them.
func findSheetNames(config utils.Config, sheets map[string][][]string) (eventSheets []string, groupsSheet, shopsSheet, parkrunSheet, tagsSheet, seriesSheet string, err error) {
	for sheetName := range sheets {
		name := strings.ToLower(sheetName)
		switch {
		case strings.HasPrefix(name, "events"):
			eventSheets = append(eventSheets, sheetName)
		case name == "groups":
			groupsSheet = sheetName
		case name == "shops":
			shopsSheet = sheetName
		case name == "parkrun":
			parkrunSheet = sheetName
		case name == "tags":
			tagsSheet = sheetName
		case name == "series":
			seriesSheet = sheetName
		case strings.Contains(name, "ignore"):
			// ignore
		default:
			log.Printf("ignoring unknown sheet: '%s'", sheetName)
		}
	}

	// we require at least 2 event sheets, so that we have some old events to show on the "Vergangene Events" page and to test the old/new logic
	if len(eventSheets) < 2 {
		return nil, "", "", "", "", "", fmt.Errorf("fetching sheets: unable to find enough 'Events' sheets")
	}

	// sort eventSheets by name, so that they are always in the same order (e.g. for testing)
	sort.Slice(eventSheets, func(i, j int) bool { return eventSheets[i] < eventSheets[j] })
	// validate eventSheets names ("EventsYYYY" format, YYYY=year), order and consecutiveness (no missing years)
	lastYear := -1
	for _, sheetName := range eventSheets {
		year, err := getYearFromEventSheetName(sheetName)
		if err != nil {
			return nil, "", "", "", "", "", fmt.Errorf("fetching sheets: %v", err)
		}
		if lastYear != -1 && year != lastYear+1 {
			return nil, "", "", "", "", "", fmt.Errorf("fetching sheets: unexpected event sheet name '%s': missing year %d", sheetName, lastYear+1)
		}
		lastYear = year
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

// loadEvents loads event data from the given event sheets and returns a list of Event structs.
// It uses the provided sheets data and validates the format of the event sheets.
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

// getVal is a helper to extract a value from a cols map and row slice.
// It returns the value as a string or an error if the column is missing.
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

// extractFields is a helper to fill struct fields from a cols map and row slice.
// It takes a slice of field definitions (name and pointer to string) and fills them using getVal.
func extractFields(cols map[string]int, row []string, fields []struct {
	name string
	dest *string
}) error {
	for _, f := range fields {
		val, err := getVal(cols, f.name, row)
		if err != nil {
			return err
		}
		*f.dest = val
	}
	return nil
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
	if err := extractFields(cols, row, fields); err != nil {
		return EventData{}, err
	}
	// Inline getLinks logic
	links := make([]string, 0)
	for i := 1; true; i++ {
		link, err := getVal(cols, fmt.Sprintf("LINK%d", i), row)
		if err != nil {
			break
		}
		links = append(links, link)
	}
	data.Links = links
	return data, nil
}

// fetchEvents extracts event data from the given sheet and returns a list of Event structs.
func fetchEvents(config utils.Config, today time.Time, eventType string, sheetName string, sheetsData map[string][][]string) ([]*Event, error) {
	sheet, ok := sheetsData[sheetName]
	if !ok {
		return nil, fmt.Errorf("sheet '%s' not found", sheetName)
	}

	requiredHeaders := []string{"DATE", "ADDED", "NAME", "NAME2", "STATUS", "URL", "DESCRIPTION", "LOCATION", "COORDINATES", "REGISTRATION", "TAGS"}
	cols, err := googlesheetswrapper.ExtractHeader(sheet, requiredHeaders, true)
	if err != nil {
		return nil, fmt.Errorf("fetching sheet '%s': %v", sheetName, err)
	}

	sheetYear := -1
	if eventType == "event" {
		sheetYear, err = getYearFromEventSheetName(sheetName)
		if err != nil {
			return nil, fmt.Errorf("fetching events: %v", err)
		}
	}

	eventsList := make([]*Event, 0)
	for line, row := range sheet[1:] {
		data, err := getEventData(cols, row)
		if err != nil {
			return nil, fmt.Errorf("sheet '%s', line '%d': %v", sheetName, line+1, err)
		}

		// process status
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
			log.Printf("sheet '%s', line '%d': skipping row with temp status", sheetName, line)
			continue
		}

		// process names
		name, nameOld := utils.SplitPair(data.Name)
		if data.Name == "" {
			log.Printf("sheet '%s', line '%d': skipping row with empty name", sheetName, line)
			continue
		}
		if !strings.Contains(data.Name, data.Name2) {
			log.Printf("sheet '%s', line '%d': name '%s' does not contain name2 '%s'", sheetName, line, data.Name, data.Name2)
		}

		// process date
		if eventType == "event" {
			if data.Date == "" {
				log.Printf("sheet '%s', line '%d': skipping row with empty date", sheetName, line)
				continue
			}
		}
		timeRange, err := utils.CreateTimeRange(data.Date)
		if err != nil {
			log.Printf("sheet '%s', line '%d': %v", sheetName, line, err)
		}
		if !timeRange.IsZero() && sheetYear >= 0 {
			if timeRange.From.Year() != sheetYear && timeRange.To.Year() != sheetYear {
				log.Printf("sheet '%s', line '%d': warning: event date '%s' does not match sheet year %d", sheetName, line, data.Date, sheetYear)
			}
		}

		isOld := timeRange.Before(today)

		// process url
		if data.Url == "" {
			log.Printf("sheet '%s', line '%d': skipping row with empty url", sheetName, line)
			continue
		}
		url := data.Url

		// process registration
		var registrationLink *utils.Link
		if data.Registration != "" {
			registrationLink = utils.CreateLink("Anmeldung", data.Registration)
		}

		// process description
		description1, description2 := utils.SplitPair(data.Description)

		// process location
		location := CreateLocation(config, data.Location, data.Coordinates)

		// process tags and series
		tags := make([]string, 0)
		series := make([]string, 0)
		for _, t := range utils.SplitList(data.Tags) {
			if strings.HasPrefix(t, "serie") {
				series = append(series, t[6:])
			} else {
				tags = append(tags, utils.SanitizeName(t))
			}
		}
		// add location tags to event tags
		tags = append(tags, location.Tags()...)

		// process links
		links, err := parseLinks(data.Links)
		if err != nil {
			return nil, fmt.Errorf("sheet '%s', line '%d': parsing links of event '%s': %w", sheetName, line, name, err)
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
			registrationLink,
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
	if err := extractFields(cols, row, fields); err != nil {
		return ParkrunEventData{}, err
	}
	return data, nil
}

// fetchParkrunEvents extracts parkrun event data from the given sheet and returns a list of ParkrunEvent structs.
func fetchParkrunEvents(config utils.Config, today time.Time, sheetName string, sheetsData map[string][][]string) ([]*ParkrunEvent, error) {
	sheet, ok := sheetsData[sheetName]
	if !ok {
		return nil, fmt.Errorf("sheet '%s' not found", sheetName)
	}
	requiredHeaders := []string{"DATE", "INDEX", "RUNNERS", "TEMP", "SPECIAL", "CAFE", "RESULTS", "REPORT", "AUTHOR", "PHOTOS"}
	cols, err := googlesheetswrapper.ExtractHeader(sheet, requiredHeaders, true)
	if err != nil {
		return nil, fmt.Errorf("fetching sheet '%s': %v", sheetName, err)
	}
	rows := sheet[1:]

	eventsList := make([]*ParkrunEvent, 0)
	for _, row := range rows {
		data, err := getParkrunEventData(cols, row)
		if err != nil {
			return nil, fmt.Errorf("sheet '%s': %v", sheetName, err)
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
	fields := []struct {
		name string
		dest *string
	}{
		{"TAG", &data.Tag},
		{"NAME", &data.Name},
		{"DESCRIPTION", &data.Description},
	}
	if err := extractFields(cols, row, fields); err != nil {
		return TagData{}, err
	}
	return data, nil
}

// fetchTags extracts tag data from the given sheet and returns a list of Tag structs.
func fetchTags(config utils.Config, sheetName string, sheetsData map[string][][]string) ([]*Tag, error) {
	sheet, ok := sheetsData[sheetName]
	if !ok {
		return nil, fmt.Errorf("sheet '%s' not found", sheetName)
	}

	requiredHeaders := []string{"TAG", "NAME", "DESCRIPTION"}
	cols, err := googlesheetswrapper.ExtractHeader(sheet, requiredHeaders, true)
	if err != nil {
		return nil, fmt.Errorf("fetching sheet '%s': %v", sheetName, err)
	}
	rows := sheet[1:]

	tags := make([]*Tag, 0)
	for _, row := range rows {
		data, err := getTagData(cols, row)
		if err != nil {
			return nil, fmt.Errorf("sheet '%s': %v", sheetName, err)
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
	fields := []struct {
		name string
		dest *string
	}{
		{"NAME", &data.Name},
		{"DESCRIPTION", &data.Description},
	}
	if err := extractFields(cols, row, fields); err != nil {
		return SerieData{}, err
	}

	// Inline getLinks logic
	data.Links = make([]string, 0)
	for i := 1; true; i++ {
		link, err := getVal(cols, fmt.Sprintf("LINK%d", i), row)
		if err != nil {
			break
		}
		data.Links = append(data.Links, link)
	}

	return data, nil
}

// fetchSeries extracts run event series data from the given sheet.
// It expects columns "NAME", "DESCRIPTION" and optional "LINK1", "LINK2", ...
func fetchSeries(config utils.Config, sheetName string, sheetsData map[string][][]string) ([]*Serie, error) {
	sheet, ok := sheetsData[sheetName]
	if !ok {
		return nil, fmt.Errorf("sheet '%s' not found", sheetName)
	}
	requiredHeaders := []string{"NAME", "DESCRIPTION"}
	cols, err := googlesheetswrapper.ExtractHeader(sheet, requiredHeaders, true)
	if err != nil {
		return nil, fmt.Errorf("fetching sheet '%s': %v", sheetName, err)
	}
	rows := sheet[1:]

	series := make([]*Serie, 0)
	for line, row := range rows {
		data, err := getSerieData(cols, row)
		if err != nil {
			return nil, fmt.Errorf("sheet '%s', line '%d': %v", sheetName, line+2, err)
		}
		links, err := parseLinks(data.Links)
		if err != nil {
			return nil, fmt.Errorf("sheet '%s', line '%d': parsing links of series '%s': %w", sheetName, line+2, data.Name, err)
		}
		series = append(series, &Serie{utils.NewName(data.Name), template.HTML(data.Description), links, make([]*Event, 0), make([]*Event, 0), make([]*Event, 0), make([]*Event, 0)})
	}

	return series, nil
}

func parseLinks(ss []string) ([]*utils.Link, error) {
	links := make([]*utils.Link, 0, len(ss))
	for _, s := range ss {
		if s == "" {
			continue
		}
		a := strings.Split(s, "|")
		if len(a) != 2 {
			return nil, fmt.Errorf("bad link: <%s>", s)
		}
		links = append(links, utils.CreateLink(a[0], a[1]))
	}
	return links, nil
}
