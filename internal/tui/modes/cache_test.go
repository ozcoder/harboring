package modes

import (
	"fmt"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/harboring/harboring/internal/har"
	"github.com/harboring/harboring/internal/theme"
)

// TestCachePageDown ensures PageUp/PageDown page through a scrollable cache
// section by the visible page size and clamp at both ends.
func TestCachePageDown(t *testing.T) {
	th := theme.New()
	entries := make([]*har.EntryDetails, 20)
	for i := range entries {
		// Status 200 with a long max-age lands in the "Good" cache section.
		entries[i] = &har.EntryDetails{
			Method: "GET", URL: fmt.Sprintf("http://a/%d", i),
			Status: 200, MaxAge: 600000,
		}
	}
	m := NewCacheModel(entries, nil, th)
	m.SetViewportSize(80, 24)

	// Switch to the Good section (20 entries) and start at the top.
	m.active = CacheSectionGood
	m.selection[CacheSectionGood] = 0
	m.scrollPos[CacheSectionGood] = 0

	// visibleLines = boxHeight(24/2=12) - 6 = 6.
	start := m.selection[CacheSectionGood]
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyPgDown})
	if m.selection[CacheSectionGood] != start+6 {
		t.Fatalf("expected selection %d after one page down, got %d", start+6, m.selection[CacheSectionGood])
	}

	for i := 0; i < 10; i++ {
		m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyPgDown})
	}
	if m.selection[CacheSectionGood] != 19 {
		t.Fatalf("expected selection at last entry 19, got %d", m.selection[CacheSectionGood])
	}

	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyPgUp})
	if m.selection[CacheSectionGood] != 13 {
		t.Fatalf("expected selection 13 after one page up, got %d", m.selection[CacheSectionGood])
	}
}
