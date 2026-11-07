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
