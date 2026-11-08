package tui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/harboring/harboring/internal/config"
	"github.com/harboring/harboring/internal/har"
	"github.com/harboring/harboring/internal/theme"
	"github.com/harboring/harboring/internal/tui/modes"
)

type Mode int

const (
	ModeTree Mode = iota
	ModeRequests
	ModeTimings
	ModeSummary
	ModeCache
	ModeSecurity
	ModeTimeline
	ModeDiff
)

type AppModel struct {
	root         *har.Node
	left         *har.Node
	right        *har.Node
	tree         modes.TreeModel
	requests     modes.RequestsModel
	timings      modes.TimingsModel
	summary      modes.SummaryModel
	cache        modes.CacheModel
	security     modes.SecurityModel
	timeline     modes.TimelineModel
	diff         modes.DiffModel
	search       modes.SearchModel
	detail       DetailModel
	help         HelpModel
	options      OptionsModel
	popover      Popover
	keys         *KeyMap
	theme        *theme.Engine
	settings     *config.Settings
	mode         Mode
	width        int
	height       int
	fileName     string
	searchActive bool
	statusMsg    string
	isDiffMode   bool
	themeFlag    string
}

type SaveMsg struct {
	Err error
}

type ThemeToggledMsg struct{}

type detailMsg struct {
	entry *har.EntryDetails
	raw   *har.Node
}

func NewApp(root *har.Node, fileName string, th *theme.Engine) AppModel {
	entries := har.GetEntries(root)
	entryDetails := make([]*har.EntryDetails, len(entries))
	for i, e := range entries {
		entryDetails[i] = har.GetEntryDetails(e)
	}

	keys := DefaultKeyMap()
	settings := config.Load()

	m := AppModel{
		root:     root,
		tree:     modes.NewTreeModel(root, th, RenderNodeLine),
		requests: modes.NewRequestsModel(entryDetails, entries, th),
		timings:  modes.NewTimingsModel(entryDetails, entries, th),
		summary:  modes.NewSummaryModel(entryDetails, th),
		cache:    modes.NewCacheModel(entryDetails, entries, th),
		security: modes.NewSecurityModel(entryDetails, th),
		timeline: modes.NewTimelineModel(entryDetails, entries, th),
		search:   modes.NewSearchModel(th),
		detail:   NewDetailModel(nil, nil, th),
		help:     NewHelpModel(th),
		options:  NewOptionsModel(settings, th),
		popover:  NewPopover(nil),
		keys:     keys,
		theme:    th,
		settings: settings,
		mode:     ModeTree,
		fileName: fileName,
	}
	m.summary.SetGraphSymbol(settings.GraphSymbol)
	m.detail.SetGraphSymbol(settings.GraphSymbol)
	m.applySettings()
	m.timeline.SetGraphSymbol(settings.GraphSymbol)
	m.mode = modeFromName(settings.DefaultMode)
	return m
}

// NewDiffApp creates an AppModel configured for diff mode with two HAR files.
func NewDiffApp(left, right *har.Node, leftName, rightName string, th *theme.Engine) AppModel {
	keys := DefaultKeyMap()
	settings := config.Load()

	m := AppModel{
		left:      left,
		right:     right,
		diff:      modes.NewDiffModel(left, right, leftName, rightName, th),
		search:    modes.NewSearchModel(th),
		detail:    NewDetailModel(nil, nil, th),
		help:      NewHelpModel(th),
		options:   NewOptionsModel(settings, th),
		popover:   NewPopover(nil),
		keys:      keys,
		theme:     th,
		settings:  settings,
		mode:      ModeDiff,
		isDiffMode: true,
		fileName:  leftName,
	}
	m.applySettings()
	return m
}

func modeFromName(name string) Mode {
	switch name {
	case "requests":
		return ModeRequests
	case "timings":
		return ModeTimings
	case "summary":
		return ModeSummary
	case "cache":
		return ModeCache
	case "security":
		return ModeSecurity
	case "timeline":
		return ModeTimeline
	default:
		return ModeTree
	}
}

func (m AppModel) Init() tea.Cmd {
	return nil
}

