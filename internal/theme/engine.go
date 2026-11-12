package theme

import (
	"fmt"
	"os"
	"path/filepath"

	"charm.land/lipgloss/v2"
	"github.com/BurntSushi/toml"
)

type Engine struct {
	current    *ThemeConfig
	dark     *ThemeConfig
	light    *ThemeConfig
	spectrum *ThemeConfig
	turbo    *ThemeConfig
	goth     *ThemeConfig
	isDark     bool
	truecolor  bool
	order      []string
	byName     map[string]*ThemeConfig
	curName    string
	cliTheme   string
	configFile string
}

func New() *Engine {
	dark := DefaultDark()
	light := DefaultLight()
	spectrum := DefaultSpectrum()
	turbo := DefaultTurbo()
	goth := DefaultGoth()
	e := &Engine{
		dark:      dark,
		light:     light,
		spectrum:  spectrum,
		turbo:     turbo,
		goth:      goth,
		isDark:    true,
		truecolor: true,
		order:     []string{"dark", "light", "spectrum", "turbo", "goth"},
	}
	e.byName = map[string]*ThemeConfig{
		"dark":     dark,
		"light":    light,
		"spectrum": spectrum,
		"turbo":    turbo,
		"goth":     goth,
	}
	e.current = dark
	e.curName = "dark"
	return e
}

func LoadFromFile(path string) (*ThemeConfig, error) {
	var tc ThemeConfig
	if _, err := toml.DecodeFile(path, &tc); err != nil {
		return nil, err
	}
	return &tc, nil
}

func (e *Engine) Current() *ThemeConfig {
	return e.current
}

// Name returns the name of the currently selected built-in theme.
func (e *Engine) Name() string {
	return e.curName
}

// Names returns the names of the built-in themes in toggle (cycle) order.
func (e *Engine) Names() []string {
	return append([]string(nil), e.order...)
}

// CLITheme returns the theme explicitly requested via the --theme flag at
// startup, or "" if none was given (or the given flag did not resolve).
func (e *Engine) CLITheme() string {
	return e.cliTheme
}

// ConfigFile returns the path of the auto-loaded user config theme
// (theme.toml in the user config directory), or "" if none was loaded at
// startup. Like CLITheme, a non-empty value marks the startup theme as
// pinned from outside settings.toml, so the persisted settings theme does
// not override it.
func (e *Engine) ConfigFile() string {
	return e.configFile
}

// Set selects a built-in theme by name (e.g. "dark", "light", "spectrum", "turbo", "goth").
// It returns false if the name is not a known built-in theme.
func (e *Engine) Set(name string) bool {
	cfg, ok := e.byName[name]
	if !ok {
		return false
	}
	e.current = cfg
	e.curName = name
	e.isDark = name != "light"
	return true
}

// SetConfig installs an arbitrary (e.g. file-loaded) theme configuration.
func (e *Engine) SetConfig(tc *ThemeConfig) {
	e.current = tc
	e.curName = ""
	e.isDark = true
}

func (e *Engine) Toggle() *ThemeConfig {
	idx := 0
	for i, n := range e.order {
		if n == e.curName {
			idx = i
			break
		}
	}
	e.Set(e.order[(idx+1)%len(e.order)])
	return e.current
}

func (e *Engine) SetDark() {
	e.Set("dark")
}

func (e *Engine) SetLight() {
	e.Set("light")
}

func (e *Engine) SetSpectrum() {
	e.Set("spectrum")
}

func (e *Engine) SetGoth() {
	e.Set("goth")
}

func (e *Engine) IsDark() bool {
	return e.isDark
}

func (e *Engine) SetTruecolor(enabled bool) {
	e.truecolor = enabled
}

func (e *Engine) IsTruecolor() bool {
	return e.truecolor
}

// LoadStartup selects the startup theme. An explicit CLI flag wins over the
// user config file; an empty flag falls back to the config file theme, then
// the built-in dark default. The TUI applies the persisted settings.toml
// theme afterwards only when neither of those pinned a theme (see
// Engine.ConfigFile).
func (e *Engine) LoadStartup(themeFlag string) {
	e.cliTheme = ""
	e.configFile = ""
	switch themeFlag {
	case "dark":
		e.SetDark()
		e.cliTheme = "dark"
		return
	case "light":
		e.SetLight()
		e.cliTheme = "light"
		return
	case "spectrum":
		e.SetSpectrum()
		e.cliTheme = "spectrum"
		return
	case "turbo":
		e.Set("turbo")
		e.cliTheme = "turbo"
		return
	case "goth":
		e.SetGoth()
		e.cliTheme = "goth"
		return
	case "":
		// fall through to the config file theme below
	default:
		if tc, err := LoadFromFile(themeFlag); err == nil {
			e.SetConfig(tc)
			e.cliTheme = themeFlag
			return
		}
	}

	configDir, err := os.UserConfigDir()
	if err != nil {
		return
	}
	themePath := filepath.Join(configDir, "harboring", "theme.toml")
	if _, statErr := os.Stat(themePath); statErr != nil {
		return
	}
	if tc, loadErr := LoadFromFile(themePath); loadErr == nil {
		e.SetConfig(tc)
		e.configFile = themePath
	}
}

func (e *Engine) resolveColor(colorRef string) string {
	if len(colorRef) > 0 && colorRef[0] == '#' {
		if !e.truecolor {
			return hexTo256(colorRef)
		}
		return colorRef
	}
	if color, ok := e.current.Palette[colorRef]; ok {
		if !e.truecolor && len(color) > 0 && color[0] == '#' {
			return hexTo256(color)
		}
		return color
	}
	return colorRef
}

func (e *Engine) GetStyle(name string) lipgloss.Style {
	def, ok := e.current.Styles[name]
	if !ok {
		return lipgloss.NewStyle()
	}

	s := lipgloss.NewStyle()

	if def.Fg != "" {
		s = s.Foreground(lipgloss.Color(e.resolveColor(def.Fg)))
	}
	if def.Bg != "" {
		s = s.Background(lipgloss.Color(e.resolveColor(def.Bg)))
	}
	if def.Bold {
		s = s.Bold(true)
	}
	if def.Italic {
		s = s.Italic(true)
	}
	if def.Underline {
		s = s.Underline(true)
	}

	return s
}

func (e *Engine) GetBg() string {
	return e.resolveColor(e.current.Palette["bg"])
}

func (e *Engine) GetFg() string {
	return e.resolveColor(e.current.Palette["fg"])
}

func (e *Engine) GetWarn() string {
	return e.resolveColor(e.current.Palette["warn"])
}

func (e *Engine) GetGray() string {
	return e.resolveColor(e.current.Palette["gray"])
}

// GetSelectionBg returns the resolved background colour used to highlight the
// currently selected row. If the theme does not define a selection style it
// falls back to the theme background.
func (e *Engine) GetSelectionBg() string {
	if def, ok := e.current.Styles["selection"]; ok && def.Bg != "" {
		return e.resolveColor(def.Bg)
	}
	return e.GetBg()
}

func hexTo256(hex string) string {
	if len(hex) < 7 {
		return hex
	}
	var r, g, b int
	_, err := fmt.Sscanf(hex, "#%02x%02x%02x", &r, &g, &b)
	if err != nil {
		return "0"
	}

	if r == 0 && g == 0 && b == 0 {
		return "0"
	}
	if r == 255 && g == 255 && b == 255 {
		return "15"
	}

	r6 := (r * 5) / 255
	g6 := (g * 5) / 255
	b6 := (b * 5) / 255
	ansi := 16 + 36*r6 + 6*g6 + b6
	return fmt.Sprintf("%d", ansi)
}
