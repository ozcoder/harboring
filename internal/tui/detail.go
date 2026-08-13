package tui

import (
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"

	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/harboring/harboring/internal/har"
	"github.com/harboring/harboring/internal/theme"
)

type DetailModel struct {
	entry         *har.EntryDetails
	rawNode       *har.Node
	viewport      viewport.Model
	theme         *theme.Engine
	width         int
	height        int
	customContent string
	graphSymbol   string
}

func NewDetailModel(entry *har.EntryDetails, rawNode *har.Node, th *theme.Engine) DetailModel {
	m := DetailModel{
		entry:    entry,
		rawNode:  rawNode,
		viewport: viewport.New(viewport.WithWidth(80), viewport.WithHeight(20)),
		theme:    th,
		width:    80,
		height:   20,
		graphSymbol: "\u2588",
	}
	m.refreshContent()
	return m
}

func (m *DetailModel) SetGraphSymbol(sym string) {
	switch sym {
	case "braille":
		m.graphSymbol = "\u28ff"
	case "ascii":
		m.graphSymbol = "#"
	default:
		m.graphSymbol = "\u2588"
	}
	m.refreshContent()
}

func (m *DetailModel) SetCustomContent(content string) {
	m.customContent = content
	m.refreshContent()
}

func (m *DetailModel) refreshContent() {
	if m.entry == nil && m.customContent == "" {
		return
	}
	m.viewport.SetContent(m.buildContent())
}

func (m DetailModel) buildContent() string {
	if m.customContent != "" {
		return m.customContent
	}
	if m.entry == nil {
		return ""
	}

	titleStyle := m.theme.GetStyle("title")
	methodStyle := m.theme.GetStyle(fmt.Sprintf("method_%s", strings.ToLower(m.entry.Method)))

	var sb strings.Builder

	sb.WriteString(titleStyle.Render("Request"))
	sb.WriteString("\n\n")

	sb.WriteString("  Method:  ")
	sb.WriteString(methodStyle.Render(m.entry.Method))
	sb.WriteString("\n")

	sb.WriteString("  URL:     " + m.entry.URL + "\n")
	sb.WriteString("  Started: " + m.entry.Started + "\n")
	sb.WriteString("\n")

	sb.WriteString(m.renderHeaders("Request Headers", m.entry.RequestHeaders))

	sb.WriteString(titleStyle.Render("Response"))
	sb.WriteString("\n\n")

	statusStyle := getStatusStyle(m.theme, m.entry.Status)
	sb.WriteString("  Status:  ")
	sb.WriteString(statusStyle.Render(fmt.Sprintf("%d %s", m.entry.Status, m.entry.StatusText)))
	sb.WriteString("\n")

	sb.WriteString(fmt.Sprintf("  Size:    %s\n", formatDetailSize(m.entry.Size)))
	sb.WriteString("\n")

	if m.entry.MimeType != "" {
		sb.WriteString(fmt.Sprintf("  Type:    %s\n", m.entry.MimeType))
	}

	if m.entry.ShouldBeCompressed {
		if m.entry.IsCompressed {
			sb.WriteString(fmt.Sprintf("  Encoding: %s ✓\n", m.entry.ContentEncoding))
		} else {
			warnStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color(m.theme.GetWarn())).
				Bold(true)
			sb.WriteString("  Encoding: ")
			sb.WriteString(warnStyle.Render("⚠ not compressed"))
			sb.WriteString("\n")
		}
	} else if m.entry.ContentEncoding != "" {
		sb.WriteString(fmt.Sprintf("  Encoding: %s\n", m.entry.ContentEncoding))
	}

	sb.WriteString("\n")

	sb.WriteString(m.renderHeaders("Response Headers", m.entry.ResponseHeaders))

	sb.WriteString(titleStyle.Render("Timings"))
	sb.WriteString("\n\n")

	timingBarWidth := 30
	sb.WriteString(m.renderTimingBar("DNS", m.entry.DNS, m.entry.Time, timingBarWidth, m.theme.GetStyle("waterfall_dns")))
	sb.WriteString(m.renderTimingBar("Connect", m.entry.Connect, m.entry.Time, timingBarWidth, m.theme.GetStyle("waterfall_connect")))
	sb.WriteString(m.renderTimingBar("SSL", m.entry.SSL, m.entry.Time, timingBarWidth, m.theme.GetStyle("waterfall_ssl")))
	sb.WriteString(m.renderTimingBar("Send", m.entry.Send, m.entry.Time, timingBarWidth, m.theme.GetStyle("waterfall_send")))
	sb.WriteString(m.renderTimingBar("Wait", m.entry.Wait, m.entry.Time, timingBarWidth, m.theme.GetStyle("waterfall_wait")))
	sb.WriteString(m.renderTimingBar("Receive", m.entry.Receive, m.entry.Time, timingBarWidth, m.theme.GetStyle("waterfall_receive")))

	sb.WriteString("\n")
	sb.WriteString(fmt.Sprintf("  Total:   %.1fms\n", m.entry.Time))

	sb.WriteString("\n")

	if m.entry.Priority != "" {
		sb.WriteString(titleStyle.Render("Priority"))
		sb.WriteString("\n\n")
		sb.WriteString(fmt.Sprintf("  Priority:  %s\n", m.entry.Priority))
		if m.entry.InitialPriority != "" && m.entry.InitialPriority != m.entry.Priority {
			sb.WriteString(fmt.Sprintf("  Initial:   %s\n", m.entry.InitialPriority))
		}
		sb.WriteString("\n")
	}

	sb.WriteString(titleStyle.Render("Cache"))
	sb.WriteString("\n\n")

	warnStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(m.theme.GetWarn())).
		Bold(true)
	goodStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(m.theme.GetFg()))

	verdict, desc := har.AnalyzeCache(m.entry)

	switch verdict {
	case har.CacheGood:
		sb.WriteString("  Status:    ")
		sb.WriteString(goodStyle.Render("✓ " + desc))
		sb.WriteString("\n")
	case har.CacheShort:
		sb.WriteString("  Status:    ")
		sb.WriteString(warnStyle.Render("⚠ " + desc))
		sb.WriteString("\n")
	case har.CacheOverkill:
		sb.WriteString("  Status:    ")
		sb.WriteString(warnStyle.Render("⚠ " + desc))
		sb.WriteString("\n")
	case har.CacheNone:
		sb.WriteString("  Status:    ")
		sb.WriteString(warnStyle.Render("⚠ " + desc))
		sb.WriteString("\n")
	}

	if m.entry.CacheControl != "" {
		sb.WriteString(fmt.Sprintf("  Cache-Control: %s\n", m.entry.CacheControl))
		if m.entry.NoStore {
			sb.WriteString("    ⚠ no-store prevents all caching\n")
		}
		if m.entry.NoCache {
			sb.WriteString("    ℹ no-cache requires revalidation\n")
		}
		if m.entry.MaxAge >= 0 {
			if m.entry.MaxAge < 60 {
				sb.WriteString(warnStyle.Render(fmt.Sprintf("    ⚠ max-age=%ds is very short", m.entry.MaxAge)))
				sb.WriteString("\n")
			} else if m.entry.MaxAge > 31536000 {
				sb.WriteString(warnStyle.Render(fmt.Sprintf("    ⚠ max-age=%ds is over 1 year", m.entry.MaxAge)))
				sb.WriteString("\n")
			}
		}
		if m.entry.MustRevalidate {
			sb.WriteString("    ℹ must-revalidate set\n")
		}
	} else {
		sb.WriteString("  Cache-Control: (not set)\n")
	}

	if m.entry.ETag != "" {
		sb.WriteString(fmt.Sprintf("  ETag:         %s\n", m.entry.ETag))
	}

	if m.entry.LastModified != "" {
		sb.WriteString(fmt.Sprintf("  Last-Modified: %s\n", m.entry.LastModified))
	}

	if m.entry.Expires != "" {
		sb.WriteString(fmt.Sprintf("  Expires:      %s\n", m.entry.Expires))
	}

	return sb.String()
}

