package modes

import (
	"fmt"
	"math"
	"strings"

	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/harboring/harboring/internal/har"
	"github.com/harboring/harboring/internal/theme"
)

type WaterfallColumns struct {
	DNS     bool
	Connect bool
	SSL     bool
	Send    bool
	Wait    bool
	Receive bool
}

type WaterfallModel struct {
	entries     []*har.EntryDetails
	rawEntries  []*har.Node
	cursor      int
	scrollY     int
	viewport    viewport.Model
	theme       *theme.Engine
	columns     WaterfallColumns
	zoom        float64
	width       int
	height      int
	graphSymbol string
}

func NewWaterfallModel(entries []*har.EntryDetails, rawEntries []*har.Node, th *theme.Engine) WaterfallModel {
	vp := viewport.New(viewport.WithWidth(80), viewport.WithHeight(20))
	return WaterfallModel{
		entries:     entries,
		rawEntries:  rawEntries,
		cursor:      0,
		viewport:    vp,
		theme:       th,
		columns: WaterfallColumns{
			DNS:     true,
			Connect: true,
			SSL:     true,
			Send:    true,
			Wait:    true,
			Receive: true,
		},
		zoom:        1.0,
		width:       80,
		height:      20,
		graphSymbol: "\u2588",
	}
}

func (m WaterfallModel) Selected() *har.EntryDetails {
	if m.cursor >= 0 && m.cursor < len(m.entries) {
		return m.entries[m.cursor]
	}
	return nil
}

func (m WaterfallModel) SelectedRaw() *har.Node {
	if m.cursor >= 0 && m.cursor < len(m.rawEntries) {
		return m.rawEntries[m.cursor]
	}
	return nil
}

func (m *WaterfallModel) SetViewportSize(w, h int) {
	m.width = w
	m.height = h
	m.viewport.SetWidth(w)
	m.viewport.SetHeight(h)
}

func (m *WaterfallModel) SetGraphSymbol(sym string) {
	switch sym {
	case "braille":
		m.graphSymbol = "\u28ff"
	case "ascii":
		m.graphSymbol = "#"
	default:
		m.graphSymbol = "\u2588"
	}
}

func (m WaterfallModel) Update(msg tea.Msg) (WaterfallModel, tea.Cmd) {
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

func (m WaterfallModel) handleKey(msg tea.KeyPressMsg) (WaterfallModel, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < len(m.entries)-1 {
			m.cursor++
		}
	case "pgdown", "space", "f":
		m.cursor += m.viewport.Height()
		if m.cursor >= len(m.entries) {
			m.cursor = len(m.entries) - 1
		}
	case "pgup", "b":
		m.cursor -= m.viewport.Height()
		if m.cursor < 0 {
			m.cursor = 0
		}
	case "g":
		m.cursor = 0
	case "G":
		m.cursor = len(m.entries) - 1
	case "+", "=":
		m.zoom *= 1.2
		if m.zoom > 10 {
			m.zoom = 10
		}
	case "-":
		m.zoom /= 1.2
		if m.zoom < 0.1 {
			m.zoom = 0.1
		}
	case "C":
		// Column picker handled by parent
	}

	m.updateScrollY()
	return m, nil
}

