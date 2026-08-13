package tui

import (
	"fmt"
	"os"
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
	ModeWaterfall
	ModeSummary
	ModeCache
	ModeSecurity
)

type AppModel struct {
	root         *har.Node
	tree         modes.TreeModel
	requests     modes.RequestsModel
	waterfall    modes.WaterfallModel
	summary      modes.SummaryModel
	cache        modes.CacheModel
	security     modes.SecurityModel
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
	dirty        bool
	searchActive bool
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
		root:      root,
		tree:      modes.NewTreeModel(root, th, RenderNodeLine),
		requests:  modes.NewRequestsModel(entryDetails, entries, th),
		waterfall: modes.NewWaterfallModel(entryDetails, entries, th),
		summary:   modes.NewSummaryModel(entryDetails, th),
		cache:     modes.NewCacheModel(entryDetails, entries, th),
		security:  modes.NewSecurityModel(entryDetails, entries, th),
		search:    modes.NewSearchModel(th),
		detail:    NewDetailModel(nil, nil, th),
		help:      NewHelpModel(th),
		options:   NewOptionsModel(settings, th),
		popover:   NewPopover(nil),
		keys:      keys,
		theme:     th,
		settings:  settings,
		mode:      ModeTree,
		fileName:  fileName,
	}
	m.summary.SetGraphSymbol(settings.GraphSymbol)
	m.detail.SetGraphSymbol(settings.GraphSymbol)
	m.applySettings()
	return m
}

func (m AppModel) Init() tea.Cmd {
	return nil
}

func (m AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.tree.SetViewportSize(msg.Width, msg.Height-2)
		m.requests.SetViewportSize(msg.Width, msg.Height-2)
		m.waterfall.SetViewportSize(msg.Width, msg.Height-2)
		m.summary.SetViewportSize(msg.Width, msg.Height-1)
		m.cache.SetViewportSize(msg.Width, msg.Height-2)
		m.security.SetViewportSize(msg.Width, msg.Height-2)
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
		m.dirty = false
		return m, nil
	}

	return m, nil
}

