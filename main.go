package main

import (
	"crypto/sha256"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"
)

type NameUrl struct {
	Name string
	Url  string
}

type Event struct {
	Name     string
	Time     string
	Location string
	Geo      string
	Details  string
	Url      string
	Series   []string
	Reports  []NameUrl
	Added    string
}

type EventData struct {
	Name     string
	Time     string
	Location string
	Geo      string
	Details  string
	Url      string
	Series   []NameUrl
	Reports  []NameUrl
	Added    string
}

type ParkrunEventData struct {
	Index   string
	Date    string
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
	Events        []EventData
	EventsPending []EventData
	Groups        []EventData
	Shops         []EventData
	Parkrun       []ParkrunEventData
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

func copyHash(src, dst string) string {
	dir := filepath.Join(".out", filepath.Dir(dst))
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
	dstHash2 := filepath.Join(".out", dstHash)
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

func downloadHash(url string, dst string) string {
	if strings.Contains(dst, "HASH") {
		tmpfile, err := os.CreateTemp("", "")
		check(err)
		defer os.Remove(tmpfile.Name())

		download(url, tmpfile.Name())

		return copyHash(tmpfile.Name(), dst)
	} else {
		dst2 := filepath.Join(".out", dst)

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
	makeDir(".out")
	f, err := os.Create(filepath.Join(".out", fileName))
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

func main() {
	timestamp := time.Now().Format("2006-01-02")

	series_data, err := os.ReadFile("data/series.json")
	check(err)
	series := make([]NameUrl, 0)
	if err := json.Unmarshal(series_data, &series); err != nil {
		panic(err)
	}

	events_data, err := os.ReadFile("data/events.json")
	check(err)
	events := make([]Event, 0)
	if err := json.Unmarshal(events_data, &events); err != nil {
		panic(err)
	}
	events_time := GetMtime("data/events.json").Format("2006-01-02")

	groups_data, err := os.ReadFile("data/groups.json")
	check(err)
	groups := make([]Event, 0)
	if err := json.Unmarshal(groups_data, &groups); err != nil {
		panic(err)
	}
	groups_time := GetMtime("data/groups.json").Format("2006-01-02")

	groups_extended := make([]EventData, 0)
	for _, e := range groups {
		ed := EventData{
			e.Name, e.Time, e.Location, e.Geo, e.Details, e.Url, nil, e.Reports, e.Added,
		}
		groups_extended = append(groups_extended, ed)
	}

	shops_data, err := os.ReadFile("data/shops.json")
	check(err)
	shops := make([]Event, 0)
	if err := json.Unmarshal(shops_data, &shops); err != nil {
		panic(err)
	}
	shops_time := GetMtime("data/shops.json").Format("2006-01-02")

	shops_extended := make([]EventData, 0)
	for _, e := range shops {
		ed := EventData{
			e.Name, e.Time, e.Location, e.Geo, e.Details, e.Url, nil, e.Reports, e.Added,
		}
		shops_extended = append(shops_extended, ed)
	}

	events_extended := make([]EventData, 0)
	events_pending := make([]EventData, 0)
	for _, e := range events {
		sd := make([]NameUrl, 0)
		for _, s := range e.Series {
			found := false
			for _, s2 := range series {
				if s2.Name == s {
					found = true
					sd = append(sd, s2)
					break
				}
			}
			if !found {
				panic(fmt.Errorf("bad series: %s", s))
			}
		}
		ed := EventData{
			e.Name, e.Time, e.Location, e.Geo, e.Details, e.Url, sd, e.Reports, e.Added,
		}
		if !strings.Contains(ed.Time, "UNBEKANNT") {
			events_extended = append(events_extended, ed)
		} else {
			events_pending = append(events_pending, ed)
		}
	}

	parkrun_data, err := os.ReadFile("data/dietenbach-parkrun.json")
	check(err)
	parkrun := make([]ParkrunEventData, 0)
	if err := json.Unmarshal(parkrun_data, &parkrun); err != nil {
		panic(err)
	}
	parkrun_time := GetMtime("data/dietenbach-parkrun.json").Format("2006-01-02")

	info_time := GetMtime("templates/info.html").Format("2006-01-02")

	genSitemap("sitemap.xml", events_time, groups_time, shops_time, parkrun_time, info_time)
	copyHash("static/.htaccess", ".htaccess")
	copyHash("static/robots.txt", "robots.txt")
	copyHash("static/favicon.png", "favicon.png")
	copyHash("static/favicon.ico", "favicon.ico")
	copyHash("static/apple-touch-icon.png", "apple-touch-icon.png")
	copyHash("static/freiburg-run.svg", "images/freiburg-run.svg")
	copyHash("static/events2023.jpg", "images/events2023.jpg")
	copyHash("static/marker-grey-icon.png", "images/marker-grey-icon.png")
	copyHash("static/marker-grey-icon-2x.png", "images/marker-grey-icon-2x.png")
	copyHash("static/circle-small.png", "images/circle-small.png")
	copyHash("static/circle-big.png", "images/circle-big.png")

	js_files := make([]string, 0)
	js_files = append(js_files, downloadHash("https://unpkg.com/leaflet@1.9.3/dist/leaflet.js", "leaflet-HASH.js"))
	js_files = append(js_files, downloadHash("https://raw.githubusercontent.com/ptma/Leaflet.Legend/master/src/leaflet.legend.js", "leaflet-legend-HASH.js"))
	js_files = append(js_files, copyHash("static/main.js", "main-HASH.js"))

	css_files := make([]string, 0)
	css_files = append(css_files, downloadHash("https://cdn.jsdelivr.net/npm/bulma@0.9.4/css/bulma.min.css", "bulma-HASH.css"))
	css_files = append(css_files, downloadHash("https://unpkg.com/leaflet@1.9.3/dist/leaflet.css", "leaflet-HASH.css"))
	css_files = append(css_files, downloadHash("https://raw.githubusercontent.com/ptma/Leaflet.Legend/master/src/leaflet.legend.css", "leaflet-legend-HASH.css"))
	css_files = append(css_files, downloadHash("https://raw.githubusercontent.com/justboil/bulma-responsive-tables/master/css/main.min.css", "bulma-responsive-tables-HASH.css"))
	css_files = append(css_files, copyHash("static/style.css", "style-HASH.css"))

	downloadHash("https://unpkg.com/leaflet@1.9.3/dist/images/marker-icon.png", "images/marker-icon.png")
	downloadHash("https://unpkg.com/leaflet@1.9.3/dist/images/marker-icon-2x.png", "images/marker-icon-2x.png")
	downloadHash("https://unpkg.com/leaflet@1.9.3/dist/images/marker-shadow.png", "images/marker-shadow.png")

	data := TemplateData{
		"Laufveranstaltungen im Raum Freiburg / Südbaden 2023",
		"Veranstaltung",
		"Liste von Laufveranstaltungen, Lauf-Wettkämpfen, Volksläufen 2023 im Raum Freiburg / Südbaden",
		"events",
		"https://freiburg.run/",
		timestamp,
		events_extended,
		events_pending,
		groups_extended,
		shops_extended,
		parkrun,
		js_files,
		css_files,
	}

	executeTemplate("events", ".out/index.html", data)

	data.Nav = "groups"
	data.Title = "Lauftreffs im Raum Freiburg / Südbaden"
	data.Type = "Lauftreff"
	data.Description = "Liste von Lauftreffs, Laufgruppen, Lauf-Trainingsgruppen im Raum Freiburg / Südbaden"
	data.Canonical = "https://freiburg.run/lauftreffs.html"
	executeTemplate("groups", ".out/lauftreffs.html", data)

	data.Nav = "shops"
	data.Title = "Lauf-Shops im Raum Freiburg / Südbaden"
	data.Type = "Lauf-Shop"
	data.Description = "Liste von Lauf-Shops, Geschäften mit Laufschuh-Auswahl im Raum Freiburg / Südbaden"
	data.Canonical = "https://freiburg.run/shops.html"
	executeTemplate("shops", ".out/shops.html", data)

	data.Nav = "parkrun"
	data.Title = "Dietenbach parkrun - Ergebnisse, Laufberichte, Fotogalerien"
	data.Type = "Dietenbach parkrun"
	data.Description = "Dietenbach parkrun - Ergebnisse, Laufberichte, Fotogalerien"
	data.Canonical = "https://freiburg.run/dietenbach-parkrun.html"
	executeTemplate("dietenbach-parkrun", ".out/dietenbach-parkrun.html", data)

	data.Nav = "datenschutz"
	data.Title = "Datenschutz"
	data.Type = "Datenschutz"
	data.Description = "Datenschutzerklärung von freiburg.run"
	data.Canonical = "https://freiburg.run/datenschutz.html"
	executeTemplate("datenschutz", ".out/datenschutz.html", data)

	data.Nav = "impressum"
	data.Title = "Impressum"
	data.Type = "Impressum"
	data.Description = "Impressum von freiburg.run"
	data.Canonical = "https://freiburg.run/impressum.html"
	executeTemplate("impressum", ".out/impressum.html", data)

	data.Nav = "info"
	data.Title = "Info"
	data.Type = "Info"
	data.Description = "Kontaktmöglichkeiten, aallgemein & technische Informationen über freiburg.run"
	data.Canonical = "https://freiburg.run/info.html"
	executeTemplate("info", ".out/info.html", data)

	data.Nav = "404"
	data.Title = "404 - Seite nicht gefunden :("
	data.Type = ""
	data.Description = "Fehlerseite von friburg.run"
	data.Canonical = "https://freiburg.run/404.html"
	executeTemplate("404", ".out/404.html", data)
}
