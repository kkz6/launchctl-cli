---
name: operate-launchctl
description: Inspect and manage launchctl servers, sites, deployments, tasks, databases, services, environment files, SSH keys, firewall rules, cron jobs, daemons, certificates, Docker workloads, backups, scripts, DNS, notifications, and other launchctl API resources through the lctl CLI. Use when a user asks an AI to operate, troubleshoot, automate, watch, or report on launchctl infrastructure from a terminal or CI environment.
---

# Operate launchctl

Use `lctl` as the authenticated control surface for launchctl. Prefer typed
high-level commands, stream long-running progress through the live commands,
and use `lctl api` only for backend resources without a dedicated command.

## Establish context

1. Run `command -v lctl` and `lctl version`.
2. If the binary is unavailable inside the `launchctl-cli` source checkout,
   use `go run .` from that repository as the command prefix. Do not install
   software unless the user asks.
3. Run `lctl whoami --json` before account-specific work. Never print, inspect,
   or persist the token itself.
4. Inspect `lctl config show` and `.launchctl.yml` when server, site, Docker
   project, or application context matters. Pass `--profile` when the user
   identifies a non-active profile.
5. Resolve human names to immutable IDs through list commands. Do not guess IDs.

Read [references/commands.md](references/commands.md) when choosing a command or
when the request spans more than one resource family.

Read [references/docker-applications.md](references/docker-applications.md)
before creating, configuring, deploying, troubleshooting, or deleting a Docker
project or application. Use the typed `lctl docker projects` and
`lctl docker applications` commands for core workflows; do not substitute the
PHP/static site deployment commands.

## Choose the safest interface

- Use list/show/status commands for inspection and diagnosis.
- Use the dedicated create/update/action command when it exists; it supplies
  validation, project resolution, terminal rendering, and confirmation.
- Use `--json` for machine processing. Parse JSON instead of terminal tables.
- Use `--ci` only for intentional non-interactive automation. Do not use it to
  bypass a destructive confirmation that the user has not authorized.
- Use `lctl api <method> <path>` only when no typed command exists. Confirm the
  endpoint from repository routes or documentation; do not invent API paths.
- For raw mutations, inspect the target with GET first when possible and keep
  the body minimal. Treat POST, PUT, PATCH, and DELETE as state-changing.

## Execute a request

1. Translate the user's outcome into the smallest command sequence.
2. Start with read-only discovery to resolve account, team, server, site, task,
   deployment, or workload IDs.
3. Explain any material mutation and target before running it when the user's
   request did not already make that exact change explicit.
4. Execute the command and preserve its exit code and structured error details.
5. For an asynchronous action, immediately attach the matching watcher instead
   of repeatedly polling by hand:
   - `lctl servers watch <server-id>` for provisioning.
   - `lctl tasks watch <task-id> --server <server-id>` for jobs and scripts.
   - `lctl deploy trigger <site-id> --server <id>` for an interactive deploy.
   - `lctl events --filter '<resource>.*'` for team-wide lifecycle events.
   - `lctl events --filter 'docker.application.*' --filter
     'deployment.gha_steps'` for Docker applications.
6. Reconcile final state with a list/show command and report IDs, status, and any
   remaining action. Do not claim success from a queued response alone.

## Handle common workflows

### Diagnose a failed deployment

1. Resolve the server and site.
2. Run `lctl deploy list <site-id> --server <server-id> --json`.
3. Inspect the failed record with `deploy show`.
4. Read stored output with `deploy logs`; use `--follow` only while active.
5. If a task ID is present, use `tasks watch` or `tasks list` for its output and
   exit code.
6. Report the failing step and evidence. Do not roll back or redeploy unless the
   user requests that mutation.

### Deploy from automation

Use environment-provided credentials and a bounded wait:

```bash
lctl deploy trigger "$SITE_ID" \
  --server "$SERVER_ID" \
  --ci --wait --timeout 600
```

Require `LAUNCHCTL_TOKEN` and `LAUNCHCTL_TEAM_ID`. launchctl is hosted-only.
Use `LAUNCHCTL_API_URL` only for an explicitly provided, approved launchctl development or staging origin.
Never echo these values and never invent self-hosting instructions.

### Deploy a Docker application

1. Resolve the Docker server with `lctl servers list --json`.
2. Resolve the project with `lctl docker projects list --server <server-id>
   --json`, then resolve the application with `lctl docker applications list
   --server <server-id> --project <project-id> --json`.
3. Run `lctl docker applications show <application-id> --server <server-id>
   --project <project-id> --json` and inspect its source type, build location,
   internal port, and current status before changing it.
4. Start an event watcher before the mutation when practical, then run
   `lctl docker applications deploy <application-id> --server <server-id>
   --project <project-id>`. Use `--wait --timeout <seconds>` for a bounded
   foreground deployment.
5. Follow events for the target `application_id`. Treat `deploying` as
   in-progress, `deployed` as terminal success, and `failed` as terminal
   failure. GitHub Actions step events are supporting progress, not final
   application state.
6. Re-fetch the application with `docker applications show` and its history
   with `docker applications deployments` after the terminal event. Do not
   report success solely from the queued API response or WebSocket event.

Use `reload` when the user wants to recreate the container from the image
already present on the server while applying the current runtime environment
and container configuration. Reload does not rebuild or pull an image and does
not add a deployment-history row. Use `deploy` when the source image or build
must be refreshed.

### Watch infrastructure in tmux

Use interactive watchers for a human session and `--json` NDJSON for an agent
or pipeline. Filter event lines with `jq` only after `lctl` has emitted complete
JSON objects.

## Guardrails

- Preserve the active profile unless the user asks to switch it permanently;
  prefer the one-command `--profile` flag.
- Do not write remote `.env` values without reviewing the redacted diff and
  explicit authorization.
- Distinguish detaching an SSH key from deleting the team key.
- Distinguish stopping/restarting a service from rebooting its server.
- Never expose token, password, private key, environment secret, or raw secret
  API fields in the response. Summarize redacted results.
- Keep Docker build secrets write-only. Do not attempt to retrieve their values,
  and never put registry passwords or secrets directly in a shell command when
  a JSON request file or saved credential can be used.
- Preserve Docker named volumes on application deletion unless the user
  explicitly authorizes the `--remove-volumes` flag after reviewing the target.
- Distinguish Docker **Deploy/Rebuild** from **Reload**: deploy rebuilds or pulls
  the source and recreates the container; reload reuses the on-server image but
  recreates the container with its current environment and configuration.
- If WebSocket progress is interrupted, rely on the client's reconnect first,
  then reconcile through REST before declaring the operation failed.
