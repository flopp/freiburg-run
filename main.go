package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/flopp/freiburg-run/internal/utils"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
)

const (
	usage = `USAGE: %s [OPTIONS...] [EVENTID...]

OPTIONS:
`
)

type CommandLineOptions struct {
	configFile string
	outDir     string
	hashFile   string
	addedFile  string
}

func parseCommandLine() CommandLineOptions {
	configFile := flag.String("config", "", "select config file")
	outDir := flag.String("out", ".out", "output directory")
	hashFile := flag.String("hashfile", ".hashes", "file storing file hashes (for sitemap)")
	addedFile := flag.String("addedfile", ".added", "file storing event addition dates")

	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), usage, os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse()

	if *configFile == "" {
		panic("You have to specify a config file, e.g. -config myconfig.json")
	}

	return CommandLineOptions{
		*configFile,
		*outDir,
		*hashFile,
		*addedFile,
	}
}

type NameUrl struct {
	Name string
	Url  string
}

func (n NameUrl) IsRegistration() bool {
	return n.Name == "Anmeldung"
}

type Location struct {
	City      string
	Country   string
	Geo       string
	Distance  string
	Direction string
}

func (loc Location) Name() string {
	if loc.City == "" {
		return ""
	}
	if loc.Country == "Frankreich" {
		return fmt.Sprintf(`%s, FR 🇫🇷`, loc.City)
	}
	if loc.Country == "Schweiz" {
		return fmt.Sprintf(`%s, CH 🇨🇭`, loc.City)
	}
	return loc.City
}

func (loc Location) NameNoFlag() string {
	if loc.City == "" {
		return ""
	}
	if loc.Country == "Frankreich" {
		return fmt.Sprintf(`%s, FR`, loc.City)
	}
	if loc.Country == "Schweiz" {
		return fmt.Sprintf(`%s, CH`, loc.City)
	}
	return loc.City
}

func (loc Location) HasGeo() bool {
	return loc.Geo != ""
}

func (loc Location) Dir() string {
	return fmt.Sprintf(`%s %s von Freiburg`, loc.Distance, loc.Direction)
}

func (loc Location) DirLong() string {
	return fmt.Sprintf(`%s %s von Freiburg Zentrum`, loc.Distance, loc.Direction)
}

func (loc Location) GoogleMaps() string {
	return fmt.Sprintf(`https://www.google.com/maps/place/%s`, loc.Geo)
}

func (loc Location) Tags() []string {
	tags := make([]string, 0)
	if loc.Country != "" {
		tags = append(tags, utils.SanitizeName(loc.Country))
	}
	// tags = append(tags, utils.SplitAndSanitize(loc.City)...)

	return tags
}

var reFr = regexp.MustCompile(`\s*^(.*)\s*,\s*FR\s*🇫🇷\s*$`)
var reCh = regexp.MustCompile(`\s*^(.*)\s*,\s*CH\s*🇨🇭\s*$`)

func createLocation(locationS, coordinatesS string) Location {
	country := ""
	if m := reFr.FindStringSubmatch(locationS); m != nil {
		country = "Frankreich"
		locationS = m[1]
	} else if m := reCh.FindStringSubmatch(locationS); m != nil {
		country = "Schweiz"
		locationS = m[1]
	}

	coordinates := utils.NormalizeGeo(coordinatesS)
	lat, lon, err := utils.LatLon(coordinates)
	distance := ""
	direction := ""
	if err == nil {
		// Freiburg
		lat0 := 47.996090
		lon0 := 7.849400
		d, b := utils.DistanceBearing(lat0, lon0, lat, lon)
		distance = fmt.Sprintf("%.1fkm", d)
		direction = utils.ApproxDirection(b)
	}

	return Location{locationS, country, coordinates, distance, direction}
}

type Event struct {
	Type      string
	Name      string
	NameOld   string
	Time      string
	TimeRange utils.TimeRange
	Old       bool
	Status    string
	Cancelled bool
	Special   bool
	Location  Location
	Details   string
	Details2  template.HTML
	Url       string
	RawTags   []string
	Tags      []*Tag
	RawSeries []string
	Series    []*Serie
	Links     []NameUrl
	Added     string
	New       bool
	Prev      *Event
	Next      *Event
}

func (event Event) IsSeparator() bool {
	return event.Type == ""
}

func createSeparatorEvent(label string) *Event {
	return &Event{
		"",
		label,
		"",
		"",
		utils.TimeRange{},
		false,
		"",
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
		true,
		nil,
		nil,
	}
}

func IsNew(s string, now time.Time) bool {
	days := 14

	d, err := utils.ParseDate(s)
	if err == nil {
		return d.AddDate(0, 0, days).After(now)
	}

	return false
}

var yearRe = regexp.MustCompile(`\b(\d\d\d\d)\b`)

func (event *Event) slug(ext string) string {
	t := event.Type

	if m := yearRe.FindStringSubmatch(event.Time); m != nil {
		return fmt.Sprintf("%s/%s-%s.%s", t, m[1], utils.SanitizeName(event.Name), ext)
	}
	return fmt.Sprintf("%s/%s.%s", t, utils.SanitizeName(event.Name), ext)
}

func (event *Event) SlugOld() string {
	if event.NameOld == "" {
		return ""
	}

	t := event.Type
	if strings.Contains(event.NameOld, "parkrun") {
		t = "event"
	}

	if m := yearRe.FindStringSubmatch(event.Time); m != nil {
		return fmt.Sprintf("%s/%s-%s.html", t, m[1], utils.SanitizeName(event.NameOld))
	}
	return fmt.Sprintf("%s/%s.html", t, utils.SanitizeName(event.NameOld))
}

func (event *Event) Slug() string {
	return event.slug("html")
}

func (event *Event) ImageSlug() string {
	return event.slug("png")
}

func (event *Event) LinkTitle() string {
	if event.Type == "event" {
		if strings.HasPrefix(event.Url, "mailto:") {
			return "Mail an Veranstalter"
		}
		return "Zur Veranstaltung"
	}
	if event.Type == "group" {
		if strings.HasPrefix(event.Url, "mailto:") {
			return "Mail an Organisator"
		}
		return "Zum Lauftreff"
	}
	if event.Type == "shop" {
		return "Zum Lauf-Shop"
	}
	return "Zur Veranstaltung"
}

