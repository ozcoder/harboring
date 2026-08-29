package modes

import (
	"fmt"
	"sort"
	"strings"

	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/harboring/harboring/internal/har"
	"github.com/harboring/harboring/internal/theme"
)

type SecurityHeader struct {
	Name        string
	Description string
	Required    bool
}

var securityHeaders = []SecurityHeader{
	{Name: "strict-transport-security", Description: "HSTS - Enforces HTTPS", Required: true},
	{Name: "x-content-type-options", Description: "Prevents MIME-sniffing", Required: true},
	{Name: "content-security-policy", Description: "CSP - Mitigates XSS", Required: true},
	{Name: "x-frame-options", Description: "Clickjacking protection", Required: true},
	{Name: "referrer-policy", Description: "Controls referrer information", Required: false},
	{Name: "permissions-policy", Description: "Controls browser features", Required: false},
	{Name: "cross-origin-opener-policy", Description: "Isolates browsing context", Required: false},
	{Name: "cross-origin-resource-policy", Description: "Controls cross-origin resources", Required: false},
}

type SecurityResult struct {
	Domain          string
	URL             string
	Present         map[string]bool
	MissingRequired int
	MissingOptional int
}

type SecurityModel struct {
	entries  []*har.EntryDetails
	theme    *theme.Engine
	results  []SecurityResult
	width    int
	height   int
	cursor   int
	scrollY  int
	viewport viewport.Model
}

func NewSecurityModel(entries []*har.EntryDetails, th *theme.Engine) SecurityModel {
	vp := viewport.New(viewport.WithWidth(80), viewport.WithHeight(20))
	m := SecurityModel{
		entries:  entries,
		theme:    th,
		width:    80,
		height:   20,
		viewport: vp,
	}
	m.compute()
	return m
}

