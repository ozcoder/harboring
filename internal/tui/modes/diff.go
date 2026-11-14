package modes

import (
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/harboring/harboring/internal/diff"
	"github.com/harboring/harboring/internal/har"
	"github.com/harboring/harboring/internal/theme"
)

type DiffModel struct {
	left      *har.Node
	right     *har.Node
	leftName  string
	rightName string
	theme     *theme.Engine
	width     int
	height    int
	scrollY   int
	scrollX   int
	cursor    int
	options   diff.Options

	leftNodes  []*har.Node
	rightNodes []*har.Node
	diffs      map[string]diff.NodeDiff
	// lineKinds  map[int]diff.Variety // line index -> diff kind for coloring

	leftContentWidth  int // max visual width of all left lines
	rightContentWidth int // max visual width of all right lines

	// lineDiffs []diff.NodeDiff // per-line diff info for partial string highlighting

	varianceInputActive bool
	varianceInput       string

	lastKeyTime time.Time
}

func NewDiffModel(left, right *har.Node, leftName, rightName string, th *theme.Engine) DiffModel {
	d := DiffModel{
		left:      left,
		right:     right,
		leftName:  leftName,
		rightName: rightName,
		theme:     th,
		width:     80,
		height:    24,
		options:   diff.Options{TimeVariancePercent: 10},
	}
	// Start with both trees fully expanded
	left.ExpandAll()
	right.ExpandAll()
	d.syncCollapse(left, right)
	d.refreshNodes()
	return d
}

func (m *DiffModel) SetViewportSize(w, h int) {
	m.width = w
	m.height = h
}

func (m *DiffModel) refreshNodes() {
	m.leftNodes = nil
	m.rightNodes = nil
	collectVisibleDiff(m.left, &m.leftNodes)
	collectVisibleDiff(m.right, &m.rightNodes)

	m.diffs = make(map[string]diff.NodeDiff)
	for _, d := range diff.CompareTrees(m.left, m.right, m.options) {
		m.diffs[d.Path] = d
	}
}

func collectVisibleDiff(n *har.Node, result *[]*har.Node) {
	if n == nil {
		return
	}
	*result = append(*result, n)
	if n.Leaf() || n.Collapsed {
		return
	}
	for _, child := range n.Children {
		collectVisibleDiff(child, result)
	}
}

func (m DiffModel) Options() diff.Options {
	return m.options
}

func (m DiffModel) IsVarianceInputActive() bool {
	return m.varianceInputActive
}

func (m DiffModel) VarianceInput() string {
	return m.varianceInput
}

func (m DiffModel) Update(msg tea.Msg) (DiffModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		return m.handleKey(msg)
	default:
		return m, nil
	}
}

