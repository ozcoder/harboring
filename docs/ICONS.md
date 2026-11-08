# Icon vocabulary

Candidate glyphs for status markers in the TUI and reports, together with the
constraints that decide which glyphs are safe to use. Uses for individual
icons are not decided yet — this is the menu.

## Rules

A glyph is usable as an inline status marker only if it satisfies **all** of:

1. **Exactly 1 terminal cell.** Columns are aligned with `padToWidth`
   (`internal/tui/modes/requests.go`) using `lipgloss.Width`, and
   `TestRequestsIconKeepsColumnsAligned` asserts rows with and without an icon
   keep identical column positions. A 2-cell glyph shifts every column right.
2. **Not default-emoji-presentation.** Emoji presentation renders 2 cells in
   emoji-aware terminals regardless of East Asian Width. This also rules out
   appending `U+FE0F` (VS16) to any glyph — always write the bare codepoint.
3. **Present in DejaVu Sans Mono** (`fc-match monospace` here), or an accepted
   fallback. A missing glyph comes from another font at a different visual
   scale — or shows as tofu. Caveat: only the *cell width* is guaranteed by
   the terminal's width tables, never the glyph's design.

## Menu

| Meaning | Options | Notes |
| --- | --- | --- |
| Warning (current) | `⚠` | U+26A0 — used by the compression flag in the requests table |
| Warning alt | `‼` `‽` | `‽` (interrobang) is unusual; `‼` reads as "urgent" |
| Error / failed request | `✗` `✖` `✘` `☒` `⊗` | `✗` matches existing usage in `security.go` |
| Success / OK | `✓` `✔` `☑` | `✓` already used in `detail.go` |
| Info / note (not a warning) | `ℹ` `ⓘ` `!` `?` | `ℹ` already used in `detail.go`; `ℹ ⓘ` need font fallback |
| Cache / revalidate | `↻` `↺` `♻` `⇄` | `↻` reload fits cache semantics |
| Timing / slow | `⏱` `⏲` `⧗` | all need font fallback (absent from DejaVu Mono) |
| Redirect | `→` `↪` `⇄` `⇒` | |
| Third-party / tracker | `⊘` `◈` `⚑` `☆` | |
| Blocked / stopped | `⊘` `■` | avoid `⏸` `⏹` (2 cells, emoji presentation) |
| Priority change | `◐` `★` `●` `○` | avoid `▲ ▼` — already the sort indicator in every table |
| Security | `◆` `◈` `§` | avoid `🔒` `🛡` (width and font coverage) |
| Truncated / stripped | `✂` `…` `–` | `✂` fits the scrubbing metaphor |
| Sort indicator (in use) | `▲` `▼` | already used in requests/cache/options/help headers |
| Nav / scroll (in use) | `→` `←` `↑` `↓` `◀` `▶` | timings panning, diff scrollbars, tree markers |

## Avoid

| Glyph(s) | Why |
| --- | --- |
| `⚡` | East Asian Wide — 2 cells, despite being a non-emoji dingbat |
| `⏸` `⏹` `⌛` `⛔` `❗` `❕` | default emoji presentation — 2 cells in emoji-aware terminals |
| `🚫` `❌` | 2 cells (emoji) |
| Any emoji (`🔒` `🔗` `📊` `🐛` `📈` `🔍` `🔧` `📄` `📁`) | 2 cells; font-dependent colour and design |
| Any glyph with `U+FE0F` (e.g. `⚠️`, `ℹ️`) | VS16 switches to emoji presentation — 2 cells |
| `！`(fullwidth) and anything East Asian `W`/`F` | 2 cells |
| `▲` `▼` for a *new* meaning | collision with the sort indicator |

## Font coverage (measured)

Checked against DejaVu Sans Mono (the `monospace` default here):

- **Covered:** `⚠ ✓ ✔ ✗ ✖ ✘ ☑ ☒ ★ ☆ ● ○ ◐ ◉ ◆ ▶ ■ ▪ ↻ ↺ ♻ ⇄ → ↪ ⇒ ⊘ ⊗ ◈ ⚑ ✂ … – • ▸ ❯ — ™ ° §`
- **Missing (terminal font fallback required):** `ℹ ⓘ ⏱ ⏲ ⧗ ⌛ ⏸ ⏹ ① ☰`
- Liberation Mono and Noto Sans Mono cover *less* (both lack `⚠ ✓ ✗ ★ ↻`),
  so don't assume another mono font is an improvement.

## How much blank space surrounds a glyph

Measured in DejaVu Sans Mono (advance 1233, em 2048, line height 2384 units):

| Glyph | ink width / cell | ink height / line |
| --- | --- | --- |
| `○` `●` `◐` `◉` `◆` `▶` `◈` | 99% | ~51% |
| `✘` | 92% | 56% |
| `⚠` | 94% | 45% |
| `♻` `★` | 91% | 45% |
| `✔` | 88% | 41% |
| `✓` | 75% | 40% |
| `▪` `▸` | 64% | 33% |
| `•` | 48% | 25% |
| `!` | 16% | 63% |

**Horizontal blank space is already near zero** for most dingbats — the font
draws them edge to edge in the cell. The visible blank space is *vertical*
(top margin alone is ~750–1000 units, i.e. ~1/3 of the line box), and that
leading belongs to the font's ascent/descent metrics, not to the glyph.

## Can icons be zoomed while staying one cell wide?

Not through any terminal or Unicode mechanism:

- Glyph outlines are baked into the font at a fixed size (units per em).
  Terminals place the em box into the fixed cell and apply **no per-glyph
  scaling**. There is no escape sequence that zooms a glyph.
- The emoji mechanism does render "bigger/bolder", but the moment the
  terminal treats the character as emoji presentation it becomes **2 cells**,
  which breaks row alignment.
- Enlarging the font size in the terminal scales the cell and the blank
  margins proportionally — the ink-to-blank ratio never changes.

What actually works, in order of effort:

1. **Pick glyphs that already fill the cell** — see the table above:
   `○ ● ◐ ◉ ◆ ▶ ◈` are at 99% cell width already.
2. **Prefer filled/heavy variants of the same idea** — more ink in the same
   cell: `✔` is 88% wide vs `✓` at 75%, `●` vs `○`, `◼` vs `▫`, `✖` vs `✗`.
3. **Render in bold** — thicker strokes with no layout change; the requests
   table already does this for `⚠` (`Bold(true)` in `requests.go`).
4. **A patched icon font** (the Nerd Fonts approach): redraw the chosen
   codepoints larger inside the cell. This is the only true "zoom", but every
   user must install that font, which gives up the "renders in any terminal"
   property of the glyphs above.

A note on consistency: fallback glyphs (e.g. `⏱` via Symbola) may be drawn at
a different scale than DejaVu's dingbats, so mixing fallback and native
glyphs in one column can look uneven even though alignment stays correct.
