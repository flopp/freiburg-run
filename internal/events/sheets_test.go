package events

import (
	"testing"
	"time"

	"github.com/flopp/freiburg-run/internal/utils"
	"github.com/flopp/go-googlesheetswrapper"
)

func TestLoadSheets_Mock(t *testing.T) {
	// Prepare mock data: map[sheetName][][]string
	mockData := map[string][][]string{
		"Events2020": {
			{"DATE", "ADDED", "NAME", "NAME2", "STATUS", "URL", "DESCRIPTION", "LOCATION", "COORDINATES", "REGISTRATION", "TAGS"},
			{"2020-01-01", "2020-01-01", "Event A", "EventA", "", "http://eventa", "desc", "loc", "1,2", "", "tagA"},
		},
		"Events2021": {
			{"DATE", "ADDED", "NAME", "NAME2", "STATUS", "URL", "DESCRIPTION", "LOCATION", "COORDINATES", "REGISTRATION", "TAGS"},
			{"2021-01-01", "2021-01-01", "Event B", "EventB", "", "http://eventb", "desc", "loc", "1,2", "", "tagB"},
		},
		"Groups": {
			{"DATE", "ADDED", "NAME", "NAME2", "STATUS", "URL", "DESCRIPTION", "LOCATION", "COORDINATES", "REGISTRATION", "TAGS"},
			{"2021-01-01", "2021-01-01", "Group A", "GroupA", "", "http://groupa", "desc", "loc", "1,2", "", "tagA"},
		},
		"Shops": {
			{"DATE", "ADDED", "NAME", "NAME2", "STATUS", "URL", "DESCRIPTION", "LOCATION", "COORDINATES", "REGISTRATION", "TAGS"},
			{"2021-01-01", "2021-01-01", "Shop A", "ShopA", "", "http://shopa", "desc", "loc", "1,2", "", "tagA"},
		},
		"Tags": {
			{"TAG", "NAME", "DESCRIPTION"},
			{"tagA", "Tag A", "desc"},
			{"tagB", "Tag B", "desc"},
		},
		"Series": {
			{"NAME", "DESCRIPTION"},
			{"Serie A", "desc"},
		},
	}

	config := utils.Config{}
	today := time.Date(2021, 6, 1, 0, 0, 0, 0, time.UTC)

	sheetsData, err := LoadSheets(config, today, googlesheetswrapper.NewMock(mockData))
	if err != nil {
		t.Fatalf("LoadSheets returned error: %v", err)
	}

	if len(sheetsData.Events) != 2 {
		t.Errorf("Expected 2 events, got %d", len(sheetsData.Events))
	}
	if len(sheetsData.Groups) != 1 {
		t.Errorf("Expected 1 group, got %d", len(sheetsData.Groups))
	}
	if len(sheetsData.Shops) != 1 {
		t.Errorf("Expected 1 shop, got %d", len(sheetsData.Shops))
	}
	if len(sheetsData.Tags) != 2 {
		t.Errorf("Expected 2 tags, got %d", len(sheetsData.Tags))
	}
	if len(sheetsData.Series) != 1 {
		t.Errorf("Expected 1 series, got %d", len(sheetsData.Series))
	}
}