func (event *Event) NiceType() string {
	if event.Old {
		return "vergangene Veranstaltung"
	}
	if event.Type == "event" {
		return "Veranstaltung"
	}
	if event.Type == "group" {
		return "Lauftreff"
	}
	if event.Type == "shop" {
		return "Lauf-Shop"
	}
	return "Veranstaltung"
}

type ParkrunEvent struct {
	IsCurrentWeek bool
	Index         string
	Date          string
	Runners       string
	Temp          string
	Special       string
	Results       string
	Report        string
	Photos        string
}

type Tag struct {
	Sanitized   string
	Name        string
	Description string
	Events      []*Event
	EventsOld   []*Event
	Groups      []*Event
	Shops       []*Event
}

func CreateTag(name string) *Tag {
	return &Tag{name, name, "", make([]*Event, 0), make([]*Event, 0), make([]*Event, 0), make([]*Event, 0)}
}

func (tag *Tag) Slug() string {
	return fmt.Sprintf("tag/%s.html", tag.Sanitized)
}

func (tag *Tag) NumEvents() int {
	return len(tag.Events)
}

func (tag *Tag) NumOldEvents() int {
	return len(tag.EventsOld)
}

func (tag *Tag) NumGroups() int {
	return len(tag.Groups)
}

func (tag *Tag) NumShops() int {
	return len(tag.Shops)
}

type Serie struct {
	Sanitized   string
	Name        string
	Description template.HTML
	Links       []NameUrl
	Events      []*Event
	EventsOld   []*Event
	Groups      []*Event
	Shops       []*Event
}

func (s Serie) IsOld() bool {
	return len(s.Events) == 0 && len(s.Groups) == 0 && len(s.Shops) == 0
}

func (s Serie) Num() int {
	return len(s.Events) + len(s.EventsOld) + len(s.Groups) + len(s.Shops)
}

func CreateSerie(id string, name string) *Serie {
	return &Serie{id, name, "", make([]NameUrl, 0), make([]*Event, 0), make([]*Event, 0), make([]*Event, 0), make([]*Event, 0)}
}

func (serie *Serie) Slug() string {
	return fmt.Sprintf("serie/%s.html", serie.Sanitized)
}

func (serie *Serie) ImageSlug() string {
	return fmt.Sprintf("serie/%s.png", serie.Sanitized)
}

type TemplateData struct {
	Title         string
	Type          string
	Description   string
	Nav           string
	Canonical     string
	Image         string
	Breadcrumbs   []utils.Breadcrumb
	Timestamp     string
	TimestampFull string
	SheetUrl      string
	Events        []*Event
	EventsOld     []*Event
	Groups        []*Event
	Shops         []*Event
	Parkrun       []*ParkrunEvent
	Tags          []*Tag
	Series        []*Serie
	SeriesOld     []*Serie
	JsFiles       []string
	CssFiles      []string
}

type EventTemplateData struct {
	Event         *Event
	Title         string
	Type          string
	Description   string
	Nav           string
	Canonical     string
	Image         string
	Breadcrumbs   []utils.Breadcrumb
	Main          string
	Timestamp     string
	TimestampFull string
	SheetUrl      string
	JsFiles       []string
	CssFiles      []string
}

type TagTemplateData struct {
	Tag           *Tag
	Title         string
	Type          string
	Description   string
	Nav           string
	Canonical     string
	Image         string
	Breadcrumbs   []utils.Breadcrumb
	Main          string
	Timestamp     string
	TimestampFull string
	SheetUrl      string
	JsFiles       []string
	CssFiles      []string
}

type SerieTemplateData struct {
	Serie         *Serie
	Title         string
	Type          string
	Description   string
	Nav           string
	Canonical     string
	Image         string
	Breadcrumbs   []utils.Breadcrumb
	Main          string
	Timestamp     string
	TimestampFull string
	SheetUrl      string
	JsFiles       []string
	CssFiles      []string
}

type SitemapTemplateData struct {
	Title         string
	Type          string
	Description   string
	Nav           string
	Canonical     string
	Image         string
	Breadcrumbs   []utils.Breadcrumb
	Timestamp     string
	TimestampFull string
	SheetUrl      string
	Categories    []utils.SitemapCategory
	JsFiles       []string
	CssFiles      []string
}

func GetMtimeYMD(filePath string) string {
	stat, err := os.Stat(filePath)
	if err != nil {
		return ""
	}

	return stat.ModTime().Format("2006-01-02")
}

type ConfigData struct {
	ApiKey  string `json:"api_key"`
	SheetId string `json:"sheet_id"`
}

func parseLinks(ss []string, registration string) []NameUrl {
	links := make([]NameUrl, 0)
	hasRegistration := registration != ""
	if hasRegistration {
		links = append(links, NameUrl{"Anmeldung", registration})
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
			links = append(links, NameUrl{a[0], a[1]})
		}
	}
	return links
}

func SplitDetails(s string) (string, string) {
	i := strings.Index(s, "|")
	if i > -1 {
		return s[:i], s[i+1:]
	}
	return s, ""
}

