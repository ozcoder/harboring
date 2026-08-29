package tui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

const popoverAlpha = "ABCDEFGHIJKLMNOPQRST"

// popoverDummy is a minimal tea.Model whose view renders a fixed string, used
// to exercise the popover's modal compositing.
type popoverDummy struct{}

func (popoverDummy) Init() tea.Cmd { return nil }
func (popoverDummy) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	return popoverDummy{}, nil
}
func (popoverDummy) View() tea.View { return tea.View{Content: "hi"} }

// TestPopoverLayerCompositing verifies the popover is composited as a true
// modal over the underlying view: every line stays at full width, the base
// content to the right of the box remains visible, the rows outside the box are
// intact, and the box itself renders its border and content.
func TestPopoverLayerCompositing(t *testing.T) {
	width := 100
	height := 20
	long := strings.Repeat(popoverAlpha, 5) // 100 columns
	base := strings.Repeat(long+"\n", height-1) + long

	p := NewPopover(nil)
	p.Open(popoverDummy{}, width, height)

	out := p.Render(base)
	lines := strings.Split(out, "\n")
	if len(lines) != height {
		t.Fatalf("expected %d lines, got %d", height, len(lines))
	}

	// Every line must render at the full screen width.
	for i, ln := range lines {
		if got := lipgloss.Width(ln); got != width {
			t.Fatalf("line %d width = %d, want %d", i, got, width)
		}
	}

	// Content to right of the box must remain visible.
	foundRight := false
	for _, ln := range lines {
		if strings.Contains(stripANSI(ln), "QRST") {
			foundRight = true
			break
		}
	}
	if !foundRight {
		t.Fatal("expected underlying content to the right of the popover to remain visible")
	}

	// Top row (outside the box) shows full base content.
	if !strings.Contains(stripANSI(lines[0]), popoverAlpha) {
		t.Fatal("expected top row to show full base content")
	}

	// The modal box border renders.
	foundBorder := false
	for _, ln := range lines {
		plain := stripANSI(ln)
		if strings.Contains(plain, "╭") || strings.Contains(plain, "╰") {
			foundBorder = true
			break
		}
	}
	if !foundBorder {
		t.Fatal("expected modal box border to be rendered")
	}

	// The modal content is rendered over the base.
	foundHi := false
	for _, ln := range lines {
		if strings.Contains(stripANSI(ln), "hi") {
			foundHi = true
			break
		}
	}
	if !foundHi {
		t.Fatal("expected modal content to be rendered")
	}
}
