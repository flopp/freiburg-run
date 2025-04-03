package generator

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/flopp/freiburg-run/internal/events"
	"github.com/flopp/freiburg-run/internal/resources"
	"github.com/flopp/freiburg-run/internal/utils"
)

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

func createHtaccess(data events.Data, outDir utils.Path) error {
	if err := utils.MakeDir(outDir.String()); err != nil {
		return err
	}

	fileName := outDir.Join(".htaccess")

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

type CountryData struct {
	slug   string
	events []*events.Event
}

func renderEmbedList(baseUrl utils.Url, out utils.Path, data TemplateData, tag *events.Tag) error {
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
			return fmt.Errorf("Country '%s' not found in countrySlugs", event.Location.Country)
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

	return nil
}

type Generator struct {
	out           utils.Path
	baseUrl       utils.Url
	now           time.Time
	timestamp     string
	timestampFull string
	jsFiles       []string
	cssFiles      []string
	umamiScript   string
	umamiId       string
	sheetUrl      string
	hashFile      string
}

func NewGenerator(
	out utils.Path, baseUrl utils.Url, now time.Time,
	jsFiles []string, cssFiles []string,
	umamiScript string, umamiId string,
	sheetUrl string,
	hashFile string,
) Generator {
	return Generator{
		out:           out,
		baseUrl:       baseUrl,
		now:           now,
		timestamp:     now.Format("2006-01-02"),
		timestampFull: now.Format("2006-01-02 15:04:05"),
		jsFiles:       jsFiles,
		cssFiles:      cssFiles,
		umamiScript:   umamiScript,
		umamiId:       umamiId,
		sheetUrl:      sheetUrl,
		hashFile:      hashFile,
	}
}

func (g Generator) Generate(eventsData events.Data) error {
	// create ics files for events
	// Helper function to create calendar files for a slice of events
	createCalendarsForEvents := func(eventList []*events.Event) error {
		for _, event := range eventList {
			if event.IsSeparator() {
				continue
			}
			calendar := event.CalendarSlug()
			if err := events.CreateEventCalendar(event, g.now, g.baseUrl, g.baseUrl.Join(calendar), g.out.Join(calendar)); err != nil {
				return fmt.Errorf("create event calendar: %v", err)
			}
			event.Calendar = "/" + calendar
		}
		return nil
	}

	// Create calendar files for current and past events
	if err := createCalendarsForEvents(eventsData.Events); err != nil {
		return err
	}
	if err := createCalendarsForEvents(eventsData.EventsOld); err != nil {
		return err
	}

	// Create calendar files for all upcoming events
	if err := events.CreateCalendar(eventsData.Events, g.now, g.baseUrl, g.baseUrl.Join("events.ics"), g.out.Join("events.ics")); err != nil {
		return fmt.Errorf("create events.ics: %v", err)
	}

	sitemap := utils.CreateSitemap(g.baseUrl)
	sitemap.AddCategory("Allgemein")
	sitemap.AddCategory("Laufveranstaltungen")
	sitemap.AddCategory("Vergangene Laufveranstaltungen")
	sitemap.AddCategory("Kategorien")
	sitemap.AddCategory("Serien")
	sitemap.AddCategory("Lauftreffs")
	sitemap.AddCategory("Lauf-Shops")

	resourceManager := resources.NewResourceManager(string(g.out))
	resourceManager.CopyExternalAssets()
	resourceManager.CopyStaticAssets()

	breadcrumbsBase := utils.InitBreadcrumbs(utils.CreateLink("freiburg.run", "/"))
	breadcrumbsEvents := breadcrumbsBase.Push(utils.CreateLink("Laufveranstaltungen", "/"))
	breadcrumbsEventsOld := breadcrumbsEvents.Push(utils.CreateLink("Archiv", "/events-old.html"))
	breadcrumbsTags := breadcrumbsEvents.Push(utils.CreateLink("Kategoriene", "/tags.html"))
	breadcrumbsGroups := breadcrumbsBase.Push(utils.CreateLink("Lauftreffs", "/lauftreffs.html"))
	breadcrumbsShops := breadcrumbsBase.Push(utils.CreateLink("Lauf-Shops", "/shops.html"))
	breadcrumbsSeries := breadcrumbsEvents.Push(utils.CreateLink("Serien", "/series.html"))
	breadcrumbsInfo := breadcrumbsBase.Push(utils.CreateLink("Info", "/info.html"))

	commondata := CommonData{
		g.timestamp,
		g.timestampFull,
		g.sheetUrl,
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
		"Liste von aktuellen und zukünftigen Laufveranstaltungen, Lauf-Wettkämpfen, Volksläufen im Raum Freiburg",
		"events",
		string(g.baseUrl),
		breadcrumbsEvents,
		"/",
	}

	// Render general pages
	renderPage := func(slug, template, nav, sitemapCategory, title, description string, breadcrumbs utils.Breadcrumbs) {
		data := TemplateData{
			commondata,
			title,
			description,
			nav,
			g.baseUrl.Join(slug),
			breadcrumbs,
			"/",
		}
		if slug == "" {
			utils.ExecuteTemplate(template, g.out.Join("index.html"), data)
		} else {
			utils.ExecuteTemplate(template, g.out.Join(slug), data)
		}
		if template != "404" {
			sitemap.Add(slug, title, sitemapCategory)
		}
		sitemap.Add(slug, title, sitemapCategory)
	}
	renderSubPage := func(slug, template, nav, sitemapCategory, title, description string, breadcrumbs utils.Breadcrumbs) {
		renderPage(slug, template, nav, sitemapCategory, title, description, breadcrumbs.Push(utils.CreateLink(data.Title, "/"+slug)))
	}

	renderPage("", "events", "events", "Laufveranstaltungen",
		"Laufveranstaltungen im Raum Freiburg",
		"Liste von Laufveranstaltungen, Lauf-Wettkämpfen, Volksläufen im Raum Freiburg",
		breadcrumbsEvents)

	renderPage("events-old.html", "events-old", "events", "Vergangene Laufveranstaltungen",
		"Vergangene Laufveranstaltungen im Raum Freiburg",
		"Liste von vergangenen Laufveranstaltungen, Lauf-Wettkämpfen, Volksläufen im Raum Freiburg",
		breadcrumbsEventsOld)

	renderPage("tags.html", "tags", "tags", "Kategorien",
		"Kategorien",
		"Liste aller Kategorien von Laufveranstaltungen, Lauf-Wettkämpfen, Volksläufen im Raum Freiburg",
		breadcrumbsTags)

	renderPage("lauftreffs.html", "groups", "groups", "Lauftreffs",
		"Lauftreffs im Raum Freiburg",
		"Liste von Lauftreffs, Laufgruppen, Lauf-Trainingsgruppen im Raum Freiburg",
		breadcrumbsGroups)

	renderPage("shops.html", "shops", "shops", "Lauf-Shops",
		"Lauf-Shops im Raum Freiburg",
		"Liste von Lauf-Shops und Einzelhandelsgeschäften mit Laufschuh-Auswahl im Raum Freiburg",
		breadcrumbsShops)

	renderSubPage("dietenbach-parkrun.html", "dietenbach-parkrun", "parkrun", "Allgemein",
		"Dietenbach parkrun",
		"Vollständige Liste aller Ergebnisse, Laufberichte und Fotogalerien des 'Dietenbach parkrun' im Freiburger Dietenbachpark.",
		breadcrumbsBase)

	renderPage("series.html", "series", "series", "Serien",
		"Lauf-Serien",
		"Liste aller Serien von Laufveranstaltungen, Lauf-Wettkämpfen, Volksläufen im Raum Freiburg",
		breadcrumbsSeries)

	renderSubPage("map.html", "map", "map", "Allgemein",
		"Karte aller Laufveranstaltungen",
		"Karte",
		breadcrumbsBase)

	renderPage("info.html", "info", "info", "Allgemein",
		"Info",
		"Kontaktmöglichkeiten, allgemeine & technische Informationen über freiburg.run",
		breadcrumbsInfo)

	renderSubPage("datenschutz.html", "datenschutz", "datenschutz", "Allgemein",
		"Datenschutz",
		"Datenschutzerklärung von freiburg.run",
		breadcrumbsInfo)

	renderSubPage("impressum.html", "impressum", "impressum", "Allgemein",
		"Impressum",
		"Impressum von freiburg.run",
		breadcrumbsInfo)

	renderSubPage("support.html", "support", "support", "Allgemein",
		"freiburg.run unterstützen",
		"Möglichkeiten freiburg.run zu unterstützen",
		breadcrumbsInfo)

	renderSubPage("404.html", "404", "404", "",
		"404 - Seite nicht gefunden :(",
		"Fehlerseite von freiburg.run",
		breadcrumbsBase)

	// Special rendering of parkrun page for wordpress
	utils.ExecuteTemplateNoMinify("dietenbach-parkrun-wordpress", g.out.Join("dietenbach-parkrun-wordpress.html"), data)

	// Render events, groups, shops lists
	renderEventList := func(eventList []*events.Event, nav, main, sitemapCategory string, breadcrumbs utils.Breadcrumbs) {
		eventdata := EventTemplateData{
			TemplateData{
				commondata,
				"",
				"",
				nav,
				"",
				breadcrumbs,
				main,
			},
			nil,
		}
		for _, event := range eventList {
			if event.IsSeparator() {
				continue
			}
			eventdata.Event = event
			eventdata.Description = event.GenerateDescription()
			slug := event.Slug()
			eventdata.SetNameLink(event.Name, slug, breadcrumbsEvents, g.baseUrl)
			utils.ExecuteTemplate("event", g.out.Join(slug), eventdata)
			sitemap.Add(slug, event.Name, sitemapCategory)
		}
	}
	renderEventList(eventsData.Events, "events", "/", "Laufveranstaltungen", breadcrumbsEvents)
	renderEventList(eventsData.EventsOld, "events", "/events-old.html", "Vergangene Laufveranstaltungen", breadcrumbsEventsOld)
	renderEventList(eventsData.Groups, "groups", "/lauftreffs.html", "Lauftreffs", breadcrumbsGroups)
	renderEventList(eventsData.Shops, "shops", "/shops.html", "Lauf-Shops", breadcrumbsShops)

	// Render tags
	tagdata := TagTemplateData{
		TemplateData{
			commondata,
			"",
			"",
			"tags",
			"",
			breadcrumbsTags,
			"/tags.html",
		},
		nil,
	}
	for _, tag := range eventsData.Tags {
		tagdata.Tag = tag
		tagdata.Description = fmt.Sprintf("Liste an Laufveranstaltungen im Raum Freiburg, die mit der Kategorie '%s' getaggt sind", tag.Name)
		slug := tag.Slug()
		tagdata.SetNameLink(tag.Name, slug, breadcrumbsTags, g.baseUrl)
		tagdata.Title = fmt.Sprintf("Laufveranstaltungen der Kategorie '%s'", tag.Name)
		utils.ExecuteTemplate("tag", g.out.Join(slug), tagdata)
		sitemap.Add(slug, tag.Name, "Kategorien")
	}

	// Special rendering of the "traillauf" tag
	for _, tag := range eventsData.Tags {
		if tag.Sanitized == "traillauf" {
			if err := renderEmbedList(g.baseUrl, g.out, data, tag); err != nil {
				return fmt.Errorf("create embed lists: %v", err)
			}
			break
		}
	}

	// Render series
	renderSeries := func(series []*events.Serie) {
		seriedata := SerieTemplateData{
			TemplateData{
				commondata,
				"",
				"",
				"series",
				"",
				breadcrumbsSeries,
				"/series.html",
			},
			nil,
		}
		for _, s := range series {
			seriedata.Serie = s
			seriedata.Description = fmt.Sprintf("Lauf-Serie '%s'", s.Name)
			slug := s.Slug()
			seriedata.SetNameLink(s.Name, slug, breadcrumbsSeries, g.baseUrl)
			utils.ExecuteTemplate("serie", g.out.Join(slug), seriedata)
			sitemap.Add(slug, s.Name, "Serien")
		}
	}
	renderSeries(eventsData.Series)
	renderSeries(eventsData.SeriesOld)

	// Render sitemap
	sitemap.Gen(g.out.Join("sitemap.xml"), g.hashFile, g.out)
	sitemapTemplate := SitemapTemplateData{
		TemplateData{
			commondata,
			"Sitemap von freiburg.run",
			"Sitemap von freiburg.run",
			"",
			fmt.Sprintf("%s/sitemap.html", g.baseUrl),
			breadcrumbsBase.Push(utils.CreateLink("Sitemap", "/sitemap.html")),
			"/",
		},
		sitemap.GenHTML(),
	}
	utils.ExecuteTemplate("sitemap", g.out.Join("sitemap.html"), sitemapTemplate)

	// Render .htaccess
	if err := createHtaccess(eventsData, g.out); err != nil {
		return fmt.Errorf("create .htaccess: %v", err)
	}

	return nil
}
