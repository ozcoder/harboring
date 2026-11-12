package modes

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/harboring/harboring/internal/har"
	"github.com/harboring/harboring/internal/theme"
)

// TimelineItem is a single request positioned on a shared absolute time axis.
type TimelineItem struct {
	entry   *har.EntryDetails
	raw     *har.Node
	startMs float64
	durMs   float64
}

// TimelineModel renders all requests on one shared time axis so that parallel
// requests line up and serialization / queueing is visible. Missing fields are
// tolerated: unparseable start times fall back to sequential placement and
// negative ("not applicable") timings are treated as zero.
type TimelineModel struct {
	entries       []*har.EntryDetails
	rawEntries    []*har.Node
	items         []TimelineItem
	maxEnd        float64
	cursor        int
	scrollY       int
	viewport      viewport.Model
	theme         *theme.Engine
	width         int
	height        int
	graphSymbol   string
	barGraphWidth int
	hScroll       int
}

const timelineTimeLayout = "2006-01-02T15:04:05.999Z07:00"

func NewTimelineModel(entries []*har.EntryDetails, rawEntries []*har.Node, th *theme.Engine) TimelineModel {
	m := TimelineModel{
		entries:     entries,
		rawEntries:  rawEntries,
		cursor:      0,
		viewport:    viewport.New(viewport.WithWidth(80), viewport.WithHeight(20)),
		theme:       th,
		graphSymbol: "\u2588",
	}
	m.compute()
	return m
}

func (m *TimelineModel) compute() {
	starts := make([]time.Time, len(m.entries))
	ok := make([]bool, len(m.entries))
	minStart := time.Time{}
	have := false
	for i, e := range m.entries {
		t, err := time.Parse(timelineTimeLayout, strings.TrimSpace(e.Started))
		if err == nil {
			starts[i] = t
			ok[i] = true
			if !have || t.Before(minStart) {
				minStart = t
				have = true
			}
		}
	}

	items := make([]TimelineItem, len(m.entries))
	cursorPos := 0.0
	for i, e := range m.entries {
		var startMs float64
		if ok[i] {
			startMs = starts[i].Sub(minStart).Seconds() * 1000.0
			if startMs < 0 {
				startMs = 0
			}
		} else {
			startMs = cursorPos
		}
		dur := durationMs(e)
		var raw *har.Node
		if i < len(m.rawEntries) {
			raw = m.rawEntries[i]
		}
		items[i] = TimelineItem{entry: e, raw: raw, startMs: startMs, durMs: dur}
		cursorPos = startMs + dur
	}

	sort.SliceStable(items, func(i, j int) bool {
		return items[i].startMs < items[j].startMs
	})
	m.items = items

	maxEnd := 0.0
	for _, it := range items {
		end := it.startMs + it.durMs
		if end > maxEnd {
			maxEnd = end
		}
	}
	if maxEnd <= 0 {
		maxEnd = 1
	}
	m.maxEnd = maxEnd
}

func durationMs(e *har.EntryDetails) float64 {
	if e == nil {
		return 0
	}
	if e.Time > 0 {
		return e.Time
	}
	return pos(e.Blocked) + pos(e.DNS) + pos(e.Connect) + pos(e.SSL) + pos(e.Send) + pos(e.Wait) + pos(e.Receive)
}

func pos(v float64) float64 {
	if v < 0 {
		return 0
	}
	return v
}

func (m *TimelineModel) SetViewportSize(w, h int) {
	m.width = w
	m.height = h
	m.viewport.SetWidth(w)
	m.viewport.SetHeight(h)
}

func (m *TimelineModel) SetGraphSymbol(sym string) {
	switch sym {
	case "braille":
		m.graphSymbol = "\u28ff"
	case "ascii":
		m.graphSymbol = "#"
	default:
		m.graphSymbol = "\u2588"
	}
}

func (m TimelineModel) Selected() *TimelineItem {
	if m.cursor >= 0 && m.cursor < len(m.items) {
		return &m.items[m.cursor]
	}
	return nil
}

func (it *TimelineItem) Entry() *har.EntryDetails { return it.entry }
func (it *TimelineItem) Raw() *har.Node           { return it.raw }

func (m TimelineModel) Init() tea.Cmd {
	return nil
}

