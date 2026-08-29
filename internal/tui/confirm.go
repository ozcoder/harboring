package tui

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/harboring/harboring/internal/theme"
)

// confirmMsg reports the outcome of a confirmModel popover.
type confirmMsg struct {
	confirmed bool
}

// confirmModel is a lightweight y/n confirmation popover used to guard
// destructive actions such as scrubbing response bodies.
type confirmModel struct {
	theme  *theme.Engine
	prompt string
}

func newConfirmModel(th *theme.Engine, prompt string) *confirmModel {
	return &confirmModel{theme: th, prompt: prompt}
}

func (c *confirmModel) Init() tea.Cmd {
	return nil
}

func (c *confirmModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if kp, ok := msg.(tea.KeyPressMsg); ok {
		switch kp.String() {
		case "y", "Y", "enter":
			return c, func() tea.Msg { return confirmMsg{confirmed: true} }
		case "n", "N", "esc", "q":
			return c, func() tea.Msg { return confirmMsg{confirmed: false} }
		}
	}
	return c, nil
}

func (c *confirmModel) View() tea.View {
	var sb strings.Builder

	sb.WriteString(lipgloss.NewStyle().
		Foreground(lipgloss.Color(c.theme.GetWarn())).
		Bold(true).
		Render("Confirm"))
	sb.WriteString("\n\n")

	sb.WriteString(c.prompt)
	sb.WriteString("\n\n")

	dimStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(c.theme.GetGray()))
	sb.WriteString(dimStyle.Render("  y = confirm    n / Esc = cancel"))

	return tea.NewView(sb.String())
}
