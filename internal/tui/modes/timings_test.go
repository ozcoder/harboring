package modes

import (
	"regexp"
	"strings"
	"testing"
	"unicode/utf8"

	tea "charm.land/bubbletea/v2"
	"github.com/harboring/harboring/internal/har"
	"github.com/harboring/harboring/internal/theme"
)

var ansiRegexp = regexp.MustCompile("\x1b\\[[0-9;]*m")

func stripANSI(s string) string {
	return ansiRegexp.ReplaceAllString(s, "")
}

func TestTimingsWidthFillsTerminal(t *testing.T) {
	th := theme.New()
	e := &har.EntryDetails{URL: "http://example.com/some/long/path/to/resource.js", URLPath: "/some/long/path/to/resource.js", Time: 100, Wait: 100}
	m := NewTimingsModel([]*har.EntryDetails{e}, []*har.Node{nil}, th)
	m.SetViewportSize(200, 30)

	view := m.View()
	lines := strings.Split(strings.TrimSuffix(view, "\n"), "\n")
	// line 0 is the sticky horizontal scrollbar row (blank until needed),
	// line 1 is the first request row.
	if trimmed := strings.TrimSpace(stripANSI(lines[0])); trimmed != "" {
		t.Fatalf("expected blank scrollbar row on first view, got: %q", trimmed)
	}
	cleaned := stripANSI(lines[1])
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

func TestTimingsURLColumnHugsPath(t *testing.T) {
	th := theme.New()
	e := &har.EntryDetails{URL: "http://example.com/short", URLPath: "/short", Time: 100, Wait: 100}
	m := NewTimingsModel([]*har.EntryDetails{e}, []*har.Node{nil}, th)
	m.SetViewportSize(200, 30)

	view := m.View()
	lines := strings.Split(strings.TrimSuffix(view, "\n"), "\n")
	cleaned := stripANSI(lines[1])

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

func TestTimingsNoTimingsNoBar(t *testing.T) {
	th := theme.New()
	e := &har.EntryDetails{URL: "http://example.com/no-timings", URLPath: "/no-timings", Time: 0, Wait: 0}
	m := NewTimingsModel([]*har.EntryDetails{e}, []*har.Node{nil}, th)
	m.SetViewportSize(200, 30)

	view := m.View()
	lines := strings.Split(strings.TrimSuffix(view, "\n"), "\n")
	cleaned := stripANSI(lines[1])

	if strings.Contains(cleaned, "\u2588") {
		t.Fatalf("expected no graph for entry without timings, got: %q", cleaned)
	}
	// Should still show the URL path
	if !strings.Contains(cleaned, "/no-timings") {
		t.Fatalf("expected path visible, got: %q", cleaned)
	}
}

func TestTimingsBarScalesWithMaxTime(t *testing.T) {
	th := theme.New()
	long := &har.EntryDetails{URL: "http://a/long", URLPath: "/long", Wait: 1000, Receive: 100}
	short := &har.EntryDetails{URL: "http://a/short", URLPath: "/short", Wait: 100, Receive: 10}
	m := NewTimingsModel([]*har.EntryDetails{long, short}, []*har.Node{nil, nil}, th)
	m.SetViewportSize(200, 30)

	lines := strings.Split(strings.TrimSuffix(stripANSI(m.View()), "\n"), "\n")
	longBar := countRunes(lines[1], '\u2588')
	shortBar := countRunes(lines[2], '\u2588')
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

func TestTimingsPlusShowsScrollbarAndGrowsBarArea(t *testing.T) {
	th := theme.New()
	e := &har.EntryDetails{URL: "http://a/one", URLPath: "/one", Wait: 100}
	m := NewTimingsModel([]*har.EntryDetails{e}, []*har.Node{nil}, th)
	m.SetViewportSize(200, 30)

	if m.barGraphWidth != 0 {
		t.Fatalf("expected bar graph area to start at 0 (auto), got %d", m.barGraphWidth)
	}

	// First view: blank space where the scrollbar would go.
	lines := strings.Split(strings.TrimSuffix(stripANSI(m.View()), "\n"), "\n")
	if trimmed := strings.TrimSpace(lines[0]); trimmed != "" {
		t.Fatalf("expected blank scrollbar row before widening, got: %q", trimmed)
	}

	// + increases the bar graph area and shows the horizontal scrollbar.
	m, _ = m.Update(tea.KeyPressMsg{Text: "+"})
	if m.barGraphWidth == 0 {
		t.Fatal("expected bar graph area width to increase on +")
	}
	before := m.barGraphWidth
	m, _ = m.Update(tea.KeyPressMsg{Text: "+"})
	if m.barGraphWidth <= before {
		t.Fatalf("expected bar graph area to keep increasing, got %d after %d", m.barGraphWidth, before)
	}
	lines = strings.Split(strings.TrimSuffix(stripANSI(m.View()), "\n"), "\n")
	if !strings.Contains(lines[0], "\u2588") {
		t.Fatalf("expected horizontal scrollbar thumb after widening, got: %q", lines[0])
	}

	// - reduces the bar graph area back to auto.
	m, _ = m.Update(tea.KeyPressMsg{Text: "-"})
	m, _ = m.Update(tea.KeyPressMsg{Text: "-"})
	m, _ = m.Update(tea.KeyPressMsg{Text: "-"})
	for i := 0; i < 20 && m.barGraphWidth > 0; i++ {
		m, _ = m.Update(tea.KeyPressMsg{Text: "-"})
	}
	if m.barGraphWidth != 0 {
		t.Fatalf("expected - to reduce bar graph area back to auto, got %d", m.barGraphWidth)
	}
	lines = strings.Split(strings.TrimSuffix(stripANSI(m.View()), "\n"), "\n")
	if trimmed := strings.TrimSpace(lines[0]); trimmed != "" {
		t.Fatalf("expected blank scrollbar row after reducing, got: %q", trimmed)
	}
}

func TestTimingsHorizontalScrollKeepsRequestsFixed(t *testing.T) {
	th := theme.New()
	e := &har.EntryDetails{URL: "http://a/one", URLPath: "/one", Wait: 100}
	m := NewTimingsModel([]*har.EntryDetails{e}, []*har.Node{nil}, th)
	m.SetViewportSize(200, 30)

	m, _ = m.Update(tea.KeyPressMsg{Text: "+"})
	maxScroll := m.barGraphWidthEffective() - m.visibleBarWidth()
	if maxScroll <= 0 {
		t.Fatalf("expected scroll room after widening, got %d", maxScroll)
	}

	before := m.hScroll
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyRight})
	if m.hScroll <= before {
		t.Fatalf("expected right to increase hScroll, got %d after %d", m.hScroll, before)
	}

	thumbPos := func(view string) int {
		lines := strings.Split(strings.TrimSuffix(stripANSI(view), "\n"), "\n")
		return runeIndex(lines[0], "\u2588")
	}

	pos0 := thumbPos(m.View())
	// The request rows are not part of the horizontal scroll: the URL column
	// stays at the same position regardless of the scroll offset.
	lineBefore := stripANSI(m.View())
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyLeft})
	if m.hScroll != 0 {
		t.Fatalf("expected left to reset hScroll to 0, got %d", m.hScroll)
	}
	pos1 := thumbPos(m.View())
	if pos0 == pos1 {
		t.Fatalf("expected scrollbar thumb to move with scroll, stayed at %d", pos1)
	}
	lineAfter := stripANSI(m.View())
	urlBefore := strings.Split(lineBefore, "\n")[1]
	urlAfter := strings.Split(lineAfter, "\n")[1]
	if runeIndex(urlBefore, "/one") != runeIndex(urlAfter, "/one") {
		t.Fatalf("request URL moved during horizontal scroll:\n%q\n%q", urlBefore, urlAfter)
	}

	// Clamp: pressing right many times must not exceed the bar graph width.
	for i := 0; i < 50; i++ {
		m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyRight})
	}
	if m.hScroll > m.barGraphWidthEffective()-m.visibleBarWidth() {
		t.Fatalf("hScroll %d exceeds max %d", m.hScroll, m.barGraphWidthEffective()-m.visibleBarWidth())
	}
}

func TestTimingsSelectedRowHighlightCoversFullRow(t *testing.T) {
	th := theme.New()
	e := &har.EntryDetails{URL: "http://a/one", URLPath: "/one", Wait: 100}
	m := NewTimingsModel([]*har.EntryDetails{e}, []*har.Node{nil}, th)
	m.SetViewportSize(200, 30)

	view := m.View()
	lines := strings.Split(strings.TrimSuffix(stripANSI(view), "\n"), "\n")
	if got := utf8.RuneCountInString(lines[1]); got != m.width {
		t.Fatalf("selected row width = %d, want %d (full row)\n%q", got, m.width, lines[1])
	}
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