func getAllSheets(config ConfigData, srv *sheets.Service) ([]string, error) {
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

func fetchTable(config ConfigData, srv *sheets.Service, table string) (Columns, [][]interface{}, error) {
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

func fetchEvents(config ConfigData, srv *sheets.Service, today time.Time, eventType string, table string) []*Event {
	cols, rows, err := fetchTable(config, srv, table)
	utils.Check(err)
	events := make([]*Event, 0)
	for line, row := range rows {
		dateS := cols.getValue("DATE", row)
		nameS := cols.getValue("NAME", row)
		statusS := cols.getValue("STATUS", row)
		special := statusS == "spezial"
		cancelled := statusS == "abgesagt"
		if special || cancelled {
			statusS = ""
		}
		urlS := cols.getValue("URL", row)
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

		name, nameOld := SplitDetails(nameS)
		url := urlS
		description1, description2 := SplitDetails(descriptionS)
		tags := make([]string, 0)
		series := make([]string, 0)
		for _, t := range utils.Split(tagsS) {
			if strings.HasPrefix(t, "serie") {
				series = append(series, t[6:])
			} else {
				tags = append(tags, utils.SanitizeName(t))
			}
		}
		location := createLocation(locationS, coordinatesS)
		tags = append(tags, location.Tags()...)
		timeRange, err := utils.ParseTimeRange(dateS)
		if err != nil {
			log.Printf("event '%s': %v", name, err)
		}
		date, err := utils.InsertWeekdays(dateS)
		if err != nil {
			log.Printf("event '%s': %v", name, err)
		}
		isOld := (!timeRange.From.IsZero()) && timeRange.To.Before(today)
		if !timeRange.From.IsZero() {
			tags = append(tags, fmt.Sprintf("%d", timeRange.From.Year()))
		}
		links := parseLinks(linksS, registration)

		events = append(events, &Event{
			eventType,
			name,
			nameOld,
			date,
			timeRange,
			isOld,
			statusS,
			cancelled,
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
		})
	}

	return events
}

func fetchParkrunEvents(config ConfigData, srv *sheets.Service, today time.Time, table string) []*ParkrunEvent {
	cols, rows, err := fetchTable(config, srv, table)
	utils.Check(err)
	events := make([]*ParkrunEvent, 0)
	for _, row := range rows {
		index := cols.getValue("INDEX", row)
		date := cols.getValue("DATE", row)
		runners := cols.getValue("RUNNERS", row)
		temp := cols.getValue("TEMP", row)
		special := cols.getValue("SPECIAL", row)
		results := cols.getValue("RESULTS", row)
		report := cols.getValue("REPORT", row)
		photos := cols.getValue("PHOTOS", row)

		if temp != "" {
			temp = fmt.Sprintf("%s°C", temp)
		}

		currentWeek := false
		d, err := utils.ParseDate(date)
		if err == nil {
			today_y, today_m, today_d := today.Date()
			d_y, d_m, d_d := d.Date()
			currentWeek = (today_y == d_y && today_m == d_m && today_d == d_d) || (today.After(d) && today.Before(d.AddDate(0, 0, 7)))
		}

		events = append(events, &ParkrunEvent{
			currentWeek,
			index,
			date,
			runners,
			temp,
			special,
			results,
			report,
			photos,
		})
	}

	return events
}

func fetchTagDescriptions(config ConfigData, srv *sheets.Service, table string) map[string]NameDescription {
	cols, rows, err := fetchTable(config, srv, table)
	utils.Check(err)
	descriptions := make(map[string]NameDescription)
	for _, row := range rows {
		tagS := cols.getValue("TAG", row)
		nameS := cols.getValue("NAME", row)
		descriptionS := cols.getValue("DESCRIPTION", row)

		tag := utils.SanitizeName(tagS)
		if tag != "" && (nameS != "" || descriptionS != "") {
			descriptions[tag] = NameDescription{nameS, descriptionS}
		}
	}

	return descriptions
}

func fetchSeries(config ConfigData, srv *sheets.Service, table string) map[string]*Serie {
	cols, rows, err := fetchTable(config, srv, table)
	utils.Check(err)
	series := make(map[string]*Serie)
	for _, row := range rows {
		nameS := cols.getValue("NAME", row)
		descriptionS := cols.getValue("DESCRIPTION", row)
		linksS := make([]string, 4)
		linksS[0] = cols.getValue("LINK1", row)
		linksS[1] = cols.getValue("LINK2", row)
		linksS[2] = cols.getValue("LINK3", row)
		linksS[3] = cols.getValue("LINK4", row)

		id := utils.SanitizeName(nameS)
		if id != "" {
			series[id] = &Serie{id, nameS, template.HTML(descriptionS), parseLinks(linksS, ""), make([]*Event, 0), make([]*Event, 0), make([]*Event, 0), make([]*Event, 0)}
		}
	}

	return series
}

func createMonthLabel(t time.Time) string {
	if t.Month() == time.January {
		return fmt.Sprintf("Januar %d", t.Year())
	}
	if t.Month() == time.February {
		return fmt.Sprintf("Februar %d", t.Year())
	}
	if t.Month() == time.March {
		return fmt.Sprintf("März %d", t.Year())
	}
	if t.Month() == time.April {
		return fmt.Sprintf("April %d", t.Year())
	}
	if t.Month() == time.May {
		return fmt.Sprintf("Mai %d", t.Year())
	}
	if t.Month() == time.June {
		return fmt.Sprintf("Juni %d", t.Year())
	}
	if t.Month() == time.July {
		return fmt.Sprintf("Juli %d", t.Year())
	}
	if t.Month() == time.August {
		return fmt.Sprintf("August %d", t.Year())
	}
	if t.Month() == time.September {
		return fmt.Sprintf("September %d", t.Year())
	}
	if t.Month() == time.October {
		return fmt.Sprintf("Oktober %d", t.Year())
	}
	if t.Month() == time.November {
		return fmt.Sprintf("November %d", t.Year())
	}
	return fmt.Sprintf("Dezember %d", t.Year())
}

func isSimilarName(s1, s2 string) bool {
	var builder1 strings.Builder
	for _, r := range s1 {
		if unicode.IsLetter(r) {
			builder1.WriteRune(unicode.ToLower(r))
		}
	}
	var builder2 strings.Builder
	for _, r := range s2 {
		if unicode.IsLetter(r) {
			builder2.WriteRune(unicode.ToLower(r))
		}
	}
	return builder1.String() == builder2.String()
}

func validateDateOrder(events []*Event) {
	var lastDate time.Time
	for _, event := range events {
		if !lastDate.IsZero() {
			if event.TimeRange.From.IsZero() {
				log.Printf("event '%s' has no date but occurs after dated event", event.Name)
				return
			}
			if event.TimeRange.From.Before(lastDate) {
				log.Printf("event '%s' has date '%s' before date of previous event '%s'", event.Name, event.Time, lastDate.Format("2006-01-02"))
				return
			}
		}

		lastDate = event.TimeRange.From
	}
}

func findPrevNextEvents(events []*Event) {
	for _, event := range events {
		var prev *Event = nil
		for _, event2 := range events {
			if event2 == event {
				break
			}

			if isSimilarName(event2.Name, event.Name) && event2.Location.Geo == event.Location.Geo {
				prev = event2
			}
		}

		if prev != nil {
			prev.Next = event
			event.Prev = prev
		}
	}
}

func splitEvents(events []*Event) ([]*Event, []*Event) {
	futureEvents := make([]*Event, 0)
	pastEvents := make([]*Event, 0)

	for _, event := range events {
		if event.Old {
			pastEvents = append(pastEvents, event)
		} else {
			futureEvents = append(futureEvents, event)
		}
	}
	return futureEvents, pastEvents
}

func addMonthSeparators(events []*Event) []*Event {
	result := make([]*Event, 0, len(events))
	var last time.Time

	for i, event := range events {
		d := event.TimeRange.From
		if event.TimeRange.From.IsZero() {
			if i == 0 {
				result = append(result, createSeparatorEvent("Wöchentlich"))
			}
			// no label
		} else if last.IsZero() {
			// initial label
			last = d
			result = append(result, createSeparatorEvent(createMonthLabel(last)))
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
					result = append(result, createSeparatorEvent(createMonthLabel(last)))
				}
			}
		}

		result = append(result, event)
	}
	return result
}

