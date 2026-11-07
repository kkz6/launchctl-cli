# CLI production-readiness plan

Date: 2026-08-14
Status: core client implemented; feature coverage reviewed 2026-11-07

## Outcome

`lctl` provides interactive and automation workflows for launchctl.
Common workflows have typed, discoverable commands; other JSON REST endpoints
can be called through the authenticated `lctl api` command. Multipart uploads,
browser authentication, and WebSocket protocols require specialized clients.
Long-running operations share one resilient WebSocket connection
and render cleanly in tmux, SSH sessions, and ordinary terminals.

## Product surface

| Area | High-level commands | Live workflow |
| --- | --- | --- |
| Authentication and context | `auth`, `login`, `logout`, `whoami`, `teams`, `config profiles`, `switch` | Browser device approval and profile-specific API origin |
| Servers | `servers list/show/create/update/reboot/ssh/metrics`, lifecycle actions | `servers watch <id>` provisioning console |
| Sites and deployments | Site lifecycle and auto-deploy controls; deployment, rollback, logs, queued cancellation | Deployment progress and task output |
| Docker workloads | `docker projects`, `docker applications`, `docker composes` | Application events plus bounded deploy waiting |
| Operations | `services`, `databases`, `ssh-keys`, `firewall`, `cron`, `ssl`, `daemons`, `logs` | Resource events through `events` |
| Repository context and automation | `init`, `status`, `env`, `run`, `tasks` | Live dashboard, task console, JSON/NDJSON streams |
| New backend modules | `api <method> <path>` | Subscribe to authorized resource channels |

High-level commands are added when a workflow benefits from validation,
selection, tables, or a dedicated TUI. `lctl api` prevents release skew from
blocking scripts when the backend adds backup, DNS, notification, script,
update, load-balancer, or advanced Docker subresource endpoints.

## Runtime architecture

```text
Cobra command / Bubble Tea model
        │
        ├── HTTP client
        │     ├── profile + env + --api-url resolution
        │     ├── bearer/team headers
        │     ├── bounded response bodies and typed API errors
        │     └── retry GET/HEAD/OPTIONS on transient failures
        │
        └── WS manager
              ├── one authenticated /ws connection
              ├── ping/pong deadlines
              ├── exponential reconnect with jitter
              ├── subscription registry + replay
              └── buffered event and connection-state streams
```

The dashboard uses WebSocket events to refresh immediately and retains a
30-second REST reconciliation interval. Task, server, deployment, and generic
event consoles expose reconnect state and continue after API restarts or laptop
sleep. JSON modes emit one complete event per line for `jq` and CI consumers.

## WebSocket contract

Channels use `resource.id` syntax:

- `team.<team-id>` and `user.<user-id>`
- `server.<server-id>`, `site.<site-id>`, `deployment.<deployment-id>`
- `task.<task-id>`

The API auto-subscribes a client to its authenticated team channel. Explicit
subscriptions receive `subscription.succeeded` or `subscription.error`.
Resource subscriptions are authorized against database ownership and fail
closed for malformed or unknown scopes. Deployment consumers also validate the
payload's `deployment_id`; a team-level event can never complete the wrong TUI.

## Configuration precedence

Highest to lowest:

1. `--api-url`, `--profile`
2. `LAUNCHCTL_API_URL`, `LAUNCHCTL_TOKEN`, `LAUNCHCTL_TEAM_ID`
3. active profile in `~/.config/launchctl/config.json`
4. `https://api.launchctl.io`

API origins are stored per profile so production and approved launchctl
development or staging accounts can coexist without separate config files.
This override is not a supported self-hosting interface; launchctl is currently
a hosted service.

## Stability and security gates

- HTTP request/response size limits and deterministic validation errors.
- Retries only for idempotent methods; mutations are never replayed.
- WebSocket read limits, heartbeat deadlines, bounded queues, and idempotent
  shutdown.
- Server-side authorization for every channel subscription.
- TUI views filter resource identifiers and preserve REST reconciliation.
- Newly added destructive commands require explicit `--yes` when unattended;
  older operations still need an automation behavior audit.

## Acceptance suite

- Unit tests cover API URL resolution, credentials, typed errors, retries,
  profile overrides, Docker project/application route construction and safety,
  event filtering, request-body parsing, and live rendering.
- WebSocket integration tests force a disconnect and verify reconnect plus
  subscription replay.
- Backend tests cover cross-team authorization, task ownership joins,
  broadcast delivery, safe close, and idempotent shutdown.
- Required release checks: `go test -race ./...`, `go vet ./...`, CLI build,
  backend tests, frontend typecheck/tests/build, and skill validation.

## Follow-up policy

When a backend endpoint becomes a frequent human workflow, add a typed command
on top of the existing client without duplicating transport code. New live
features must use `WSManager`, include a JSON stream, enforce resource filtering,
and keep a periodic REST reconciliation path when state convergence matters.

## Feature coverage review — 2026-11-07

The authenticated `api` command remains available for backend operations that
do not yet have dedicated commands. Typed command coverage is incomplete;
the following work should land in small, tested groups.

| Area | Existing dedicated coverage | Remaining dedicated workflows |
| --- | --- | --- |
| Native deployments | Trigger, list, detail, rollback, logs, cancel queued | Advanced deployment configuration |
| Native sites | Create/update/delete, auto-deploy, environment files, remote commands, logs | Advanced repository setup, queue workers, site configuration |
| Servers | Create/options/update, retry provisioning, connection check, archive/unarchive, SSH, metrics, provisioning watch, reboot | Provider administration, deletion, advanced settings |
| Docker | Project/application CRUD, raw Compose creation and lifecycle, deployment history, deploy/wait, start/stop/reload | Git Compose creation, managed databases, domains, variables, volumes, build secrets, schedules |
| Operations | Services, database creation/deletion/users listing, SSH keys, cron, firewall, daemons, certificate listing | Missing update/install actions, database-user management, PHP versions, certificate lifecycle |
| Backups and storage | Raw API access | Backup configuration/run, storage destinations, Docker backup history/restore |
| Scripts and DNS | Raw API access | Saved scripts and execution history; providers, domains, records |
| Account and team | Authentication, profiles, team selection/members | Team administration, credentials, integrations, tokens, notification settings, billing |

The first implementation group fixes native deployment automation and progress:
strict JSON output, explicit rollback/cancellation confirmation, project defaults,
bounded waits, terminal-state reconciliation, and exact task-log lookup. Its
backend companion change scopes task detail responses to the requested server.

The next implementation group adds native site/server lifecycle, raw Docker
Compose workflows, and browser device login, with command-level tests for JSON,
noninteractive behavior, failure exits, and resource scoping. `RebootServer` now
returns non-success HTTP responses as errors.

Remaining work includes Docker databases and advanced application configuration,
backups/scripts, operations and account administration, and older commands whose
confirmation, JSON output, or HTTP error handling is inconsistent. The separate
documentation repository's `content/docs/cli/feature-coverage.mdx` records the
module-by-module audit and distinguishes dedicated commands, JSON REST access,
and specialized protocols. Source documentation describes unreleased changes;
browser login requires the matching API and web application release.