func (m AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		bodyHeight := msg.Height - 3
		m.tree.SetViewportSize(msg.Width, bodyHeight)
		m.requests.SetViewportSize(msg.Width, bodyHeight)
		m.timings.SetViewportSize(msg.Width, bodyHeight)
		m.summary.SetViewportSize(msg.Width, bodyHeight)
		m.cache.SetViewportSize(msg.Width, bodyHeight)
		m.security.SetViewportSize(msg.Width, bodyHeight)
		m.timeline.SetViewportSize(msg.Width, bodyHeight)
		return m, nil

	case tea.KeyPressMsg:
		return m.handleKey(msg)

	case detailMsg:
		m.detail = NewDetailModel(msg.entry, msg.raw, m.theme)
		m.detail.SetGraphSymbol(m.settings.GraphSymbol)
		boxW := m.width*2/3 - 8
		if boxW < 20 {
			boxW = 20
		}
		boxH := m.height*2/3 - 4
		if boxH < 6 {
			boxH = 6
		}
		m.detail.viewport.SetWidth(boxW)
		m.detail.viewport.SetHeight(boxH)
		m.popover.Open(&m.detail, m.width, m.height)
		return m, nil

	case modes.SecurityDetailMsg:
		m.detail = NewDetailModel(nil, nil, m.theme)
		m.detail.SetGraphSymbol(m.settings.GraphSymbol)
		m.detail.SetCustomContent(modes.RenderSecurityDetail(msg.Result, m.theme, m.width*2/3-8))
		boxW := m.width*2/3 - 8
		if boxW < 20 {
			boxW = 20
		}
		boxH := m.height*2/3 - 4
		if boxH < 6 {
			boxH = 6
		}
		m.detail.viewport.SetWidth(boxW)
		m.detail.viewport.SetHeight(boxH)
		m.popover.Open(&m.detail, m.width, m.height)
		return m, nil

	case ThemeToggledMsg:
		return m, nil

	case SaveMsg:
		if msg.Err != nil {
			m.statusMsg = "Save failed: " + msg.Err.Error()
		} else {
			m.statusMsg = "Saved"
		}
		return m, nil

	case confirmMsg:
		m.popover.Close()
		if msg.confirmed {
			har.ScrubContent(m.root)
			m.statusMsg = "Content scrubbed (Ctrl+S to save)"
		} else {
			m.statusMsg = "Scrub cancelled"
		}
		return m, nil
	}

	return m, nil
}

