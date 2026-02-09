package splash

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	logoStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#4ade80"))

	versionStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#64748b"))
)

const logo = "" +
	" _                   _       _   _ \n" +
	"| |__ _ _  _ _ _  __| |_  __| |_| |\n" +
	"| / _` | || | ' \\/ _| ' \\/ _|  _| |\n" +
	"|_\\__,_|\\_,_|_||_\\__|_||_\\__|\\__|_|"

func Render(version string) string {
	lines := strings.Split(logo, "\n")
	var b strings.Builder

	for i, line := range lines {
		b.WriteString(logoStyle.Render(line))
		if i == len(lines)-1 {
			b.WriteString("  ")
			b.WriteString(versionStyle.Render("v" + version))
		}
		b.WriteString("\n")
	}

	return b.String()
}
