// Command preview renders the real launchctl splash without loading account
// configuration, credentials, or API clients.
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/kkz6/launchctl/internal/splash"
)

func main() {
	width := flag.Int("width", 0, "terminal width to preview (0 detects it)")
	color := flag.String("color", "auto", "color mode: auto, always, or never")
	version := flag.String("version", "dev", "version displayed in the header")
	updateVersion := flag.String("update-version", "", "available update version to preview")
	flag.Parse()

	options := splash.TerminalOptions(os.Stdout)
	if *width > 0 {
		options.Width = *width
	}
	options.UpdateVersion = *updateVersion

	switch *color {
	case "auto":
	case "always":
		options.Color = true
	case "never":
		options.Color = false
	default:
		fmt.Fprintf(os.Stderr, "invalid --color %q (use auto, always, or never)\n", *color)
		os.Exit(2)
	}

	fmt.Print(splash.Render(*version, options))
}
