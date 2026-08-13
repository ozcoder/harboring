package tui

import (
	"regexp"
	"strings"
	"testing"

	"github.com/harboring/harboring/internal/har"
	"github.com/harboring/harboring/internal/theme"
	"github.com/harboring/harboring/internal/tui/modes"
)

var ansiRegex = regexp.MustCompile("\x1b\\[[0-9;]*m")

func stripANSI(s string) string {
	return ansiRegex.ReplaceAllString(s, "")
}

func TestTreeScrollbarAndIndent(t *testing.T) {
	root := &har.Node{Kind: har.Object, Key: "log"}
	entries := &har.Node{Kind: har.Array, Key: "entries"}
	n1 := &har.Node{Kind: har.Object, Key: "0"}
	s1 := &har.Node{Kind: har.String, Key: "url", Value: "http://x"}
	n1.Children = []*har.Node{s1}
	req := &har.Node{Kind: har.Object, Key: "request"}
	n2 := &har.Node{Kind: har.Object, Key: "0"}
	s2 := &har.Node{Kind: har.String, Key: "method", Value: "GET"}
	n2.Children = []*har.Node{s2}
	req.Children = []*har.Node{n2}
	entries.Children = []*har.Node{n1, req}
	root.Children = []*har.Node{entries}

	th := theme.New()
	tm := modes.NewTreeModel(root, th, RenderNodeLine)
	tm.SetViewportSize(60, 8)

	view := stripANSI(tm.View())
	if !strings.Contains(view, "▲") {
		t.Error("expected scrollbar top marker in tree view")
	}
	if !strings.Contains(view, "│") {
		t.Error("expected scrollbar track in tree view")
	}
	if !strings.Contains(view, "  ▾") {
		t.Error("expected child branch indented by 2 spaces")
	}
	if !strings.Contains(view, "    ▾") {
		t.Error("expected grandchild indented by 4 spaces")
	}
}
