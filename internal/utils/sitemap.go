package utils

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type SitemapEntry struct {
	Slug     string
	SlugFile string
	Name     string
	Category string
	Lastmod  time.Time
}

type Sitemap struct {
	BaseUrl    Url
	Categories []string
	Entries    []*SitemapEntry
}

func CreateSitemap(baseUrl Url) *Sitemap {
	return &Sitemap{baseUrl, make([]string, 0), make([]*SitemapEntry, 0)}
}

func (sitemap *Sitemap) AddCategory(name string) {
	sitemap.Categories = append(sitemap.Categories, name)
}

func (sitemap *Sitemap) Add(slug string, slugfile string, name string, category string) {
	sitemap.Entries = append(sitemap.Entries, &SitemapEntry{Slug: slug, SlugFile: slugfile, Name: name, Category: category})
}

func (sitemap *Sitemap) AddWithLastmod(slug string, slugfile string, name string, category string, lastmod time.Time) {
	sitemap.Entries = append(sitemap.Entries, &SitemapEntry{
		Slug:     slug,
		SlugFile: slugfile,
		Name:     name,
		Category: category,
		Lastmod:  lastmod,
	})
}

func writeSitemapEntry(f *os.File, url string, timeStamp string) {
	f.WriteString(fmt.Sprintf("<url><loc>%s</loc><lastmod>%s</lastmod></url>\n", url, timeStamp))
}

const dirPerms = 0770

func (sitemap Sitemap) Gen(fileName string, outDir Path) error {
	if err := os.MkdirAll(filepath.Dir(fileName), dirPerms); err != nil {
		return err
	}

	f, err := os.Create(fileName)
	if err != nil {
		return err
	}
	defer f.Close()

	f.WriteString("<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n")
	f.WriteString("<urlset xmlns=\"http://www.sitemaps.org/schemas/sitemap/0.9\">\n")

	for _, entry := range sitemap.Entries {
		lastmod := entry.Lastmod
		if lastmod.IsZero() {
			path := outDir.Join(entry.SlugFile)
			var err error
			lastmod, err = GetMtime(path)
			if err != nil {
				return fmt.Errorf("get mtime for sitemap entry %q: %w", path, err)
			}
		}

		writeSitemapEntry(f, sitemap.BaseUrl.Join(entry.Slug), lastmod.Format("2006-01-02"))
	}

	f.WriteString("</urlset>")

	return nil
}

type SitemapCategory struct {
	Name  string
	Links []*Link
}

func (sitemap Sitemap) GenHTML() []SitemapCategory {
	byCategory := make(map[string][]*SitemapEntry)

	for _, e := range sitemap.Entries {
		byCategory[e.Category] = append(byCategory[e.Category], e)
	}

	categories := make([]SitemapCategory, 0)
	for _, c := range sitemap.Categories {
		links := make([]*Link, 0)
		for _, e := range byCategory[c] {
			links = append(links, CreateLink(e.Name, sitemap.BaseUrl.Join(e.Slug)))
		}
		categories = append(categories, SitemapCategory{c, links})
	}

	return categories
}
