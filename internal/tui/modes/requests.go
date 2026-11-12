package modes

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/harboring/harboring/internal/har"
	"github.com/harboring/harboring/internal/theme"
)

type sortField int

const (
	sortIndex sortField = iota
	sortMethod
	sortURL
	sortStatus
	sortSize
	sortTime
)

type RequestsModel struct {
	entries        []*har.EntryDetails
	rawEntries     []*har.Node
	cursor         int
	scrollY        int
	viewport       viewport.Model
	theme          *theme.Engine
	filter         string
	filtered       []int
	width          int
	height         int
	sortField      sortField
	sortAsc        bool
	typeFilterMode bool
	typeCursor     int
	typeSelected   map[string]bool
}

type resourceTypeChip struct {
	key   string
	label string
}

var resourceTypeChips = []resourceTypeChip{
	{"all", "All"},
	{"document", "Doc"},
	{"stylesheet", "CSS"},
	{"script", "JS"},
	{"image", "Img"},
	{"font", "Font"},
	{"xhr", "XHR"},
	{"media", "Media"},
	{"text", "Text"},
	{"other", "Other"},
}

func normalizeResourceType(rt string) string {
	switch rt {
	case "document", "stylesheet", "script", "image", "font", "xhr", "media", "text":
		return rt
	default:
		return "other"
	}
}

func NewRequestsModel(entries []*har.EntryDetails, rawEntries []*har.Node, th *theme.Engine) RequestsModel {
	vp := viewport.New(viewport.WithWidth(80), viewport.WithHeight(20))
	rm := RequestsModel{
		entries:    entries,
		rawEntries: rawEntries,
		cursor:     0,
		viewport:   vp,
		theme:      th,
		width:      80,
		height:     20,
		sortField:  sortIndex,
		sortAsc:    true,
	}
	rm.rebuildFilter()
	return rm
}

