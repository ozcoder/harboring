package tui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/harboring/harboring/internal/har"
	"github.com/harboring/harboring/internal/theme"
)

func RenderNodeLine(n *har.Node, th *theme.Engine, isSelected bool, width int, depth int) string {
	return renderNodeLine(n, th, isSelected, width, depth)
}

func renderNodeLine(n *har.Node, th *theme.Engine, isSelected bool, width int, depth int) string {
	indent := strings.Repeat("  ", depth)
	keyStyle := th.GetStyle("json_key")
	stringStyle := th.GetStyle("json_string")
	numberStyle := th.GetStyle("json_number")
	boolStyle := th.GetStyle("json_bool")
	nullStyle := th.GetStyle("json_null")
	unknownStyle := th.GetStyle("json_unknown")
	bracketStyle := th.GetStyle("json_bracket")

	nodeKey := n.Key
	if nodeKey == "" || nodeKey[0] == '[' {
		nodeKey = ""
	}

	switch n.Kind {
	case har.String:
		val := fmt.Sprintf("%q", n.Value)
		return buildLeafLine(indent, nodeKey, val, stringStyle, keyStyle, unknownStyle, n.IsKnown, th, isSelected, width)

	case har.Number:
		val := n.NumberText()
		return buildLeafLine(indent, nodeKey, val, numberStyle, keyStyle, unknownStyle, n.IsKnown, th, isSelected, width)

	case har.Bool:
		val := fmt.Sprintf("%v", n.Value)
		return buildLeafLine(indent, nodeKey, val, boolStyle, keyStyle, unknownStyle, n.IsKnown, th, isSelected, width)

	case har.Null:
		return buildLeafLine(indent, nodeKey, "null", nullStyle, keyStyle, unknownStyle, n.IsKnown, th, isSelected, width)

	case har.Object, har.Array:
		return buildContainerLine(n, indent, nodeKey, depth, bracketStyle, keyStyle, unknownStyle, th, isSelected, width)
	}

	return ""
}

// truncateWidth shortens s to at most max display columns, appending "...".
// It operates on rune boundaries so multi-byte characters are never split.
func truncateWidth(s string, max int) string {
	if max <= 0 {
		return ""
	}
	if lipgloss.Width(s) <= max {
		return s
	}
	if max <= 3 {
		return strings.Repeat(".", max)
	}
	var b strings.Builder
	w := 0
	for _, r := range s {
		rw := lipgloss.Width(string(r))
		if w+rw > max-3 {
			break
		}
		b.WriteRune(r)
		w += rw
	}
	return b.String() + "..."
}

func buildLeafLine(indent, key, val string, valStyle, keyStyle, unknownStyle lipgloss.Style, isKnown bool, th *theme.Engine, isSelected bool, width int) string {
	prefix := ""
	if key != "" {
		prefix = key + ": "
	}
	avail := width - lipgloss.Width(indent) - lipgloss.Width(prefix)
	if avail < 1 {
		avail = 1
	}
	if lipgloss.Width(val) > avail {
		val = truncateWidth(val, avail)
	}

	var sb strings.Builder
	sb.WriteString(indent)

	valStr := valStyle.Render(val)
	if !isKnown {
		valStr = unknownStyle.Render(val)
	}

	if key != "" {
		sb.WriteString(keyStyle.Render(key))
		sb.WriteString(": ")
	}
	sb.WriteString(valStr)

	line := sb.String()

	if isSelected {
		return th.GetStyle("selection").Render(line)
	}
	return line
}

func buildContainerLine(n *har.Node, indent, key string, depth int, bracketStyle, keyStyle, unknownStyle lipgloss.Style, th *theme.Engine, isSelected bool, width int) string {
	openBracket := "{"
	closeBracket := "}"
	count := len(n.Children)
	if n.Kind == har.Array {
		openBracket = "["
		closeBracket = "]"
	}

	budget := width - lipgloss.Width(indent)
	var sb strings.Builder
	write := func(s string) bool {
		w := lipgloss.Width(s)
		if w > budget {
			return false
		}
		sb.WriteString(s)
		budget -= w
		return true
	}

	sb.WriteString(indent)

	arrow := "▸ "
	if !n.Collapsed {
		arrow = "▾ "
	}
	if !write(bracketStyle.Render(arrow)) {
		return sb.String()
	}

	if key != "" {
		if !write(keyStyle.Render(key)) {
			return sb.String()
		}
		if !write(bracketStyle.Render(": ")) {
			return sb.String()
		}
	}

	if n.Collapsed {
		write(bracketStyle.Render(fmt.Sprintf("%s %d items %s", openBracket, count, closeBracket)))
	} else {
		write(bracketStyle.Render(openBracket))
	}

	line := sb.String()

	if isSelected {
		return th.GetStyle("selection").Render(line)
	}
	return line
}
