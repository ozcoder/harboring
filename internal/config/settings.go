package config

import (
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

type Settings struct {
	Theme       string `toml:"theme"`
	Truecolor   bool   `toml:"truecolor"`
	GraphSymbol string `toml:"graph_symbol"`
	DefaultMode string `toml:"default_mode"`
	CompactMode bool   `toml:"compact_mode"`
}

func DefaultSettings() *Settings {
	return &Settings{
		Theme:       "dark",
		Truecolor:   true,
		GraphSymbol: "block",
		DefaultMode: "tree",
		CompactMode: false,
	}
}

func configPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "harboring", "settings.toml"), nil
}

func Load() *Settings {
	s := DefaultSettings()
	path, err := configPath()
	if err != nil {
		return s
	}
	if _, err := toml.DecodeFile(path, s); err != nil {
		return s
	}
	return s
}

func (s *Settings) Save() error {
	path, err := configPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := toml.NewEncoder(f)
	return enc.Encode(s)
}

func (s *Settings) BlockChar() string {
	switch s.GraphSymbol {
	case "braille":
		return "\u28ff"
	case "ascii":
		return "#"
	default:
		return "\u2588"
	}
}

var ThemeOptions = []string{"dark", "light", "spectrum", "turbo"}
var TruecolorOptions = []string{"on", "off"}
var GraphSymbolOptions = []string{"block", "braille", "ascii"}
var DefaultModeOptions = []string{"tree", "requests", "timings", "summary", "cache", "timeline"}
var CompactModeOptions = []string{"off", "on"}
