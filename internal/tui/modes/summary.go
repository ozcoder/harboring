package modes

import (
	"fmt"
	"sort"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/harboring/harboring/internal/har"
	"github.com/harboring/harboring/internal/theme"
)

type SummaryModel struct {
	entries    []*har.EntryDetails
	theme      *theme.Engine
	width      int
	height     int
	totalSize  int64
	totalTime  float64
	statusDist map[int]int
	typeDist   map[string]int
	typeSize   map[string]int64
	domainSize map[string]int64
	domainTime map[string]float64
	domainCnt  map[string]int
	graphSymbol string
}

func NewSummaryModel(entries []*har.EntryDetails, th *theme.Engine) SummaryModel {
	m := SummaryModel{
		entries:    entries,
		theme:      th,
		width:      80,
		height:     24,
		statusDist: make(map[int]int),
		typeDist:   make(map[string]int),
		typeSize:   make(map[string]int64),
		domainSize: make(map[string]int64),
		domainTime: make(map[string]float64),
		domainCnt:  make(map[string]int),
		graphSymbol: "\u2588",
	}
	m.compute()
	return m
}

func (m *SummaryModel) SetGraphSymbol(sym string) {
	switch sym {
	case "braille":
		m.graphSymbol = "\u28ff"
	case "ascii":
		m.graphSymbol = "#"
	default:
		m.graphSymbol = "\u2588"
	}
}

func (m *SummaryModel) compute() {
	m.totalSize = 0
	m.totalTime = 0
	m.statusDist = make(map[int]int)
	m.typeDist = make(map[string]int)
	m.typeSize = make(map[string]int64)
	m.domainSize = make(map[string]int64)
	m.domainTime = make(map[string]float64)
	m.domainCnt = make(map[string]int)

	for _, e := range m.entries {
		m.totalSize += e.Size
		m.totalTime += e.Time

		bucket := (e.Status / 100) * 100
		m.statusDist[bucket]++

		rt := e.ResourceType
		if rt == "" {
			rt = classifyByMime(e.MimeType)
		}
		if rt == "" {
			rt = "other"
		}
		m.typeDist[rt]++
		m.typeSize[rt] += e.Size

		if e.Domain != "" {
			m.domainSize[e.Domain] += e.Size
			m.domainTime[e.Domain] += e.Time
			m.domainCnt[e.Domain]++
		}
	}
}

func classifyByMime(mime string) string {
	if mime == "" {
		return ""
	}
	m := strings.ToLower(mime)
	switch {
	case strings.HasPrefix(m, "text/html"):
		return "document"
	case strings.HasPrefix(m, "text/css"):
		return "stylesheet"
	case strings.HasPrefix(m, "text/javascript"), strings.HasPrefix(m, "application/javascript"), strings.HasPrefix(m, "application/x-javascript"):
		return "script"
	case strings.HasPrefix(m, "image/"):
		return "image"
	case strings.HasPrefix(m, "font/"), strings.Contains(m, "font"):
		return "font"
	case strings.Contains(m, "json"), strings.Contains(m, "xml"):
		return "xhr"
	default:
		return "other"
	}
}

func (m *SummaryModel) SetViewportSize(w, h int) {
	m.width = w
	m.height = h
}

func (m SummaryModel) Update(msg tea.Msg) (SummaryModel, tea.Cmd) {
	return m, nil
}

func (m SummaryModel) View() string {
	if len(m.entries) == 0 {
		return "No entries"
	}

	titleStyle := m.theme.GetStyle("title")
	valueStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(m.theme.GetFg()))
	warnStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(m.theme.GetWarn()))

	halfW := m.width / 2
	thirdH := m.height / 3

	overviewH := thirdH
	statusH := thirdH
	typeH := thirdH
	domainH := thirdH

	overview := m.renderOverview(overviewH, halfW, titleStyle, valueStyle)
	status := m.renderStatusCodes(statusH, halfW, titleStyle, valueStyle, warnStyle)
	types := m.renderResourceTypes(typeH, halfW, titleStyle, valueStyle)
	domains := m.renderDomains(domainH, halfW, titleStyle, valueStyle)

	left := joinVertical(overview, types, m.height)
	right := joinVertical(status, domains, m.height)

	return lipgloss.JoinHorizontal(lipgloss.Top, left, right)
}

func (m SummaryModel) renderOverview(h, w int, titleStyle, valueStyle lipgloss.Style) string {
	var lines []string

	title := titleStyle.Render(" Overview ")
	lines = append(lines, title)
	lines = append(lines, "")

	lines = append(lines, valueStyle.Render(fmt.Sprintf("  Requests:    %d", len(m.entries))))
	lines = append(lines, valueStyle.Render(fmt.Sprintf("  Total Size:  %s", formatSize(m.totalSize))))
	lines = append(lines, valueStyle.Render(fmt.Sprintf("  Total Time:  %s", formatTime(m.totalTime))))

	avgTime := float64(0)
	if len(m.entries) > 0 {
		avgTime = m.totalTime / float64(len(m.entries))
	}
	lines = append(lines, valueStyle.Render(fmt.Sprintf("  Avg Time:    %s", formatTime(avgTime))))

	domains := make(map[string]bool)
	for _, e := range m.entries {
		if e.Domain != "" {
			domains[e.Domain] = true
		}
	}
	lines = append(lines, valueStyle.Render(fmt.Sprintf("  Domains:     %d", len(domains))))

	box := m.buildBox("Overview", lines, w, h)
	return box
}

