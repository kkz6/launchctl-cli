# lctl command map

Use `lctl <command> --help` as the source of truth for flags in the installed
version. The global flags are `--json`, `--ci`, `--profile`, and `--api-url`.

## Local CLI lifecycle

| Outcome | Command |
| --- | --- |
| Print installed version | `lctl version` |
| Check for an update | `lctl update --check [--json]` |
| Install latest stable release | `lctl update` |
| Reinstall latest stable release | `lctl update --force` |
| Disable passive notices | `LAUNCHCTL_NO_UPDATE_CHECK=1` |

Updating the CLI changes the local executable, not launchctl infrastructure.
Do it only when the user explicitly asks to install or update local software.
Homebrew installations delegate to `brew upgrade kkz6/tap/lctl`; never bypass
that ownership by replacing a Cellar binary. If Homebrew requires trust, show
`brew trust --formula kkz6/tap/lctl` and let the user make that trust decision.
After a binary update, update the separately managed skill with
`lctl ai update` only when the user requests it.

## Account and context

| Outcome | Command |
| --- | --- |
| Authenticate | `lctl login` |
| Check identity/team | `lctl whoami --json` |
| Check token status | `lctl auth status` |
| List/switch teams | `lctl teams list`, `lctl teams switch` |
| List/add/use profiles | `lctl config profiles`, `config profiles add/use` |
| One-command profile | `lctl --profile <name> …` |
| Set an approved development API origin | `lctl config set api_url <origin>` |
| Bind repository to site or Docker workload | `lctl init` |
| Live dashboard | `lctl status` |

## Servers and sites

| Outcome | Command |
| --- | --- |
| List/show server | `lctl servers list`, `servers show <id>` |
| Provisioning events | `lctl servers watch <id>` |
| Metrics | `lctl servers metrics <id> [--watch]` |
| Reboot | `lctl servers reboot <id>` |
| Terminal | `lctl servers ssh <id> [--user <name>]` |
| List/show site | `lctl sites list --server <id>`, `sites show <id> --server <id>` |
| Server/site logs | `lctl logs --server <id> [--site <id>] [--type <name>]` |
| Pull/push environment | `lctl env pull`, `lctl env push --file <path>` |
| Remote command | `lctl run <command>`, `lctl run --history` |

## Deployments and tasks

| Outcome | Command |
| --- | --- |
| Deploy | `lctl deploy trigger <site-id> --server <id>` |
| CI deploy | add `--ci --wait --timeout <seconds>` |
| History/detail | `deploy list <site-id>`, `deploy show <deployment-id>` |
| Output | `deploy logs <site-id> [deployment-id] [--follow]` |
| Rollback | `deploy rollback <deployment-id> --server <id> --site <id>` |
| List task records | `lctl tasks list --server <id>` |
| Live task | `lctl tasks watch <task-id> --server <id>` |
| Event console | `lctl events [--filter 'task.*'] [--channel resource.id]` |

## Docker projects and applications

Use the typed Docker commands for core project and application workflows. Pass
`--server` and `--project`, or initialize Docker context with `lctl init`. Read
[docker-applications.md](docker-applications.md) before mutations.

| Outcome | Command |
| --- | --- |
| List/show projects | `lctl docker projects list`, `lctl docker projects show [project-id]` |
| Create/update project | `lctl docker projects create --name <name>`, `lctl docker projects update [project-id]` |
| Delete empty project | `lctl docker projects delete [project-id] [--yes]` |
| List/show applications | `lctl docker applications list`, `lctl docker applications show [application-id]` |
| Create/update application | `lctl docker applications create`, `lctl docker applications update [application-id]` |
| Deploy/rebuild | `lctl docker applications deploy [application-id] [--wait --timeout <seconds>]` |
| Reload with current env/config | `lctl docker applications reload [application-id]` |
| Start/stop | `lctl docker applications start [application-id]`, `lctl docker applications stop [application-id]` |
| Deployment history | `lctl docker applications deployments [application-id]` |
| Delete, preserving volumes | `lctl docker applications delete [application-id] [--yes]` |
| Delete, removing volumes | add `--remove-volumes` after explicit authorization |
| Live progress | `lctl events --filter 'docker.application.*' --filter 'deployment.gha_steps'` |

Docker projects may also contain Compose stacks and container databases. List
them with confirmed raw `GET …/composes` and `GET …/databases` calls until they
gain typed commands; verify exact routes and payloads before mutations.

## Server operations

| Resource | Commands |
| --- | --- |
| Services | `services list/start/stop/restart` |
| Databases | `databases list/create/delete/users` |
| Team SSH keys | `ssh-keys list/add/delete/attach/detach/server-list` |
| Firewall | `firewall list/add/delete` |
| Cron | `cron list/add/delete` |
| Daemons | `daemons list/add/delete/restart` |
| Certificates | `ssl list` |

Server-scoped operations accept `--server <id>` or project context. Interactive
create/add flows collect required fields and destructive operations confirm.

## Complete API coverage

Use the raw client only for a confirmed backend path without a typed command:

```bash
lctl api GET /api/servers/<server-id>/backups
lctl api GET /api/servers/<server-id>/docker/projects/<project-id>/applications/<application-id>/env-vars
lctl api GET /api/scripts
lctl api GET /api/settings/notifications
lctl api POST /api/scripts --data @request.json
```

The path must start with `/api`. Request data must be valid JSON. Safe GET,
HEAD, and OPTIONS calls retry transient 429/502/503/504 responses; mutation
methods are never replayed automatically.
