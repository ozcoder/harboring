package modes

import (
	"regexp"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/harboring/harboring/internal/har"
	"github.com/harboring/harboring/internal/theme"
)

var ansiRegexp = regexp.MustCompile("\x1b\\[[0-9;]*m")

func stripANSI(s string) string {
	return ansiRegexp.ReplaceAllString(s, "")
}

func TestWaterfallWidthFillsTerminal(t *testing.T) {
	th := theme.New()
	e := &har.EntryDetails{URL: "http://example.com/some/long/path/to/resource.js", URLPath: "/some/long/path/to/resource.js", Time: 100, Wait: 100}
	m := NewWaterfallModel([]*har.EntryDetails{e}, []*har.Node{nil}, th)
	m.SetViewportSize(200, 30)

	view := m.View()
	lines := strings.Split(strings.TrimSuffix(view, "\n"), "\n")
	cleaned := stripANSI(lines[0])
	runes := utf8.RuneCountInString(cleaned)
	if runes < 195 {
		t.Fatalf("expected line near full 200 runes, got %d", runes)
	}
	// URL part should show the full path, not truncated
	if !strings.Contains(cleaned, "resource.js") {
		t.Fatalf("expected full path visible, got: %q", cleaned)
	}
	t.Logf("line runes: %d", runes)
}

func TestWaterfallURLColumnHugsPath(t *testing.T) {
	th := theme.New()
	e := &har.EntryDetails{URL: "http://example.com/short", URLPath: "/short", Time: 100, Wait: 100}
	m := NewWaterfallModel([]*har.EntryDetails{e}, []*har.Node{nil}, th)
	m.SetViewportSize(200, 30)

	view := m.View()
	lines := strings.Split(strings.TrimSuffix(view, "\n"), "\n")
	cleaned := stripANSI(lines[0])

	// URL column is sized to content, so the path ends right before the bars.
	// The path "/short" (6 chars) plus a small gap should be far less than
	// half the terminal width.
	idx := strings.Index(cleaned, "/short")
	if idx < 0 {
		t.Fatalf("path not found in: %q", cleaned)
	}
	barIdx := strings.Index(cleaned, "\u2588")
	if barIdx < 0 {
		t.Fatalf("no bar graph in: %q", cleaned)
	}
	if barIdx > 25 {
		t.Fatalf("expected bars to start soon after path, got gap of %d chars: %q", barIdx, cleaned)
	}
	t.Logf("bar starts at col %d", barIdx)
}

func TestWaterfallNoTimingsNoBar(t *testing.T) {
	th := theme.New()
	e := &har.EntryDetails{URL: "http://example.com/no-timings", URLPath: "/no-timings", Time: 0, Wait: 0}
	m := NewWaterfallModel([]*har.EntryDetails{e}, []*har.Node{nil}, th)
	m.SetViewportSize(200, 30)

	view := m.View()
	lines := strings.Split(strings.TrimSuffix(view, "\n"), "\n")
	cleaned := stripANSI(lines[0])

	if strings.Contains(cleaned, "\u2588") {
		t.Fatalf("expected no graph for entry without timings, got: %q", cleaned)
	}
	// Should still show the URL path
	if !strings.Contains(cleaned, "/no-timings") {
		t.Fatalf("expected path visible, got: %q", cleaned)
	}
}

func TestWaterfallBarScalesWithMaxTime(t *testing.T) {
	th := theme.New()
	long := &har.EntryDetails{URL: "http://a/long", URLPath: "/long", Wait: 1000, Receive: 100}
	short := &har.EntryDetails{URL: "http://a/short", URLPath: "/short", Wait: 100, Receive: 10}
	m := NewWaterfallModel([]*har.EntryDetails{long, short}, []*har.Node{nil, nil}, th)
	m.SetViewportSize(200, 30)

	lines := strings.Split(strings.TrimSuffix(stripANSI(m.View()), "\n"), "\n")
	longBar := countRunes(lines[0], '\u2588')
	shortBar := countRunes(lines[1], '\u2588')
	if shortBar >= longBar {
		t.Fatalf("expected short request bar (%d) < long request bar (%d)", shortBar, longBar)
	}
	if longBar < 100 {
		t.Fatalf("expected long bar to be substantial, got %d", longBar)
	}
	if shortBar > longBar/3 {
		t.Fatalf("expected short bar much smaller than long bar, got short=%d long=%d", shortBar, longBar)
	}
	t.Logf("long bar: %d, short bar: %d", longBar, shortBar)
}

func countRunes(s string, r rune) int {
	n := 0
	for _, c := range s {
		if c == r {
			n++
		}
	}
	return n
}
