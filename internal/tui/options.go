package tui

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/harboring/harboring/internal/config"
	"github.com/harboring/harboring/internal/theme"
)

type optionSetting struct {
	label   string
	options []string
	current int
}

type OptionsModel struct {
	settings *config.Settings
	theme    *theme.Engine
	cursor   int
	options  []optionSetting
	// themeTouched records whether the user adjusted the Color Theme option
	// in this panel session. Only then does closing the panel apply (and
	// persist) a theme change; otherwise the active theme — possibly pinned
	// by a --theme flag or a config-file theme — is left alone.
	themeTouched bool
	viewport     viewport.Model
	width        int
	height       int
}

func NewOptionsModel(settings *config.Settings, th *theme.Engine) OptionsModel {
	opts := []optionSetting{
		{label: "Color Theme", options: config.ThemeOptions, current: indexOf(config.ThemeOptions, settings.Theme)},
		{label: "Truecolor", options: config.TruecolorOptions, current: indexOf(config.TruecolorOptions, boolToOnOff(settings.Truecolor))},
		{label: "Graph Symbol", options: config.GraphSymbolOptions, current: indexOf(config.GraphSymbolOptions, settings.GraphSymbol)},
		{label: "Default Mode", options: config.DefaultModeOptions, current: indexOf(config.DefaultModeOptions, settings.DefaultMode)},
		{label: "Compact Mode", options: config.CompactModeOptions, current: indexOf(config.CompactModeOptions, boolToOnOff(settings.CompactMode))},
	}
	return OptionsModel{
		settings: settings,
		theme:    th,
		options:  opts,
		viewport: viewport.New(viewport.WithWidth(60), viewport.WithHeight(20)),
		width:    60,
		height:   20,
	}
}

func (m OptionsModel) Init() tea.Cmd {
	return nil
}

func (m OptionsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.viewport.SetWidth(msg.Width - 8)
		m.viewport.SetHeight(msg.Height - 8)
		return m, nil

	case tea.KeyPressMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

func (m OptionsModel) handleKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < len(m.options)-1 {
			m.cursor++
		}
	case "left", "h":
		opt := &m.options[m.cursor]
		opt.current--
		if opt.current < 0 {
			opt.current = len(opt.options) - 1
		}
		if opt.label == "Color Theme" {
			m.themeTouched = true
		}
	case "right", "l":
		opt := &m.options[m.cursor]
		opt.current = (opt.current + 1) % len(opt.options)
		if opt.label == "Color Theme" {
			m.themeTouched = true
		}
	case "g":
		m.cursor = 0
	case "G":
		m.cursor = len(m.options) - 1
	}
	return m, nil
}

func (m OptionsModel) SettingsFromOptions() *config.Settings {
	s := config.DefaultSettings()
	for _, opt := range m.options {
		val := opt.options[opt.current]
		switch opt.label {
		case "Color Theme":
			s.Theme = val
		case "Truecolor":
			s.Truecolor = val == "on"
		case "Graph Symbol":
			s.GraphSymbol = val
		case "Default Mode":
			s.DefaultMode = val
		case "Compact Mode":
			s.CompactMode = val == "on"
		}
	}
	return s
}

func (m OptionsModel) View() tea.View {
	titleStyle := m.theme.GetStyle("title")
	keyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(m.theme.GetFg())).Bold(true)
	valueStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(m.theme.GetWarn()))
	selectedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(m.theme.GetWarn())).Bold(true)
	arrowStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(m.theme.GetGray()))

	var sb strings.Builder
	sb.WriteString(titleStyle.Render("Options"))
	sb.WriteString("\n\n")

	for i, opt := range m.options {
		label := keyStyle.Render(fmt.Sprintf("  %-16s", opt.label))

		val := opt.options[opt.current]
		leftArrow := arrowStyle.Render(" ◄ ")
		rightArrow := arrowStyle.Render(" ► ")

		var valStr string
		if i == m.cursor {
			valStr = selectedStyle.Render(fmt.Sprintf("[%s]", val))
			sb.WriteString(label + leftArrow + valStr + rightArrow + "\n")
		} else {
			valStr = valueStyle.Render(fmt.Sprintf("[%s]", val))
			sb.WriteString(label + "  " + valStr + "\n")
		}
	}

	sb.WriteString("\n")
	footerStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(m.theme.GetGray()))
	sb.WriteString(footerStyle.Render("  ← → change  Esc close"))

	content := sb.String()
	m.viewport.SetContent(content)

	return tea.NewView(m.renderWithScrollbar(content))
}

func (m OptionsModel) renderWithScrollbar(rawContent string) string {
	content := m.viewport.View()
	lines := strings.Split(content, "\n")
	totalLines := len(strings.Split(rawContent, "\n"))
	visibleLines := m.viewport.Height()

	if totalLines <= visibleLines {
		return content
	}

	scrollbar := m.buildScrollbar(totalLines, visibleLines, m.viewport.YOffset())
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

func (m OptionsModel) buildScrollbar(total, visible, offset int) []string {
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
		if i == 0 {
			result[i] = "▲"
		} else if i == scrollbarHeight-1 {
			result[i] = "▼"
		} else if i == barPos {
			result[i] = "█"
		} else {
			result[i] = "│"
		}
	}
	return result
}

func indexOf(slice []string, val string) int {
	for i, s := range slice {
		if s == val {
			return i
		}
	}
	return 0
}

func boolToOnOff(b bool) string {
	if b {
		return "on"
	}
	return "off"
}
