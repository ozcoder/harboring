package report

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/harboring/harboring/internal/har"
)

func TestStatsJSONReport(t *testing.T) {
	data, err := StatsJSONReport(sampleDetails())
	if err != nil {
		t.Fatal(err)
	}
	if !json.Valid(data) {
		t.Fatalf("invalid JSON: %s", data)
	}
	var out StatsJSON
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatal(err)
	}
	if out.Requests != 3 {
		t.Fatalf("expected 3 requests, got %d", out.Requests)
	}
	if out.TotalSize != 3584 {
		t.Fatalf("expected total size 3584, got %d", out.TotalSize)
	}
	if out.TotalSizeHuman != "3.5 KB" {
		t.Fatalf("expected '3.5 KB', got %q", out.TotalSizeHuman)
	}
	if out.Methods["GET"] != 2 {
		t.Fatalf("expected 2 GET requests, got %d", out.Methods["GET"])
	}
	if len(out.Slowest) != 3 {
		t.Fatalf("expected 3 slowest, got %d", len(out.Slowest))
	}
	if out.Slowest[0].TimeMs != 500 {
		t.Fatalf("expected slowest at 500ms, got %.0f", out.Slowest[0].TimeMs)
	}
}

func TestPerfJSONReport(t *testing.T) {
	data, err := PerfJSONReport(sampleDetails())
	if err != nil {
		t.Fatal(err)
	}
	if !json.Valid(data) {
		t.Fatalf("invalid JSON: %s", data)
	}
	var out PerfJSON
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatal(err)
	}
	if out.Requests != 3 {
		t.Fatalf("expected 3 requests, got %d", out.Requests)
	}
	if out.TTFBPercentiles.P50 != 5 {
		t.Fatalf("expected p50=5, got %.0f", out.TTFBPercentiles.P50)
	}
	if out.ByResourceType["document"] != 1024 {
		t.Fatalf("expected document=1024, got %d", out.ByResourceType["document"])
	}
	if out.Largest == nil || out.Largest.Size != 2048 {
		t.Fatalf("expected largest size 2048")
	}
	if out.Slowest == nil || out.Slowest.TimeMs != 500 {
		t.Fatalf("expected slowest time 500ms")
	}
}

func TestCacheJSONReport(t *testing.T) {
	data, err := CacheJSONReport(sampleDetails())
	if err != nil {
		t.Fatal(err)
	}
	if !json.Valid(data) {
		t.Fatalf("invalid JSON: %s", data)
	}
	var out CacheJSON
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatal(err)
	}
	if out.Total != 3 {
		t.Fatalf("expected 3 total, got %d", out.Total)
	}
	if out.GoodCache != 1 {
		t.Fatalf("expected 1 good cache, got %d", out.GoodCache)
	}
	if out.NoCache != 2 {
		t.Fatalf("expected 2 no cache, got %d", out.NoCache)
	}
	if out.OverCached != 0 {
		t.Fatalf("expected 0 over cached, got %d", out.OverCached)
	}
}

func TestSecurityJSONReport(t *testing.T) {
	data, err := SecurityJSONReport(sampleDetails())
	if err != nil {
		t.Fatal(err)
	}
	if !json.Valid(data) {
		t.Fatalf("invalid JSON: %s", data)
	}
	var out SecurityJSON
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatal(err)
	}
	if len(out.Domains) != 2 {
		t.Fatalf("expected 2 domains, got %d", len(out.Domains))
	}
	found := false
	for _, d := range out.Domains {
		if d.Domain == "example.com" {
			found = true
			if d.MissingRequired != 1 {
				t.Fatalf("expected example.com to have 1 missing required, got %d", d.MissingRequired)
			}
		}
	}
	if !found {
		t.Fatal("expected example.com in results")
	}
}

func TestJSONReportsEmpty(t *testing.T) {
	for _, f := range []func([]*har.EntryDetails) ([]byte, error){
		StatsJSONReport, PerfJSONReport, CacheJSONReport, SecurityJSONReport,
	} {
		data, err := f(nil)
		if err != nil {
			t.Fatalf("unexpected error for empty entries: %v", err)
		}
		if !json.Valid(data) {
			t.Fatalf("expected valid JSON for empty entries: %s", data)
		}
	}
}

func TestJSONReportsSample(t *testing.T) {
	for _, tc := range []struct {
		name string
		fn   func([]*har.EntryDetails) ([]byte, error)
		key  string
	}{
		{"stats", StatsJSONReport, `"requests"`},
		{"perf", PerfJSONReport, `"ttfb_percentiles"`},
		{"cache", CacheJSONReport, `"good_cache"`},
		{"security", SecurityJSONReport, `"domains"`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			data, err := tc.fn(sampleDetails())
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(string(data), tc.key) {
				t.Fatalf("expected key %s in output: %s", tc.key, data)
			}
		})
	}
}
