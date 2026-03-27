package utils

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCreateSitemap(t *testing.T) {
	baseUrl := Url("https://example.com")
	sitemap := CreateSitemap(baseUrl)

	if sitemap.BaseUrl != baseUrl {
		t.Errorf("Expected BaseUrl %s, got %s", baseUrl, sitemap.BaseUrl)
	}

	if len(sitemap.Categories) != 0 {
		t.Errorf("Expected empty categories, got %d", len(sitemap.Categories))
	}

	if len(sitemap.Entries) != 0 {
		t.Errorf("Expected empty entries, got %d", len(sitemap.Entries))
	}
}

func TestSitemapAddCategory(t *testing.T) {
	sitemap := CreateSitemap(Url("https://example.com"))
	sitemap.AddCategory("events")

	if len(sitemap.Categories) != 1 || sitemap.Categories[0] != "events" {
		t.Errorf("Expected categories ['events'], got %v", sitemap.Categories)
	}
}

func TestSitemapAdd(t *testing.T) {
	sitemap := CreateSitemap(Url("https://example.com"))
	sitemap.Add("slug1", "file1.html", "Name1", "cat1")

	if len(sitemap.Entries) != 1 {
		t.Errorf("Expected 1 entry, got %d", len(sitemap.Entries))
	}

	entry := sitemap.Entries[0]
	if entry.Slug != "slug1" || entry.SlugFile != "file1.html" || entry.Name != "Name1" || entry.Category != "cat1" {
		t.Errorf("Entry mismatch: %+v", entry)
	}
}

func TestGenHTML(t *testing.T) {
	sitemap := CreateSitemap(Url("https://example.com"))
	sitemap.AddCategory("events")
	sitemap.AddCategory("groups")
	sitemap.Add("event1", "event1.html", "Event 1", "events")
	sitemap.Add("event2", "event2.html", "Event 2", "events")
	sitemap.Add("group1", "group1.html", "Group 1", "groups")

	categories := sitemap.GenHTML()

	if len(categories) != 2 {
		t.Errorf("Expected 2 categories, got %d", len(categories))
	}

	// Check events category
	eventsCat := categories[0]
	if eventsCat.Name != "events" {
		t.Errorf("Expected category 'events', got '%s'", eventsCat.Name)
	}
	if len(eventsCat.Links) != 2 {
		t.Errorf("Expected 2 links in events, got %d", len(eventsCat.Links))
	}
	if eventsCat.Links[0].Name != "Event 1" || eventsCat.Links[0].Url != "https://example.com/event1" {
		t.Errorf("Link mismatch: %+v", eventsCat.Links[0])
	}

	// Check groups category
	groupsCat := categories[1]
	if groupsCat.Name != "groups" {
		t.Errorf("Expected category 'groups', got '%s'", groupsCat.Name)
	}
	if len(groupsCat.Links) != 1 {
		t.Errorf("Expected 1 link in groups, got %d", len(groupsCat.Links))
	}
}

func TestDetermineHash(t *testing.T) {
	// Create a temp file
	content := `<html><body><span class="timestamp">2023-01-01</span><script src="app.js"></script><link rel="stylesheet" href="style.css">Content</body></html>`
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "test.html")
	err := os.WriteFile(filePath, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	hash, err := determineHash(filePath)
	if err != nil {
		t.Errorf("determineHash failed: %v", err)
	}

	// The hash should be consistent and ignore the removed parts
	expectedHash := "ec766c35d08e05df" // Actual output from current code
	if hash != expectedHash {
		t.Errorf("Expected hash %s, got %s", expectedHash, hash)
	}
}

func TestReadWriteHashFile(t *testing.T) {
	tempDir := t.TempDir()
	hashFile := filepath.Join(tempDir, "hashes.txt")

	// Write some data
	data := map[string]*FileHashDate{
		"file1.html": {"file1.html", "hash1", "2023-01-01"},
		"file2.html": {"file2.html", "hash2", "2023-01-02"},
	}
	writeHashFile(hashFile, data)

	// Read it back
	readData := readHashFile(hashFile)

	if len(readData) != 2 {
		t.Errorf("Expected 2 entries, got %d", len(readData))
	}

	if readData["file1.html"].hash != "hash1" || readData["file1.html"].date != "2023-01-01" {
		t.Errorf("Data mismatch for file1: %+v", readData["file1.html"])
	}
}

func TestGen(t *testing.T) {
	tempDir := t.TempDir()
	sitemapFile := filepath.Join(tempDir, "sitemap.xml")
	hashFile := filepath.Join(tempDir, "hashes.txt")
	outDir := Path(tempDir)

	// Create a dummy HTML file
	htmlContent := `<html><body>Content</body></html>`
	htmlFile := filepath.Join(tempDir, "event1.html")
	err := os.WriteFile(htmlFile, []byte(htmlContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create HTML file: %v", err)
	}

	sitemap := CreateSitemap(Url("https://example.com"))
	sitemap.Add("event1", "event1.html", "Event 1", "events")

	err = sitemap.Gen(sitemapFile, hashFile, outDir)
	if err != nil {
		t.Errorf("Gen failed: %v", err)
	}

	// Check sitemap file content
	content, err := os.ReadFile(sitemapFile)
	if err != nil {
		t.Fatalf("Failed to read sitemap: %v", err)
	}

	expected := `<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
<url><loc>https://example.com/event1</loc><lastmod>`
	if !strings.Contains(string(content), expected) {
		t.Errorf("Sitemap content mismatch. Got:\n%s", string(content))
	}

	// Check hash file
	hashData := readHashFile(hashFile)
	if len(hashData) != 1 {
		t.Errorf("Expected 1 hash entry, got %d", len(hashData))
	}
}
