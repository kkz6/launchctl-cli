package tui

import (
	"math"
	"strconv"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestAdaptiveTextColorsMeetContrast(t *testing.T) {
	colors := []struct {
		name  string
		color lipgloss.AdaptiveColor
	}{
		{name: "indigo", color: Indigo},
		{name: "green", color: Green},
		{name: "slate", color: Slate},
		{name: "red", color: Red},
		{name: "yellow", color: Yellow},
		{name: "blue", color: Blue},
		{name: "cyan", color: Cyan},
		{name: "orange", color: Orange},
		{name: "primary", color: White},
		{name: "muted", color: Muted},
	}

	for _, color := range colors {
		t.Run(color.name+"/light", func(t *testing.T) {
			assertContrast(t, color.color.Light, "#FFFFFF", 4.5)
		})
		t.Run(color.name+"/dark", func(t *testing.T) {
			assertContrast(t, color.color.Dark, "#0F172A", 4.5)
		})
	}
}

func TestAdaptiveSurfaceColorsMeetContrast(t *testing.T) {
	assertContrast(t, DarkSlate.Light, "#FFFFFF", 3)
	assertContrast(t, DarkSlate.Dark, "#0F172A", 3)
	assertContrast(t, Slate.Light, Panel.Light, 4.5)
	assertContrast(t, Slate.Dark, Panel.Dark, 4.5)
	assertContrast(t, OnAccent.Light, Indigo.Light, 4.5)
	assertContrast(t, OnAccent.Dark, Indigo.Dark, 4.5)
}

func TestConfigureThemeOverride(t *testing.T) {
	previous := lipgloss.HasDarkBackground()
	t.Cleanup(func() {
		lipgloss.SetHasDarkBackground(previous)
	})

	if err := ConfigureTheme("light"); err != nil {
		t.Fatal(err)
	}
	if lipgloss.HasDarkBackground() {
		t.Fatal("light theme override selected a dark background")
	}

	if err := ConfigureTheme(" DARK "); err != nil {
		t.Fatal(err)
	}
	if !lipgloss.HasDarkBackground() {
		t.Fatal("dark theme override selected a light background")
	}

	if err := ConfigureTheme("auto"); err != nil {
		t.Fatal(err)
	}
	if !lipgloss.HasDarkBackground() {
		t.Fatal("auto unexpectedly replaced the existing renderer decision")
	}

	if err := ConfigureTheme("sepia"); err == nil {
		t.Fatal("invalid theme override was accepted")
	}
}

func assertContrast(t *testing.T, foreground, background string, minimum float64) {
	t.Helper()

	foregroundLuminance := relativeLuminance(t, foreground)
	backgroundLuminance := relativeLuminance(t, background)
	lighter := math.Max(foregroundLuminance, backgroundLuminance)
	darker := math.Min(foregroundLuminance, backgroundLuminance)
	ratio := (lighter + 0.05) / (darker + 0.05)
	if ratio < minimum {
		t.Fatalf("contrast %s on %s is %.2f:1, want at least %.1f:1", foreground, background, ratio, minimum)
	}
}

func relativeLuminance(t *testing.T, value string) float64 {
	t.Helper()

	hex := strings.TrimPrefix(value, "#")
	if len(hex) != 6 {
		t.Fatalf("invalid RGB color %q", value)
	}

	channels := make([]float64, 3)
	for index := range channels {
		parsed, err := strconv.ParseUint(hex[index*2:index*2+2], 16, 8)
		if err != nil {
			t.Fatalf("invalid RGB color %q: %v", value, err)
		}
		channel := float64(parsed) / 255
		if channel <= 0.04045 {
			channels[index] = channel / 12.92
		} else {
			channels[index] = math.Pow((channel+0.055)/1.055, 2.4)
		}
	}

	return 0.2126*channels[0] + 0.7152*channels[1] + 0.0722*channels[2]
}