func (m *WaterfallModel) updateScrollY() {
	totalLines := len(m.entries)
	cursorLine := m.cursor
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

func (m WaterfallModel) View() string {
	if len(m.entries) == 0 {
		return "No entries"
	}

	urlWidth := 0
	for _, e := range m.entries {
		path := e.URLPath
		if path == "" {
			path = e.URL
		}
		if w := lipgloss.Width(path); w > urlWidth {
			urlWidth = w
		}
	}
	if urlWidth > m.width*40/100 {
		urlWidth = m.width * 40 / 100
	}
	if urlWidth < 10 {
		urlWidth = 10
	}
	barAreaWidth := m.width - urlWidth - 3
	if barAreaWidth < 10 {
		barAreaWidth = 10
	}

	var maxTime float64
	for _, e := range m.entries {
		if t := m.entryTime(e); t > maxTime {
			maxTime = t
		}
	}

	var allLines []string
	for i, e := range m.entries {
		displayURL := e.URLPath
		if displayURL == "" {
			displayURL = e.URL
		}
		if lipgloss.Width(displayURL) > urlWidth {
			displayURL = displayURL[:urlWidth-3] + "..."
		}

		urlPart := fmt.Sprintf("%-*s", urlWidth, displayURL)

		barStr := m.renderBar(e, barAreaWidth, maxTime)

		line := urlPart + "  " + barStr

		if i == m.cursor {
			selectedStyle := lipgloss.NewStyle().
				Background(lipgloss.Color(m.theme.GetBg())).
				Foreground(lipgloss.Color(m.theme.GetFg()))
			line = selectedStyle.Render(line)
		}

		allLines = append(allLines, line)
	}

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

	m.viewport.SetContent(strings.Join(allLines, "\n"))
	return result.String()
}

func (m WaterfallModel) renderBar(e *har.EntryDetails, barWidth int, maxTime float64) string {
	totalMs := m.entryTime(e)
	if totalMs == 0 {
		return ""
	}

	totalBarWidth := int(float64(barWidth) * m.zoom * totalMs / maxTime)
	if totalBarWidth < 1 {
		totalBarWidth = 1
	}

	var sb strings.Builder

	timingWidths := m.getTimingWidths(e, totalBarWidth)

	if m.columns.DNS && e.DNS > 0 {
		chunk := m.renderChunk(timingWidths.dns, m.theme.GetStyle("waterfall_dns"))
		sb.WriteString(chunk)
	}
	if m.columns.Connect && e.Connect > 0 {
		chunk := m.renderChunk(timingWidths.connect, m.theme.GetStyle("waterfall_connect"))
		sb.WriteString(chunk)
	}
	if m.columns.SSL && e.SSL > 0 {
		chunk := m.renderChunk(timingWidths.ssl, m.theme.GetStyle("waterfall_ssl"))
		sb.WriteString(chunk)
	}
	if m.columns.Send && e.Send > 0 {
		chunk := m.renderChunk(timingWidths.send, m.theme.GetStyle("waterfall_send"))
		sb.WriteString(chunk)
	}
	if m.columns.Wait && e.Wait > 0 {
		chunk := m.renderChunk(timingWidths.wait, m.theme.GetStyle("waterfall_wait"))
		sb.WriteString(chunk)
	}
	if m.columns.Receive && e.Receive > 0 {
		chunk := m.renderChunk(timingWidths.receive, m.theme.GetStyle("waterfall_receive"))
		sb.WriteString(chunk)
	}

	return sb.String()
}

type timingWidths struct {
	dns     int
	connect int
	ssl     int
	send    int
	wait    int
	receive int
}

func (m WaterfallModel) entryTime(e *har.EntryDetails) float64 {
	sum := 0.0
	if e.DNS > 0 {
		sum += e.DNS
	}
	if e.Connect > 0 {
		sum += e.Connect
	}
	if e.SSL > 0 {
		sum += e.SSL
	}
	if e.Send > 0 {
		sum += e.Send
	}
	if e.Wait > 0 {
		sum += e.Wait
	}
	if e.Receive > 0 {
		sum += e.Receive
	}
	return sum
}

func (m WaterfallModel) getTimingWidths(e *har.EntryDetails, totalWidth int) timingWidths {
	totalMs := m.entryTime(e)
	if totalMs == 0 {
		return timingWidths{}
	}

	return timingWidths{
		dns:     int(math.Round(math.Max(e.DNS, 0) / totalMs * float64(totalWidth))),
		connect: int(math.Round(math.Max(e.Connect, 0) / totalMs * float64(totalWidth))),
		ssl:     int(math.Round(math.Max(e.SSL, 0) / totalMs * float64(totalWidth))),
		send:    int(math.Round(math.Max(e.Send, 0) / totalMs * float64(totalWidth))),
		wait:    int(math.Round(math.Max(e.Wait, 0) / totalMs * float64(totalWidth))),
		receive: int(math.Round(math.Max(e.Receive, 0) / totalMs * float64(totalWidth))),
	}
}

func (m WaterfallModel) renderChunk(width int, style lipgloss.Style) string {
	if width < 1 {
		return ""
	}
	return style.Render(strings.Repeat(m.graphSymbol, width))
}
