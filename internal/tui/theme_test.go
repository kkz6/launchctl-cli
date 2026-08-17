package tui

import "testing"

func TestFormThemeDoesNotAddMenuLeftBorders(t *testing.T) {
	theme := FormTheme()

	if theme.Focused.Base.GetBorderLeft() {
		t.Fatal("focused form base must not render a left border")
	}
	if theme.Blurred.Base.GetBorderLeft() {
		t.Fatal("blurred form base must not render a left border")
	}
}