func (m DiffModel) handleKey(msg tea.KeyPressMsg) (DiffModel, tea.Cmd) {
	key := msg.String()

	// Variance input mode: capture numeric input for custom percentage
	if m.varianceInputActive {
		switch key {
		case "esc", "v":
			m.varianceInputActive = false
			m.varianceInput = ""
			return m, nil
		case "enter":
			m.varianceInputActive = false
			if m.varianceInput != "" {
				var pct float64
				if _, err := fmt.Sscanf(m.varianceInput, "%f", &pct); err == nil && pct >= 0 && pct <= 100 {
					m.options.TimeVariancePercent = pct
					m.refreshNodes()
				}
			}
			m.varianceInput = ""
			return m, nil
		case "backspace":
			if len(m.varianceInput) > 0 {
				m.varianceInput = m.varianceInput[:len(m.varianceInput)-1]
			}
			return m, nil
		default:
			// Accept digits and one decimal point
			for _, r := range key {
				if r >= '0' && r <= '9' {
					m.varianceInput += string(r)
					return m, nil
				}
				if r == '.' && !strings.Contains(m.varianceInput, ".") {
					m.varianceInput += string(r)
					return m, nil
				}
			}
			return m, nil
		}
	}

	// Auto-repeat debounce for navigation keys
	debounceKeys := map[string]bool{
		"up": true, "k": true, "down": true, "j": true,
		"pgup": true, "b": true, "pgdown": true, "space": true, "f": true,
	}
	if debounceKeys[key] {
		now := time.Now()
		if !m.lastKeyTime.IsZero() && now.Sub(m.lastKeyTime) < 150*time.Millisecond {
			return m, nil
		}
		m.lastKeyTime = now
	}

	switch key {
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		maxCursor := len(m.leftNodes)
		if len(m.rightNodes) > maxCursor {
			maxCursor = len(m.rightNodes)
		}
		if m.cursor < maxCursor-1 {
			m.cursor++
		}
	case "pgdown", "space", "f":
		m.cursor += m.dataHeight()
		maxCursor := len(m.leftNodes)
		if len(m.rightNodes) > maxCursor {
			maxCursor = len(m.rightNodes)
		}
		if m.cursor >= maxCursor {
			m.cursor = maxCursor - 1
		}
	case "pgup", "b":
		m.cursor -= m.dataHeight()
		if m.cursor < 0 {
			m.cursor = 0
		}
	case "left":
		m.scrollX -= 5
		if m.scrollX < 0 {
			m.scrollX = 0
		}
	case "right":
		m.scrollX += 5
	case "h":
		if m.cursor < len(m.leftNodes) {
			node := m.leftNodes[m.cursor]
			if (node.Kind == har.Object || node.Kind == har.Array) && !node.Collapsed {
				node.ToggleCollapse()
				m.syncCollapse(m.left, m.right)
				m.refreshNodes()
				if m.cursor >= len(m.leftNodes) {
					m.cursor = len(m.leftNodes) - 1
				}
			}
		}
	case "l":
		if m.cursor < len(m.leftNodes) {
			node := m.leftNodes[m.cursor]
			if (node.Kind == har.Object || node.Kind == har.Array) && node.Collapsed {
				node.ToggleCollapse()
				m.syncCollapse(m.left, m.right)
				m.refreshNodes()
			}
		}
	case "enter":
		if m.cursor < len(m.leftNodes) {
			node := m.leftNodes[m.cursor]
			if node.Kind == har.Object || node.Kind == har.Array {
				node.ToggleCollapse()
				m.syncCollapse(m.left, m.right)
				m.refreshNodes()
				if m.cursor >= len(m.leftNodes) {
					m.cursor = len(m.leftNodes) - 1
				}
			}
		}
	case "E":
		m.left.ExpandAll()
		m.syncCollapse(m.left, m.right)
		m.refreshNodes()
		m.cursor = 0
	case "c":
		m.left.CollapseAll()
		m.syncCollapse(m.left, m.right)
		m.refreshNodes()
		m.cursor = 0
	case "v":
		m.varianceInputActive = true
		m.varianceInput = ""
	}

	m.updateScrollY()
	return m, nil
}

// syncCollapse propagates the collapse state from left to right by path.
func (m *DiffModel) syncCollapse(left, right *har.Node) {
	if left == nil || right == nil {
		return
	}
	left.Collapsed = right.Collapsed
	if left.Kind != right.Kind {
		return
	}
	if left.Kind == har.Object || left.Kind == har.Array {
		rightMap := make(map[string]*har.Node)
		for _, c := range right.Children {
			rightMap[diff.CanonicalName(c.Key)] = c
		}
		for _, lc := range left.Children {
			if rc, ok := rightMap[diff.CanonicalName(lc.Key)]; ok {
				m.syncCollapse(lc, rc)
			}
		}
	}
}

func (m *DiffModel) updateScrollY() {
	totalLines := len(m.leftNodes)
	if len(m.rightNodes) > totalLines {
		totalLines = len(m.rightNodes)
	}
	height := m.dataHeight()

	if m.cursor < m.scrollY {
		m.scrollY = m.cursor
	} else if m.cursor >= m.scrollY+height {
		m.scrollY = m.cursor - height + 1
	}

	maxScroll := totalLines - height
	if maxScroll < 0 {
		maxScroll = 0
	}
	if m.scrollY > maxScroll {
		m.scrollY = maxScroll
	}
	if m.scrollY < 0 {
		m.scrollY = 0
	}
}

