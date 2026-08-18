# Docker project and application operations

Use this reference for Docker projects and single-container applications. Use
the typed `lctl docker projects` and `lctl docker applications` commands for
core workflows. Use `lctl api` only for confirmed advanced subresources that do
not yet have a typed command.

## Safety and terminology

- A **site deployment** uses `lctl deploy …` and the PHP/static site API.
- A **Docker application deployment** uses `lctl docker applications deploy`.
- **Deploy/Rebuild** pulls or builds the configured source and recreates the
  container.
- **Reload** reuses the image already present on the server and recreates the
  container with the current runtime environment and container configuration.
  It does not rebuild or pull an image and does not create a deployment-history
  row.
- Application deletion preserves named volumes by default. Use
  `--remove-volumes` only with explicit authorization to destroy stored volume
  data.
- Runtime secret values are masked on read. Build-secret values are write-only.
  Never promise to recover a stored secret.

## Resolve the hierarchy

Resolve immutable IDs through read-only typed commands. Never infer a project
or application ID from its name.

```bash
lctl servers list --json
lctl docker projects list --server <server-id> --json
lctl docker applications list \
  --server <server-id> \
  --project <project-id> \
  --json
lctl docker applications show <application-id> \
  --server <server-id> \
  --project <project-id> \
  --json
```

Confirm that the selected server has type `docker`, the project belongs to that
server, and the application record matches the requested target.

For a repository that maps to one Docker application, save context with:

```bash
lctl init \
  --server <server-id> \
  --project <project-id> \
  --application <application-id>
```

This records `server`, `docker_project`, and `docker_application` in
`.launchctl.yml`. Project and application commands can then resolve omitted
server, project, and application targets without changing the active profile.
Explicit arguments and flags take precedence.

## Manage projects

```bash
# Create
lctl docker projects create \
  --server <server-id> \
  --name acme-prod \
  --description "Production workloads"

# List and inspect
lctl docker projects list --server <server-id>
lctl docker projects show <project-id> --server <server-id>

# Rename or change the description
lctl docker projects update <project-id> \
  --server <server-id> \
  --name acme-production \
  --description "Production Docker workloads"

# Delete after every workload has been removed
lctl docker projects delete <project-id> --server <server-id>
```

Project names must be unique on the server. Project deletion is blocked while
applications, Compose stacks, or container databases remain. Deletion prompts
for confirmation; `--yes` is the explicit non-interactive confirmation.

Project environment variables remain an advanced raw-API workflow. They live
under `/projects/<project-id>/env-vars` and can be referenced by workloads as
`${{project.KEY}}` at deploy or reload time.

## Create an application

Every create command must select exactly one `--source`: `image`, `git`, or
`dockerfile`. The internal port defaults to 80 and must be within 1–65535.

### Pre-built image

```bash
lctl docker applications create \
  --server <server-id> \
  --project <project-id> \
  --name web \
  --source image \
  --image nginx:1.27 \
  --port 80
```

The image reference needs an explicit tag. For a private image backed by a
saved credential, add `--registry-credential <credential-id>`. The typed command
does not accept a registry password on the command line; use an approved saved
credential or a protected JSON request file with `lctl api` for an exceptional
inline-credential workflow.

### Git repository

```bash
lctl docker applications create \
  --server <server-id> \
  --project <project-id> \
  --name api \
  --source git \
  --repo https://github.com/acme/api.git \
  --branch main \
  --build-type dockerfile \
  --dockerfile-path services/api/Dockerfile \
  --build-location server \
  --port 3000
```

Omit `--build-type` for auto-detection (root Dockerfile, otherwise Nixpacks),
or set it to `nixpacks` or `dockerfile`. `--build-location` is `server` or
`github_actions`. Private repositories and GitHub Actions builds require a
confirmed `--source-control <source-control-id>`; never guess that ID.

### Inline Dockerfile

```bash
lctl docker applications create \
  --server <server-id> \
  --project <project-id> \
  --name worker \
  --source dockerfile \
  --dockerfile ./Dockerfile \
  --port 8080
```

Use `--dockerfile -` to read Dockerfile content from standard input. Prefer a
file or stdin over embedding multiline content in shell history.

## Inspect and update

```bash
lctl docker applications show <application-id> \
  --server <server-id> \
  --project <project-id>

lctl docker applications update <application-id> \
  --server <server-id> \
  --project <project-id> \
  --name api-v2
```

For a Git source, update build settings with `--build-type
auto|nixpacks|dockerfile` and `--dockerfile-path <path>`. Updating GitHub
Actions build settings may leave its managed workflow out of sync; inspect the
application and use the confirmed GHA re-sync route when required.