func addMonthSeparatorsDescending(events []*Event) []*Event {
	result := make([]*Event, 0, len(events))
	var last time.Time

	for _, event := range events {
		d := event.TimeRange.From
		if event.TimeRange.From.IsZero() {
			// no label
		} else if last.IsZero() {
			// initial label
			last = d
			result = append(result, createSeparatorEvent(createMonthLabel(last)))
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
					result = append(result, createSeparatorEvent(createMonthLabel(last)))
				}
			}
		}

		result = append(result, event)
	}
	return result
}

func reverse(s []*Event) []*Event {
	a := make([]*Event, len(s))
	copy(a, s)

	for i := len(a)/2 - 1; i >= 0; i-- {
		opp := len(a) - 1 - i
		a[i], a[opp] = a[opp], a[i]
	}

	return a
}

func getTag(tags map[string]*Tag, name string) *Tag {
	if tag, found := tags[name]; found {
		return tag
	}
	tag := CreateTag(name)
	tags[name] = tag
	return tag
}

type NameDescription struct {
	Name        string
	Description string
}

func collectTags(descriptions map[string]NameDescription, events []*Event, eventsOld []*Event, groups []*Event, shops []*Event) (map[string]*Tag, []*Tag) {
	tags := make(map[string]*Tag)
	for _, e := range events {
		e.Tags = make([]*Tag, 0, len(e.RawTags))
		for _, t := range e.RawTags {
			tag := getTag(tags, t)
			e.Tags = append(e.Tags, tag)
			tag.Events = append(tag.Events, e)
		}
	}
	for _, e := range eventsOld {
		e.Tags = make([]*Tag, 0, len(e.RawTags))
		for _, t := range e.RawTags {
			tag := getTag(tags, t)
			e.Tags = append(e.Tags, tag)
			tag.EventsOld = append(tag.EventsOld, e)
		}
	}
	for _, e := range groups {
		e.Tags = make([]*Tag, 0, len(e.RawTags))
		for _, t := range e.RawTags {
			tag := getTag(tags, t)
			e.Tags = append(e.Tags, tag)
			tag.Groups = append(tag.Groups, e)
		}
	}
	for _, e := range shops {
		e.Tags = make([]*Tag, 0, len(e.RawTags))
		for _, t := range e.RawTags {
			tag := getTag(tags, t)
			e.Tags = append(e.Tags, tag)
			tag.Shops = append(tag.Shops, e)
		}
	}

	tagsList := make([]*Tag, 0, len(tags))
	for _, tag := range tags {
		desc, found := descriptions[tag.Sanitized]
		if found {
			if desc.Name != "" {
				tag.Name = desc.Name
			}
			tag.Description = desc.Description
		}
		tagsList = append(tagsList, tag)
	}
	sort.Slice(tagsList, func(i, j int) bool { return tagsList[i].Sanitized < tagsList[j].Sanitized })

	return tags, tagsList
}

func getSerie(series map[string]*Serie, name string) *Serie {
	id := utils.SanitizeName(name)
	if s, found := series[id]; found {
		return s
	}
	serie := CreateSerie(id, name)
	series[id] = serie
	return serie
}

func collectSeries(series map[string]*Serie, events []*Event, eventsOld []*Event, groups []*Event, shops []*Event) (map[string]*Serie, []*Serie, []*Serie) {
	for _, e := range events {
		e.Series = make([]*Serie, 0)
		for _, t := range e.RawSeries {
			serie := getSerie(series, t)
			e.Series = append(e.Series, serie)
			serie.Events = append(serie.Events, e)
		}
	}
	for _, e := range eventsOld {
		e.Series = make([]*Serie, 0)
		for _, t := range e.RawSeries {
			serie := getSerie(series, t)
			e.Series = append(e.Series, serie)
			serie.EventsOld = append(serie.EventsOld, e)
		}
	}
	for _, e := range groups {
		e.Series = make([]*Serie, 0)
		for _, t := range e.RawSeries {
			serie := getSerie(series, t)
			e.Series = append(e.Series, serie)
			serie.Groups = append(serie.Groups, e)
		}
	}
	for _, e := range shops {
		e.Series = make([]*Serie, 0)
		for _, t := range e.RawSeries {
			serie := getSerie(series, t)
			e.Series = append(e.Series, serie)
			serie.Shops = append(serie.Shops, e)
		}
	}

	seriesList := make([]*Serie, 0, len(series))
	seriesListOld := make([]*Serie, 0, len(series))
	for _, s := range series {
		if s.IsOld() {
			seriesListOld = append(seriesListOld, s)
		} else {
			seriesList = append(seriesList, s)
		}
	}
	sort.Slice(seriesList, func(i, j int) bool { return seriesList[i].Sanitized < seriesList[j].Sanitized })
	sort.Slice(seriesListOld, func(i, j int) bool { return seriesListOld[i].Sanitized < seriesListOld[j].Sanitized })

	return series, seriesList, seriesListOld
}

func updateAddedDates(events []*Event, added *utils.Added, eventType string, timestamp string, now time.Time) {
	for _, event := range events {
		fromFile, err := added.GetAdded(eventType, event.Slug())
		if err == nil {
			if fromFile == "" {
				if event.Added == "" {
					event.Added = timestamp
				}
				_ = added.SetAdded(eventType, event.Slug(), event.Added)
			} else {
				if event.Added == "" {
					event.Added = fromFile
				}
			}

		}
		event.New = IsNew(event.Added, now)
	}
}

