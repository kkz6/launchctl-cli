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
| Browser sign-in | `lctl auth`, `lctl auth login`, `lctl login` |
| Sign in on a headless machine | `lctl auth --no-browser` (user approves in browser) |
| Check identity/team | `lctl whoami --json` |
| Check token status | `lctl auth status` |
| List/switch teams | `lctl teams list`, `lctl teams switch` |
| List/add/use profiles | `lctl config profiles`, `config profiles add/use` |
| One-command profile | `lctl --profile <name> …` |
| Set an approved development API origin | `lctl config set api_url <origin>` |
| Bind repository to site or Docker workload | `lctl init` |
| Live dashboard | `lctl status` |

Permissions apply to the selected team. Members have read-only resource access;
Editors can create/update resources, deploy, roll back, and change deployment
settings. Admins and Owners can delete servers/sites and manage restricted
actions such as provider connections; team settings and member role changes
require the Owner. Platform staff access is separate. Spectate sessions are
always read-only, even when the viewed user is an Owner. For a permission denial,
check the active account/team and ask its owner for the needed role; do not use
a deployment webhook to bypass the denial.

## Servers and sites

| Outcome | Command |
| --- | --- |
| List/show server | `lctl servers list`, `servers show <id>` |
| Provisioning events | `lctl servers watch <id>` |
| Metrics | `lctl servers metrics <id> [--watch]` |
| Creation options | `lctl servers create-options` |
| Create/update server | `lctl servers create`, `lctl servers update [server-id]` |
| Retry/check connection | `lctl servers retry-provision [server-id] --yes`, `lctl servers try-connection [server-id] --yes` |
| Archive/restore record | `lctl servers archive [server-id] --yes`, `lctl servers unarchive [server-id]` |
| Reboot | `lctl servers reboot [server-id] --yes` |
| Terminal | `lctl servers ssh <id> [--user <name>]` |
| Create/update/delete site | `lctl sites create`, `lctl sites update [site-id]`, `lctl sites delete [site-id] --yes` |
| Auto-deploy | `lctl sites auto-deploy enable [site-id]`, `lctl sites auto-deploy disable [site-id]` |
| List/show site | `lctl sites list --server <id>`, `sites show <id> --server <id>` |
| Server/site logs | `lctl logs --server <id> [--site <id>] [--type <name>]` |
| Pull/push environment | `lctl env pull`, `lctl env push --file <path>` |
| Remote command | `lctl run <command>`, `lctl run --history` |

## Deployments and tasks

| Outcome | Command |
| --- | --- |
| Deploy | `lctl deploy trigger <site-id> --server <id>` |
| Bounded deploy | add `--wait --timeout <seconds> --json` (works without `--ci`) |
| History/detail | `deploy list <site-id>`, `deploy show <deployment-id>` |
| Output | `deploy logs <site-id> [deployment-id] [--follow]` |
| Rollback | `lctl deploy rollback <deployment-id> --server <id> --site <id> --yes --wait --json` |
| Cancel queued deployments | `lctl deploy cancel-queued [site-id] --server <id> --yes --json` |
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

Compose stacks have dedicated `lctl docker composes` commands (`compose` alias):
list, show, create, update, deploy, reload, deployments and delete. Read
[resource-management.md](resource-management.md) for their flags and differences
from single-container applications. Container databases and advanced workload
configuration still require confirmed raw API routes.

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

## Remaining JSON API workflows

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


Dedicated commands do not cover the entire backend. Queue-worker CRUD, TLS
management, providers, backups/storage, scripts, DNS, notifications, billing,
platform updates and advanced Docker subresources still need confirmed API
paths or the dashboard. Browser authentication, uploads and integration webhooks
are not interchangeable with an authenticated JSON API request.

Older operations are not uniformly scriptable: database/firewall/cron/daemon
forms may prompt even with `--ci`; `run` execution and environment operations
produce plain output. Some action clients ignore non-2xx responses. Verify their
result through read-only state, or use the raw API to preserve HTTP errors.
Avoid `config show --json`, which includes credentials, and raw environment output.