func (m AppModel) handleKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	m.statusMsg = ""

	if m.popover.IsActive() {
		switch key {
		case "esc", "q":
			if opts, ok := m.popover.Content().(OptionsModel); ok {
				prevTheme := m.settings.Theme
				m.settings = opts.SettingsFromOptions()
				if !opts.themeTouched {
					// Color Theme was not adjusted: leave the active theme
					// alone and keep whatever settings.toml held (it may be a
					// custom theme path the panel's built-in list cannot
					// represent).
					m.settings.Theme = prevTheme
				}
				m.applySettings()
				if opts.themeTouched {
					// An explicit Color Theme choice applies immediately,
					// even when a --theme flag or config file pinned the
					// startup theme.
					m.applyTheme()
				}
				m.settings.Save()
			}
			m.popover.Close()
			return m, nil
		default:
			content := m.popover.Content()
			if content != nil {
				result, cmd := content.Update(msg)
				m.popover.SetContent(result)
				return m, cmd
			}
		}
		return m, nil
	}

	if m.searchActive {
		switch key {
		case "esc":
			m.searchActive = false
			if m.mode == ModeRequests {
				m.requests.SetFilter("")
			}
			return m, nil
		case "enter":
			m.searchActive = false
			if m.mode == ModeRequests {
				m.requests.SetFilter(m.search.Input.Value())
				return m, nil
			}
			sel := m.search.Selected()
			if sel != nil && sel.Node != nil {
				m.tree.SetCursor(sel.LineIndex)
				return m, nil
			}
			return m, nil
		default:
			var cmd tea.Cmd
			m.search, cmd = m.search.Update(msg)
			if m.mode == ModeRequests {
				m.requests.SetFilter(m.search.Input.Value())
				return m, cmd
			}
			m.search.Search(m.root, m.search.Input.Value())
			return m, cmd
		}
	}

	switch key {
	case "q", "ctrl+c":
		return m, tea.Quit

	case "1":
		if !m.isDiffMode {
			m.mode = ModeTree
			return m, nil
		}

	case "2":
		if !m.isDiffMode {
			m.mode = ModeRequests
			return m, nil
		}

	case "3":
		if !m.isDiffMode {
			m.mode = ModeTimings
			return m, nil
		}

	case "4":
		if !m.isDiffMode {
			m.mode = ModeSummary
			return m, nil
		}

	case "5":
		if !m.isDiffMode {
			m.mode = ModeCache
			return m, nil
		}

	case "6":
		if !m.isDiffMode {
			m.mode = ModeSecurity
			return m, nil
		}

	case "7":
		if !m.isDiffMode {
			m.mode = ModeTimeline
			return m, nil
		}

	case "/":
		m.searchActive = true
		if m.mode == ModeRequests {
			m.search.Input.SetValue(m.requests.Filter())
		} else {
			m.search.Input.SetValue("")
		}
		return m, m.search.Input.Focus()

	case "?":
		m.help.viewport.SetWidth(m.width - 8)
		m.help.viewport.SetHeight(m.height - 8)
		m.popover.Open(&m.help, m.width, m.height)
		return m, nil

	case "o":
		m.options = NewOptionsModel(m.settings, m.theme)
		m.options.viewport.SetWidth(m.width - 8)
		m.options.viewport.SetHeight(m.height - 8)
		m.popover.Open(&m.options, m.width, m.height)
		return m, nil

	case "t":
		m.theme.Toggle()
		// Persist the newly selected theme so it is remembered next launch.
		if name := m.theme.Name(); name != "" {
			m.settings.Theme = name
			m.settings.Save()
		}
		return m, func() tea.Msg { return ThemeToggledMsg{} }

	case "s":
		if m.mode == ModeRequests {
			m.requests.CycleSort()
			return m, nil
		}
		m.popover.Open(newConfirmModel(m.theme, "Scrub response bodies? This removes all content text, postData and encoding from the file in memory."), m.width, m.height)
		return m, nil

	case "ctrl+s":
		return m, func() tea.Msg {
			data, err := har.ExportJSON(m.root)
			if err != nil {
				return SaveMsg{Err: err}
			}
			if err := har.WriteFileAuto(m.fileName, data); err != nil {
				return SaveMsg{Err: err}
			}
			return SaveMsg{Err: nil}
		}

	case "n":
		if m.searchActive || len(m.search.Results) > 0 {
			m.search.Cursor++
			if m.search.Cursor >= len(m.search.Results) {
				m.search.Cursor = 0
			}
			sel := m.search.Selected()
			if sel != nil && sel.Node != nil {
				m.tree.SetCursor(sel.LineIndex)
			}
		}

	case "N":
		if m.searchActive || len(m.search.Results) > 0 {
			if m.search.Cursor > 0 {
				m.search.Cursor--
			}
			sel := m.search.Selected()
			if sel != nil && sel.Node != nil {
				m.tree.SetCursor(sel.LineIndex)
			}
		}
	}

	switch m.mode {
	case ModeTree:
		return m.updateTree(msg)
	case ModeRequests:
		return m.updateRequests(msg)
	case ModeTimings:
		return m.updateTimings(msg)
	case ModeSummary:
		return m.updateSummary(msg)
	case ModeCache:
		return m.updateCache(msg)
	case ModeSecurity:
		return m.updateSecurity(msg)
	case ModeTimeline:
		return m.updateTimeline(msg)
	case ModeDiff:
		return m.updateDiff(msg)
	}

	return m, nil
}

func (m AppModel) updateTree(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.tree, cmd = m.tree.Update(msg)
	return m, cmd
}

func (m AppModel) updateRequests(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	if m.requests.FilterMode() {
		switch key {
		case "esc", "f", "enter":
			m.requests.SetTypeFilterMode(false)
			return m, nil
		case "left", "h":
			m.requests.MoveTypeCursor(-1)
			return m, nil
		case "right", "l":
			m.requests.MoveTypeCursor(1)
			return m, nil
		case "space":
			m.requests.ToggleTypeAtCursor()
			return m, nil
		default:
			var cmd tea.Cmd
			m.requests, cmd = m.requests.Update(msg)
			return m, cmd
		}
	}

	switch key {
	case "enter":
		entry := m.requests.Selected()
		raw := m.requests.SelectedRaw()
		if entry != nil {
			return m, func() tea.Msg {
				return detailMsg{entry: entry, raw: raw}
			}
		}

	case "f":
		m.requests.ToggleTypeFilterMode()
		return m, nil

	case "S":
		m.requests.ToggleSortDir()
		return m, nil
	}

	var cmd tea.Cmd
	m.requests, cmd = m.requests.Update(msg)
	return m, cmd
}

