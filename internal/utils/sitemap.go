package utils

import (
	"bufio"
	"crypto/sha256"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
)

type FileHashDate struct {
	name string
	hash string
	date string
}

type SitemapEntry struct {
	Slug     string
	SlugFile string
	Name     string
	Category string
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
	sitemap.Entries = append(sitemap.Entries, &SitemapEntry{slug, slugfile, name, category})
}

func genSitemapEntry(f *os.File, url string, timeStamp string) {
	f.WriteString(fmt.Sprintf("<url><loc>%s</loc><lastmod>%s</lastmod></url>\n", url, timeStamp))
}

func AddSitemapEntry(entries []string, slug string) []string {
	return append(entries, slug)
}

func readHashFile(fileName string) map[string]*FileHashDate {
	m := make(map[string]*FileHashDate)
	if _, err := os.Stat(fileName); err != nil {
		return m
	}

	f, err := os.Open(fileName)
	if err != nil {
		return m
	}
	defer f.Close()

	fileScanner := bufio.NewScanner(f)
	fileScanner.Split(bufio.ScanLines)

	r := regexp.MustCompile(`^([^\t]+)\t([^\t]+)\t([^\t]+)\s*$`)
	for fileScanner.Scan() {
		line := fileScanner.Text()
		if match := r.FindStringSubmatch(line); match != nil {
			m[match[1]] = &FileHashDate{match[1], match[2], match[3]}
		} else {
			log.Printf("%s: cannot parse line <%s>", fileName, line)
		}
	}
	return m
}

func writeHashFile(fileName string, m map[string]*FileHashDate) {
	f, err := os.Create(fileName)
	if err != nil {
		log.Printf("cannot create hash file: %s, %v", fileName, err)
		return
	}
	defer f.Close()

	for _, data := range m {
		f.WriteString(fmt.Sprintf("%s\t%s\t%s\n", data.name, data.hash, data.date))
	}
}

var reTimestamp = regexp.MustCompile(`<span class="timestamp">[^<]*</span>`)
var reScript = regexp.MustCompile(`<script [^>]*>`)
var reStyle = regexp.MustCompile(`<link [^>]*>`)

func replaceRegexp(s []byte, r regexp.Regexp) []byte {
	for {
		match := r.FindIndex(s)
		if match != nil {
			matchStart := match[0]
			matchEnd := match[1]
			replaced := make([]byte, 0, len(s))
			replaced = append(replaced, s[:matchStart]...)
			replaced = append(replaced, s[matchEnd:]...)
			s = replaced
		} else {
			break
		}
	}
	return s
}

func determineHash(fileName string) (string, error) {
	buf, err := os.ReadFile(fileName)
	if err != nil {
		return "", err
	}

	buf = replaceRegexp(buf, *reTimestamp)
	buf = replaceRegexp(buf, *reScript)
	buf = replaceRegexp(buf, *reStyle)

	h := sha256.New()
	h.Write(buf)

	return fmt.Sprintf("%.8x", h.Sum(nil)), nil
}

func getMtimeYMD(filePath string) string {
	if t, err := GetMtime(filePath); err != nil {
		return ""
	} else {
		return t.Format("2006-01-02")
	}
}

func (sitemap Sitemap) Gen(fileName string, hashFileName string, outDir Path) error {
	if err := os.MkdirAll(filepath.Dir(fileName), 0770); err != nil {
		return err
	}

	f, err := os.Create(fileName)
	if err != nil {
		return err
	}
	defer f.Close()

	m := readHashFile(hashFileName)
	mNew := make(map[string]*FileHashDate)

	f.WriteString("<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n")
	f.WriteString("<urlset xmlns=\"http://www.sitemaps.org/schemas/sitemap/0.9\">\n")

	for _, entry := range sitemap.Entries {
		fileName := outDir.Join(entry.SlugFile)
		timeStamp := getMtimeYMD(fileName)
		if timeStamp == "" {
			log.Printf("cannot get mtime '%s'", fileName)
		}
		currentHash, err := determineHash(fileName)
		if err != nil {
			log.Printf("cannot create hash for '%s': %v", fileName, err)
		}

		oldHash, ok := m[fileName]
		if ok {
			if currentHash == oldHash.hash {
				timeStamp = oldHash.date
			}
		} else {
			log.Printf("initial hash for: %s", fileName)
		}
		mNew[fileName] = &FileHashDate{fileName, currentHash, timeStamp}

		genSitemapEntry(f, sitemap.BaseUrl.Join(entry.Slug), timeStamp)
	}

	f.WriteString("</urlset>")

	writeHashFile(hashFileName, mNew)

	return nil
}

type SitemapCategory struct {
	Name  string
	Links []*Link
}

func (sitemap Sitemap) GenHTML() []SitemapCategory {
	byCategory := make(map[string][]*SitemapEntry)
	for _, c := range sitemap.Categories {
		byCategory[c] = make([]*SitemapEntry, 0)
	}

	for _, e := range sitemap.Entries {
		c, found := byCategory[e.Category]
		if !found {
			log.Printf("Sitemap: event '%s' has bad category '%s'", e.Name, e.Category)
			continue
		}

		c = append(c, e)
		byCategory[e.Category] = c
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
