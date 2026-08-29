package modes

import (
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/harboring/harboring/internal/theme"
)

// renderHScrollbar draws a horizontal scrollbar over a window of
// [scroll, scroll+width) cells within a virtual area total wide. It returns ""
// when everything fits, so callers can render a blank row instead.
func renderHScrollbar(width, total, scroll int, trackStyle, thumbStyle lipgloss.Style) string {
	if total <= width {
		return ""
	}
	maxScroll := total - width
	thumbLen := width * width / total
	if thumbLen < 1 {
		thumbLen = 1
	}
	if thumbLen > width {
		thumbLen = width
	}
	thumbPos := scroll * (width - thumbLen) / maxScroll

	var sb strings.Builder
	for i := 0; i < width; i++ {
		if i >= thumbPos && i < thumbPos+thumbLen {
			sb.WriteString(thumbStyle.Render("\u2588"))
		} else {
			sb.WriteString(trackStyle.Render("\u2500"))
		}
	}
	return sb.String()
}

// renderHScrollbarWithArrows draws a horizontal scrollbar with ← and → arrows
// at each end, and the thumb in the middle. The arrows indicate that the
// content can be scrolled horizontally.
func renderHScrollbarWithArrows(width, total, scroll int, trackStyle, thumbStyle lipgloss.Style) string {
	// Reserve 2 chars for arrows (one at each end)
	barWidth := width - 2
	if barWidth < 3 {
		barWidth = 3
	}

	maxScroll := total - barWidth
	if maxScroll < 0 {
		maxScroll = 0
	}

	// Calculate thumb position and size
	thumbLen := barWidth * barWidth / total
	if thumbLen < 1 {
		thumbLen = 1
	}
	if thumbLen > barWidth {
		thumbLen = barWidth
	}
	var thumbPos int
	if maxScroll > 0 {
		thumbPos = scroll * (barWidth - thumbLen) / maxScroll
	}

	var sb strings.Builder

	// Left arrow
	sb.WriteString(thumbStyle.Render("◀"))

	// Scrollbar track
	for i := 0; i < barWidth; i++ {
		if i >= thumbPos && i < thumbPos+thumbLen {
			sb.WriteString(thumbStyle.Render("\u2588"))
		} else {
			sb.WriteString(trackStyle.Render("\u2500"))
		}
	}

	// Right arrow
	sb.WriteString(thumbStyle.Render("▶"))

	return sb.String()
}

// hScrollbarStyles returns (track, thumb) styles. The thumb is kept muted so
// it reads as a scroll indicator without competing with the request text.
func hScrollbarStyles(th *theme.Engine) (lipgloss.Style, lipgloss.Style) {
	track := lipgloss.NewStyle().Foreground(lipgloss.Color(th.GetGray()))
	thumb := lipgloss.NewStyle().Foreground(lipgloss.Color(th.GetFg()))
	return track, thumb
}
