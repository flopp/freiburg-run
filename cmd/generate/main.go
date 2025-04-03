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
	"github.com/flopp/freiburg-run/internal/resources"
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

type CommonData struct {
	Timestamp     string
	TimestampFull string
	SheetUrl      string
	Data          *events.Data
	JsFiles       []string
	CssFiles      []string
	Umami         UmamiData
}

type TemplateData struct {
	CommonData
	Title       string
	Type        string
	Description string
	Nav         string
	Canonical   string
	Breadcrumbs utils.Breadcrumbs
	Main        string
}

func (t *TemplateData) SetNameLink(name, link string, baseBreakcrumbs utils.Breadcrumbs, baseUrl utils.Url) {
	t.Title = name
	t.Canonical = baseUrl.Join(link)
	t.Breadcrumbs = baseBreakcrumbs.Push(utils.CreateLink(name, "/"+link))
}

func (t TemplateData) Image() string {
	if t.Nav == "parkrun" {
		return "https://freiburg.run/images/parkrun.png"
	}
	return "https://freiburg.run/images/512.png"
}

func (t TemplateData) NiceTitle() string {
	return t.Title
}

func (t TemplateData) CountEvents() int {
	count := 0
	for _, event := range t.Data.Events {
		if !event.IsSeparator() {
			count += 1
		}
	}
	return count
}

type EventTemplateData struct {
	TemplateData
	Event *events.Event
}

func (d EventTemplateData) NiceTitle() string {
	if d.Event.Type != "event" {
		return d.Title
	}

	if d.Event.Time.IsZero() {
		return d.Title
	}

	yearS := fmt.Sprintf("%d", d.Event.Time.Year())
	if strings.Contains(d.Title, yearS) {
		return d.Title
	}

	return fmt.Sprintf("%s %s", d.Title, yearS)
}

type TagTemplateData struct {
	TemplateData
	Tag *events.Tag
}

func (d TagTemplateData) NiceTitle() string {
	return d.Title
}

type SerieTemplateData struct {
	TemplateData
	Serie *events.Serie
}

func (d SerieTemplateData) NiceTitle() string {
	return d.Title
}

type EmbedListTemplateData struct {
	TemplateData
	Events []*events.Event
}

type SitemapTemplateData struct {
	TemplateData
	Categories []utils.SitemapCategory
}

func (d SitemapTemplateData) NiceTitle() string {
	return d.Title
}