func (m DiffModel) dataHeight() int {
	h := m.height - 4 // header (2 lines) + hscrollbar (1 line) + footer (1 line)
	if h < 1 {
		h = 1
	}
	return h
}

func (m DiffModel) View() string {
	if m.left == nil || m.right == nil {
		return "No HAR files loaded"
	}

	dataH := m.dataHeight()
	paneW := (m.width - 1) / 2 // width available for content in each pane (total - connector) / 2

	var leftLines, rightLines []string
	var leftKinds, rightKinds []diff.Variety
	var lineDiffList []diff.NodeDiff

	leftIdx := 0
	rightIdx := 0
	m.renderSideBySide(m.left, m.right, "", &leftLines, &rightLines, &leftKinds, &rightKinds, &lineDiffList, &leftIdx, &rightIdx, paneW)

	// Pad to same length
	for len(leftLines) < len(rightLines) {
		leftLines = append(leftLines, "")
		leftKinds = append(leftKinds, diff.Identical)
		lineDiffList = append(lineDiffList, diff.NodeDiff{Kind: diff.Identical})
	}
	for len(rightLines) < len(leftLines) {
		rightLines = append(rightLines, "")
		rightKinds = append(rightKinds, diff.Identical)
		lineDiffList = append(lineDiffList, diff.NodeDiff{Kind: diff.Identical})
	}

	// Compute max content widths for horizontal scroll (ANSI-safe)
	m.leftContentWidth = 0
	m.rightContentWidth = 0
	for _, l := range leftLines {
		if w := lipgloss.Width(l); w > m.leftContentWidth {
			m.leftContentWidth = w
		}
	}
	for _, r := range rightLines {
		if w := lipgloss.Width(r); w > m.rightContentWidth {
			m.rightContentWidth = w
		}
	}
	contentWidth := m.leftContentWidth
	if m.rightContentWidth > contentWidth {
		contentWidth = m.rightContentWidth
	}

	// Clamp horizontal scroll
	maxScrollX := contentWidth - paneW
	if maxScrollX < 0 {
		maxScrollX = 0
	}
	if m.scrollX > maxScrollX {
		m.scrollX = maxScrollX
	}
	if m.scrollX < 0 {
		m.scrollX = 0
	}

	// Slice to visible area
	endLine := m.scrollY + dataH
	if endLine > len(leftLines) {
		endLine = len(leftLines)
	}
	visLeft := leftLines[m.scrollY:endLine]
	visRight := rightLines[m.scrollY:endLine]
	visLeftKinds := leftKinds[m.scrollY:endLine]
	visRightKinds := rightKinds[m.scrollY:endLine]

	// Build vertical scroll chars
	scrollChars := buildScrollChars(len(leftLines), m.scrollY, len(visLeft))

	var result strings.Builder

	// Header
	leftTitle := fmt.Sprintf(" %s ", m.leftName)
	rightTitle := fmt.Sprintf(" %s ", m.rightName)
	leftTitleStyle := lipgloss.NewStyle().Width(paneW).Align(lipgloss.Center).Bold(true)
	rightTitleStyle := lipgloss.NewStyle().Width(paneW).Align(lipgloss.Center).Bold(true)
	result.WriteString(leftTitleStyle.Render(m.theme.GetStyle("title").Render(leftTitle)))
	result.WriteString(m.theme.GetStyle("title").Render("│"))
	result.WriteString(rightTitleStyle.Render(m.theme.GetStyle("title").Render(rightTitle)))
	result.WriteString("\n")

	// Separator
	sep := strings.Repeat("─", m.width)
	result.WriteString(m.theme.GetStyle("status_line").Render(sep))
	result.WriteString("\n")

	// Horizontal scrollbar with arrows (only shown when content overflows pane width)
	hscrollTrack, hscrollThumb := hScrollbarStyles(m.theme)
	if contentWidth > paneW {
		result.WriteString(renderHScrollbarWithArrows(m.width, contentWidth, m.scrollX, hscrollTrack, hscrollThumb))
		result.WriteString("\n")
	} else {
		result.WriteString(strings.Repeat(" ", m.width) + "\n")
	}

	// Content
	for i := range visLeft {
		sc := " "
		if i < len(scrollChars) {
			sc = scrollChars[i]
		}

		leftLine := visLeft[i]
		rightLine := visRight[i]

		// Apply horizontal scroll — truncate from the left (ANSI-safe)
		if m.scrollX > 0 {
			leftLine = truncateRenderedLeft(leftLine, m.scrollX)
			rightLine = truncateRenderedLeft(rightLine, m.scrollX)
		}

		// Render with diff style, then truncate+pad to pane width
		leftRendered := fitRendered(m.diffStyle(visLeftKinds[i]).Render(leftLine), paneW)
		rightRendered := fitRendered(m.diffStyle(visRightKinds[i]).Render(rightLine), paneW)

		// Visual connector
		conn := "│"
		switch visLeftKinds[i] {
		case diff.Modified:
			conn = "⇔"
		case diff.Added:
			conn = "+"
		case diff.Removed:
			conn = "-"
		}

		result.WriteString(sc)
		result.WriteString(leftRendered)
		result.WriteString(conn)
		result.WriteString(rightRendered)
		result.WriteString("\n")
	}

	// Pad to dataH to keep footer sticky at bottom
	for len(visLeft) < dataH {
		result.WriteString(strings.Repeat(" ", m.width) + "\n")
		visLeft = append(visLeft, "")
	}

	// Footer with statistics
	var allDiffs []diff.NodeDiff
	for _, d := range m.diffs {
		allDiffs = append(allDiffs, d)
	}
	added, removed, modified := diff.CountByKind(allDiffs)
	total := len(m.diffs)
	cursorLine := m.scrollY + (m.cursor - m.scrollY)
	if cursorLine < 0 {
		cursorLine = 0
	}
	if cursorLine >= len(leftLines) {
		cursorLine = len(leftLines) - 1
	}

	footerStyle := m.theme.GetStyle("status_line")
	footer := fmt.Sprintf(" %d diffs | +%d -%d ~%d | %d/%d lines | line %d | variance %.0f%%",
		total, added, removed, modified, len(leftLines), len(rightLines), cursorLine+1,
		m.options.TimeVariancePercent)
	result.WriteString(fitLine(footerStyle.Render(footer), m.width))

	return result.String()
}

