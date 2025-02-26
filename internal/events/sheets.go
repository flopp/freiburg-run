package events

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"os"
	"strconv"
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
		eventList = append(eventList, fetchEvents(config, srv, today, "event", sheet)...)
	}
	groups := fetchEvents(config, srv, today, "group", groupsSheet)
	shops := fetchEvents(config, srv, today, "shop", shopsSheet)
	parkrun := fetchParkrunEvents(config, srv, today, parkrunSheet)
	tags := fetchTags(config, srv, tagsSheet)
	series := fetchSeries(config, srv, seriesSheet)

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

func (cols *Columns) getValue(title string, row []interface{}) string {
	col, found := cols.index[title]
	if !found {
		panic(fmt.Errorf("requested column not found: %s", title))
	}

	if col >= len(row) {
		return ""
	}
	return fmt.Sprintf("%v", row[col])
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

func fetchEvents(config SheetsConfigData, srv *sheets.Service, today time.Time, eventType string, table string) []*Event {
	cols, rows, err := fetchTable(config, srv, table)
	utils.Check(err)
	eventsList := make([]*Event, 0)
	for line, row := range rows {
		dateS := cols.getValue("DATE", row)
		nameS := cols.getValue("NAME", row)
		statusS := cols.getValue("STATUS", row)
		cancelled := strings.HasPrefix(statusS, "abgesagt")
		if cancelled && statusS == "abgesagt" {
			statusS = ""
		}
		special := statusS == "spezial"
		obsolete := statusS == "obsolete"
		if special || obsolete {
			statusS = ""
		}
		urlS := cols.getValue("URL", row)
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
		if urlS == "" {
			log.Printf("table '%s', line '%d': skipping row with empty url", table, line)
			continue
		}

		descriptionS := cols.getValue("DESCRIPTION", row)
		locationS := cols.getValue("LOCATION", row)
		coordinatesS := cols.getValue("COORDINATES", row)
		registration := cols.getValue("REGISTRATION", row)
		tagsS := cols.getValue("TAGS", row)
		linksS := make([]string, 4)
		linksS[0] = cols.getValue("LINK1", row)
		linksS[1] = cols.getValue("LINK2", row)
		linksS[2] = cols.getValue("LINK3", row)
		linksS[3] = cols.getValue("LINK4", row)

		name, nameOld := utils.SplitDetails(nameS)
		url := urlS
		description1, description2 := utils.SplitDetails(descriptionS)
		tags := make([]string, 0)
		series := make([]string, 0)
		for _, t := range utils.Split(tagsS) {
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
		links := parseLinks(linksS, registration)

		eventsList = append(eventsList, &Event{
			eventType,
			name,
			nameOld,
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
			false,
			nil,
			nil,
			nil,
		})
	}

	return eventsList
}

func fetchParkrunEvents(config SheetsConfigData, srv *sheets.Service, today time.Time, table string) []*ParkrunEvent {
	cols, rows, err := fetchTable(config, srv, table)
	utils.Check(err)
	eventsList := make([]*ParkrunEvent, 0)
	for _, row := range rows {
		index := cols.getValue("INDEX", row)
		date := cols.getValue("DATE", row)
		runners := cols.getValue("RUNNERS", row)
		temp := cols.getValue("TEMP", row)
		special := cols.getValue("SPECIAL", row)
		cafe := cols.getValue("CAFE", row)
		results := cols.getValue("RESULTS", row)
		report := cols.getValue("REPORT", row)
		author := cols.getValue("AUTHOR", row)
		photos := cols.getValue("PHOTOS", row)

		if temp != "" {
			temp = fmt.Sprintf("%s°C", temp)
		}

		if results != "" {
			// if "results" only contains a number, build full url
			if _, err := strconv.ParseInt(results, 10, 64); err == nil {
				results = fmt.Sprintf("https://www.parkrun.com.de/dietenbach/results/%s", results)
			}
		}

		currentWeek := false
		d, err := utils.ParseDate(date)
		if err == nil {
			today_y, today_m, today_d := today.Date()
			d_y, d_m, d_d := d.Date()
			currentWeek = (today_y == d_y && today_m == d_m && today_d == d_d) || (today.After(d) && today.Before(d.AddDate(0, 0, 7)))
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

	return eventsList
}

func fetchTags(config SheetsConfigData, srv *sheets.Service, table string) []*Tag {
	cols, rows, err := fetchTable(config, srv, table)
	utils.Check(err)
	tags := make([]*Tag, 0)
	for _, row := range rows {
		tagS := cols.getValue("TAG", row)
		nameS := cols.getValue("NAME", row)
		descriptionS := cols.getValue("DESCRIPTION", row)

		tag := utils.SanitizeName(tagS)
		if tag != "" && (nameS != "" || descriptionS != "") {
			t := CreateTag(tag)
			t.Name = nameS
			t.Description = descriptionS
			tags = append(tags, t)
		}
	}

	return tags
}

func fetchSeries(config SheetsConfigData, srv *sheets.Service, table string) []*Serie {
	cols, rows, err := fetchTable(config, srv, table)
	utils.Check(err)
	series := make([]*Serie, 0)
	for _, row := range rows {
		nameS := cols.getValue("NAME", row)
		descriptionS := cols.getValue("DESCRIPTION", row)
		linksS := make([]string, 4)
		linksS[0] = cols.getValue("LINK1", row)
		linksS[1] = cols.getValue("LINK2", row)
		linksS[2] = cols.getValue("LINK3", row)
		linksS[3] = cols.getValue("LINK4", row)

		id := utils.SanitizeName(nameS)
		series = append(series, &Serie{id, nameS, template.HTML(descriptionS), parseLinks(linksS, ""), make([]*Event, 0), make([]*Event, 0), make([]*Event, 0), make([]*Event, 0)})
	}

	return series
}

func parseLinks(ss []string, registration string) []*utils.NameUrl {
	links := make([]*utils.NameUrl, 0, len(ss))
	hasRegistration := registration != ""
	if hasRegistration {
		links = append(links, &utils.NameUrl{"Anmeldung", registration})
	}
	for _, s := range ss {
		if s == "" {
			continue
		}
		a := strings.Split(s, "|")
		if len(a) != 2 {
			panic(fmt.Errorf("bad link: <%s>", s))
		}
		if !hasRegistration || a[0] != "Anmeldung" {
			links = append(links, &utils.NameUrl{a[0], a[1]})
		}
	}
	return links
}