Let `BASE` mean:

```text
/api/servers/<server-id>/docker/projects/<project-id>/applications/<application-id>
```

These advanced subresources do not yet have typed commands:

| Area | Confirmed route |
| --- | --- |
| Project runtime env | `GET/POST /api/servers/<server-id>/docker/projects/<project-id>/env-vars` |
| Application runtime env | `GET/POST/PUT BASE/env-vars`, `PATCH/DELETE BASE/env-vars/<id>` |
| Build secrets | `GET/POST BASE/build-secrets`, `PATCH/DELETE BASE/build-secrets/<id>` |
| Domains | `GET/POST BASE/domains`, `PATCH/DELETE BASE/domains/<id>` |
| DNS check | `GET BASE/domains/<id>/validate-dns` |
| Redirects | `GET/POST BASE/redirects`, `PATCH/DELETE BASE/redirects/<id>` |
| Volumes | `GET/POST BASE/volumes`, `PATCH/DELETE BASE/volumes/<id>` |
| Schedules | `GET/POST BASE/schedules`, `PATCH/DELETE BASE/schedules/<id>` |
| Runtime/security | `PATCH BASE/advanced` |
| Traefik config | `GET/PATCH BASE/traefik-config` |

Build settings and build secrets require a new deploy. Runtime environment and
container-configuration changes can be applied with `reload`, which recreates
the container from the existing on-server image without rebuilding it.

## Deploy with live progress

For best event coverage, start the team event stream before the mutation in a
separate terminal or tmux pane:

```bash
lctl events --json \
  --filter 'docker.application.*' \
  --filter 'deployment.gha_steps'
```

Then queue the deployment:

```bash
lctl docker applications deploy <application-id> \
  --server <server-id> \
  --project <project-id>
```

For a bounded foreground deployment, add `--wait --timeout <seconds>`. The
initial response means queued or accepted, not successful.

| Event | Meaning |
| --- | --- |
| `docker.application.deploying` | Queued/building/deploying; not terminal |
| `docker.application.deployed` | Terminal success candidate |
| `docker.application.failed` | Terminal failure candidate; capture the redacted error/step |
| `deployment.gha_steps` | GitHub Actions job/step timeline; supporting progress |
| `docker.application.gha_synced` | Managed GHA workflow synchronization completed |
| `docker.application.gha_permissions_missing` | GitHub App permissions need repair |
| `docker.application.gha_installation_broken` | GitHub App installation is unavailable |

After a terminal event, timeout, or stream interruption, reconcile through the
typed read commands:

```bash
lctl docker applications show <application-id> \
  --server <server-id> \
  --project <project-id> \
  --json
lctl docker applications deployments <application-id> \
  --server <server-id> \
  --project <project-id> \
  --json
```

Report the application ID, deployment ID, final status, image/commit reference
when safe, and the failing step when applicable. Do not report success until
the application is `running` and the target deployment is `success`.

## Lifecycle and deletion

```bash
lctl docker applications reload <application-id> --server <server-id> --project <project-id>
lctl docker applications stop <application-id> --server <server-id> --project <project-id>
lctl docker applications start <application-id> --server <server-id> --project <project-id>
lctl docker applications deployments <application-id> --server <server-id> --project <project-id>
```

Reload is the correct action for applying saved runtime environment or
container configuration without rebuilding. Deploy is the correct action when
the source image, repository, Dockerfile, build settings, or build secrets must
be refreshed.

Delete the application while preserving named volumes:

```bash
lctl docker applications delete <application-id> \
  --server <server-id> \
  --project <project-id>
```

Only after explicit authorization to destroy named-volume data:

```bash
lctl docker applications delete <application-id> \
  --server <server-id> \
  --project <project-id> \
  --remove-volumes
```

Deletion prompts for confirmation. Use `--yes` only when the user has already
authorized the exact target and deletion behavior. Do not delete deployment
history or the application while diagnosing unless the user explicitly asks.

## GitHub Actions controls

The following controls remain raw-API workflows:

```text
POST BASE/gha/resync
POST BASE/gha/rotate-token
POST BASE/gha/auto-deploy       body: {"enabled": true|false}
POST BASE/gha/disable
GET  BASE/deployments/<deployment-id>/gha-steps
DELETE BASE/deployments/<deployment-id>
```

These actions queue background work. Disabling GHA returns future builds to the
server but deliberately leaves the workflow file in the customer's repository;
do not delete that file without separate authorization.

## Compose stacks and container databases

Docker projects can also contain Compose stacks and container databases. They
do not yet have typed `lctl docker` commands. Use only confirmed nested raw API
routes after read-only discovery; do not substitute application commands for a
different workload kind.
