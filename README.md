# harboring

**H**ttp **AR**chive **B**... **O**... **R**eader **I**n **G**o — a terminal UI for inspecting HAR files.

What information is your HAR file harboring?

Built with [Bubble Tea](https://github.com/charmbracelet/bubbletea) and [Lip Gloss](https://github.com/charmbracelet/lipgloss).

## Features

- **Tree view** — full HAR file as a collapsible, colourised JSON tree
- **Request list** — tabular view with colour-coded methods and status codes, resource-type filtering (`f`) and sortable columns
- **Timings chart** — horizontal timing bars with configurable columns (DNS, Connect, SSL, Send, Wait, Receive) and a resizable bar graph area with horizontal scrolling (`+`/`-`, `←`/`→`)
- **Timeline view** — all requests plotted on one shared time axis by start time, revealing parallel vs. serialised loading and queueing/blocking
- **Summary view** — aggregate overview: request counts, total size/time, and method/status breakdowns
- **Cache analysis** — per-entry cache-header analysis (max-age, ETag, no-store, etc.) with warnings
- **Security headers** — required/optional security header presence per domain, with detail popovers
- **Search** — full-text search across all HAR fields with next/previous navigation
- **Detail popover** — per-request breakdown with timing bars **and the full request and response headers**
- **Content scrubbing** — strip response bodies for safe sharing
- **Compression support** — reads and writes `.har.gz` (gzip) and `.har.bz2` (bzip2) files transparently, detected by magic bytes or file extension
- **Theme engine** — dark/light/spectrum/turbo built-in themes, custom themes via TOML

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
harboring <file.har>              Open TUI viewer (default)
harboring --view <file.har>       Open TUI viewer
harboring --dump <file.har>       Dump formatted JSON to stdout
harboring --scrub <file.har>      Scrub content sections
harboring --shrink <file.har>     Minify JSON (remove whitespace)
harboring --expand <file.har>     Pretty-print JSON
harboring --stats <file.har>      Show summary statistics
harboring --perf <file.har>       Generate performance report
harboring --cache <file.har>      Show cache summary
harboring --security <file.har>   Show security headers analysis
harboring --trackers <file.har>   Show third-party tracker analysis
harboring --priority <file.har>   Show requests with changed resource priority
harboring --fields <fields> <file.har>  Output HAR with only the specified fields
harboring --diff <left.har> <right.har> Compare two HAR files side by side
harboring --validate <file.har>   Validate HAR structure
harboring --list themes           List available built-in themes
harboring --list trackers         List known tracker domains
harboring --version               Show version
harboring --help                  Show this help
```

### CLI Examples

```bash
# View a HAR file in the TUI
harboring recording.har

# Dump formatted JSON
harboring --dump recording.har > pretty.json

# Scrub response bodies into a new file
harboring --scrub recording.har -o scrubbed.har

# Quick stats
harboring --stats recording.har

# Validate structure
harboring --validate recording.har

# Minify / pretty-print JSON
harboring --shrink recording.har -o min.har
harboring --expand recording.har -o pretty.har

# Performance report (TTFB percentiles, by type/domain, slowest/largest)
harboring --perf recording.har

# Cache summary (good / short / no-cache / over-cached breakdown)
harboring --cache recording.har

# Security headers analysis (per-domain required/optional headers)
harboring --security recording.har

# Third-party tracker analysis
harboring --trackers recording.har

# Requests whose resource priority changed (_initialPriority vs _priority)
harboring --priority recording.har

# Show only certain fields as a new HAR
harboring --fields request.url,timings.wait recording.har -o slim.har

# Compare two HAR files side by side in the TUI
harboring --diff left.har right.har

# List available themes / known trackers
harboring --list themes
harboring --list trackers

# Add custom tracker domains for a tracker run
harboring --trackers recording.har --trackers-file trackers.toml

# Show version
harboring --version
```

### Markdown reports

`--stats`, `--perf`, `--cache`, `--security`, `--trackers`, and `--priority`
print plain text by default. Pass `--markdown` (or `-m`) to emit a clean
GitHub-flavored Markdown version instead — handy for pasting into issues,
wikis, or PR descriptions:

```bash
harboring --perf recording.har --markdown > perf.md
harboring --cache recording.har -m
harboring --security recording.har --markdown | tee security.md
harboring --priority recording.har --markdown

### Flags

| Flag                     | Description                                                 |
| ------------------------ | ----------------------------------------------------------- |
| `--view`                 | Open the TUI viewer (default when no mode is given)         |
| `--dump`                 | Dump formatted JSON to stdout                               |
| `--scrub`                | Scrub content sections                                      |
| `--shrink`               | Minify JSON (remove whitespace)                             |
| `--expand`               | Pretty-print JSON                                           |
| `--stats`                | Show summary statistics                                     |
| `--perf`                 | Generate performance report                                 |
| `--cache`                | Show cache summary                                          |
| `--security`             | Show security headers analysis                              |
| `--trackers`             | Show third-party tracker analysis                           |
| `--priority`             | Show requests whose resource priority changed (`_initialPriority` vs `_priority`) |
| `--validate`             | Validate HAR structure                                      |
| `--theme <name\|path>`   | Override theme (`dark`, `light`, `spectrum`, `turbo`, or path to TOML file) |
| `--list <themes\|trackers>` | List built-in themes or known tracker domains and exit      |
| `--trackers-file <path>` | Load additional tracker domains from a TOML file (used with `--trackers`) |
| `--markdown`, `-m`       | Render reports as Markdown                                  |
| `--json`, `-j`           | Render reports as JSON                                      |
| `--version`, `-v`        | Print the harboring version and exit                        |
| `--help`, `-h`           | Print help and exit                                         |
| `-o <file>`              | Output file (for `--scrub`/`--shrink`/`--expand` and report modes) |

At most one mode flag (`--view` … `--validate`) may be given; combine
transformations by piping, e.g. `harboring --scrub in.har | harboring --shrink`.

### Custom tracker lists

`harboring --trackers file.har --trackers-file trackers.toml` augments the
built-in tracker list with your own domains for that run. The file is TOML with
an optional `category` (defaults to `other`):

```toml
[[trackers]]
domain = "pixel.company.example"
category = "analytics"

[[trackers]]
domain = "ads.company.example"
```

`harboring --list trackers` prints the resulting known domains grouped by
category; `harboring --list themes` prints the available themes.

### Priority changes

`harboring --priority file.har` lists every request whose resource priority
changed after it was initially queued. Chrome records the initial priority in
`_initialPriority` and the final one in `_priority`; when the two differ the
request made it in the report. Each row shows the URL, method, HTTP version,
status, and the `initial → final` priority pair that changed:

```bash
harboring --priority file.har
harboring --priority file.har --markdown
harboring --priority file.har --json
```

### Pipelines

The non-interactive modes (`--dump`, `--scrub`, `--shrink`, `--expand`,
`--stats`, `--perf`, `--cache`, `--security`, `--trackers`, `--priority`,
`--validate`, `--fields`) accept input from a file path, an explicit
`-` for stdin, or piped input with no argument. They write to stdout unless
`-o` is given, so they compose naturally in a shell pipeline. Compressed
streams (`.gz`/`.bz2`) are detected by their magic bytes, so you can pipe
compressed data directly.

```bash
# Scrub a compressed HAR and count the lines of output
cat large.har.bz2 | harboring --scrub | wc -l

# Generate a performance report from a gzipped HAR into a file
cat large.har.gz | harboring --perf > large.perf.txt

# Validate a stream and check the result
cat recording.har | harboring --validate

# Explicit stdin marker
harboring --stats - < recording.har

# Chain transforms: scrub, then minify to a file
cat recording.har.gz | harboring --scrub | harboring --shrink -o clean.har
```

The TUI viewer (`--view`) requires a file path and does not read from stdin.

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
default_mode = "tree"       # tree | requests | timings | summary | cache
compact_mode = false
```

## TUI Keyboard Shortcuts

### Global

| Key                               | Action                                                                |
| --------------------------------- | --------------------------------------------------------------------- |
| `1` / `2` / `3` / `4` / `5` / `6` / `7` | Switch to Tree / Requests / Timings / Summary / Cache / Security / Timeline |
| `/`                               | Open search                                                           |
| `?`                               | Show help                                                             |
| `o`                               | Open options                                                          |
| `t`                               | Cycle through themes (dark / light / spectrum / turbo)                |
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

### Timings Mode

| Key                    | Action                      |
| ---------------------- | --------------------------- |
| ↑ / ↓ or `j` / `k`     | Move down / up              |
| `PgDn` / `Space` / `f` | Page down                   |
| `PgUp` / `b`           | Page up                     |
| `g` / `G`              | First / last entry          |
| `Enter`                | Open timing detail popover  |
| `+` / `-`              | Widen / narrow the bar graph area (a horizontal scrollbar appears when it exceeds the screen) |
| `→` / `←` or `l` / `h` | Scroll the bar graph area horizontally |

### Summary Mode

| Key                | Action         |
| ------------------ | -------------- |
| ↑ / ↓ or `j` / `k` | Scroll content |

### Timeline Mode

| Key                       | Action                     |
| ------------------------- | -------------------------- |
| ↑ / ↓ or `j` / `k`        | Move down / up             |
| `PgDn` / `Space` / `f`    | Page down                  |
| `PgUp` / `b`              | Page up                    |
| `g` / `G`                 | First / last entry         |
| `Enter`                   | Open detail popover        |
| `+` / `-`                 | Widen / narrow the timeline area (a horizontal scrollbar appears when it exceeds the screen) |
| `→` / `←` or `l` / `h`    | Scroll the timeline area horizontally |

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

Harboring ships with four built-in themes: **Dark**, **Light**, **Spectrum** (a retro ZX Spectrum palette), and **Turbo** (a Borland Turbo Pascal IDE palette). Cycle through them with `t` in the TUI, or pass `--theme dark` / `--theme light` / `--theme spectrum` / `--theme turbo` on the command line.

### Custom Themes

Place a TOML file at `~/.config/harboring/theme.toml` and it is loaded as the theme at startup (a `--theme` flag still overrides it for that run). You can also load a file ad-hoc with `--theme <path>`, or set `theme = "/path/to/theme.toml"` in `settings.toml`. See [docs/themes.md](docs/themes.md) for the full format, the reference of every style and palette key, and a complete annotated example (the built-in `turbo` theme). The shape of a theme file:

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

## Project Structure

```
cmd/harboring/           CLI entry point
internal/
  har/                   Custom HAR parser (no external HAR libraries)
  tui/                   Bubble Tea TUI
    modes/               Tree, requests, timings, summary, cache, security views
  theme/                 Theme engine (built-in dark/light/spectrum/turbo, TOML loading)
  filter/                Method/status/URL filtering
docs/                    Theme and icon references (themes.md, ICONS.md)
```

## License

See [LICENSE](LICENSE).
