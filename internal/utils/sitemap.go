package utils

import (
	"fmt"
	"os"
	"path/filepath"
)

type SitemapEntry struct {
	slug      string
	timestamp string
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

func AddSitemapEntry(entries []SitemapEntry, slug string, timeStamp string) []SitemapEntry {
	return append(entries, SitemapEntry{slug, timeStamp})
}

func GenSitemap(fileName string, baseUrl string, entries []SitemapEntry, forceDate string) error {
	outDir := filepath.Dir(fileName)
	if err := os.MkdirAll(outDir, 0770); err != nil {
		return err
	}

	f, err := os.Create(fileName)
	if err != nil {
		return err
	}

	defer f.Close()

	f.WriteString(`<?xml version="1.0" encoding="UTF-8"?>`)
	nl(f)
	f.WriteString(`<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">`)
	nl(f)

	for _, e := range entries {
		if forceDate > e.timestamp {
			genSitemapEntry(f, fmt.Sprintf("%s/%s", baseUrl, e.slug), forceDate)
		} else {
			genSitemapEntry(f, fmt.Sprintf("%s/%s", baseUrl, e.slug), e.timestamp)
		}
	}

	f.WriteString(`</urlset>`)
	return nil
}
