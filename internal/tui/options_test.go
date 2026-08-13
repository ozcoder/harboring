package tui

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/harboring/harboring/internal/config"
	"github.com/harboring/harboring/internal/theme"
)

func TestNewOptionsModel(t *testing.T) {
	th := theme.New()
	settings := config.DefaultSettings()
	m := NewOptionsModel(settings, th)

	if m.cursor != 0 {
		t.Errorf("expected cursor 0, got %d", m.cursor)
	}
	if len(m.options) != 5 {
		t.Errorf("expected 5 options, got %d", len(m.options))
	}
}

func TestOptionsModelNavigation(t *testing.T) {
	th := theme.New()
	settings := config.DefaultSettings()
	m := NewOptionsModel(settings, th)

	m2 := m
	m2.cursor = 2

	result, _ := m2.Update(tea.KeyPressMsg{Code: tea.KeyUp})
	opts := result.(OptionsModel)
	if opts.cursor != 1 {
		t.Errorf("expected cursor 1 after up, got %d", opts.cursor)
	}

	result, _ = opts.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	opts = result.(OptionsModel)
	if opts.cursor != 2 {
		t.Errorf("expected cursor 2 after down, got %d", opts.cursor)
	}
}

func TestOptionsModelValueChange(t *testing.T) {
	th := theme.New()
	settings := config.DefaultSettings()
	m := NewOptionsModel(settings, th)

	m.cursor = 0
	result, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyRight})
	opts := result.(OptionsModel)
	if opts.options[0].current != 1 {
		t.Errorf("expected option 0 value index 1 after right, got %d", opts.options[0].current)
	}

	result, _ = opts.Update(tea.KeyPressMsg{Code: tea.KeyLeft})
	opts = result.(OptionsModel)
	if opts.options[0].current != 0 {
		t.Errorf("expected option 0 value index 0 after left, got %d", opts.options[0].current)
	}
}

func TestSettingsFromOptions(t *testing.T) {
	th := theme.New()
	settings := config.DefaultSettings()
	m := NewOptionsModel(settings, th)

	m.cursor = 0
	result, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyRight})
	opts := result.(OptionsModel)

	got := opts.SettingsFromOptions()
	if got.Theme != "light" {
		t.Errorf("expected theme 'light', got %q", got.Theme)
	}
	if !got.Truecolor {
		t.Error("expected truecolor true")
	}
}

func TestOptionsBoundaryNavigation(t *testing.T) {
	th := theme.New()
	settings := config.DefaultSettings()
	m := NewOptionsModel(settings, th)

	m.cursor = 0
	result, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyUp})
	opts := result.(OptionsModel)
	if opts.cursor != 0 {
		t.Errorf("expected cursor to stay at 0, got %d", opts.cursor)
	}

	m.cursor = len(m.options) - 1
	result, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	opts = result.(OptionsModel)
	if opts.cursor != len(opts.options)-1 {
		t.Errorf("expected cursor to stay at %d, got %d", len(opts.options)-1, opts.cursor)
	}
}