func GetMtimeYMD(filePath string) string {
	t, err := utils.GetMtime(filePath)
	if err != nil {
		return ""
	}

	return t.Format("2006-01-02")
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

func MustRel(basepath, path string) string {
	rel, err := filepath.Rel(basepath, path)
	utils.Check(err)
	return rel
}

type Path string

func (p Path) Join(s string) string {
	return filepath.Join(string(p), s)
}

func downloadHash(url string, targetBase Path, targetFile string) string {
	target := targetBase.Join(targetFile)
	res := utils.MustDownloadHash(url, target)
	return MustRel(string(targetBase), res)
}

func copyHash(sourcePath string, targetBase Path, targetFile string) string {
	res := utils.MustCopyHash(sourcePath, targetBase.Join(targetFile))
	return MustRel(string(targetBase), res)
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

type CountryData struct {
	slug   string
	events []*events.Event
}

func renderEmbedList(baseUrl utils.Url, out Path, data TemplateData, tag *events.Tag) {
	countryData := map[string]*CountryData{
		"":           &CountryData{"embed/trailrun-de.html", make([]*events.Event, 0)}, // Default (Germany)
		"Frankreich": &CountryData{"embed/trailrun-fr.html", make([]*events.Event, 0)},
		"Schweiz":    &CountryData{"embed/trailrun-ch.html", make([]*events.Event, 0)},
	}

	// Distribute events into the appropriate country-specific data
	for _, event := range tag.Events {
		if event.IsSeparator() {
			continue
		}
		if d, ok := countryData[event.Location.Country]; ok {
			d.events = append(d.events, event)
		} else {
			panic(fmt.Errorf("Country '%s' not found in countrySlugs", event.Location.Country))
		}
	}

	// Render templates for each country
	for _, d := range countryData {
		t := EmbedListTemplateData{
			TemplateData: data,
			Events:       d.events,
		}
		t.Canonical = baseUrl.Join(d.slug)
		utils.ExecuteTemplate("embed-list", out.Join(d.slug), t)
	}
}

func main() {
	baseUrl := utils.Url("https://freiburg.run")
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

	resourceManager := resources.NewResourceManager(string(out))
	resourceManager.CopyExternalAssets()
	resourceManager.CopyStaticAssets()

	breadcrumbsBase := utils.InitBreadcrumbs(utils.CreateLink("freiburg.run", "/"))
	breadcrumbsEvents := breadcrumbsBase.Push(utils.CreateLink("Laufveranstaltungen", "/"))

	commondata := CommonData{
		timestamp,
		timestampFull,
		sheetUrl,
		&eventsData,
		resourceManager.JsFiles,
		resourceManager.CssFiles,
		UmamiData{
			resourceManager.UmamiScript,
			"6609164f-5e79-4041-b1ed-f37da10a84d2",
		},
	}

	data := TemplateData{
		commondata,
		"Laufveranstaltungen im Raum Freiburg",
		"Veranstaltung",
		"Liste von aktuellen und zukünftigen Laufveranstaltungen, Lauf-Wettkämpfen, Volksläufen im Raum Freiburg",
		"events",
		string(baseUrl),
		breadcrumbsEvents,
		"/",
	}

	utils.ExecuteTemplate("events", out.Join("index.html"), data)

	slug := "events-old.html"
	data.Title = "Vergangene Laufveranstaltungen im Raum Freiburg "
	data.Description = "Liste von vergangenen Laufveranstaltungen, Lauf-Wettkämpfen, Volksläufen im Raum Freiburg "
	data.Canonical = baseUrl.Join(slug)
	breadcrumbsEventsOld := breadcrumbsEvents.Push(utils.CreateLink("Archiv", "/"+slug))
	data.Breadcrumbs = breadcrumbsEventsOld
	utils.ExecuteTemplate("events-old", out.Join(slug), data)

	slug = "tags.html"
	data.Nav = "tags"
	data.Title = "Kategorien"
	data.Description = "Liste aller Kategorien von Laufveranstaltungen, Lauf-Wettkämpfen, Volksläufen im Raum Freiburg"
	data.Canonical = baseUrl.Join(slug)
	breadcrumbsEventsTags := breadcrumbsEvents.Push(utils.CreateLink(data.Title, "/"+slug))
	data.Breadcrumbs = breadcrumbsEventsTags
	utils.ExecuteTemplate("tags", out.Join(slug), data)

	slug = "lauftreffs.html"
	data.Nav = "groups"
	data.Title = "Lauftreffs im Raum Freiburg"
	data.Type = "Lauftreff"
	data.Description = "Liste von Lauftreffs, Laufgruppen, Lauf-Trainingsgruppen im Raum Freiburg"
	data.Canonical = baseUrl.Join(slug)
	breadcrumbsGroups := breadcrumbsBase.Push(utils.CreateLink("Lauftreffs", "/"+slug))
	data.Breadcrumbs = breadcrumbsGroups
	utils.ExecuteTemplate("groups", out.Join(slug), data)

	slug = "shops.html"
	data.Nav = "shops"
	data.Title = "Lauf-Shops im Raum Freiburg"
	data.Type = "Lauf-Shop"
	data.Description = "Liste von Lauf-Shops und Einzelhandelsgeschäften mit Laufschuh-Auswahl im Raum Freiburg"
	data.Canonical = baseUrl.Join(slug)
	breadcrumbsShops := breadcrumbsBase.Push(utils.CreateLink("Lauf-Shops", "/"+slug))
	data.Breadcrumbs = breadcrumbsShops
	utils.ExecuteTemplate("shops", out.Join(slug), data)

	slug = "dietenbach-parkrun.html"
	data.Nav = "parkrun"
	data.Type = "Dietenbach parkrun"
	data.Description = "Vollständige Liste aller Ergebnisse, Laufberichte und Fotogalerien des 'Dietenbach parkrun' im Freiburger Dietenbachpark."
	data.SetNameLink("Dietenbach parkrun", slug, breadcrumbsBase, baseUrl)
	utils.ExecuteTemplate("dietenbach-parkrun", out.Join(slug), data)
	utils.ExecuteTemplateNoMinify("dietenbach-parkrun-wordpress", out.Join("dietenbach-parkrun-wordpress.html"), data)

	slug = "series.html"
	data.Nav = "series"
	data.Title = "Serien"
	data.Description = "Liste aller Serien von Laufveranstaltungen, Lauf-Wettkämpfen, Volksläufen im Raum Freiburg "
	data.Canonical = baseUrl.Join(slug)
	breadcrumbsEventsSeries := breadcrumbsEvents.Push(utils.CreateLink(data.Title, "/"+slug))
	data.Breadcrumbs = breadcrumbsEventsSeries
	utils.ExecuteTemplate("series", out.Join(slug), data)

	slug = "map.html"
	data.Nav = "map"
	data.Title = "Karte aller Laufveranstaltungen"
	data.Type = "Karte"
	data.Description = "Karte"
	data.Canonical = baseUrl.Join(slug)
	data.Breadcrumbs = breadcrumbsBase.Push(utils.CreateLink(data.Title, "/"+slug))
	utils.ExecuteTemplate("map", out.Join(slug), data)

	slug = "info.html"
	data.Nav = "info"
	data.Title = "Info"
	data.Type = "Info"
	data.Description = "Kontaktmöglichkeiten, allgemeine & technische Informationen über freiburg.run"
	data.Canonical = baseUrl.Join(slug)
	breadcrumbsInfo := breadcrumbsBase.Push(utils.CreateLink(data.Title, "/"+slug))
	data.Breadcrumbs = breadcrumbsInfo
	utils.ExecuteTemplate("info", out.Join(slug), data)

	slug = "datenschutz.html"
	data.Nav = "datenschutz"
	data.Title = "Datenschutz"
	data.Type = "Datenschutz"
	data.Description = "Datenschutzerklärung von freiburg.run"
	data.Canonical = baseUrl.Join(slug)
	data.Breadcrumbs = breadcrumbsInfo.Push(utils.CreateLink(data.Title, "/"+slug))
	utils.ExecuteTemplate("datenschutz", out.Join(slug), data)

	slug = "impressum.html"
	data.Nav = "impressum"
	data.Title = "Impressum"
	data.Type = "Impressum"
	data.Description = "Impressum von freiburg.run"
	data.Canonical = baseUrl.Join(slug)
	data.Breadcrumbs = breadcrumbsInfo.Push(utils.CreateLink(data.Title, "/"+slug))
	utils.ExecuteTemplate("impressum", out.Join(slug), data)

	slug = "support.html"
	data.Nav = "support"
	data.Title = "freiburg.run unterstützen"
	data.Type = "Support"
	data.Description = "Möglichkeiten freiburg.run zu unterstützen"
	data.Canonical = baseUrl.Join(slug)
	data.Breadcrumbs = breadcrumbsInfo.Push(utils.CreateLink(data.Title, "/"+slug))
	utils.ExecuteTemplate("support", out.Join(slug), data)

	slug = "404.html"
	data.Nav = "404"
	data.Title = "404 - Seite nicht gefunden :("
	data.Type = ""
	data.Description = "Fehlerseite von freiburg.run"
	data.Canonical = baseUrl.Join(slug)
	data.Breadcrumbs = breadcrumbsBase.Push(utils.CreateLink(data.Title, "/"+slug))
	utils.ExecuteTemplate("404", out.Join(slug), data)

	eventdata := EventTemplateData{
		TemplateData{
			commondata,
			"",
			"Veranstaltung",
			"",
			"events",
			"",
			breadcrumbsEvents,
			"/",
		},
		nil,
	}
	for _, event := range eventsData.Events {
		if event.IsSeparator() {
			continue
		}
		eventdata.Event = event
		eventdata.Description = event.GenerateDescription()
		slug := event.Slug()
		eventdata.SetNameLink(event.Name, slug, breadcrumbsEvents, baseUrl)
		utils.ExecuteTemplate("event", out.Join(slug), eventdata)
		sitemap.Add(slug, event.Name, "Laufveranstaltungen")
	}

	eventdata.Main = "/events-old.html"
	for _, event := range eventsData.EventsOld {
		if event.IsSeparator() {
			continue
		}
		eventdata.Event = event
		eventdata.Description = event.GenerateDescription()
		slug := event.Slug()
		eventdata.SetNameLink(event.Name, slug, breadcrumbsEventsOld, baseUrl)
		utils.ExecuteTemplate("event", out.Join(slug), eventdata)
		sitemap.Add(slug, event.Name, "Vergangene Laufveranstaltungen")
	}

	eventdata.Type = "Lauftreff"
	eventdata.Nav = "groups"
	eventdata.Main = "/lauftreffs.html"
	for _, event := range eventsData.Groups {
		eventdata.Event = event
		eventdata.Description = event.GenerateDescription()
		slug := event.Slug()
		eventdata.SetNameLink(event.Name, slug, breadcrumbsGroups, baseUrl)
		utils.ExecuteTemplate("event", out.Join(slug), eventdata)
		sitemap.Add(slug, event.Name, "Lauftreffs")
	}

	eventdata.Type = "Lauf-Shop"
	eventdata.Nav = "shops"
	eventdata.Main = "/shops.html"
	for _, event := range eventsData.Shops {
		eventdata.Event = event
		eventdata.Description = event.GenerateDescription()
		slug := event.Slug()
		eventdata.SetNameLink(event.Name, slug, breadcrumbsShops, baseUrl)
		utils.ExecuteTemplate("event", out.Join(slug), eventdata)
		sitemap.Add(slug, event.Name, "Lauf-Shops")
	}

	tagdata := TagTemplateData{
		TemplateData{
			commondata,
			"",
			"Kategorie",
			"",
			"tags",
			"",
			breadcrumbsEventsTags,
			"/tags.html",
		},
		nil,
	}
	for _, tag := range eventsData.Tags {
		tagdata.Tag = tag
		tagdata.Description = fmt.Sprintf("Liste an Laufveranstaltungen im Raum Freiburg, die mit der Kategorie '%s' getaggt sind", tag.Name)
		slug := tag.Slug()
		tagdata.SetNameLink(tag.Name, slug, breadcrumbsEventsTags, baseUrl)
		tagdata.Title = fmt.Sprintf("Laufveranstaltungen der Kategorie '%s'", tag.Name)
		utils.ExecuteTemplate("tag", out.Join(slug), tagdata)
		sitemap.Add(slug, tag.Name, "Kategorien")
	}

	// Special rendering of the "traillauf" tag
	for _, tag := range eventsData.Tags {
		if tag.Sanitized == "traillauf" {
			renderEmbedList(baseUrl, out, data, tag)
			break
		}
	}

	seriedata := SerieTemplateData{
		TemplateData{
			commondata,
			"",
			"Serie",
			"",
			"series",
			"",
			breadcrumbsEventsSeries,
			"/series.html",
		},
		nil,
	}
	for _, s := range eventsData.Series {
		seriedata.Serie = s
		seriedata.Description = fmt.Sprintf("Lauf-Serie '%s'", s.Name)
		slug := s.Slug()
		seriedata.SetNameLink(s.Name, slug, breadcrumbsEventsSeries, baseUrl)
		utils.ExecuteTemplate("serie", out.Join(slug), seriedata)
		sitemap.Add(slug, s.Name, "Serien")
	}
	for _, s := range eventsData.SeriesOld {
		seriedata.Serie = s
		seriedata.Description = fmt.Sprintf("Lauf-Serie '%s'", s.Name)
		slug := s.Slug()
		seriedata.SetNameLink(s.Name, slug, breadcrumbsEventsSeries, baseUrl)
		utils.ExecuteTemplate("serie", out.Join(slug), seriedata)
		sitemap.Add(slug, s.Name, "Serien")
	}

	sitemap.Gen(out.Join("sitemap.xml"), options.hashFile, options.outDir)
	sitemapTemplate := SitemapTemplateData{
		TemplateData{
			commondata,
			"Sitemap von freiburg.run",
			"",
			"Sitemap von freiburg.run",
			"",
			fmt.Sprintf("%s/sitemap.html", baseUrl),
			breadcrumbsBase.Push(utils.CreateLink("Sitemap", "/sitemap.html")),
			"/",
		},
		sitemap.GenHTML(),
	}
	utils.ExecuteTemplate("sitemap", out.Join("sitemap.html"), sitemapTemplate)

	err = CreateHtaccess(eventsData, options.outDir)
	utils.Check(err)

	err = events.CreateCalendar(eventsData.Events, now, baseUrl, fmt.Sprintf("%s/events.ics", baseUrl), out.Join("events.ics"))
	utils.Check(err)
}
