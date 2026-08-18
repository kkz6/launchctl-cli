package tui

import (
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
)

func FormTheme() *huh.Theme {
	t := huh.ThemeBase()

	t.Focused.Base = lipgloss.NewStyle().
		PaddingLeft(1)

	t.Focused.Title = lipgloss.NewStyle().
		Bold(true).
		Foreground(Indigo).
		MarginBottom(1)

	t.Focused.Description = lipgloss.NewStyle().
		Foreground(Slate).
		MarginBottom(1)

	t.Focused.TextInput.Prompt = lipgloss.NewStyle().
		Foreground(Indigo)

	t.Focused.TextInput.Text = lipgloss.NewStyle().
		Foreground(White)

	t.Focused.TextInput.Placeholder = lipgloss.NewStyle().
		Foreground(Muted)

	t.Focused.TextInput.Cursor = lipgloss.NewStyle().
		Foreground(Indigo)

	t.Focused.SelectSelector = lipgloss.NewStyle().
		Foreground(Indigo)

	t.Focused.Option = lipgloss.NewStyle().
		Foreground(White)

	t.Focused.SelectedOption = lipgloss.NewStyle().
		Foreground(Indigo).
		Bold(true)

	t.Focused.FocusedButton = lipgloss.NewStyle().
		Bold(true).
		Foreground(OnAccent).
		Background(Indigo).
		Padding(0, 2)

	t.Focused.BlurredButton = lipgloss.NewStyle().
		Foreground(Muted).
		Padding(0, 2)

	t.Blurred.Base = lipgloss.NewStyle().
		PaddingLeft(1)

	t.Blurred.Title = lipgloss.NewStyle().
		Foreground(Muted)

	t.Blurred.TextInput.Text = lipgloss.NewStyle().
		Foreground(Muted)

	t.Blurred.TextInput.Placeholder = lipgloss.NewStyle().
		Foreground(DarkSlate)

	return t
}
