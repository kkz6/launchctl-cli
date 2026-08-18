package output

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"github.com/kkz6/launchctl/internal/tui"
)

var (
	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(tui.OnAccent).
			Background(tui.Indigo).
			Padding(0, 1)

	cellStyle = lipgloss.NewStyle().Padding(0, 1)

	borderStyle = lipgloss.NewStyle().Foreground(tui.DarkSlate)
)

func RenderTable(title string, headers []string, rows [][]string) {
	if len(rows) == 0 {
		fmt.Println(tui.Dim.Render("No results found."))
		return
	}

	fmt.Println()
	fmt.Println(tui.Title.Render(title))
	fmt.Println()

	t := table.New().
		Border(lipgloss.RoundedBorder()).
		BorderStyle(borderStyle).
		Headers(headers...).
		Rows(rows...).
		StyleFunc(func(row, col int) lipgloss.Style {
			if row == table.HeaderRow {
				return headerStyle
			}
			return cellStyle
		})

	fmt.Println(t)

	fmt.Println(tui.Dim.Render(fmt.Sprintf("%d items", len(rows))))
	fmt.Println()
}

func StatusDot(status string) string {
	style := tui.StatusStyle(status)
	return style.Render("●") + " " + style.Render(status)
}
