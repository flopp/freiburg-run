package generator

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/flopp/freiburg-run/internal/events"
	"github.com/flopp/freiburg-run/internal/utils"
)

func createFakeData() *events.Data {
	return &events.Data{
		Events: []*events.Event{
			{
				Type: "event",
				Name: utils.Name{Orig: "Test Marathon", Sanitized: "test-marathon"},
				Time: utils.TimeRange{
					From:     time.Date(2026, 5, 15, 10, 0, 0, 0, time.UTC),
					To:       time.Date(2026, 5, 15, 13, 0, 0, 0, time.UTC),
					Original: "15.05.2026",
				},
			},
			{
				Type: "event",
				Name: utils.Name{Orig: "Summer Run", Sanitized: "summer-run"},
			},
		},
		EventsOld: []*events.Event{},
		Groups:    []*events.Event{},
		Shops:     []*events.Event{},
		Tags:      []*events.Tag{},
		Series:    []*events.Serie{},
	}
}

func TestTemplateDataCountEvents(t *testing.T) {
	data := createFakeData()
	templateData := TemplateData{
		CommonData: CommonData{
			Data: data,
		},
	}

	count := templateData.CountEvents()
	expected := 2
	if count != expected {
		t.Errorf("CountEvents() = %d, want %d", count, expected)
	}
}

func TestTemplateDataCountEventsWithSeparators(t *testing.T) {
	data := &events.Data{
		Events: []*events.Event{
			{
				Type: "event",
				Name: utils.Name{Orig: "Test Marathon", Sanitized: "test-marathon"},
			},
			{
				Type: "",
				Name: utils.Name{Orig: "", Sanitized: ""},
			},
			{
				Type: "event",
				Name: utils.Name{Orig: "Summer Run", Sanitized: "summer-run"},
			},
		},
	}

	templateData := TemplateData{
		CommonData: CommonData{
			Data: data,
		},
	}

	count := templateData.CountEvents()
	expected := 2
	if count != expected {
		t.Errorf("CountEvents() with separator = %d, want %d", count, expected)
	}
}

func TestTemplateDataImageParkrun(t *testing.T) {
	config := utils.Config{}
	config.Website.Url = "https://example.com"

	templateData := TemplateData{
		CommonData: CommonData{
			Config: config,
		},
		Nav: "parkrun",
	}

	image := templateData.Image()
	expected := "https://example.com/images/parkrun.png"
	if image != expected {
		t.Errorf("Image() = %s, want %s", image, expected)
	}
}

func TestTemplateDataImageDefault(t *testing.T) {
	config := utils.Config{}
	config.Website.Url = "https://example.com"

	templateData := TemplateData{
		CommonData: CommonData{
			Config: config,
		},
		Nav: "events",
	}

	image := templateData.Image()
	expected := "https://example.com/images/512.png"
	if image != expected {
		t.Errorf("Image() = %s, want %s", image, expected)
	}
}

func TestTemplateDataNiceTitle(t *testing.T) {
	templateData := TemplateData{
		Title: "Test Title",
	}

	niceTitle := templateData.NiceTitle()
	expected := "Test Title"
	if niceTitle != expected {
		t.Errorf("NiceTitle() = %s, want %s", niceTitle, expected)
	}
}

func TestEventTemplateDataNiceTitleGroupType(t *testing.T) {
	eventData := EventTemplateData{
		TemplateData: TemplateData{
			Title: "Running Group",
		},
		Event: &events.Event{
			Type: "group",
		},
	}

	niceTitle := eventData.NiceTitle()
	expected := "Running Group"
	if niceTitle != expected {
		t.Errorf("NiceTitle() for group = %s, want %s", niceTitle, expected)
	}
}

func TestEventTemplateDataNiceTitleEventNoTime(t *testing.T) {
	eventData := EventTemplateData{
		TemplateData: TemplateData{
			Title: "Test Event",
		},
		Event: &events.Event{
			Type: "event",
			Time: utils.TimeRange{},
		},
	}

	niceTitle := eventData.NiceTitle()
	expected := "Test Event"
	if niceTitle != expected {
		t.Errorf("NiceTitle() for event without time = %s, want %s", niceTitle, expected)
	}
}

func TestEventTemplateDataNiceTitleEventWithTimeYearInTitle(t *testing.T) {
	eventData := EventTemplateData{
		TemplateData: TemplateData{
			Title: "Marathon 2026",
		},
		Event: &events.Event{
			Type: "event",
			Time: utils.TimeRange{
				From: time.Date(2026, 5, 15, 10, 0, 0, 0, time.UTC),
			},
		},
	}

	niceTitle := eventData.NiceTitle()
	expected := "Marathon 2026"
	if niceTitle != expected {
		t.Errorf("NiceTitle() for event with year in title = %s, want %s", niceTitle, expected)
	}
}

