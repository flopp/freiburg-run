package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/flopp/freiburg-run/internal/events"
	"github.com/flopp/freiburg-run/internal/utils"
)

const (
	usage = `USAGE: %s [OPTIONS...]

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

func IsNew(s string, now time.Time) bool {
	days := 14

	d, err := utils.ParseDate(s)
	if err == nil {
		return d.AddDate(0, 0, days).After(now)
	}

	return false
}

type UmamiData struct {
	Url string
	Id  string
}

type TemplateData struct {
	Title         string
	Type          string
	Description   string
	Nav           string
	Canonical     string
	Breadcrumbs   []utils.Breadcrumb
	Main          string
	Timestamp     string
	TimestampFull string
	SheetUrl      string
	Data          *events.Data
	JsFiles       []string
	CssFiles      []string
	Umami         UmamiData
}

func (t TemplateData) Image() string {
	if t.Nav == "parkrun" {
		return "https://freiburg.run/images/parkrun.png"
	}
	return "https://freiburg.run/images/512.png"
}

type GenericTemplateData struct {
	Base TemplateData
}

func (d GenericTemplateData) YearTitle() string {
	return d.Base.Title
}

func (d GenericTemplateData) CountEvents() int {
	count := 0
	for _, event := range d.Base.Data.Events {
		if !event.IsSeparator() {
			count += 1
		}
	}
	return count
}

type EventTemplateData struct {
	Event *events.Event
	Base  TemplateData
}

func (d EventTemplateData) YearTitle() string {
	if d.Event.Type != "event" {
		return d.Base.Title
	}

	if d.Event.Time.IsZero() {
		return d.Base.Title
	}

	yearS := fmt.Sprintf("%d", d.Event.Time.Year())
	if strings.Contains(d.Base.Title, yearS) {
		return d.Base.Title
	}

	return fmt.Sprintf("%s %s", d.Base.Title, yearS)
}

type TagTemplateData struct {
	Tag  *events.Tag
	Base TemplateData
}

func (d TagTemplateData) YearTitle() string {
	return d.Base.Title
}

type SerieTemplateData struct {
	Serie *events.Serie
	Base  TemplateData
}

func (d SerieTemplateData) YearTitle() string {
	return d.Base.Title
}

type SitemapTemplateData struct {
	Categories []utils.SitemapCategory
	Base       TemplateData
}

func (d SitemapTemplateData) YearTitle() string {
	return d.Base.Title
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

func updateAddedDates(events []*events.Event, added *utils.Added, eventType string, timestamp string, now time.Time) {
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

func CreateHtaccess(data events.Data, outDir string) error {
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
	destination.WriteString("Content-Security-Policy: frame-ancestors 'self' https://2oc.de;\n")
	destination.WriteString("Redirect /parkrun /dietenbach-parkrun.html\n")
	destination.WriteString("Redirect /groups.html /lauftreffs.html\n")
	destination.WriteString("Redirect /event/2024-32-teninger-allmendlauf.html?back=event /event/2024-32-teninger-allmendlauf.html\n")
	destination.WriteString("Redirect /event/dietenbach-parkrun.html /group/dietenbach-parkrun.html\n")
	destination.WriteString("Redirect /event/dreilaendergarten-parkrun.html /group/dreilaendergarten-parkrun.html\n")
	destination.WriteString("Redirect /tag/serie-intersport-denzer-cup-2024.html /serie/intersport-denzer-cup-2024.html\n")
	destination.WriteString("Redirect /event/2023-4-crosslauf-am-opfinger-see.html /event/2024-4-crosslauf-am-opfinger-see.html\n")

	for _, e := range data.Events {
		if old := e.SlugOld(); old != "" {
			destination.WriteString(fmt.Sprintf("Redirect /%s /%s\n", old, e.Slug()))
		}
	}
	for _, e := range data.EventsOld {
		if old := e.SlugOld(); old != "" {
			destination.WriteString(fmt.Sprintf("Redirect /%s /%s\n", old, e.Slug()))
		}
	}
	for _, e := range data.Groups {
		if old := e.SlugOld(); old != "" {
			destination.WriteString(fmt.Sprintf("Redirect /%s /%s\n", old, e.Slug()))
		}
	}
	for _, e := range data.Shops {
		if old := e.SlugOld(); old != "" {
			destination.WriteString(fmt.Sprintf("Redirect /%s /%s\n", old, e.Slug()))
		}
	}

	for _, e := range data.EventsObsolete {
		destination.WriteString(fmt.Sprintf("Redirect /%s /\n", e.Slug()))
	}
	for _, e := range data.GroupsObsolete {
		destination.WriteString(fmt.Sprintf("Redirect /%s /lauftreffs.html\n", e.Slug()))
	}
	for _, e := range data.ShopsObsolete {
		destination.WriteString(fmt.Sprintf("Redirect /%s /shops.html\n", e.Slug()))
	}

	return nil
}

type FileSet struct {
	paths []string
}

func CreateFileSet() FileSet {
	return FileSet{make([]string, 0)}
}

func (fs *FileSet) Add(path string) {
	fs.paths = append(fs.paths, path)
}

func (fs FileSet) Rel(basePath string) []string {
	relPaths := make([]string, 0, len(fs.paths))
	for _, path := range fs.paths {
		relPath, err := filepath.Rel(basePath, path)
		utils.Check(err)
		relPaths = append(relPaths, relPath)
	}
	return relPaths
}

func MustRel(basepath, path string) string {
	rel, err := filepath.Rel(basepath, path)
	utils.Check(err)
	return rel
}

type Path string

func (p Path) Join(s string) string {
	return filepath.Join(string(p), s)
}

func retry[T any](attempts int, sleep time.Duration, f func() (T, error)) (result T, err error) {
	for attempt := range attempts {
		if attempt > 0 {
			time.Sleep(sleep)
			sleep *= 2
		}
		result, err = f()
		if err == nil {
			return result, nil
		}
	}
	return result, fmt.Errorf("after %d attempts, last error: %s", attempts, err)
}

func main() {
	baseUrl := "https://freiburg.run"
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	timestamp := now.Format("2006-01-02")
	timestampFull := now.Format("2006-01-02 15:04:05")
	options := parseCommandLine()
	out := Path(options.outDir)

	config_data, err := events.LoadSheetsConfig(options.configFile)
	if err != nil {
		log.Fatalf("failed to load config file: %v", err)
		return
	}

	sheetUrl := fmt.Sprintf("https://docs.google.com/spreadsheets/d/%s", config_data.SheetId)

	// try 3 times to fetch data with increasing timeouts (sometimes the google api is not available)
	eventsData, err := retry(3, 8*time.Second, func() (events.Data, error) {
		return events.FetchData(config_data, today)
	})
	if err != nil {
		log.Fatalf("failed to fetch data: %v", err)
		return
	}

	if options.addedFile != "" {
		added, err := utils.ReadAdded(options.addedFile)
		if err != nil {
			log.Printf("failed to parse added file: '%s' - %v", options.addedFile, err)
		}

		updateAddedDates(eventsData.Events, added, "event", timestamp, now)
		updateAddedDates(eventsData.EventsOld, added, "event", timestamp, now)
		updateAddedDates(eventsData.Groups, added, "group", timestamp, now)
		updateAddedDates(eventsData.Shops, added, "shop", timestamp, now)

		if err = added.Write(options.addedFile); err != nil {
			log.Printf("failed to write added file: '%s' - %v", options.addedFile, err)
		}
	}

	// create ics files for events
	for _, event := range eventsData.Events {
		if event.IsSeparator() {
			continue
		}
		calendar := event.CalendarSlug()
		if err := events.CreateEventCalendar(event, now, baseUrl, fmt.Sprintf("%s/%s", baseUrl, calendar), out.Join(calendar)); err != nil {
			log.Printf("failed to create event calendar: %v", err)
		} else {
			event.Calendar = fmt.Sprintf("/%s", calendar)
		}
	}
	for _, event := range eventsData.EventsOld {
		if event.IsSeparator() {
			continue
		}
		calendar := event.CalendarSlug()
		if err := events.CreateEventCalendar(event, now, baseUrl, fmt.Sprintf("%s/%s", baseUrl, calendar), out.Join(calendar)); err != nil {
			log.Printf("failed to create event calendar: %v", err)
		} else {
			event.Calendar = fmt.Sprintf("/%s", calendar)
		}
	}

	sitemap := utils.CreateSitemap(baseUrl)
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
	sitemap.Add("support.html", "Unterstützen", "Allgemein")
	sitemap.Add("datenschutz.html", "Datenschutz", "Allgemein")
	sitemap.Add("impressum.html", "Impressum", "Allgemein")

	utils.MustCopy("static/robots.txt", out.Join("robots.txt"))
	utils.MustCopy("static/manifest.json", out.Join("manifest.json"))

	// ahrefs validation
	utils.MustCopy("static/5vkf9hdnfkay895vyx33zdvesnyaphgv.txt", out.Join("5vkf9hdnfkay895vyx33zdvesnyaphgv.txt"))
	utils.MustCopy("static/512.png", out.Join("favicon.png"))
	utils.MustCopy("static/favicon.ico", out.Join("favicon.ico"))
	utils.MustCopy("static/180.png", out.Join("apple-touch-icon.png"))
	utils.MustCopy("static/192.png", out.Join("android-chrome-192x192.png"))
	utils.MustCopy("static/512.png", out.Join("android-chrome-512x512.png"))
	utils.MustCopy("static/freiburg-run.svg", out.Join("images/freiburg-run.svg"))
	utils.MustCopy("static/freiburg-run-new.svg", out.Join("images/freiburg-run-new.svg"))
	utils.MustCopy("static/freiburg-run-new-blue.svg", out.Join("images/freiburg-run-new-blue.svg"))
	utils.MustCopy("static/512.png", out.Join("images/512.png"))
	utils.MustCopy("static/parkrun.png", out.Join("images/parkrun.png"))
	utils.MustCopy("static/marker-grey-icon.png", out.Join("images/marker-grey-icon.png"))
	utils.MustCopy("static/marker-grey-icon-2x.png", out.Join("images/marker-grey-icon-2x.png"))
	utils.MustCopy("static/marker-green-icon.png", out.Join("images/marker-green-icon.png"))
	utils.MustCopy("static/marker-green-icon-2x.png", out.Join("images/marker-green-icon-2x.png"))
	utils.MustCopy("static/marker-red-icon.png", out.Join("images/marker-red-icon.png"))
	utils.MustCopy("static/marker-red-icon-2x.png", out.Join("images/marker-red-icon-2x.png"))
	utils.MustCopy("static/circle-small.png", out.Join("images/circle-small.png"))
	utils.MustCopy("static/circle-big.png", out.Join("images/circle-big.png"))
	utils.MustCopy("static/freiburg-run-flyer.pdf", out.Join("freiburg-run-flyer.pdf"))

	// renovate: datasource=npm depName=bulma
	bulma_version := "1.0.3"
	// renovate: datasource=npm depName=leaflet
	leaflet_version := "1.9.4"
	// renovate: datasource=npm depName=leaflet-gesture-handling
	leaflet_gesture_handling_version := "1.2.2"

	leaflet_legend_version := "v1.0.0"

	bulma_url := fmt.Sprintf("https://unpkg.com/bulma@%s", bulma_version)
	leaflet_url := fmt.Sprintf("https://unpkg.com/leaflet@%s", leaflet_version)
	leaflet_gesture_handling_url := fmt.Sprintf("https://unpkg.com/leaflet-gesture-handling@%s", leaflet_gesture_handling_version)
	leaflet_legend_url := fmt.Sprintf("https://raw.githubusercontent.com/ptma/Leaflet.Legend/%s", leaflet_legend_version)

	js_files := CreateFileSet()
	css_files := CreateFileSet()
	js_files.Add(utils.MustDownloadHash(fmt.Sprintf("%s/dist/leaflet.js", leaflet_url), out.Join("leaflet-HASH.js")))
	js_files.Add(utils.MustDownloadHash(fmt.Sprintf("%s/src/leaflet.legend.js", leaflet_legend_url), out.Join("leaflet-legend-HASH.js")))
	js_files.Add(utils.MustDownloadHash(fmt.Sprintf("%s/dist/leaflet-gesture-handling.min.js", leaflet_gesture_handling_url), out.Join("leaflet-gesture-handling-HASH.js")))
	js_files.Add(utils.MustCopyHash("static/parkrun-track.js", out.Join("parkrun-track-HASH.js")))
	js_files.Add(utils.MustCopyHash("static/main.js", out.Join("main-HASH.js")))
	umamiScript := utils.MustDownloadHash("https://cloud.umami.is/script.js", out.Join("umami-HASH.js"))

	css_files.Add(utils.MustDownloadHash(fmt.Sprintf("%s/css/bulma.min.css", bulma_url), out.Join("bulma-HASH.css")))
	css_files.Add(utils.MustDownloadHash(fmt.Sprintf("%s/dist/leaflet.css", leaflet_url), out.Join("leaflet-HASH.css")))
	css_files.Add(utils.MustDownloadHash(fmt.Sprintf("%s/src/leaflet.legend.css", leaflet_legend_url), out.Join("leaflet-legend-HASH.css")))
	css_files.Add(utils.MustDownloadHash(fmt.Sprintf("%s/dist/leaflet-gesture-handling.min.css", leaflet_gesture_handling_url), out.Join("leaflet-gesture-handling-HASH.css")))
	css_files.Add(utils.MustCopyHash("static/style.css", out.Join("style-HASH.css")))

	utils.MustDownload(fmt.Sprintf("%s/dist/images/marker-icon.png", leaflet_url), out.Join("images/marker-icon.png"))
	utils.MustDownload(fmt.Sprintf("%s/dist/images/marker-icon-2x.png", leaflet_url), out.Join("images/marker-icon-2x.png"))
	utils.MustDownload(fmt.Sprintf("%s/dist/images/marker-shadow.png", leaflet_url), out.Join("images/marker-shadow.png"))

	breadcrumbsBase := utils.InitBreadcrumbs(utils.CreateLink("freiburg.run", "/"))
	breadcrumbsEvents := utils.PushBreadcrumb(breadcrumbsBase, utils.CreateLink("Laufveranstaltungen", "/"))

	umami := UmamiData{MustRel(options.outDir, umamiScript), "6609164f-5e79-4041-b1ed-f37da10a84d2"}

	data := GenericTemplateData{
		TemplateData{
			"Laufveranstaltungen im Raum Freiburg",
			"Veranstaltung",
			"Liste von aktuellen und zukünftigen Laufveranstaltungen, Lauf-Wettkämpfen, Volksläufen im Raum Freiburg",
			"events",
			baseUrl,
			breadcrumbsEvents,
			"/",
			timestamp,
			timestampFull,
			sheetUrl,
			&eventsData,
			js_files.Rel(options.outDir),
			css_files.Rel(options.outDir),
			umami,
		},
	}

	utils.ExecuteTemplate("events", out.Join("index.html"), data)

	breadcrumbsEventsOld := utils.PushBreadcrumb(breadcrumbsEvents, utils.CreateLink("Archiv", "/events-old.html"))
	data.Base.Title = "Vergangene Laufveranstaltungen im Raum Freiburg "
	data.Base.Description = "Liste von vergangenen Laufveranstaltungen, Lauf-Wettkämpfen, Volksläufen im Raum Freiburg "
	data.Base.Canonical = fmt.Sprintf("%s/%s", baseUrl, "events-old.html")
	data.Base.Breadcrumbs = breadcrumbsEventsOld
	utils.ExecuteTemplate("events-old", out.Join("events-old.html"), data)

	breadcrumbsEventsTags := utils.PushBreadcrumb(breadcrumbsEvents, utils.CreateLink("Kategorien", "/tags.html"))
	data.Base.Nav = "tags"
	data.Base.Title = "Kategorien"
	data.Base.Description = "Liste aller Kategorien von Laufveranstaltungen, Lauf-Wettkämpfen, Volksläufen im Raum Freiburg"
	data.Base.Canonical = fmt.Sprintf("%s/%s", baseUrl, "tags.html")
	data.Base.Breadcrumbs = breadcrumbsEventsTags
	utils.ExecuteTemplate("tags", out.Join("tags.html"), data)

	breadcrumbsGroups := utils.PushBreadcrumb(breadcrumbsBase, utils.CreateLink("Lauftreffs", "/lauftreffs.html"))
	data.Base.Nav = "groups"
	data.Base.Title = "Lauftreffs im Raum Freiburg"
	data.Base.Type = "Lauftreff"
	data.Base.Description = "Liste von Lauftreffs, Laufgruppen, Lauf-Trainingsgruppen im Raum Freiburg "
	data.Base.Canonical = fmt.Sprintf("%s/%s", baseUrl, "lauftreffs.html")
	data.Base.Breadcrumbs = breadcrumbsGroups
	utils.ExecuteTemplate("groups", out.Join("lauftreffs.html"), data)

	breadcrumbsShops := utils.PushBreadcrumb(breadcrumbsBase, utils.CreateLink("Lauf-Shops", "/shops.html"))
	data.Base.Nav = "shops"
	data.Base.Title = "Lauf-Shops im Raum Freiburg"
	data.Base.Type = "Lauf-Shop"
	data.Base.Description = "Liste von Lauf-Shops und Einzelhandelsgeschäften mit Laufschuh-Auswahl im Raum Freiburg "
	data.Base.Canonical = fmt.Sprintf("%s/%s", baseUrl, "shops.html")
	data.Base.Breadcrumbs = breadcrumbsShops
	utils.ExecuteTemplate("shops", out.Join("shops.html"), data)

	data.Base.Nav = "parkrun"
	data.Base.Title = "Dietenbach parkrun"
	data.Base.Type = "Dietenbach parkrun"
	data.Base.Description = "Vollständige Liste aller Ergebnisse, Laufberichte und Fotogalerien des 'Dietenbach parkrun' im Freiburger Dietenbachpark."
	data.Base.Canonical = fmt.Sprintf("%s/%s", baseUrl, "dietenbach-parkrun.html")
	data.Base.Breadcrumbs = utils.PushBreadcrumb(breadcrumbsBase, utils.CreateLink("Dietenbach parkrun", "/dietenbach-parkrun.html"))
	utils.ExecuteTemplate("dietenbach-parkrun", out.Join("dietenbach-parkrun.html"), data)
	utils.ExecuteTemplateNoMinify("dietenbach-parkrun-wordpress", out.Join("dietenbach-parkrun-wordpress.html"), data)

	breadcrumbsEventsSeries := utils.PushBreadcrumb(breadcrumbsEvents, utils.CreateLink("Serien", "/series.html"))
	data.Base.Nav = "series"
	data.Base.Title = "Serien"
	data.Base.Description = "Liste aller Serien von Laufveranstaltungen, Lauf-Wettkämpfen, Volksläufen im Raum Freiburg "
	data.Base.Canonical = fmt.Sprintf("%s/%s", baseUrl, "series.html")
	data.Base.Breadcrumbs = breadcrumbsEventsSeries
	utils.ExecuteTemplate("series", out.Join("series.html"), data)

	data.Base.Nav = "map"
	data.Base.Title = "Karte aller Laufveranstaltungen"
	data.Base.Type = "Karte"
	data.Base.Description = "Karte"
	data.Base.Canonical = fmt.Sprintf("%s/%s", baseUrl, "map.html")
	data.Base.Breadcrumbs = utils.PushBreadcrumb(breadcrumbsBase, utils.CreateLink("Karte", "/map.html"))
	utils.ExecuteTemplate("map", out.Join("map.html"), data)

	breadcrumbsInfo := utils.PushBreadcrumb(breadcrumbsBase, utils.CreateLink("Info", "/info.html"))
	data.Base.Nav = "datenschutz"
	data.Base.Title = "Datenschutz"
	data.Base.Type = "Datenschutz"
	data.Base.Description = "Datenschutzerklärung von freiburg.run"
	data.Base.Canonical = fmt.Sprintf("%s/%s", baseUrl, "datenschutz.html")
	data.Base.Breadcrumbs = utils.PushBreadcrumb(breadcrumbsInfo, utils.CreateLink("Datenschutz", "/datenschutz.html"))
	utils.ExecuteTemplate("datenschutz", out.Join("datenschutz.html"), data)

	data.Base.Nav = "impressum"
	data.Base.Title = "Impressum"
	data.Base.Type = "Impressum"
	data.Base.Description = "Impressum von freiburg.run"
	data.Base.Canonical = fmt.Sprintf("%s/%s", baseUrl, "impressum.html")
	data.Base.Breadcrumbs = utils.PushBreadcrumb(breadcrumbsInfo, utils.CreateLink("Impressum", "/impressum.html"))
	utils.ExecuteTemplate("impressum", out.Join("impressum.html"), data)

	data.Base.Nav = "info"
	data.Base.Title = "Info"
	data.Base.Type = "Info"
	data.Base.Description = "Kontaktmöglichkeiten, allgemeine & technische Informationen über freiburg.run"
	data.Base.Canonical = fmt.Sprintf("%s/%s", baseUrl, "info.html")
	data.Base.Breadcrumbs = breadcrumbsInfo
	utils.ExecuteTemplate("info", out.Join("info.html"), data)

	data.Base.Nav = "support"
	data.Base.Title = "freiburg.run unterstützen"
	data.Base.Type = "Support"
	data.Base.Description = "Möglichkeiten freiburg.run zu unterstützen"
	data.Base.Canonical = fmt.Sprintf("%s/%s", baseUrl, "support.html")
	data.Base.Breadcrumbs = utils.PushBreadcrumb(breadcrumbsInfo, utils.CreateLink("Unterstützung", "/support.html"))
	utils.ExecuteTemplate("support", out.Join("support.html"), data)

	data.Base.Nav = "404"
	data.Base.Title = "404 - Seite nicht gefunden :("
	data.Base.Type = ""
	data.Base.Description = "Fehlerseite von freiburg.run"
	data.Base.Canonical = fmt.Sprintf("%s/%s", baseUrl, "404.html")
	data.Base.Breadcrumbs = utils.PushBreadcrumb(breadcrumbsBase, utils.CreateLink("Fehlerseite", "/404.html"))
	utils.ExecuteTemplate("404", out.Join("404.html"), data)

	eventdata := EventTemplateData{
		nil,
		TemplateData{
			"",
			"Veranstaltung",
			"",
			"events",
			"",
			breadcrumbsEvents,
			"/",
			timestamp,
			timestampFull,
			sheetUrl,
			&eventsData,
			js_files.Rel(options.outDir),
			css_files.Rel(options.outDir),
			umami,
		},
	}
	for _, event := range eventsData.Events {
		if event.IsSeparator() {
			continue
		}
		eventdata.Event = event
		eventdata.Base.Title = event.Name
		eventdata.Base.Description = event.GenerateDescription()
		slug := event.Slug()
		eventdata.Base.Canonical = fmt.Sprintf("%s/%s", baseUrl, slug)
		eventdata.Base.Breadcrumbs = utils.PushBreadcrumb(breadcrumbsEvents, utils.CreateLink(event.Name, fmt.Sprintf("/%s", slug)))
		utils.ExecuteTemplate("event", out.Join(slug), eventdata)
		sitemap.Add(slug, event.Name, "Laufveranstaltungen")
	}

	eventdata.Base.Main = "/events-old.html"
	for _, event := range eventsData.EventsOld {
		if event.IsSeparator() {
			continue
		}
		eventdata.Event = event
		eventdata.Base.Title = event.Name
		eventdata.Base.Description = event.GenerateDescription()
		slug := event.Slug()
		eventdata.Base.Canonical = fmt.Sprintf("%s/%s", baseUrl, slug)
		eventdata.Base.Breadcrumbs = utils.PushBreadcrumb(breadcrumbsEventsOld, utils.CreateLink(event.Name, fmt.Sprintf("/%s", slug)))
		utils.ExecuteTemplate("event", out.Join(slug), eventdata)
		sitemap.Add(slug, event.Name, "Vergangene Laufveranstaltungen")
	}

	eventdata.Base.Type = "Lauftreff"
	eventdata.Base.Nav = "groups"
	eventdata.Base.Main = "/lauftreffs.html"
	for _, event := range eventsData.Groups {
		eventdata.Event = event
		eventdata.Base.Title = event.Name
		eventdata.Base.Description = event.GenerateDescription()
		slug := event.Slug()
		eventdata.Base.Canonical = fmt.Sprintf("%s/%s", baseUrl, slug)
		eventdata.Base.Breadcrumbs = utils.PushBreadcrumb(breadcrumbsGroups, utils.CreateLink(event.Name, fmt.Sprintf("/%s", slug)))
		utils.ExecuteTemplate("event", out.Join(slug), eventdata)
		sitemap.Add(slug, event.Name, "Lauftreffs")
	}

	eventdata.Base.Type = "Lauf-Shop"
	eventdata.Base.Nav = "shops"
	eventdata.Base.Main = "/shops.html"
	for _, event := range eventsData.Shops {
		eventdata.Event = event
		eventdata.Base.Title = event.Name
		eventdata.Base.Description = event.GenerateDescription()
		slug := event.Slug()
		eventdata.Base.Canonical = fmt.Sprintf("%s/%s", baseUrl, slug)
		eventdata.Base.Breadcrumbs = utils.PushBreadcrumb(breadcrumbsShops, utils.CreateLink(event.Name, fmt.Sprintf("/%s", slug)))
		utils.ExecuteTemplate("event", out.Join(slug), eventdata)
		sitemap.Add(slug, event.Name, "Lauf-Shops")
	}

	tagdata := TagTemplateData{
		nil,
		TemplateData{
			"",
			"Kategorie",
			"",
			"tags",
			"",
			breadcrumbsEventsTags,
			"/tags.html",
			timestamp,
			timestampFull,
			sheetUrl,
			&eventsData,
			js_files.Rel(options.outDir),
			css_files.Rel(options.outDir),
			umami,
		},
	}
	for _, tag := range eventsData.Tags {
		tagdata.Tag = tag
		tagdata.Base.Title = fmt.Sprintf("Laufveranstaltungen der Kategorie '%s'", tag.Name)
		tagdata.Base.Description = fmt.Sprintf("Liste an Laufveranstaltungen im Raum Freiburg, die mit der Kategorie '%s' getaggt sind", tag.Name)
		slug := tag.Slug()
		tagdata.Base.Canonical = fmt.Sprintf("%s/%s", baseUrl, slug)
		tagdata.Base.Breadcrumbs = utils.PushBreadcrumb(breadcrumbsEventsTags, utils.CreateLink(tag.Name, fmt.Sprintf("/%s", slug)))
		utils.ExecuteTemplate("tag", out.Join(slug), tagdata)
		sitemap.Add(slug, tag.Name, "Kategorien")
	}

	// special rendering of the "traillauf" tag
	for _, tag := range eventsData.Tags {
		if tag.Sanitized == "traillauf" {
			tagdata.Tag = tag
			tagdata.Base.Title = "Aktuelle Trail-Veranstaltungen"
			tagdata.Base.Description = "Aktuelle Trail-Veranstaltungen im Raum Freiburg"
			slug := "trailrun-frame.html"
			tagdata.Base.Canonical = fmt.Sprintf("%s/%s", baseUrl, slug)
			utils.ExecuteTemplate("trailrun-frame", out.Join(slug), tagdata)
		}
	}

	seriedata := SerieTemplateData{
		nil,
		TemplateData{
			"",
			"Serie",
			"",
			"series",
			"",
			breadcrumbsEventsSeries,
			"/series.html",
			timestamp,
			timestampFull,
			sheetUrl,
			&eventsData,
			js_files.Rel(options.outDir),
			css_files.Rel(options.outDir),
			umami,
		},
	}
	for _, s := range eventsData.Series {
		seriedata.Serie = s
		seriedata.Base.Title = s.Name
		seriedata.Base.Description = fmt.Sprintf("Lauf-Serie '%s'", s.Name)
		slug := s.Slug()
		seriedata.Base.Canonical = fmt.Sprintf("%s/%s", baseUrl, slug)
		seriedata.Base.Breadcrumbs = utils.PushBreadcrumb(breadcrumbsEventsSeries, utils.CreateLink(s.Name, fmt.Sprintf("/%s", slug)))
		utils.ExecuteTemplate("serie", out.Join(slug), seriedata)
		sitemap.Add(slug, s.Name, "Serien")
	}
	for _, s := range eventsData.SeriesOld {
		seriedata.Serie = s
		seriedata.Base.Title = s.Name
		seriedata.Base.Description = fmt.Sprintf("Lauf-Serie '%s'", s.Name)
		slug := s.Slug()
		seriedata.Base.Canonical = fmt.Sprintf("%s/%s", baseUrl, slug)
		seriedata.Base.Breadcrumbs = utils.PushBreadcrumb(breadcrumbsEventsSeries, utils.CreateLink(s.Name, fmt.Sprintf("/%s", slug)))
		utils.ExecuteTemplate("serie", out.Join(slug), seriedata)
		sitemap.Add(slug, s.Name, "Serien")
	}

	sitemap.Gen(out.Join("sitemap.xml"), options.hashFile, options.outDir)
	sitemapTemplate := SitemapTemplateData{
		sitemap.GenHTML(),
		TemplateData{
			"Sitemap von freiburg.run",
			"",
			"Sitemap von freiburg.run",
			"",
			fmt.Sprintf("%s/sitemap.html", baseUrl),
			utils.PushBreadcrumb(breadcrumbsBase, utils.CreateLink("Sitemap", "/sitemap.html")),
			"/",
			timestamp,
			timestampFull,
			sheetUrl,
			&eventsData,
			js_files.Rel(options.outDir),
			css_files.Rel(options.outDir),
			umami,
		},
	}
	utils.ExecuteTemplate("sitemap", out.Join("sitemap.html"), sitemapTemplate)

	err = CreateHtaccess(eventsData, options.outDir)
	utils.Check(err)

	err = events.CreateCalendar(eventsData.Events, now, baseUrl, fmt.Sprintf("%s/events.ics", baseUrl), out.Join("events.ics"))
	utils.Check(err)
}
