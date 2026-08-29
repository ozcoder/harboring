package modes

import (
	"strings"
	"testing"

	"github.com/harboring/harboring/internal/har"
	"github.com/harboring/harboring/internal/theme"
)

func TestSummaryShowsTrackers(t *testing.T) {
	th := theme.New()
	entries := []*har.EntryDetails{
		{Method: "GET", URL: "https://example.com/page", URLPath: "/page", Status: 200, Size: 1000, Time: 100, Domain: "example.com", ResourceType: "document"},
		{Method: "GET", URL: "https://www.google-analytics.com/collect", URLPath: "/collect", Status: 200, Size: 200, Time: 10, Domain: "www.google-analytics.com", ResourceType: "xhr"},
		{Method: "GET", URL: "https://pagead2.googlesyndication.com/ad", URLPath: "/ad", Status: 200, Size: 300, Time: 15, Domain: "pagead2.googlesyndication.com", ResourceType: "script"},
	}
	m := NewSummaryModel(entries, th)
	m.SetViewportSize(120, 30)

	view := m.View()
	if !strings.Contains(view, "Third-party Trackers") {
		t.Fatalf("expected trackers section in summary view, got:\n%s", view)
	}
	if !strings.Contains(view, "Tracker Requests: 2 / 3") {
		t.Fatalf("expected tracker request count, got:\n%s", view)
	}
	if !strings.Contains(view, "analytics") {
		t.Fatalf("expected analytics category in summary view, got:\n%s", view)
	}
	if !strings.Contains(view, "advertising") {
		t.Fatalf("expected advertising category in summary view, got:\n%s", view)
	}
}

func TestSummaryTrackersEmpty(t *testing.T) {
	th := theme.New()
	entries := []*har.EntryDetails{
		{Method: "GET", URL: "https://example.com/page", URLPath: "/page", Status: 200, Size: 1000, Time: 100, Domain: "example.com", ResourceType: "document"},
	}
	m := NewSummaryModel(entries, th)
	m.SetViewportSize(120, 30)

	view := m.View()
	if !strings.Contains(view, "Third-party Trackers") {
		t.Fatalf("expected trackers section even with no trackers, got:\n%s", view)
	}
	if !strings.Contains(view, "Tracker Requests: 0 / 1") {
		t.Fatalf("expected 0 tracker requests, got:\n%s", view)
	}
}
