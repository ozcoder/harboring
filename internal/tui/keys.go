package tui

import "charm.land/bubbles/v2/key"

type KeyMap struct {
	Up            key.Binding
	Down          key.Binding
	Left          key.Binding
	Right         key.Binding
	Enter         key.Binding
	Escape        key.Binding
	Quit          key.Binding
	TreeMode      key.Binding
	RequestsMode  key.Binding
	WaterfallMode key.Binding
	Search        key.Binding
	Scrub         key.Binding
	Save          key.Binding
	SaveAs        key.Binding
	ToggleTheme   key.Binding
	Edit          key.Binding
	ExpandAll     key.Binding
	CollapseAll   key.Binding
	Filter        key.Binding
	ColumnPicker  key.Binding
	ZoomIn        key.Binding
	ZoomOut       key.Binding
	Next          key.Binding
	Prev          key.Binding
	Options       key.Binding
}

func DefaultKeyMap() *KeyMap {
	return &KeyMap{
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "down"),
		),
		Left: key.NewBinding(
			key.WithKeys("left", "h"),
			key.WithHelp("←/h", "collapse"),
		),
		Right: key.NewBinding(
			key.WithKeys("right", "l"),
			key.WithHelp("→/l", "expand"),
		),
		Enter: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "select"),
		),
		Escape: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "back"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
		TreeMode: key.NewBinding(
			key.WithKeys("1"),
			key.WithHelp("1", "tree view"),
		),
		RequestsMode: key.NewBinding(
			key.WithKeys("2"),
			key.WithHelp("2", "requests"),
		),
		WaterfallMode: key.NewBinding(
			key.WithKeys("3"),
			key.WithHelp("3", "waterfall"),
		),
		Search: key.NewBinding(
			key.WithKeys("/"),
			key.WithHelp("/", "search"),
		),
		Scrub: key.NewBinding(
			key.WithKeys("s"),
			key.WithHelp("s", "scrub"),
		),
		Save: key.NewBinding(
			key.WithKeys("ctrl+s"),
			key.WithHelp("ctrl+s", "save"),
		),
		SaveAs: key.NewBinding(
			key.WithKeys("ctrl+shift+s"),
			key.WithHelp("ctrl+shift+s", "save as"),
		),
		ToggleTheme: key.NewBinding(
			key.WithKeys("t"),
			key.WithHelp("t", "toggle theme"),
		),
		Edit: key.NewBinding(
			key.WithKeys("e"),
			key.WithHelp("e", "edit"),
		),
		ExpandAll: key.NewBinding(
			key.WithKeys("E"),
			key.WithHelp("E", "expand all"),
		),
		CollapseAll: key.NewBinding(
			key.WithKeys("c"),
			key.WithHelp("c", "collapse all"),
		),
		Filter: key.NewBinding(
			key.WithKeys("f"),
			key.WithHelp("f", "filter"),
		),
		ColumnPicker: key.NewBinding(
			key.WithKeys("C"),
			key.WithHelp("C", "columns"),
		),
		ZoomIn: key.NewBinding(
			key.WithKeys("+", "="),
			key.WithHelp("+", "zoom in"),
		),
		ZoomOut: key.NewBinding(
			key.WithKeys("-"),
			key.WithHelp("-", "zoom out"),
		),
		Next: key.NewBinding(
			key.WithKeys("n"),
			key.WithHelp("n", "next"),
		),
		Prev: key.NewBinding(
			key.WithKeys("N"),
			key.WithHelp("N", "prev"),
		),
		Options: key.NewBinding(
			key.WithKeys("o"),
			key.WithHelp("o", "options"),
		),
	}
}

func (k *KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Quit, k.Search, k.Options, k.TreeMode, k.RequestsMode, k.WaterfallMode}
}

func (k *KeyMap) FullHelp() []key.Binding {
	return []key.Binding{
		k.Quit, k.Search, k.Scrub, k.Save, k.SaveAs, k.ToggleTheme, k.Options,
		k.TreeMode, k.RequestsMode, k.WaterfallMode,
		k.Up, k.Down, k.Left, k.Right, k.Enter, k.Escape,
		k.Edit, k.ExpandAll, k.CollapseAll, k.Filter, k.ColumnPicker,
		k.ZoomIn, k.ZoomOut, k.Next, k.Prev,
	}
}