func (m SummaryModel) renderStatusCodes(h, w int, titleStyle, valueStyle, warnStyle lipgloss.Style) string {
	title := titleStyle.Render(" Status Codes ")

	type bucket struct {
		label string
		count int
		code  int
	}
	buckets := []bucket{
		{label: "2xx Success", code: 200},
		{label: "3xx Redirect", code: 300},
		{label: "4xx Client Err", code: 400},
		{label: "5xx Server Err", code: 500},
	}

	var lines []string
	lines = append(lines, title)
	lines = append(lines, "")

	maxCount := 0
	for _, b := range buckets {
		c := m.statusDist[b.code]
		if c > maxCount {
			maxCount = c
		}
	}

	inner := w - 8

	for _, b := range buckets {
		c := m.statusDist[b.code]
		countStr := fmt.Sprintf("%d", c)
		barWidth := inner - (4 + 14 + 1 + len(countStr))
		if barWidth < 0 {
			barWidth = 0
		}
		barLen := 0
		if maxCount > 0 {
			barLen = (c * barWidth) / maxCount
		}
		bar := strings.Repeat(m.graphSymbol, barLen)

		style := valueStyle
		if b.code >= 400 && c > 0 {
			style = warnStyle
		}
		line := fmt.Sprintf("  %-14s %s %s", b.label, style.Render(bar), countStr)
		lines = append(lines, line)
	}

	box := m.buildBox("Status Codes", lines, w, h)
	return box
}

func (m SummaryModel) renderResourceTypes(h, w int, titleStyle, valueStyle lipgloss.Style) string {
	type typeInfo struct {
		name  string
		count int
		size  int64
	}
	var types []typeInfo
	for name, count := range m.typeDist {
		types = append(types, typeInfo{name: name, count: count, size: m.typeSize[name]})
	}
	sort.Slice(types, func(i, j int) bool {
		return types[i].count > types[j].count
	})

	var lines []string
	lines = append(lines, titleStyle.Render(" Resource Types "))
	lines = append(lines, "")

	inner := w - 8

	maxCount := 0
	for _, t := range types {
		if t.count > maxCount {
			maxCount = t.count
		}
	}

	for _, t := range types {
		name := t.name
		if len(name) > 12 {
			name = name[:12]
		}
		countStr := fmt.Sprintf("%d", t.count)
		sizeStr := formatSize(t.size)
		barWidth := inner - (4 + 12 + 1 + len(countStr) + 3 + len(sizeStr))
		if barWidth < 0 {
			barWidth = 0
		}
		barLen := 0
		if maxCount > 0 {
			barLen = (t.count * barWidth) / maxCount
		}
		bar := strings.Repeat(m.graphSymbol, barLen)
		line := fmt.Sprintf("  %-12s %s %d (%s)", name, valueStyle.Render(bar), t.count, sizeStr)
		lines = append(lines, line)
	}

	box := m.buildBox("Resource Types", lines, w, h)
	return box
}

func (m SummaryModel) renderDomains(h, w int, titleStyle, valueStyle lipgloss.Style) string {
	type domainInfo struct {
		name  string
		size  int64
		time  float64
		count int
	}
	var domains []domainInfo
	for name, size := range m.domainSize {
		domains = append(domains, domainInfo{
			name:  name,
			size:  size,
			time:  m.domainTime[name],
			count: m.domainCnt[name],
		})
	}
	sort.Slice(domains, func(i, j int) bool {
		return domains[i].size > domains[j].size
	})

	var lines []string
	lines = append(lines, titleStyle.Render(" Top Domains "))
	lines = append(lines, "")

	inner := w - 8

	maxSize := int64(0)
	for _, d := range domains {
		if d.size > maxSize {
			maxSize = d.size
		}
	}

	limit := 8
	if len(domains) < limit {
		limit = len(domains)
	}

	for _, d := range domains[:limit] {
		name := d.name
		if len(name) > 20 {
			name = name[:20]
		}
		sizeStr := formatSize(d.size)
		barWidth := inner - (4 + 20 + 1 + len(sizeStr))
		if barWidth < 0 {
			barWidth = 0
		}
		barLen := 0
		if maxSize > 0 {
			barLen = int((d.size * int64(barWidth)) / maxSize)
		}
		bar := strings.Repeat(m.graphSymbol, barLen)
		line := fmt.Sprintf("  %-20s %s %s", name, valueStyle.Render(bar), sizeStr)
		lines = append(lines, line)
	}

	box := m.buildBox("Top Domains", lines, w, h)
	return box
}

func (m SummaryModel) buildBox(title string, lines []string, w, h int) string {
	content := strings.Join(lines, "\n")

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(m.theme.GetFg())).
		Width(w - 4).
		Height(h - 2).
		Padding(0, 1)

	return boxStyle.Render(content)
}

func joinVertical(top, bottom string, totalH int) string {
	topLines := strings.Split(top, "\n")
	bottomLines := strings.Split(bottom, "\n")

	topH := totalH / 2
	bottomH := totalH - topH

	result := make([]string, 0, totalH)
	for i := 0; i < topH && i < len(topLines); i++ {
		result = append(result, topLines[i])
	}
	for len(result) < topH {
		result = append(result, "")
	}
	for i := 0; i < bottomH && i < len(bottomLines); i++ {
		result = append(result, bottomLines[i])
	}
	for len(result) < totalH {
		result = append(result, "")
	}

	return strings.Join(result, "\n")
}
