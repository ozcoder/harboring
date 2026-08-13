# harboring

**H**ttp **AR**chive **B**... **O**... **R**eader **I**n **G**o — a terminal UI for inspecting HAR files.

Built with [Bubble Tea](https://github.com/charmbracelet/bubbletea) and [Lip Gloss](https://github.com/charmbracelet/lipgloss).

## Features

- **Tree view** — full HAR file as a collapsible, colourised JSON tree
- **Request list** — tabular view with colour-coded methods and status codes, resource-type filtering (`f`) and sortable columns
- **Waterfall chart** — horizontal timing bars with configurable columns (DNS, Connect, SSL, Send, Wait, Receive) and zoom (`+`/`-`)
- **Summary view** — aggregate overview: request counts, total size/time, and method/status breakdowns
- **Cache analysis** — per-entry cache-header analysis (max-age, ETag, no-store, etc.) with warnings
- **Security headers** — required/optional security header presence per domain, with detail popovers
- **Search** — full-text search across all HAR fields with next/previous navigation
- **Detail popover** — per-request breakdown with timing bars **and the full request and response headers**
- **Content scrubbing** — strip response bodies for safe sharing
- **Gzip support** — reads `.har.gz` files transparently
- **Theme engine** — dark/light built-in themes, custom themes via TOML

## Installation

Requires Go 1.22+.

```
go install github.com/ozcoder/harboring/cmd/harboring@latest
```

Or build from source:

```
git clone https://github.com/harboring/harboring.git
cd harboring
go build -o harboring ./cmd/harboring/
```

## Usage

```
harboring <file.har>              Open TUI viewer
harboring view <file.har>         Open TUI viewer
harboring dump <file.har>         Dump formatted JSON to stdout
harboring scrub <file.har> [-o]   Scrub content sections
harboring shrink <file.har> [-o]  Minify JSON (remove whitespace)
harboring expand <file.har> [-o]  Pretty-print JSON
harboring stats <file.har>        Show summary statistics
harboring perf <file.har>         Generate performance report
harboring validate <file.har>     Validate HAR structure
harboring --version               Show version
harboring help                    Show this help
```

### CLI Examples

```bash
# View a HAR file in the TUI
harboring recording.har

# Dump formatted JSON
harboring dump recording.har > pretty.json

# Scrub response bodies into a new file
harboring scrub recording.har -o scrubbed.har

# Quick stats
harboring stats recording.har

# Validate structure
harboring validate recording.har

# Minify / pretty-print JSON
harboring shrink recording.har -o min.har
harboring expand recording.har -o pretty.har

# Performance report (TTFB percentiles, by type/domain, slowest/largest)
harboring perf recording.har

# Show version
harboring --version
```

### Flags

| Flag                     | Description                                                 |
| ------------------------ | ----------------------------------------------------------- |
| `--theme <name\|path>`   | Override theme (`dark`, `light`, or path to TOML file)      |
| `--scrub`                | Scrub content sections (for `view`/`dump`)                  |
| `--shrink`               | Remove whitespace when dumping (for `view`/`dump`)          |
| `--version`              | Print the harboring version and exit                        |
| `-o <file>`              | Output file (for `scrub`/`shrink`/`expand` commands)        |

## Configuration

Options changed in the TUI (**`o`** → Options) are saved automatically and
reloaded on the next launch. Settings are stored in a TOML file at:

- **Linux**: `~/.config/harboring/settings.toml`
- **macOS**: `~/Library/Application Support/harboring/settings.toml`
- **Windows**: `%AppData%\harboring\settings.toml`

The file holds the persisted preferences:

```toml
theme = "dark"
truecolor = true
graph_symbol = "block"      # block | braille | ascii
default_mode = "tree"       # tree | requests | waterfall | summary | cache
compact_mode = false
```

## TUI Keyboard Shortcuts

### Global

| Key                               | Action                                                                |
| --------------------------------- | --------------------------------------------------------------------- |
| `1` / `2` / `3` / `4` / `5` / `6` | Switch to Tree / Requests / Waterfall / Summary / Cache / Security    |
| `/`                               | Open search                                                           |
| `?`                               | Show help                                                             |
| `o`                               | Open options                                                          |
| `t`                               | Toggle dark/light theme                                               |
| `s`                               | Scrub content sections (or cycle sort in Requests mode)               |
| `Ctrl+S`                          | Save file                                                             |
| `q` / `Ctrl+C`                    | Quit                                                                  |

### Tree Mode

| Key                | Action                  |
| ------------------ | ----------------------- |
| ↑ / ↓ or `j` / `k` | Move down / up          |
| ← / → or `h` / `l` | Collapse / expand node  |
| `Enter`            | Toggle expand/collapse  |
| `E`                | Expand all              |
| `c`                | Collapse all            |

### Requests Mode

| Key                    | Action                            |
| ---------------------- | --------------------------------- |
| ↑ / ↓ or `j` / `k`     | Move down / up                    |
| `PgDn` / `Space` / `f` | Page down                         |
| `PgUp` / `b`           | Page up                           |
| `g` / `G`              | First / last entry                |
| `Enter`                | Open detail popover               |
| `f`                    | Toggle resource-type filter       |
| `s`                    | Cycle sort: Size / Time / URL     |
| `S`                    | Toggle sort direction             |

### Waterfall Mode

| Key                    | Action                      |
| ---------------------- | --------------------------- |
| ↑ / ↓ or `j` / `k`     | Move down / up              |
| `PgDn` / `Space` / `f` | Page down                   |
| `PgUp` / `b`           | Page up                     |
| `g` / `G`              | First / last entry          |
| `Enter`                | Open timing detail popover  |
| `+` / `-`              | Zoom chart in / out         |

### Summary Mode

| Key                | Action         |
| ------------------ | -------------- |
| ↑ / ↓ or `j` / `k` | Scroll content |

### Cache Mode

| Key                    | Action                  |
| ---------------------- | ----------------------- |
| `Tab` / `Shift+Tab`    | Switch between sections |
| ↑ / ↓ or `j` / `k`     | Scroll through list     |
| `g` / `G`              | Jump to top / bottom    |
| `Enter`                | Open detail popover     |

### Security Mode

| Key                    | Action                      |
| ---------------------- | --------------------------- |
| ↑ / ↓ or `j` / `k`     | Move down / up              |
| `PgDn` / `Space` / `f` | Page down                   |
| `PgUp` / `b`           | Page up                     |
| `g` / `G`              | First / last entry          |
| `Enter`                | Open header detail popover  |

### Detail Popover

| Key                | Action          |
| ------------------ | --------------- |
| ↑ / ↓ or `j` / `k` | Scroll content  |
| `Esc` / `q`        | Close popover   |

### Search

| Key          | Action                |
| ------------ | --------------------- |
| `/`          | Open search prompt    |
| `n` / `N`    | Next / previous match |
| `Esc`        | Close search          |

## Themes

Harboring ships with two built-in themes (Catppuccin Mocha and Catppuccin Latte). Toggle between them with `t` in the TUI, or pass `--theme dark` / `--theme light` on the command line.

### Custom Themes

Place a TOML file at `~/.config/harboring/theme.toml` to load a custom theme on startup. See `themes/dark.toml` for the format:

```toml
[meta]
name = "My Theme"
author = "you"

[palette]
bg = "#1e1e2e"
fg = "#cdd6f4"
green = "#a6e3a1"
red = "#f38ba8"
# ...

[styles]
method_get = { fg = "green" }
method_post = { fg = "yellow" }
status_2xx = { fg = "green" }
json_key = { fg = "blue" }
# ...
```

Palette colours can be referenced by name in styles (e.g. `fg = "green"`) or specified as hex (e.g. `fg = "#a6e3a1"`).

## Project Structure

```
cmd/harboring/           CLI entry point
internal/
  har/                   Custom HAR parser (no external HAR libraries)
  tui/                   Bubble Tea TUI
    modes/               Tree, requests, waterfall, summary, cache, security views
  theme/                 Theme engine (TOML loading, dark/light)
  filter/                Method/status/URL filtering
themes/                  Built-in theme TOML files
```

## License

See [LICENSE](LICENSE).
