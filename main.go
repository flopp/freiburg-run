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

type Series struct {
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
}

type EventData struct {
	Name     string
	Time     string
	Location string
	Geo      string
	Details  string
	Url      string
	Series   []Series
}

type TemplateData struct {
	Title     string
	Nav       string
	Timestamp string
	Events    []EventData
	JsFiles   []string
	CssFiles  []string
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

	return fmt.Sprintf("%x", h.Sum(nil))
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

func main() {
	timestamp := time.Now().Format("2006-01-02 15:04:05")

	series_data, err := os.ReadFile("data/series.json")
	check(err)
	series := make([]Series, 0)
	if err := json.Unmarshal(series_data, &series); err != nil {
		panic(err)
	}

	events_data, err := os.ReadFile("data/events.json")
	check(err)
	events := make([]Event, 0)
	if err := json.Unmarshal(events_data, &events); err != nil {
		panic(err)
	}

	/*
		groups_data, err := os.ReadFile("data/groups.json")
		check(err)
		groups := make([]Event, 0)
		if err := json.Unmarshal(groups_data, &groups); err != nil {
			panic(err)
		}

		groups_extended := make([]EventData, 0)
		for _, e := range groups {
			ed := EventData{
				e.Name, e.Time, e.Location, e.Geo, e.Details, e.Url, nil,
			}
			groups_extended = append(groups_extended, ed)
		}
	*/

	events_extended := make([]EventData, 0)
	for _, e := range events {
		sd := make([]Series, 0)
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
			e.Name, e.Time, e.Location, e.Geo, e.Details, e.Url, sd,
		}
		events_extended = append(events_extended, ed)
	}

	copyHash("static/favicon.png", "favison.png")
	copyHash("static/freiburg-run.svg", "images/freiburg-run.svg")

	js_files := make([]string, 0)
	js_files = append(js_files, downloadHash("https://unpkg.com/leaflet@1.9.3/dist/leaflet.js", "leaflet-HASH.js"))
	js_files = append(js_files, copyHash("static/main.js", "main-HASH.js"))

	css_files := make([]string, 0)
	css_files = append(css_files, downloadHash("https://cdn.jsdelivr.net/npm/bulma@0.9.4/css/bulma.min.css", "bulma-HASH.css"))
	css_files = append(css_files, downloadHash("https://unpkg.com/leaflet@1.9.3/dist/leaflet.css", "leaflet-HASH.css"))
	css_files = append(css_files, downloadHash("https://raw.githubusercontent.com/justboil/bulma-responsive-tables/master/css/main.min.css", "bulma-responsive-tables-HASH.css"))
	css_files = append(css_files, copyHash("static/style.css", "style-HASH.css"))

	downloadHash("https://unpkg.com/leaflet@1.9.3/dist/images/marker-icon.png", "images/marker-icon.png")
	downloadHash("https://unpkg.com/leaflet@1.9.3/dist/images/marker-icon-2x.png", "images/marker-icon-2x.png")
	downloadHash("https://unpkg.com/leaflet@1.9.3/dist/images/marker-shadow.png", "images/marker-shadow.png")

	data := TemplateData{
		"Laufveranstaltungen",
		"events",
		timestamp,
		events_extended,
		js_files,
		css_files,
	}

	executeTemplate("events", ".out/index.html", data)
	data.Nav = "datenschutz"
	data.Title = "Datenschutz"
	executeTemplate("datenschutz", ".out/datenschutz.html", data)
	data.Nav = "impressum"
	data.Title = "Impressum"
	executeTemplate("impressum", ".out/impressum.html", data)
	data.Nav = "info"
	data.Title = "Info"
	executeTemplate("info", ".out/info.html", data)

	/*
		data2 := TemplateData{
			"Lauftreffs",
			"groups",
			timestamp,
			groups_extended,
			js_files,
			css_files,
		}
		executeTemplate("groups", ".out/lauftreffs.html", data2)
	*/
}
