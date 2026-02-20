package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/kkz6/launchctl/internal/notify"
)

var (
	breadcrumbSep    = lipgloss.NewStyle().Foreground(Muted).Render(" > ")
	breadcrumbItem   = lipgloss.NewStyle().Foreground(Slate)
	breadcrumbActive = lipgloss.NewStyle().Foreground(Indigo).Bold(true)
	dividerStyle     = lipgloss.NewStyle().Foreground(DarkSlate)
	promptStyle      = lipgloss.NewStyle().Foreground(Muted).Italic(true)
)

func ClearScreen() {
	fmt.Print("\033[H\033[2J\033[3J")
}

func PrintHeader(parts ...string) {
	fmt.Println()
	var b strings.Builder
	for i, p := range parts {
		if i > 0 {
			b.WriteString(breadcrumbSep)
		}
		if i == len(parts)-1 {
			b.WriteString(breadcrumbActive.Render(p))
		} else {
			b.WriteString(breadcrumbItem.Render(p))
		}
	}
	fmt.Println(b.String())
	PrintDivider()

	if bar := notify.Render(); bar != "" {
		fmt.Println(bar)
	}

	fmt.Println()
}

type waitModel struct{}

func (m waitModel) Init() tea.Cmd                           { return nil }
func (m waitModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if _, ok := msg.(tea.KeyMsg); ok {
		return m, tea.Quit
	}
	return m, nil
}
func (m waitModel) View() string {
	return "\n" + promptStyle.Render("Press Enter to continue...")
}

func WaitForEnter() {
	tea.NewProgram(waitModel{}).Run()
}

func PrintDivider() {
	fmt.Println(dividerStyle.Render(strings.Repeat("─", 50)))
}

func PrintHint(text string) {
	fmt.Println(promptStyle.Render(text))
}