func (m DiffModel) renderSideBySide(left, right *har.Node, path string, leftLines, rightLines *[]string, leftKinds, rightKinds *[]diff.Variety, lineDiffs *[]diff.NodeDiff, leftIdx, rightIdx *int, width int) {
	if left == nil && right == nil {
		return
	}
	if left == nil {
		m.renderRightOnly(right, path, rightLines, rightKinds, lineDiffs, rightIdx, width)
		return
	}
	if right == nil {
		m.renderLeftOnly(left, path, leftLines, leftKinds, lineDiffs, leftIdx, width)
		return
	}

	// Both present — render side by side
	d := m.diffs[path]
	kind := d.Kind

	leftLine := m.renderNode(left, path, *leftIdx == m.cursor, width, &d, true)
	rightLine := m.renderNode(right, path, *rightIdx == m.cursor, width, &d, false)

	*leftLines = append(*leftLines, leftLine)
	*rightLines = append(*rightLines, rightLine)
	*leftKinds = append(*leftKinds, kind)
	*rightKinds = append(*rightKinds, kind)
	*lineDiffs = append(*lineDiffs, d)
	*leftIdx++
	*rightIdx++

	if left.Leaf() || left.Collapsed {
		return
	}

	// Recurse into children
	rightMap := make(map[string]*har.Node)
	for _, c := range right.Children {
		rightMap[diff.CanonicalName(c.Key)] = c
	}

	for _, lc := range left.Children {
		childPath := joinPath(path, lc.Key)
		if rc, ok := rightMap[diff.CanonicalName(lc.Key)]; ok {
			m.renderSideBySide(lc, rc, childPath, leftLines, rightLines, leftKinds, rightKinds, lineDiffs, leftIdx, rightIdx, width)
		} else {
			m.renderLeftOnly(lc, childPath, leftLines, leftKinds, lineDiffs, leftIdx, width)
			// Add empty line to right
			*rightLines = append(*rightLines, strings.Repeat(" ", width))
			*rightKinds = append(*rightKinds, diff.Removed)
			*lineDiffs = append(*lineDiffs, diff.NodeDiff{Kind: diff.Removed})
		}
	}

	// Check for right-only children
	leftMap := make(map[string]*har.Node)
	for _, lc := range left.Children {
		leftMap[diff.CanonicalName(lc.Key)] = lc
	}
	for _, rc := range right.Children {
		if _, ok := leftMap[diff.CanonicalName(rc.Key)]; !ok {
			childPath := joinPath(path, rc.Key)
			m.renderRightOnly(rc, childPath, rightLines, rightKinds, lineDiffs, rightIdx, width)
			// Add empty line to left
			*leftLines = append(*leftLines, strings.Repeat(" ", width))
			*leftKinds = append(*leftKinds, diff.Added)
			*lineDiffs = append(*lineDiffs, diff.NodeDiff{Kind: diff.Added})
		}
	}

	// Closing bracket
	closeBracket := "}"
	if left.Kind == har.Array {
		closeBracket = "]"
	}
	indent := strings.Repeat("  ", depthOf(path))
	*leftLines = append(*leftLines, indent+closeBracket)
	*rightLines = append(*rightLines, indent+closeBracket)
	*leftKinds = append(*leftKinds, diff.Identical)
	*rightKinds = append(*rightKinds, diff.Identical)
	*lineDiffs = append(*lineDiffs, diff.NodeDiff{Kind: diff.Identical})
}

