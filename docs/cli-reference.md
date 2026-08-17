# lctl — CLI Reference

`lctl` is a command-line tool for managing your Launchctl servers, sites, and deployments.

## Installation

### Homebrew

```bash
brew install kkz6/tap/lctl
lctl version
```

The fully qualified formula trusts only `lctl`. If Homebrew reports that a
previously added tap is untrusted, trust the formula rather than the entire tap:

```bash
brew trust --formula kkz6/tap/lctl
brew install kkz6/tap/lctl
```

### Source checkout

Repository contributors with source access can build the current checkout:

```bash
go build -o lctl .
./lctl version
```

The source repository is private, so `go install github.com/kkz6/launchctl@latest`
is not a public installation method. Published macOS and Linux archives support
AMD64 and ARM64.

## Quick Start

```bash
# Authenticate with your API token
lctl login

# Check who you're logged in as
lctl whoami

# Launch the interactive dashboard
lctl

# Bind a project to a server + site
cd ~/my-project
lctl init

# Deploy
lctl deploy trigger <site-id>
```

---

## CLI Updates

### Check without changing the installation

```bash
lctl update --check
lctl update --check --json
```

The JSON result includes `current_version`, `latest_version`,
`update_available`, the check timestamp, and the metadata source. A successful
check exits normally whether or not an update is available; automation should
read `update_available`.

### Install the latest stable release

```bash
lctl update
```

`lctl upgrade` is an alias. Homebrew-managed binaries delegate to
`brew upgrade kkz6/tap/lctl`, preserving Homebrew's Cellar and receipts. If the
formula is not trusted, run `brew trust --formula kkz6/tap/lctl` and retry.
Direct installations download the matching macOS or Linux archive, verify its
SHA-256 digest and staged version, and atomically replace the executable. The
updater refuses development builds, unsupported platforms, and unmanaged
symlink targets.

Use `lctl update --force` to reinstall the latest release. For Homebrew this
delegates to `brew reinstall kkz6/tap/lctl`. The updater never grants trust,
invokes `sudo`, or changes the bundled AI skill automatically. Run
`lctl ai update` separately when that skill is managed by the CLI.

### Passive update notice

The interactive `lctl` header displays `Update available: vX.Y.Z` from a local
cache. Startup does not wait for the network: a detached check refreshes
successful results at most once every 24 hours, while failed checks retry after
one hour. Passive checks and notices are disabled in CI, JSON, non-interactive,
and development-build contexts.

To disable them explicitly:

```bash
export LAUNCHCTL_NO_UPDATE_CHECK=1
```

This setting does not disable explicit `lctl update` or
`lctl update --check` commands.

---

## AI Skill

The `operate-launchctl` skill teaches Codex how to select safe `lctl` commands,
follow WebSocket progress, reconcile asynchronous work, and avoid exposing
credentials. Use either the CLI-managed standalone skill or the repository
plugin, not both.

### CLI-managed installation

```bash
lctl ai install
lctl ai doctor
lctl ai update
lctl ai uninstall
```

The default destination is `$CODEX_HOME/skills/operate-launchctl`, falling back
to `~/.codex/skills/operate-launchctl`. Use `--codex-home <path>` to target a
different Codex home. `doctor` and all mutations support the global `--json`
flag.

`update` and `uninstall` refuse to replace locally modified managed files unless
`--force` is provided. Unmanaged directories and symlink destinations are never
overwritten or removed.

### Marketplace plugin

Install the plugin from the GitHub repository at the matching CLI release tag:

```bash
codex plugin marketplace add kkz6/launchctl-cli --ref v0.2.3
codex plugin add launchctl@launchctl
```

Start a new Codex task if the newly installed skill is not visible in the
current task. Invoke it explicitly with `$operate-launchctl`, or let Codex select
it when the request concerns launchctl infrastructure.

---

## Authentication

### `lctl login`

Authenticate using a personal access token generated from the web dashboard. Supports two-factor authentication.

```bash
lctl login
```

You'll be prompted for your API token and, if enabled, a 2FA code.

