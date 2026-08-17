package tui

import (
	"strings"
	"testing"
)

func TestClearScreenPreservesScrollback(t *testing.T) {
	var output strings.Builder
	clearScreen(&output)

	if got, want := output.String(), "\x1b[H\x1b[2J"; got != want {
		t.Fatalf("clearScreen() = %q, want %q", got, want)
	}
	if strings.Contains(output.String(), "\x1b[3J") {
		t.Fatal("clearScreen erased terminal scrollback")
	}
}

func TestDividerWidthFitsNarrowTerminals(t *testing.T) {
	tests := map[int]int{
		0:  50,
		1:  1,
		24: 24,
		40: 40,
		50: 50,
		80: 50,
	}

	for terminalWidth, want := range tests {
		if got := dividerWidth(terminalWidth); got != want {
			t.Errorf("dividerWidth(%d) = %d, want %d", terminalWidth, got, want)
		}
	}
}
