package buildversion

import (
	"runtime/debug"
	"testing"
)

func TestResolvePrefersValidLinkerVersion(t *testing.T) {
	buildInfo := &debug.BuildInfo{Main: debug.Module{Version: "v9.8.7"}}

	tests := map[string]struct {
		linkerValue string
		want        string
	}{
		"plain release":      {linkerValue: "1.2.3", want: "1.2.3"},
		"v-prefixed release": {linkerValue: "v2.3.4", want: "2.3.4"},
		"surrounding space":  {linkerValue: "  3.4.5\n", want: "3.4.5"},
		"prerelease":         {linkerValue: "4.5.6-rc.1", want: "4.5.6-rc.1"},
		"build metadata":     {linkerValue: "5.6.7+build.42", want: "5.6.7"},
		"short version":      {linkerValue: "6.7", want: "6.7.0"},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			if got := Resolve(test.linkerValue, buildInfo); got != test.want {
				t.Fatalf("Resolve(%q, buildInfo) = %q, want %q", test.linkerValue, got, test.want)
			}
		})
	}
}

func TestResolveFallsBackToGoInstallModuleVersion(t *testing.T) {
	tests := map[string]string{
		"v0.2.2":                       "0.2.2",
		"v1.0.0-beta.2":                "1.0.0-beta.2",
		"v0.0.0-20260817010101-abcdef": "0.0.0-20260817010101-abcdef",
		"2.4.6":                        "2.4.6",
	}

	for moduleVersion, want := range tests {
		t.Run(moduleVersion, func(t *testing.T) {
			buildInfo := &debug.BuildInfo{Main: debug.Module{Version: moduleVersion}}
			if got := Resolve("dev", buildInfo); got != want {
				t.Fatalf("Resolve(dev, module %q) = %q, want %q", moduleVersion, got, want)
			}
		})
	}
}

func TestResolveFallsBackWhenLinkerVersionIsInvalid(t *testing.T) {
	buildInfo := &debug.BuildInfo{Main: debug.Module{Version: "v7.8.9"}}

	for _, linkerValue := range []string{"", "dev", "(devel)", "latest", "1.02.3", "1.2.3.4"} {
		t.Run(linkerValue, func(t *testing.T) {
			if got := Resolve(linkerValue, buildInfo); got != "7.8.9" {
				t.Fatalf("Resolve(%q, buildInfo) = %q, want 7.8.9", linkerValue, got)
			}
		})
	}
}

func TestResolveDevelopmentBuilds(t *testing.T) {
	tests := map[string]struct {
		linkerValue  string
		buildVersion string
		buildInfo    bool
	}{
		"no metadata":          {},
		"local go build":       {linkerValue: "dev", buildVersion: "(devel)", buildInfo: true},
		"invalid module":       {buildVersion: "not-a-version", buildInfo: true},
		"invalid linker only":  {linkerValue: "latest"},
		"empty module version": {linkerValue: "(devel)", buildInfo: true},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			var buildInfo *debug.BuildInfo
			if test.buildInfo {
				buildInfo = &debug.BuildInfo{Main: debug.Module{Version: test.buildVersion}}
			}

			if got := Resolve(test.linkerValue, buildInfo); got != developmentVersion {
				t.Fatalf("Resolve(%q, %#v) = %q, want %q", test.linkerValue, buildInfo, got, developmentVersion)
			}
		})
	}
}

func TestResolveDoesNotLetInvalidLinkerOverrideValidModule(t *testing.T) {
	buildInfo := &debug.BuildInfo{Main: debug.Module{Version: "v1.2.3"}}

	if got := Resolve("v1.2.3-01", buildInfo); got != "1.2.3" {
		t.Fatalf("Resolve returned %q, want module fallback 1.2.3", got)
	}
}
