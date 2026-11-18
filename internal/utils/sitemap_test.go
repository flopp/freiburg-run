package utils

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
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

func TestGen(t *testing.T) {
	tempDir := t.TempDir()
	sitemapFile := filepath.Join(tempDir, "sitemap.xml")
	outDir := Path(tempDir)

	// Create a dummy HTML file
	htmlContent := `<html><body>Content</body></html>`
	htmlFile := filepath.Join(tempDir, "event1.html")
	err := os.WriteFile(htmlFile, []byte(htmlContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create HTML file: %v", err)
	}
	mtime := time.Date(2025, 4, 12, 12, 0, 0, 0, time.Local)
	if err := os.Chtimes(htmlFile, mtime, mtime); err != nil {
		t.Fatalf("Failed to set HTML file mtime: %v", err)
	}

	sitemap := CreateSitemap(Url("https://example.com"))
	sitemap.Add("event1", "event1.html", "Event 1", "events")

	err = sitemap.Gen(sitemapFile, outDir)
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
<url><loc>https://example.com/event1</loc><lastmod>2025-04-12</lastmod></url>`
	if !strings.Contains(string(content), expected) {
		t.Errorf("Sitemap content mismatch. Got:\n%s", string(content))
	}

}

func TestGenUsesExplicitLastmod(t *testing.T) {
	tempDir := t.TempDir()
	sitemapFile := filepath.Join(tempDir, "sitemap.xml")
	outDir := Path(tempDir)

	htmlFile := filepath.Join(tempDir, "event1.html")
	if err := os.WriteFile(htmlFile, []byte("<html></html>"), 0644); err != nil {
		t.Fatalf("Failed to create HTML file: %v", err)
	}

	sitemap := CreateSitemap(Url("https://example.com"))
	sitemap.AddWithLastmod(
		"event1",
		"event1.html",
		"Event 1",
		"events",
		time.Date(2026, 2, 15, 0, 0, 0, 0, time.UTC),
	)

	if err := sitemap.Gen(sitemapFile, outDir); err != nil {
		t.Fatalf("Gen failed: %v", err)
	}

	content, err := os.ReadFile(sitemapFile)
	if err != nil {
		t.Fatalf("Failed to read sitemap: %v", err)
	}
	if !strings.Contains(string(content), "<lastmod>2026-02-15</lastmod>") {
		t.Errorf("Sitemap does not contain explicit lastmod date:\n%s", content)
	}

	sitemap.Entries[0].Lastmod = time.Date(2026, 3, 5, 0, 0, 0, 0, time.UTC)
	if err := sitemap.Gen(sitemapFile, outDir); err != nil {
		t.Fatalf("Second Gen failed: %v", err)
	}
	content, err = os.ReadFile(sitemapFile)
	if err != nil {
		t.Fatalf("Failed to read sitemap: %v", err)
	}
	if !strings.Contains(string(content), "<lastmod>2026-03-05</lastmod>") {
		t.Errorf("Sitemap does not contain updated lastmod date:\n%s", content)
	}
}

func TestGenFailsWhenFileMtimeUnavailable(t *testing.T) {
	tempDir := t.TempDir()
	sitemap := CreateSitemap(Url("https://example.com"))
	sitemap.Add("missing", "missing.html", "Missing", "events")

	err := sitemap.Gen(filepath.Join(tempDir, "sitemap.xml"), Path(tempDir))
	if err == nil || !strings.Contains(err.Error(), "missing.html") {
		t.Fatalf("Gen error = %v, want an error identifying the missing file", err)
	}
}
