package tui

import (
	"strings"

	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/harboring/harboring/internal/theme"
)

type HelpModel struct {
	viewport viewport.Model
	theme    *theme.Engine
	width    int
	height   int
}

func NewHelpModel(th *theme.Engine) HelpModel {
	vp := viewport.New(viewport.WithWidth(60), viewport.WithHeight(20))
	return HelpModel{
		viewport: vp,
		theme:    th,
		width:    60,
		height:   20,
	}
}

func (m HelpModel) Init() tea.Cmd {
	return nil
}

func (m HelpModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case tea.KeyPressMsg:
		var cmd tea.Cmd
		m.viewport, cmd = m.viewport.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m HelpModel) renderWithScrollbar(rawContent string) string {
	content := m.viewport.View()
	lines := strings.Split(content, "\n")
	totalLines := len(strings.Split(rawContent, "\n"))
	visibleLines := m.viewport.Height()

	if totalLines <= visibleLines {
		return content
	}

	scrollbar := m.buildHelpScrollbar(totalLines, visibleLines, m.viewport.YOffset())
	scrollStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(m.theme.GetGray()))

	result := make([]string, len(lines))
	for i, line := range lines {
		sc := " "
		if i < len(scrollbar) {
			sc = scrollbar[i]
		}
		result[i] = scrollStyle.Render(sc) + " " + line
	}

	return strings.Join(result, "\n")
}

func (m HelpModel) buildHelpScrollbar(total, visible, offset int) []string {
	result := make([]string, visible)
	for i := range result {
		result[i] = " "
	}
	if total <= visible {
		return result
	}

	scrollbarHeight := visible
	maxScroll := total - visible
	barPos := 1
	if maxScroll > 0 {
		barPos = 1 + (offset*(scrollbarHeight-2))/maxScroll
	}
	if barPos > scrollbarHeight-2 {
		barPos = scrollbarHeight - 2
	}

	for i := 0; i < scrollbarHeight; i++ {
		switch i {
		case 0:
			result[i] = "▲"
		case scrollbarHeight-1:
			result[i] = "▼"
		case barPos:
			result[i] = "█"
		default:
			result[i] = "│"
		}
	}
	return result
}

