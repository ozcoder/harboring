package modes

import (
	"fmt"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/harboring/harboring/internal/har"
	"github.com/harboring/harboring/internal/theme"
)

// TestTreeExpandKeepsChildVisible ensures that expanding a node sitting on the
// last body row scrolls so its first child remains visible instead of being
// hidden behind the footer.
func TestTreeExpandKeepsChildVisible(t *testing.T) {
	root := &har.Node{Kind: har.Object, Key: "root", Children: []*har.Node{
		{Kind: har.Object, Key: "X", Children: []*har.Node{
			{Kind: har.String, Key: "c1", Value: "1"},
			{Kind: har.String, Key: "c2", Value: "2"},
		}},
	}}
	th := theme.New()
	m := NewTreeModel(root, th, func(n *har.Node, th *theme.Engine, sel bool, w, d int) string {
		return n.Key
	})
	m.SetViewportSize(40, 2) // body height of 2

	// X starts collapsed so the cursor sits on the last node.
	m.root.Collapsed = false
	X := m.root.Children[0]
	X.Collapsed = true
	m.refreshNodes()
	m.Cursor = 1 // node X
	m.updateScrollY()

	if m.Cursor != len(m.allNodes)-1 {
		t.Fatalf("precondition: cursor should be last node, got %d of %d", m.Cursor, len(m.allNodes))
	}

	// Expand X while it is on the last visible row.
	X.Collapsed = false
	m.refreshNodes()
	m.updateScrollY()

	// The view must scroll so the first child is visible (scrollY advanced).
	if m.scrollY != 1 {
		t.Fatalf("expected scrollY 1 after expand-at-bottom, got %d", m.scrollY)
	}

	lines := strings.Split(strings.TrimRight(m.View(), "\n"), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 visible lines, got %d: %v", len(lines), lines)
	}
	if !strings.Contains(lines[1], "c1") {
		t.Fatalf("expected first child c1 visible after expand, got %q", lines[1])
	}
}

// TestTreeScrollRevealsTrailingLines ensures that when the cursor sits on the
// last node, the document bottom (its trailing closing brackets) is scrolled
// into view rather than being permanently hidden behind the footer.
func TestTreeScrollRevealsTrailingLines(t *testing.T) {
	root := &har.Node{Kind: har.Object, Key: "root", Children: []*har.Node{
		{Kind: har.Object, Key: "A", Children: []*har.Node{
			{Kind: har.String, Key: "a1", Value: "1"},
			{Kind: har.String, Key: "a2", Value: "2"},
		}},
		{Kind: har.Object, Key: "B", Children: []*har.Node{
			{Kind: har.String, Key: "b1", Value: "x"},
		}},
	}}
	th := theme.New()
	m := NewTreeModel(root, th, func(n *har.Node, th *theme.Engine, sel bool, w, d int) string {
		return n.Key
	})
	m.SetViewportSize(40, 4)
	m.root.ExpandAll()
	m.refreshNodes()
	m.Cursor = len(m.allNodes) - 1 // last node: b1
	m.updateScrollY()

	lines := strings.Split(strings.TrimRight(m.View(), "\n"), "\n")
	// Drop the leading scrollbar char (▼/█/space) prepended by View().
	lastRunes := []rune(lines[len(lines)-1])
	last := strings.TrimSpace(string(lastRunes[1:]))
	if last != "}" {
		t.Fatalf("expected trailing closing bracket visible at bottom, got %q (lines: %v)", last, lines)
	}
}

// TestTreePageDown ensures PageUp/PageDown move the cursor by a full page and
// engage scrolling, both directions and clamped at the ends.
func TestTreePageDown(t *testing.T) {
	children := make([]*har.Node, 20)
	for i := range children {
		children[i] = &har.Node{Kind: har.String, Key: fmt.Sprintf("c%d", i), Value: "x"}
	}
	root := &har.Node{Kind: har.Object, Key: "root", Children: children}
	th := theme.New()
	m := NewTreeModel(root, th, func(n *har.Node, th *theme.Engine, sel bool, w, d int) string {
		return n.Key
	})
	m.SetViewportSize(40, 5)
	m.root.ExpandAll()
	m.refreshNodes()

	start := m.Cursor // 0
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyPgDown})
	if m.Cursor != start+5 {
		t.Fatalf("expected cursor %d after one page down, got %d", start+5, m.Cursor)
	}
	if m.scrollY <= 0 {
		t.Fatalf("expected scrollY > 0 after page down, got %d", m.scrollY)
	}

	for i := 0; i < 10; i++ {
		m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyPgDown})
	}
	if m.Cursor != len(m.allNodes)-1 {
		t.Fatalf("expected cursor at last node %d, got %d", len(m.allNodes)-1, m.Cursor)
	}

	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyPgUp})
	if m.Cursor != len(m.allNodes)-1-5 {
		t.Fatalf("expected cursor %d after one page up, got %d", len(m.allNodes)-1-5, m.Cursor)
	}
}
