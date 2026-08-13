package theme

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
	"charm.land/lipgloss/v2"
)

type Engine struct {
	current   *ThemeConfig
	dark      *ThemeConfig
	light     *ThemeConfig
	isDark     bool
	truecolor  bool
}

func New() *Engine {
	dark := DefaultDark()
	return &Engine{
		current:   dark,
		dark:      dark,
		light:     DefaultLight(),
		isDark:    true,
		truecolor: true,
	}
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

func (e *Engine) Toggle() *ThemeConfig {
	if e.isDark {
		e.SetLight()
	} else {
		e.SetDark()
	}
	return e.current
}

func (e *Engine) SetDark() {
	e.current = e.dark
	e.isDark = true
}

func (e *Engine) SetLight() {
	e.current = e.light
	e.isDark = false
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

func (e *Engine) LoadStartup(themeFlag string) {
	configDir, err := os.UserConfigDir()
	if err == nil {
		themePath := filepath.Join(configDir, "harboring", "theme.toml")
		if _, statErr := os.Stat(themePath); statErr == nil {
			if tc, loadErr := LoadFromFile(themePath); loadErr == nil {
				e.current = tc
				return
			}
		}
	}

	switch themeFlag {
	case "dark":
		e.SetDark()
	case "light":
		e.SetLight()
	case "":
	default:
		if tc, err := LoadFromFile(themeFlag); err == nil {
			e.current = tc
		}
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

func hexTo256(hex string) string {
	if len(hex) < 7 {
		return hex
	}
	var r, g, b int
	fmt.Sscanf(hex, "#%02x%02x%02x", &r, &g, &b)

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
