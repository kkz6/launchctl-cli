// Package launchctlplugin exposes the bundled launchctl skill to the lctl
// binary. Keeping the embedded files inside the plugin makes this directory
// the canonical source for both marketplace and CLI-managed installations.
package launchctlplugin

import "embed"

// SkillFS contains the complete operate-launchctl skill directory.
//
//go:embed skills/operate-launchctl
var SkillFS embed.FS

// PluginManifest is the marketplace plugin manifest shipped from this source
// tree. Release checks keep its version aligned with the lctl tag.
//
//go:embed .codex-plugin/plugin.json
var PluginManifest []byte
