# launchctl

A command-line tool for managing servers, sites, and deployments on the [launchctl](https://launchctl.dev) platform.

```
 _                   _       _   _
| |__ _ _  _ _ _  __| |_  __| |_| |
| / _` | || | ' \/ _| ' \/ _|  _| |
|_\__,_|\_,_|_||_\__|_||_\__|\__|_|
```

## Features

- **Server Management** — List, create, reboot, and SSH into servers
- **Site Management** — Create and manage sites across your servers
- **Zero-Downtime Deploys** — Trigger deployments with live log streaming via WebSocket
- **Dashboard** — Full-screen terminal UI with server overview and recent activity
- **Team Switching** — Manage and switch between teams
- **JSON Output** — Machine-readable output for scripting with `--json`

## Installation

### From source

```bash
git clone https://github.com/kkz6/launchctl-cli.git
cd launchctl-cli
make build
```

The binary will be at `./bin/launchctl`.

### Go install

```bash
go install github.com/kkz6/launchctl@latest
```

## Quick Start

```bash
# Authenticate with your account
launchctl login

# View your dashboard
launchctl status

# List all servers
launchctl servers list

# Deploy a site
launchctl deploy trigger --server <server-id> --site <site-id>
```

## Commands

### Authentication

| Command | Description |
|---------|-------------|
| `launchctl login` | Interactive login with email and password |
| `launchctl logout` | Log out and clear stored credentials |

### Servers

| Command | Description |
|---------|-------------|
| `launchctl servers list` | List all servers |
| `launchctl servers show <id>` | Show server details |
| `launchctl servers create` | Interactive server creation |
| `launchctl servers reboot <id>` | Reboot a server |
| `launchctl servers ssh <id>` | SSH into a server |
| `launchctl servers metrics <id>` | Show latest server metrics |

### Sites

| Command | Description |
|---------|-------------|
| `launchctl sites list --server <id>` | List sites on a server |
| `launchctl sites show <id> --server <id>` | Show site details |
| `launchctl sites create --server <id>` | Interactive site creation |

### Deployments

| Command | Description |
|---------|-------------|
| `launchctl deploy trigger --server <id> --site <id>` | Deploy with live log streaming |
| `launchctl deploy list --server <id> --site <id>` | List deployment history |
| `launchctl deploy show <id> --server <id> --site <id>` | Show deployment details |
| `launchctl deploy rollback --server <id> --site <id>` | Rollback to previous release |

### Teams

| Command | Description |
|---------|-------------|
| `launchctl teams list` | List your teams |
| `launchctl teams switch` | Interactive team switcher |
| `launchctl teams members` | List team members |

### Dashboard

```bash
launchctl status
```

Full-screen TUI with server overview and recent deployments. Auto-refreshes every 30 seconds. Press `r` to refresh manually, `q` to quit.

### Configuration

```bash
# Show current config
launchctl config show

# Set a config value
launchctl config set api_url https://api.launchctl.dev
```

Config is stored at `~/.config/launchctl/config.json`.

## Global Flags

| Flag | Description |
|------|-------------|
| `--json` | Output in JSON format |
| `--api-url` | Override the API URL |

## Requirements

- Go 1.24+
- A [launchctl](https://launchctl.dev) account

## License

Private — All rights reserved.
