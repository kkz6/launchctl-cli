package splash

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// 3-row uppercase glyphs using Unicode half-block characters.
// Follows the Charm Crush wordmark style.
var glyphs = map[rune][3]string{
	'L': {"█     ", "█     ", "▀▀▀▀▀ "},
	'A': {"▄▀▀▀▄ ", "█▀▀▀█ ", "▀   ▀ "},
	'U': {"█   █ ", "█   █ ", " ▀▀▀  "},
	'N': {"█▄  █ ", "█ ▀▄█ ", "▀   ▀ "},
	'C': {"▄▀▀▀▀ ", "█     ", " ▀▀▀▀ "},
	'H': {"█   █ ", "█▀▀▀█ ", "▀   ▀ "},
	'T': {"▀▀█▀▀ ", "  █   ", "  ▀   "},
}

func lerp(a, b, t float64) float64 {
	return a + (b-a)*t
}

func gradientColor(r1, g1, b1, r2, g2, b2 int, t float64) lipgloss.Color {
	r := int(lerp(float64(r1), float64(r2), t))
	g := int(lerp(float64(g1), float64(g2), t))
	b := int(lerp(float64(b1), float64(b2), t))

	return lipgloss.Color(fmt.Sprintf("#%02x%02x%02x", r, g, b))
}

func Render(version string) string {
	isDark := lipgloss.HasDarkBackground()

	// Gradient: indigo → violet
	var r1, g1, b1, r2, g2, b2 int
	if isDark {
		r1, g1, b1 = 0x81, 0x8C, 0xF8 // #818CF8
		r2, g2, b2 = 0xC0, 0x84, 0xFC // #C084FC
	} else {
		r1, g1, b1 = 0x4F, 0x46, 0xE5 // #4F46E5
		r2, g2, b2 = 0x7C, 0x3A, 0xED // #7C3AED
	}

	word := "LAUNCHCTL"

	var rows [3]string
	for row := 0; row < 3; row++ {
		var parts []string
		for _, ch := range word {
			if g, ok := glyphs[ch]; ok {
				parts = append(parts, g[row])
			}
		}
		rows[row] = strings.Join(parts, "")
	}

	var b strings.Builder
	b.WriteString("\n\n\n")

	for _, row := range rows {
		runes := []rune(row)
		width := len(runes)
		b.WriteString("  ")

		for i, r := range runes {
			if r == ' ' {
				b.WriteRune(' ')
				continue
			}

			t := 0.0
			if width > 1 {
				t = float64(i) / float64(width-1)
			}

			color := gradientColor(r1, g1, b1, r2, g2, b2, t)
			style := lipgloss.NewStyle().
				Bold(true).
				Foreground(color)
			b.WriteString(style.Render(string(r)))
		}

		b.WriteString("\n")
	}

	b.WriteString("  Server management & deployment toolkit")
	b.WriteString("\n\n")
	b.WriteString("  launchctl.io  v" + version)
	b.WriteString("\n")

	return b.String()
}