func (m *RequestsModel) rebuildFilter() {
	m.filtered = nil
	re, err := regexp.Compile("(?i)" + m.filter)
	if err != nil {
		re = nil
	}
	for i, e := range m.entries {
		if re != nil && !re.MatchString(e.URL) {
			continue
		}
		if !m.typeMatches(e.ResourceType) {
			continue
		}
		m.filtered = append(m.filtered, i)
	}
	m.applySort()
	if m.cursor >= len(m.filtered) {
		m.cursor = len(m.filtered) - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
}

func (m *RequestsModel) typeMatches(rt string) bool {
	if len(m.typeSelected) == 0 {
		return true
	}
	return m.typeSelected[normalizeResourceType(rt)]
}

func (m *RequestsModel) ToggleTypeFilterMode() {
	m.typeFilterMode = !m.typeFilterMode
	if m.typeFilterMode {
		m.typeCursor = 0
	}
}

func (m *RequestsModel) SetTypeFilterMode(active bool) {
	m.typeFilterMode = active
	if active {
		m.typeCursor = 0
	}
}

func (m *RequestsModel) FilterMode() bool {
	return m.typeFilterMode
}

func (m *RequestsModel) MoveTypeCursor(delta int) {
	m.typeCursor += delta
	if m.typeCursor < 0 {
		m.typeCursor = len(resourceTypeChips) - 1
	}
	if m.typeCursor >= len(resourceTypeChips) {
		m.typeCursor = 0
	}
}

func (m *RequestsModel) ToggleTypeAtCursor() {
	rt := resourceTypeChips[m.typeCursor].key
	if rt == "all" {
		m.typeSelected = nil
	} else {
		if m.typeSelected == nil {
			m.typeSelected = map[string]bool{}
		}
		if m.typeSelected[rt] {
			delete(m.typeSelected, rt)
			if len(m.typeSelected) == 0 {
				m.typeSelected = nil
			}
		} else {
			m.typeSelected[rt] = true
		}
	}
	m.rebuildFilter()
}

func (m *RequestsModel) applySort() {
	sort.SliceStable(m.filtered, func(i, j int) bool {
		ai := m.filtered[i]
		aj := m.filtered[j]
		a, b := m.entries[ai], m.entries[aj]
		var less bool
		switch m.sortField {
		case sortIndex:
			less = ai < aj
		case sortMethod:
			less = strings.ToLower(a.Method) < strings.ToLower(b.Method)
			if strings.EqualFold(a.Method, b.Method) {
				less = ai < aj
			}
		case sortURL:
			less = strings.ToLower(a.URL) < strings.ToLower(b.URL)
			if strings.EqualFold(a.URL, b.URL) {
				less = ai < aj
			}
		case sortStatus:
			less = a.Status < b.Status
			if a.Status == b.Status {
				less = ai < aj
			}
		case sortSize:
			less = a.Size < b.Size
			if a.Size == b.Size {
				less = ai < aj
			}
		case sortTime:
			less = a.Time < b.Time
			if a.Time == b.Time {
				less = ai < aj
			}
		default:
			less = ai < aj
		}
		if m.sortAsc {
			return less
		}
		return !less
	})
}

func (m *RequestsModel) CycleSort() {
	m.sortField = (m.sortField + 1) % 6
	switch m.sortField {
	case sortIndex:
		m.sortAsc = true
	case sortMethod, sortURL, sortStatus:
		m.sortAsc = true
	case sortSize, sortTime:
		m.sortAsc = false
	}
	m.applySort()
}

func (m *RequestsModel) ToggleSortDir() {
	if m.sortField >= sortIndex {
		m.sortAsc = !m.sortAsc
		m.applySort()
	}
}

func (m *RequestsModel) SetFilter(filter string) {
	m.filter = filter
	m.rebuildFilter()
}

func (m *RequestsModel) Filter() string {
	return m.filter
}

func (m *RequestsModel) Selected() *har.EntryDetails {
	if len(m.filtered) == 0 || m.cursor >= len(m.filtered) {
		return nil
	}
	return m.entries[m.filtered[m.cursor]]
}

func (m *RequestsModel) SelectedRaw() *har.Node {
	if len(m.filtered) == 0 || m.cursor >= len(m.filtered) {
		return nil
	}
	return m.rawEntries[m.filtered[m.cursor]]
}

func (m *RequestsModel) SetViewportSize(w, h int) {
	m.width = w
	m.height = h
	m.viewport.SetWidth(w)
	m.viewport.SetHeight(h)
}

func (m RequestsModel) Update(msg tea.Msg) (RequestsModel, tea.Cmd) {
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

func (m RequestsModel) handleKey(msg tea.KeyPressMsg) (RequestsModel, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < len(m.filtered)-1 {
			m.cursor++
		}
	case "pgdown", "space", "f":
		m.cursor += m.viewport.Height()
		if m.cursor >= len(m.filtered) {
			m.cursor = len(m.filtered) - 1
		}
	case "pgup", "b":
		m.cursor -= m.viewport.Height()
		if m.cursor < 0 {
			m.cursor = 0
		}
	case "g":
		m.cursor = 0
	case "G":
		m.cursor = len(m.filtered) - 1
	}

	m.updateScrollY()
	return m, nil
}

func (m *RequestsModel) dataHeight() int {
	h := m.viewport.Height() - 3 // filter bar, header, separator are fixed
	if h < 1 {
		h = 1
	}
	return h
}

func (m *RequestsModel) updateScrollY() {
	totalLines := len(m.filtered)
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

func (m RequestsModel) View() string {
	if len(m.entries) == 0 {
		return "No entries"
	}

	methodWidth := 8
	statusWidth := 6
	sizeWidth := 10
	timeWidth := 10
	numWidth := 5
	warnWidth := 2
	urlWidth := m.width - numWidth - warnWidth - methodWidth - statusWidth - sizeWidth - timeWidth - 10
	if urlWidth < 20 {
		urlWidth = 20
	}

	urlLabel := "URL"
	sizeLabel := "Size"
	timeLabel := "Time"

	numLabel := "#"
	if m.sortField == sortIndex {
		if m.sortAsc {
			numLabel = "#▼"
		} else {
			numLabel = "#▲"
		}
	}
	methodLabel := "Method"
	if m.sortField == sortMethod {
		if m.sortAsc {
			methodLabel = "Method▼"
		} else {
			methodLabel = "Method▲"
		}
	}
	if m.sortField == sortURL {
		if m.sortAsc {
			urlLabel = "URL▼"
		} else {
			urlLabel = "URL▲"
		}
	}
	statusLabel := "Status"
	if m.sortField == sortStatus {
		if m.sortAsc {
			statusLabel = "Status▼"
		} else {
			statusLabel = "Status▲"
		}
	}
	if m.sortField == sortSize {
		if m.sortAsc {
			sizeLabel = "Size▼"
		} else {
			sizeLabel = "Size▲"
		}
	}
	if m.sortField == sortTime {
		if m.sortAsc {
			timeLabel = "Time▼"
		} else {
			timeLabel = "Time▲"
		}
	}
	header := fmt.Sprintf(
		"%-*s %-*s %-*s %-*s %-*s %-*s %s",
		numWidth, numLabel,
		methodWidth, methodLabel,
		urlWidth, urlLabel,
		statusWidth, statusLabel,
		sizeWidth, sizeLabel,
		timeWidth, timeLabel,
		"",
	)

	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(m.theme.GetFg()))
	headerLine := headerStyle.Render(header)

	separator := strings.Repeat("─", m.width)

	var fixedLines []string
	fixedLines = append(fixedLines, m.renderTypeFilterBar())
	fixedLines = append(fixedLines, headerLine)
	fixedLines = append(fixedLines, separator)

	var dataLines []string
	for _, idx := range m.filtered {
		e := m.entries[idx]
		isSelected := idx == m.filtered[m.cursor]

		// When the row is selected every column and separator carries the
		// highlight background so it spans the entire row.
		plainStyle := lipgloss.NewStyle()
		if isSelected {
			plainStyle = plainStyle.
				Background(lipgloss.Color(m.theme.GetSelectionBg())).
				Foreground(lipgloss.Color(m.theme.GetFg()))
		}

		methodStyle := m.theme.GetStyle(fmt.Sprintf("method_%s", strings.ToLower(e.Method)))
		statusStyle := getStatusStyle(m.theme, e.Status)
		if isSelected {
			bg := lipgloss.Color(m.theme.GetSelectionBg())
			methodStyle = methodStyle.Background(bg)
			statusStyle = statusStyle.Background(bg)
		}

		numStr := plainStyle.Render(fmt.Sprintf("%-*d", numWidth, idx))
		methodStr := methodStyle.Render(fmt.Sprintf("%-*s", methodWidth, e.Method))

		urlDisplay := e.URLPath
		if lipgloss.Width(urlDisplay) > urlWidth {
			urlDisplay = truncateWidth(urlDisplay, urlWidth)
		}
		urlStr := plainStyle.Render(fmt.Sprintf("%-*s", urlWidth, urlDisplay))

		statusStr := statusStyle.Render(fmt.Sprintf("%-*d", statusWidth, e.Status))

		sizeStr := plainStyle.Render(fmt.Sprintf("%-*s", sizeWidth, formatSize(e.Size)))
		timeStr := plainStyle.Render(fmt.Sprintf("%-*s", timeWidth, formatTime(e.Time)))

		warnStr := plainStyle.Render(padToWidth(" ", warnWidth))
		if e.ShouldBeCompressed && !e.IsCompressed {
			warnStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color(m.theme.GetWarn())).
				Bold(true)
			if isSelected {
				warnStyle = warnStyle.Background(lipgloss.Color(m.theme.GetSelectionBg()))
			}
			warnStr = warnStyle.Render(padToWidth("⚠", warnWidth))
		}

		sep := func() string { return plainStyle.Render(" ") }
		line := numStr + sep() + methodStr + sep() + warnStr + sep() + urlStr + sep() + statusStr + sep() + sizeStr + sep() + timeStr

		if isSelected {
			// Pad to the full row width so the highlight reaches the edge.
			if pad := m.width - 1 - lipgloss.Width(line); pad > 0 {
				line += plainStyle.Render(strings.Repeat(" ", pad))
			}
		}

		dataLines = append(dataLines, line)
	}

	totalLines := len(dataLines)
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
	visibleLines := dataLines[yOff:endLine]
	scrollChars := buildScrollChars(totalLines, yOff, len(visibleLines))

	var result strings.Builder
	for _, line := range fixedLines {
		result.WriteString(" ")
		result.WriteString(line)
		result.WriteString("\n")
	}
	for i, line := range visibleLines {
		sc := " "
		if i < len(scrollChars) {
			sc = scrollChars[i]
		}
		result.WriteString(sc)
		result.WriteString(line)
		result.WriteString("\n")
	}

	content := make([]string, 0, len(fixedLines)+len(visibleLines))
	content = append(content, fixedLines...)
	content = append(content, visibleLines...)
	m.viewport.SetContent(strings.Join(content, "\n"))
	return result.String()
}

