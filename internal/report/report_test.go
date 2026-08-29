package report

import (
	"strings"
	"testing"

	"github.com/harboring/harboring/internal/har"
)

func sampleDetails() []*har.EntryDetails {
	return []*har.EntryDetails{
		{
			Method:          "GET",
			URL:             "https://example.com/a",
			URLPath:         "/a",
			Status:          200,
			Size:            1024,
			Time:            100,
			Wait:            10,
			Domain:          "example.com",
			ResourceType:    "document",
			MaxAge:          3600,
			HasCacheHeaders: true,
			ResponseHeaders: []har.Header{
				{Name: "Strict-Transport-Security", Value: "max-age=31536000"},
				{Name: "Content-Security-Policy", Value: "default-src 'self'"},
			},
		},
		{
			Method:          "POST",
			URL:             "https://example.com/b",
			URLPath:         "/b",
			Status:          404,
			Size:            2048,
			Time:            500,
			Wait:            0,
			Domain:          "example.com",
			ResourceType:    "script",
			HasCacheHeaders: false,
			ResponseHeaders: []har.Header{
				{Name: "X-Content-Type-Options", Value: "nosniff"},
			},
		},
		{
			Method:          "GET",
			URL:             "https://other.com/c",
			URLPath:         "/c",
			Status:          200,
			Size:            512,
			Time:            50,
			Wait:            5,
			Domain:          "other.com",
			ResourceType:    "image",
			MaxAge:          -1,
			HasCacheHeaders: false,
		},
	}
}

func TestStatsReportText(t *testing.T) {
	out := StatsReport(sampleDetails(), false)
	if !strings.Contains(out, "Requests:    3") {
		t.Fatalf("expected request count, got:\n%s", out)
	}
	if !strings.Contains(out, "Top 5 slowest:") {
		t.Fatalf("expected slowest section, got:\n%s", out)
	}
}

func TestStatsReportMarkdown(t *testing.T) {
	out := StatsReport(sampleDetails(), true)
	if !strings.HasPrefix(out, "# HAR Statistics") {
		t.Fatalf("expected markdown heading, got:\n%s", out)
	}
	if !strings.Contains(out, "| Metric | Value |") {
		t.Fatalf("expected markdown table, got:\n%s", out)
	}
}

func TestPerfReportText(t *testing.T) {
	out := PerfReport(sampleDetails(), false)
	if !strings.Contains(out, "HAR Performance Report") {
		t.Fatalf("expected title, got:\n%s", out)
	}
	if !strings.Contains(out, "TTFB Percentiles:") {
		t.Fatalf("expected TTFB section, got:\n%s", out)
	}
}

func TestPerfReportMarkdown(t *testing.T) {
	out := PerfReport(sampleDetails(), true)
	if !strings.Contains(out, "# HAR Performance Report") {
		t.Fatalf("expected markdown heading, got:\n%s", out)
	}
	if !strings.Contains(out, "## By Resource Type") {
		t.Fatalf("expected resource type section, got:\n%s", out)
	}
}

func TestCacheReportText(t *testing.T) {
	out := CacheReport(sampleDetails(), false)
	if !strings.Contains(out, "Cache Summary") {
		t.Fatalf("expected title, got:\n%s", out)
	}
	if !strings.Contains(out, "Good Cache:") {
		t.Fatalf("expected good cache section, got:\n%s", out)
	}
}

func TestCacheReportMarkdown(t *testing.T) {
	out := CacheReport(sampleDetails(), true)
	if !strings.Contains(out, "# Cache Summary") {
		t.Fatalf("expected markdown heading, got:\n%s", out)
	}
	if !strings.Contains(out, "| Metric | Value |") {
		t.Fatalf("expected markdown table, got:\n%s", out)
	}
}

func TestSecurityReportText(t *testing.T) {
	out := SecurityReport(sampleDetails(), false)
	if !strings.Contains(out, "Security Headers Analysis") {
		t.Fatalf("expected title, got:\n%s", out)
	}
	if !strings.Contains(out, "example.com") {
		t.Fatalf("expected domain, got:\n%s", out)
	}
}

func TestSecurityReportMarkdown(t *testing.T) {
	out := SecurityReport(sampleDetails(), true)
	if !strings.Contains(out, "# Security Headers Analysis") {
		t.Fatalf("expected markdown heading, got:\n%s", out)
	}
	if !strings.Contains(out, "| Domain | Required Missing | Optional Missing |") {
		t.Fatalf("expected summary table, got:\n%s", out)
	}
	if !strings.Contains(out, "## example.com") {
		t.Fatalf("expected per-domain section, got:\n%s", out)
	}
}

func TestReportsEmpty(t *testing.T) {
	for _, f := range []func([]*har.EntryDetails, bool) string{
		StatsReport, PerfReport, CacheReport, SecurityReport,
	} {
		if out := f(nil, false); out == "" {
			t.Fatalf("expected non-empty output for empty entries")
		}
		if out := f(nil, true); out == "" {
			t.Fatalf("expected non-empty markdown output for empty entries")
		}
	}
}