func (m HelpModel) View() tea.View {
	titleStyle := m.theme.GetStyle("title")
	sectionStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(m.theme.GetFg()))
	keyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(m.theme.GetFg())).Bold(true)

	var sb strings.Builder

	sb.WriteString(titleStyle.Render("Keyboard Shortcuts"))
	sb.WriteString("\n\n")

	sb.WriteString(sectionStyle.Render("Global"))
	sb.WriteString("\n")
	sb.WriteString(renderHelpLine(keyStyle, "1/2/3/4/5/6/7", "Switch to Tree / Requests / Timings / Summary / Cache / Security / Timeline mode"))
	sb.WriteString(renderHelpLine(keyStyle, "/", "Open search"))
	sb.WriteString(renderHelpLine(keyStyle, "?", "Show this help"))
	sb.WriteString(renderHelpLine(keyStyle, "o", "Open options"))
	sb.WriteString(renderHelpLine(keyStyle, "t", "Toggle dark/light theme"))
	sb.WriteString(renderHelpLine(keyStyle, "s", "Scrub content sections"))
	sb.WriteString(renderHelpLine(keyStyle, "Ctrl+S", "Save file"))
	sb.WriteString(renderHelpLine(keyStyle, "q / Ctrl+C", "Quit"))
	sb.WriteString("\n")

	sb.WriteString(sectionStyle.Render("Tree Mode"))
	sb.WriteString("\n")
	sb.WriteString(renderHelpLine(keyStyle, "j/k or Up/Down", "Move down/up"))
	sb.WriteString(renderHelpLine(keyStyle, "h/l or Left/Right", "Collapse/expand node"))
	sb.WriteString(renderHelpLine(keyStyle, "Enter", "Toggle expand/collapse"))
	sb.WriteString(renderHelpLine(keyStyle, "E", "Expand all"))
	sb.WriteString(renderHelpLine(keyStyle, "c", "Collapse all"))
	sb.WriteString("\n")

	sb.WriteString(sectionStyle.Render("Requests Mode"))
	sb.WriteString("\n")
	sb.WriteString(renderHelpLine(keyStyle, "j/k or Up/Down", "Move down/up"))
	sb.WriteString(renderHelpLine(keyStyle, "PgDn/Space/f", "Page down"))
	sb.WriteString(renderHelpLine(keyStyle, "PgUp/b", "Page up"))
	sb.WriteString(renderHelpLine(keyStyle, "g/G", "First/last entry"))
	sb.WriteString(renderHelpLine(keyStyle, "Enter", "Open detail popover"))
	sb.WriteString(renderHelpLine(keyStyle, "f", "Toggle resource-type filter"))
	sb.WriteString(renderHelpLine(keyStyle, "s", "Cycle sort: Size/Time/URL"))
	sb.WriteString(renderHelpLine(keyStyle, "S", "Toggle sort direction"))
	sb.WriteString("\n")

	sb.WriteString(sectionStyle.Render("Timings Mode"))
	sb.WriteString("\n")
	sb.WriteString(renderHelpLine(keyStyle, "j/k or Up/Down", "Move down/up"))
	sb.WriteString(renderHelpLine(keyStyle, "PgDn/Space/f", "Page down"))
	sb.WriteString(renderHelpLine(keyStyle, "PgUp/b", "Page up"))
	sb.WriteString(renderHelpLine(keyStyle, "g/G", "First/last entry"))
	sb.WriteString(renderHelpLine(keyStyle, "Enter", "Open timing detail"))
	sb.WriteString(renderHelpLine(keyStyle, "+/-", "Zoom chart in/out"))
	sb.WriteString("\n")

	sb.WriteString(sectionStyle.Render("Search"))
	sb.WriteString("\n")
	sb.WriteString(renderHelpLine(keyStyle, "n/N", "Next/previous match"))
	sb.WriteString(renderHelpLine(keyStyle, "Esc", "Close search"))
	sb.WriteString("\n")

	sb.WriteString(sectionStyle.Render("Cache Mode"))
	sb.WriteString("\n")
	sb.WriteString(renderHelpLine(keyStyle, "Tab/Shift+Tab", "Switch between sections"))
	sb.WriteString(renderHelpLine(keyStyle, "j/k or Up/Down", "Scroll through list"))
	sb.WriteString(renderHelpLine(keyStyle, "g/G", "Jump to top/bottom"))
	sb.WriteString(renderHelpLine(keyStyle, "Enter", "Open detail popover"))
	sb.WriteString("\n")

	sb.WriteString(sectionStyle.Render("Security Mode"))
	sb.WriteString("\n")
	sb.WriteString(renderHelpLine(keyStyle, "j/k or Up/Down", "Move down/up"))
	sb.WriteString(renderHelpLine(keyStyle, "PgDn/Space/f", "Page down"))
	sb.WriteString(renderHelpLine(keyStyle, "PgUp/b", "Page up"))
	sb.WriteString(renderHelpLine(keyStyle, "g/G", "First/last entry"))
	sb.WriteString(renderHelpLine(keyStyle, "Enter", "Open header detail"))
	sb.WriteString("\n")

	sb.WriteString(sectionStyle.Render("Detail Popover"))
	sb.WriteString("\n")
	sb.WriteString(renderHelpLine(keyStyle, "↑/↓ or j/k", "Scroll content"))
	sb.WriteString(renderHelpLine(keyStyle, "Esc/q", "Close popover"))

	content := sb.String()
	m.viewport.SetContent(content)
	return tea.NewView(m.renderWithScrollbar(content))
}

func renderHelpLine(keyStyle lipgloss.Style, key, desc string) string {
	return "  " + keyStyle.Render(key) + "  " + desc + "\n"
}
