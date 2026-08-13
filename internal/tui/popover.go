package tui

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type Popover struct {
	content tea.Model
	active  bool
	width   int
	height  int
}

func NewPopover(content tea.Model) Popover {
	return Popover{
		content: content,
		active:  false,
	}
}

func (p *Popover) Open(content tea.Model, width, height int) {
	p.content = content
	p.width = width
	p.height = height
	p.active = true
}

func (p *Popover) Close() {
	p.active = false
}

func (p *Popover) IsActive() bool {
	return p.active
}

func (p *Popover) Content() tea.Model {
	return p.content
}

func (p *Popover) SetContent(content tea.Model) {
	p.content = content
}

func (p *Popover) Render(baseView string) string {
	if !p.active {
		return baseView
	}

	boxWidth := p.width * 2 / 3
	if boxWidth < 40 {
		boxWidth = 40
	}
	boxHeight := p.height * 2 / 3
	if boxHeight < 10 {
		boxHeight = 10
	}

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#aaaaaa")).
		Padding(1, 2).
		Width(boxWidth).
		Height(boxHeight)

	contentView := ""
	if p.content != nil {
		contentView = p.content.View().Content
	}

	box := boxStyle.Render(contentView)
	boxLines := strings.Split(box, "\n")

	dimStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#555555"))

	baseLines := strings.Split(baseView, "\n")
	totalLines := len(baseLines)

	boxStartY := (totalLines - len(boxLines)) / 2
	if boxStartY < 0 {
		boxStartY = 0
	}
	boxEndY := boxStartY + len(boxLines)

	result := make([]string, totalLines)
	for i := 0; i < totalLines; i++ {
		if i >= boxStartY && i < boxEndY {
			result[i] = boxLines[i-boxStartY]
		} else {
			result[i] = dimStyle.Render(padLine(baseLines[i], p.width))
		}
	}

	return strings.Join(result, "\n")
}

func padLine(line string, width int) string {
	w := lipgloss.Width(line)
	if w >= width {
		return line
	}
	return line + strings.Repeat(" ", width-w)
}