func (m AppModel) updateTimings(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	switch key {
	case "enter":
		entry := m.timings.Selected()
		raw := m.timings.SelectedRaw()
		if entry != nil {
			return m, func() tea.Msg {
				return detailMsg{entry: entry, raw: raw}
			}
		}
	}

	var cmd tea.Cmd
	m.timings, cmd = m.timings.Update(msg)
	return m, cmd
}

func (m AppModel) updateSummary(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.summary, cmd = m.summary.Update(msg)
	return m, cmd
}

func (m AppModel) updateCache(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	if key == "enter" {
		entry := m.cache.Selected()
		raw := m.cache.SelectedRaw()
		if entry != nil {
			return m, func() tea.Msg {
				return detailMsg{entry: entry, raw: raw}
			}
		}
	}

	var cmd tea.Cmd
	m.cache, cmd = m.cache.Update(msg)
	return m, cmd
}

func (m AppModel) updateSecurity(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.security, cmd = m.security.Update(msg)
	return m, cmd
}

func (m AppModel) updateTimeline(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	if key == "enter" {
		if it := m.timeline.Selected(); it != nil {
			return m, func() tea.Msg {
				return detailMsg{entry: it.Entry(), raw: it.Raw()}
			}
		}
	}
	var cmd tea.Cmd
	m.timeline, cmd = m.timeline.Update(msg)
	return m, cmd
}

func (m AppModel) updateDiff(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.diff, cmd = m.diff.Update(msg)
	return m, cmd
}

func (m AppModel) View() tea.View {
	if m.width == 0 || m.height == 0 {
		v := tea.NewView("Initializing...")
		v.AltScreen = true
		return v
	}

	// Layout: 1-line header (view name) + scrollable body + 2-line footer (controls).
	bodyHeight := m.height - 3
	if bodyHeight < 1 {
		bodyHeight = 1
	}

	var contentView string
	switch m.mode {
	case ModeTree:
		m.tree.SetViewportSize(m.width, bodyHeight)
		contentView = m.tree.View()
	case ModeRequests:
		m.requests.SetViewportSize(m.width, bodyHeight)
		contentView = m.requests.View()
	case ModeTimings:
		m.timings.SetViewportSize(m.width, bodyHeight)
		contentView = m.timings.View()
	case ModeSummary:
		m.summary.SetViewportSize(m.width, bodyHeight)
		contentView = m.summary.View()
	case ModeCache:
		m.cache.SetViewportSize(m.width, bodyHeight)
		contentView = m.cache.View()
	case ModeSecurity:
		m.security.SetViewportSize(m.width, bodyHeight)
		contentView = m.security.View()
	case ModeTimeline:
		m.timeline.SetViewportSize(m.width, bodyHeight)
		contentView = m.timeline.View()
	case ModeDiff:
		m.diff.SetViewportSize(m.width, bodyHeight)
		contentView = m.diff.View()
	}

	contentView = padContentToHeight(contentView, bodyHeight)

	header := m.renderViewHeader()
	footer1, footer2 := m.renderControlsFooter()

	view := header + "\n" + contentView + "\n" + footer1 + "\n" + footer2

	if m.searchActive {
		view = m.renderSearchOverlay(view)
	}

	if m.popover.IsActive() {
		view = m.popover.Render(view)
	}

	v := tea.NewView(view)
	v.AltScreen = true
	// Paint the theme's background color across the whole terminal. The tea
	// renderer restores the terminal default when the altscreen is exited.
	v.BackgroundColor = lipgloss.Color(m.theme.GetBg())
	return v
}

func padContentToHeight(content string, targetLines int) string {
	parts := strings.Split(content, "\n")
	if len(parts) > targetLines {
		parts = parts[:targetLines]
	}
	for len(parts) < targetLines {
		parts = append(parts, "")
	}
	return strings.Join(parts, "\n")
}

