# Themes

Harboring paints the entire TUI from a single theme: a **palette** of named
colours plus a set of **styles** (foreground/background/attributes) that
reference those colours. Five themes are built in — `dark`, `light`, `spectrum` (retro ZX Spectrum),
`turbo` (Borland Turbo Pascal IDE), and `goth` (moody black/gray with
blood-red, cherry red, neon pink, purple and electric/neon blue) — and you
can load your own from a TOML file.

## Switching themes

| How | What it does |
| --- | --- |
| `t` in the TUI | Cycles `dark → light → spectrum → turbo → goth` and saves the choice to `settings.toml` |
| Options panel (`o`) → *Color Theme* | Same four built-ins; applied and saved when the panel closes (`Esc`/`q`) |
| `--theme <name\|path>` | Selects a built-in by name or loads a TOML file at launch |
| `--list themes` | Prints the built-in names and exits |

The active theme's name is shown in the title bar badge, e.g. `❮ turbo ❯`.

## Which theme wins at startup

1. **`--theme` flag** — highest priority. A built-in name selects that
   built-in; any other value is treated as a path to a TOML file. If that
   file cannot be loaded, the lower tiers are tried instead (the flag then
   behaves as if it had not been given).
2. **`~/.config/harboring/theme.toml`** — the user config theme, loaded
   automatically when no flag is in effect. Placing a TOML file here is how a
   custom theme becomes the default loaded at startup.
3. **`settings.toml` → `theme`** — what `t` and the options panel persist.
   Accepts a built-in name *or* a path to a TOML file
   (`theme = "/home/me/.config/harboring/mytheme.toml"`). Applied at launch
   only when neither a flag nor a config-file theme is present.
4. **Built-in `dark`** — fallback when none of the above resolve, e.g. the
   theme path in `settings.toml` points at a missing or malformed file.

While a `theme.toml` exists it defines the startup theme, so a theme picked
with `t` is remembered in `settings.toml` but only takes effect at the next
start once you remove (or rename) `theme.toml`. A `--theme` flag overrides
`theme.toml` for that run.

In-session changes always work: `t` and the options panel switch themes even
when the startup theme came from `--theme` or `theme.toml`. A theme chosen in
the options panel is applied as soon as the panel closes; a `t` choice is
applied immediately and saved to `settings.toml`.

## File format

A theme file has three tables:

```toml
[meta]      # informational only — not displayed by the UI
name = "Turbo"
author = "harboring"

[palette]   # named colours: name = "#rrggbb"
bg = "#0000A8"

[styles]    # style key = { fg, bg, bold, italic, underline }
method_get = { fg = "green" }
```

### Colour values

A style's `fg`/`bg` may be:

- **A palette name** — `"green"`, `"warn"`, `"bg"`, … (looked up in `[palette]`), or
- **A hex literal** — `"#003A28"`, useful for one-off backgrounds such as the
  diff colours.

Two silent-failure rules to be aware of:

- A style key the UI never asks for is ignored; a style key the UI asks for
  but your theme **omits** renders unstyled (terminal default colour), so
  start from a complete theme (copy `turbo` below) rather than a skeleton.
- A colour that is neither `#hex` nor a palette name resolves to *no colour*
  (the text keeps the terminal default) — typos fail silently.

### Truecolor

With `truecolor = true` (the default) hex colours are sent as 24-bit RGB.
With `truecolor = false` (settings or the options panel) every hex colour is
quantised to the xterm-256 colour cube at render time, so themes degrade
gracefully on 256-colour terminals. It is worth previewing your
theme in both modes.

## Style keys

These are the keys the UI looks up (`GetStyle("...")`):

| Key | Used for |
| --- | --- |
| `method_get`, `method_post`, `method_put`, `method_delete`, `method_patch`, `method_options`, `method_head` | HTTP method colouring in the Requests table and Detail view. Lookup is `method_<lowercase method>`, so an unlisted method (e.g. `PROPFIND` → `method_propfind`) renders unstyled |
| `status_2xx`, `status_3xx`, `status_4xx`, `status_5xx` | Status-code colouring (Requests, Detail) and Security view semantics (`2xx` = header present, `3xx` = optional header missing, `4xx` = header missing). 1xx responses render unstyled |
| `json_key`, `json_string`, `json_number`, `json_bool`, `json_null`, `json_bracket`, `json_unknown` | JSON viewer and diff body |
| `timing_dns`, `timing_connect`, `timing_ssl`, `timing_send`, `timing_wait`, `timing_receive`, `timing_blocked` | Waterfall bars in Timeline, Timings, and Detail |
| `diff_added`, `diff_removed`, `diff_modified`, `diff_changed` | Side-by-side diff lines |
| `title` | Mode titles and section headers |
| `status_line` | Footer/status text and the theme badge |
| `selection` | Highlighted selected row (tables/trees), selected search result, selected line in the JSON viewer (background colour; falls back to palette `bg` if the style has no `bg`) |
| `border`, `search_match` | Defined by all built-ins but not currently read by the UI — reserved |

Every style accepts `fg`, `bg` (colour values as above), `bold`, `italic`,
and `underline`.

## Palette keys

All four built-ins define these twelve names; styles and layout code
reference them directly:

