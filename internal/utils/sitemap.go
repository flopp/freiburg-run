package utils

import (
	"bufio"
	"crypto/sha256"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"
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

func AddSitemapEntry(entries []string, slug string) []string {
	return append(entries, slug)
}

type FileHashDate struct {
	name string
	hash string
	date string
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
var reScript = regexp.MustCompile(`<script src="[^"]*"></script>`)
var reStyle = regexp.MustCompile(`<link rel="stylesheet" href="[^"]*"/>`)

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
	buf, err := ioutil.ReadFile(fileName)
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
	stat, err := os.Stat(filePath)
	if err != nil {
		return ""
	}

	return stat.ModTime().Format("2006-01-02")
}

func GenSitemap(fileName string, hashFileName string, outDir string, baseUrl string, entries []string) error {
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

	f.WriteString(`<?xml version="1.0" encoding="UTF-8"?>`)
	nl(f)
	f.WriteString(`<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">`)
	nl(f)

	for _, e := range entries {
		var fileName string
		if e == "" {
			fileName = filepath.Join(outDir, "index.html")
		} else {
			fileName = filepath.Join(outDir, e)
		}
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
			} else {
				log.Printf("changed hash for: %s", fileName)
			}
		} else {
			log.Printf("initial hash for: %s", fileName)
		}
		mNew[fileName] = &FileHashDate{fileName, currentHash, timeStamp}

		genSitemapEntry(f, fmt.Sprintf("%s/%s", baseUrl, e), timeStamp)
	}

	f.WriteString(`</urlset>`)

	writeHashFile(hashFileName, mNew)

	return nil
}
