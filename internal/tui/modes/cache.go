package modes

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/harboring/harboring/internal/har"
	"github.com/harboring/harboring/internal/theme"
)

type CacheSection int

const (
	CacheSectionOverview CacheSection = iota
	CacheSectionIssues
	CacheSectionGood
	CacheSectionOver
)

type CacheModel struct {
	entries      []*har.EntryDetails
	rawEntries   []*har.Node
	theme        *theme.Engine
	width        int
	height       int
	noCache      []*har.EntryDetails
	shortCache   []*har.EntryDetails
	goodCache    []*har.EntryDetails
	overCache    []*har.EntryDetails
	noCacheIdx   []int
	shortIdx     []int
	goodIdx      []int
	overIdx      []int
	verdicts     map[int]har.CacheVerdict
	descriptions map[int]string
	active       CacheSection
	scrollPos    map[CacheSection]int
	selection    map[CacheSection]int
}

func NewCacheModel(entries []*har.EntryDetails, rawEntries []*har.Node, th *theme.Engine) CacheModel {
	m := CacheModel{
		entries:      entries,
		rawEntries:   rawEntries,
		theme:        th,
		width:        80,
		height:       24,
		noCache:      make([]*har.EntryDetails, 0),
		shortCache:   make([]*har.EntryDetails, 0),
		goodCache:    make([]*har.EntryDetails, 0),
		overCache:    make([]*har.EntryDetails, 0),
		noCacheIdx:   make([]int, 0),
		shortIdx:     make([]int, 0),
		goodIdx:      make([]int, 0),
		overIdx:      make([]int, 0),
		verdicts:     make(map[int]har.CacheVerdict),
		descriptions: make(map[int]string),
		active:       CacheSectionOverview,
		scrollPos:    make(map[CacheSection]int),
		selection:    make(map[CacheSection]int),
	}
	m.compute()
	return m
}

func (m *CacheModel) compute() {
	m.noCache = m.noCache[:0]
	m.shortCache = m.shortCache[:0]
	m.goodCache = m.goodCache[:0]
	m.overCache = m.overCache[:0]
	m.noCacheIdx = m.noCacheIdx[:0]
	m.shortIdx = m.shortIdx[:0]
	m.goodIdx = m.goodIdx[:0]
	m.overIdx = m.overIdx[:0]

	for i, e := range m.entries {
		verdict, desc := har.AnalyzeCache(e)
		m.verdicts[i] = verdict
		m.descriptions[i] = desc

		switch verdict {
		case har.CacheNone:
			m.noCache = append(m.noCache, e)
			m.noCacheIdx = append(m.noCacheIdx, i)
		case har.CacheShort:
			m.shortCache = append(m.shortCache, e)
			m.shortIdx = append(m.shortIdx, i)
		case har.CacheGood:
			m.goodCache = append(m.goodCache, e)
			m.goodIdx = append(m.goodIdx, i)
		case har.CacheOverkill:
			m.overCache = append(m.overCache, e)
			m.overIdx = append(m.overIdx, i)
		}
	}
}

func (m CacheModel) sectionEntries(s CacheSection) []*har.EntryDetails {
	switch s {
	case CacheSectionIssues:
		var all []*har.EntryDetails
		all = append(all, m.noCache...)
		all = append(all, m.shortCache...)
		return all
	case CacheSectionGood:
		return m.goodCache
	case CacheSectionOver:
		return m.overCache
	}
	return nil
}

func (m CacheModel) sectionRawIdx(s CacheSection, listIdx int) int {
	switch s {
	case CacheSectionIssues:
		if listIdx < len(m.noCache) {
			return m.noCacheIdx[listIdx]
		}
		return m.shortIdx[listIdx-len(m.noCache)]
	case CacheSectionGood:
		return m.goodIdx[listIdx]
	case CacheSectionOver:
		return m.overIdx[listIdx]
	}
	return -1
}

func (m CacheModel) Selected() *har.EntryDetails {
	entries := m.sectionEntries(m.active)
	sel := m.selection[m.active]
	if sel < 0 || sel >= len(entries) {
		return nil
	}
	return entries[sel]
}

func (m CacheModel) SelectedRaw() *har.Node {
	entries := m.sectionEntries(m.active)
	sel := m.selection[m.active]
	if sel < 0 || sel >= len(entries) {
		return nil
	}
	ri := m.sectionRawIdx(m.active, sel)
	if ri < 0 || ri >= len(m.rawEntries) {
		return nil
	}
	return m.rawEntries[ri]
}

