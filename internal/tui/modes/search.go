package modes

import (
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/harboring/harboring/internal/har"
	"github.com/harboring/harboring/internal/theme"
)

type SearchResult struct {
	Path      string
	Node      *har.Node
	LineIndex int
}

type SearchModel struct {
	Input   textinput.Model
	Results []SearchResult
	Cursor  int
	theme   *theme.Engine
	width   int
	height  int
}

func NewSearchModel(th *theme.Engine) SearchModel {
	ti := textinput.New()
	ti.Placeholder = "Search..."
	ti.Focus()
	return SearchModel{
		Input:  ti,
		theme:  th,
		width:  80,
		height: 20,
	}
}

func (m SearchModel) Update(msg tea.Msg) (SearchModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case tea.KeyPressMsg:
		switch msg.String() {
		case "enter":
			if len(m.Results) > 0 {
				m.Cursor++
				if m.Cursor >= len(m.Results) {
					m.Cursor = 0
				}
			}
		case "up", "k":
			if m.Cursor > 0 {
				m.Cursor--
			}
		case "down", "j":
			if m.Cursor < len(m.Results)-1 {
				m.Cursor++
			}
		}
	}

	var cmd tea.Cmd
	m.Input, cmd = m.Input.Update(msg)
	return m, cmd
}

func (m SearchModel) View() string {
	inputLine := m.Input.View()

	resultStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(m.theme.GetFg()))
	selectedStyle := lipgloss.NewStyle().
		Background(lipgloss.Color(m.theme.GetSelectionBg())).
		Foreground(lipgloss.Color(m.theme.GetFg())).
		Bold(true)

	var sb strings.Builder
	sb.WriteString(inputLine)
	sb.WriteString("\n\n")

	if len(m.Results) == 0 && m.Input.Value() != "" {
		sb.WriteString(resultStyle.Render("No results"))
		sb.WriteString("\n")
	} else {
		maxResults := m.height - 4
		if maxResults < 1 {
			maxResults = 1
		}

		start := 0
		if m.Cursor >= maxResults {
			start = m.Cursor - maxResults + 1
		}

		end := start + maxResults
		if end > len(m.Results) {
			end = len(m.Results)
		}

		for i := start; i < end; i++ {
			r := m.Results[i]
			line := r.Path
			if r.Node != nil && r.Node.Kind == har.String {
				val := r.Node.Value.(string)
				if len(val) > 60 {
					val = val[:57] + "..."
				}
				line = line + " = " + val
			}

			if i == m.Cursor {
				sb.WriteString(selectedStyle.Render(line))
			} else {
				sb.WriteString(resultStyle.Render(line))
			}
			sb.WriteString("\n")
		}

		if len(m.Results) > 0 {
			status := lipgloss.NewStyle().Foreground(lipgloss.Color("#666666"))
			sb.WriteString(status.Render(strings.Repeat("─", m.width-2)))
			sb.WriteString("\n")
			sb.WriteString(status.Render(strings.Repeat("─", m.width-2)))
		}
	}

	return sb.String()
}

func (m *SearchModel) Search(root *har.Node, query string) {
	m.Results = nil
	m.Cursor = 0
	if query == "" || root == nil {
		return
	}
	query = strings.ToLower(query)
	searchWalk(root, "", query, &m.Results, 0)
}

func searchWalk(n *har.Node, path string, query string, results *[]SearchResult, lineIndex int) int {
	currentLine := lineIndex

	nodePath := path
	if n.Key != "" {
		if path != "" {
			nodePath = path + "." + n.Key
		} else {
			nodePath = n.Key
		}
	}

	if n.Kind == har.String {
		if n.Value != nil {
			val := strings.ToLower(n.Value.(string))
			if strings.Contains(val, query) {
				*results = append(*results, SearchResult{
					Path:      nodePath,
					Node:      n,
					LineIndex: currentLine,
				})
			}
		}
		currentLine++
	} else if n.Kind == har.Number || n.Kind == har.Bool || n.Kind == har.Null {
		currentLine++
	} else {
		if n.Collapsed {
			currentLine++
		} else {
			currentLine++
			for _, child := range n.Children {
				currentLine = searchWalk(child, nodePath, query, results, currentLine)
			}
			if n.Kind == har.Object || n.Kind == har.Array {
				currentLine++
			}
		}
	}

	return currentLine
}

func (m SearchModel) Selected() *SearchResult {
	if len(m.Results) == 0 || m.Cursor >= len(m.Results) {
		return nil
	}
	return &m.Results[m.Cursor]
}
