package events

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"os"
	"strings"
	"time"

	"github.com/flopp/freiburg-run/internal/utils"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
)

type SheetsConfigData struct {
	ApiKey  string `json:"api_key"`
	SheetId string `json:"sheet_id"`
}

func LoadSheetsConfig(path string) (SheetsConfigData, error) {
	config_data, err := os.ReadFile(path)
	if err != nil {
		return SheetsConfigData{}, fmt.Errorf("load sheets config file '%s': %w", path, err)
	}
	var config SheetsConfigData
	err = json.Unmarshal(config_data, &config)
	if err != nil {
		return SheetsConfigData{}, fmt.Errorf("unmarshall sheets config data: %w", err)
	}

	return config, nil
}

func LoadSheets(config SheetsConfigData, today time.Time) ([]*Event, []*Event, []*Event, []*ParkrunEvent, []*Tag, []*Serie, error) {
	ctx := context.Background()

	srv, err := sheets.NewService(ctx, option.WithAPIKey(config.ApiKey))
	if err != nil {
		return nil, nil, nil, nil, nil, nil, fmt.Errorf("creating sheets service: %w", err)
	}

	sheets, err := getAllSheets(config, srv)
	if err != nil {
		return nil, nil, nil, nil, nil, nil, fmt.Errorf("fetching all sheets: %w", err)
	}

	eventSheets := make([]string, 0)
	groupsSheet := ""
	shopsSheet := ""
	parkrunSheet := ""
	tagsSheet := ""
	seriesSheet := ""
	for _, sheet := range sheets {
		if strings.HasPrefix(sheet, "Events") {
			eventSheets = append(eventSheets, sheet)
		} else if sheet == "Groups" {
			groupsSheet = sheet
		} else if sheet == "Shops" {
			shopsSheet = sheet
		} else if sheet == "Parkrun" {
			parkrunSheet = sheet
		} else if sheet == "Tags" {
			tagsSheet = sheet
		} else if sheet == "Series" {
			seriesSheet = sheet
		} else if strings.Contains(sheet, "ignore") {
			// ignore
		} else {
			log.Printf("ignoring unknown sheet: '%s'", sheet)
		}
	}
	if len(eventSheets) < 2 {
		return nil, nil, nil, nil, nil, nil, fmt.Errorf("fetching sheets: unable to find enough 'Events' sheets")
	}
	if groupsSheet == "" {
		return nil, nil, nil, nil, nil, nil, fmt.Errorf("fetching sheets: unable to find 'Groups' sheet")
	}
	if shopsSheet == "" {
		return nil, nil, nil, nil, nil, nil, fmt.Errorf("fetching sheets: unable to find 'Shops' sheet")
	}
	if parkrunSheet == "" {
		return nil, nil, nil, nil, nil, nil, fmt.Errorf("fetching sheets: unable to find 'Parkrun' sheet")
	}
	if tagsSheet == "" {
		return nil, nil, nil, nil, nil, nil, fmt.Errorf("fetching sheets: unable to find 'Tags' sheet")
	}
	if seriesSheet == "" {
		return nil, nil, nil, nil, nil, nil, fmt.Errorf("fetching sheets: unable to find 'Series' sheet")
	}

	eventList := make([]*Event, 0)
	for _, sheet := range eventSheets {
		yearList, err := fetchEvents(config, srv, today, "event", sheet)
		if err != nil {
			return nil, nil, nil, nil, nil, nil, fmt.Errorf("fetching events: %w", err)
		}
		eventList = append(eventList, yearList...)
	}

	groups, err := fetchEvents(config, srv, today, "group", groupsSheet)
	if err != nil {
		return nil, nil, nil, nil, nil, nil, fmt.Errorf("fetching groups: %w", err)
	}

	shops, err := fetchEvents(config, srv, today, "shop", shopsSheet)
	if err != nil {
		return nil, nil, nil, nil, nil, nil, fmt.Errorf("fetching shops: %w", err)
	}

	parkrun, err := fetchParkrunEvents(config, srv, today, parkrunSheet)
	if err != nil {
		return nil, nil, nil, nil, nil, nil, fmt.Errorf("fetching parkrun events: %w", err)
	}

	tags, err := fetchTags(config, srv, tagsSheet)
	if err != nil {
		return nil, nil, nil, nil, nil, nil, fmt.Errorf("fetching tags: %w", err)
	}

	series, err := fetchSeries(config, srv, seriesSheet)
	if err != nil {
		return nil, nil, nil, nil, nil, nil, fmt.Errorf("fetching series: %w", err)
	}

	return eventList, groups, shops, parkrun, tags, series, nil
}