func (m *CacheModel) SetViewportSize(w, h int) {
	m.width = w
	m.height = h
}

func (m CacheModel) Update(msg tea.Msg) (CacheModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

func (m CacheModel) isScrollableSection() bool {
	return m.active == CacheSectionIssues || m.active == CacheSectionGood || m.active == CacheSectionOver
}

func (m CacheModel) handleKey(msg tea.KeyPressMsg) (CacheModel, tea.Cmd) {
	switch msg.String() {
	case "tab":
		m.active = (m.active + 1) % 4
		if m.selection[m.active] < 0 {
			m.selection[m.active] = 0
		}
		return m, nil
	case "shift+tab":
		m.active = (m.active + 3) % 4
		if m.selection[m.active] < 0 {
			m.selection[m.active] = 0
		}
		return m, nil
	case "j", "down":
		if m.isScrollableSection() {
			entries := m.sectionEntries(m.active)
			sel := m.selection[m.active] + 1
			if sel < len(entries) {
				m.selection[m.active] = sel
			}
			m.ensureSelectionVisible()
		}
	case "k", "up":
		if m.isScrollableSection() {
			sel := m.selection[m.active] - 1
			if sel >= 0 {
				m.selection[m.active] = sel
			}
			m.ensureSelectionVisible()
		}
	case "pgdown", "space", "f":
		if m.isScrollableSection() {
			entries := m.sectionEntries(m.active)
			sel := m.selection[m.active] + m.visibleLines()
			if sel >= len(entries) {
				sel = len(entries) - 1
			}
			m.selection[m.active] = sel
			m.ensureSelectionVisible()
		}
	case "pgup", "b":
		if m.isScrollableSection() {
			sel := m.selection[m.active] - m.visibleLines()
			if sel < 0 {
				sel = 0
			}
			m.selection[m.active] = sel
			m.ensureSelectionVisible()
		}
	case "g":
		if m.isScrollableSection() {
			m.selection[m.active] = 0
			m.scrollPos[m.active] = 0
		}
	case "G":
		if m.isScrollableSection() {
			entries := m.sectionEntries(m.active)
			if len(entries) > 0 {
				m.selection[m.active] = len(entries) - 1
			}
			m.scrollPos[m.active] = m.maxScrollForSection(m.active)
		}
	}
	return m, nil
}

func (m *CacheModel) ensureSelectionVisible() {
	sel := m.selection[m.active]
	scrollY := m.scrollPos[m.active]
	visible := m.visibleLines()

	if sel < scrollY {
		m.scrollPos[m.active] = sel
	} else if sel >= scrollY+visible {
		m.scrollPos[m.active] = sel - visible + 1
	}
}

func (m CacheModel) boxHeight() int {
	return m.height / 2
}

func (m CacheModel) visibleLines() int {
	h := m.boxHeight() - 6
	if h < 1 {
		return 1
	}
	return h
}

func (m CacheModel) maxScrollForSection(s CacheSection) int {
	var total int
	switch s {
	case CacheSectionIssues:
		total = len(m.noCache) + len(m.shortCache)
	case CacheSectionGood:
		total = len(m.goodCache)
	case CacheSectionOver:
		total = len(m.overCache)
	}
	visible := m.visibleLines()
	if total <= visible {
		return 0
	}
	max := total - visible
	if m.scrollPos[s] > max {
		m.scrollPos[s] = max
	}
	return max
}

func (m CacheModel) View() string {
	if len(m.entries) == 0 {
		return "No entries"
	}

	titleStyle := m.theme.GetStyle("title")
	valueStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(m.theme.GetFg()))
	warnStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(m.theme.GetWarn()))
	goodStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(m.theme.GetFg()))

	halfW := m.width / 2
	boxH := m.boxHeight()

	overview := m.renderCacheOverview(halfW, boxH, titleStyle, valueStyle)
	issues := m.renderCacheIssues(halfW, boxH, titleStyle, valueStyle, warnStyle)
	good := m.renderGoodCache(halfW, boxH, titleStyle, valueStyle, goodStyle)
	details := m.renderCacheDetails(halfW, boxH, titleStyle, valueStyle, warnStyle)

	left := joinVertical2(overview, issues, m.height)
	right := joinVertical2(good, details, m.height)

	return lipgloss.JoinHorizontal(lipgloss.Top, left, right)
}

