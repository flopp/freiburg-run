package events

import (
	"context"
	"fmt"
	"html/template"
	"log"
	"strings"
	"time"

	"github.com/flopp/freiburg-run/internal/utils"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
)

type SheetsData struct {
	Events  []*Event
	Groups  []*Event
	Shops   []*Event
	Parkrun []*ParkrunEvent
	Tags    []*Tag
	Series  []*Serie
}

func LoadSheets(config utils.Config, today time.Time) (SheetsData, error) {
	ctx := context.Background()
	srv, err := sheets.NewService(ctx, option.WithAPIKey(config.Google.ApiKey))
	if err != nil {
		return SheetsData{}, fmt.Errorf("creating sheets service: %w", err)
	}

	sheets, err := getAllSheets(config, srv)
	if err != nil {
		return SheetsData{}, fmt.Errorf("fetching all sheets: %w", err)
	}

	eventSheets, groupsSheet, shopsSheet, parkrunSheet, tagsSheet, seriesSheet, err := findSheetNames(config, sheets)
	if err != nil {
		return SheetsData{}, err
	}

	events, err := loadEvents(config, srv, today, eventSheets)
	if err != nil {
		return SheetsData{}, fmt.Errorf("fetching events: %w", err)
	}
	groups, err := fetchEvents(config, srv, today, "group", groupsSheet)
	if err != nil {
		return SheetsData{}, fmt.Errorf("fetching groups: %w", err)
	}
	shops, err := fetchEvents(config, srv, today, "shop", shopsSheet)
	if err != nil {
		return SheetsData{}, fmt.Errorf("fetching shops: %w", err)
	}
	var parkrun []*ParkrunEvent
	if config.Pages.Parkrun {
		parkrun, err = fetchParkrunEvents(config, srv, today, parkrunSheet)
		if err != nil {
			return SheetsData{}, fmt.Errorf("fetching parkrun events: %w", err)
		}
	}
	tags, err := fetchTags(config, srv, tagsSheet)
	if err != nil {
		return SheetsData{}, fmt.Errorf("fetching tags: %w", err)
	}
	series, err := fetchSeries(config, srv, seriesSheet)
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

func findSheetNames(config utils.Config, sheets []string) (eventSheets []string, groupsSheet, shopsSheet, parkrunSheet, tagsSheet, seriesSheet string, err error) {
	for _, sheet := range sheets {
		switch {
		case strings.HasPrefix(sheet, "Events"):
			eventSheets = append(eventSheets, sheet)
		case sheet == "Groups":
			groupsSheet = sheet
		case sheet == "Shops":
			shopsSheet = sheet
		case sheet == "Parkrun":
			parkrunSheet = sheet
		case sheet == "Tags":
			tagsSheet = sheet
		case sheet == "Series":
			seriesSheet = sheet
		case strings.Contains(sheet, "ignore"):
			// ignore
		default:
			log.Printf("ignoring unknown sheet: '%s'", sheet)
		}
	}
	if len(eventSheets) < 2 {
		return nil, "", "", "", "", "", fmt.Errorf("fetching sheets: unable to find enough 'Events' sheets")
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

func loadEvents(config utils.Config, srv *sheets.Service, today time.Time, eventSheets []string) ([]*Event, error) {
	eventList := make([]*Event, 0)
	for _, sheet := range eventSheets {
		yearList, err := fetchEvents(config, srv, today, "event", sheet)
		if err != nil {
			return nil, err
		}
		eventList = append(eventList, yearList...)
	}
	return eventList, nil
}

func getAllSheets(config utils.Config, srv *sheets.Service) ([]string, error) {
	response, err := srv.Spreadsheets.Get(config.Google.SheetId).Fields("sheets(properties(sheetId,title))").Do()
	if err != nil {
		return nil, err
	}
	if response.HTTPStatusCode != 200 {
		return nil, fmt.Errorf("http status %v when trying to get sheets", response.HTTPStatusCode)
	}
	sheets := make([]string, 0)
	for _, v := range response.Sheets {
		prop := v.Properties
		sheets = append(sheets, prop.Title)
	}
	return sheets, nil
}

type Columns struct {
	index map[string]int
}

func initColumns(row []interface{}) (Columns, error) {
	index := make(map[string]int)
	for col, value := range row {
		s := fmt.Sprintf("%v", value)
		if existingCol, found := index[s]; found {
			return Columns{}, fmt.Errorf("duplicate title '%s' in columns %d and %d", s, existingCol, col)
		}
		index[s] = col
	}
	return Columns{index}, nil
}

func (cols Columns) getIndex(title string) int {
	col, found := cols.index[title]
	if !found {
		return -1
	}
	return col
}

func (cols *Columns) getVal(col string, row []interface{}) (string, error) {
	colIndex := cols.getIndex(col)
	if colIndex < 0 {
		return "", fmt.Errorf("missing column '%s'", col)
	}
	if colIndex >= len(row) {
		return "", nil
	}
	return fmt.Sprintf("%v", row[colIndex]), nil
}

func fetchTable(config utils.Config, srv *sheets.Service, table string) (Columns, [][]interface{}, error) {
	resp, err := srv.Spreadsheets.Values.Get(config.Google.SheetId, fmt.Sprintf("%s!A1:Z", table)).Do()
	if err != nil {
		return Columns{}, nil, fmt.Errorf("cannot fetch table '%s': %v", table, err)
	}
	if len(resp.Values) == 0 {
		return Columns{}, nil, fmt.Errorf("got 0 rows when fetching table '%s'", table)
	}
	cols := Columns{}
	rows := make([][]interface{}, 0, len(resp.Values)-1)
	for line, row := range resp.Values {
		if line == 0 {
			cols, err = initColumns(row)
			if err != nil {
				return Columns{}, nil, fmt.Errorf("failed to parse rows when fetching table '%s': %v", table, err)
			}
			continue
		}
		rows = append(rows, row)
	}
	return cols, rows, nil
}

func getLinks(cols Columns, row []interface{}) []string {
	links := make([]string, 0)

	for i := 1; true; i += 1 {
		link, err := cols.getVal(fmt.Sprintf("LINK%d", i), row)
		if err != nil {
			break
		}
		links = append(links, link)
	}

	return links
}

type EventData struct {
	Date         string
	Name         string
	Name2        string
	Seo          string
	Status       string
	Url          string
	Description  string
	Location     string
	Coordinates  string
	Registration string
	Tags         string
	Links        []string
}

func getEventData(cols Columns, row []interface{}) (EventData, error) {
	var data EventData
	var err error
	fields := []struct {
		name string
		dest *string
	}{
		{"DATE", &data.Date},
		{"NAME", &data.Name},
		{"NAME2", &data.Name2},
		{"SEO", &data.Seo},
		{"STATUS", &data.Status},
		{"URL", &data.Url},
		{"DESCRIPTION", &data.Description},
		{"LOCATION", &data.Location},
		{"COORDINATES", &data.Coordinates},
		{"REGISTRATION", &data.Registration},
		{"TAGS", &data.Tags},
	}
	for _, f := range fields {
		*f.dest, err = cols.getVal(f.name, row)
		if err != nil {
			return EventData{}, err
		}
	}
	data.Links = getLinks(cols, row)
	return data, nil
}

func fetchEvents(config utils.Config, srv *sheets.Service, today time.Time, eventType string, table string) ([]*Event, error) {
	cols, rows, err := fetchTable(config, srv, table)
	if err != nil {
		return nil, err
	}

	eventsList := make([]*Event, 0)
	for line, row := range rows {
		data, err := getEventData(cols, row)
		if err != nil {
			return nil, fmt.Errorf("table '%s', line '%d': %v", table, line, err)
		}
		cancelled := strings.Contains(data.Status, "abgesagt") || strings.Contains(data.Status, "geschlossen")
		if cancelled && data.Status == "abgesagt" {
			data.Status = ""
		}
		special := data.Status == "spezial"
		obsolete := data.Status == "obsolete"
		if special || obsolete {
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
			data.Status,
			cancelled,
			obsolete,
			special,
			location,
			template.HTML(description1),
			template.HTML(description2),
			utils.CreateUnnamedLink(url),
			utils.SortAndUniquify(tags),
			nil,
			series,
			nil,
			links,
			"",
			"",
			"",
			false,
			nil,
			nil,
			nil,
			EventMeta{
				false,
				utils.NewName(data.Name2),
				data.Seo,
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

func getParkrunEventData(cols Columns, row []interface{}) (ParkrunEventData, error) {
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
		*f.dest, err = cols.getVal(f.name, row)
		if err != nil {
			return ParkrunEventData{}, err
		}
	}
	return data, nil
}

func fetchParkrunEvents(config utils.Config, srv *sheets.Service, today time.Time, table string) ([]*ParkrunEvent, error) {
	cols, rows, err := fetchTable(config, srv, table)
	if err != nil {
		return nil, err
	}

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

func getTagData(cols Columns, row []interface{}) (TagData, error) {
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
		*f.dest, err = cols.getVal(f.name, row)
		if err != nil {
			return TagData{}, err
		}
	}
	return data, nil
}

func fetchTags(config utils.Config, srv *sheets.Service, table string) ([]*Tag, error) {
	cols, rows, err := fetchTable(config, srv, table)
	if err != nil {
		return nil, err
	}

	tags := make([]*Tag, 0)
	for _, row := range rows {
		data, err := getTagData(cols, row)
		if err != nil {
			return nil, fmt.Errorf("table '%s': %v", table, err)
		}

		tag := utils.SanitizeName(data.Tag)
		if tag != "" && (data.Name != "" || data.Description != "") {
			t := CreateTag(tag)
			t.Name.Orig = data.Name
			t.Description = data.Description
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

func getSerieData(cols Columns, row []interface{}) (SerieData, error) {
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
		*f.dest, err = cols.getVal(f.name, row)
		if err != nil {
			return SerieData{}, err
		}
	}
	data.Links = getLinks(cols, row)
	return data, nil
}

func fetchSeries(config utils.Config, srv *sheets.Service, table string) ([]*Serie, error) {
	cols, rows, err := fetchTable(config, srv, table)
	if err != nil {
		return nil, err
	}

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
