package tui

import (
	"regexp"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/harboring/harboring/internal/har"
	"github.com/harboring/harboring/internal/theme"
)

func makeDetailEntry() *har.EntryDetails {
	return &har.EntryDetails{
		Method:       "GET",
		URL:          "http://example.com/very/long/path/that/scrolls",
		Status:       200,
		StatusText:   "OK",
		Size:         12345,
		Time:         1500.5,
		MimeType:     "text/html",
		Started:      "2024-01-01T00:00:00Z",
		DNS:          10.0,
		Connect:      20.0,
		SSL:          5.0,
		Send:         1.0,
		Wait:         100.0,
		Receive:      1364.5,
		CacheControl: "public, max-age=3600",
		MaxAge:       3600,
		ETag:         "\"abc123\"",
	}
}

func TestDetailModelScrolling(t *testing.T) {
	th := theme.New()
	entry := makeDetailEntry()
	m := NewDetailModel(entry, nil, th)
	m.viewport.SetWidth(60)
	m.viewport.SetHeight(10)
	m.refreshContent()

	y0 := m.viewport.YOffset()
	if y0 != 0 {
		t.Fatalf("expected initial YOffset 0, got %d", y0)
	}

	// Send down key
	result, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	m2 := result.(DetailModel)
	y1 := m2.viewport.YOffset()
	if y1 <= 0 {
		t.Fatalf("expected YOffset > 0 after down key, got %d", y1)
	}

	// Send up key
	result, _ = m2.Update(tea.KeyPressMsg{Code: tea.KeyUp})
	m3 := result.(DetailModel)
	y2 := m3.viewport.YOffset()
	if y2 >= y1 {
		t.Fatalf("expected YOffset to decrease after up key (was %d, now %d)", y1, y2)
	}
}

func TestDetailModelViewContainsScrollbar(t *testing.T) {
	th := theme.New()
	entry := makeDetailEntry()
	m := NewDetailModel(entry, nil, th)
	m.viewport.SetWidth(60)
	m.viewport.SetHeight(10)
	m.refreshContent()

	// The view should contain scrollbar characters
	view := m.View().Content
	if !strings.Contains(view, "▲") && !strings.Contains(view, "│") && !strings.Contains(view, "█") {
		t.Fatal("expected scrollbar characters in view, got:\n", view)
	}
}

func TestNewDetailModelContentPreserved(t *testing.T) {
	th := theme.New()
	entry := makeDetailEntry()
	m := NewDetailModel(entry, nil, th)

	// Content should be set on the viewport
	if m.viewport.YOffset() < 0 {
		t.Fatal("expected viewport to be initialized")
	}

	// View should not be empty
	view := m.View().Content
	if view == "" {
		t.Fatal("expected non-empty view")
	}
	if !strings.Contains(view, "GET") {
		t.Fatal("expected view to contain method")
	}
}

func TestRefreshContent(t *testing.T) {
	th := theme.New()
	entry := makeDetailEntry()
	m := NewDetailModel(entry, nil, th)

	// Scroll down
	m.viewport, _ = m.viewport.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	yBefore := m.viewport.YOffset()

	// Refresh content
	m.refreshContent()
	yAfter := m.viewport.YOffset()

	// YOffset should be preserved after refresh
	if yBefore != yAfter {
		t.Fatalf("refreshContent changed YOffset from %d to %d", yBefore, yAfter)
	}
}

func TestDetailWindowResize(t *testing.T) {
	th := theme.New()
	entry := makeDetailEntry()
	m := NewDetailModel(entry, nil, th)

	// Simulate window resize
	msg := tea.WindowSizeMsg{Width: 120, Height: 40}
	result, _ := m.Update(msg)
	m2 := result.(DetailModel)

	if m2.viewport.Width() <= 0 || m2.viewport.Height() <= 0 {
		t.Fatal("viewport dimensions should be positive after resize")
	}
}

func TestFormatDetailSize(t *testing.T) {
	tests := []struct {
		in   int64
		want string
	}{
		{0, "0 B (0 bytes)"},
		{512, "512 B (512 bytes)"},
		{1024, "1.0 KB (1024 bytes)"},
		{12345, "12.1 KB (12345 bytes)"},
		{1024 * 1024, "1.0 MB (1048576 bytes)"},
	}
	for _, tt := range tests {
		if got := formatDetailSize(tt.in); got != tt.want {
			t.Errorf("formatDetailSize(%d) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestRenderTimingBarValueBeforeBarAligned(t *testing.T) {
	th := theme.New()
	m := NewDetailModel(nil, nil, th)
	// Two different label lengths and values: bars must still start at the same column.
	lineShort := m.renderTimingBar("DNS", 10.0, 100.0, 30, th.GetStyle("timing_dns"))
	lineLong := m.renderTimingBar("Receive", 1364.5, 1500.5, 30, th.GetStyle("timing_receive"))

	clean := func(s string) string { return strings.TrimSuffix(s, "\n") }
	short, long := clean(lineShort), clean(lineLong)

	// Value should come before the bar.
	idxValShort := strings.Index(short, "10.00ms")
	idxBarShort := strings.Index(short, "█")
	if idxValShort < 0 || idxBarShort < 0 {
		t.Fatalf("expected value and bar in %q", short)
	}
	if idxValShort > idxBarShort {
		t.Fatalf("expected value before bar, got value@%d bar@%d: %q", idxValShort, idxBarShort, short)
	}

	// Bars must start at the same rune position on both lines.
	an := regexp.MustCompile("\x1b\\[[0-9;]*m")
	posBar := func(s string) int {
		return strings.Index(an.ReplaceAllString(s, ""), "█")
	}
	if posBar(short) != posBar(long) {
		t.Fatalf("bars not aligned:\n%q\n%q", short, long)
	}
	t.Logf("short: %q\nlong:  %q", short, long)
}
