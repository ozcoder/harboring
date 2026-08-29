package modes

import (
	"regexp"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/harboring/harboring/internal/har"
	"github.com/harboring/harboring/internal/theme"
)

func TestTimelineHorizontalScrollBarGraph(t *testing.T) {
	th := theme.New()
	entries := []*har.EntryDetails{
		{Method: "GET", URL: "http://a/one", URLPath: "/one", Status: 200, Time: 10, Started: tlTime(0), ResourceType: "other"},
		{Method: "GET", URL: "http://a/two", URLPath: "/two", Status: 200, Time: 10, Started: tlTime(20), ResourceType: "other"},
	}
	m := NewTimelineModel(entries, nil, th)
	m.SetViewportSize(120, 20)

	// First view: blank space where the horizontal scrollbar would go.
	lines := strings.Split(strings.TrimSuffix(stripANSI(m.View()), "\n"), "\n")
	if trimmed := strings.TrimSpace(lines[0]); trimmed != "" {
		t.Fatalf("expected blank scrollbar row on first view, got: %q", trimmed)
	}

	// + widens the timeline area and shows the scrollbar.
	m, _ = m.Update(tea.KeyPressMsg{Text: "+"})
	if m.barGraphWidth == 0 {
		t.Fatal("expected + to widen the timeline area")
	}
	before := m.barGraphWidth
	m, _ = m.Update(tea.KeyPressMsg{Text: "+"})
	if m.barGraphWidth <= before {
		t.Fatalf("expected timeline area to keep growing, got %d after %d", m.barGraphWidth, before)
	}
	lines = strings.Split(strings.TrimSuffix(stripANSI(m.View()), "\n"), "\n")
	if !strings.Contains(lines[0], "\u2588") {
		t.Fatalf("expected horizontal scrollbar thumb after widening, got: %q", lines[0])
	}

	// Scroll right: the request labels stay put, the time scale scrolls.
	labelBefore := lines[1]
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyRight})
	if m.hScroll <= 0 {
		t.Fatalf("expected right to increase hScroll, got %d", m.hScroll)
	}
	lines = strings.Split(strings.TrimSuffix(stripANSI(m.View()), "\n"), "\n")
	labelAfter := lines[1]
	if runeIndex(labelBefore, "/one") != runeIndex(labelAfter, "/one") {
		t.Fatalf("request label moved during horizontal scroll:\n%q\n%q", labelBefore, labelAfter)
	}
	zeroMarker := regexp.MustCompile(`\b0ms\b`)
	if !zeroMarker.MatchString(labelBefore) {
		t.Fatalf("expected 0ms scale label before scrolling, got: %q", labelBefore)
	}
	if zeroMarker.MatchString(labelAfter) {
		t.Fatalf("expected the 0ms scale label to scroll out of view, got: %q", labelAfter)
	}

	// left returns to the start of the timeline.
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyLeft})
	if m.hScroll != 0 {
		t.Fatalf("expected left to reset hScroll to 0, got %d", m.hScroll)
	}
	lines = strings.Split(strings.TrimSuffix(stripANSI(m.View()), "\n"), "\n")
	if !zeroMarker.MatchString(lines[1]) {
		t.Fatalf("expected 0ms scale label visible again after scrolling back, got: %q", lines[1])
	}

	// Clamp: many rights must not exceed the timeline width.
	for i := 0; i < 50; i++ {
		m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyRight})
	}
	if m.hScroll > m.barGraphWidthEffective()-m.timelineVisibleWidth() {
		t.Fatalf("hScroll %d exceeds max %d", m.hScroll, m.barGraphWidthEffective()-m.timelineVisibleWidth())
	}

	// - reduces back to auto and blanks the scrollbar row again.
	for i := 0; i < 20 && m.barGraphWidth > 0; i++ {
		m, _ = m.Update(tea.KeyPressMsg{Text: "-"})
	}
	if m.barGraphWidth != 0 {
		t.Fatalf("expected - to reduce timeline area back to auto, got %d", m.barGraphWidth)
	}
	lines = strings.Split(strings.TrimSuffix(stripANSI(m.View()), "\n"), "\n")
	if trimmed := strings.TrimSpace(lines[0]); trimmed != "" {
		t.Fatalf("expected blank scrollbar row after reducing, got: %q", trimmed)
	}
}

func TestTimelineScaleShowsMoreMarkersWhenWidened(t *testing.T) {
	th := theme.New()
	entries := []*har.EntryDetails{
		{Method: "GET", URL: "http://a/x", URLPath: "/x", Status: 200, Time: 100, Started: tlTime(0), ResourceType: "other"},
		{Method: "GET", URL: "http://a/y", URLPath: "/y", Status: 200, Time: 80, Started: tlTime(20), ResourceType: "other"},
	}
	m := NewTimelineModel(entries, nil, th)
	m.SetViewportSize(60, 20)

	// line 1 is the Request / scale header row.
	header := func() string {
		lines := strings.Split(strings.TrimSuffix(stripANSI(m.View()), "\n"), "\n")
		return lines[1]
	}

	fit := header()
	if !strings.Contains(fit, "0ms") || !strings.Contains(fit, "20ms") {
		t.Fatalf("expected coarse markers at fit width, got: %q", fit)
	}
	if strings.Contains(fit, "10ms") {
		t.Fatalf("expected no 10ms marker at fit width, got: %q", fit)
	}

	for i := 0; i < 4; i++ {
		m, _ = m.Update(tea.KeyPressMsg{Text: "+"})
	}
	widened := header()
	if !strings.Contains(widened, "10ms") {
		t.Fatalf("expected a 10ms marker to appear after widening, got: %q", widened)
	}
	if !strings.Contains(widened, "0ms") {
		t.Fatalf("0ms marker should still be present, got: %q", widened)
	}
}