| Key | Role |
| --- | --- |
| `bg` | Full-window background |
| `fg` | Default text, headers, borders, scroll thumb |
| `gray` | Secondary/dim text, scroll track, help hints |
| `warn` | Warning accents: `⚠` compression flag, confirm dialogs, summary/cache highlights |
| `green`, `red`, `yellow`, `cyan`, `magenta`, `blue`, `orange`, `unknown` | Semantic colours assigned to the style keys above (methods, statuses, JSON, timings) |

## Creating a new theme

The practical route is to copy an existing theme and edit it — below is the
complete **Turbo** theme, which is the built-in `internal/theme/turbo.go`
written out as a file. Saving this as `turbo.toml` and running
`harboring --theme turbo.toml page.har` reproduces the built-in exactly.

### The complete Turbo theme

```toml
[meta]
name = "Turbo"
author = "harboring"

[palette]
bg      = "#0000A8" # Turbo IDE royal blue
fg      = "#FFFFFF" # bright white text
green   = "#00FFC8" # aqua/turquoise (strings' cousin)
red     = "#FF4040" # error red
yellow  = "#FFFF6B" # keyword yellow
cyan    = "#00E0FF" # turquoise strings
magenta = "#FF80E0" # bright pink/violet
blue    = "#6BC8FF" # light blue
orange  = "#FFB060" # warm amber warnings
gray    = "#9A9A9A" # grey comments / chrome
unknown = "#FF80E0" # bright magenta
warn    = "#FFB060" # warm amber

[styles]
# HTTP methods
method_get     = { fg = "green" }
method_post    = { fg = "yellow" }
method_put     = { fg = "orange" }
method_delete  = { fg = "red" }
method_patch   = { fg = "magenta" }
method_options = { fg = "gray" }
method_head    = { fg = "gray" }

# Status classes
status_2xx = { fg = "green" }
status_3xx = { fg = "cyan" }
status_4xx = { fg = "yellow" }
status_5xx = { fg = "red" }

# JSON viewer
json_key      = { fg = "blue" }
json_string   = { fg = "cyan" }
json_number   = { fg = "magenta" }
json_bool     = { fg = "yellow" }
json_null     = { fg = "gray" }
json_bracket  = { fg = "fg" }           # palette cross-reference
json_unknown  = { fg = "unknown", italic = true }

# Timeline / waterfall phases
timing_dns     = { fg = "green" }
timing_connect = { fg = "cyan" }
timing_ssl     = { fg = "blue" }
timing_send    = { fg = "yellow" }
timing_wait    = { fg = "orange" }
timing_receive = { fg = "magenta" }
timing_blocked = { fg = "gray" }

# Chrome
title       = { fg = "yellow", bold = true }
status_line = { fg = "gray" }
border      = { fg = "cyan" }
search_match = { fg = "bg", bg = "yellow" }

# Diff (hex backgrounds — per-theme, not palette entries)
diff_added     = { fg = "green",  bg = "#003A28", bold = true }
diff_removed   = { fg = "red",    bg = "#480000", bold = true }
diff_modified  = { fg = "yellow", bg = "#3A3A00", bold = true }
diff_changed   = { fg = "red",    bg = "#500000", bold = true, underline = true }

# Selected row
selection = { bg = "#4040D0", bold = true }
```

Things worth noticing in this example:

- **Palette names carry the identity.** The DOS look lives almost entirely in
  `[palette]`; most styles just map a role onto a palette entry, so retinting
  the theme is usually a one-table edit.
- **Styles may mix references and hex.** `json_bracket = { fg = "fg" }`
  follows the text colour, while `diff_added` and `selection` use bespoke hex
  backgrounds because they are background fills, not palette roles.
- **Attributes are part of the design.** Turbo's diff lines are `bold`, the
  unknown JSON token is `italic`, and the selected row is `bold` on
  `#4040D0`.
- **Copy all style keys.** Turbo defines the full set; anything you drop falls
  back to unstyled rather than to another theme.

### Making it your default

1. Save the file, e.g. `~/.config/harboring/mytheme.toml`.
2. Try it: `harboring --theme ~/.config/harboring/mytheme.toml page.har`.
3. Make it the startup theme with either:
   - **save it as `~/.config/harboring/theme.toml`** — auto-loaded at every
     start (a `--theme` flag overrides it for that run), or
   - **set it in `settings.toml`**:

     ```toml
     theme = "/home/you/.config/harboring/mytheme.toml"
     ```

Two quirks to know:

- The title badge uses the engine's *registry* name, which file-loaded themes
  do not have — the badge renders empty (`❮ ❯`) for custom themes.
- Pressing `t` cycles back through the built-ins and saves that choice to
  `settings.toml`; while a `theme.toml` sits in place it still defines the
  next startup, so remove the file to fall back to your cycled choice.

## Adding a built-in theme (contributors)

Built-ins are compiled in as Go functions returning `*ThemeConfig`
(`internal/theme/turbo.go` is the template). To add one end to end:

1. Create `internal/theme/<name>.go` with `Default<Name>()`.
2. Register it in `internal/theme/engine.go` (`New()` — the struct field,
   `order` slice, and `byName` map — plus a `LoadStartup` case).
3. Add the name to `config.ThemeOptions` in `internal/config/settings.go`
   (options panel list) and to `applyTheme` in `internal/tui/app.go`
   (settings persistence).
4. Document it in the README theme section.
