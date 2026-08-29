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

type TimingsColumns struct {
	DNS     bool
	Connect bool
	SSL     bool
	Send    bool
	Wait    bool
	Receive bool
}

type TimingsModel struct {
	entries       []*har.EntryDetails
	rawEntries    []*har.Node
	cursor        int
	scrollY       int
	viewport      viewport.Model
	theme         *theme.Engine
	columns       TimingsColumns
	barGraphWidth int
	hScroll       int
	width         int
	height        int
	graphSymbol   string
}

func NewTimingsModel(entries []*har.EntryDetails, rawEntries []*har.Node, th *theme.Engine) TimingsModel {
	vp := viewport.New(viewport.WithWidth(80), viewport.WithHeight(20))
	return TimingsModel{
		entries:    entries,
		rawEntries: rawEntries,
		cursor:     0,
		viewport:   vp,
		theme:      th,
		columns: TimingsColumns{
			DNS:     true,
			Connect: true,
			SSL:     true,
			Send:    true,
			Wait:    true,
			Receive: true,
		},
		width:       80,
		height:      20,
		graphSymbol: "\u2588",
	}
}

func (m TimingsModel) Selected() *har.EntryDetails {
	if m.cursor >= 0 && m.cursor < len(m.entries) {
		return m.entries[m.cursor]
	}
	return nil
}

func (m TimingsModel) SelectedRaw() *har.Node {
	if m.cursor >= 0 && m.cursor < len(m.rawEntries) {
		return m.rawEntries[m.cursor]
	}
	return nil
}

func (m *TimingsModel) SetViewportSize(w, h int) {
	m.width = w
	m.height = h
	m.viewport.SetWidth(w)
	m.viewport.SetHeight(h)
}

func (m *TimingsModel) SetGraphSymbol(sym string) {
	switch sym {
	case "braille":
		m.graphSymbol = "\u28ff"
	case "ascii":
		m.graphSymbol = "#"
	default:
		m.graphSymbol = "\u2588"
	}
}

func (m TimingsModel) Update(msg tea.Msg) (TimingsModel, tea.Cmd) {
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

func (m TimingsModel) handleKey(msg tea.KeyPressMsg) (TimingsModel, tea.Cmd) {
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
		if m.barGraphWidth <= 0 {
			m.barGraphWidth = m.visibleBarWidth()
		}
		m.barGraphWidth += m.barStep()
	case "-":
		if m.barGraphWidth > 0 {
			m.barGraphWidth -= m.barStep()
			if m.barGraphWidth <= 0 {
				m.barGraphWidth = 0
			}
		}
	case "left", "h":
		m.hScroll -= m.hScrollStep()
		if m.hScroll < 0 {
			m.hScroll = 0
		}
	case "right", "l":
		m.hScroll += m.hScrollStep()
	}

	m.clampHScroll()
	m.updateScrollY()
	return m, nil
}

func (m TimingsModel) urlWidth() int {
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
	return urlWidth
}

func (m TimingsModel) visibleBarWidth() int {
	barWidth := m.width - m.urlWidth() - 3
	if barWidth < 10 {
		barWidth = 10
	}
	return barWidth
}

func (m TimingsModel) barGraphWidthEffective() int {
	if m.barGraphWidth <= 0 {
		return m.visibleBarWidth()
	}
	return m.barGraphWidth
}

func (m TimingsModel) barStep() int {
	step := m.visibleBarWidth() / 4
	if step < 10 {
		step = 10
	}
	return step
}

func (m TimingsModel) hScrollStep() int {
	step := m.visibleBarWidth() / 3
	if step < 1 {
		step = 1
	}
	return step
}

func (m TimingsModel) dataHeight() int {
	h := m.viewport.Height() - 1
	if h < 0 {
		h = 0
	}
	return h
}

func (m *TimingsModel) clampHScroll() {
	visibleWindow := m.visibleBarWidth()
	maxHScroll := m.barGraphWidthEffective() - visibleWindow
	if maxHScroll < 0 {
		maxHScroll = 0
	}
	if m.hScroll > maxHScroll {
		m.hScroll = maxHScroll
	}
	if m.hScroll < 0 {
		m.hScroll = 0
	}
}