func (m CacheModel) renderCacheOverview(w, h int, titleStyle, valueStyle lipgloss.Style) string {
	var lines []string
	lines = append(lines, titleStyle.Render(" Cache Overview "))
	lines = append(lines, "")

	total := len(m.entries)
	withCache := len(m.goodCache) + len(m.overCache)
	pctGood := 0
	if total > 0 {
		pctGood = (len(m.goodCache) * 100) / total
	}

	lines = append(lines, valueStyle.Render(fmt.Sprintf("  Total Resources:   %d", total)))
	lines = append(lines, valueStyle.Render(fmt.Sprintf("  Good Cache:        %d (%d%%)", len(m.goodCache), pctGood)))
	lines = append(lines, valueStyle.Render(fmt.Sprintf("  Short Cache:       %d", len(m.shortCache))))
	lines = append(lines, valueStyle.Render(fmt.Sprintf("  No Cache:          %d", len(m.noCache))))
	lines = append(lines, valueStyle.Render(fmt.Sprintf("  Over-Cached:       %d", len(m.overCache))))
	lines = append(lines, valueStyle.Render(fmt.Sprintf("  With Any Caching:  %d (%d%%)", withCache, (withCache*100)/max(1, total))))

	box := m.buildBox("Overview", lines, w, h, m.active == CacheSectionOverview)
	return box
}

func (m CacheModel) renderCacheIssues(w, h int, titleStyle, valueStyle, warnStyle lipgloss.Style) string {
	var lines []string
	lines = append(lines, titleStyle.Render(" Cache Issues "))
	lines = append(lines, "")

	issues := m.allIssues()

	if len(issues) == 0 {
		lines = append(lines, valueStyle.Render("  No cache issues found"))
		box := m.buildBox("Issues", lines, w, h, m.active == CacheSectionIssues)
		return box
	}

	scrollY := m.scrollPos[CacheSectionIssues]
	visible := m.visibleLines()
	total := len(issues)

	if scrollY > total-visible {
		scrollY = total - visible
		if scrollY < 0 {
			scrollY = 0
		}
		m.scrollPos[CacheSectionIssues] = scrollY
	}

	start := scrollY
	end := start + visible
	if end > total {
		end = total
	}

	scrollChars := m.buildScrollChars(total, start, visible)
	selPos := m.selection[CacheSectionIssues]

	for i, e := range issues[start:end] {
		cacheInfo := ""
		if e.MaxAge >= 0 {
			cacheInfo = fmt.Sprintf("max-age=%ds", e.MaxAge)
		} else if !e.HasCacheHeaders {
			cacheInfo = "no headers"
		}
		sc := " "
		if i < len(scrollChars) {
			sc = scrollChars[i]
		}
		listIdx := start + i
		line := fmt.Sprintf("%s %s %s %s", sc, e.Method, truncateStr(e.URLPath, 22), cacheInfo)
		if listIdx == selPos {
			selStyle := lipgloss.NewStyle().
				Background(lipgloss.Color(m.theme.GetSelectionBg())).
				Foreground(lipgloss.Color(m.theme.GetWarn()))
			line = selStyle.Render(line)
		} else {
			line = warnStyle.Render(line)
		}
		lines = append(lines, line)
	}

	box := m.buildBox("Issues", lines, w, h, m.active == CacheSectionIssues)
	return box
}

func (m CacheModel) allIssues() []*har.EntryDetails {
	var all []*har.EntryDetails
	all = append(all, m.noCache...)
	all = append(all, m.shortCache...)
	return all
}

func (m CacheModel) renderGoodCache(w, h int, titleStyle, valueStyle, goodStyle lipgloss.Style) string {
	var lines []string
	lines = append(lines, titleStyle.Render(" Good Cache "))
	lines = append(lines, "")

	if len(m.goodCache) == 0 {
		lines = append(lines, valueStyle.Render("  No well-cached resources"))
		box := m.buildBox("Good Cache", lines, w, h, m.active == CacheSectionGood)
		return box
	}

	scrollY := m.scrollPos[CacheSectionGood]
	visible := m.visibleLines()
	total := len(m.goodCache)

	if scrollY > total-visible {
		scrollY = total - visible
		if scrollY < 0 {
			scrollY = 0
		}
		m.scrollPos[CacheSectionGood] = scrollY
	}

	start := scrollY
	end := start + visible
	if end > total {
		end = total
	}

	scrollChars := m.buildScrollChars(total, start, visible)
	selPos := m.selection[CacheSectionGood]

	for i, e := range m.goodCache[start:end] {
		cacheInfo := ""
		if e.MaxAge > 0 {
			cacheInfo = fmt.Sprintf("max-age=%ds", e.MaxAge)
		} else if e.ETag != "" {
			cacheInfo = "ETag"
		} else if e.LastModified != "" {
			cacheInfo = "Last-Modified"
		}
		sc := " "
		if i < len(scrollChars) {
			sc = scrollChars[i]
		}
		listIdx := start + i
		line := fmt.Sprintf("%s %s %s %s", sc, e.Method, truncateStr(e.URLPath, 22), cacheInfo)
		if listIdx == selPos {
			selStyle := lipgloss.NewStyle().
				Background(lipgloss.Color(m.theme.GetSelectionBg())).
				Foreground(lipgloss.Color(m.theme.GetFg()))
			line = selStyle.Render(line)
		} else {
			line = goodStyle.Render(line)
		}
		lines = append(lines, line)
	}

	box := m.buildBox("Good Cache", lines, w, h, m.active == CacheSectionGood)
	return box
}

