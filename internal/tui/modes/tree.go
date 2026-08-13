package modes

import (
	"strings"

	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"github.com/harboring/harboring/internal/har"
	"github.com/harboring/harboring/internal/theme"
)

type RenderLineFunc func(n *har.Node, th *theme.Engine, isSelected bool, width int, depth int) string

type TreeModel struct {
	root     *har.Node
	Cursor   int
	scrollY  int
	viewport viewport.Model
	theme    *theme.Engine
	render   RenderLineFunc
	width    int
	height   int
	allNodes []*har.Node
}

func NewTreeModel(root *har.Node, th *theme.Engine, render RenderLineFunc) TreeModel {
	vp := viewport.New(viewport.WithWidth(80), viewport.WithHeight(20))
	tm := TreeModel{
		root:   root,
		Cursor: 0,
		viewport: vp,
		theme:  th,
		render: render,
		width:  80,
		height: 20,
	}
	tm.refreshNodes()
	return tm
}

func (m *TreeModel) SetCursor(c int) {
	m.Cursor = c
}

func (m *TreeModel) SetViewportSize(w, h int) {
	m.width = w
	m.height = h
	m.viewport.SetWidth(w)
	m.viewport.SetHeight(h)
}

func (m *TreeModel) refreshNodes() {
	m.allNodes = nil
	collectVisible(m.root, &m.allNodes)
}

func collectVisible(n *har.Node, result *[]*har.Node) {
	*result = append(*result, n)
	if n.Leaf() || n.Collapsed {
		return
	}
	for _, child := range n.Children {
		collectVisible(child, result)
	}
}

func (m TreeModel) Update(msg tea.Msg) (TreeModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.viewport.SetWidth(msg.Width)
		m.viewport.SetHeight(msg.Height - 2)
	case tea.KeyPressMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

func (m TreeModel) handleKey(msg tea.KeyPressMsg) (TreeModel, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.Cursor > 0 {
			m.Cursor--
		}
	case "down", "j":
		if m.Cursor < len(m.allNodes)-1 {
			m.Cursor++
		}
	case "left", "h":
		if len(m.allNodes) > 0 {
			node := m.allNodes[m.Cursor]
			if node.Kind == har.Object || node.Kind == har.Array {
				if !node.Collapsed {
					node.ToggleCollapse()
					m.refreshNodes()
					if m.Cursor >= len(m.allNodes) {
						m.Cursor = len(m.allNodes) - 1
					}
				}
			}
		}
	case "right", "l", "enter":
		if len(m.allNodes) > 0 {
			node := m.allNodes[m.Cursor]
			if node.Kind == har.Object || node.Kind == har.Array {
				if node.Collapsed {
					node.ToggleCollapse()
					m.refreshNodes()
				}
			}
		}
	case "E":
		m.root.ExpandAll()
		m.refreshNodes()
		m.Cursor = 0
	case "c":
		m.root.CollapseAll()
		m.refreshNodes()
		m.Cursor = 0
	}

	m.updateScrollY()
	return m, nil
}

func (m *TreeModel) updateScrollY() {
	if len(m.allNodes) == 0 {
		m.scrollY = 0
		return
	}

	totalLines := m.lineCount()
	cursorLine := m.lineNumberOfCursor()
	height := m.viewport.Height()

	yOff := m.scrollY
	if cursorLine < yOff {
		yOff = cursorLine
	} else if cursorLine >= yOff+height {
		yOff = cursorLine - height + 1
	}
	maxYOff := totalLines - height
	if maxYOff < 0 {
		maxYOff = 0
	}
	if yOff > maxYOff {
		yOff = maxYOff
	}
	if yOff < 0 {
		yOff = 0
	}
	m.scrollY = yOff
}

func (m TreeModel) View() string {
	if m.root == nil {
		return ""
	}

	cursorIdx := 0
	var allLines []string
	m.renderRecursive(m.root, &allLines, 0, &cursorIdx)

	totalLines := len(allLines)
	height := m.viewport.Height()
	yOff := m.scrollY
	maxYOff := totalLines - height
	if maxYOff < 0 {
		maxYOff = 0
	}
	if yOff > maxYOff {
		yOff = maxYOff
		m.scrollY = yOff
	}

	endLine := yOff + height
	if endLine > totalLines {
		endLine = totalLines
	}
	visibleLines := allLines[yOff:endLine]
	scrollChars := buildScrollChars(totalLines, yOff, len(visibleLines))

	var result strings.Builder
	for i, line := range visibleLines {
		sc := " "
		if i < len(scrollChars) {
			sc = scrollChars[i]
		}
		result.WriteString(sc)
		result.WriteString(line)
		result.WriteString("\n")
	}

	return result.String()
}

func (m TreeModel) renderRecursive(n *har.Node, lines *[]string, depth int, cursorIdx *int) {
	idx := *cursorIdx
	isSelected := idx == m.Cursor
	line := m.render(n, m.theme, isSelected, m.width-1, depth)
	*lines = append(*lines, line)
	*cursorIdx++

	if n.Leaf() || n.Collapsed {
		return
	}

	for _, child := range n.Children {
		m.renderRecursive(child, lines, depth+1, cursorIdx)
	}

	closeBracket := "}"
	if n.Kind == har.Array {
		closeBracket = "]"
	}
	indent := strings.Repeat("  ", depth)
	*lines = append(*lines, indent+closeBracket)
}

func (m TreeModel) lineCount() int {
	count := 0
	m.lineCountWalkTotal(m.root, &count)
	return count
}

func (m TreeModel) lineCountWalkTotal(n *har.Node, count *int) {
	*count++
	if n.Leaf() || n.Collapsed {
		return
	}
	for _, child := range n.Children {
		m.lineCountWalkTotal(child, count)
	}
	*count++
}

func (m TreeModel) lineNumberOfCursor() int {
	count := 0
	idx := 0
	return m.lineCountWalk(m.root, &count, &idx)
}

func (m TreeModel) lineCountWalk(n *har.Node, count *int, nodeIdx *int) int {
	idx := *nodeIdx
	if idx == m.Cursor {
		return *count
	}
	*count++
	*nodeIdx++

	if n.Leaf() || n.Collapsed {
		return -1
	}

	for _, child := range n.Children {
		result := m.lineCountWalk(child, count, nodeIdx)
		if result >= 0 {
			return result
		}
	}

	*count++
	return -1
}
