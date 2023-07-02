package main

import (
	"context"
	"crypto/sha256"
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"
	"time"

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
	Name     string
	Time     string
	Location string
	Geo      string
	Details  string
	Url      string
	Reports  []NameUrl
	Added    string
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
	SheetUrl      string
	Events        []Event
	EventsPending []Event
	Groups        []Event
	Shops         []Event
	Parkrun       []ParkrunEvent
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

func copyHash(src, dst, outDir string) string {
	dir := filepath.Join(outDir, filepath.Dir(dst))
	makeDir(dir)

	hash := computeHash(src)

	sourceFileStat, err := os.Stat(src)
	check(err)

	if !sourceFileStat.Mode().IsRegular() {
		panic(fmt.Errorf("%s is not a regular file", src))
	}

	source, err := os.Open(src)
	check(err)
	defer source.Close()

	dstHash := strings.Replace(dst, "HASH", hash, -1)
	dstHash2 := filepath.Join(outDir, dstHash)
	destination, err := os.Create(dstHash2)
	check(err)
	defer destination.Close()
	_, err = io.Copy(destination, source)
	check(err)

	return dstHash
}

func download(url string, dst string) {
	makeDir(filepath.Dir(dst))

	out, err := os.Create(dst)
	check(err)
	defer out.Close()

	// temporarily skip insecure certificates
	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	resp, err := http.Get(url)
	check(err)
	defer resp.Body.Close()

	_, err = io.Copy(out, resp.Body)
	check(err)
}

func downloadHash(url string, dst, outDir string) string {
	if strings.Contains(dst, "HASH") {
		tmpfile, err := os.CreateTemp("", "")
		check(err)
		defer os.Remove(tmpfile.Name())

		download(url, tmpfile.Name())

		return copyHash(tmpfile.Name(), dst, outDir)
	} else {
		dst2 := filepath.Join(outDir, dst)

		download(url, dst2)

		return dst
	}
}

func computeHash(fileName string) string {
	f, err := os.Open(fileName)
	check(err)
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		panic(err)
	}

	return fmt.Sprintf("%.8x", h.Sum(nil))
}

func loadTemplate(name string) *template.Template {
	t, err := template.ParseFiles(fmt.Sprintf("templates/%s.html", name), "templates/header.html", "templates/footer.html")
	check(err)
	return t
}

func executeTemplate(templateName string, fileName string, data TemplateData) {
	out, err := os.Create(fileName)
	check(err)
	defer out.Close()
	if err := loadTemplate(templateName).Execute(out, data); err != nil {
		panic(err)
	}
}

func nl(f *os.File) {
	f.WriteString("\n")
}
func genSitemapEntry(f *os.File, url string, timeStamp string) {
	f.WriteString(`    <url>`)
	nl(f)
	f.WriteString(fmt.Sprintf(`        <loc>%s</loc>`, url))
	nl(f)
	f.WriteString(fmt.Sprintf(`        <lastmod>%s</lastmod>`, timeStamp))
	nl(f)
	f.WriteString(`    </url>`)
	nl(f)
}

func genSitemap(fileName, events_time, groups_time, shops_time, parkrun_time, info_time string) {
	outDir := filepath.Dir(fileName)
	makeDir(outDir)
	f, err := os.Create(fileName)
	check(err)

	defer f.Close()

	f.WriteString(`<?xml version="1.0" encoding="UTF-8"?>`)
	nl(f)
	f.WriteString(`<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">`)
	nl(f)

	genSitemapEntry(f, "https://freiburg.run/", events_time)
	genSitemapEntry(f, "https://freiburg.run/lauftreffs.html", groups_time)
	genSitemapEntry(f, "https://freiburg.run/shops.html", shops_time)
	genSitemapEntry(f, "https://freiburg.run/dietenbach-parkrun.html", parkrun_time)
	genSitemapEntry(f, "https://freiburg.run/info.html", info_time)

	f.WriteString(`</urlset>`)
}

func GetMtime(filePath string) time.Time {
	stat, err := os.Stat(filePath)
	check(err)
	return stat.ModTime()
}

var geoRe1 = regexp.MustCompile(`^\s*(\d*\.?\d*)\s*,\s*(\d*\.?\d*)\s*$`)
var geoRe2 = regexp.MustCompile(`^\s*N\s*(\d*\.?\d*)\s*E\s*(\d*\.?\d*)\s*$`)

func parseGeo(s string) string {
	m := geoRe1.FindStringSubmatch(s)
	if m != nil {
		return fmt.Sprintf("%s,%s", m[1], m[2])
	}
	m = geoRe2.FindStringSubmatch(s)
	if m != nil {
		return fmt.Sprintf("%s,%s", m[1], m[2])
	}
	return ""
}

var dateRe = regexp.MustCompile(`^\s*(\d\d\d\d)-(\d\d)-(\d\d)\s*$`)