func (m *TimingsModel) updateScrollY() {
	totalLines := len(m.entries)
	cursorLine := m.cursor
	height := m.dataHeight()

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

func (m TimingsModel) View() string {
	if len(m.entries) == 0 {
		return "No entries"
	}

	urlWidth := m.urlWidth()
	visibleWindow := m.visibleBarWidth()
	graphWidth := m.barGraphWidthEffective()
	m.clampHScroll()

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
			displayURL = truncateWidth(displayURL, urlWidth)
		}

		isSelected := i == m.cursor
		plainStyle := lipgloss.NewStyle()
		if isSelected {
			plainStyle = plainStyle.
				Background(lipgloss.Color(m.theme.GetSelectionBg())).
				Foreground(lipgloss.Color(m.theme.GetFg()))
		}

		urlPart := plainStyle.Render(fmt.Sprintf("%-*s", urlWidth, displayURL))
		barStr := m.renderBarWindow(e, graphWidth, visibleWindow, m.hScroll, maxTime, isSelected)

		line := urlPart + plainStyle.Render("  ") + barStr

		if isSelected {
			if pad := m.width - 1 - lipgloss.Width(line); pad > 0 {
				line += plainStyle.Render(strings.Repeat(" ", pad))
			}
		}

		allLines = append(allLines, line)
	}

	totalLines := len(allLines)
	height := m.dataHeight()
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

	// Sticky horizontal scrollbar row: blank until the bar graph area is
	// wider than the visible window, then it shows the scroll position.
	trackStyle, thumbStyle := hScrollbarStyles(m.theme)
	scrollbar := renderHScrollbar(visibleWindow, graphWidth, m.hScroll, trackStyle, thumbStyle)
	result.WriteString(" ")
	result.WriteString(strings.Repeat(" ", urlWidth))
	result.WriteString("  ")
	if scrollbar == "" {
		result.WriteString(strings.Repeat(" ", visibleWindow))
	} else {
		result.WriteString(scrollbar)
	}
	result.WriteString("\n")

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

// renderBarWindow renders only the visible portion of the bar graph for one
// entry. The bar is laid out on a virtual canvas graphWidth wide and the
// window [scroll, scroll+visibleWidth) is drawn, so the request rows stay
// fixed while the bar graph scrolls horizontally.
func (m TimingsModel) renderBarWindow(e *har.EntryDetails, graphWidth, visibleWidth, scroll int, maxTime float64, selected bool) string {
	if visibleWidth < 1 {
		return ""
	}

	plainStyle := lipgloss.NewStyle()
	if selected {
		plainStyle = plainStyle.
			Background(lipgloss.Color(m.theme.GetSelectionBg())).
			Foreground(lipgloss.Color(m.theme.GetFg()))
	}

	space := func(n int) string {
		if n < 1 {
			return ""
		}
		if selected {
			return plainStyle.Render(strings.Repeat(" ", n))
		}
		return strings.Repeat(" ", n)
	}

	totalMs := m.entryTime(e)
	if totalMs == 0 {
		return space(visibleWidth)
	}

	bg := lipgloss.Color(m.theme.GetSelectionBg())
	withBG := func(s lipgloss.Style) lipgloss.Style {
		if selected {
			return s.Background(bg)
		}
		return s
	}

	totalBarWidth := int(float64(graphWidth) * totalMs / maxTime)
	if totalBarWidth < 1 {
		totalBarWidth = 1
	}

	type chunk struct {
		start, width int
		style        lipgloss.Style
	}
	var chunks []chunk
	pos := 0
	add := func(w int, st lipgloss.Style) {
		if w < 1 {
			return
		}
		chunks = append(chunks, chunk{start: pos, width: w, style: st})
		pos += w
	}

	widths := m.getTimingWidths(e, totalBarWidth)
	if m.columns.DNS && e.DNS > 0 {
		add(widths.dns, withBG(m.theme.GetStyle("timing_dns")))
	}
	if m.columns.Connect && e.Connect > 0 {
		add(widths.connect, withBG(m.theme.GetStyle("timing_connect")))
	}
	if m.columns.SSL && e.SSL > 0 {
		add(widths.ssl, withBG(m.theme.GetStyle("timing_ssl")))
	}
	if m.columns.Send && e.Send > 0 {
		add(widths.send, withBG(m.theme.GetStyle("timing_send")))
	}
	if m.columns.Wait && e.Wait > 0 {
		add(widths.wait, withBG(m.theme.GetStyle("timing_wait")))
	}
	if m.columns.Receive && e.Receive > 0 {
		add(widths.receive, withBG(m.theme.GetStyle("timing_receive")))
	}

	end := scroll + visibleWidth
	var sb strings.Builder
	cursor := scroll
	for _, c := range chunks {
		if c.start+c.width <= scroll {
			continue
		}
		if c.start >= end {
			break
		}
		lo := max(c.start, scroll)
		hi := min(c.start+c.width, end)
		sb.WriteString(space(lo - cursor))
		sb.WriteString(c.style.Render(strings.Repeat(m.graphSymbol, hi-lo)))
		cursor = hi
	}
	sb.WriteString(space(end - cursor))
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

func (m TimingsModel) entryTime(e *har.EntryDetails) float64 {
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

func (m TimingsModel) getTimingWidths(e *har.EntryDetails, totalWidth int) timingWidths {
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
