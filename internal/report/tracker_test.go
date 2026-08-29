package report

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/harboring/harboring/internal/har"
)

func TestIsTracker(t *testing.T) {
	tests := []struct {
		domain   string
		expected bool
	}{
		{"google-analytics.com", true},
		{"www.google-analytics.com", true},
		{"analytics.google.com", true},
		{"sub.doubleclick.net", true},
		{"example.com", false},
		{"notgoogle-analytics.com", false},
		{"doubleclick.net.evil.com", false},
		{"facebook.com", true},
		{"connect.facebook.net", true},
		{"GOOGLE-ANALYTICS.COM", true}, // case insensitive
		{"", false},
	}
	for _, tc := range tests {
		_, got := IsTracker(tc.domain)
		if got != tc.expected {
			t.Errorf("IsTracker(%q) = %v, want %v", tc.domain, got, tc.expected)
		}
	}
}

func TestIsTrackerCategory(t *testing.T) {
	cat, ok := IsTracker("www.google-analytics.com")
	if !ok {
		t.Fatal("expected google-analytics.com to be a tracker")
	}
	if cat != TrackerAnalytics {
		t.Fatalf("expected analytics category, got %q", cat)
	}

	cat, ok = IsTracker("pixels.facebook.com")
	if !ok {
		t.Fatal("expected facebook.com to be a tracker")
	}
	if cat != TrackerSocial {
		t.Fatalf("expected social category, got %q", cat)
	}

	cat, ok = IsTracker("ad.doubleclick.net")
	if !ok {
		t.Fatal("expected doubleclick.net to be a tracker")
	}
	if cat != TrackerAdvertising {
		t.Fatalf("expected advertising category, got %q", cat)
	}
}

func TestAnalyzeTrackers(t *testing.T) {
	entries := []*har.EntryDetails{
		{Method: "GET", URL: "https://example.com/page", Domain: "example.com", Size: 1000, Time: 100},
		{Method: "GET", URL: "https://www.google-analytics.com/collect", Domain: "www.google-analytics.com", Size: 200, Time: 10},
		{Method: "GET", URL: "https://cdn.example.com/style.css", Domain: "cdn.example.com", Size: 500, Time: 20},
		{Method: "GET", URL: "https://pagead2.googlesyndication.com/ad", Domain: "pagead2.googlesyndication.com", Size: 300, Time: 15},
		{Method: "GET", URL: "https://connect.facebook.net/en_US/sdk.js", Domain: "connect.facebook.net", Size: 100, Time: 5},
	}

	result := AnalyzeTrackers(entries)

	if result.TotalRequests != 5 {
		t.Fatalf("expected 5 total requests, got %d", result.TotalRequests)
	}
	if result.TrackerRequests != 3 {
		t.Fatalf("expected 3 tracker requests, got %d", result.TrackerRequests)
	}
	if result.TrackerPercent != 60.0 {
		t.Fatalf("expected 60%% tracker percent, got %.1f", result.TrackerPercent)
	}
	if result.TrackerSize != 600 {
		t.Fatalf("expected 600 tracker size, got %d", result.TrackerSize)
	}
	if len(result.Domains) != 3 {
		t.Fatalf("expected 3 tracker domains, got %d", len(result.Domains))
	}

	cats := result.ByCategory
	if cats["analytics"] != 1 {
		t.Fatalf("expected 1 analytics, got %d", cats["analytics"])
	}
	if cats["advertising"] != 1 {
		t.Fatalf("expected 1 advertising, got %d", cats["advertising"])
	}
	if cats["social"] != 1 {
		t.Fatalf("expected 1 social, got %d", cats["social"])
	}
}

func TestAnalyzeTrackersEmpty(t *testing.T) {
	result := AnalyzeTrackers(nil)
	if result.TrackerRequests != 0 {
		t.Fatalf("expected 0 tracker requests for nil, got %d", result.TrackerRequests)
	}
	if result.TrackerPercent != 0 {
		t.Fatalf("expected 0%% tracker percent, got %.1f", result.TrackerPercent)
	}
}