func TestEventTemplateDataNiceTitleEventWithTimeYearNotInTitle(t *testing.T) {
	eventData := EventTemplateData{
		TemplateData: TemplateData{
			Title: "Summer Marathon",
		},
		Event: &events.Event{
			Type: "event",
			Time: utils.TimeRange{
				From: time.Date(2026, 5, 15, 10, 0, 0, 0, time.UTC),
			},
		},
	}

	niceTitle := eventData.NiceTitle()
	expected := "Summer Marathon 2026"
	if niceTitle != expected {
		t.Errorf("NiceTitle() for event without year in title = %s, want %s", niceTitle, expected)
	}
}

func TestTagTemplateDataNiceTitle(t *testing.T) {
	tagData := TagTemplateData{
		TemplateData: TemplateData{
			Title: "Running",
		},
		Tag: &events.Tag{},
	}

	niceTitle := tagData.NiceTitle()
	expected := "Running"
	if niceTitle != expected {
		t.Errorf("NiceTitle() for tag = %s, want %s", niceTitle, expected)
	}
}

func TestSerieTemplateDataNiceTitle(t *testing.T) {
	serieData := SerieTemplateData{
		TemplateData: TemplateData{
			Title: "Cup Series",
		},
		Serie: &events.Serie{},
	}

	niceTitle := serieData.NiceTitle()
	expected := "Cup Series"
	if niceTitle != expected {
		t.Errorf("NiceTitle() for serie = %s, want %s", niceTitle, expected)
	}
}

func TestSetNameLink(t *testing.T) {
	config := utils.Config{}
	config.Website.Url = "https://example.com"
	baseUrl := config.BaseUrl()
	baseBreadcrumbs := utils.Breadcrumbs{}

	templateData := &TemplateData{
		CommonData: CommonData{
			Config: config,
		},
	}

	templateData.SetNameLink("Test Event", "event/test-event", baseBreadcrumbs, baseUrl)

	if templateData.Title != "Test Event" {
		t.Errorf("SetNameLink() Title = %s, want %s", templateData.Title, "Test Event")
	}

	expectedCanonical := "https://example.com/event/test-event"
	if templateData.Canonical != expectedCanonical {
		t.Errorf("SetNameLink() Canonical = %s, want %s", templateData.Canonical, expectedCanonical)
	}

	if len(templateData.Breadcrumbs) != 1 {
		t.Errorf("SetNameLink() Breadcrumbs length = %d, want 1", len(templateData.Breadcrumbs))
	}

	if templateData.Breadcrumbs[0].Link.Name != "Test Event" {
		t.Errorf("SetNameLink() Breadcrumb name = %s, want %s", templateData.Breadcrumbs[0].Link.Name, "Test Event")
	}

	if templateData.Breadcrumbs[0].Link.Url != "/event/test-event" {
		t.Errorf("SetNameLink() Breadcrumb URL = %s, want %s", templateData.Breadcrumbs[0].Link.Url, "/event/test-event")
	}
}

func TestCreateIndexNowFileWithKey(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "indexnow-test-")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	config := utils.Config{}
	config.IndexNow.Key = "test-indexnow-key"

	outDir := utils.NewPath(tempDir)

	err = createIndexNowFile(config, outDir)
	if err != nil {
		t.Fatalf("createIndexNowFile() error = %v, want nil", err)
	}

	expectedFilePath := filepath.Join(tempDir, "test-indexnow-key.txt")
	_, err = os.Stat(expectedFilePath)
	if os.IsNotExist(err) {
		t.Errorf("Expected file %s was not created", expectedFilePath)
	}

	content, err := os.ReadFile(expectedFilePath)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	expectedContent := "test-indexnow-key"
	if string(content) != expectedContent {
		t.Errorf("File content = %s, want %s", string(content), expectedContent)
	}
}

func TestCreateIndexNowFileWithoutKey(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "indexnow-test-")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	config := utils.Config{}
	config.IndexNow.Key = ""

	outDir := utils.NewPath(tempDir)

	err = createIndexNowFile(config, outDir)
	if err != nil {
		t.Fatalf("createIndexNowFile() with empty key should not error, got %v", err)
	}

	files, err := os.ReadDir(tempDir)
	if err != nil {
		t.Fatalf("Failed to read directory: %v", err)
	}

	if len(files) != 0 {
		t.Errorf("Expected no files to be created with empty key, but found %d files", len(files))
	}
}