func getAllSheets(config SheetsConfigData, srv *sheets.Service) ([]string, error) {
	response, err := srv.Spreadsheets.Get(config.SheetId).Fields("sheets(properties(sheetId,title))").Do()
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

func getIndexValue(index int, row []interface{}) string {
	if index >= len(row) {
		return ""
	}
	return fmt.Sprintf("%v", row[index])
}

func (cols *Columns) getValue(col int, row []interface{}) string {
	return getIndexValue(col, row)
}

func fetchTable(config SheetsConfigData, srv *sheets.Service, table string) (Columns, [][]interface{}, error) {
	resp, err := srv.Spreadsheets.Values.Get(config.SheetId, fmt.Sprintf("%s!A1:Z", table)).Do()
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
		index := cols.getIndex(fmt.Sprintf("LINK%d", i))
		if index < 0 {
			break
		}
		link := getIndexValue(index, row)
		if link != "" {
			links = append(links, link)
		}
	}

	return links
}

func fetchEvents(config SheetsConfigData, srv *sheets.Service, today time.Time, eventType string, table string) ([]*Event, error) {
	// determine required columns
	cols, rows, err := fetchTable(config, srv, table)
	if err != nil {
		return nil, err
	}
	getCol := func(name string) (int, error) {
		col := cols.getIndex(name)
		if col < 0 {
			return -1, fmt.Errorf("table '%s': missing column '%s'", table, name)
		}
		return col, nil
	}
	colDate, err := getCol("DATE")
	if err != nil {
		return nil, err
	}
	colName, err := getCol("NAME")
	if err != nil {
		return nil, err
	}
	colName2, err := getCol("NAME2")
	if err != nil {
		return nil, err
	}
	colStatus, err := getCol("STATUS")
	if err != nil {
		return nil, err
	}
	colUrl, err := getCol("URL")
	if err != nil {
		return nil, err
	}
	colDescription, err := getCol("DESCRIPTION")
	if err != nil {
		return nil, err
	}
	colLocation, err := getCol("LOCATION")
	if err != nil {
		return nil, err
	}
	colCoordinates, err := getCol("COORDINATES")
	if err != nil {
		return nil, err
	}
	colRegistration, err := getCol("REGISTRATION")
	if err != nil {
		return nil, err
	}
	colTags, err := getCol("TAGS")
	if err != nil {
		return nil, err
	}

	eventsList := make([]*Event, 0)
	for line, row := range rows {
		dateS := cols.getValue(colDate, row)
		nameS := cols.getValue(colName, row)
		name2S := cols.getValue(colName2, row)
		statusS := cols.getValue(colStatus, row)
		cancelled := strings.HasPrefix(statusS, "abgesagt")
		if cancelled && statusS == "abgesagt" {
			statusS = ""
		}
		special := statusS == "spezial"
		obsolete := statusS == "obsolete"
		if special || obsolete {
			statusS = ""
		}
		urlS := cols.getValue(colUrl, row)
		if statusS == "temp" {
			log.Printf("table '%s', line '%d': skipping row with temp status", table, line)
			continue
		}
		if eventType == "event" {
			if dateS == "" {
				log.Printf("table '%s', line '%d': skipping row with empty date", table, line)
				continue
			}
		}
		if nameS == "" {
			log.Printf("table '%s', line '%d': skipping row with empty name", table, line)
			continue
		}
		if !strings.Contains(nameS, name2S) {
			log.Printf("table '%s', line '%d': name '%s' does not contain name2 '%s'", table, line, nameS, name2S)
		}
		if urlS == "" {
			log.Printf("table '%s', line '%d': skipping row with empty url", table, line)
			continue
		}

		descriptionS := cols.getValue(colDescription, row)
		locationS := cols.getValue(colLocation, row)
		coordinatesS := cols.getValue(colCoordinates, row)
		registration := cols.getValue(colRegistration, row)
		tagsS := cols.getValue(colTags, row)
		linksS := getLinks(cols, row)

		name, nameOld := utils.SplitPair(nameS)
		url := urlS
		description1, description2 := utils.SplitPair(descriptionS)
		tags := make([]string, 0)
		series := make([]string, 0)
		for _, t := range utils.SplitList(tagsS) {
			if strings.HasPrefix(t, "serie") {
				series = append(series, t[6:])
			} else {
				tags = append(tags, utils.SanitizeName(t))
			}
		}
		location := CreateLocation(locationS, coordinatesS)
		tags = append(tags, location.Tags()...)
		timeRange, err := utils.CreateTimeRange(dateS)
		if err != nil {
			log.Printf("event '%s': %v", name, err)
		}
		isOld := timeRange.Before(today)
		year := timeRange.Year()
		if year > 0 {
			tags = append(tags, fmt.Sprintf("%d", year))
		}
		links, err := parseLinks(linksS, registration)
		if err != nil {
			return nil, fmt.Errorf("parsing links of event '%s': %w", name, err)
		}

		eventsList = append(eventsList, &Event{
			eventType,
			utils.NewName(name),
			utils.NewName(nameOld),
			timeRange,
			isOld,
			statusS,
			cancelled,
			obsolete,
			special,
			location,
			description1,
			template.HTML(description2),
			url,
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
				utils.NewName(name2S),
				nil,
			},
		})
	}

	return eventsList, nil
}