func (m AppModel) handleKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	if m.popover.IsActive() {
		switch key {
		case "esc", "q":
			if opts, ok := m.popover.Content().(OptionsModel); ok {
				m.settings = opts.SettingsFromOptions()
				m.applySettings()
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
			return m, nil
		case "enter":
			m.searchActive = false
			sel := m.search.Selected()
			if sel != nil && sel.Node != nil {
				m.tree.SetCursor(sel.LineIndex)
				return m, nil
			}
			return m, nil
		default:
			var cmd tea.Cmd
			m.search, cmd = m.search.Update(msg)
			m.search.Search(m.root, m.search.Input.Value())
			return m, cmd
		}
	}

	switch key {
	case "q", "ctrl+c":
		return m, tea.Quit

	case "1":
		m.mode = ModeTree
		return m, nil

	case "2":
		m.mode = ModeRequests
		return m, nil

	case "3":
		m.mode = ModeWaterfall
		return m, nil

	case "4":
		m.mode = ModeSummary
		return m, nil

	case "5":
		m.mode = ModeCache
		return m, nil

	case "6":
		m.mode = ModeSecurity
		return m, nil

	case "/":
		m.searchActive = true
		m.search.Input.SetValue("")
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
		return m, func() tea.Msg { return ThemeToggledMsg{} }

	case "s":
		if m.mode == ModeRequests {
			m.requests.CycleSort()
			return m, nil
		}
		har.ScrubContent(m.root)
		m.dirty = true
		return m, nil

	case "ctrl+s":
		return m, func() tea.Msg {
			data, err := har.ExportJSON(m.root)
			if err != nil {
				return SaveMsg{Err: err}
			}
			if err := os.WriteFile(m.fileName, data, 0644); err != nil {
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
	case ModeWaterfall:
		return m.updateWaterfall(msg)
	case ModeSummary:
		return m.updateSummary(msg)
	case ModeCache:
		return m.updateCache(msg)
	case ModeSecurity:
		return m.updateSecurity(msg)
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

func (m AppModel) updateWaterfall(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	switch key {
	case "enter":
		entry := m.waterfall.Selected()
		raw := m.waterfall.SelectedRaw()
		if entry != nil {
			return m, func() tea.Msg {
				return detailMsg{entry: entry, raw: raw}
			}
		}

	case "C":
		return m, nil
	}

	var cmd tea.Cmd
	m.waterfall, cmd = m.waterfall.Update(msg)
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

func (m AppModel) View() tea.View {
	if m.width == 0 || m.height == 0 {
		v := tea.NewView("Initializing...")
		v.AltScreen = true
		return v
	}

	contentHeight := m.height - 1
	statusLine := m.renderStatusLine()

	var contentView string
	contentLines := contentHeight - 1
	switch m.mode {
	case ModeTree:
		m.tree.SetViewportSize(m.width, contentHeight-1)
		contentView = m.tree.View()
	case ModeRequests:
		m.requests.SetViewportSize(m.width, contentHeight-1)
		contentView = m.requests.View()
	case ModeWaterfall:
		m.waterfall.SetViewportSize(m.width, contentHeight-1)
		contentView = m.waterfall.View()
	case ModeSummary:
		m.summary.SetViewportSize(m.width, contentHeight)
		contentView = m.summary.View()
		contentLines = contentHeight
	case ModeCache:
		m.cache.SetViewportSize(m.width, contentHeight-1)
		contentView = m.cache.View()
	case ModeSecurity:
		m.security.SetViewportSize(m.width, contentHeight-1)
		contentView = m.security.View()
	}

	contentView = padContentToHeight(contentView, contentLines)

	view := contentView + "\n" + statusLine

	if m.searchActive {
		view = m.renderSearchOverlay(view)
	}

	if m.popover.IsActive() {
		view = m.popover.Render(view)
	}

	v := tea.NewView(view)
	v.AltScreen = true
	return v
}

func padContentToHeight(content string, targetLines int) string {
	lines := strings.Count(content, "\n")
	if content != "" && !strings.HasSuffix(content, "\n") {
		lines++
	}
	for lines < targetLines {
		content += "\n"
		lines++
	}
	if lines > targetLines {
		parts := strings.Split(content, "\n")
		content = strings.Join(parts[:targetLines], "\n")
	}
	return content
}

func (m AppModel) renderStatusLine() string {
	statusStyle := m.theme.GetStyle("status_line")

	modeNames := map[Mode]string{
		ModeTree:      "Tree",
		ModeRequests:  "Requests",
		ModeWaterfall: "Waterfall",
		ModeSummary:   "Summary",
		ModeCache:     "Cache",
		ModeSecurity:  "Security",
	}

	modeName := modeNames[m.mode]

	dirtyIndicator := ""
	if m.dirty {
		dirtyIndicator = " [modified]"
	}

	left := fmt.Sprintf(" %s%s", m.fileName, dirtyIndicator)
	right := fmt.Sprintf("%s │ 1:Tree 2:Req 3:Wf 4:Sum 5:Cache 6:Sec │ /:search ?:help q:quit ", modeName)

	usedWidth := lipgloss.Width(left) + lipgloss.Width(right)
	padding := m.width - usedWidth
	if padding < 0 {
		padding = 0
	}

	line := left + strings.Repeat(" ", padding) + right
	return statusStyle.Render(line)
}

func (m *AppModel) applySettings() {
	if m.settings.Theme == "dark" {
		m.theme.SetDark()
	} else {
		m.theme.SetLight()
	}
	m.theme.SetTruecolor(m.settings.Truecolor)
	m.waterfall.SetGraphSymbol(m.settings.GraphSymbol)
	m.summary.SetGraphSymbol(m.settings.GraphSymbol)
	m.detail.SetGraphSymbol(m.settings.GraphSymbol)
}

func (m AppModel) renderSearchOverlay(baseView string) string {
	searchBar := m.search.Input.View()

	box := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color(m.theme.GetFg())).
		Width(m.width - 4).
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
