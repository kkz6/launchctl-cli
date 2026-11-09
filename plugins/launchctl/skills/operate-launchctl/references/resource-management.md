# Server, site and Compose management

Resolve IDs before mutations. Use the installed command help to verify support;
these workflows require a CLI release containing the new lifecycle commands.

## Servers

```bash
lctl servers create-options
lctl servers create --name staging --provider custom_server --type php --ip 192.0.2.10 --yes --json
lctl servers update <server-id> --monitoring=false --description='' --json
lctl servers retry-provision <server-id> --yes --json
lctl servers try-connection <server-id> --yes --json
lctl servers archive <server-id> --yes --json
lctl servers unarchive <server-id> --json
```

Creation requires name/provider/type. Cloud creation also requires a connected
credential ID, region and size and can incur charges. Custom creation only
registers the machine: complete its dashboard setup before connection checking.
The server type is `php`, `database`, `loadbalancer` or `docker`; use current
creation options for provider/OS/software values.

Updates send only explicit flags; false and empty descriptions are preserved.
`--ssh-port` changes the recorded port, not the remote SSH configuration.
Provision/reboot/connection receipts are queued. Watch and re-fetch the server.
Archive changes the record and does not stop or destroy the VM. Server deletion
and advanced provider/setup management remain API/dashboard workflows.

## Sites

```bash
lctl sites create --server <server-id> --address example.com --type laravel --php-version php84 --json
lctl sites update <site-id> --server <server-id> --branch main --release-retention 5 --queue-deployments=false --json
lctl sites auto-deploy enable <site-id> --server <server-id> --json
lctl sites auto-deploy disable <site-id> --server <server-id> --json
lctl sites delete <site-id> --server <server-id> --yes --json
```

Creation requires address/type/PHP version; supported types are laravel,
wordpress, static, generic and phpmyadmin. Repository options are saved source
control ID, numeric repository ID, and branch. Updates support PHP version,
web folder, branch, retention (1–50) and deployment queuing. Empty strings clear
supported fields. A pending PHP version is not yet active; deletion_requested
is not completed cleanup. Use raw API routes for hooks/TLS/queue-worker settings.

Site deployment tokens and webhook URLs are credentials. Authorized site-detail
responses expose them only to deployment-capable Editor/Admin/Owner sessions,
never Members or Spectate sessions. Lists and shared WebSocket messages omit
them. Do not print these values; regenerating a token requires Admin/Owner access.

## Staff administration

Staff actions have no dedicated CLI commands. Use confirmed `/admin/*` routes
with `lctl api` (prepend `/api` for the CLI), and resolve the user/site from
read-only admin responses first. These are platform permissions, separate from
team membership. Support staff can inspect; the mutations below require
`super_admin` and are unavailable while impersonating another user.

- `PATCH /admin/users/<user-id>/staff-role` requires an explicit `staff_role`:
  `null` removes staff access, or use `support` / `super_admin`. Self role
  changes and demoting the last super administrator are blocked. A successful
  response includes the saved role; subsequent admin requests use the new role.
- `DELETE /admin/users/<user-id>` blocks staff accounts; paid billing history
  remains a blocker after demotion. Do not demote someone merely to work around
  a failed deletion unless that role change is also authorized.
- `DELETE /admin/sites/<site-id>` requires `confirmation_domain` matching the
  exact stored address. The API uses the site's actual owning server/team and
  returns `deletion_requested`. Re-fetch `/admin/users/<user-id>/sites`:
  `uninstallation_requested_at` means pending; failure retains the record with
  `uninstallation_failed_at` and permits a later explicit retry. A successful
  request alone does not mean remote files are gone.
- `DELETE /admin/servers/<server-id>` requires the exact `confirmation_name`.
  Remove all sites and Docker projects first; starting, provisioning, and
  deleting servers are blocked. Custom-server deletion removes only the record;
  ordinary cloud deletion queues provider instance destruction using saved
  credentials, without SSH. Provider failure retains the record for an explicit
  retry. A queued response is not completion; re-fetch the server.
- `DELETE /admin/servers/<server-id>/record` removes only the launchctl record
  and requires the exact `confirmation_name` in the request body. Use it only
  when that outcome is explicitly authorized, such as when cloud access is
  revoked. A successful response returns `status: "removed"`.
  It keeps the same authorization, child-resource, and lifecycle guards, makes
  no provider request, and leaves the machine, its data, and provider billing
  unchanged. Do not use record removal as a fallback for failed cloud deletion
  unless the user authorized that distinct outcome. If the dedicated endpoint
  is unavailable, stop; never substitute the ordinary server DELETE endpoint.

Raw mutations have no built-in confirmation. Use them only for an authorized,
resolved target and never treat a role change as authorization to delete data.

## Compose stacks

Use `docker composes` (`docker compose` alias), scoped by server and project.
Creation accepts local YAML up to 128 KiB and does not deploy immediately.

```bash
lctl docker composes list --server <server-id> --project <project-id> --json
lctl docker composes create --server <server-id> --project <project-id> --name web --file compose.yml --json
lctl docker composes show <compose-id> --server <server-id> --project <project-id> --json
lctl docker composes update <compose-id> --server <server-id> --project <project-id> --env-file .env.production
lctl docker composes deploy <compose-id> --server <server-id> --project <project-id> --wait --timeout 600 --json
lctl docker composes deployments <compose-id> --server <server-id> --project <project-id> --json
lctl docker composes reload <compose-id> --server <server-id> --project <project-id> --yes --json
lctl docker composes delete <compose-id> --server <server-id> --project <project-id> --yes --json
```

Only name, environment-file contents and run-command override are patchable.
An empty environment file clears variables; `--run-command=''` resets the override.
Deploy or reload applies environment updates; run-command changes require deploy.
Source/YAML is immutable through the current update endpoint. Git creation,
registry attachment and container database management remain API/dashboard work.

Default JSON output redacts YAML/environment/command contents. Do not request
`show --json --show-config` unless viewing secret-bearing configuration is needed
and authorized, and never repeat those raw values in a response.

Deploy waiting tracks the exact deployment record, not application health.
Reload and deletion return queued receipts; reconcile the stack afterward.
Delete preserves named volumes. Add `--remove-volumes` only when the user has
explicitly authorized destroying that volume data. Pass `--yes` only for an
already authorized mutation; it is required for unattended reload/deletion.