func TestTrackersReportText(t *testing.T) {
	entries := []*har.EntryDetails{
		{Method: "GET", URL: "https://example.com/page", Domain: "example.com", Size: 1000, Time: 100},
		{Method: "GET", URL: "https://www.google-analytics.com/collect", Domain: "www.google-analytics.com", Size: 200, Time: 10},
	}
	out := TrackersReport(entries, false)
	if !strings.Contains(out, "Third-Party Tracker Analysis") {
		t.Fatalf("expected title, got:\n%s", out)
	}
	if !strings.Contains(out, "google-analytics.com") {
		t.Fatalf("expected tracker domain, got:\n%s", out)
	}
}

func TestTrackersReportMarkdown(t *testing.T) {
	entries := []*har.EntryDetails{
		{Method: "GET", URL: "https://example.com/page", Domain: "example.com", Size: 1000, Time: 100},
		{Method: "GET", URL: "https://www.google-analytics.com/collect", Domain: "www.google-analytics.com", Size: 200, Time: 10},
	}
	out := TrackersReport(entries, true)
	if !strings.HasPrefix(out, "# Third-Party Tracker Analysis") {
		t.Fatalf("expected markdown heading, got:\n%s", out)
	}
	if !strings.Contains(out, "| Domain | Category | Requests | Size |") {
		t.Fatalf("expected markdown table header, got:\n%s", out)
	}
}

func TestTrackersJSONReport(t *testing.T) {
	entries := []*har.EntryDetails{
		{Method: "GET", URL: "https://example.com/page", Domain: "example.com", Size: 1000, Time: 100},
		{Method: "GET", URL: "https://www.google-analytics.com/collect", Domain: "www.google-analytics.com", Size: 200, Time: 10},
	}
	data, err := TrackersJSONReport(entries)
	if err != nil {
		t.Fatal(err)
	}
	out := string(data)
	if !strings.Contains(out, `"tracker_requests"`) {
		t.Fatalf("expected tracker_requests key, got:\n%s", out)
	}
	if !strings.Contains(out, `"www.google-analytics.com"`) {
		t.Fatalf("expected tracker domain in JSON, got:\n%s", out)
	}
}

func TestAddTrackersUserDefined(t *testing.T) {
	// A domain that is not in the built-in list.
	if _, ok := IsTracker("custom-tracker.example.com"); ok {
		t.Fatal("expected custom domain to be unknown before adding")
	}

	AddTrackers([]TrackerDef{
		{Domain: "custom-tracker.example.com", Category: TrackerAnalytics},
	})

	cat, ok := IsTracker("custom-tracker.example.com")
	if !ok {
		t.Fatal("expected custom domain to be a tracker after adding")
	}
	if cat != TrackerAnalytics {
		t.Fatalf("expected analytics category, got %q", cat)
	}

	// Suffix matching applies to user trackers too.
	if _, ok := IsTracker("sub.custom-tracker.example.com"); !ok {
		t.Fatal("expected subdomain of custom tracker to match")
	}

	// Empty category defaults to "other".
	AddTrackers([]TrackerDef{{Domain: "other.custom-example.com"}})
	if cat, ok := IsTracker("other.custom-example.com"); !ok {
		t.Fatal("expected empty-category custom tracker to be added")
	} else if cat != TrackerOther {
		t.Fatalf("expected default other category, got %q", cat)
	}

	// User trackers appear in KnownTrackers alongside built-ins.
	found := false
	for _, k := range KnownTrackers() {
		if k.Domain == "custom-tracker.example.com" {
			found = true
		}
	}
	if !found {
		t.Fatal("expected custom tracker in KnownTrackers()")
	}
}

func TestLoadTrackersFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "trackers.toml")
	if err := os.WriteFile(path, []byte(`
[[trackers]]
domain = "file.tracker.example"
category = "advertising"
`), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := LoadTrackersFile(path); err != nil {
		t.Fatal(err)
	}
	if _, ok := IsTracker("file.tracker.example"); !ok {
		t.Fatal("expected file-loaded tracker to be active")
	}

	// Missing file returns an error.
	if err := LoadTrackersFile(filepath.Join(dir, "nope.toml")); err == nil {
		t.Fatal("expected error for missing trackers file")
	}

	// Empty path is a no-op.
	if err := LoadTrackersFile(""); err != nil {
		t.Fatalf("empty path should be a no-op, got %v", err)
	}
}

func TestKnownTrackersNonEmpty(t *testing.T) {
	list := KnownTrackers()
	if len(list) < len(knownTrackers) {
		t.Fatalf("expected at least %d known trackers, got %d", len(knownTrackers), len(list))
	}
}
