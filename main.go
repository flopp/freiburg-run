package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"
	"time"

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
	useJSON    bool
	exportCSV  bool
}

func parseCommandLine() CommandLineOptions {
	configFile := flag.String("config", "", "select config file")
	outDir := flag.String("out", ".out", "output directory")
	useJSON := flag.Bool("loadjson", false, "use JSON files")
	exportCSV := flag.Bool("exportcsv", false, "export CSV files")

	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), usage, os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse()

	if !*useJSON && *configFile == "" {
		panic("You have to specify a config file, e.g. -config myconfig.json, when not using JSON files")
	}

	return CommandLineOptions{
		*configFile,
		*outDir,
		*useJSON,
		*exportCSV,
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

type Event struct {
	Type     string
	Name     string
	Time     string
	Location string
	Geo      string
	Details  string
	Url      string
	Reports  []NameUrl
	Added    string
	New      bool
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

func (event *Event) Slug() string {
	if m := yearRe.FindStringSubmatch(event.Time); m != nil {
		return fmt.Sprintf("%s/%s-%s.html", event.Type, m[1], utils.SanitizeName(event.Name))
	}
	return fmt.Sprintf("%s/%s.html", event.Type, utils.SanitizeName(event.Name))
}

func (event *Event) LinkTitle() string {
	if event.Type == "event" {
		return "Zur Veranstaltung"
	}
	if event.Type == "group" {
		return "Zum Lauftreff"
	}
	if event.Type == "event" {
		return "Zum Lauf-Shop"
	}
	return "Zur Veranstaltung"
}

type ParkrunEvent struct {
	Index   string
	Date    string
	Special string
	Results string
	Report  string
	Photos  string
}

type TemplateData struct {
	Title         string
	Type          string
	Description   string
	Nav           string
	Canonical     string
	Timestamp     string
	TimestampFull string
	SheetUrl      string
	Events        []Event
	EventsOld     []Event
	Groups        []Event
	Shops         []Event
	Parkrun       []ParkrunEvent
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

func GetMtimeYMD(filePath string) string {
	stat, err := os.Stat(filePath)
	if err != nil {
		return ""
	}

	return stat.ModTime().Format("2006-01-02")
}

func writeCsv(fileName string, events []Event) {
	f, err := os.Create(fileName)
	check(err)
	defer f.Close()

	fmt.Fprintf(f, "\"%s\",\"%s\",\"%s\",\"%s\",\"%s\",\"%s\",\"%s\",\"%s\"\n", "ADDED", "DATE", "NAME", "URL", "DESCRIPTION", "LOCATION", "COORDINATES", "LINKS...")
	for _, event := range events {
		fmt.Fprintf(f, "\"%s\",\"%s\",\"%s\",\"%s\",\"%s\",\"%s\",\"%s\"", utils.DateYMS(event.Added), event.Time, event.Name, event.Url, event.Details, event.Location, event.Geo)
		for _, link := range event.Reports {
			fmt.Fprintf(f, ",\"%s|%s\"", link.Name, link.Url)
		}
		fmt.Fprintf(f, "\n")
	}
}

func writeParkrunCsv(fileName string, events []ParkrunEvent) {
	f, err := os.Create(fileName)
	check(err)
	defer f.Close()

	fmt.Fprintf(f, "\"%s\",\"%s\",\"%s\",\"%s\",\"%s\",\"%s\"\n", "INDEX", "DATE", "SPECIAL", "RESULTS", "REPORT", "PHOTOS")
	for _, event := range events {
		fmt.Fprintf(f, "\"%s\",\"%s\",\"%s\",\"%s\",\"%s\",\"%s\"\n", event.Index, event.Date, event.Special, event.Results, event.Report, event.Photos)
	}
}

type ConfigData struct {
	ApiKey  string `json:"api_key"`
	SheetId string `json:"sheet_id"`
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

func fetchEventsJson(eventType string, fileName string, now time.Time) ([]Event, string) {
	data, err := os.ReadFile(fileName)
	check(err)
	unmarshalled := make([]EventJson, 0)
	if err := json.Unmarshal(data, &unmarshalled); err != nil {
		panic(err)
	}
	mtime := GetMtimeYMD(fileName)

	events := make([]Event, 0)
	for _, e := range unmarshalled {
		ed := Event{
			eventType, e.Name, e.Time, e.Location, utils.NormalizeGeo(e.Geo), e.Details, e.Url, e.Reports, e.Added, IsNew(e.Added, now),
		}
		events = append(events, ed)
	}

	return events, mtime
}

func fetchParkrunEventsJson(fileName string) ([]ParkrunEvent, string) {
	data, err := os.ReadFile(fileName)
	check(err)
	unmarshalled := make([]ParkrunEvent, 0)
	if err := json.Unmarshal(data, &unmarshalled); err != nil {
		panic(err)
	}
	mtime := GetMtimeYMD(fileName)

	return unmarshalled, mtime
}

func fetchEvents(config ConfigData, srv *sheets.Service, eventType string, table string, now time.Time) ([]Event, string) {
	events := make([]Event, 0)
	mtime := ""
	resp, err := srv.Spreadsheets.Values.Get(config.SheetId, fmt.Sprintf("%s!A2:Z", table)).Do()
	check(err)
	if len(resp.Values) == 0 {
		panic("No events data found.")
	} else {
		for _, row := range resp.Values {
			var added, date, name, url, description, location, coordinates string
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
				description = fmt.Sprintf("%v", row[4])
			}
			if ll > 5 {
				location = fmt.Sprintf("%v", row[5])
			}
			if ll > 6 {
				coordinates = utils.NormalizeGeo(fmt.Sprintf("%v", row[6]))
			}
			if ll > 7 {
				links = parseLinks(row[7:])
			}
			events = append(events, Event{
				eventType,
				name,
				date,
				location,
				coordinates,
				description,
				url,
				links,
				added,
				IsNew(added, now),
			})
			t := utils.DateYMS(added)
			if mtime == "" || (t != "" && t > mtime) {
				mtime = t
			}
		}
	}

	return events, mtime
}

func fetchParkrunEvents(config ConfigData, srv *sheets.Service, table string) ([]ParkrunEvent, string) {
	events := make([]ParkrunEvent, 0)
	mtime := ""

	resp, err := srv.Spreadsheets.Values.Get(config.SheetId, fmt.Sprintf("%s!A1", table)).Do()
	check(err)
	if len(resp.Values) != 0 {
		mtime = utils.DateYMS(fmt.Sprintf("%v", resp.Values[0][0]))
	}

	resp, err = srv.Spreadsheets.Values.Get(config.SheetId, fmt.Sprintf("%s!A3:Z", table)).Do()
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
			events = append(events, ParkrunEvent{
				index,
				date,
				special,
				results,
				report,
				photos,
			})
		}
	}

	return events, mtime
}

func main() {
	now := time.Now()
	timestamp := now.Format("2006-01-02")
	timestampFull := now.Format("2006-01-02 15:04:05")
	sheetUrl := ""
	options := parseCommandLine()

	var events []Event
	var events_old []Event
	var events_time string

	var groups []Event
	var groups_time string

	var shops []Event
	var shops_time string

	var parkrun []ParkrunEvent
	var parkrun_time string

	if options.useJSON {
		var all_events []Event
		all_events, events_time = fetchEventsJson("event", "data/events.json", now)
		for _, e := range all_events {
			if !strings.Contains(e.Time, "UNBEKANNT") {
				events = append(events, e)
			}
		}

		groups, groups_time = fetchEventsJson("group", "data/groups.json", now)
		shops, shops_time = fetchEventsJson("shop", "data/shops.json", now)
		parkrun, parkrun_time = fetchParkrunEventsJson("data/dietenbach-parkrun.json")
	} else {
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

		events, events_time = fetchEvents(config, srv, "event", "Events", now)
		groups, groups_time = fetchEvents(config, srv, "group", "Groups", now)
		shops, shops_time = fetchEvents(config, srv, "shop", "Shops", now)
		parkrun, parkrun_time = fetchParkrunEvents(config, srv, "Parkrun")

		if events_time == "" {
			events_time = timestamp
		}
		if groups_time == "" {
			groups_time = timestamp
		}
		if shops_time == "" {
			shops_time = timestamp
		}
		if parkrun_time == "" {
			parkrun_time = timestamp
		}
	}

	events_tmp := make([]Event, 0)
	dateRe := regexp.MustCompile(`\b(\d\d\.\d\d\.\d\d\d\d)\b`)
	for _, event := range events {
		hasDate := false
		hasFutureDate := false
		m := dateRe.FindAllStringSubmatch(event.Time, -1)
		for _, mm := range m {
			d, err := time.Parse("02.01.2006 15:04:05", fmt.Sprintf("%s 23:59:59", mm[1]))
			if err != nil {
				continue
			}
			hasDate = true
			if d.After(now) {
				hasFutureDate = true
			}
		}
		if !hasDate || hasFutureDate {
			events_tmp = append(events_tmp, event)
		} else {
			events_old = append(events_old, event)
		}
	}
	events = events_tmp

	if options.exportCSV {
		writeCsv("events.csv", events)
		writeCsv("groups.csv", groups)
		writeCsv("shops.csv", shops)
		writeParkrunCsv("parkrun.csv", parkrun)
	}

	sitemapEntries := make([]utils.SitemapEntry, 0)
	sitemapEntries = utils.AddSitemapEntry(sitemapEntries, "index.html", events_time)
	sitemapEntries = utils.AddSitemapEntry(sitemapEntries, "events-old.html", events_time)
	sitemapEntries = utils.AddSitemapEntry(sitemapEntries, "lauftreffs.html", groups_time)
	sitemapEntries = utils.AddSitemapEntry(sitemapEntries, "shops.html", shops_time)
	sitemapEntries = utils.AddSitemapEntry(sitemapEntries, "dietenbach-parkrun.html", parkrun_time)
	sitemapEntries = utils.AddSitemapEntry(sitemapEntries, "map.html", events_time)
	sitemapEntries = utils.AddSitemapEntry(sitemapEntries, "info.html", GetMtimeYMD("templates/info.html"))
	sitemapEntries = utils.AddSitemapEntry(sitemapEntries, "datenschutz.html", GetMtimeYMD("templates/datenschutz.html"))
	sitemapEntries = utils.AddSitemapEntry(sitemapEntries, "impressum.html", GetMtimeYMD("templates/impressum.html"))

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

	data := TemplateData{
		"Aktuelle und zukünftige Laufveranstaltungen im Raum Freiburg / Südbaden",
		"Veranstaltung",
		"Liste von aktuellen und zukünftigen Laufveranstaltungen, Lauf-Wettkämpfen, Volksläufen im Raum Freiburg / Südbaden",
		"events",
		"https://freiburg.run/",
		timestamp,
		timestampFull,
		sheetUrl,
		events,
		events_old,
		groups,
		shops,
		parkrun,
		js_files,
		css_files,
	}

	executeTemplate("events", filepath.Join(options.outDir, "index.html"), data)
	executeTemplate("events-old", filepath.Join(options.outDir, "events-old.html"), data)

	data.Nav = "groups"
	data.Title = "Lauftreffs im Raum Freiburg / Südbaden"
	data.Type = "Lauftreff"
	data.Description = "Liste von Lauftreffs, Laufgruppen, Lauf-Trainingsgruppen im Raum Freiburg / Südbaden"
	data.Canonical = "https://freiburg.run/lauftreffs.html"
	executeTemplate("groups", filepath.Join(options.outDir, "lauftreffs.html"), data)

	data.Nav = "shops"
	data.Title = "Lauf-Shops im Raum Freiburg / Südbaden"
	data.Type = "Lauf-Shop"
	data.Description = "Liste von Lauf-Shops und Einzelhandelsgeschäften mit Laufschuh-Auswahl im Raum Freiburg / Südbaden"
	data.Canonical = "https://freiburg.run/shops.html"
	executeTemplate("shops", filepath.Join(options.outDir, "shops.html"), data)

	data.Nav = "parkrun"
	data.Title = "Dietenbach parkrun"
	data.Type = "Dietenbach parkrun"
	data.Description = "Vollständige Liste aller Ergebnisse, Laufberichte und Fotogalerien des 'Dietenbach parkrun' im Freiburger Dietenbachpark."
	data.Canonical = "https://freiburg.run/dietenbach-parkrun.html"
	executeTemplate("dietenbach-parkrun", filepath.Join(options.outDir, "dietenbach-parkrun.html"), data)

	data.Nav = "map"
	data.Title = "Karte aller Laufveranstaltunge"
	data.Type = "Karte"
	data.Description = "Karte"
	data.Canonical = "https://freiburg.run/map.html"
	executeTemplate("map", filepath.Join(options.outDir, "map.html"), data)

	data.Nav = "datenschutz"
	data.Title = "Datenschutz"
	data.Type = "Datenschutz"
	data.Description = "Datenschutzerklärung von freiburg.run"
	data.Canonical = "https://freiburg.run/datenschutz.html"
	executeTemplate("datenschutz", filepath.Join(options.outDir, "datenschutz.html"), data)

	data.Nav = "impressum"
	data.Title = "Impressum"
	data.Type = "Impressum"
	data.Description = "Impressum von freiburg.run"
	data.Canonical = "https://freiburg.run/impressum.html"
	executeTemplate("impressum", filepath.Join(options.outDir, "impressum.html"), data)

	data.Nav = "info"
	data.Title = "Info"
	data.Type = "Info"
	data.Description = "Kontaktmöglichkeiten, allgemeine & technische Informationen über freiburg.run"
	data.Canonical = "https://freiburg.run/info.html"
	executeTemplate("info", filepath.Join(options.outDir, "info.html"), data)

	data.Nav = "404"
	data.Title = "404 - Seite nicht gefunden :("
	data.Type = ""
	data.Description = "Fehlerseite von freiburg.run"
	data.Canonical = "https://freiburg.run/404.html"
	executeTemplate("404", filepath.Join(options.outDir, "404.html"), data)

	eventdata := EventTemplateData{
		nil,
		"",
		"Veranstaltung",
		"",
		"events",
		"",
		"/index.html",
		timestamp,
		timestampFull,
		sheetUrl,
		js_files,
		css_files,
	}
	for _, event := range events {
		eventdata.Event = &event
		eventdata.Title = event.Name
		eventdata.Description = fmt.Sprintf("Informationen zu %s in %s am %s", event.Name, event.Location, event.Time)
		slug := event.Slug()
		eventdata.Canonical = fmt.Sprintf("https://freiburg.run/%s", slug)
		executeEventTemplate("event", filepath.Join(options.outDir, slug), eventdata)
		t := utils.DateYMS(event.Added)
		if t == "" {
			t = events_time
		}
		sitemapEntries = utils.AddSitemapEntry(sitemapEntries, slug, t)
	}

	eventdata.Main = "/events-old.html"
	for _, event := range events_old {
		eventdata.Event = &event
		eventdata.Title = event.Name
		eventdata.Description = fmt.Sprintf("Informationen zu %s in %s am %s", event.Name, event.Location, event.Time)
		slug := event.Slug()
		eventdata.Canonical = fmt.Sprintf("https://freiburg.run/%s", slug)
		executeEventTemplate("event", filepath.Join(options.outDir, slug), eventdata)
		t := utils.DateYMS(event.Added)
		if t == "" {
			t = events_time
		}
		sitemapEntries = utils.AddSitemapEntry(sitemapEntries, slug, t)
	}

	eventdata.Type = "Lauftreff"
	eventdata.Nav = "groups"
	eventdata.Main = "/lauftreffs.html"
	for _, event := range groups {
		eventdata.Event = &event
		eventdata.Title = event.Name
		eventdata.Description = fmt.Sprintf("Informationen zu %s in %s am %s", event.Name, event.Location, event.Time)
		slug := event.Slug()
		eventdata.Canonical = fmt.Sprintf("https://freiburg.run/%s", slug)
		executeEventTemplate("event", filepath.Join(options.outDir, slug), eventdata)
		t := utils.DateYMS(event.Added)
		if t == "" {
			t = groups_time
		}
		sitemapEntries = utils.AddSitemapEntry(sitemapEntries, slug, t)
	}

	eventdata.Type = "Lauf-Shop"
	eventdata.Nav = "shops"
	eventdata.Main = "/shops.html"
	for _, event := range shops {
		eventdata.Event = &event
		eventdata.Title = event.Name
		eventdata.Description = fmt.Sprintf("Informationen zu %s in %s", event.Name, event.Location)
		slug := event.Slug()
		eventdata.Canonical = fmt.Sprintf("https://freiburg.run/%s", slug)
		executeEventTemplate("event", filepath.Join(options.outDir, slug), eventdata)
		t := utils.DateYMS(event.Added)
		if t == "" {
			t = shops_time
		}
		sitemapEntries = utils.AddSitemapEntry(sitemapEntries, slug, t)
	}

	utils.GenSitemap(filepath.Join(options.outDir, "sitemap.xml"), "https://freiburg.run", sitemapEntries)
}
