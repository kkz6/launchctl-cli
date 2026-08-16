# CLI production-readiness plan

Date: 2026-08-14
Status: implemented

## Outcome

`lctl` is a stable interactive and automation client for the complete launchctl
API. Common workflows remain typed, discoverable commands; every newly shipped
backend endpoint is immediately available through the authenticated `lctl api`
escape hatch. Long-running operations share one resilient WebSocket connection
and render cleanly in tmux, SSH sessions, and ordinary terminals.

## Product surface

| Area | High-level commands | Live workflow |
| --- | --- | --- |
| Authentication and context | `login`, `logout`, `whoami`, `teams`, `config profiles`, `switch` | Profile-specific hosted or approved development API origin |
| Servers | `servers list/show/reboot/ssh/metrics` | `servers watch <id>` provisioning console |
| Sites and deployments | `sites list/show`, `deploy trigger/list/show/logs/rollback` | Deployment progress and task output |
| Operations | `services`, `databases`, `ssh-keys`, `firewall`, `cron`, `ssl`, `daemons`, `logs` | Resource events through `events` |
| Projects and automation | `init`, `status`, `env`, `run`, `tasks` | Live dashboard, task console, JSON/NDJSON streams |
| New backend modules | `api <method> <path>` | Subscribe to authorized resource channels |

High-level commands are added when a workflow benefits from validation,
selection, tables, or a dedicated TUI. `lctl api` prevents release skew from
blocking scripts when the backend adds Docker, backup, DNS, notification,
script, update, or load-balancer endpoints.

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
              ├── one authenticated /api/ws connection
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
4. `https://launchctl.io`

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
- Destructive commands retain interactive confirmation outside explicit CI
  mode.

## Acceptance suite

- Unit tests cover API URL resolution, credentials, typed errors, retries,
  profile overrides, event filtering, request-body parsing, and live rendering.
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
