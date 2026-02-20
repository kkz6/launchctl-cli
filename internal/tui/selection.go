package tui

import (
	"fmt"
	"io"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	itemNormalStyle = lipgloss.NewStyle().
			PaddingLeft(4).
			Foreground(Slate)

	itemSelectedStyle = lipgloss.NewStyle().
				PaddingLeft(2).
				Foreground(Indigo).
				Bold(true)

	itemNumberStyle = lipgloss.NewStyle().
			Foreground(Muted)

	itemNumberSelectedStyle = lipgloss.NewStyle().
				Foreground(Indigo)

	actionNormalStyle = lipgloss.NewStyle().
				PaddingLeft(4).
				Foreground(DarkSlate)

	actionSelectedStyle = lipgloss.NewStyle().
				PaddingLeft(2).
				Foreground(Slate).
				Bold(true)
)

type simpleItem struct {
	title    string
	index    int
	isAction bool
}

func (i simpleItem) Title() string       { return i.title }
func (i simpleItem) Description() string { return "" }
func (i simpleItem) FilterValue() string { return i.title }

type separatorItem struct{}

func (s separatorItem) Title() string       { return "" }
func (s separatorItem) Description() string { return "" }
func (s separatorItem) FilterValue() string { return "" }

type simpleDelegate struct{}

func (d simpleDelegate) Height() int                             { return 1 }
func (d simpleDelegate) Spacing() int                            { return 0 }
func (d simpleDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }
func (d simpleDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	if _, isSep := listItem.(separatorItem); isSep {
		return
	}

	i, ok := listItem.(simpleItem)
	if !ok {
		return
	}

	if i.isAction {
		if index == m.Index() {
			fmt.Fprint(w, actionSelectedStyle.Render("▸ "+i.title))
		} else {
			fmt.Fprint(w, actionNormalStyle.Render(i.title))
		}
		return
	}

	if index == m.Index() {
		num := itemNumberSelectedStyle.Render(fmt.Sprintf("%d.", i.index+1))
		fmt.Fprint(w, itemSelectedStyle.Render(fmt.Sprintf("▸ %s %s", num, i.title)))
	} else {
		num := itemNumberStyle.Render(fmt.Sprintf("%d.", i.index+1))
		fmt.Fprint(w, itemNormalStyle.Render(fmt.Sprintf("%s %s", num, i.title)))
	}
}

type selectionModel struct {
	list     list.Model
	choice   int
	quitting bool
	title    string
}

func (m selectionModel) Init() tea.Cmd {
	return nil
}

func (m selectionModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	prevIdx := m.list.Index()

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.list.SetWidth(msg.Width)
		return m, nil

	case tea.KeyMsg:
		switch keypress := msg.String(); keypress {
		case "ctrl+c", "esc":
			m.choice = -1
			m.quitting = true
			return m, tea.Quit

		case "enter":
			if i, ok := m.list.SelectedItem().(simpleItem); ok {
				m.choice = i.index
			}
			return m, tea.Quit

		case "1", "2", "3", "4", "5", "6", "7", "8", "9":
			num := int(keypress[0] - '0')
			count := 0
			for idx, item := range m.list.Items() {
				if si, ok := item.(simpleItem); ok {
					count++
					if count == num {
						m.list.Select(idx)
						m.choice = si.index
						return m, tea.Quit
					}
				}
			}
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)

	if _, isSep := m.list.SelectedItem().(separatorItem); isSep {
		newIdx := m.list.Index()
		if newIdx > prevIdx {
			m.list.CursorDown()
		} else {
			m.list.CursorUp()
		}
	}

	return m, cmd
}

func (m selectionModel) View() string {
	if m.quitting {
		return ""
	}

	titleStyle := lipgloss.NewStyle().
		Foreground(Indigo).
		Bold(true).
		MarginBottom(1)

	view := titleStyle.Render(m.title) + "\n"
	view += m.list.View()

	helpStyle := lipgloss.NewStyle().
		Foreground(Muted).
		MarginTop(1)
	view += "\n" + helpStyle.Render("↑/k up • ↓/j down • 1-9 quick select • enter select • esc cancel")

	return lipgloss.NewStyle().Margin(1, 0).PaddingLeft(2).Render(view)
}

// SelectFromList allows selecting from a list using arrow keys.
// Regular options are numbered; actions (variadic) are rendered below a
// separator in a muted style. Returned index spans both slices continuously:
// 0..len(options)-1 for options, len(options)..len(options)+len(actions)-1 for actions.
func SelectFromList(title string, options []string, actions ...string) (int, error) {
	items := []list.Item{}
	idx := 0
	for _, option := range options {
		items = append(items, simpleItem{title: option, index: idx})
		idx++
	}
	if len(actions) > 0 {
		items = append(items, separatorItem{})
		for _, action := range actions {
			items = append(items, simpleItem{title: action, index: idx, isAction: true})
			idx++
		}
	}

	listHeight := len(items) + 2
	if listHeight > 15 {
		listHeight = 15
	}

	l := list.New(items, simpleDelegate{}, 60, listHeight)
	l.SetShowTitle(false)
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.SetShowHelp(false)

	l.KeyMap = list.KeyMap{
		CursorUp: key.NewBinding(
			key.WithKeys("up", "k"),
		),
		CursorDown: key.NewBinding(
			key.WithKeys("down", "j"),
		),
	}

	m := selectionModel{
		list:   l,
		title:  title,
		choice: -1,
	}

	p := tea.NewProgram(m)
	finalModel, err := p.Run()
	if err != nil {
		return -1, fmt.Errorf("error running selection: %w", err)
	}

	if m, ok := finalModel.(selectionModel); ok {
		if m.choice == -1 {
			return -1, fmt.Errorf("cancelled")
		}
		return m.choice, nil
	}

	return -1, fmt.Errorf("unexpected model type")
}