func CreateHtaccess(events, events_old, groups, shops []*Event, outDir string) error {
	if err := utils.MakeDir(outDir); err != nil {
		return err
	}

	fileName := filepath.Join(outDir, ".htaccess")

	destination, err := os.Create(fileName)
	if err != nil {
		return err
	}
	defer destination.Close()

	destination.WriteString("ErrorDocument 404 /404.html\n")
	destination.WriteString("Redirect /parkrun /dietenbach-parkrun.html\n")
	destination.WriteString("Redirect /groups.html /lauftreffs.html\n")
	destination.WriteString("Redirect /event/2024-32-teninger-allmendlauf.html?back=event /event/2024-32-teninger-allmendlauf.html\n")
	destination.WriteString("Redirect /event/dietenbach-parkrun.html /group/dietenbach-parkrun.html\n")
	destination.WriteString("Redirect /event/dreilaendergarten-parkrun.html /group/dreilaendergarten-parkrun.html\n")
	for _, e := range events {
		if old := e.SlugOld(); old != "" {
			destination.WriteString(fmt.Sprintf("Redirect /%s /%s\n", old, e.Slug()))
		}
	}
	for _, e := range events_old {
		if old := e.SlugOld(); old != "" {
			destination.WriteString(fmt.Sprintf("Redirect /%s /%s\n", old, e.Slug()))
		}
	}
	for _, e := range groups {
		if old := e.SlugOld(); old != "" {
			destination.WriteString(fmt.Sprintf("Redirect /%s /%s\n", old, e.Slug()))
		}
	}
	for _, e := range shops {
		if old := e.SlugOld(); old != "" {
			destination.WriteString(fmt.Sprintf("Redirect /%s /%s\n", old, e.Slug()))
		}
	}

	return nil
}