func (m *SecurityModel) compute() {
	domainMap := make(map[string][]*har.EntryDetails)
	for _, e := range m.entries {
		if e.Domain != "" {
			domainMap[e.Domain] = append(domainMap[e.Domain], e)
		}
	}

	m.results = nil
	for domain, entries := range domainMap {
		present := make(map[string]bool)
		for _, h := range securityHeaders {
			present[h.Name] = false
		}

		for _, e := range entries {
			for _, hdr := range e.ResponseHeaders {
				lower := strings.ToLower(hdr.Name)
				for _, sh := range securityHeaders {
					if lower == sh.Name {
						present[sh.Name] = true
					}
				}
			}
		}

		missingRequired := 0
		missingOptional := 0
		for _, sh := range securityHeaders {
			if !present[sh.Name] {
				if sh.Required {
					missingRequired++
				} else {
					missingOptional++
				}
			}
		}

		m.results = append(m.results, SecurityResult{
			Domain:          domain,
			URL:             entries[0].URL,
			Present:         present,
			MissingRequired: missingRequired,
			MissingOptional: missingOptional,
		})
	}

	sort.Slice(m.results, func(i, j int) bool {
		if m.results[i].MissingRequired != m.results[j].MissingRequired {
			return m.results[i].MissingRequired > m.results[j].MissingRequired
		}
		return m.results[i].Domain < m.results[j].Domain
	})

	if m.cursor >= len(m.results) {
		m.cursor = len(m.results) - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
}

func (m *SecurityModel) SetViewportSize(w, h int) {
	m.width = w
	m.height = h
	m.viewport.SetWidth(w)
	m.viewport.SetHeight(h)
}

func (m SecurityModel) Update(msg tea.Msg) (SecurityModel, tea.Cmd) {
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

func (m SecurityModel) handleKey(msg tea.KeyPressMsg) (SecurityModel, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < len(m.results)-1 {
			m.cursor++
		}
	case "pgdown", "space", "f":
		m.cursor += m.viewport.Height()
		if m.cursor >= len(m.results) {
			m.cursor = len(m.results) - 1
		}
	case "pgup", "b":
		m.cursor -= m.viewport.Height()
		if m.cursor < 0 {
			m.cursor = 0
		}
	case "g":
		m.cursor = 0
	case "G":
		m.cursor = len(m.results) - 1
	case "enter":
		if len(m.results) > 0 && m.cursor < len(m.results) {
			return m, func() tea.Msg {
				return SecurityDetailMsg{Result: m.results[m.cursor]}
			}
		}
	}
	m.updateScrollY()
	return m, nil
}

func (m *SecurityModel) updateScrollY() {
	totalLines := len(m.results) + 3
	cursorLine := m.cursor + 3
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

func padCell(s string, width int) string {
	pad := width - lipgloss.Width(s)
	if pad < 0 {
		pad = 0
	}
	return s + strings.Repeat(" ", pad)
}

func (m SecurityModel) View() string {
	if len(m.entries) == 0 {
		return "No entries"
	}

	titleStyle := m.theme.GetStyle("title")
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(m.theme.GetFg()))
	presentStyle := m.theme.GetStyle("status_2xx")
	missingStyle := m.theme.GetStyle("status_4xx")
	missingOptionalStyle := m.theme.GetStyle("status_3xx")

	header := fmt.Sprintf("%-*s %-*s %-*s %s",
		30, "Domain",
		8, "Required",
		8, "Optional",
		"Details")

	var allLines []string
	allLines = append(allLines, titleStyle.Render(" Security Headers Analysis "))
	allLines = append(allLines, "")
	allLines = append(allLines, headerStyle.Render(header))
	allLines = append(allLines, strings.Repeat("─", m.width))

	for i, r := range m.results {
		domainDisplay := r.Domain
		if lipgloss.Width(domainDisplay) > 28 {
			domainDisplay = truncateWidth(domainDisplay, 28)
		}

		reqStr := fmt.Sprintf("%d missing", r.MissingRequired)
		optStr := fmt.Sprintf("%d missing", r.MissingOptional)

		isSelected := i == m.cursor
		var domainStyle, reqStyle, optStyle lipgloss.Style

		if isSelected {
			domainStyle = lipgloss.NewStyle().
				Background(lipgloss.Color(m.theme.GetSelectionBg())).
				Foreground(lipgloss.Color(m.theme.GetFg()))
			reqStyle = domainStyle
			optStyle = domainStyle
		} else {
			domainStyle = lipgloss.NewStyle()
			if r.MissingRequired > 0 {
				reqStyle = missingStyle
			} else {
				reqStyle = presentStyle
			}
			if r.MissingOptional > 0 {
				optStyle = missingOptionalStyle
			} else {
				optStyle = presentStyle
			}
		}

		line := padCell(domainStyle.Render(domainDisplay), 30) + " " +
			padCell(reqStyle.Render(reqStr), 8) + " " +
			padCell(optStyle.Render(optStr), 8)

		if isSelected {
			line = domainStyle.Render(line)
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

type SecurityDetailMsg struct {
	Result SecurityResult
}

func RenderSecurityDetail(result SecurityResult, th *theme.Engine, width int) string {
	titleStyle := th.GetStyle("title")
	presentStyle := th.GetStyle("status_2xx")
	missingStyle := th.GetStyle("status_4xx")
	missingOptionalStyle := th.GetStyle("status_3xx")
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(th.GetFg())).Faint(true)

	var lines []string
	lines = append(lines, titleStyle.Render(fmt.Sprintf(" Security Headers: %s ", result.Domain)))
	lines = append(lines, "")
	lines = append(lines, dimStyle.Render(fmt.Sprintf(" URL: %s", result.URL)))
	lines = append(lines, "")

	for _, sh := range securityHeaders {
		isPresent := result.Present[sh.Name]
		var status string
		var style lipgloss.Style
		if isPresent {
			status = "✓ Present"
			style = presentStyle
		} else if sh.Required {
			status = "✗ Missing (Required)"
			style = missingStyle
		} else {
			status = "○ Missing (Optional)"
			style = missingOptionalStyle
		}
		headerName := titleHeader(sh.Name)
		lines = append(lines, fmt.Sprintf("  %-40s %s", headerName+":", style.Render(status)))
		lines = append(lines, fmt.Sprintf("     %s", dimStyle.Render(sh.Description)))
		lines = append(lines, "")
	}

	return strings.Join(lines, "\n")
}

// titleHeader capitalises the first letter of each dash-separated segment,
// e.g. "cross-origin-opener-policy" -> "Cross-Origin-Opener-Policy".
func titleHeader(name string) string {
	parts := strings.Split(name, "-")
	for i, p := range parts {
		if p == "" {
			continue
		}
		parts[i] = strings.ToUpper(p[:1]) + p[1:]
	}
	return strings.Join(parts, "-")
}
