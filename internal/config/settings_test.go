package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/BurntSushi/toml"
)

func TestDefaultSettings(t *testing.T) {
	s := DefaultSettings()
	if s.Theme != "dark" {
		t.Errorf("expected theme 'dark', got %q", s.Theme)
	}
	if !s.Truecolor {
		t.Error("expected truecolor to be true")
	}
	if s.GraphSymbol != "block" {
		t.Errorf("expected graph_symbol 'block', got %q", s.GraphSymbol)
	}
	if s.DefaultMode != "tree" {
		t.Errorf("expected default_mode 'tree', got %q", s.DefaultMode)
	}
	if s.CompactMode {
		t.Error("expected compact_mode to be false")
	}
}

func TestBlockChar(t *testing.T) {
	tests := []struct {
		symbol string
		want   string
	}{
		{"block", "\u2588"},
		{"braille", "\u28ff"},
		{"ascii", "#"},
		{"unknown", "\u2588"},
	}
	for _, tt := range tests {
		s := &Settings{GraphSymbol: tt.symbol}
		got := s.BlockChar()
		if got != tt.want {
			t.Errorf("BlockChar(%q) = %q, want %q", tt.symbol, got, tt.want)
		}
	}
}

func TestSaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "harboring", "settings.toml")

	s := &Settings{
		Theme:       "light",
		Truecolor:   false,
		GraphSymbol: "braille",
		DefaultMode: "waterfall",
		CompactMode: true,
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	enc := toml.NewEncoder(f)
	if err := enc.Encode(s); err != nil {
		f.Close()
		t.Fatal(err)
	}
	f.Close()

	var loaded Settings
	if _, err := toml.DecodeFile(path, &loaded); err != nil {
		t.Fatal(err)
	}
	if loaded.Theme != "light" {
		t.Errorf("expected theme 'light', got %q", loaded.Theme)
	}
	if loaded.Truecolor {
		t.Error("expected truecolor false")
	}
	if loaded.GraphSymbol != "braille" {
		t.Errorf("expected graph_symbol 'braille', got %q", loaded.GraphSymbol)
	}
	if loaded.DefaultMode != "waterfall" {
		t.Errorf("expected default_mode 'waterfall', got %q", loaded.DefaultMode)
	}
	if !loaded.CompactMode {
		t.Error("expected compact_mode true")
	}
}

func TestLoadMissingFile(t *testing.T) {
	s := Load()
	if s.Theme != "dark" {
		t.Errorf("expected default theme 'dark', got %q", s.Theme)
	}
}