func (m TimelineModel) Update(msg tea.Msg) (TimelineModel, tea.Cmd) {
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

func (m TimelineModel) handleKey(msg tea.KeyPressMsg) (TimelineModel, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < len(m.items)-1 {
			m.cursor++
		}
	case "pgdown", "space", "f":
		m.cursor += m.viewport.Height()
		if m.cursor >= len(m.items) {
			m.cursor = len(m.items) - 1
		}
	case "pgup", "b":
		m.cursor -= m.viewport.Height()
		if m.cursor < 0 {
			m.cursor = 0
		}
	case "g":
		m.cursor = 0
	case "G":
		m.cursor = len(m.items) - 1
	case "+", "=":
		if m.barGraphWidth <= 0 {
			m.barGraphWidth = m.timelineVisibleWidth()
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

func (m TimelineModel) timelineVisibleWidth() int {
	w := m.width - m.labelWidth() - 3
	if w < 10 {
		w = 10
	}
	return w
}

func (m TimelineModel) barGraphWidthEffective() int {
	if m.barGraphWidth <= 0 {
		return m.timelineVisibleWidth()
	}
	return m.barGraphWidth
}

func (m TimelineModel) barStep() int {
	step := m.timelineVisibleWidth() / 4
	if step < 10 {
		step = 10
	}
	return step
}

func (m TimelineModel) hScrollStep() int {
	step := m.timelineVisibleWidth() / 3
	if step < 1 {
		step = 1
	}
	return step
}

// dataHeight is the number of request rows visible below the sticky
// horizontal scrollbar row and the "Request"/scale header row.
func (m TimelineModel) dataHeight() int {
	h := m.viewport.Height() - 2
	if h < 1 {
		h = 1
	}
	return h
}

func (m *TimelineModel) clampHScroll() {
	visibleWindow := m.timelineVisibleWidth()
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

func (m *TimelineModel) updateScrollY() {
	height := m.dataHeight()
	cursorLine := m.cursor
	yOff := m.scrollY
	if cursorLine < yOff {
		yOff = cursorLine
	} else if cursorLine >= yOff+height {
		yOff = cursorLine - height + 1
	}
	maxYOff := len(m.items) - height
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

func (m TimelineModel) labelWidth() int {
	maxW := 0
	for _, it := range m.items {
		e := it.entry
		path := e.URLPath
		if path == "" {
			path = e.URL
		}
		display := fmt.Sprintf("%s %s", e.Method, path)
		if w := lipgloss.Width(display); w > maxW {
			maxW = w
		}
	}
	limit := m.width * 30 / 100
	if maxW > limit {
		maxW = limit
	}
	if maxW < 12 {
		maxW = 12
	}
	return maxW
}

func (m TimelineModel) View() string {
	if len(m.items) == 0 {
		return "No entries"
	}

	labelWidth := m.labelWidth()
	visibleWindow := m.timelineVisibleWidth()
	graphWidth := m.barGraphWidthEffective()
	m.clampHScroll()

	scale := m.renderScale(visibleWindow, m.hScroll, graphWidth)
	header := fmt.Sprintf("%-*s", labelWidth, "Request") + "  " + scale

	var itemLines []string
	for i, it := range m.items {
		itemLines = append(itemLines, m.renderRow(it, labelWidth, graphWidth, visibleWindow, m.hScroll, i == m.cursor))
	}

	height := m.dataHeight()
	yOff := m.scrollY
	maxYOff := len(itemLines) - height
	if maxYOff < 0 {
		maxYOff = 0
	}
	if yOff > maxYOff {
		yOff = maxYOff
		m.scrollY = yOff
	}
	if yOff < 0 {
		yOff = 0
	}

	endLine := yOff + height
	if endLine > len(itemLines) {
		endLine = len(itemLines)
	}
	visible := itemLines[yOff:endLine]
	scrollChars := buildScrollChars(len(itemLines), yOff, len(visible))

	var result strings.Builder

	// Sticky horizontal scrollbar row: blank until the timeline area is wider
	// than the visible window, then it shows the scroll position.
	trackStyle, thumbStyle := hScrollbarStyles(m.theme)
	scrollbar := renderHScrollbar(visibleWindow, graphWidth, m.hScroll, trackStyle, thumbStyle)
	result.WriteString(" ")
	result.WriteString(strings.Repeat(" ", labelWidth))
	result.WriteString("  ")
	if scrollbar == "" {
		result.WriteString(strings.Repeat(" ", visibleWindow))
	} else {
		result.WriteString(scrollbar)
	}
	result.WriteString("\n")

	result.WriteString(" ")
	result.WriteString(header)
	result.WriteString("\n")
	for i, line := range visible {
		sc := " "
		if i < len(scrollChars) {
			sc = scrollChars[i]
		}
		result.WriteString(sc)
		result.WriteString(line)
		result.WriteString("\n")
	}

	m.viewport.SetContent(strings.Join(itemLines, "\n"))
	return result.String()
}

// renderScale draws the time markers for the window [scroll, scroll+windowWidth)
// of a virtual timeline graphWidth wide. It picks a "nice" tick step so markers
// stay readable; as the timeline area widens the pixels-per-ms grows, so a
// smaller step satisfies the minimum spacing and more time markers appear.
// Labels that would be cut off by the window edges are skipped entirely.
func (m TimelineModel) renderScale(windowWidth, scroll, graphWidth int) string {
	cells := make([]byte, windowWidth)
	for i := range cells {
		cells[i] = ' '
	}

	maxEnd := m.maxEnd
	if maxEnd <= 0 || windowWidth < 1 || graphWidth < 1 {
		return string(cells)
	}

	ppms := float64(graphWidth) / maxEnd
	minSpacing := 8.0
	step := niceStep(minSpacing / ppms)

	windowEnd := scroll + windowWidth
	for k := 0; ; k++ {
		t := step * float64(k)
		if t > maxEnd+1e-9 {
			break
		}
		x := int(math.Round(t / maxEnd * float64(graphWidth)))
		label := formatTime(t)
		if x < scroll || x+len(label) > windowEnd {
			continue
		}
		for j, r := range label {
			// #nosec G115 - r is guaranteed to be within byte bounds here
			cells[x-scroll+j] = byte(r)
		}
	}
	return string(cells)
}

// niceStep returns the smallest "nice" tick step (1, 2, 5, 10, 20, 50, ...)
// that is >= min.
func niceStep(min float64) float64 {
	if min <= 0 {
		return 1
	}
	for pow := 1.0; ; pow *= 10 {
		for _, mult := range []float64{1, 2, 5} {
			if candidate := mult * pow; candidate >= min {
				return candidate
			}
		}
	}
}

func (m TimelineModel) renderRow(it TimelineItem, labelWidth, graphWidth, visibleWindow, scroll int, selected bool) string {
	e := it.entry
	path := e.URLPath
	if path == "" {
		path = e.URL
	}
	display := fmt.Sprintf("%s %s", e.Method, path)
	if lipgloss.Width(display) > labelWidth {
		display = truncateWidth(display, labelWidth)
	}

	plainStyle := lipgloss.NewStyle()
	if selected {
		plainStyle = plainStyle.
			Background(lipgloss.Color(m.theme.GetSelectionBg())).
			Foreground(lipgloss.Color(m.theme.GetFg()))
	}

	labelPart := plainStyle.Render(fmt.Sprintf("%-*s", labelWidth, display))
	line := labelPart + plainStyle.Render("  ") + m.renderBarWindow(it, graphWidth, visibleWindow, scroll, selected)

	if selected {
		if pad := m.width - 1 - lipgloss.Width(line); pad > 0 {
			line += plainStyle.Render(strings.Repeat(" ", pad))
		}
	}
	return line
}

// renderBarWindow renders only the visible portion of one request's bar on the
// shared time axis. The bar is laid out on a virtual canvas graphWidth wide
// starting at the request's start time and the window
// [scroll, scroll+visibleWidth) is drawn, so the request labels stay fixed
// while the timeline scrolls horizontally.
func (m TimelineModel) renderBarWindow(it TimelineItem, graphWidth, visibleWidth, scroll int, selected bool) string {
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

	startX := int(math.Round(it.startMs / m.maxEnd * float64(graphWidth)))
	if startX < 0 {
		startX = 0
	}
	if startX > graphWidth {
		startX = graphWidth
	}

	phases := []struct {
		val float64
		key string
	}{
		{it.entry.Blocked, "timing_blocked"},
		{it.entry.DNS, "timing_dns"},
		{it.entry.Connect, "timing_connect"},
		{it.entry.SSL, "timing_ssl"},
		{it.entry.Send, "timing_send"},
		{it.entry.Wait, "timing_wait"},
		{it.entry.Receive, "timing_receive"},
	}

	bg := lipgloss.Color(m.theme.GetSelectionBg())
	withBG := func(s lipgloss.Style) lipgloss.Style {
		if selected {
			return s.Background(bg)
		}
		return s
	}

	type chunk struct {
		start, width int
		style        lipgloss.Style
	}
	var chunks []chunk
	posX := startX
	for _, p := range phases {
		w := int(math.Round(pos(p.val) / m.maxEnd * float64(graphWidth)))
		if w < 0 {
			w = 0
		}
		if w > 0 {
			chunks = append(chunks, chunk{start: posX, width: w, style: withBG(m.theme.GetStyle(p.key))})
			posX += w
		}
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
