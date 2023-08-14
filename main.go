package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"text/template"
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
}

func parseCommandLine() CommandLineOptions {
	configFile := flag.String("config", "", "select config file")
	outDir := flag.String("out", ".out", "output directory")
	hashFile := flag.String("hashfile", ".hashes", "file storing file hashes (for sitemap)")

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
	Time      string
	TimeRange TimeRange
	Location  string
	Geo       string
	Details   string
	Details2  string
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
		TimeRange{},
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
	Name      string
	Events    []*Event
	EventsOld []*Event
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

func check(err error) {
	if err != nil {
		panic(err)
	}
}

func makeDir(dir string) {
	if err := os.MkdirAll(dir, 0770); err != nil {
		panic(err)
	}
}

func loadTemplate(name string) *template.Template {
	t, err := template.ParseFiles(fmt.Sprintf("templates/%s.html", name), "templates/header.html", "templates/footer.html", "templates/tail.html", "templates/card.html")
	check(err)
	return t
}

func executeTemplate(templateName string, fileName string, data TemplateData) {
	out, err := os.Create(fileName)
	check(err)
	defer out.Close()
	err = loadTemplate(templateName).Execute(out, data)
	check(err)
}

func executeEventTemplate(templateName string, fileName string, data EventTemplateData) {
	outDir := filepath.Dir(fileName)
	makeDir(outDir)
	out, err := os.Create(fileName)
	check(err)
	defer out.Close()
	err = loadTemplate(templateName).Execute(out, data)
	check(err)
}

func executeTagTemplate(templateName string, fileName string, data TagTemplateData) {
	outDir := filepath.Dir(fileName)
	makeDir(outDir)
	out, err := os.Create(fileName)
	check(err)
	defer out.Close()
	err = loadTemplate(templateName).Execute(out, data)
	check(err)
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

func parseLinks(ss []interface{}) []NameUrl {
	links := make([]NameUrl, 0)
	for _, i := range ss {
		s := fmt.Sprintf("%v", i)
		if s == "" {
			break
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

func fetchEvents(config ConfigData, srv *sheets.Service, eventType string, table string, now time.Time) []*Event {
	events := make([]*Event, 0)
	resp, err := srv.Spreadsheets.Values.Get(config.SheetId, fmt.Sprintf("%s!A2:Z", table)).Do()
	check(err)
	if len(resp.Values) == 0 {
		panic("No events data found.")
	} else {
		for _, row := range resp.Values {
			var added, date, name, url, description1, description2, location, coordinates string
			tags := make([]string, 0)
			links := make([]NameUrl, 0)

			ll := len(row)
			if ll > 0 {
				added = fmt.Sprintf("%v", row[0])
			}
			if ll > 1 {
				date = fmt.Sprintf("%v", row[1])
			}
			if ll > 2 {
				name = fmt.Sprintf("%v", row[2])
			}
			if ll > 3 {
				url = fmt.Sprintf("%v", row[3])
			}
			if ll > 4 {
				description1, description2 = SplitDetails(fmt.Sprintf("%v", row[4]))
			}
			if ll > 5 {
				location = fmt.Sprintf("%v", row[5])
			}
			if ll > 6 {
				coordinates = utils.NormalizeGeo(fmt.Sprintf("%v", row[6]))
			}
			if ll > 7 {
				tags = parseTags(fmt.Sprintf("%v", row[7]))
			}
			if ll > 8 {
				links = parseLinks(row[8:])
			}

			timeRange, err := parseTimeRange(date)
			if err != nil {
				log.Printf("event '%s': %v", name, err)
			}
			events = append(events, &Event{
				eventType,
				name,
				date,
				timeRange,
				location,
				coordinates,
				description1,
				description2,
				url,
				tags,
				links,
				added,
				IsNew(added, now),
				nil,
				nil,
			})
		}
	}

	return events
}

func fetchParkrunEvents(config ConfigData, srv *sheets.Service, table string, now time.Time) []*ParkrunEvent {
	events := make([]*ParkrunEvent, 0)
	resp, err := srv.Spreadsheets.Values.Get(config.SheetId, fmt.Sprintf("%s!A2:Z", table)).Do()
	check(err)
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
				currentWeek = now.After(d) && now.Before(d.AddDate(0, 0, 7))
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

func splitEvents(events []*Event, today time.Time) ([]*Event, []*Event) {
	futureEvents := make([]*Event, 0)
	pastEvents := make([]*Event, 0)

	for _, event := range events {
		if event.TimeRange.From.IsZero() {
			futureEvents = append(futureEvents, event)
		} else if event.TimeRange.To.Before(today) {
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

	for _, event := range events {
		d := event.TimeRange.From
		if event.TimeRange.From.IsZero() {
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

func collectTags(events []*Event, eventsOld []*Event) (map[string]*Tag, []*Tag) {
	tags := make(map[string]*Tag)
	for _, e := range events {
		for _, t := range e.Tags {
			if tag, found := tags[t]; found {
				tag.Events = append(tag.Events, e)
			} else {
				tag := &Tag{t, make([]*Event, 0), make([]*Event, 0)}
				tag.Events = append(tag.Events, e)
				tags[t] = tag
			}
		}
	}
	for _, e := range eventsOld {
		for _, t := range e.Tags {
			if tag, found := tags[t]; found {
				tag.EventsOld = append(tag.EventsOld, e)
			} else {
				tag := &Tag{t, make([]*Event, 0), make([]*Event, 0)}
				tag.EventsOld = append(tag.EventsOld, e)
				tags[t] = tag
			}
		}
	}

	tagsList := make([]*Tag, 0, len(tags))
	for _, tag := range tags {
		tagsList = append(tagsList, tag)
	}
	sort.Slice(tagsList, func(i, j int) bool { return tagsList[i].Name < tagsList[j].Name })

	return tags, tagsList
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
	check(err)
	var config ConfigData
	if err := json.Unmarshal(config_data, &config); err != nil {
		panic(err)
	}

	sheetUrl = fmt.Sprintf("https://docs.google.com/spreadsheets/d/%s", config.SheetId)

	ctx := context.Background()
	srv, err := sheets.NewService(ctx, option.WithAPIKey(config.ApiKey))
	check(err)

	events = fetchEvents(config, srv, "event", "Events", now)
	groups = fetchEvents(config, srv, "group", "Groups", now)
	shops = fetchEvents(config, srv, "shop", "Shops", now)
	parkrun = fetchParkrunEvents(config, srv, "Parkrun", now)

	validateDateOrder(events)
	findPrevNextEvents(events)
	events, events_old = splitEvents(events, today)
	events = addMonthSeparators(events)
	events_old = addMonthSeparators(events_old)
	tags, tagsList := collectTags(events, events_old)

	sitemapEntries := make([]string, 0)
	sitemapEntries = utils.AddSitemapEntry(sitemapEntries, "index.html")
	sitemapEntries = utils.AddSitemapEntry(sitemapEntries, "events-old.html")
	sitemapEntries = utils.AddSitemapEntry(sitemapEntries, "tags.html")
	sitemapEntries = utils.AddSitemapEntry(sitemapEntries, "lauftreffs.html")
	sitemapEntries = utils.AddSitemapEntry(sitemapEntries, "shops.html")
	sitemapEntries = utils.AddSitemapEntry(sitemapEntries, "dietenbach-parkrun.html")
	sitemapEntries = utils.AddSitemapEntry(sitemapEntries, "map.html")
	sitemapEntries = utils.AddSitemapEntry(sitemapEntries, "info.html")
	sitemapEntries = utils.AddSitemapEntry(sitemapEntries, "datenschutz.html")
	sitemapEntries = utils.AddSitemapEntry(sitemapEntries, "impressum.html")

	utils.MustCopyHash("static/.htaccess", ".htaccess", options.outDir)
	utils.MustCopyHash("static/robots.txt", "robots.txt", options.outDir)
	utils.MustCopyHash("static/favicon.png", "favicon.png", options.outDir)
	utils.MustCopyHash("static/favicon.ico", "favicon.ico", options.outDir)
	utils.MustCopyHash("static/apple-touch-icon.png", "apple-touch-icon.png", options.outDir)
	utils.MustCopyHash("static/freiburg-run.svg", "images/freiburg-run.svg", options.outDir)
	utils.MustCopyHash("static/events2023.jpg", "images/events2023.jpg", options.outDir)
	utils.MustCopyHash("static/marker-grey-icon.png", "images/marker-grey-icon.png", options.outDir)
	utils.MustCopyHash("static/marker-grey-icon-2x.png", "images/marker-grey-icon-2x.png", options.outDir)
	utils.MustCopyHash("static/circle-small.png", "images/circle-small.png", options.outDir)
	utils.MustCopyHash("static/circle-big.png", "images/circle-big.png", options.outDir)

	js_files := make([]string, 0)
	js_files = append(js_files, utils.MustDownloadHash("https://unpkg.com/leaflet@1.9.4/dist/leaflet.js", "leaflet-HASH.js", options.outDir))
	js_files = append(js_files, utils.MustDownloadHash("https://raw.githubusercontent.com/ptma/Leaflet.Legend/master/src/leaflet.legend.js", "leaflet-legend-HASH.js", options.outDir))
	js_files = append(js_files, utils.MustCopyHash("static/parkrun-track.js", "parkrun-track-HASH.js", options.outDir))
	js_files = append(js_files, utils.MustCopyHash("static/main.js", "main-HASH.js", options.outDir))

	css_files := make([]string, 0)
	css_files = append(css_files, utils.MustDownloadHash("https://cdn.jsdelivr.net/npm/bulma@0.9.4/css/bulma.min.css", "bulma-HASH.css", options.outDir))
	css_files = append(css_files, utils.MustDownloadHash("https://unpkg.com/leaflet@1.9.4/dist/leaflet.css", "leaflet-HASH.css", options.outDir))
	css_files = append(css_files, utils.MustDownloadHash("https://raw.githubusercontent.com/ptma/Leaflet.Legend/master/src/leaflet.legend.css", "leaflet-legend-HASH.css", options.outDir))
	css_files = append(css_files, utils.MustDownloadHash("https://raw.githubusercontent.com/justboil/bulma-responsive-tables/master/css/main.min.css", "bulma-responsive-tables-HASH.css", options.outDir))
	css_files = append(css_files, utils.MustCopyHash("static/style.css", "style-HASH.css", options.outDir))

	utils.MustDownloadHash("https://unpkg.com/leaflet@1.9.4/dist/images/marker-icon.png", "images/marker-icon.png", options.outDir)
	utils.MustDownloadHash("https://unpkg.com/leaflet@1.9.4/dist/images/marker-icon-2x.png", "images/marker-icon-2x.png", options.outDir)
	utils.MustDownloadHash("https://unpkg.com/leaflet@1.9.4/dist/images/marker-shadow.png", "images/marker-shadow.png", options.outDir)

	breadcrumbsBase := utils.InitBreadcrumbs(utils.Link{Name: "freiburg.run", Url: "/index.html"})
	breadcrumbsEvents := utils.PushBreadcrumb(breadcrumbsBase, utils.Link{Name: "Laufveranstaltungen", Url: "/index.html"})

	defaultImage := "/images/events2023.jpg"

	data := TemplateData{
		"Aktuelle und zukünftige Laufveranstaltungen im Raum Freiburg / Südbaden",
		"Veranstaltung",
		"Liste von aktuellen und zukünftigen Laufveranstaltungen, Lauf-Wettkämpfen, Volksläufen im Raum Freiburg / Südbaden",
		"events",
		"https://freiburg.run/index.html",
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

	executeTemplate("events", filepath.Join(options.outDir, "index.html"), data)

	breadcrumbsEventsOld := utils.PushBreadcrumb(breadcrumbsEvents, utils.Link{Name: "Archiv", Url: "/events-old.html"})
	data.Title = "Vergangene Laufveranstaltungen im Raum Freiburg / Südbaden"
	data.Description = "Liste von vergangenen Laufveranstaltungen, Lauf-Wettkämpfen, Volksläufen im Raum Freiburg / Südbaden"
	data.Canonical = "https://freiburg.run/events-old.html"
	data.Breadcrumbs = breadcrumbsEventsOld
	executeTemplate("events-old", filepath.Join(options.outDir, "events-old.html"), data)

	breadcrumbsEventsTags := utils.PushBreadcrumb(breadcrumbsEvents, utils.Link{Name: "Kategorien", Url: "/tags.html"})
	data.Title = "Kategorien"
	data.Description = "Liste aller Kategorien von vergangenen Laufveranstaltungen, Lauf-Wettkämpfen, Volksläufen im Raum Freiburg / Südbaden"
	data.Canonical = "https://freiburg.run/tags.html"
	data.Breadcrumbs = breadcrumbsEventsTags
	executeTemplate("tags", filepath.Join(options.outDir, "tags.html"), data)

	breadcrumbsGroups := utils.PushBreadcrumb(breadcrumbsBase, utils.Link{Name: "Lauftreffs", Url: "/lauftreffs.html"})
	data.Nav = "groups"
	data.Title = "Lauftreffs im Raum Freiburg / Südbaden"
	data.Type = "Lauftreff"
	data.Description = "Liste von Lauftreffs, Laufgruppen, Lauf-Trainingsgruppen im Raum Freiburg / Südbaden"
	data.Canonical = "https://freiburg.run/lauftreffs.html"
	data.Breadcrumbs = breadcrumbsGroups
	executeTemplate("groups", filepath.Join(options.outDir, "lauftreffs.html"), data)

	breadcrumbsShops := utils.PushBreadcrumb(breadcrumbsBase, utils.Link{Name: "Lauf-Shops", Url: "/shops.html"})
	data.Nav = "shops"
	data.Title = "Lauf-Shops im Raum Freiburg / Südbaden"
	data.Type = "Lauf-Shop"
	data.Description = "Liste von Lauf-Shops und Einzelhandelsgeschäften mit Laufschuh-Auswahl im Raum Freiburg / Südbaden"
	data.Canonical = "https://freiburg.run/shops.html"
	data.Breadcrumbs = breadcrumbsShops
	executeTemplate("shops", filepath.Join(options.outDir, "shops.html"), data)

	data.Nav = "parkrun"
	data.Title = "Dietenbach parkrun"
	data.Type = "Dietenbach parkrun"
	data.Description = "Vollständige Liste aller Ergebnisse, Laufberichte und Fotogalerien des 'Dietenbach parkrun' im Freiburger Dietenbachpark."
	data.Canonical = "https://freiburg.run/dietenbach-parkrun.html"
	data.Breadcrumbs = utils.PushBreadcrumb(breadcrumbsBase, utils.Link{Name: "Dietenbach parkrun", Url: "/dietenbach-parkrun.html"})
	executeTemplate("dietenbach-parkrun", filepath.Join(options.outDir, "dietenbach-parkrun.html"), data)

	data.Nav = "map"
	data.Title = "Karte aller Laufveranstaltunge"
	data.Type = "Karte"
	data.Description = "Karte"
	data.Canonical = "https://freiburg.run/map.html"
	data.Breadcrumbs = utils.PushBreadcrumb(breadcrumbsBase, utils.Link{Name: "Karte", Url: "/map.html"})
	executeTemplate("map", filepath.Join(options.outDir, "map.html"), data)

	breadcrumbsInfo := utils.PushBreadcrumb(breadcrumbsBase, utils.Link{Name: "Info", Url: "/info.html"})
	data.Nav = "datenschutz"
	data.Title = "Datenschutz"
	data.Type = "Datenschutz"
	data.Description = "Datenschutzerklärung von freiburg.run"
	data.Canonical = "https://freiburg.run/datenschutz.html"
	data.Breadcrumbs = utils.PushBreadcrumb(breadcrumbsInfo, utils.Link{Name: "Datenschutz", Url: "/datenschutz.html"})
	executeTemplate("datenschutz", filepath.Join(options.outDir, "datenschutz.html"), data)

	data.Nav = "impressum"
	data.Title = "Impressum"
	data.Type = "Impressum"
	data.Description = "Impressum von freiburg.run"
	data.Canonical = "https://freiburg.run/impressum.html"
	data.Breadcrumbs = utils.PushBreadcrumb(breadcrumbsInfo, utils.Link{Name: "Impressum", Url: "/impressum.html"})
	executeTemplate("impressum", filepath.Join(options.outDir, "impressum.html"), data)

	data.Nav = "info"
	data.Title = "Info"
	data.Type = "Info"
	data.Description = "Kontaktmöglichkeiten, allgemeine & technische Informationen über freiburg.run"
	data.Canonical = "https://freiburg.run/info.html"
	data.Breadcrumbs = breadcrumbsInfo
	executeTemplate("info", filepath.Join(options.outDir, "info.html"), data)

	data.Nav = "404"
	data.Title = "404 - Seite nicht gefunden :("
	data.Type = ""
	data.Description = "Fehlerseite von freiburg.run"
	data.Canonical = "https://freiburg.run/404.html"
	data.Breadcrumbs = utils.PushBreadcrumb(breadcrumbsBase, utils.Link{Name: "Fehlerseite", Url: "/404.html"})
	executeTemplate("404", filepath.Join(options.outDir, "404.html"), data)

	eventdata := EventTemplateData{
		nil,
		"",
		"Veranstaltung",
		"",
		"events",
		"",
		defaultImage,
		breadcrumbsEvents,
		"/index.html",
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
		executeEventTemplate("event", filepath.Join(options.outDir, slug), eventdata)
		sitemapEntries = utils.AddSitemapEntry(sitemapEntries, slug)
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
		executeEventTemplate("event", filepath.Join(options.outDir, slug), eventdata)
		sitemapEntries = utils.AddSitemapEntry(sitemapEntries, slug)
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
		executeEventTemplate("event", filepath.Join(options.outDir, slug), eventdata)
		sitemapEntries = utils.AddSitemapEntry(sitemapEntries, slug)
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
		executeEventTemplate("event", filepath.Join(options.outDir, slug), eventdata)
		sitemapEntries = utils.AddSitemapEntry(sitemapEntries, slug)
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
		tagdata.Description = fmt.Sprintf("Liste an Laufveranstaltungen im Raum Freiburg/Südbaden, die mit der Kategorie Kategorie '%s' getaggt sind", tag.Name)
		slug := fmt.Sprintf("tag/%s.html", tag.Name)
		tagdata.Canonical = fmt.Sprintf("https://freiburg.run/%s", slug)
		tagdata.Breadcrumbs = utils.PushBreadcrumb(breadcrumbsEventsTags, utils.Link{Name: tag.Name, Url: fmt.Sprintf("/%s", slug)})
		executeTagTemplate("tag", filepath.Join(options.outDir, slug), tagdata)
		sitemapEntries = utils.AddSitemapEntry(sitemapEntries, slug)
	}

	utils.GenSitemap(filepath.Join(options.outDir, "sitemap.xml"), options.hashFile, options.outDir, "https://freiburg.run", sitemapEntries)
}
