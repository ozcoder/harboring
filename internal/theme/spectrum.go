package theme

// DefaultSpectrum returns the "Spectrum" theme, inspired by the ZX Spectrum's
// 16-colour palette as it appeared on a PAL television in the 1980s. The eight
// base hues are shown at the ULA's ~0.75 drive level (the #C8 values) while the
// "bright" bit pushes them to full saturation (#FF). There was no orange or grey
// in the original hardware palette, so those are approximated with warm/neutral
// TV tones that sit naturally alongside the primaries.
func DefaultSpectrum() *ThemeConfig {
	return &ThemeConfig{
		Meta: MetaConfig{Name: "Spectrum", Author: "harboring"},
		Palette: map[string]string{
			"bg":      "#000000", // black
			"fg":      "#FFFFFF", // bright white
			"green":   "#00FF00", // bright green
			"red":     "#FF0000", // bright red
			"yellow":  "#FFFF00", // bright yellow
			"cyan":    "#00FFFF", // bright cyan
			"magenta": "#FF00FF", // bright magenta
			"blue":    "#0000FF", // bright blue
			"orange":  "#FF8000", // warm orange (approximation; no pure orange on the Spectrum)
			"gray":    "#888888", // dim grey for muted UI chrome
			"unknown": "#FF00FF", // bright magenta
			"warn":    "#FF8000", // warm orange
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
			"json_string":    {Fg: "green"},
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
			"title":          {Fg: "yellow", Bold: true},
			"status_line":    {Fg: "gray"},
			"border":         {Fg: "cyan"},
			"search_match":   {Fg: "bg", Bg: "yellow"},
			"diff_added":     {Fg: "green", Bg: "#1a3a1a", Bold: true},
			"diff_removed":   {Fg: "red", Bg: "#3a1a1a", Bold: true},
			"diff_modified":  {Fg: "yellow", Bg: "#3a3a1a", Bold: true},
			"diff_changed":   {Fg: "red", Bg: "#4a1a1a", Bold: true, Underline: true},
			"selection":      {Bg: "#333333", Bold: true},
		},
	}
}
