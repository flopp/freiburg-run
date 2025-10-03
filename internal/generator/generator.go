package generator

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/flopp/freiburg-run/internal/config"
	"github.com/flopp/freiburg-run/internal/events"
	"github.com/flopp/freiburg-run/internal/resources"
	"github.com/flopp/freiburg-run/internal/utils"
)

type UmamiData struct {
	Url string
	Id  string
}

type CommonData struct {
	Config          config.Config
	Timestamp       string
	TimestampFull   string
	BaseUrl         string
	BasePath        string
	FeedbackFormUrl string // URL for feedback form
	Data            *events.Data
	JsFiles         []string
	CssFiles        []string
	Umami           UmamiData
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

type OldEventsTemplateData struct {
	TemplateData
	Year   string
	Years  []*utils.Link
	Events []*events.Event
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
	destination.WriteString("Redirect /tag/2023.html /events-old-2023.html\n")
	destination.WriteString("Redirect /tag/2024.html /events-old-2024.html\n")
	destination.WriteString("Redirect /tag/2025.html /events-old.html\n")
	destination.WriteString("Redirect /tag/2026.html /\n")

	for _, e := range data.Events {
		slug := e.Slug()
		if old := e.SlugOld(); old != "" {
			destination.WriteString(fmt.Sprintf("Redirect /%s /%s\n", old, slug))
		}
		if slugNoBase := e.SlugNoBase(); slugNoBase != slug {
			destination.WriteString(fmt.Sprintf("Redirect /%s /%s\n", slugNoBase, slug))
		}
	}
	for _, e := range data.EventsOld {
		slug := e.Slug()
		if old := e.SlugOld(); old != "" {
			destination.WriteString(fmt.Sprintf("Redirect /%s /%s\n", old, slug))
		}
		if slugNoBase := e.SlugNoBase(); slugNoBase != slug {
			destination.WriteString(fmt.Sprintf("Redirect /%s /%s\n", slugNoBase, slug))
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
		"":           {"embed/trailrun-de.html", make([]*events.Event, 0)}, // Default (Germany)
		"Frankreich": {"embed/trailrun-fr.html", make([]*events.Event, 0)},
		"Schweiz":    {"embed/trailrun-ch.html", make([]*events.Event, 0)},
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
		if err := utils.ExecuteTemplate("embed-list", out.Join(d.slug), t.BasePath, t); err != nil {
			return fmt.Errorf("render embed list for %q: %w", d.slug, err)
		}
	}

	return nil
}

type Generator struct {
	config          config.Config
	out             utils.Path
	baseUrl         utils.Url
	basePath        string
	now             time.Time
	timestamp       string
	timestampFull   string
	jsFiles         []string
	cssFiles        []string
	umamiScript     string
	umamiId         string
	feedbackFormUrl string
	hashFile        string
}

func NewGenerator(
	config config.Config,
	out utils.Path,
	baseUrl utils.Url, basePath string,
	now time.Time,
	jsFiles []string, cssFiles []string,
	umamiScript string, umamiId string,
	feedbackFormUrl string,
	hashFile string,
) Generator {
	return Generator{
		config:          config,
		out:             out,
		baseUrl:         baseUrl,
		basePath:        basePath,
		now:             now,
		timestamp:       now.Format("2006-01-02"),
		timestampFull:   now.Format("2006-01-02 15:04:05"),
		jsFiles:         jsFiles,
		cssFiles:        cssFiles,
		umamiScript:     umamiScript,
		umamiId:         umamiId,
		feedbackFormUrl: feedbackFormUrl,
		hashFile:        hashFile,
	}
}

func (g Generator) Generate(eventsData events.Data) error {
	// Prepare assets
	resourceManager := resources.NewResourceManager(".", string(g.out))
	resourceManager.CopyExternalAssets()
	resourceManager.CopyStaticAssets()

	// create ics files for events
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
	if err := createCalendarsForEvents(eventsData.Events); err != nil {
		return err
	}
	/*
		if err := createCalendarsForEvents(eventsData.EventsOld); err != nil {
			return err
		}
	*/

	// Create calendar files for all upcoming events
	if err := events.CreateCalendar(eventsData.Events, g.now, g.baseUrl, g.baseUrl.Join("events.ics"), g.out.Join("events.ics")); err != nil {
		return fmt.Errorf("create events.ics: %v", err)
	}

	sitemap := utils.CreateSitemap(g.baseUrl)
	sitemap.AddCategory("Allgemein")
	sitemap.AddCategory("Club")
	sitemap.AddCategory("Laufveranstaltungen")
	sitemap.AddCategory("Vergangene Laufveranstaltungen")
	sitemap.AddCategory("Kategorien")
	sitemap.AddCategory("Serien")
	sitemap.AddCategory("Lauftreffs")
	sitemap.AddCategory("Lauf-Shops")

	breadcrumbsBase := utils.InitBreadcrumbs(utils.CreateLink("freiburg.run", "/"))
	breadcrumbsEvents := breadcrumbsBase.Push(utils.CreateLink("Laufveranstaltungen", "/"))
	breadcrumbsTags := breadcrumbsEvents.Push(utils.CreateLink("Kategorien", "/tags.html"))
	breadcrumbsSeries := breadcrumbsEvents.Push(utils.CreateLink("Serien", "/series.html"))
	breadcrumbsGroups := breadcrumbsBase.Push(utils.CreateLink("Lauftreffs", "/lauftreffs.html"))
	breadcrumbsShops := breadcrumbsBase.Push(utils.CreateLink("Lauf-Shops", "/shops.html"))
	breadcrumbsInfo := breadcrumbsBase.Push(utils.CreateLink("Info", "/info.html"))

	commondata := CommonData{
		g.config,
		g.timestamp,
		g.timestampFull,
		string(g.baseUrl),
		g.basePath,
		g.feedbackFormUrl,
		&eventsData,
		resourceManager.JsFiles,
		resourceManager.CssFiles,
		UmamiData{
			resourceManager.UmamiScript,
			g.umamiId,
		},
	}

	// Render general pages
	renderPage := func(slug, slugFile, template, nav, sitemapCategory, title, description string, breadcrumbs utils.Breadcrumbs) error {
		data := TemplateData{
			commondata,
			title,
			description,
			nav,
			g.baseUrl.Join(slug),
			breadcrumbs,
			"/",
		}
		if err := utils.ExecuteTemplate(template, g.out.Join(slugFile), data.BasePath, data); err != nil {
			return fmt.Errorf("render template %q to %q: %w", template, g.out.Join(slugFile), err)
		}
		if template != "404" {
			sitemap.Add(slug, slugFile, title, sitemapCategory)
		}
		return nil
	}
	renderSubPage := func(slug, slugFile, template, nav, sitemapCategory, title, description string, breadcrumbsParent utils.Breadcrumbs) error {
		breadcrumbs := breadcrumbsParent.Push(utils.CreateLink(title, "/"+slug))
		return renderPage(slug, slugFile, template, nav, sitemapCategory, title, description, breadcrumbs)
	}

	if err := renderPage("", "index.html", "events", "events", "Laufveranstaltungen",
		"Laufveranstaltungen im Raum Freiburg",
		"Liste von Laufveranstaltungen, Lauf-Wettkämpfen, Volksläufen im Raum Freiburg",
		breadcrumbsEvents); err != nil {
		return fmt.Errorf("render index page: %w", err)
	}

	if err := renderPage("tags.html", "tags.html", "tags", "tags", "Kategorien",
		"Kategorien",
		"Liste aller Kategorien von Laufveranstaltungen, Lauf-Wettkämpfen, Volksläufen im Raum Freiburg",
		breadcrumbsTags); err != nil {
		return fmt.Errorf("render tags page: %w", err)
	}

	if err := renderPage("lauftreffs.html", "lauftreffs.html", "groups", "groups", "Lauftreffs",
		"Lauftreffs im Raum Freiburg",
		"Liste von Lauftreffs, Laufgruppen, Lauf-Trainingsgruppen im Raum Freiburg",
		breadcrumbsGroups); err != nil {
		return fmt.Errorf("render groups page: %w", err)
	}

	if err := renderPage("shops.html", "shops.html", "shops", "shops", "Lauf-Shops",
		"Lauf-Shops im Raum Freiburg",
		"Liste von Lauf-Shops und Einzelhandelsgeschäften mit Laufschuh-Auswahl im Raum Freiburg",
		breadcrumbsShops); err != nil {
		return fmt.Errorf("render shops page: %w", err)
	}

	if err := renderSubPage("dietenbach-parkrun.html", "dietenbach-parkrun.html", "dietenbach-parkrun", "parkrun", "Allgemein",
		"Dietenbach parkrun",
		"Vollständige Liste aller Ergebnisse, Laufberichte und Fotogalerien des 'Dietenbach parkrun' im Freiburger Dietenbachpark.",
		breadcrumbsBase); err != nil {
		return fmt.Errorf("render subpage %q: %w", "dietenbach-parkrun.html", err)
	}

	if err := renderPage("series.html", "series.html", "series", "series", "Serien",
		"Lauf-Serien",
		"Liste aller Serien von Laufveranstaltungen, Lauf-Wettkämpfen, Volksläufen im Raum Freiburg",
		breadcrumbsSeries); err != nil {
		return fmt.Errorf("render series page: %w", err)
	}

	if err := renderSubPage("map.html", "map.html", "map", "map", "Allgemein",
		"Karte aller Laufveranstaltungen",
		"Karte",
		breadcrumbsBase); err != nil {
		return fmt.Errorf("render subpage %q: %w", "map.html", err)
	}

	if err := renderPage("info.html", "info.html", "info", "info", "Allgemein",
		"Info",
		"Kontaktmöglichkeiten, allgemeine & technische Informationen über freiburg.run",
		breadcrumbsInfo); err != nil {
		return fmt.Errorf("render info page: %w", err)
	}

	if err := renderSubPage("datenschutz.html", "datenschutz.html", "datenschutz", "datenschutz", "Allgemein",
		"Datenschutz",
		"Datenschutzerklärung von freiburg.run",
		breadcrumbsInfo); err != nil {
		return fmt.Errorf("render subpage %q: %w", "datenschutz.html", err)
	}

	if err := renderSubPage("impressum.html", "impressum.html", "impressum", "impressum", "Allgemein",
		"Impressum",
		"Impressum von freiburg.run",
		breadcrumbsInfo); err != nil {
		return fmt.Errorf("render subpage %q: %w", "impressum.html", err)
	}

	if err := renderSubPage("support.html", "support.html", "support", "support", "Allgemein",
		"freiburg.run unterstützen",
		"Möglichkeiten freiburg.run zu unterstützen",
		breadcrumbsInfo); err != nil {
		return fmt.Errorf("render subpage %q: %w", "support.html", err)
	}

	if err := renderSubPage("404.html", "404.html", "404", "404", "",
		"404 - Seite nicht gefunden :(",
		"Fehlerseite von freiburg.run",
		breadcrumbsBase); err != nil {
		return fmt.Errorf("render subpage %q: %w", "404.html", err)
	}

	if err := renderSubPage("club/", "club/index.html", "club", "club", "Club",
		"freiburg.run Club",
		"freiburg.run Club - die Lauf-Community",
		breadcrumbsBase); err != nil {
		return fmt.Errorf("render subpage %q: %w", "club.html", err)
	}

	// Special rendering of parkrun page for wordpress
	data := TemplateData{commondata, "", "", "", "", breadcrumbsBase, "/"}
	if err := utils.ExecuteTemplateNoMinify("dietenbach-parkrun-wordpress", g.out.Join("dietenbach-parkrun-wordpress.html"), data.BasePath, data); err != nil {
		return fmt.Errorf("render wordpress template: %w", err)
	}

	// Render old events lists
	oldYearsLinks := make(map[string]*utils.Link)
	oldYears := make([]*utils.Link, 0, len(eventsData.OldEvents))
	for index, oldEvents := range eventsData.OldEvents {
		url := "/events-old.html"
		if index != 0 {
			url = fmt.Sprintf("/events-old-%s.html", oldEvents.Year)
		}
		oldYearsLinks[oldEvents.Year] = utils.CreateLink(
			fmt.Sprintf("Vergangene Laufveranstaltungen (%s)", oldEvents.Year),
			url,
		)
		oldYears = append(oldYears, utils.CreateLink(
			oldEvents.Year,
			url,
		))
	}
	for index, oldEvents := range eventsData.OldEvents {
		name := fmt.Sprintf("Vergangene Laufveranstaltungen (%s)", oldEvents.Year)
		fname := "events-old.html"
		if index != 0 {
			fname = fmt.Sprintf("events-old-%s.html", oldEvents.Year)
		}
		data := OldEventsTemplateData{
			TemplateData: TemplateData{
				commondata,
				name,
				"BLUBB",
				"events",
				"",
				breadcrumbsEvents,
				"/",
			},
			Year:   oldEvents.Year,
			Years:  oldYears,
			Events: oldEvents.Events,
		}
		data.SetNameLink(name, fname, breadcrumbsEvents, g.baseUrl)

		if err := utils.ExecuteTemplate("events-old", g.out.Join(fname), data.BasePath, data); err != nil {
			return fmt.Errorf("render old events template for %q: %w", oldEvents.Year, err)
		}
		sitemap.Add(fname, fname, name, "Vergangene Laufveranstaltungen")
	}

	// Render events, groups, shops lists
	renderEventList := func(eventList []*events.Event, nav, main, sitemapCategory string, breadcrumbs utils.Breadcrumbs) error {
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

			eventdata.Main = main
			parentBreadcrumbs := breadcrumbs
			if event.Old {
				if link, ok := oldYearsLinks[fmt.Sprintf("%d", event.Time.Year())]; ok {
					eventdata.Main = link.Url
					parentBreadcrumbs = parentBreadcrumbs.Push(link)
				}
			}

			eventdata.Event = event
			eventdata.Description = event.GenerateDescription()
			slug := event.Slug()
			fileSlug := event.SlugFile()
			name := event.Name.Orig
			if event.Meta.SeoTitle != "" {
				name = event.Meta.SeoTitle
			}
			eventdata.SetNameLink(name, slug, parentBreadcrumbs, g.baseUrl)
			if err := utils.ExecuteTemplate("event", g.out.Join(fileSlug), eventdata.BasePath, eventdata); err != nil {
				return fmt.Errorf("render event template to %q: %w", g.out.Join(fileSlug), err)
			}
			sitemap.Add(slug, fileSlug, event.Name.Orig, sitemapCategory)
		}
		return nil
	}
	if err := renderEventList(eventsData.Events, "events", "/", "Laufveranstaltungen", breadcrumbsEvents); err != nil {
		return fmt.Errorf("render event list: %w", err)
	}
	if err := renderEventList(eventsData.EventsOld, "events", "/events-old.html", "Vergangene Laufveranstaltungen", breadcrumbsEvents); err != nil {
		return fmt.Errorf("render old event list: %w", err)
	}
	if err := renderEventList(eventsData.Groups, "groups", "/lauftreffs.html", "Lauftreffs", breadcrumbsGroups); err != nil {
		return fmt.Errorf("render group event list: %w", err)
	}
	if err := renderEventList(eventsData.Shops, "shops", "/shops.html", "Lauf-Shops", breadcrumbsShops); err != nil {
		return fmt.Errorf("render shop event list: %w", err)
	}

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
		tagdata.Description = fmt.Sprintf("Laufveranstaltungen der Kategorie '%s' im Raum Freiburg; Vollständige Übersicht mit Terminen, Details und Anmeldelinks für alle Events dieser Kategorie.", tag.Name.Orig)
		slug := tag.Slug()
		tagdata.SetNameLink(tag.Name.Orig, slug, breadcrumbsTags, g.baseUrl)
		tagdata.Title = fmt.Sprintf("Laufveranstaltungen der Kategorie '%s'", tag.Name.Orig)
		if err := utils.ExecuteTemplate("tag", g.out.Join(slug), tagdata.BasePath, tagdata); err != nil {
			return fmt.Errorf("render tag template to %q: %w", g.out.Join(slug), err)
		}
		sitemap.Add(slug, slug, tag.Name.Orig, "Kategorien")
	}

	// Special rendering of the "traillauf" tag
	for _, tag := range eventsData.Tags {
		if tag.Name.Sanitized == "traillauf" {
			if err := renderEmbedList(g.baseUrl, g.out, data, tag); err != nil {
				return fmt.Errorf("create embed lists: %v", err)
			}
			break
		}
	}

	// Render series
	renderSeries := func(series []*events.Serie) error {
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
			seriedata.SetNameLink(s.Name.Orig, slug, breadcrumbsSeries, g.baseUrl)
			if err := utils.ExecuteTemplate("serie", g.out.Join(slug), seriedata.BasePath, seriedata); err != nil {
				return fmt.Errorf("render serie template to %q: %w", g.out.Join(slug), err)
			}
			sitemap.Add(slug, slug, s.Name.Orig, "Serien")
		}
		return nil
	}
	if err := renderSeries(eventsData.Series); err != nil {
		return fmt.Errorf("render series: %w", err)
	}
	if err := renderSeries(eventsData.SeriesOld); err != nil {
		return fmt.Errorf("render old series: %w", err)
	}

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
	if err := utils.ExecuteTemplate("sitemap", g.out.Join("sitemap.html"), sitemapTemplate.BasePath, sitemapTemplate); err != nil {
		return fmt.Errorf("render sitemap template to %q: %w", g.out.Join("sitemap.html"), err)
	}

	// Render .htaccess
	if err := createHtaccess(eventsData, g.out); err != nil {
		return fmt.Errorf("create .htaccess: %v", err)
	}

	return nil
}