// renderViewHeader renders the fixed 1-line header: the bold, centered view
// name with the theme name right-aligned in decorative brackets.
func (m AppModel) renderViewHeader() string {
	if m.isDiffMode {
		boldStyle := m.theme.GetStyle("title")
		title := m.titleWithBadge(boldStyle.Render("HAR Difference"))
		if m.diff.IsVarianceInputActive() {
			inputStyle := m.theme.GetStyle("status_line").Bold(true)
			options := fmt.Sprintf(" Variance: %s%% (Enter to confirm, Esc to cancel) ", m.diff.VarianceInput())
			optionsLine := lipgloss.NewStyle().Width(m.width).Align(lipgloss.Center).Render(inputStyle.Render(options))
			return title + "\n" + optionsLine
		}
		options := fmt.Sprintf(" Time Variance Limit: %.0f%% ", m.diff.Options().TimeVariancePercent)
		optStyle := m.theme.GetStyle("status_line")
		optionsLine := lipgloss.NewStyle().Width(m.width).Align(lipgloss.Center).Render(optStyle.Render(options))
		return title + "\n" + optionsLine
	}

	modeNames := map[Mode]string{
		ModeTree:     "Tree",
		ModeRequests: "Requests",
		ModeTimings:  "Timings",
		ModeSummary:  "Summary",
		ModeCache:    "Cache",
		ModeSecurity: "Security",
		ModeTimeline: "Timeline",
	}
	name := modeNames[m.mode]

	boldStyle := m.theme.GetStyle("title")
	return m.titleWithBadge(boldStyle.Render(name))
}

// titleWithBadge centers the (already styled) title text and right-aligns the
// theme name in decorative brackets, e.g. "❮ Turbo ❯". On narrow screens where
// the badge would collide with the centered title, the title is centered alone.
func (m AppModel) titleWithBadge(titleText string) string {
	badge := "❮ " + m.theme.Name() + " ❯"
	badgeText := m.theme.GetStyle("status_line").Render(badge)

	badgeW := lipgloss.Width(badgeText)
	nameW := lipgloss.Width(titleText)
	leftPad := (m.width - nameW) / 2
	if leftPad < 0 {
		leftPad = 0
	}
	rightPad := m.width - nameW - leftPad
	badgeSpace := rightPad - badgeW
	if badgeSpace < 3 {
		// Not enough room for the badge next to the title; center title only.
		return lipgloss.NewStyle().Width(m.width).Align(lipgloss.Center).Render(titleText)
	}
	line := strings.Repeat(" ", leftPad) + titleText + strings.Repeat(" ", badgeSpace) + badgeText
	return fitLine(line, m.width)
}

// footerHint is a "key: description" pair rendered in the footer, with the key
// bolded.
type footerHint struct {
	key  string
	desc string
}

// renderControlsFooter renders the fixed 2-line footer with global and view-specific controls.
func (m AppModel) renderControlsFooter() (string, string) {
	style := m.theme.GetStyle("status_line")
	keyStyle := style.Bold(true)

	modes := []footerHint{
		{"1", "Tree"},
		{"2", "Requests"},
		{"3", "Timings"},
		{"4", "Summary"},
		{"5", "Cache"},
		{"6", "Security"},
		{"7", "Timeline"},
	}
	actions := []footerHint{
		{"s", "Scrub"},
		{"/", "Search"},
		{"o", "Options"},
		{"?", "Help"},
		{"q", "Quit"},
	}

	// Spelt-out view names may not fit; drop one letter from each currently
	// longest name until the whole line fits.
	for hintsWidth(append(modes, actions...)) > m.width {
		longest := 0
		for _, it := range modes {
			if len(it.desc) > longest {
				longest = len(it.desc)
			}
		}
		if longest <= 1 {
			break
		}
		for i := range modes {
			if len(modes[i].desc) == longest {
				modes[i].desc = modes[i].desc[:len(modes[i].desc)-1]
			}
		}
	}
	global := fitLine(renderFooterHints(append(modes, actions...), style, keyStyle), m.width)

	controls := map[Mode][]footerHint{
		ModeTree:     {{"j/k", "move"}, {"PgUp/PgDn", "page"}, {"h/l", "collapse/expand"}, {"Enter", "toggle"}, {"E", "expand all"}, {"c", "collapse all"}},
		ModeRequests: {{"j/k", "move"}, {"PgUp/PgDn", "page"}, {"f", "filter"}, {"s", "sort"}, {"S", "direction"}, {"Enter", "detail"}},
		ModeTimings:  {{"j/k", "move"}, {"PgUp/PgDn", "page"}, {"Enter", "detail"}, {"+/−", "widen/narrow"}, {"←/→", "scroll"}},
		ModeSummary:  {{"j/k", "scroll"}},
		ModeCache:    {{"Tab", "sections"}, {"j/k", "scroll"}, {"PgUp/PgDn", "page"}, {"Enter", "detail"}},
		ModeSecurity: {{"j/k", "move"}, {"PgUp/PgDn", "page"}, {"Enter", "detail"}},
		ModeTimeline: {{"j/k", "move"}, {"PgUp/PgDn", "page"}, {"Enter", "detail"}, {"+/−", "widen/narrow"}, {"←/→", "scroll"}},
		ModeDiff:     {{"j/k", "move"}, {"PgUp/PgDn", "page"}, {"h/l", "collapse/expand"}, {"←/→", "scroll"}, {"Enter", "toggle"}, {"E", "expand all"}, {"c", "collapse all"}, {"v", "variance %"}},
	}

	specific := renderFooterHints(controls[m.mode], style, keyStyle)
	if m.statusMsg != "" {
		specific = style.Render(fitLine(m.statusMsg, m.width))
	} else {
		specific = fitLine(specific, m.width)
	}

	return specific, global
}

