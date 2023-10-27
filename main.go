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

type EventJson struct {
	Name     string
	Time     string
	Location string
	Geo      string
	Details  string
	Url      string
	Reports  []NameUrl
	Added    string
}

type TimeRange struct {
	From time.Time
	To   time.Time
}

func parseTimeRange(s string) (TimeRange, error) {
	dateRe := regexp.MustCompile(`\b(\d\d\.\d\d\.\d\d\d\d)\b`)

	var from, to time.Time
	for _, mm := range dateRe.FindAllStringSubmatch(s, -1) {
		d, err := time.Parse("02.01.2006", mm[1])
		if err != nil {
			return TimeRange{}, fmt.Errorf("cannot parse date '%s' from '%s'", mm[1], s)
		}
		if from.IsZero() {
			from = d
		} else {
			if d.Before(to) {
				return TimeRange{}, fmt.Errorf("invalid time range '%s' (wrongly ordered components)", s)
			}
		}
		to = d
	}

	return TimeRange{from, to}, nil
}

type Event struct {
	Type      string
	Name      string
	NameOld   string
	Time      string
	TimeRange TimeRange
	Old       bool
	Cancelled bool
	Special   bool
	Location  string
	Geo       string
	Distance  string
	Direction string
	Details   string
	Details2  template.HTML
	Url       string
	Tags      []string
	Reports   []NameUrl
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
		TimeRange{},
		false,
		false,
		false,
		"",
		"",
		"",
		"",
		"",
		"",
		"",
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
	if strings.Contains(event.Name, "parkrun") {
		t = "event"
	}

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

type ParkrunEvent struct {
	IsCurrentWeek bool
	Index         string
	Date          string
	Special       string
	Results       string
	Report        string
	Photos        string
}

type Tag struct {
	Name        string
	Description string
	Events      []*Event
	EventsOld   []*Event
	Groups      []*Event
	Shops       []*Event
}

func CreateTag(name string) *Tag {
	return &Tag{name, "", make([]*Event, 0), make([]*Event, 0), make([]*Event, 0), make([]*Event, 0)}
}

func (tag *Tag) Slug() string {
	return fmt.Sprintf("/tag/%s.html", tag.Name)
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

func parseTags(s string) []string {
	tags := make([]string, 0)
	for _, tag := range strings.Split(s, ",") {
		tag = utils.SanitizeName(tag)
		if len(tag) > 0 {
			tags = append(tags, tag)
		}
	}
	return tags
}

func parseLinks(ss []string) []NameUrl {
	links := make([]NameUrl, 0)
	for _, s := range ss {
		if s == "" {
			continue
		}
		a := strings.Split(s, "|")
		if len(a) != 2 {
			panic(fmt.Errorf("bad link: <%s>", s))
		}
		links = append(links, NameUrl{a[0], a[1]})
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

func fetchEvents(config ConfigData, srv *sheets.Service, today time.Time, eventType string, table string) []*Event {
	events := make([]*Event, 0)
	resp, err := srv.Spreadsheets.Values.Get(config.SheetId, fmt.Sprintf("%s!A1:Z", table)).Do()
	utils.Check(err)
	if len(resp.Values) == 0 {
		panic("No events data found.")
	} else {
		cols := Columns{}
		for line, row := range resp.Values {
			if line == 0 {
				cols, err = initColumns(row)
				if err != nil {
					panic(fmt.Errorf("when fetching table '%s': %v", table, err))
				}
				continue
			}
			dateS := cols.getValue("DATE", row)
			nameS := cols.getValue("NAME", row)
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
			tagsS := cols.getValue("TAGS", row)
			linksS := make([]string, 4)
			linksS[0] = cols.getValue("LINK1", row)
			linksS[1] = cols.getValue("LINK2", row)
			linksS[2] = cols.getValue("LINK3", row)
			linksS[3] = cols.getValue("LINK4", row)

			date := dateS
			name, nameOld := SplitDetails(nameS)
			url := urlS
			description1, description2 := SplitDetails(descriptionS)
			location := locationS
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
			tags := parseTags(tagsS)
			links := parseLinks(linksS)

			timeRange, err := parseTimeRange(date)
			if err != nil {
				log.Printf("event '%s': %v", name, err)
			}
			isOld := (!timeRange.From.IsZero()) && timeRange.To.Before(today)
			if !timeRange.From.IsZero() {
				tags = append(tags, fmt.Sprintf("%d", timeRange.From.Year()))
			}

			events = append(events, &Event{
				eventType,
				name,
				nameOld,
				date,
				timeRange,
				isOld,
				strings.Contains(strings.ToLower(date), "abgesagt"),
				name == "100. Dietenbach parkrun",
				location,
				coordinates,
				distance,
				direction,
				description1,
				template.HTML(description2),
				url,
				tags,
				links,
				"",
				false,
				nil,
				nil,
			})
		}
	}

	return events
}

func fetchParkrunEvents(config ConfigData, srv *sheets.Service, today time.Time, table string) []*ParkrunEvent {
	events := make([]*ParkrunEvent, 0)
	resp, err := srv.Spreadsheets.Values.Get(config.SheetId, fmt.Sprintf("%s!A2:Z", table)).Do()
	utils.Check(err)
	if len(resp.Values) == 0 {
		panic("No events data found.")
	} else {
		for _, row := range resp.Values {
			var index, date, special, results, report, photos string

			ll := len(row)
			if ll > 0 {
				index = fmt.Sprintf("%v", row[0])
			}
			if ll > 1 {
				date = fmt.Sprintf("%v", row[1])
			}
			if ll > 2 {
				special = fmt.Sprintf("%v", row[2])
			}
			if ll > 3 {
				results = fmt.Sprintf("%v", row[3])
			}
			if ll > 4 {
				report = fmt.Sprintf("%v", row[4])
			}
			if ll > 5 {
				photos = fmt.Sprintf("%v", row[5])
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
				special,
				results,
				report,
				photos,
			})
		}
	}

	return events
}

func fetchTagDescriptions(config ConfigData, srv *sheets.Service, table string) map[string]string {
	descriptions := make(map[string]string)
	resp, err := srv.Spreadsheets.Values.Get(config.SheetId, fmt.Sprintf("%s!A1:B", table)).Do()
	utils.Check(err)
	if len(resp.Values) == 0 {
		panic("No tags data found.")
	} else {
		for _, row := range resp.Values {
			if len(row) >= 2 {
				name := utils.SanitizeName(fmt.Sprintf("%v", row[0]))
				desc := fmt.Sprintf("%v", row[1])
				if name != "" && desc != "" {
					descriptions[name] = desc
				}
			}
		}
	}

	return descriptions
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

			if isSimilarName(event2.Name, event.Name) && event2.Geo == event.Geo {
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

func collectTags(descriptions map[string]string, events []*Event, eventsOld []*Event, groups []*Event, shops []*Event) (map[string]*Tag, []*Tag) {
	tags := make(map[string]*Tag)
	for _, e := range events {
		for _, t := range e.Tags {
			tag := getTag(tags, t)
			tag.Events = append(tag.Events, e)
		}
	}
	for _, e := range eventsOld {
		for _, t := range e.Tags {
			tag := getTag(tags, t)
			tag.EventsOld = append(tag.EventsOld, e)
		}
	}
	for _, e := range groups {
		for _, t := range e.Tags {
			tag := getTag(tags, t)
			tag.Groups = append(tag.Groups, e)
		}
	}
	for _, e := range shops {
		for _, t := range e.Tags {
			tag := getTag(tags, t)
			tag.Shops = append(tag.Shops, e)
		}
	}

	tagsList := make([]*Tag, 0, len(tags))
	for _, tag := range tags {
		desc, found := descriptions[tag.Name]
		if found {
			tag.Description = desc
		}
		tagsList = append(tagsList, tag)
	}
	sort.Slice(tagsList, func(i, j int) bool { return tagsList[i].Name < tagsList[j].Name })

	return tags, tagsList
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

	events = make([]*Event, 0)
	for _, sheet := range eventSheets {
		events = append(events, fetchEvents(config, srv, today, "event", sheet)...)
	}
	groups = fetchEvents(config, srv, today, "group", groupsSheet)
	shops = fetchEvents(config, srv, today, "shop", shopsSheet)
	parkrun = fetchParkrunEvents(config, srv, today, parkrunSheet)
	tagDescriptions := fetchTagDescriptions(config, srv, tagsSheet)

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

	sitemap := utils.CreateSitemap("https://freiburg.run")
	sitemap.AddCategory("Allgemein")
	sitemap.AddCategory("Laufveranstaltungen")
	sitemap.AddCategory("Vergangene Laufveranstaltungen")
	sitemap.AddCategory("Kategorien")
	sitemap.AddCategory("Lauftreffs")
	sitemap.AddCategory("Lauf-Shops")
	sitemap.Add("", "Alle Laufveranstaltungen", "Laufveranstaltungen")
	sitemap.Add("events-old.html", "Alle vergangenen Laufveranstaltungen", "Vergangene Laufveranstaltungen")
	sitemap.Add("tags.html", "Alle Kategorieren", "Kategorien")
	sitemap.Add("lauftreffs.html", "Alle Lauftreffes", "Lauftreffs")
	sitemap.Add("shops.html", "Alle Lauf-Shops", "Lauf-Shops")
	sitemap.Add("dietenbach-parkrun.html", "Dietenbach parkrun", "Allgemein")
	sitemap.Add("map.html", "Karte", "Laufveranstaltungen")
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
	js_files = append(js_files, utils.MustCopyHash("static/parkrun-track.js", "parkrun-track-HASH.js", options.outDir))
	js_files = append(js_files, utils.MustCopyHash("static/main.js", "main-HASH.js", options.outDir))

	css_files := make([]string, 0)
	css_files = append(css_files, utils.MustDownloadHash("https://cdn.jsdelivr.net/npm/bulma@0.9.4/css/bulma.min.css", "bulma-HASH.css", options.outDir))
	css_files = append(css_files, utils.MustDownloadHash("https://unpkg.com/leaflet@1.9.4/dist/leaflet.css", "leaflet-HASH.css", options.outDir))
	css_files = append(css_files, utils.MustDownloadHash("https://raw.githubusercontent.com/ptma/Leaflet.Legend/master/src/leaflet.legend.css", "leaflet-legend-HASH.css", options.outDir))
	css_files = append(css_files, utils.MustCopyHash("static/style.css", "style-HASH.css", options.outDir))

	utils.MustDownloadHash("https://unpkg.com/leaflet@1.9.4/dist/images/marker-icon.png", "images/marker-icon.png", options.outDir)
	utils.MustDownloadHash("https://unpkg.com/leaflet@1.9.4/dist/images/marker-icon-2x.png", "images/marker-icon-2x.png", options.outDir)
	utils.MustDownloadHash("https://unpkg.com/leaflet@1.9.4/dist/images/marker-shadow.png", "images/marker-shadow.png", options.outDir)

	breadcrumbsBase := utils.InitBreadcrumbs(utils.Link{Name: "freiburg.run", Url: "/"})
	breadcrumbsEvents := utils.PushBreadcrumb(breadcrumbsBase, utils.Link{Name: "Laufveranstaltungen", Url: "/"})

	defaultImage := "/images/events2023.jpg"

	data := TemplateData{
		"Aktuelle und zukünftige Laufveranstaltungen im Raum Freiburg",
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
	data.Title = "Kategorien"
	data.Description = "Liste aller Kategorien von vergangenen Laufveranstaltungen, Lauf-Wettkämpfen, Volksläufen im Raum Freiburg "
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

	data.Nav = "map"
	data.Title = "Karte aller Laufveranstaltunge"
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
		eventdata.Description = fmt.Sprintf("Informationen zu %s in %s am %s", event.Name, event.Location, event.Time)
		slug := event.Slug()
		eventdata.Canonical = fmt.Sprintf("https://freiburg.run/%s", slug)
		image := event.ImageSlug()
		if utils.GenImage(filepath.Join(options.outDir, image), event.Name, event.Time, event.Location) == nil {
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
		eventdata.Description = fmt.Sprintf("Informationen zu %s in %s am %s", event.Name, event.Location, event.Time)
		slug := event.Slug()
		eventdata.Canonical = fmt.Sprintf("https://freiburg.run/%s", slug)
		image := event.ImageSlug()
		if err = utils.GenImage(filepath.Join(options.outDir, image), event.Name, event.Time, event.Location); err != nil {
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
		if strings.Contains(event.Name, "parkrun") {
			continue
		}
		eventdata.Event = event
		eventdata.Title = event.Name
		eventdata.Description = fmt.Sprintf("Informationen zu %s in %s am %s", event.Name, event.Location, event.Time)
		slug := event.Slug()
		eventdata.Canonical = fmt.Sprintf("https://freiburg.run/%s", slug)
		image := event.ImageSlug()
		if err = utils.GenImage(filepath.Join(options.outDir, image), event.Name, event.Time, event.Location); err != nil {
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
		eventdata.Description = fmt.Sprintf("Informationen zu %s in %s", event.Name, event.Location)
		slug := event.Slug()
		eventdata.Canonical = fmt.Sprintf("https://freiburg.run/%s", slug)
		image := event.ImageSlug()
		if err = utils.GenImage2(filepath.Join(options.outDir, image), event.Name, event.Location); err != nil {
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
		"events",
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
		slug := fmt.Sprintf("tag/%s.html", tag.Name)
		tagdata.Canonical = fmt.Sprintf("https://freiburg.run/%s", slug)
		tagdata.Breadcrumbs = utils.PushBreadcrumb(breadcrumbsEventsTags, utils.Link{Name: tag.Name, Url: fmt.Sprintf("/%s", slug)})
		utils.ExecuteTemplate("tag", filepath.Join(options.outDir, slug), tagdata)
		sitemap.Add(slug, tag.Name, "Kategorien")
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
