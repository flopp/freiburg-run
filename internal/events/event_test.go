package events

import (
	"strings"
	"testing"
	"time"

	"github.com/flopp/freiburg-run/internal/utils"
)

func TestEventLastmodUsesAddedDate(t *testing.T) {
	event := Event{
		Name:  utils.NewName("Test event"),
		Added: "15.02.2026",
		Time:  utils.TimeRange{From: time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)},
	}

	lastmod, err := event.Lastmod()
	if err != nil {
		t.Fatalf("Lastmod() returned an error: %v", err)
	}
	if got, want := lastmod.Format("2006-01-02"), "2026-02-15"; got != want {
		t.Errorf("Lastmod() = %s, want %s", got, want)
	}
}

func TestEventLastmodFallsBackToEventDate(t *testing.T) {
	event := Event{
		Name: utils.NewName("Test event"),
		Time: utils.TimeRange{From: time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)},
	}

	lastmod, err := event.Lastmod()
	if err != nil {
		t.Fatalf("Lastmod() returned an error: %v", err)
	}
	if got, want := lastmod.Format("2006-01-02"), "2026-03-01"; got != want {
		t.Errorf("Lastmod() = %s, want %s", got, want)
	}
}

func TestEventLastmodRejectsInvalidAddedDate(t *testing.T) {
	event := Event{Name: utils.NewName("Test event"), Added: "not-a-date"}

	_, err := event.Lastmod()
	if err == nil {
		t.Fatal("Lastmod() returned nil error for invalid Added date")
	}
	if !strings.Contains(err.Error(), "invalid added date") {
		t.Errorf("Lastmod() error = %q, want invalid added date context", err)
	}
}
