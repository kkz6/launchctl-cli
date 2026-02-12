package splash

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Lowercase block-letter glyphs (variable width, 5 rows tall).
// Ascenders (l, h, t) use all 5 rows; body letters use rows 1-4.
var glyphs = map[rune][5]string{
	'l': {"██", "██", "██", "██", "██"},
	'a': {"     ", " ███ ", "█  ██", " ████", " ███ "},
	'u': {"      ", "██  ██", "██  ██", "██  ██", " ████ "},
	'n': {"      ", "██████", "██  ██", "██  ██", "██  ██"},
	'c': {"    ", " ███", "███ ", "███ ", " ███"},
	'h': {"██    ", "██████", "██  ██", "██  ██", "██  ██"},
	't': {" ██ ", "████", " ██ ", " ██ ", " ██ "},
}

var (
	// Dark terminal: light green gradient
	darkRowColors = []string{
		"#86efac",
		"#4ade80",
		"#22c55e",
		"#16a34a",
		"#15803d",
	}

	// Light terminal: dark green gradient
	lightRowColors = []string{
		"#15803d",
		"#16a34a",
		"#166534",
		"#14532d",
		"#052e16",
	}
)

func Render(version string) string {
	isDark := lipgloss.HasDarkBackground()

	rowColors := darkRowColors
	if !isDark {
		rowColors = lightRowColors
	}

	taglineFg := lipgloss.Color("#0f172a")
	taglineBg := lipgloss.Color("#4ade80")
	versionFg := lipgloss.Color("#64748b")
	subtitleFg := lipgloss.Color("#94a3b8")

	if !isDark {
		taglineFg = lipgloss.Color("#f8fafc")
		taglineBg = lipgloss.Color("#15803d")
		versionFg = lipgloss.Color("#475569")
		subtitleFg = lipgloss.Color("#475569")
	}

	taglineStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(taglineFg).
		Background(taglineBg).
		Padding(0, 1)

	versionStyle := lipgloss.NewStyle().
		Foreground(versionFg)

	subtitleStyle := lipgloss.NewStyle().
		Foreground(subtitleFg).
		Italic(true)

	word := "launchctl"

	var rows [5]string
	for row := 0; row < 5; row++ {
		var parts []string
		for _, ch := range word {
			if g, ok := glyphs[ch]; ok {
				parts = append(parts, g[row])
			}
		}
		rows[row] = strings.Join(parts, " ")
	}

	var b strings.Builder
	b.WriteString("\n")

	for i, row := range rows {
		style := lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color(rowColors[i]))
		b.WriteString("  ")
		b.WriteString(style.Render(row))
		b.WriteString("\n")
	}

	b.WriteString("  ")
	b.WriteString(subtitleStyle.Render("Server management & deployment toolkit"))
	b.WriteString("\n\n")
	b.WriteString("  ")
	b.WriteString(taglineStyle.Render("launchctl.io"))
	b.WriteString("  ")
	b.WriteString(versionStyle.Render("v" + version))
	b.WriteString("\n")

	return b.String()
}
