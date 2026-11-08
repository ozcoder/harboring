package theme

// DefaultGoth returns the "Goth" theme — black/violet abyss with blood red,
// cherry, neon pink/magenta and electric blue accents.
func DefaultGoth() *ThemeConfig {
	return &ThemeConfig{
		Meta: MetaConfig{Name: "Goth", Author: "harboring"},
		Palette: map[string]string{
			"bg":      "#280128", // purple 240
			"fg":      "#f6f0ff", // ghost-white lavender
			"green":   "#39ff14", // toxic neon green
			"red":     "#a30404", // blood red (idx 4)
			"yellow":  "#f90e0e", // cherry red (idx 7)
			"cyan":    "#2f06e1", // bright electric blue (idx 216)
			"magenta": "#fa2dfa", // neon pink/magenta (idx 248)
			"blue":    "#2f06e1", // bright electric blue (idx 216)
			"orange":  "#ff4500", // crimson-orange
			"gray":    "#6b5b8a", // gunmetal gray-violet
			"unknown": "#b300ff", // rich purple
			"warn":    "#a30404", // blood red warnings
		},
		Styles: map[string]StyleDef{
			"method_get":     {Fg: "green"},
			"method_post":    {Fg: "yellow"},
			"method_put":     {Fg: "orange"},
			"method_delete":  {Fg: "red"},
			"method_patch":   {Fg: "magenta"},
			"method_options": {Fg: "gray"},
			"method_head":    {Fg: "gray"},
			"status_2xx":     {Fg: "green"},
			"status_3xx":     {Fg: "cyan"},
			"status_4xx":     {Fg: "yellow"},
			"status_5xx":     {Fg: "red"},
			"json_key":       {Fg: "blue"},
			"json_string":    {Fg: "cyan"},
			"json_number":    {Fg: "magenta"},
			"json_bool":      {Fg: "yellow"},
			"json_null":      {Fg: "gray"},
			"json_bracket":   {Fg: "fg"},
			"json_unknown":   {Fg: "unknown", Italic: true},
			"timing_dns":     {Fg: "green"},
			"timing_connect": {Fg: "cyan"},
			"timing_ssl":     {Fg: "blue"},
			"timing_send":    {Fg: "yellow"},
			"timing_wait":    {Fg: "orange"},
			"timing_receive": {Fg: "magenta"},
			"timing_blocked": {Fg: "gray"},
			"title":          {Fg: "magenta", Bold: true},
			"status_line":    {Fg: "gray"},
			"border":         {Fg: "unknown"},
			"search_match":   {Fg: "#140214", Bg: "#fa2dfa"},
			"diff_added":     {Fg: "green", Bg: "#071a07", Bold: true},
			"diff_removed":   {Fg: "red", Bg: "#1a0505", Bold: true},
			"diff_modified":  {Fg: "yellow", Bg: "#1a080a", Bold: true},
			"diff_changed":   {Fg: "red", Bg: "#260404", Bold: true, Underline: true},
			"selection":      {Bg: "#0f0247", Bold: true}, // dark blue idx 211
		},
	}
}