func (m DetailModel) renderHeaders(title string, headers []har.Header) string {
	if len(headers) == 0 {
		return ""
	}

	var sb strings.Builder
	titleStyle := m.theme.GetStyle("title")
	sb.WriteString(titleStyle.Render(title))
	sb.WriteString("\n\n")

	nameWidth := 22
	avail := m.viewport.Width() - nameWidth - 4
	if avail < 10 {
		avail = 10
	}
	for _, h := range headers {
		value := h.Value
		if utf8.RuneCountInString(value) > avail {
			runes := []rune(value)
			value = string(runes[:avail])
		}
		sb.WriteString(fmt.Sprintf("  %-*s %s\n", nameWidth, h.Name+":", value))
	}
	sb.WriteString("\n")
	return sb.String()
}

func (m DetailModel) Init() tea.Cmd {
	return nil
}

func (m DetailModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		boxW := msg.Width*2/3 - 8
		if boxW < 20 {
			boxW = 20
		}
		boxH := msg.Height*2/3 - 4
		if boxH < 6 {
			boxH = 6
		}
		m.viewport.SetWidth(boxW)
		m.viewport.SetHeight(boxH)
		m.refreshContent()
	case tea.KeyPressMsg:
		var cmd tea.Cmd
		m.viewport, cmd = m.viewport.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m DetailModel) View() tea.View {
	if m.entry == nil && m.customContent == "" {
		return tea.NewView("")
	}
	return tea.NewView(m.renderWithScrollbar(m.buildContent()))
}

func (m DetailModel) renderWithScrollbar(rawContent string) string {
	content := m.viewport.View()
	lines := strings.Split(content, "\n")
	totalLines := len(strings.Split(rawContent, "\n"))
	visibleLines := m.viewport.Height()

	scrollbar := m.buildDetailScrollbar(totalLines, visibleLines, m.viewport.YOffset())

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

func (m DetailModel) buildDetailScrollbar(total, visible, offset int) []string {
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
		barPos = 1 + (offset * (scrollbarHeight - 2)) / maxScroll
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

func (m DetailModel) renderTimingBar(label string, value float64, total float64, barWidth int, style lipgloss.Style) string {
	bar := ""
	if total > 0 && value > 0 {
		fillWidth := int((value / total) * float64(barWidth))
		if fillWidth < 1 && value > 0 {
			fillWidth = 1
		}
		bar = strings.Repeat(m.graphSymbol, fillWidth)
	}

	ms := strconv.FormatFloat(value, 'f', 2, 64) + "ms"
	return fmt.Sprintf("  %8s: %12s %s\n", label, ms, style.Render(bar))
}

func getStatusStyle(th *theme.Engine, status int) lipgloss.Style {
	switch {
	case status >= 200 && status < 300:
		return th.GetStyle("status_2xx")
	case status >= 300 && status < 400:
		return th.GetStyle("status_3xx")
	case status >= 400 && status < 500:
		return th.GetStyle("status_4xx")
	case status >= 500:
		return th.GetStyle("status_5xx")
	default:
		return lipgloss.NewStyle()
	}
}

func formatDetailSize(b int64) string {
	switch {
	case b >= 1024*1024:
		return fmt.Sprintf("%.1f MB", float64(b)/(1024*1024))
	case b >= 1024:
		return fmt.Sprintf("%.1f KB", float64(b)/1024)
	default:
		return fmt.Sprintf("%d B", b)
	}
}
