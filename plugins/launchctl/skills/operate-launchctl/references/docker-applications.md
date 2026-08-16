# Docker application operations

Use this reference for Docker projects and single-container applications. The
current CLI has no typed Docker command group, so operate these resources with
the authenticated `lctl api` escape hatch and use `lctl events` for live team
events.

## Safety and terminology

- A **site deployment** uses `lctl deploy …` and the PHP/static site API.
- A **Docker application deployment** uses the nested Docker application API.
- **Deploy/Rebuild** pulls or builds the source image and recreates the
  container.
- **Reload** restarts the existing container. It does not recreate it or apply
  newly saved runtime environment values.
- Delete preserves named volumes by default. Add `?remove_volumes=true` only
  with explicit authorization to destroy the application's stored volume data.
- Runtime secret values are masked on read. Build-secret values are write-only.
  Never promise to recover a stored secret.

## Resolve the hierarchy

Resolve IDs through read-only calls. Never infer a project or application ID
from its name.

```bash
lctl servers list --json
lctl api GET /api/servers/<server-id>/docker/projects
lctl api GET /api/servers/<server-id>/docker/projects/<project-id>/applications
lctl api GET /api/servers/<server-id>/docker/projects/<project-id>/applications/<application-id>
```

Confirm that the selected server has type `docker`, the project belongs to that
server, and the application record matches the user's requested target.

## Create a project

```bash
lctl api POST /api/servers/<server-id>/docker/projects \
  --data '{"name":"acme-prod","description":"Production workloads"}'
```

Project names must be unique on the server. Project environment variables live
under `/projects/<project-id>/env-vars` and can be referenced by workloads as
`${{project.KEY}}` at deploy/run time.

## Create an application

The collection path is:

```text
/api/servers/<server-id>/docker/projects/<project-id>/applications
```

Exactly one source object must match `source_type`. The internal port defaults
to 80 and must be within 1–65535.

### Pre-built image

```json
{
  "name": "web",
  "source_type": "image",
  "internal_port": 80,
  "image": { "image": "nginx:1.27" }
}
```

The image reference needs an explicit tag. A private image accepts either a
saved `registry_credential_id` or inline `registry_username`,
`registry_password`, and optional `registry_url`; never send both mechanisms.

### Git repository

```json
{
  "name": "api",
  "source_type": "git",
  "internal_port": 3000,
  "git": {
    "repo": "https://github.com/acme/api.git",
    "branch": "main",
    "build_type": "dockerfile",
    "dockerfile_path": "services/api/Dockerfile",
    "build_location": "server"
  }
}
```

Omit `build_type` for auto-detection (root Dockerfile, otherwise Nixpacks), or
set it to `nixpacks` or `dockerfile`. `build_location` is `server` by default or
`github_actions`. Private repositories and GitHub Actions builds require a
confirmed `source_control_id`; never guess it.

### Inline Dockerfile

```json
{
  "name": "worker",
  "source_type": "dockerfile",
  "internal_port": 8080,
  "dockerfile": { "contents": "FROM caddy:2-alpine\nEXPOSE 8080" }
}
```

Prefer `--data @request.json` for multiline Dockerfiles or any payload that
contains credentials. Do not expose request-file contents in the response.

## Inspect and configure

Let `BASE` mean:

```text
/api/servers/<server-id>/docker/projects/<project-id>/applications/<application-id>
```

| Area | Confirmed route |
| --- | --- |
| Application | `GET/PATCH BASE` |
| Runtime env | `GET/POST/PUT BASE/env-vars`, `PATCH/DELETE BASE/env-vars/<id>` |
| Build secrets | `GET/POST BASE/build-secrets`, `PATCH/DELETE BASE/build-secrets/<id>` |
| Domains | `GET/POST BASE/domains`, `PATCH/DELETE BASE/domains/<id>` |
| DNS check | `GET BASE/domains/<id>/validate-dns` |
| Redirects | `GET/POST BASE/redirects`, `PATCH/DELETE BASE/redirects/<id>` |
| Volumes | `GET/POST BASE/volumes`, `PATCH/DELETE BASE/volumes/<id>` |
| Schedules | `GET/POST BASE/schedules`, `PATCH/DELETE BASE/schedules/<id>` |
| Runtime/security | `PATCH BASE/advanced` |
| Traefik config | `GET/PATCH BASE/traefik-config` |

Build settings, build secrets, and runtime environment changes require a new
deploy. `POST BASE/reload` only restarts the existing container. For a GitHub
Actions-backed application, changed build secrets mark the managed workflow out
of sync; use `POST BASE/gha/resync` and wait for the GHA synchronization event.

## Deploy with live progress

For best event coverage, start the team event stream before the mutation in a
separate terminal, tmux pane, or managed process:

```bash
lctl events --json \
  --filter 'docker.application.*' \
  --filter 'deployment.gha_steps'
```

Then queue the deployment:

```bash
lctl api POST /api/servers/<server-id>/docker/projects/<project-id>/applications/<application-id>/deploy
```

The POST response means queued or accepted, not successful. Filter event data
to the target `application_id` and record the returned deployment ID when
available.

| Event | Meaning |
| --- | --- |
| `docker.application.deploying` | Queued/building/deploying; not terminal |
| `docker.application.deployed` | Terminal success candidate |
| `docker.application.failed` | Terminal failure candidate; capture the redacted error/step |
| `deployment.gha_steps` | GitHub Actions job/step timeline; supporting progress |
| `docker.application.gha_synced` | Managed GHA workflow synchronization completed |
| `docker.application.gha_permissions_missing` | GitHub App permissions need repair |
| `docker.application.gha_installation_broken` | GitHub App installation is unavailable |

After a terminal event or stream interruption, reconcile through REST:

```bash
lctl api GET /api/servers/<server-id>/docker/projects/<project-id>/applications/<application-id>
lctl api GET /api/servers/<server-id>/docker/projects/<project-id>/applications/<application-id>/deployments
```

Report the application ID, deployment ID, final status, image/commit reference
when safe, and the failing step when applicable. Do not report success until the
application is `running` and the target deployment is `success`.

## Lifecycle and history

```text
POST   BASE/deploy
POST   BASE/reload
POST   BASE/stop
POST   BASE/start
GET    BASE/deployments
GET    BASE/deployments/<deployment-id>/gha-steps
DELETE BASE/deployments/<deployment-id>
DELETE BASE
DELETE BASE?remove_volumes=true
```

Do not delete deployment history or the application while diagnosing unless the
user explicitly asks. When deleting, GET the application and volumes first and
state whether volumes will be preserved.

## GitHub Actions controls

```text
POST BASE/gha/resync
POST BASE/gha/rotate-token
POST BASE/gha/auto-deploy       body: {"enabled": true|false}
POST BASE/gha/disable
```

These actions queue background work. Disabling GHA returns future builds to the
server but deliberately leaves the workflow file in the customer's repository;
do not delete that file without separate authorization.
