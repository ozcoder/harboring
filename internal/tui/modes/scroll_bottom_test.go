package modes

import (
	"fmt"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/harboring/harboring/internal/har"
	"github.com/harboring/harboring/internal/theme"
)

const tlLayout = "2006-01-02T15:04:05.999Z07:00"

func tlTime(i int) string {
	base := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	return base.Add(time.Duration(i) * time.Millisecond).Format(tlLayout)
}

func mkEntries(n int) []*har.EntryDetails {
	entries := make([]*har.EntryDetails, n)
	for i := 0; i < n; i++ {
		entries[i] = &har.EntryDetails{
			Method:       "GET",
			URL:          fmt.Sprintf("item%d", i),
			URLPath:      fmt.Sprintf("item%d", i),
			Status:       200,
			Size:         int64(i),
			Time:         float64(i + 1),
			Started:      tlTime(i),
			ResourceType: "other",
		}
	}
	return entries
}

// TestTimelineScrollToBottom verifies the last item is reachable.
func TestTimelineScrollToBottom(t *testing.T) {
	entries := mkEntries(20)
	m := NewTimelineModel(entries, nil, theme.New())
	m.SetViewportSize(40, 6) // bodyHeight 6 -> viewH 5 visible rows

	for i := 0; i < 40; i++ {
		m, _ = m.Update(tea.KeyPressMsg{Text: "j"})
	}
	if !strings.Contains(m.View(), "item19") {
		t.Fatalf("last timeline item 'item19' not visible after scrolling to bottom")
	}
}

// TestRequestsScrollToBottom verifies the last item is reachable.
func TestRequestsScrollToBottom(t *testing.T) {
	entries := mkEntries(20)
	m := NewRequestsModel(entries, nil, theme.New())
	m.SetViewportSize(60, 9) // bodyHeight 9 -> dataHeight 6

	for i := 0; i < 40; i++ {
		m, _ = m.Update(tea.KeyPressMsg{Text: "j"})
	}
	if !strings.Contains(m.View(), "item19") {
		t.Fatalf("last requests item 'item19' not visible after scrolling to bottom")
	}
}

// TestTimingsScrollToBottom verifies the last item is reachable.
func TestTimingsScrollToBottom(t *testing.T) {
	entries := mkEntries(20)
	m := NewTimingsModel(entries, nil, theme.New())
	m.SetViewportSize(60, 8) // bodyHeight 8

	for i := 0; i < 40; i++ {
		m, _ = m.Update(tea.KeyPressMsg{Text: "j"})
	}
	if !strings.Contains(m.View(), "item19") {
		t.Fatalf("last timings item 'item19' not visible after scrolling to bottom")
	}
}