func main() {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	timestamp := now.Format("2006-01-02")
	timestampFull := now.Format("2006-01-02 15:04:05")
	sheetUrl := ""
	options := parseCommandLine()

	var events []*Event
	var events_old []*Event
	var groups []*Event
	var shops []*Event
	var parkrun []*ParkrunEvent

	config_data, err := os.ReadFile(options.configFile)
	utils.Check(err)
	var config ConfigData
	if err := json.Unmarshal(config_data, &config); err != nil {
		panic(err)
	}

	sheetUrl = fmt.Sprintf("https://docs.google.com/spreadsheets/d/%s", config.SheetId)

	ctx := context.Background()
	srv, err := sheets.NewService(ctx, option.WithAPIKey(config.ApiKey))
	utils.Check(err)

	sheets, err := getAllSheets(config, srv)
	utils.Check(err)
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
		} else if strings.HasPrefix(sheet, "Notes") {
			// ignore
		} else {
			log.Printf("irgnoring unknown sheet: '%s'", sheet)
		}
	}
	if len(eventSheets) < 2 {
		panic("unable to find enough 'Events' sheets")
	}
	if groupsSheet == "" {
		panic("unable to find 'Groups' sheet")
	}
	if shopsSheet == "" {
		panic("unable to find 'Shops' sheet")
	}
	if parkrunSheet == "" {
		panic("unable to find 'Parkrun' sheet")
	}
	if tagsSheet == "" {
		panic("unable to find 'Tags' sheet")
	}
	if seriesSheet == "" {
		panic("unable to find 'Series' sheet")
	}

	events = make([]*Event, 0)
	for _, sheet := range eventSheets {
		events = append(events, fetchEvents(config, srv, today, "event", sheet)...)
	}
	groups = fetchEvents(config, srv, today, "group", groupsSheet)
	shops = fetchEvents(config, srv, today, "shop", shopsSheet)
	parkrun = fetchParkrunEvents(config, srv, today, parkrunSheet)
	tagDescriptions := fetchTagDescriptions(config, srv, tagsSheet)
	series := fetchSeries(config, srv, seriesSheet)

	if options.addedFile != "" {
		added, err := utils.ReadAdded(options.addedFile)
		if err != nil {
			log.Printf("failed to parse added file: '%s' - %v", options.addedFile, err)
		}

		updateAddedDates(events, added, "event", timestamp, now)
		updateAddedDates(groups, added, "group", timestamp, now)
		updateAddedDates(shops, added, "shop", timestamp, now)

		if err = added.Write(options.addedFile); err != nil {
			log.Printf("failed to write added file: '%s' - %v", options.addedFile, err)
		}
	}

	validateDateOrder(events)
	findPrevNextEvents(events)
	events, events_old = splitEvents(events)
	events = addMonthSeparators(events)
	events_old = reverse(events_old)
	events_old = addMonthSeparatorsDescending(events_old)
	tags, tagsList := collectTags(tagDescriptions, events, events_old, groups, shops)
	series, seriesList, seriesListOld := collectSeries(series, events, events_old, groups, shops)

	sitemap := utils.CreateSitemap("https://freiburg.run")
	sitemap.AddCategory("Allgemein")
	sitemap.AddCategory("Laufveranstaltungen")
	sitemap.AddCategory("Vergangene Laufveranstaltungen")
	sitemap.AddCategory("Kategorien")
	sitemap.AddCategory("Serien")
	sitemap.AddCategory("Lauftreffs")
	sitemap.AddCategory("Lauf-Shops")
	sitemap.Add("", "Alle Laufveranstaltungen", "Laufveranstaltungen")
	sitemap.Add("events-old.html", "Alle vergangenen Laufveranstaltungen", "Vergangene Laufveranstaltungen")
	sitemap.Add("tags.html", "Alle Kategorieren", "Kategorien")
	sitemap.Add("lauftreffs.html", "Alle Lauftreffes", "Lauftreffs")
	sitemap.Add("shops.html", "Alle Lauf-Shops", "Lauf-Shops")
	sitemap.Add("dietenbach-parkrun.html", "Dietenbach parkrun", "Allgemein")
	sitemap.Add("map.html", "Karte", "Laufveranstaltungen")
	sitemap.Add("series.html", "Alle Lauf-Serien", "Serien")
	sitemap.Add("info.html", "Informationen", "Allgemein")
	sitemap.Add("datenschutz.html", "Datenschutz", "Allgemein")
	sitemap.Add("impressum.html", "Impressum", "Allgemein")

	utils.MustCopyHash("static/robots.txt", "robots.txt", options.outDir)
	utils.MustCopyHash("static/favicon.png", "favicon.png", options.outDir)
	utils.MustCopyHash("static/favicon.ico", "favicon.ico", options.outDir)
	utils.MustCopyHash("static/apple-touch-icon.png", "apple-touch-icon.png", options.outDir)
	utils.MustCopyHash("static/freiburg-run.svg", "images/freiburg-run.svg", options.outDir)
	utils.MustCopyHash("static/events2023.jpg", "images/events2023.jpg", options.outDir)
	utils.MustCopyHash("static/parkrun.png", "images/parkrun.png", options.outDir)
	utils.MustCopyHash("static/marker-grey-icon.png", "images/marker-grey-icon.png", options.outDir)
	utils.MustCopyHash("static/marker-grey-icon-2x.png", "images/marker-grey-icon-2x.png", options.outDir)
	utils.MustCopyHash("static/marker-green-icon.png", "images/marker-green-icon.png", options.outDir)
	utils.MustCopyHash("static/marker-green-icon-2x.png", "images/marker-green-icon-2x.png", options.outDir)
	utils.MustCopyHash("static/marker-red-icon.png", "images/marker-red-icon.png", options.outDir)
	utils.MustCopyHash("static/marker-red-icon-2x.png", "images/marker-red-icon-2x.png", options.outDir)
	utils.MustCopyHash("static/circle-small.png", "images/circle-small.png", options.outDir)
	utils.MustCopyHash("static/circle-big.png", "images/circle-big.png", options.outDir)
	utils.MustCopyHash("static/freiburg-run-flyer.pdf", "freiburg-run-flyer.pdf", options.outDir)

	js_files := make([]string, 0)
	js_files = append(js_files, utils.MustDownloadHash("https://unpkg.com/leaflet@1.9.4/dist/leaflet.js", "leaflet-HASH.js", options.outDir))
	js_files = append(js_files, utils.MustDownloadHash("https://raw.githubusercontent.com/ptma/Leaflet.Legend/master/src/leaflet.legend.js", "leaflet-legend-HASH.js", options.outDir))
	js_files = append(js_files, utils.MustDownloadHash("https://unpkg.com/leaflet-gesture-handling@1.2.2/dist/leaflet-gesture-handling.min.js", "leaflet-gesture-handling-HASH.js", options.outDir))
	js_files = append(js_files, utils.MustCopyHash("static/parkrun-track.js", "parkrun-track-HASH.js", options.outDir))
	js_files = append(js_files, utils.MustCopyHash("static/main.js", "main-HASH.js", options.outDir))

	css_files := make([]string, 0)
	css_files = append(css_files, utils.MustDownloadHash("https://cdn.jsdelivr.net/npm/bulma@0.9.4/css/bulma.min.css", "bulma-HASH.css", options.outDir))
	css_files = append(css_files, utils.MustDownloadHash("https://unpkg.com/leaflet@1.9.4/dist/leaflet.css", "leaflet-HASH.css", options.outDir))
	css_files = append(css_files, utils.MustDownloadHash("https://raw.githubusercontent.com/ptma/Leaflet.Legend/master/src/leaflet.legend.css", "leaflet-legend-HASH.css", options.outDir))
	css_files = append(css_files, utils.MustDownloadHash("https://unpkg.com/leaflet-gesture-handling@1.2.2/dist/leaflet-gesture-handling.min.css", "leaflet-gesture-handling-HASH.css", options.outDir))
	css_files = append(css_files, utils.MustCopyHash("static/style.css", "style-HASH.css", options.outDir))

	utils.MustDownloadHash("https://unpkg.com/leaflet@1.9.4/dist/images/marker-icon.png", "images/marker-icon.png", options.outDir)
	utils.MustDownloadHash("https://unpkg.com/leaflet@1.9.4/dist/images/marker-icon-2x.png", "images/marker-icon-2x.png", options.outDir)
	utils.MustDownloadHash("https://unpkg.com/leaflet@1.9.4/dist/images/marker-shadow.png", "images/marker-shadow.png", options.outDir)

	breadcrumbsBase := utils.InitBreadcrumbs(utils.Link{Name: "freiburg.run", Url: "/"})
	breadcrumbsEvents := utils.PushBreadcrumb(breadcrumbsBase, utils.Link{Name: "Laufveranstaltungen", Url: "/"})

	defaultImage := "/images/preview.png"
	if err = utils.GenImage2(filepath.Join(options.outDir, "images/preview.png"), "Laufveranstaltungen", "im Raum Freiburg", "static/background.png"); err != nil {
		defaultImage = "/images/events2023.jpg"
		log.Printf("defaultimage: %v", err)
	}

	data := TemplateData{
		"Laufveranstaltungen im Raum Freiburg",
		"Veranstaltung",
		"Liste von aktuellen und zukünftigen Laufveranstaltungen, Lauf-Wettkämpfen, Volksläufen im Raum Freiburg",
		"events",
		"https://freiburg.run/",
		defaultImage,
		breadcrumbsEvents,
		timestamp,
		timestampFull,
		sheetUrl,
		events,
		events_old,
		groups,
		shops,
		parkrun,
		tagsList,
		seriesList,
		seriesListOld,
		js_files,
		css_files,
	}

	utils.ExecuteTemplate("events", filepath.Join(options.outDir, "index.html"), data)

	breadcrumbsEventsOld := utils.PushBreadcrumb(breadcrumbsEvents, utils.Link{Name: "Archiv", Url: "/events-old.html"})
	data.Title = "Vergangene Laufveranstaltungen im Raum Freiburg "
	data.Description = "Liste von vergangenen Laufveranstaltungen, Lauf-Wettkämpfen, Volksläufen im Raum Freiburg "
	data.Canonical = "https://freiburg.run/events-old.html"
	data.Breadcrumbs = breadcrumbsEventsOld
	utils.ExecuteTemplate("events-old", filepath.Join(options.outDir, "events-old.html"), data)

	breadcrumbsEventsTags := utils.PushBreadcrumb(breadcrumbsEvents, utils.Link{Name: "Kategorien", Url: "/tags.html"})
	data.Nav = "tags"
	data.Title = "Kategorien"
	data.Description = "Liste aller Kategorien von Laufveranstaltungen, Lauf-Wettkämpfen, Volksläufen im Raum Freiburg "
	data.Canonical = "https://freiburg.run/tags.html"
	data.Breadcrumbs = breadcrumbsEventsTags
	utils.ExecuteTemplate("tags", filepath.Join(options.outDir, "tags.html"), data)

	breadcrumbsGroups := utils.PushBreadcrumb(breadcrumbsBase, utils.Link{Name: "Lauftreffs", Url: "/lauftreffs.html"})
	data.Nav = "groups"
	data.Title = "Lauftreffs im Raum Freiburg "
	data.Type = "Lauftreff"
	data.Description = "Liste von Lauftreffs, Laufgruppen, Lauf-Trainingsgruppen im Raum Freiburg "
	data.Canonical = "https://freiburg.run/lauftreffs.html"
	data.Breadcrumbs = breadcrumbsGroups
	utils.ExecuteTemplate("groups", filepath.Join(options.outDir, "lauftreffs.html"), data)

	breadcrumbsShops := utils.PushBreadcrumb(breadcrumbsBase, utils.Link{Name: "Lauf-Shops", Url: "/shops.html"})
	data.Nav = "shops"
	data.Title = "Lauf-Shops im Raum Freiburg "
	data.Type = "Lauf-Shop"
	data.Description = "Liste von Lauf-Shops und Einzelhandelsgeschäften mit Laufschuh-Auswahl im Raum Freiburg "
	data.Canonical = "https://freiburg.run/shops.html"
	data.Breadcrumbs = breadcrumbsShops
	utils.ExecuteTemplate("shops", filepath.Join(options.outDir, "shops.html"), data)

	data.Nav = "parkrun"
	data.Title = "Dietenbach parkrun"
	data.Type = "Dietenbach parkrun"
	data.Image = "/images/parkrun.png"
	data.Description = "Vollständige Liste aller Ergebnisse, Laufberichte und Fotogalerien des 'Dietenbach parkrun' im Freiburger Dietenbachpark."
	data.Canonical = "https://freiburg.run/dietenbach-parkrun.html"
	data.Breadcrumbs = utils.PushBreadcrumb(breadcrumbsBase, utils.Link{Name: "Dietenbach parkrun", Url: "/dietenbach-parkrun.html"})
	utils.ExecuteTemplate("dietenbach-parkrun", filepath.Join(options.outDir, "dietenbach-parkrun.html"), data)

	breadcrumbsEventsSeries := utils.PushBreadcrumb(breadcrumbsEvents, utils.Link{Name: "Serien", Url: "/series.html"})
	data.Nav = "series"
	data.Title = "Serien"
	data.Description = "Liste aller Serien von Laufveranstaltungen, Lauf-Wettkämpfen, Volksläufen im Raum Freiburg "
	data.Canonical = "https://freiburg.run/series.html"
	data.Breadcrumbs = breadcrumbsEventsSeries
	utils.ExecuteTemplate("series", filepath.Join(options.outDir, "series.html"), data)

	data.Nav = "map"
	data.Title = "Karte aller Laufveranstaltungen"
	data.Type = "Karte"
	data.Image = defaultImage
	data.Description = "Karte"
	data.Canonical = "https://freiburg.run/map.html"
	data.Breadcrumbs = utils.PushBreadcrumb(breadcrumbsBase, utils.Link{Name: "Karte", Url: "/map.html"})
	utils.ExecuteTemplate("map", filepath.Join(options.outDir, "map.html"), data)

	breadcrumbsInfo := utils.PushBreadcrumb(breadcrumbsBase, utils.Link{Name: "Info", Url: "/info.html"})
	data.Nav = "datenschutz"
	data.Title = "Datenschutz"
	data.Type = "Datenschutz"
	data.Description = "Datenschutzerklärung von freiburg.run"
	data.Canonical = "https://freiburg.run/datenschutz.html"
	data.Breadcrumbs = utils.PushBreadcrumb(breadcrumbsInfo, utils.Link{Name: "Datenschutz", Url: "/datenschutz.html"})
	utils.ExecuteTemplate("datenschutz", filepath.Join(options.outDir, "datenschutz.html"), data)

	data.Nav = "impressum"
	data.Title = "Impressum"
	data.Type = "Impressum"
	data.Description = "Impressum von freiburg.run"
	data.Canonical = "https://freiburg.run/impressum.html"
	data.Breadcrumbs = utils.PushBreadcrumb(breadcrumbsInfo, utils.Link{Name: "Impressum", Url: "/impressum.html"})
	utils.ExecuteTemplate("impressum", filepath.Join(options.outDir, "impressum.html"), data)

	data.Nav = "info"
	data.Title = "Info"
	data.Type = "Info"
	data.Description = "Kontaktmöglichkeiten, allgemeine & technische Informationen über freiburg.run"
	data.Canonical = "https://freiburg.run/info.html"
	data.Breadcrumbs = breadcrumbsInfo
	utils.ExecuteTemplate("info", filepath.Join(options.outDir, "info.html"), data)

	data.Nav = "404"
	data.Title = "404 - Seite nicht gefunden :("
	data.Type = ""
	data.Description = "Fehlerseite von freiburg.run"
	data.Canonical = "https://freiburg.run/404.html"
	data.Breadcrumbs = utils.PushBreadcrumb(breadcrumbsBase, utils.Link{Name: "Fehlerseite", Url: "/404.html"})
	utils.ExecuteTemplate("404", filepath.Join(options.outDir, "404.html"), data)

	eventdata := EventTemplateData{
		nil,
		"",
		"Veranstaltung",
		"",
		"events",
		"",
		defaultImage,
		breadcrumbsEvents,
		"/",
		timestamp,
		timestampFull,
		sheetUrl,
		js_files,
		css_files,
	}
	for _, event := range events {
		if event.IsSeparator() {
			continue
		}
		eventdata.Event = event
		eventdata.Title = event.Name
		eventdata.Description = fmt.Sprintf("Informationen zu %s in %s am %s", event.Name, event.Location.NameNoFlag(), event.Time)
		slug := event.Slug()
		eventdata.Canonical = fmt.Sprintf("https://freiburg.run/%s", slug)
		image := event.ImageSlug()
		if utils.GenImage(filepath.Join(options.outDir, image), event.Name, event.Time, event.Location.NameNoFlag(), "static/background.png") == nil {
			eventdata.Image = fmt.Sprintf("/%s", image)
		} else {
			eventdata.Image = defaultImage
		}
		eventdata.Breadcrumbs = utils.PushBreadcrumb(breadcrumbsEvents, utils.Link{Name: event.Name, Url: fmt.Sprintf("/%s", slug)})
		utils.ExecuteTemplate("event", filepath.Join(options.outDir, slug), eventdata)
		sitemap.Add(slug, event.Name, "Laufveranstaltungen")
	}

	eventdata.Main = "/events-old.html"
	for _, event := range events_old {
		if event.IsSeparator() {
			continue
		}
		eventdata.Event = event
		eventdata.Title = event.Name
		eventdata.Description = fmt.Sprintf("Informationen zu %s in %s am %s", event.Name, event.Location.NameNoFlag(), event.Time)
		slug := event.Slug()
		eventdata.Canonical = fmt.Sprintf("https://freiburg.run/%s", slug)
		image := event.ImageSlug()
		if err = utils.GenImage(filepath.Join(options.outDir, image), event.Name, event.Time, event.Location.NameNoFlag(), "static/background.png"); err != nil {
			eventdata.Image = defaultImage
			log.Printf("event '%s': %v", event.Name, err)
		} else {
			eventdata.Image = image
		}
		eventdata.Breadcrumbs = utils.PushBreadcrumb(breadcrumbsEventsOld, utils.Link{Name: event.Name, Url: fmt.Sprintf("/%s", slug)})
		utils.ExecuteTemplate("event", filepath.Join(options.outDir, slug), eventdata)
		sitemap.Add(slug, event.Name, "Vergangene Laufveranstaltungen")
	}

	eventdata.Type = "Lauftreff"
	eventdata.Nav = "groups"
	eventdata.Main = "/lauftreffs.html"
	for _, event := range groups {
		eventdata.Event = event
		eventdata.Title = event.Name
		eventdata.Description = fmt.Sprintf("Informationen zu %s in %s am %s", event.Name, event.Location.NameNoFlag(), event.Time)
		slug := event.Slug()
		eventdata.Canonical = fmt.Sprintf("https://freiburg.run/%s", slug)
		image := event.ImageSlug()
		if err = utils.GenImage(filepath.Join(options.outDir, image), event.Name, event.Time, event.Location.NameNoFlag(), "static/background.png"); err != nil {
			eventdata.Image = defaultImage
			log.Printf("event '%s': %v", event.Name, err)
		} else {
			eventdata.Image = image
		}
		eventdata.Breadcrumbs = utils.PushBreadcrumb(breadcrumbsGroups, utils.Link{Name: event.Name, Url: fmt.Sprintf("/%s", slug)})
		utils.ExecuteTemplate("event", filepath.Join(options.outDir, slug), eventdata)
		sitemap.Add(slug, event.Name, "Lauftreffs")
	}

	eventdata.Type = "Lauf-Shop"
	eventdata.Nav = "shops"
	eventdata.Main = "/shops.html"
	for _, event := range shops {
		eventdata.Event = event
		eventdata.Title = event.Name
		eventdata.Description = fmt.Sprintf("Informationen zu %s in %s", event.Name, event.Location.NameNoFlag())
		slug := event.Slug()
		eventdata.Canonical = fmt.Sprintf("https://freiburg.run/%s", slug)
		image := event.ImageSlug()
		if err = utils.GenImage2(filepath.Join(options.outDir, image), event.Name, event.Location.NameNoFlag(), "static/background.png"); err != nil {
			eventdata.Image = defaultImage
			log.Printf("event '%s': %v", event.Name, err)
		} else {
			eventdata.Image = image
		}
		eventdata.Breadcrumbs = utils.PushBreadcrumb(breadcrumbsShops, utils.Link{Name: event.Name, Url: fmt.Sprintf("/%s", slug)})
		utils.ExecuteTemplate("event", filepath.Join(options.outDir, slug), eventdata)
		sitemap.Add(slug, event.Name, "Lauf-Shops")
	}

	tagdata := TagTemplateData{
		nil,
		"",
		"Kategorie",
		"",
		"tags",
		"",
		defaultImage,
		breadcrumbsEventsTags,
		"/tags.html",
		timestamp,
		timestampFull,
		sheetUrl,
		js_files,
		css_files,
	}
	for _, tag := range tags {
		tagdata.Tag = tag
		tagdata.Title = fmt.Sprintf("Laufveranstaltungen der Kategorie '%s'", tag.Name)
		tagdata.Description = fmt.Sprintf("Liste an Laufveranstaltungen im Raum Freiburg, die mit der Kategorie '%s' getaggt sind", tag.Name)
		slug := tag.Slug()
		tagdata.Canonical = fmt.Sprintf("https://freiburg.run/%s", slug)
		tagdata.Breadcrumbs = utils.PushBreadcrumb(breadcrumbsEventsTags, utils.Link{Name: tag.Name, Url: fmt.Sprintf("/%s", slug)})
		utils.ExecuteTemplate("tag", filepath.Join(options.outDir, slug), tagdata)
		sitemap.Add(slug, tag.Name, "Kategorien")
	}

	seriedata := SerieTemplateData{
		nil,
		"",
		"Serie",
		"",
		"series",
		"",
		defaultImage,
		breadcrumbsEventsSeries,
		"/series.html",
		timestamp,
		timestampFull,
		sheetUrl,
		js_files,
		css_files,
	}
	for _, s := range series {
		seriedata.Serie = s
		seriedata.Title = s.Name
		seriedata.Description = fmt.Sprintf("Lauf-Serie '%s'", s.Name)
		slug := s.Slug()
		seriedata.Canonical = fmt.Sprintf("https://freiburg.run/%s", slug)
		image := s.ImageSlug()
		if utils.GenImage(filepath.Join(options.outDir, image), s.Name, "", "", "static/background.png") == nil {
			seriedata.Image = fmt.Sprintf("/%s", image)
		} else {
			seriedata.Image = defaultImage
		}
		seriedata.Breadcrumbs = utils.PushBreadcrumb(breadcrumbsEventsSeries, utils.Link{Name: s.Name, Url: fmt.Sprintf("/%s", slug)})
		utils.ExecuteTemplate("serie", filepath.Join(options.outDir, slug), seriedata)
		sitemap.Add(slug, s.Name, "Serien")
	}

	sitemap.Gen(filepath.Join(options.outDir, "sitemap.xml"), options.hashFile, options.outDir)
	sitemapTemplate := SitemapTemplateData{
		"Sitemap von freiburg.run",
		"",
		"Sitemap von freiburg.run",
		"",
		"https://freiburg.run/sitemap.html",
		defaultImage,
		utils.PushBreadcrumb(breadcrumbsBase, utils.Link{Name: "Sitemap", Url: "/sitemap.html"}),
		timestamp,
		timestampFull,
		sheetUrl,
		sitemap.GenHTML(),
		js_files,
		css_files,
	}
	utils.ExecuteTemplate("sitemap", filepath.Join(options.outDir, "sitemap.html"), sitemapTemplate)

	err = CreateHtaccess(events, events_old, groups, shops, options.outDir)
	utils.Check(err)
}
