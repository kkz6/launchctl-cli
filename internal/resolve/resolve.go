package resolve

import (
	"fmt"

	"github.com/kkz6/launchctl/internal/config"
)

func ServerID(flagValue string) (string, error) {
	if flagValue != "" {
		return flagValue, nil
	}

	proj, err := config.LoadProject()
	if err == nil && proj.Server != "" {
		return proj.Server, nil
	}

	return "", fmt.Errorf("--server flag is required (or run `lctl init` to set a default)")
}

func SiteID(flagValue string) (string, error) {
	if flagValue != "" {
		return flagValue, nil
	}

	proj, err := config.LoadProject()
	if err == nil && proj.Site != "" {
		return proj.Site, nil
	}

	return "", fmt.Errorf("--site flag is required (or run `lctl init` to set a default)")
}