func (m CacheModel) buildScrollChars(total, start, visible int) []string {
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
		barPos = 1 + (start*(scrollbarHeight-2))/maxScroll
	}
	if barPos > scrollbarHeight-2 {
		barPos = scrollbarHeight - 2
	}

	for i := 0; i < scrollbarHeight; i++ {
		if i >= len(result) {
			break
		}
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

func (m CacheModel) renderCacheDetails(w, h int, titleStyle, valueStyle, warnStyle lipgloss.Style) string {
	var lines []string
	lines = append(lines, titleStyle.Render(" Over-Cached "))
	lines = append(lines, "")

	if len(m.overCache) == 0 {
		lines = append(lines, valueStyle.Render("  No over-cached resources"))
		box := m.buildBox("Over-Cached", lines, w, h, m.active == CacheSectionOver)
		return box
	}

	scrollY := m.scrollPos[CacheSectionOver]
	visible := m.visibleLines()
	total := len(m.overCache)

	if scrollY > total-visible {
		scrollY = total - visible
		if scrollY < 0 {
			scrollY = 0
		}
		m.scrollPos[CacheSectionOver] = scrollY
	}

	start := scrollY
	end := start + visible
	if end > total {
		end = total
	}

	scrollChars := m.buildScrollChars(total, start, visible)
	selPos := m.selection[CacheSectionOver]

	for i, e := range m.overCache[start:end] {
		sc := " "
		if i < len(scrollChars) {
			sc = scrollChars[i]
		}
		listIdx := start + i
		line := fmt.Sprintf("%s %s %s max-age=%ds", sc, e.Method, truncateStr(e.URLPath, 22), e.MaxAge)
		if listIdx == selPos {
			selStyle := lipgloss.NewStyle().
				Background(lipgloss.Color(m.theme.GetSelectionBg())).
				Foreground(lipgloss.Color(m.theme.GetWarn()))
			line = selStyle.Render(line)
		} else {
			line = warnStyle.Render(line)
		}
		lines = append(lines, line)
	}

	box := m.buildBox("Over-Cached", lines, w, h, m.active == CacheSectionOver)
	return box
}

func (m CacheModel) buildBox(title string, lines []string, w, h int, active bool) string {
	content := strings.Join(lines, "\n")

	borderColor := lipgloss.Color(m.theme.GetFg())
	if active {
		borderColor = lipgloss.Color(m.theme.GetWarn())
	}

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Width(w-4).
		Height(h-2).
		Padding(0, 1)

	return boxStyle.Render(content)
}

func joinVertical2(top, bottom string, h int) string {
	topLines := strings.Split(top, "\n")
	bottomLines := strings.Split(bottom, "\n")

	topH := h / 2
	bottomH := h - topH

	result := make([]string, 0, h)
	for i := 0; i < topH && i < len(topLines); i++ {
		result = append(result, topLines[i])
	}
	for len(result) < topH {
		result = append(result, "")
	}
	for i := 0; i < bottomH && i < len(bottomLines); i++ {
		result = append(result, bottomLines[i])
	}
	for len(result) < h {
		result = append(result, "")
	}

	return strings.Join(result, "\n")
}

// truncateWidth shortens s to at most max display columns, appending "...".
// It operates on rune boundaries so multi-byte characters are never split.
func truncateWidth(s string, max int) string {
	if max <= 0 {
		return ""
	}
	if lipgloss.Width(s) <= max {
		return s
	}
	if max <= 3 {
		return strings.Repeat(".", max)
	}
	var b strings.Builder
	w := 0
	for _, r := range s {
		rw := lipgloss.Width(string(r))
		if w+rw > max-3 {
			break
		}
		b.WriteRune(r)
		w += rw
	}
	return b.String() + "..."
}

func truncateStr(s string, max int) string {
	return truncateWidth(s, max)
}