func (m DiffModel) renderLeftOnly(n *har.Node, path string, lines *[]string, kinds *[]diff.Variety, lineDiffs *[]diff.NodeDiff, idx *int, width int) {
	d := diff.NodeDiff{Kind: diff.Removed, Left: n}
	line := m.renderNode(n, path, *idx == m.cursor, width, nil, false)
	*lines = append(*lines, line)
	*kinds = append(*kinds, diff.Removed)
	*lineDiffs = append(*lineDiffs, d)
	*idx++

	if n.Leaf() || n.Collapsed {
		return
	}
	for _, child := range n.Children {
		m.renderLeftOnly(child, joinPath(path, child.Key), lines, kinds, lineDiffs, idx, width)
	}
	closeBracket := "}"
	if n.Kind == har.Array {
		closeBracket = "]"
	}
	indent := strings.Repeat("  ", depthOf(path))
	*lines = append(*lines, indent+closeBracket)
	*kinds = append(*kinds, diff.Removed)
	*lineDiffs = append(*lineDiffs, diff.NodeDiff{Kind: diff.Removed})
}

func (m DiffModel) renderRightOnly(n *har.Node, path string, lines *[]string, kinds *[]diff.Variety, lineDiffs *[]diff.NodeDiff, idx *int, width int) {
	d := diff.NodeDiff{Kind: diff.Added, Right: n}
	line := m.renderNode(n, path, *idx == m.cursor, width, nil, false)
	*lines = append(*lines, line)
	*kinds = append(*kinds, diff.Added)
	*lineDiffs = append(*lineDiffs, d)
	*idx++

	if n.Leaf() || n.Collapsed {
		return
	}
	for _, child := range n.Children {
		m.renderRightOnly(child, joinPath(path, child.Key), lines, kinds, lineDiffs, idx, width)
	}
	closeBracket := "}"
	if n.Kind == har.Array {
		closeBracket = "]"
	}
	indent := strings.Repeat("  ", depthOf(path))
	*lines = append(*lines, indent+closeBracket)
	*kinds = append(*kinds, diff.Added)
	*lineDiffs = append(*lineDiffs, diff.NodeDiff{Kind: diff.Added})
}