### `lctl logout`

Log out and clear stored credentials.

```bash
lctl logout
```

### `lctl auth status`

Show current authentication status.

```bash
lctl auth status
```

### `lctl whoami`

Show the current user and team.

```bash
lctl whoami
```

```
╭──────────────────────────────────────────────────────╮
│  User      John Doe (john@acme.com)                  │
│  Team      Acme Corp                                 │
╰──────────────────────────────────────────────────────╯
```

```bash
# JSON output
lctl whoami --json
```

```json
{
  "user_id": "abc123",
  "name": "John Doe",
  "email": "john@acme.com",
  "team": "Acme Corp",
  "team_id": "team_456"
}
```

---

## Servers

### `lctl servers list`

List all servers in the current team.

```bash
lctl servers list
lctl servers list --json
```

### `lctl servers show <id>`

Show detailed information about a server.

```bash
lctl servers show abc123
```

### `lctl servers ssh <id>`

Open an interactive terminal session on a server via WebSocket.

```bash
lctl servers ssh abc123
lctl servers ssh abc123 -u root
```

| Flag | Description |
|------|-------------|
| `-u, --user` | SSH user (defaults to server username) |

### `lctl servers reboot <id>`

Reboot a server.

```bash
lctl servers reboot abc123
```

### `lctl servers metrics <id>`

Show server metrics (CPU, memory, disk).

```bash
lctl servers metrics abc123
lctl servers metrics abc123 --watch
```

| Flag | Description |
|------|-------------|
| `-w, --watch` | Stream metrics in real-time |

### `lctl servers watch <id>`

Follow provisioning and lifecycle events in a reconnecting tmux-friendly
console. Add `--json` to emit NDJSON and exit on a terminal provision event.

```bash
lctl servers watch <server-id>
lctl servers watch <server-id> --json
```

---

## Tasks and Events

### `lctl tasks list`

```bash
lctl tasks list --server <server-id>
lctl tasks list --server <server-id> --json
```

### `lctl tasks watch <task-id>`

Show stored task output, then follow output, progress, markers, and status over
the authorized `task.<id>` WebSocket channel.

```bash
lctl tasks watch <task-id> --server <server-id>
```

### `lctl events`

Open the team event console. Event filters use shell-style globs; additional
resource channels may be repeated.

```bash
lctl events --filter 'deployment.*' --filter 'task.*'
lctl events --channel server.<server-id> --json
```

Interactive keys: `space` pause, `c` clear, arrow/Page keys scroll, `q` quit.

---

## Authenticated API

### `lctl api <method> <path>`

Call any known backend endpoint with the active token, team, profile, and API
origin. Paths must begin with `/api`; request bodies must be JSON.

```bash
lctl api GET /api/servers/<server-id>/backups
lctl api GET /api/servers/<server-id>/docker/projects
lctl api POST /api/scripts --data '{"name":"health-check"}'
lctl api PATCH /api/example --data @request.json
```

Safe reads retry transient failures. Mutation methods are never replayed.

---

## Sites

### `lctl sites list`

List all sites on a server.

```bash
lctl sites list --server <server-id>
lctl sites list --json
```

If you've run `lctl init` in your project, the `--server` flag is optional.

### `lctl sites show <site-id>`

Show detailed information about a site.

```bash
lctl sites show <site-id> --server <server-id>
```

---

## Deployments

### `lctl deploy trigger <site-id>`

Trigger a deployment and stream live logs.

```bash
lctl deploy trigger <site-id> --server <server-id>
```

| Flag | Description |
|------|-------------|
| `--server` | Server ID (optional if `lctl init` is configured) |
| `--wait` | Wait for deployment to complete (for CI/CD) |
| `--timeout` | Timeout in seconds when using `--wait` (default: 300) |

### `lctl deploy list <site-id>`

List deployments for a site.

```bash
lctl deploy list <site-id> --server <server-id>
```

### `lctl deploy show <deployment-id>`

Show deployment details.

```bash
lctl deploy show <deployment-id> --server <server-id> --site <site-id>
```

