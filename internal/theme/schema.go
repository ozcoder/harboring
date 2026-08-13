package theme

type ThemeConfig struct {
	Meta    MetaConfig          `toml:"meta"`
	Palette map[string]string   `toml:"palette"`
	Styles  map[string]StyleDef `toml:"styles"`
}

type MetaConfig struct {
	Name   string `toml:"name"`
	Author string `toml:"author"`
}

type StyleDef struct {
	Fg     string `toml:"fg"`
	Bg     string `toml:"bg"`
	Bold   bool   `toml:"bold"`
	Italic bool   `toml:"italic"`
}
