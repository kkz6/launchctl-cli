# lctl

`lctl` is the launchctl terminal client for servers, sites, Docker projects and
applications, deployments, and day-to-day infrastructure operations. It
combines scriptable Cobra commands, Bubble Tea views that work cleanly in tmux,
resilient WebSocket progress, and an authenticated raw API escape hatch for
every backend feature.

launchctl is currently a hosted service. The CLI does not install or operate a
self-managed launchctl control plane.

## Install

```bash
brew install kkz6/tap/lctl
```

Using the fully qualified formula trusts only `lctl`, rather than every current
and future package in the tap. If an older, separately tapped installation asks
for trust, grant it to the formula and retry:

```bash
brew trust --formula kkz6/tap/lctl
brew install kkz6/tap/lctl
```

Release archives include macOS and Linux builds for AMD64 and ARM64. Repository
contributors can build the current checkout with `go build -o lctl .`; the
private source repository is not a public `go install` distribution channel.

### Install the AI skill

The CLI contains the same `operate-launchctl` skill published through the
repository's Codex plugin. Choose one installation method to avoid loading the
same skill twice.

For an offline installation managed by the CLI:

```bash
lctl ai install
lctl ai doctor
```

After upgrading `lctl`, run `lctl ai update`. The installer writes atomically,
detects local changes, and never replaces an unmanaged skill directory.

For the versioned Codex marketplace plugin:

```bash
codex plugin marketplace add kkz6/launchctl-cli --ref v0.3.0
codex plugin add launchctl@launchctl
```

Restart Codex if the skill does not appear immediately, then invoke it with
`$operate-launchctl` or ask naturally for help operating launchctl resources.

## Quick start

```bash
lctl login
lctl whoami
lctl servers list
lctl init
lctl deploy trigger <site-id>

# Docker application workflow
lctl docker projects list --server <docker-server-id>
lctl docker applications list \
  --server <docker-server-id> \
  --project <project-id>
```

Run `lctl` without a subcommand for the interactive navigator, or use
`lctl status` for the live team dashboard.

## Keep the CLI current

```bash
lctl update --check
lctl update
```

`lctl update` verifies the release archive and then uses the installation's
original update path. Homebrew installations run
`brew upgrade kkz6/tap/lctl`; direct binary installations are replaced
atomically after checksum and staged-version validation. Use
`lctl update --check --json` for automation and `lctl update --force` to
reinstall the latest release. The `upgrade` command is an alias for `update`.

The interactive header reads a local update cache and shows
`Update available: vX.Y.Z` without delaying startup. A detached check refreshes
successful results at most once every 24 hours; failures are retried after one
hour. Set `LAUNCHCTL_NO_UPDATE_CHECK=1` to disable passive checks and notices.
Explicit `lctl update` commands still work. If the CLI manages your AI skill,
run `lctl ai update` after updating `lctl` to install the newly bundled files.

## Command surface

| Area | Commands |
| --- | --- |
| Account | `login`, `logout`, `whoami`, `auth`, `teams`, `config`, `switch` |
| Servers | `servers list/show/reboot/ssh/metrics/watch` |
| Sites | `sites list/show`, `env pull/push`, `logs`, `run` |
| Deployments | `deploy trigger/list/show/logs/rollback` |
| Docker | `docker projects list/show/create/update/delete`, `docker applications list/show/create/update/deploy/reload/start/stop/deployments/delete` |
| Operations | `services`, `databases`, `ssh-keys`, `firewall`, `cron`, `ssl`, `daemons` |
| Live work | `status`, `events`, `tasks list/watch` |
| CLI lifecycle | `version`, `update`, `update --check` |
| Automation | `init`, `api`, `completion`, `--json`, `--ci` |
| AI | `ai install/doctor/update/uninstall`, `$operate-launchctl` |

Use `lctl <command> --help` for the installed version's exact flags.

## Live progress

```bash
lctl servers watch <server-id>
lctl tasks watch <task-id> --server <server-id>
lctl events --filter 'deployment.*'
lctl events --json | jq -c 'select(.event == "task.failed")'
```

The shared WebSocket manager sends heartbeats, reconnects with exponential
backoff, and restores authorized subscriptions after a disconnect. Interactive
views expose connection state, pause/clear/scroll controls, and clean alternate
screen handling. The dashboard reacts to events immediately and retains a
30-second REST reconciliation interval.

## Complete API access

High-level commands cover common human workflows. New backend modules are
available immediately through the authenticated client:

```bash
lctl api GET /api/servers/<server-id>/backups
lctl api POST /api/scripts --data @script.json
lctl api GET /api/settings/notifications
```

`lctl api` applies the active token, team, profile, API origin, safe-read
retries, response bounds, and typed errors. Mutation requests are never retried.

## Profiles and CI

```bash
lctl config profiles add staging
lctl config profiles use staging
lctl --profile staging servers list
```

Configuration lives at `~/.config/launchctl/config.json`; site or Docker
application context lives in `.launchctl.yml`. Runtime environment variables
are:

- `LAUNCHCTL_API_URL`
- `LAUNCHCTL_TOKEN`
- `LAUNCHCTL_TEAM_ID`
- `LAUNCHCTL_NO_UPDATE_CHECK` (disables passive update checks and notices)
- `LAUNCHCTL_THEME` (`auto`, `light`, or `dark`; useful when a terminal multiplexer cannot report its background)

`--api-url` overrides the environment and profile for one invocation. API-origin
overrides are intended for launchctl development and explicitly assigned test
or staging environments; they are not a supported self-hosting interface.

## Development

```bash
go test -race ./...
go vet ./...
go build ./...
```

Preview the real startup header without loading local credentials or API
clients:

```bash
make splash-preview
go run ./internal/splash/preview --width 28 --color never --version 0.3.0
```

See [the CLI reference](docs/cli-reference.md) and the
[production-readiness design](docs/plans/2026-08-14-cli-production-readiness.md).

## License

Private — all rights reserved.