### `lctl deploy rollback <deployment-id>`

Rollback to a previous deployment.

```bash
lctl deploy rollback <deployment-id> --server <server-id> --site <site-id>
```

### `lctl deploy logs <site-id> [deployment-id]`

View deployment logs. Shows the latest deployment if no deployment ID is specified.

```bash
# View latest deployment logs
lctl deploy logs <site-id> --server <server-id>

# View specific deployment logs
lctl deploy logs <site-id> <deployment-id> --server <server-id>

# Stream logs in real-time
lctl deploy logs <site-id> --server <server-id> --follow
```

| Flag | Description |
|------|-------------|
| `-f, --follow` | Stream logs in real-time for active deployments |

---

## Teams

### `lctl teams list`

List all teams you belong to.

```bash
lctl teams list
```

### `lctl teams switch <team-id>`

Switch the active team context.

```bash
lctl teams switch <team-id>
```

### `lctl teams members <team-id>`

List members of a team.

```bash
lctl teams members <team-id>
```

---

## Environment Files

### `lctl env pull`

Pull the `.env` file from a remote site. Prints to stdout by default.

```bash
# Print to stdout
lctl env pull --server <server-id> --site <site-id>

# Write to a local file
lctl env pull --server <server-id> --site <site-id> -o .env.local
```

| Flag | Description |
|------|-------------|
| `--server` | Server ID (optional if `lctl init` is configured) |
| `--site` | Site ID (optional if `lctl init` is configured) |
| `-o, --output` | Write to file instead of stdout |

### `lctl env push`

Push a local `.env` file to a remote site. Shows a diff and asks for confirmation before pushing.

```bash
lctl env push --server <server-id> --site <site-id> -f .env
```

In CI/CD mode (`--ci`), the confirmation prompt is skipped.

| Flag | Description |
|------|-------------|
| `--server` | Server ID (optional if `lctl init` is configured) |
| `--site` | Site ID (optional if `lctl init` is configured) |
| `-f, --file` | Local `.env` file to push (required) |

---

## Logs

### `lctl logs`

List available server logs or tail a specific log type.

```bash
# List available server logs
lctl logs --server <server-id>

# Tail a specific log type
lctl logs --server <server-id> --type nginx

# Follow log output (polls every 2s)
lctl logs --server <server-id> --type nginx --follow

# Show last 100 lines
lctl logs --server <server-id> --type nginx --lines 100

# Site-specific logs
lctl logs --server <server-id> --site <site-id>
lctl logs --server <server-id> --site <site-id> --type laravel
```

Log output is color-coded: lines containing `error`/`fatal`/`panic` are shown in red, lines containing `warn` are shown in yellow.

| Flag | Description |
|------|-------------|
| `--server` | Server ID (optional if `lctl init` is configured) |
| `--site` | Site ID (for site-specific logs) |
| `-t, --type` | Log type to view (e.g., `nginx`, `laravel`) |
| `-f, --follow` | Follow log output, polling every 2 seconds |
| `-n, --lines` | Number of lines to show from tail (default: 50) |

---

## Remote Commands

### `lctl run`

Execute a command on a remote server/site and display the output.

```bash
# Run a command
lctl run "php artisan migrate" --server <server-id> --site <site-id>

# View command history
lctl run --history --server <server-id> --site <site-id>
```

The command is submitted and polled for completion. Output is displayed when the command finishes. The CLI exits with the same exit code as the remote command.

In CI/CD mode (`--ci`), the spinner is suppressed and output is plain.

| Flag | Description |
|------|-------------|
| `--server` | Server ID (optional if `lctl init` is configured) |
| `--site` | Site ID (optional if `lctl init` is configured) |
| `--history` | List previous commands in a table |

---

## Project Configuration

### `lctl init`

Initialize a project by creating a `.launchctl.yml` file in the current directory. This binds the project to a specific server and site, so you don't need to pass `--server` and `--site` flags on every command.

```bash
# Interactive — pick server and site from a list
lctl init

# Non-interactive
lctl init --server <server-id> --site <site-id>
```