func (m DiffModel) renderNode(n *har.Node, path string, isSelected bool, width int, d *diff.NodeDiff, isLeft bool) string {
	if n == nil {
		return ""
	}

	indent := strings.Repeat("  ", depthOf(path))
	keyStyle := m.theme.GetStyle("json_key")
	stringStyle := m.theme.GetStyle("json_string")
	numberStyle := m.theme.GetStyle("json_number")
	boolStyle := m.theme.GetStyle("json_bool")
	nullStyle := m.theme.GetStyle("json_null")
	bracketStyle := m.theme.GetStyle("json_bracket")
	changedStyle := m.theme.GetStyle("diff_changed")

	var sb strings.Builder
	sb.WriteString(indent)

	if n.Key != "" {
		sb.WriteString(keyStyle.Render(n.Key + ": "))
	}

	switch n.Kind {
	case har.Object:
		arrow := "▸ "
		if !n.Collapsed {
			arrow = "▾ "
		}
		sb.WriteString(bracketStyle.Render(arrow + "{"))
		if n.Collapsed {
			sb.WriteString(bracketStyle.Render(fmt.Sprintf(" %d items }", len(n.Children))))
		}
	case har.Array:
		arrow := "▸ "
		if !n.Collapsed {
			arrow = "▾ "
		}
		sb.WriteString(bracketStyle.Render(arrow + "["))
		if n.Collapsed {
			sb.WriteString(bracketStyle.Render(fmt.Sprintf(" %d items ]", len(n.Children))))
		}
	case har.String:
		strVal := fmt.Sprintf("%q", n.Value)
		// Partial string diff highlighting: show exactly which part changed
		if d != nil && d.Kind == diff.Modified && d.Left != nil && d.Right != nil &&
			d.Left.Kind == har.String && d.Right.Kind == har.String {
			leftVal := d.LeftVal
			rightVal := d.RightVal
			myVal := leftVal
			if !isLeft {
				myVal = rightVal
			}
			prefix, suffix := commonPrefixSuffix(leftVal, rightVal)
			if len(prefix)+len(suffix) > 0 && len(prefix)+len(suffix) < len(myVal) {
				diffPart := myVal[len(prefix) : len(myVal)-len(suffix)]
				// Render opening quote + prefix, then highlighted diff, then suffix + closing quote
				sb.WriteString(stringStyle.Render(`"` + prefix))
				sb.WriteString(changedStyle.Render(diffPart))
				sb.WriteString(stringStyle.Render(suffix + `"`))
			} else {
				sb.WriteString(stringStyle.Render(strVal))
			}
		} else {
			sb.WriteString(stringStyle.Render(strVal))
		}
	case har.Number:
		numText := n.NumberText()
		// Partial number diff highlighting
		if d != nil && d.Kind == diff.Modified && d.Left != nil && d.Right != nil &&
			d.Left.Kind == har.Number && d.Right.Kind == har.Number {
			leftNum := d.LeftVal
			rightNum := d.RightVal
			myNum := leftNum
			if !isLeft {
				myNum = rightNum
			}
			prefix, suffix := commonPrefixSuffix(leftNum, rightNum)
			if len(prefix)+len(suffix) > 0 && len(prefix)+len(suffix) < len(myNum) {
				diffPart := myNum[len(prefix) : len(myNum)-len(suffix)]
				sb.WriteString(numberStyle.Render(prefix))
				sb.WriteString(changedStyle.Render(diffPart))
				sb.WriteString(numberStyle.Render(suffix))
			} else {
				sb.WriteString(numberStyle.Render(numText))
			}
		} else {
			sb.WriteString(numberStyle.Render(numText))
		}
	case har.Bool:
		if v, ok := n.Value.(bool); ok && v {
			sb.WriteString(boolStyle.Render("true"))
		} else {
			sb.WriteString(boolStyle.Render("false"))
		}
	case har.Null:
		sb.WriteString(nullStyle.Render("null"))
	}

	line := sb.String()
	if isSelected {
		selectedStyle := lipgloss.NewStyle().
			Background(lipgloss.Color(m.theme.GetSelectionBg())).
			Foreground(lipgloss.Color(m.theme.GetFg())).
			Bold(true)
		return selectedStyle.Render(padRendered(line, width))
	}
	return line
}

func (m DiffModel) diffStyle(kind diff.Variety) lipgloss.Style {
	switch kind {
	case diff.Added:
		return m.theme.GetStyle("diff_added")
	case diff.Removed:
		return m.theme.GetStyle("diff_removed")
	case diff.Modified:
		return m.theme.GetStyle("diff_modified")
	default:
		return lipgloss.NewStyle()
	}
}