// hintsWidth returns the rendered (plain-text) width of a hint list joined by
// single spaces, with empty descriptions omitted.
func hintsWidth(items []footerHint) int {
	w := 0
	for i, it := range items {
		if i > 0 {
			w++
		}
		w += len(it.key)
		if it.desc != "" {
			w += 1 + len(it.desc)
		}
	}
	return w
}

// renderFooterHints renders hints as "key: desc" pairs joined by single spaces,
// bolding each key.
func renderFooterHints(items []footerHint, style, keyStyle lipgloss.Style) string {
	parts := make([]string, 0, len(items))
	for _, it := range items {
		if it.desc == "" {
			parts = append(parts, style.Render(it.key))
			continue
		}
		parts = append(parts, keyStyle.Render(it.key)+style.Render(":"+it.desc))
	}
	return strings.Join(parts, " ")
}

// fitLine truncates s to at most maxWidth visible columns, preserving any
// trailing ANSI escape sequence rather than splitting it.
func fitLine(s string, maxWidth int) string {
	if maxWidth <= 0 {
		return ""
	}
	if lipgloss.Width(s) <= maxWidth {
		return s
	}
	width := 0
	out := ""
	inAnsi := false
	for _, r := range s {
		if inAnsi {
			out += string(r)
			if r == 'm' {
				inAnsi = false
			}
			continue
		}
		if r == '\x1b' {
			out += string(r)
			inAnsi = true
			continue
		}
		rw := lipgloss.Width(string(r))
		if width+rw > maxWidth {
			break
		}
		width += rw
		out += string(r)
	}
	return out
}

func (m *AppModel) applySettings() {
	// A --theme flag or an auto-loaded theme.toml takes precedence over the
	// persisted settings theme; the engine already applied it at startup.
	if m.theme.CLITheme() == "" && m.theme.ConfigFile() == "" {
		m.applyTheme()
	}
	m.theme.SetTruecolor(m.settings.Truecolor)
	m.timings.SetGraphSymbol(m.settings.GraphSymbol)
	m.summary.SetGraphSymbol(m.settings.GraphSymbol)
	m.detail.SetGraphSymbol(m.settings.GraphSymbol)
	m.timeline.SetGraphSymbol(m.settings.GraphSymbol)
}

// applyTheme applies the theme selected in settings. It is separate from
// applySettings so that an explicit in-session theme change (e.g. from the
// options panel) can be applied immediately even when a --theme flag or an
// auto-loaded theme.toml pinned the startup theme.
func (m *AppModel) applyTheme() {
	switch m.settings.Theme {
	case "light":
		m.theme.SetLight()
	case "spectrum":
		m.theme.SetSpectrum()
	case "turbo":
		m.theme.Set("turbo")
	case "dark":
		m.theme.SetDark()
	default:
		if tc, err := theme.LoadFromFile(m.settings.Theme); err == nil {
			m.theme.SetConfig(tc)
		} else {
			m.theme.SetDark()
		}
	}
}

func (m AppModel) renderSearchOverlay(baseView string) string {
	searchBar := m.search.Input.View()

	box := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color(m.theme.GetFg())).
		Width(m.width-4).
		Padding(0, 1).
		Render(searchBar)

	lines := strings.Split(baseView, "\n")
	result := make([]string, 0, len(lines)+2)
	result = append(result, box)

	for _, line := range lines {
		result = append(result, line)
	}

	return strings.Join(result, "\n")
}
