package theme

// DefaultTurbo returns the "Turbo" theme, based on the colour scheme of
// Borland's Turbo Pascal 6/7 IDE. The default editor used a deep royal-blue
// background with white normal text, a bright yellow-to-white head for
// keywords, cyan/turquoise for strings, and grey for comments — the same
// palette that millions of DOS-era developers stared at for hours.
func DefaultTurbo() *ThemeConfig {
	return &ThemeConfig{
		Meta: MetaConfig{Name: "Turbo", Author: "harboring"},
		Palette: map[string]string{
			"bg":      "#0000A8", // classic Turbo IDE royal blue
			"fg":      "#FFFFFF", // bright white normal text
			"green":   "#00FFC8", // aqua/turquoise (strings' cousin)
			"red":     "#FF4040", // error red
			"yellow":  "#FFFF6B", // keyword yellow
			"cyan":    "#00E0FF", // turquoise strings
			"magenta": "#FF80E0", // bright pink/violet
			"blue":    "#6BC8FF", // light blue (keywords often rendered blue-white)
			"orange":  "#FFB060", // warm amber for warnings
			"gray":    "#9A9A9A", // grey comments / UI chrome
			"unknown": "#FF80E0", // bright magenta
			"warn":    "#FFB060", // warm amber
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
			"title":          {Fg: "yellow", Bold: true},
			"status_line":    {Fg: "gray"},
			"border":         {Fg: "cyan"},
			"search_match":   {Fg: "bg", Bg: "yellow"},
			"diff_added":     {Fg: "green", Bg: "#003A28", Bold: true},
			"diff_removed":   {Fg: "red", Bg: "#480000", Bold: true},
			"diff_modified":  {Fg: "yellow", Bg: "#3A3A00", Bold: true},
			"diff_changed":   {Fg: "red", Bg: "#500000", Bold: true, Underline: true},
			"selection":      {Bg: "#4040D0", Bold: true},
		},
	}
}
