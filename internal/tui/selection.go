package tui

import (
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/kkz6/launchctl/internal/notify"
)

var (
	itemNormalStyle = lipgloss.NewStyle().
			PaddingLeft(2).
			Foreground(Slate)

	itemSelectedStyle = lipgloss.NewStyle().
				Foreground(Indigo).
				Bold(true)

	itemNumberStyle = lipgloss.NewStyle().
			Foreground(Muted)

	itemNumberSelectedStyle = lipgloss.NewStyle().
				Foreground(Indigo)

	actionNormalStyle = lipgloss.NewStyle().
				PaddingLeft(2).
				Foreground(DarkSlate)

	actionSelectedStyle = lipgloss.NewStyle().
				Foreground(Slate).
				Bold(true)

	tableColNormal = lipgloss.NewStyle().
			Foreground(Slate)

	tableColSelected = lipgloss.NewStyle().
				Foreground(Indigo).
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

type tableItem struct {
	columns []string
	widths  []int
	index   int
}

func (t tableItem) Title() string       { return t.columns[0] }
func (t tableItem) Description() string { return "" }
func (t tableItem) FilterValue() string { return t.columns[0] }

type simpleDelegate struct{}

func (d simpleDelegate) Height() int                             { return 1 }
func (d simpleDelegate) Spacing() int                            { return 0 }
func (d simpleDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }
func (d simpleDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	if _, isSep := listItem.(separatorItem); isSep {
		return
	}

	if ti, ok := listItem.(tableItem); ok {
		renderTableRow(w, m, index, ti)
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

func renderTableRow(w io.Writer, m list.Model, index int, ti tableItem) {
	selected := index == m.Index()

	var row strings.Builder
	if selected {
		row.WriteString("▸ ")
		numStr := tableColSelected.Render(fmt.Sprintf("%d.", ti.index+1))
		row.WriteString(numStr)
		row.WriteString(" ")
	} else {
		row.WriteString("  ")
		numStr := itemNumberStyle.Render(fmt.Sprintf("%d.", ti.index+1))
		row.WriteString(numStr)
		row.WriteString(" ")
	}

	for col, val := range ti.columns {
		w := ti.widths[col]
		cell := val
		visualWidth := lipgloss.Width(cell)
		if visualWidth > w {
			cell = ansi.Truncate(cell, w-1, "…")
			visualWidth = lipgloss.Width(cell)
		}
		padded := cell + strings.Repeat(" ", max(0, w-visualWidth))
		if selected {
			row.WriteString(tableColSelected.Render(padded))
		} else {
			row.WriteString(tableColNormal.Render(padded))
		}
		if col < len(ti.columns)-1 {
			row.WriteString("  ")
		}
	}

	fmt.Fprint(w, row.String())
}

type selectionModel struct {
	list         list.Model
	choice       int
	quitting     bool
	title        string
	termHeight   int
	termWidth    int
	tableHeaders []string
	tableWidths  []int
}

func (m selectionModel) Init() tea.Cmd {
	return nil
}

func (m selectionModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	prevIdx := m.list.Index()

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.termHeight = msg.Height
		m.termWidth = msg.Width
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
			if i, ok := m.list.SelectedItem().(tableItem); ok {
				m.choice = i.index
			}
			return m, tea.Quit

		case "1", "2", "3", "4", "5", "6", "7", "8", "9":
			num := int(keypress[0] - '0')
			count := 0
			for idx, item := range m.list.Items() {
				switch si := item.(type) {
				case simpleItem:
					if !si.isAction {
						count++
						if count == num {
							m.list.Select(idx)
							m.choice = si.index
							return m, tea.Quit
						}
					}
				case tableItem:
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

	helpStyle := lipgloss.NewStyle().
		Foreground(Muted)

	helpText := helpStyle.Render("↑/k up • ↓/j down • 1-9 quick select • enter select • esc cancel")

	notifyBar := notify.Render()

	content := titleStyle.Render(m.title) + "\n"

	if len(m.tableHeaders) > 0 {
		headerStyle := lipgloss.NewStyle().Foreground(Muted).Bold(true)
		dividerStyle := lipgloss.NewStyle().Foreground(DarkSlate)

		var headerRow strings.Builder
		headerRow.WriteString("     ")
		var dividerRow strings.Builder
		dividerRow.WriteString("     ")

		for i, h := range m.tableHeaders {
			w := m.tableWidths[i]
			cell := h
			if len(cell) > w {
				cell = cell[:w-1] + "…"
			}
			padded := cell + strings.Repeat(" ", max(0, w-len(cell)))
			headerRow.WriteString(headerStyle.Render(padded))
			dividerRow.WriteString(dividerStyle.Render(strings.Repeat("─", w)))
			if i < len(m.tableHeaders)-1 {
				headerRow.WriteString("  ")
				dividerRow.WriteString("  ")
			}
		}

		content += headerRow.String() + "\n"
		content += dividerRow.String() + "\n"
	}

	content += m.list.View()

	contentLines := strings.Count(content, "\n") + 1

	footerLines := 1
	if notifyBar != "" {
		footerLines = 2
	}

	// +2 for top margin
	usedLines := contentLines + footerLines + 2

	padding := 0
	if m.termHeight > 0 && m.termHeight > usedLines {
		padding = m.termHeight - usedLines
	}

	var view strings.Builder
	view.WriteString(content)

	if padding > 0 {
		view.WriteString(strings.Repeat("\n", padding))
	}

	if notifyBar != "" {
		view.WriteString("\n" + notifyBar)
	}
	view.WriteString("\n" + helpText)

	return lipgloss.NewStyle().Margin(1, 0).Render(view.String())
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

// TableColumn defines a column for SelectFromTable.
type TableColumn struct {
	Header string
	Width  int
}

// TableRow holds cell values for one row.
type TableRow struct {
	Columns []string
}

// SelectFromTable renders an interactive list where each row shows multi-column
// data inline. Actions (like "Back") are rendered below a separator.
func SelectFromTable(title string, columns []TableColumn, rows []TableRow, actions ...string) (int, error) {
	widths := make([]int, len(columns))
	for i, c := range columns {
		widths[i] = c.Width
	}

	items := []list.Item{}
	for i, r := range rows {
		cols := make([]string, len(columns))
		for j := range columns {
			if j < len(r.Columns) {
				cols[j] = r.Columns[j]
			}
		}
		items = append(items, tableItem{columns: cols, widths: widths, index: i})
	}

	if len(actions) > 0 {
		items = append(items, separatorItem{})
		idx := len(rows)
		for _, action := range actions {
			items = append(items, simpleItem{title: action, index: idx, isAction: true})
			idx++
		}
	}

	listHeight := len(items) + 2
	if listHeight > 20 {
		listHeight = 20
	}

	totalWidth := 0
	for _, c := range columns {
		totalWidth += c.Width + 2
	}
	totalWidth += 6
	if totalWidth < 60 {
		totalWidth = 60
	}

	l := list.New(items, simpleDelegate{}, totalWidth, listHeight)
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

	headers := make([]string, len(columns))
	for i, c := range columns {
		headers[i] = c.Header
	}

	m := selectionModel{
		list:         l,
		title:        title,
		choice:       -1,
		tableHeaders: headers,
		tableWidths:  widths,
	}

	p := tea.NewProgram(m)
	finalModel, err := p.Run()
	if err != nil {
		return -1, fmt.Errorf("error running selection: %w", err)
	}

	if fm, ok := finalModel.(selectionModel); ok {
		if fm.choice == -1 {
			return -1, fmt.Errorf("cancelled")
		}
		return fm.choice, nil
	}

	return -1, fmt.Errorf("unexpected model type")
}
