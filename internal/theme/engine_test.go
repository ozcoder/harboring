package theme

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNew(t *testing.T) {
	th := New()
	if th == nil {
		t.Fatal("New() returned nil")
	}
}

func TestGetStyle(t *testing.T) {
	th := New()
	style := th.GetStyle("title")
}

func TestToggle(t *testing.T) {
	th := New()
	start := th.GetFg()

	// A full cycle through all built-in themes returns to the start.
	names := []string{"dark", "light", "spectrum", "turbo", "goth"}
	for range names {
		th.Toggle()
	}
	if th.GetFg() != start {
		t.Fatal("theme toggle should cycle back to original after a full cycle")
	}

	// Each individual toggle changes the foreground.
	th.Set("dark")
	fg1 := th.GetFg()
	th.Toggle()
	if fg1 == th.GetFg() {
		t.Fatal("theme toggle should change foreground color")
	}
}

func TestSetSpectrum(t *testing.T) {
	th := New()
	if !th.Set("spectrum") {
		t.Fatal("Set(\"spectrum\") should succeed for a built-in theme")
	}
	if th.Name() != "spectrum" {
		t.Fatalf("expected current theme name \"spectrum\", got %q", th.Name())
	}
	// Authentic Spectrum screen is black background with white foreground.
	if th.GetBg() != "#000000" {
		t.Fatalf("expected Spectrum background #000000, got %q", th.GetBg())
	}
	if th.GetFg() != "#FFFFFF" {
		t.Fatalf("expected Spectrum foreground #FFFFFF, got %q", th.GetFg())
	}
	if th.Set("not-a-theme") {
		t.Fatal("Set should reject unknown theme names")
	}
	if th.Name() != "spectrum" {
		t.Fatalf("Set should be a no-op for unknown names, name=%q", th.Name())
	}
}

func TestGetWarn(t *testing.T) {
	th := New()
	warn := th.GetWarn()
	if warn == "" {
		t.Fatal("GetWarn should return a non-empty color")
	}
}

func TestGetGray(t *testing.T) {
	th := New()
	gray := th.GetGray()
	if gray == "" {
		t.Fatal("GetGray should return a non-empty color")
	}
}

func TestLoadStartup(t *testing.T) {
	th := New()
	// Passing empty string should not panic
	th.LoadStartup("")
}

func TestLoadStartupFlagOverridesConfig(t *testing.T) {
	configDir := t.TempDir()
	if _, err := os.MkdirAll(filepath.Join(configDir, "harboring"), 0o755); err != nil {
		t.Fatal(err)
	}

	configFile := filepath.Join(configDir, "harboring", "theme.toml")
	if err := os.WriteFile(configFile, []byte(`
[palette]
fg = "#ABCDEF"
`), 0o644); err != nil {
		t.Fatal(err)
	}

	t.Setenv("XDG_CONFIG_HOME", configDir)

	// Empty flag: the config-file theme applies.
	th := New()
	th.LoadStartup("")
	if got := th.GetFg(); got != "#ABCDEF" {
		t.Fatalf("expected config theme foreground, got %q", got)
	}

	// Explicit flag: must win over the config file.
	th = New()
	th.LoadStartup("dark")
	if got := th.GetFg(); got == "#ABCDEF" {
		t.Fatal("explicit --theme flag should override the config-file theme")
	}
}

func TestConfigFileTracking(t *testing.T) {
	// With a theme.toml present, LoadStartup records it as the pinned
	// startup source so the TUI knows not to apply the settings theme.
	configDir := t.TempDir()
	if _, err := os.MkdirAll(filepath.Join(configDir, "harboring"), 0o755); err != nil {
		t.Fatal(err)
	}

	configFile := filepath.Join(configDir, "harboring", "theme.toml")
	if err := os.WriteFile(configFile, []byte(`
[palette]
fg = "#ABCDEF"
`), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("XDG_CONFIG_HOME", configDir)

	th := New()
	th.LoadStartup("")
	if got := th.ConfigFile(); got != configFile {
		t.Fatalf("expected ConfigFile %q after loading theme.toml, got %q", configFile, got)
	}
	if got := th.CLITheme(); got != "" {
		t.Fatalf("expected CLITheme empty when only theme.toml was loaded, got %q", got)
	}

	// An explicit flag means the config file was not used.
	th = New()
	th.LoadStartup("dark")
	if got := th.ConfigFile(); got != "" {
		t.Fatalf("expected ConfigFile empty when a --theme flag was given, got %q", got)
	}

	// No config file: nothing recorded.
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	th = New()
	th.LoadStartup("")
	if got := th.ConfigFile(); got != "" {
		t.Fatalf("expected ConfigFile empty when theme.toml is absent, got %q", got)
	}
}

func TestCLITheme(t *testing.T) {
	th := New()
	th.LoadStartup("")
	if got := th.CLITheme(); got != "" {
		t.Fatalf("expected CLITheme empty when no flag given, got %q", got)
	}

	for _, name := range []string{"dark", "light", "spectrum", "turbo", "goth"} {
		th = New()
		th.LoadStartup(name)
		if got := th.CLITheme(); got != name {
			t.Fatalf("expected CLITheme %q, got %q", name, got)
		}
	}

	// Unknown names resolve to "", so the persisted theme can apply.
	th = New()
	th.LoadStartup("bogus")
	if got := th.CLITheme(); got != "" {
		t.Fatalf("expected CLITheme empty for unknown flag, got %q", got)
	}

	// A TUI toggle does not affect the recorded startup flag.
	th = New()
	th.LoadStartup("turbo")
	th.Toggle()
	if got := th.CLITheme(); got != "turbo" {
		t.Fatalf("expected CLITheme to remain %q after a toggle, got %q", "turbo", got)
	}
}
