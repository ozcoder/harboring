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
		val := fmt.Sprintf("%v", n.Value)
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

func buildLeafLine(indent, key, val string, valStyle, keyStyle, unknownStyle lipgloss.Style, isKnown bool, th *theme.Engine, isSelected bool, width int) string {
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
	if lipgloss.Width(line) > width {
		line = line[:width]
	}

	if isSelected {
		selectedStyle := lipgloss.NewStyle().
			Background(lipgloss.Color(th.GetBg())).
			Foreground(lipgloss.Color(th.GetFg())).
			Bold(true)
		return selectedStyle.Render(line)
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

	var sb strings.Builder
	sb.WriteString(indent)

	if n.Collapsed {
		sb.WriteString("▸ ")
	} else {
		sb.WriteString("▾ ")
	}

	if key != "" {
		sb.WriteString(keyStyle.Render(key))
		sb.WriteString(bracketStyle.Render(": "))
	}

	if n.Collapsed {
		sb.WriteString(bracketStyle.Render(fmt.Sprintf("%s %d items %s", openBracket, count, closeBracket)))
	} else {
		sb.WriteString(bracketStyle.Render(openBracket))
	}

	line := sb.String()
	if lipgloss.Width(line) > width {
		line = line[:width]
	}

	if isSelected {
		selectedStyle := lipgloss.NewStyle().
			Background(lipgloss.Color(th.GetBg())).
			Foreground(lipgloss.Color(th.GetFg())).
			Bold(true)
		return selectedStyle.Render(line)
	}
	return line
}
