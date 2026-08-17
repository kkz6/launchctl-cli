// Package buildversion resolves the version reported by an lctl binary.
package buildversion

import (
	"runtime/debug"
	"strings"

	"golang.org/x/mod/semver"
)

const developmentVersion = "dev"

// Resolve returns the effective build version without a leading "v".
//
// A valid linker-supplied version takes precedence because release binaries
// receive it from GoReleaser. When the linker value is absent or invalid,
// Resolve falls back to the main module version recorded by Go. This fallback
// covers binaries installed with `go install module@version`, which do not use
// the GoReleaser linker flags. Local and otherwise invalid builds resolve to
// "dev".
func Resolve(linkerValue string, buildInfo *debug.BuildInfo) string {
	if version, ok := normalize(linkerValue); ok {
		return version
	}

	if buildInfo != nil {
		if version, ok := normalize(buildInfo.Main.Version); ok {
			return version
		}
	}

	return developmentVersion
}

func normalize(value string) (string, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", false
	}

	if !strings.HasPrefix(value, "v") {
		value = "v" + value
	}

	canonical := semver.Canonical(value)
	if canonical == "" {
		return "", false
	}

	return strings.TrimPrefix(canonical, "v"), true
}