func fetchParkrunEvents(config SheetsConfigData, srv *sheets.Service, today time.Time, table string) ([]*ParkrunEvent, error) {
	// determine required columns
	cols, rows, err := fetchTable(config, srv, table)
	if err != nil {
		return nil, err
	}
	getCol := func(name string) (int, error) {
		col := cols.getIndex(name)
		if col < 0 {
			return -1, fmt.Errorf("table '%s': missing column '%s'", table, name)
		}
		return col, nil
	}
	colIndex, err := getCol("INDEX")
	if err != nil {
		return nil, err
	}
	colDate, err := getCol("DATE")
	if err != nil {
		return nil, err
	}
	colRunners, err := getCol("RUNNERS")
	if err != nil {
		return nil, err
	}
	colTemp, err := getCol("TEMP")
	if err != nil {
		return nil, err
	}
	colSpecial, err := getCol("SPECIAL")
	if err != nil {
		return nil, err
	}
	colCafe, err := getCol("CAFE")
	if err != nil {
		return nil, err
	}
	colResults, err := getCol("RESULTS")
	if err != nil {
		return nil, err
	}
	colReport, err := getCol("REPORT")
	if err != nil {
		return nil, err
	}
	colAuthor, err := getCol("AUTHOR")
	if err != nil {
		return nil, err
	}
	colPhotos, err := getCol("PHOTOS")
	if err != nil {
		return nil, err
	}

	eventsList := make([]*ParkrunEvent, 0)
	for _, row := range rows {
		index := cols.getValue(colIndex, row)
		date := cols.getValue(colDate, row)
		runners := cols.getValue(colRunners, row)
		temp := cols.getValue(colTemp, row)
		special := cols.getValue(colSpecial, row)
		cafe := cols.getValue(colCafe, row)
		results := cols.getValue(colResults, row)
		report := cols.getValue(colReport, row)
		author := cols.getValue(colAuthor, row)
		photos := cols.getValue(colPhotos, row)

		if temp != "" {
			temp = fmt.Sprintf("%s°C", temp)
		}

		if results != "" {
			results = fmt.Sprintf("https://www.parkrun.com.de/dietenbach/results/%s", results)
		}

		// determine is this is for the current week (but only for "real" parkrun events with index)
		currentWeek := false
		if index != "" {
			d, err := utils.ParseDate(date)
			if err == nil {
				today_y, today_m, today_d := today.Date()
				d_y, d_m, d_d := d.Date()
				currentWeek = (today_y == d_y && today_m == d_m && today_d == d_d) || (today.After(d) && today.Before(d.AddDate(0, 0, 7)))
			}
		}

		eventsList = append(eventsList, &ParkrunEvent{
			currentWeek,
			index,
			date,
			runners,
			temp,
			special,
			cafe,
			results,
			report,
			author,
			photos,
		})
	}

	return eventsList, nil
}

func fetchTags(config SheetsConfigData, srv *sheets.Service, table string) ([]*Tag, error) {
	// determine required columns
	cols, rows, err := fetchTable(config, srv, table)
	if err != nil {
		return nil, err
	}
	getCol := func(name string) (int, error) {
		col := cols.getIndex(name)
		if col < 0 {
			return -1, fmt.Errorf("table '%s': missing column '%s'", table, name)
		}
		return col, nil
	}
	colTag, err := getCol("TAG")
	if err != nil {
		return nil, err
	}
	colName, err := getCol("NAME")
	if err != nil {
		return nil, err
	}
	colDescription, err := getCol("DESCRIPTION")
	if err != nil {
		return nil, err
	}

	tags := make([]*Tag, 0)
	for _, row := range rows {
		tagS := cols.getValue(colTag, row)
		nameS := cols.getValue(colName, row)
		descriptionS := cols.getValue(colDescription, row)

		tag := utils.SanitizeName(tagS)
		if tag != "" && (nameS != "" || descriptionS != "") {
			t := CreateTag(tag)
			t.Name.Orig = nameS
			t.Description = descriptionS
			tags = append(tags, t)
		}
	}

	return tags, nil
}

func fetchSeries(config SheetsConfigData, srv *sheets.Service, table string) ([]*Serie, error) {
	// determine required columns
	cols, rows, err := fetchTable(config, srv, table)
	if err != nil {
		return nil, err
	}
	getCol := func(name string) (int, error) {
		col := cols.getIndex(name)
		if col < 0 {
			return -1, fmt.Errorf("table '%s': missing column '%s'", table, name)
		}
		return col, nil
	}
	colName, err := getCol("NAME")
	if err != nil {
		return nil, err
	}
	colDescription, err := getCol("DESCRIPTION")
	if err != nil {
		return nil, err
	}

	series := make([]*Serie, 0)
	for _, row := range rows {
		nameS := cols.getValue(colName, row)
		descriptionS := cols.getValue(colDescription, row)
		linksS := getLinks(cols, row)
		links, err := parseLinks(linksS, "")
		if err != nil {
			return nil, fmt.Errorf("parsing links of series '%s': %w", nameS, err)
		}
		series = append(series, &Serie{utils.NewName(nameS), template.HTML(descriptionS), links, make([]*Event, 0), make([]*Event, 0), make([]*Event, 0), make([]*Event, 0)})
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