func (m RequestsModel) renderTypeFilterBar() string {
	counts := make(map[string]int)
	for _, e := range m.entries {
		counts[normalizeResourceType(e.ResourceType)]++
	}

	var chips []string
	var selected bool
	for i, chip := range resourceTypeChips {
		count := counts[chip.key]
		if chip.key == "all" {
			count = len(m.entries)
		}

		if chip.key == "all" {
			selected = len(m.typeSelected) == 0
		} else {
			selected = m.typeSelected[chip.key]
		}

		style := lipgloss.NewStyle()
		if m.typeFilterMode && i == m.typeCursor {
			style = style.
				Background(lipgloss.Color(m.theme.GetFg())).
				Foreground(lipgloss.Color(m.theme.GetBg()))
		} else if selected {
			style = style.
				Foreground(lipgloss.Color(m.theme.GetFg())).
				Bold(true)
		} else {
			style = style.Foreground(lipgloss.Color(m.theme.GetGray()))
		}

		chips = append(chips, style.Render(fmt.Sprintf("%s:%d", chip.label, count)))
	}

	_ = m.typeFilterMode // used implicitly for rendering
	labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(m.theme.GetFg()))
	boldStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(m.theme.GetFg())).Bold(true)
	renderedLabel := boldStyle.Render("F") + labelStyle.Render("ilter")
	bar := renderedLabel + "  " + strings.Join(chips, "  ")
	if m.filter != "" {
		filterStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(m.theme.GetGray()))
		bar += "  " + filterStyle.Render("Search: ") + filterStyle.Render(m.filter)
	}
	return bar
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

func padToWidth(s string, width int) string {
	w := lipgloss.Width(s)
	if w >= width {
		return s
	}
	return s + strings.Repeat(" ", width-w)
}

func buildScrollChars(total, start, visible int) []string {
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

func formatSize(bytes int64) string {
	switch {
	case bytes >= 1024*1024:
		return fmt.Sprintf("%.1fMB", float64(bytes)/(1024*1024))
	case bytes >= 1024:
		return fmt.Sprintf("%.1fKB", float64(bytes)/1024)
	default:
		return fmt.Sprintf("%dB", bytes)
	}
}

func formatTime(ms float64) string {
	if ms >= 1000 {
		return fmt.Sprintf("%.2fs", ms/1000)
	}
	return fmt.Sprintf("%.0fms", ms)
}