// commonPrefixSuffix returns the longest common prefix and suffix of two strings.
func commonPrefixSuffix(a, b string) (prefix, suffix string) {
	i := 0
	for i < len(a) && i < len(b) && a[i] == b[i] {
		i++
	}
	prefix = a[:i]
	j := 0
	for j < len(a)-i && j < len(b)-i && a[len(a)-1-j] == b[len(b)-1-j] {
		j++
	}
	suffix = a[len(a)-j:]
	return
}

func depthOf(path string) int {
	if path == "" {
		return 0
	}
	return strings.Count(path, ".") + 1
}

func joinPath(parent, key string) string {
	if parent == "" {
		return key
	}
	if key == "" {
		return parent
	}
	return parent + "." + key
}

// padRendered pads an already-rendered (possibly styled) string to the target
// visual width using lipgloss.Width for ANSI-safe measurement.
func padRendered(s string, width int) string {
	w := lipgloss.Width(s)
	if w >= width {
		return s
	}
	return s + strings.Repeat(" ", width-w)
}

// fitRendered truncates or pads a rendered string to exactly width visual chars.
func fitRendered(s string, width int) string {
	w := lipgloss.Width(s)
	if w == width {
		return s
	}
	if w > width {
		// Truncate from the right, preserving ANSI integrity
		return truncateRenderedRight(s, width)
	}
	return s + strings.Repeat(" ", width-w)
}

// truncateRenderedRight truncates a rendered string to max visual width from
// the left. Properly skips ANSI escape sequences.
func truncateRenderedRight(s string, max int) string {
	if max <= 0 {
		return ""
	}
	var b strings.Builder
	w := 0
	i := 0
	for i < len(s) && w < max {
		// Skip ANSI escape sequences (ESC [ ... final byte)
		if s[i] == '\x1b' && i+1 < len(s) && s[i+1] == '[' {
			end := i + 2
			for end < len(s) && ((s[end] >= 0x30 && s[end] <= 0x3f) || (s[end] >= 0x20 && s[end] <= 0x2f)) {
				end++
			}
			if end < len(s) {
				end++ // skip the final byte (0x40-0x7e)
			}
			// Write the entire escape sequence without counting visual width
			b.WriteString(s[i:end])
			i = end
			continue
		}
		// Normal rune — measure its visual width
		r, size := utf8.DecodeRuneInString(s[i:])
		rw := lipgloss.Width(string(r))
		if w+rw > max {
			break
		}
		b.WriteRune(r)
		w += rw
		i += size
	}
	return b.String()
}

// truncateRenderedLeft removes visual characters from the left of a rendered
// string up to maxSkip visual columns. Properly skips ANSI escape sequences.
func truncateRenderedLeft(s string, maxSkip int) string {
	if maxSkip <= 0 {
		return s
	}
	skipped := 0
	i := 0
	for i < len(s) && skipped < maxSkip {
		// Skip ANSI escape sequences (ESC [ ... final byte)
		if s[i] == '\x1b' && i+1 < len(s) && s[i+1] == '[' {
			end := i + 2
			for end < len(s) && ((s[end] >= 0x30 && s[end] <= 0x3f) || (s[end] >= 0x20 && s[end] <= 0x2f)) {
				end++
			}
			if end < len(s) {
				end++ // skip the final byte (0x40-0x7e)
			}
			i = end
			continue
		}
		// Normal rune — measure its visual width
		r, size := utf8.DecodeRuneInString(s[i:])
		rw := lipgloss.Width(string(r))
		if skipped+rw > maxSkip {
			break
		}
		skipped += rw
		i += size
	}
	return s[i:]
}

// fitLine truncates or pads a rendered string to exactly width visual chars.
func fitLine(s string, width int) string {
	w := lipgloss.Width(s)
	if w == width {
		return s
	}
	if w > width {
		return truncateRenderedRight(s, width)
	}
	return s + strings.Repeat(" ", width-w)
}