func parseDate(s string) string {
	if s == "" {
		return ""
	}

	m := dateRe.FindStringSubmatch(s)
	if m != nil {
		return fmt.Sprintf("%s.%s.%s", m[3], m[2], m[1])
	}

	panic(fmt.Errorf("bad date: <%s>", s))
}

func writeCsv(fileName string, events []Event) {
	f, err := os.Create(fileName)
	check(err)
	defer f.Close()

	fmt.Fprintf(f, "\"%s\",\"%s\",\"%s\",\"%s\",\"%s\",\"%s\",\"%s\",\"%s\"\n", "ADDED", "DATE", "NAME", "URL", "DESCRIPTION", "LOCATION", "COORDINATES", "LINKS...")
	for _, event := range events {
		fmt.Fprintf(f, "\"%s\",\"%s\",\"%s\",\"%s\",\"%s\",\"%s\",\"%s\"", parseDate(event.Added), event.Time, event.Name, event.Url, event.Details, event.Location, event.Geo)
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

func fetchEventsJson(fileName string) ([]Event, string) {
	data, err := os.ReadFile(fileName)
	check(err)
	unmarshalled := make([]EventJson, 0)
	if err := json.Unmarshal(data, &unmarshalled); err != nil {
		panic(err)
	}
	mtime := GetMtime(fileName).Format("2006-01-02")

	events := make([]Event, 0)
	for _, e := range unmarshalled {
		ed := Event{
			e.Name, e.Time, e.Location, parseGeo(e.Geo), e.Details, e.Url, e.Reports, e.Added,
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
	mtime := GetMtime(fileName).Format("2006-01-02")

	return unmarshalled, mtime
}

func fetchEvents(config ConfigData, srv *sheets.Service, table string) ([]Event, string) {
	events := make([]Event, 0)
	mtime := ""

	resp, err := srv.Spreadsheets.Values.Get(config.SheetId, fmt.Sprintf("%s!A1", table)).Do()
	check(err)
	if len(resp.Values) != 0 {
		mtime = fmt.Sprintf("%v", resp.Values[0][0])
		if mtime != "" {
			r := regexp.MustCompile(`^\s*(\d+)\.(\d+)\.(\d+)\s*$`)
			m := r.FindStringSubmatch(mtime)
			if m == nil {
				log.Printf("GOOGLE SHEETS: bad mtime for '%s': '%s'\n", table, mtime)
				mtime = ""
			} else {
				mtime = fmt.Sprintf("%s-%s-%s", m[3], m[2], m[1])
			}
		}
	}

	resp, err = srv.Spreadsheets.Values.Get(config.SheetId, fmt.Sprintf("%s!A3:Z", table)).Do()
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
				coordinates = parseGeo(fmt.Sprintf("%v", row[6]))
			}
			if ll > 7 {
				links = parseLinks(row[7:])
			}
			events = append(events, Event{
				name,
				date,
				location,
				coordinates,
				description,
				url,
				links,
				added,
			})
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
		mtime = fmt.Sprintf("%v", resp.Values[0][0])
		if mtime != "" {
			r := regexp.MustCompile(`^\s*(\d+)\.(\d+)\.(\d+)\s*$`)
			m := r.FindStringSubmatch(mtime)
			if m == nil {
				log.Printf("GOOGLE SHEETS: bad mtime for '%s': '%s'\n", table, mtime)
				mtime = ""
			} else {
				mtime = fmt.Sprintf("%s-%s-%s", m[3], m[2], m[1])
			}
		}
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
	timestamp := time.Now().Format("2006-01-02")
	sheetUrl := ""
	options := parseCommandLine()

	var events []Event
	var events_pending []Event
	var events_time string

	var groups []Event
	var groups_time string

	var shops []Event
	var shops_time string

	var parkrun []ParkrunEvent
	var parkrun_time string

	if options.useJSON {
		var all_events []Event
		all_events, events_time = fetchEventsJson("data/events.json")
		for _, e := range all_events {
			if !strings.Contains(e.Time, "UNBEKANNT") {
				events = append(events, e)
			} else {
				events_pending = append(events_pending, e)
			}
		}

		groups, groups_time = fetchEventsJson("data/groups.json")
		shops, shops_time = fetchEventsJson("data/shops.json")
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

		events, events_time = fetchEvents(config, srv, "Events")
		events_pending, _ = fetchEvents(config, srv, "Events2")
		groups, groups_time = fetchEvents(config, srv, "Groups")
		shops, shops_time = fetchEvents(config, srv, "Shops")
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

	if options.exportCSV {
		writeCsv("events.csv", events)
		writeCsv("events2.csv", events_pending)
		writeCsv("groups.csv", groups)
		writeCsv("shops.csv", shops)
		writeParkrunCsv("parkrun.csv", parkrun)
	}

	info_time := GetMtime("templates/info.html").Format("2006-01-02")

	genSitemap(filepath.Join(options.outDir, "sitemap.xml"), events_time, groups_time, shops_time, parkrun_time, info_time)
	copyHash("static/.htaccess", ".htaccess", options.outDir)
	copyHash("static/robots.txt", "robots.txt", options.outDir)
	copyHash("static/favicon.png", "favicon.png", options.outDir)
	copyHash("static/favicon.ico", "favicon.ico", options.outDir)
	copyHash("static/apple-touch-icon.png", "apple-touch-icon.png", options.outDir)
	copyHash("static/freiburg-run.svg", "images/freiburg-run.svg", options.outDir)
	copyHash("static/events2023.jpg", "images/events2023.jpg", options.outDir)
	copyHash("static/marker-grey-icon.png", "images/marker-grey-icon.png", options.outDir)
	copyHash("static/marker-grey-icon-2x.png", "images/marker-grey-icon-2x.png", options.outDir)
	copyHash("static/circle-small.png", "images/circle-small.png", options.outDir)
	copyHash("static/circle-big.png", "images/circle-big.png", options.outDir)

	js_files := make([]string, 0)
	js_files = append(js_files, downloadHash("https://unpkg.com/leaflet@1.9.3/dist/leaflet.js", "leaflet-HASH.js", options.outDir))
	js_files = append(js_files, downloadHash("https://raw.githubusercontent.com/ptma/Leaflet.Legend/master/src/leaflet.legend.js", "leaflet-legend-HASH.js", options.outDir))
	js_files = append(js_files, copyHash("static/parkrun-track.js", "parkrun-track-HASH.js", options.outDir))
	js_files = append(js_files, copyHash("static/main.js", "main-HASH.js", options.outDir))

	css_files := make([]string, 0)
	css_files = append(css_files, downloadHash("https://cdn.jsdelivr.net/npm/bulma@0.9.4/css/bulma.min.css", "bulma-HASH.css", options.outDir))
	css_files = append(css_files, downloadHash("https://unpkg.com/leaflet@1.9.3/dist/leaflet.css", "leaflet-HASH.css", options.outDir))
	css_files = append(css_files, downloadHash("https://raw.githubusercontent.com/ptma/Leaflet.Legend/master/src/leaflet.legend.css", "leaflet-legend-HASH.css", options.outDir))
	css_files = append(css_files, downloadHash("https://raw.githubusercontent.com/justboil/bulma-responsive-tables/master/css/main.min.css", "bulma-responsive-tables-HASH.css", options.outDir))
	css_files = append(css_files, copyHash("static/style.css", "style-HASH.css", options.outDir))

	downloadHash("https://unpkg.com/leaflet@1.9.3/dist/images/marker-icon.png", "images/marker-icon.png", options.outDir)
	downloadHash("https://unpkg.com/leaflet@1.9.3/dist/images/marker-icon-2x.png", "images/marker-icon-2x.png", options.outDir)
	downloadHash("https://unpkg.com/leaflet@1.9.3/dist/images/marker-shadow.png", "images/marker-shadow.png", options.outDir)

	data := TemplateData{
		"Laufveranstaltungen im Raum Freiburg / Südbaden 2023",
		"Veranstaltung",
		"Liste von Laufveranstaltungen, Lauf-Wettkämpfen, Volksläufen 2023 im Raum Freiburg / Südbaden",
		"events",
		"https://freiburg.run/",
		timestamp,
		sheetUrl,
		events,
		events_pending,
		groups,
		shops,
		parkrun,
		js_files,
		css_files,
	}

	executeTemplate("events", filepath.Join(options.outDir, "index.html"), data)

	data.Nav = "groups"
	data.Title = "Lauftreffs im Raum Freiburg / Südbaden"
	data.Type = "Lauftreff"
	data.Description = "Liste von Lauftreffs, Laufgruppen, Lauf-Trainingsgruppen im Raum Freiburg / Südbaden"
	data.Canonical = "https://freiburg.run/lauftreffs.html"
	executeTemplate("groups", filepath.Join(options.outDir, "lauftreffs.html"), data)

	data.Nav = "shops"
	data.Title = "Lauf-Shops im Raum Freiburg / Südbaden"
	data.Type = "Lauf-Shop"
	data.Description = "Liste von Lauf-Shops, Geschäften mit Laufschuh-Auswahl im Raum Freiburg / Südbaden"
	data.Canonical = "https://freiburg.run/shops.html"
	executeTemplate("shops", filepath.Join(options.outDir, "shops.html"), data)

	data.Nav = "parkrun"
	data.Title = "Dietenbach parkrun - Karte, Ergebnisse, Laufberichte, Fotogalerien"
	data.Type = "Dietenbach parkrun"
	data.Description = "Dietenbach parkrun - Karte, Ergebnisse, Laufberichte, Fotogalerien"
	data.Canonical = "https://freiburg.run/dietenbach-parkrun.html"
	executeTemplate("dietenbach-parkrun", filepath.Join(options.outDir, "dietenbach-parkrun.html"), data)

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
}
