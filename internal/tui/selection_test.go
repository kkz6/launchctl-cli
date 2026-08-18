package tui

import (
	"testing"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

func TestSelectionModelMouseWheelMovesCursor(t *testing.T) {
	m := testSelectionModel([]list.Item{
		simpleItem{title: "First", index: 0},
		simpleItem{title: "Second", index: 1},
		simpleItem{title: "Third", index: 2},
	})

	updated, _ := m.Update(tea.MouseMsg{
		Button: tea.MouseButtonWheelDown,
		Action: tea.MouseActionPress,
	})
	m = updated.(selectionModel)
	if got := m.list.Index(); got != 1 {
		t.Fatalf("wheel down selected index %d, want 1", got)
	}

	updated, _ = m.Update(tea.MouseMsg{
		Button: tea.MouseButtonWheelUp,
		Action: tea.MouseActionPress,
	})
	m = updated.(selectionModel)
	if got := m.list.Index(); got != 0 {
		t.Fatalf("wheel up selected index %d, want 0", got)
	}
}

func TestSelectionModelMouseWheelSkipsSeparator(t *testing.T) {
	m := testSelectionModel([]list.Item{
		simpleItem{title: "Server", index: 0},
		separatorItem{},
		simpleItem{title: "Back", index: 1, isAction: true},
	})

	updated, _ := m.Update(tea.MouseMsg{
		Button: tea.MouseButtonWheelDown,
		Action: tea.MouseActionPress,
	})
	m = updated.(selectionModel)
	if got := m.list.Index(); got != 2 {
		t.Fatalf("wheel down selected index %d, want action at 2", got)
	}

	updated, _ = m.Update(tea.MouseMsg{
		Button: tea.MouseButtonWheelUp,
		Action: tea.MouseActionPress,
	})
	m = updated.(selectionModel)
	if got := m.list.Index(); got != 0 {
		t.Fatalf("wheel up selected index %d, want item at 0", got)
	}
}

func TestSelectionModelIgnoresMouseMotion(t *testing.T) {
	m := testSelectionModel([]list.Item{
		simpleItem{title: "First", index: 0},
		simpleItem{title: "Second", index: 1},
	})

	updated, _ := m.Update(tea.MouseMsg{
		Button: tea.MouseButtonNone,
		Action: tea.MouseActionMotion,
	})
	m = updated.(selectionModel)
	if got := m.list.Index(); got != 0 {
		t.Fatalf("mouse motion selected index %d, want 0", got)
	}
}

func testSelectionModel(items []list.Item) selectionModel {
	l := list.New(items, simpleDelegate{}, 60, 15)
	l.SetShowTitle(false)
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.SetShowHelp(false)
	l.KeyMap = list.KeyMap{}

	return selectionModel{list: l, choice: -1}
}
