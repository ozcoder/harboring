package modes

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/harboring/harboring/internal/har"
	"github.com/harboring/harboring/internal/theme"
)

func runeIndex(s, sub string) int {
	byteIdx := strings.Index(s, sub)
	if byteIdx < 0 {
		return -1
	}
	return utf8.RuneCountInString(s[:byteIdx])
}

func TestRequestsIconKeepsColumnsAligned(t *testing.T) {
	th := theme.New()

	// Both entries: same method width, status, size, time.
	// One has the un-compressed warning icon, the other does not.
	withIcon := &har.EntryDetails{
		Method: "GET", URL: "http://a/image.svg", URLPath: "/image.svg",
		Status: 200, Size: 1024, Time: 50,
		ShouldBeCompressed: true, IsCompressed: false,
	}
	noIcon := &har.EntryDetails{
		Method: "GET", URL: "http://a/image.svg", URLPath: "/image.svg",
		Status: 200, Size: 1024, Time: 50,
		ShouldBeCompressed: true, IsCompressed: true,
	}
	m := NewRequestsModel([]*har.EntryDetails{noIcon, withIcon}, []*har.Node{nil, nil}, th)
	m.SetViewportSize(120, 30)

	lines := strings.Split(strings.TrimSuffix(stripANSI(m.View()), "\n"), "\n")
	// line 0 = filter bar, line 1 = header, line 2 = separator, line 3 = noIcon, line 4 = withIcon
	base := lines[3]
	icon := lines[4]

	// The URL/status/size/time columns must start at the same rune position on both lines.
	for _, col := range []string{"/image.svg", "200", "1.0KB", "50ms"} {
		idxBase := runeIndex(base, col)
		idxIcon := runeIndex(icon, col)
		if idxBase != idxIcon {
			t.Fatalf("column %q misaligned: base=%d icon=%d\nbase: %q\nicon: %q", col, idxBase, idxIcon, base, icon)
		}
	}
	t.Logf("base: %q\nicon: %q", base, icon)
}

func TestRequestsSelectedRowHighlightCoversFullRow(t *testing.T) {
	th := theme.New()
	entries := []*har.EntryDetails{
		{Method: "GET", URL: "http://a/1.png", URLPath: "/1.png", ResourceType: "image", Status: 200, Size: 10, Time: 1},
		{Method: "GET", URL: "http://a/2.js", URLPath: "/2.js", ResourceType: "script", Status: 200, Size: 10, Time: 1},
	}
	m := NewRequestsModel(entries, []*har.Node{nil, nil}, th)
	m.SetViewportSize(120, 30)

	view := m.View()
	// The selected row must carry the highlight background, not just the number column.
	if !strings.Contains(view, "48;2;") {
		t.Fatalf("expected a background highlight on the selected row, got:\n%s", view)
	}

	lines := strings.Split(strings.TrimSuffix(stripANSI(view), "\n"), "\n")
	// line 0 = filter bar, line 1 = header, line 2 = separator, line 3 = selected row.
	sel := lines[3]
	if got := utf8.RuneCountInString(sel); got != m.width {
		t.Fatalf("selected row width = %d, want %d (full row)\n%q", got, m.width, sel)
	}
	unsel := lines[4]
	if got := utf8.RuneCountInString(unsel); got >= m.width {
		t.Fatalf("non-selected row should not span the full width, got %d\n%q", got, unsel)
	}
}

func TestRequestsFilterBarRendered(t *testing.T) {
	th := theme.New()
	entries := []*har.EntryDetails{
		{Method: "GET", URL: "http://a/1.png", URLPath: "/1.png", ResourceType: "image", Status: 200, Size: 10, Time: 1},
		{Method: "GET", URL: "http://a/2.js", URLPath: "/2.js", ResourceType: "script", Status: 200, Size: 10, Time: 1},
		{Method: "GET", URL: "http://a/3.css", URLPath: "/3.css", ResourceType: "stylesheet", Status: 200, Size: 10, Time: 1},
	}
	m := NewRequestsModel(entries, []*har.Node{nil, nil, nil}, th)
	m.SetViewportSize(120, 30)

	view := m.View()
	lines := strings.Split(strings.TrimSuffix(stripANSI(view), "\n"), "\n")
	bar := lines[0]
	for _, want := range []string{"All:3", "Img:1", "JS:1", "CSS:1"} {
		if !strings.Contains(bar, want) {
			t.Fatalf("filter bar missing %q: %q", want, bar)
		}
	}
}

func TestRequestsTypeFilterFiltersEntries(t *testing.T) {
	th := theme.New()
	entries := []*har.EntryDetails{
		{Method: "GET", URL: "http://a/1.png", URLPath: "/1.png", ResourceType: "image", Status: 200, Size: 10, Time: 1},
		{Method: "GET", URL: "http://a/2.js", URLPath: "/2.js", ResourceType: "script", Status: 200, Size: 10, Time: 1},
		{Method: "GET", URL: "http://a/3.css", URLPath: "/3.css", ResourceType: "stylesheet", Status: 200, Size: 10, Time: 1},
	}
	m := NewRequestsModel(entries, []*har.Node{nil, nil, nil}, th)

	// All three visible initially.
	view := stripANSI(m.View())
	if strings.Count(view, "/") < 3 {
		t.Fatalf("expected all entries visible, got:\n%s", view)
	}

	// Move cursor to "script" (index 3 in chips) and toggle it.
	for i := 0; i < 3; i++ {
		m.MoveTypeCursor(1)
	}
	m.ToggleTypeAtCursor()

	view = stripANSI(m.View())
	if !strings.Contains(view, "/2.js") {
		t.Fatalf("expected /2.js after filter, got:\n%s", view)
	}
	if strings.Contains(view, "/1.png") || strings.Contains(view, "/3.css") {
		t.Fatalf("expected only script entries after filter, got:\n%s", view)
	}
}
