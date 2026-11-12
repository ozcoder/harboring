package tui

import (
	"os"
	"path/filepath"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/harboring/harboring/internal/config"
	"github.com/harboring/harboring/internal/theme"
)

// writeThemeTOML creates a config dir containing theme.toml with the given
// foreground and points XDG_CONFIG_HOME at it (hermetic: no real user
// config is read or written by these tests).
func writeThemeTOML(t *testing.T, fg string) {
	t.Helper()
	configDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configDir)
	if err := os.MkdirAll(filepath.Join(configDir, "harboring"), 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(configDir, "harboring", "theme.toml")
	if err := os.WriteFile(path, []byte("[palette]\nfg = \""+fg+"\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
}

// openAndCloseOptions presses o, sends the given keys inside the options
// popover, then closes it with Esc.
func openAndCloseOptions(t *testing.T, m AppModel, keys ...tea.KeyPressMsg) AppModel {
	t.Helper()
	res, _ := m.handleKey(tea.KeyPressMsg{Text: "o"})
	m = res.(AppModel)
	if !m.popover.IsActive() {
		t.Fatal("expected options popover to open")
	}
	for _, k := range keys {
		res, _ = m.handleKey(k)
		m = res.(AppModel)
	}
	res, _ = m.handleKey(tea.KeyPressMsg{Code: tea.KeyEscape})
	m = res.(AppModel)
	if m.popover.IsActive() {
		t.Fatal("expected options popover to close")
	}
	return m
}

// Regression: ~/.config/harboring/theme.toml used to be loaded at engine
// startup and then immediately overridden by the persisted (or default)
// settings theme in NewApp -> applySettings.
func TestConfigFileThemeWinsOverSettings(t *testing.T) {
	writeThemeTOML(t, "#ABCDEF")

	th := theme.New()
	th.LoadStartup("") // picks up theme.toml
	m := makeAppWithTheme(th)
	if got := th.GetFg(); got != "#ABCDEF" {
		t.Fatalf("theme.toml should survive NewApp/applySettings, got fg %q (default dark applied?)", got)
	}

	// Even an explicitly persisted settings theme must not override the
	// config file; it applies only when no theme is pinned.
	m.settings.Theme = "light"
	m.applySettings()
	if got := th.GetFg(); got != "#ABCDEF" {
		t.Fatalf("persisted settings theme must not override theme.toml, got fg %q", got)
	}

	// Without a config file the persisted theme applies as before.
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	th2 := theme.New()
	th2.LoadStartup("")
	m2 := makeAppWithTheme(th2)
	m2.settings.Theme = "light"
	m2.applySettings()
	if th2.Name() != "light" {
		t.Fatalf("persisted theme should apply when no theme.toml exists, got name %q", th2.Name())
	}
}

// Closing the options panel without touching Color Theme must not revert a
// --theme-pinned theme (previously applyTheme() ran unconditionally and
// reset it to the settings value).
func TestOptionsCloseKeepsFlagPinnedTheme(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	th := theme.New()
	th.LoadStartup("turbo")
	m := makeAppWithTheme(th)
	m.settings = config.DefaultSettings() // theme = "dark", never chosen

	m = openAndCloseOptions(t, m, tea.KeyPressMsg{Code: tea.KeyDown})
	if got := m.theme.Name(); got != "turbo" {
		t.Fatalf("closing options without changing Color Theme should keep --theme turbo, got %q", got)
	}
	if got := m.settings.Theme; got != "dark" {
		t.Fatalf("settings theme should be untouched, got %q", got)
	}
}

// The same holds for a config-file-pinned theme.
func TestOptionsCloseKeepsConfigFileTheme(t *testing.T) {
	writeThemeTOML(t, "#ABCDEF")

	th := theme.New()
	th.LoadStartup("")
	m := makeAppWithTheme(th)

	_ = openAndCloseOptions(t, m, tea.KeyPressMsg{Code: tea.KeyDown})
	if got := th.GetFg(); got != "#ABCDEF" {
		t.Fatalf("closing options without changing Color Theme should keep theme.toml, got fg %q", got)
	}
}

// An explicit Color Theme change in the options panel must apply
// immediately, even over a --theme flag.
func TestOptionsColorThemeChangeApplies(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	th := theme.New()
	th.LoadStartup("turbo")
	m := makeAppWithTheme(th)
	m.settings = config.DefaultSettings()

	// Cursor starts on Color Theme; one step right selects "light".
	m = openAndCloseOptions(t, m, tea.KeyPressMsg{Code: tea.KeyRight})
	if got := m.theme.Name(); got != "light" {
		t.Fatalf("changing Color Theme in options should apply even with --theme, got %q", got)
	}
	if got := m.settings.Theme; got != "light" {
		t.Fatalf("changed theme should persist to settings, got %q", got)
	}
}

// The options panel's built-in list cannot represent a custom theme path
// stored in settings.toml; closing the panel must not rewrite it to a
// built-in name.
func TestOptionsClosePreservesCustomThemePath(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	customPath := "/home/you/.config/harboring/mytheme.toml"
	m := makeApp()
	m.settings = config.DefaultSettings()
	m.settings.Theme = customPath

	m = openAndCloseOptions(t, m, tea.KeyPressMsg{Code: tea.KeyDown})
	if got := m.settings.Theme; got != customPath {
		t.Fatalf("expected custom theme path to be preserved, got %q", got)
	}
}
