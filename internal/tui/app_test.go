package tui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/harboring/harboring/internal/har"
	"github.com/harboring/harboring/internal/theme"
)

func makeApp() AppModel {
	root := &har.Node{Kind: har.Object, Key: "log"}
	entries := &har.Node{Kind: har.Array, Key: "entries"}
	n1 := &har.Node{Kind: har.Object, Key: "0"}
	url := &har.Node{Kind: har.String, Key: "url", Value: "http://example.com"}
	n1.Children = []*har.Node{url}
	entries.Children = []*har.Node{n1}
	root.Children = []*har.Node{entries}

	th := theme.New()
	m := NewApp(root, "test.har", th)
	result, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 24})
	return result.(AppModel)
}

func TestFooterAlwaysAtBottom(t *testing.T) {
	m := makeApp()

	view := m.View().Content
	lines := strings.Split(view, "\n")

	if len(lines) != 24 {
		t.Fatalf("expected 24 lines, got %d", len(lines))
	}

	last := lines[len(lines)-1]
	if !strings.Contains(last, "test.har") {
		t.Fatalf("expected footer on last line, got %q", last)
	}
}

func TestFooterAtBottomSmallContent(t *testing.T) {
	m := makeApp()
	m.tree.SetCursor(0)

	view := m.View().Content
	lines := strings.Split(view, "\n")

	if len(lines) != 24 {
		t.Fatalf("expected 24 lines even with short content, got %d", len(lines))
	}

	last := lines[len(lines)-1]
	if !strings.Contains(last, "test.har") {
		t.Fatalf("expected footer on last line, got %q", last)
	}
}

func TestSecurityModeSwitch(t *testing.T) {
	m := makeApp()

	// Press "6" to switch to security mode
	result, _ := m.handleKey(tea.KeyPressMsg{Text: "6"})
	m2 := result.(AppModel)

	if m2.mode != ModeSecurity {
		t.Fatalf("expected ModeSecurity, got %d", m2.mode)
	}

	view := m2.View().Content
	if !strings.Contains(view, "Security") {
		t.Errorf("expected security content in view")
	}
}

func TestRequestsTypeFilterModeToggle(t *testing.T) {
	m := makeApp()
	m.mode = ModeRequests

	// Press "f" to enter filter mode.
	res, _ := m.Update(tea.KeyPressMsg{Text: "f"})
	m2 := res.(AppModel)
	if !m2.requests.FilterMode() {
		t.Fatal("expected filter mode active after pressing f")
	}

	// Toggle a type with space.
	res, _ = m2.Update(tea.KeyPressMsg{Code: tea.KeySpace})
	m3 := res.(AppModel)
	if !m3.requests.FilterMode() {
		t.Fatal("expected filter mode still active after space")
	}

	// Exit filter mode with esc.
	res, _ = m3.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	m4 := res.(AppModel)
	if m4.requests.FilterMode() {
		t.Fatal("expected filter mode inactive after esc")
	}
}
