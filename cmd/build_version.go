package cmd

import (
	"runtime/debug"

	"github.com/kkz6/launchctl/internal/buildversion"
)

func init() {
	buildInfo, _ := debug.ReadBuildInfo()
	Version = buildversion.Resolve(Version, buildInfo)
}