This creates a `.launchctl.yml` file:

```yaml
server: abc123
site: def456
```

Once initialized, commands like `lctl sites list`, `lctl deploy trigger`, etc. will automatically use the configured server and site.

The CLI walks up from the current directory looking for `.launchctl.yml`, stopping at the nearest `.git` boundary. This means you can run commands from any subdirectory of your project.

---

## Configuration

### `lctl config show`

Show the current configuration.

```bash
lctl config show
lctl config show --json
```

### `lctl config set <key> <value>`

Set a configuration value.

```bash
lctl config set team_id <value>
lctl config set team_name <value>
```

Available keys: `team_id`, `team_name`

---

## Profiles

Profiles let you manage multiple authenticated contexts (e.g., production vs. staging) without re-logging in.

### `lctl config profiles`

List all profiles. The active profile is marked with `*`.

```bash
lctl config profiles
```

```
Profiles

  * default              john@acme.com (Acme Corp)
    staging              john@staging.acme.com (Staging)
```

### `lctl config profiles add <name>`

Add a new profile. This triggers the login flow for the new profile.

```bash
lctl config profiles add staging
```

### `lctl config profiles use <name>`

Switch to a different profile.

```bash
lctl config profiles use staging
```

### `lctl config profiles remove <name>`

Remove a profile. You cannot remove the currently active profile.

```bash
lctl config profiles remove staging
```

### Migration

Existing configurations are automatically migrated. If you were already logged in before profiles were added, your credentials are moved into a `default` profile on the next config load. No action required.

---

## CI/CD Mode

Use `lctl` in automation pipelines with the `--ci` flag and environment variables.

### Environment Variables

| Variable | Description |
|----------|-------------|
| `LAUNCHCTL_TOKEN` | API token (overrides stored credentials) |
| `LAUNCHCTL_TEAM_ID` | Team ID (overrides stored team) |
| `LAUNCHCTL_NO_UPDATE_CHECK` | Disable passive update checks and notices |

### Usage

```bash
# Deploy and wait for completion
LAUNCHCTL_TOKEN=lctl_xxx \
LAUNCHCTL_TEAM_ID=team_abc \
lctl deploy trigger <site-id> --server <server-id> --ci --wait --timeout 600

# List servers
LAUNCHCTL_TOKEN=lctl_xxx lctl servers list --ci --json
```

The `--ci` flag disables interactive prompts. All required values must be provided via flags or environment variables.

| Flag | Description |
|------|-------------|
| `--ci` | Enable CI/CD mode (global flag) |
| `--wait` | Wait for deployment to finish (deploy trigger) |
| `--timeout` | Timeout in seconds for `--wait` (default: 300) |

### GitHub Actions Example

```yaml
jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Install lctl
        run: brew install kkz6/tap/lctl

      - name: Deploy
        env:
          LAUNCHCTL_TOKEN: ${{ secrets.LAUNCHCTL_TOKEN }}
          LAUNCHCTL_TEAM_ID: ${{ secrets.LAUNCHCTL_TEAM_ID }}
        run: |
          lctl deploy trigger ${{ vars.SITE_ID }} \
            --server ${{ vars.SERVER_ID }} \
            --ci --wait --timeout 600
```

---

## Global Flags

These flags are available on all commands.

| Flag | Description |
|------|-------------|
| `--json` | Output in JSON format |
| `--ci` | CI/CD mode (non-interactive) |
| `-h, --help` | Help for any command |

---

## Interactive Dashboard

Running `lctl` with no arguments launches the interactive TUI dashboard with navigation, server/site selection, and favorites.

```bash
lctl
```

### `lctl status`

Launch the dashboard overview directly.

```bash
lctl status
```

---

## Shell Completion

Generate shell completions for your preferred shell.

```bash
# Bash
lctl completion bash > /etc/bash_completion.d/lctl

# Zsh
lctl completion zsh > "${fpath[1]}/_lctl"

# Fish
lctl completion fish > ~/.config/fish/completions/lctl.fish
```
