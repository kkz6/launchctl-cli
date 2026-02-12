# launchctl CLI — Feature Roadmap

Complete design document for all planned CLI features, organized by priority tier.

---

## Currently Implemented

| Command | Description |
|---------|-------------|
| `launchctl login` | Authenticate with API token + 2FA challenge |
| `launchctl logout` | Clear credentials |
| `launchctl version` | Display CLI version |
| `launchctl status` | Dashboard TUI (servers, recent deployments) |
| `launchctl config show` | Show current config |
| `launchctl config set` | Set config values |
| `launchctl servers list` | List all servers |
| `launchctl servers show <id>` | Show server details |
| `launchctl servers create` | Create a server (interactive) |
| `launchctl servers reboot <id>` | Reboot a server |
| `launchctl servers ssh <id>` | SSH into a server |
| `launchctl servers metrics <id>` | Show server metrics |
| `launchctl sites list --server <id>` | List sites on a server |
| `launchctl sites show <id>` | Show site details |
| `launchctl sites create --server <id>` | Create a site (interactive) |
| `launchctl deploy trigger` | Trigger deployment with live streaming |
| `launchctl deploy list` | List deployments |
| `launchctl deploy show <id>` | Show deployment details |
| `launchctl deploy rollback` | Rollback to a previous deployment |
| `launchctl teams list` | List teams |
| `launchctl teams switch` | Switch active team |
| `launchctl teams members` | List team members |

---

## Tier 1: Daily Workflow (High Impact)

### 1.1 `launchctl init` — Project Configuration

Create a `.launchctl.yml` in the repo root to bind the project to a server + site. Eliminates repetitive `--server` and `--site` flags.

**Commands:**

```
launchctl init                                Interactive setup
launchctl init --server <id> --site <id>      Non-interactive
```

**Config File (`.launchctl.yml`):**

```yaml
server: 01HQGZ7V8K3M5N2P4R6T8W0X
site: 01HQGZ7V8K3M5N2P4R6T8W0Y
```

**How It Works:**
- All commands that accept `--server` and `--site` flags check for `.launchctl.yml` in the current directory (and parent directories, like `.git`)
- Explicit flags always override the config file
- `launchctl init` uses `huh` forms to pick server and site interactively

**Commands That Benefit:**

```bash
# Before (without init)
launchctl deploy trigger --server abc --site def
launchctl env pull --server abc --site def
launchctl logs --server abc --site def --type app
launchctl run --server abc --site def "node migrate.js"

# After (with .launchctl.yml in repo)
launchctl deploy trigger
launchctl env pull
launchctl logs --type app
launchctl run "node migrate.js"
```

**Multi-Environment Support:**

```yaml
default: production

environments:
  production:
    server: 01HQGZ7V8K3M5N2P4R6T8W0X
    site: 01HQGZ7V8K3M5N2P4R6T8W0Y
  staging:
    server: 01HQGZ7V8K3M5N2P4R6T8W0Z
    site: 01HQGZ7V8K3M5N2P4R6T8W0A
```

Then: `launchctl deploy trigger --env staging`

**Implementation Notes:**
- Walk up directories until `.launchctl.yml` or `.git` is found
- Add to `.gitignore` template (contains server-specific IDs)

**Files to Create:**

| File | Purpose |
|------|---------|
| `cmd/init.go` | Init command |
| `internal/config/project.go` | Project config loader (.launchctl.yml) |

**Files to Modify:**

| File | Change |
|------|--------|
| `cmd/root.go` | Load project config, inject defaults into flags |

---

### 1.2 `launchctl env` — Environment File Management

Push and pull `.env` files between local machine and remote servers.

**Commands:**

```
launchctl env pull --server <id> --site <id>           Pull .env to stdout
launchctl env pull --server <id> --site <id> -o .env   Pull .env to file
launchctl env push --server <id> --site <id> -f .env   Push local .env to server
launchctl env edit --server <id> --site <id>            Open .env in $EDITOR
launchctl env diff --server <id> --site <id> -f .env   Diff local vs remote
```

**API Endpoints Used:**

| Method | Endpoint | Purpose |
|--------|----------|---------|
| `GET` | `/servers/:serverId/sites/:id/files` | List available files (get encoded .env param) |
| `GET` | `/servers/:serverId/sites/:id/files/:file` | Get .env content |
| `PUT` | `/servers/:serverId/sites/:id/files/:file` | Update .env content |

**Implementation Notes:**
- The `:file` parameter is encrypted via `EncodeFileRouteParam()` — CLI must first call the list endpoint to get the encoded param for the .env file
- Available file types depend on site type and stack
- Environment file path varies by deploy mode:
  - Zero-downtime: `/sites/shared/.env`
  - Standard: `/sites/repository/.env`

**Flags:**

| Flag | Description |
|------|-------------|
| `--server`, `-s` | Server ID (required) |
| `--site` | Site ID (required) |
| `--output`, `-o` | Output file path (default: stdout) |
| `--file`, `-f` | Input file path for push/diff |
| `--confirm` | Skip confirmation prompt on push |

**Safety:**
- `env push` shows a diff and requires confirmation before overwriting
- `env push` creates a backup of the remote .env before writing

**Files to Create:**

| File | Purpose |
|------|---------|
| `cmd/env/env.go` | Parent command |
| `cmd/env/pull.go` | Pull .env from server |
| `cmd/env/push.go` | Push .env to server |
| `cmd/env/edit.go` | Edit remote .env in $EDITOR |
| `cmd/env/diff.go` | Diff local vs remote |
| `internal/api/files.go` | API methods for file management |

---

### 1.3 `launchctl logs` — Log Tailing

Tail server and site logs in real-time with color-coded output.

**Commands:**

```
launchctl logs --server <id>                            List available logs
launchctl logs --server <id> --type app                 Tail application log
launchctl logs --server <id> --site <id> --type access  Tail access log
launchctl logs --server <id> --type database            Tail database log
launchctl logs --server <id> --type webserver            Tail web server log
```

**API Endpoints Used:**

| Method | Endpoint | Purpose |
|--------|----------|---------|
| `GET` | `/servers/:id/logs` | List available server logs |
| `GET` | `/servers/:id/logs/:log` | Get log content (encoded param) |
| `GET` | `/servers/:serverId/sites/:id/logs` | List available site logs |

**Available Log Types (by installed software):**
- Web server access/error logs
- Database error logs
- Process manager logs
- Cache server logs
- Site-specific application logs

**Implementation Notes:**
- Like .env files, the `:log` parameter is encrypted — must first list logs to get the encoded param
- For "tailing", poll the log endpoint every 2 seconds and show only new lines since last fetch
- Alternative: if the WebSocket `task.output` event supports log streaming, prefer that
- Color-code log levels: ERROR (red), WARNING (yellow), INFO (green), DEBUG (dim)

**Flags:**

| Flag | Description |
|------|-------------|
| `--server`, `-s` | Server ID (required) |
| `--site` | Site ID (for site-specific logs) |
| `--type`, `-t` | Log type (app, access, database, webserver, etc.) |
| `--lines`, `-n` | Number of lines to show (default: 50) |
| `--follow`, `-f` | Follow mode — continuously tail (default: true) |

**Files to Create:**

| File | Purpose |
|------|---------|
| `cmd/logs.go` | Log tailing command |
| `internal/api/logs.go` | API methods for log endpoints |

---

### 1.4 `launchctl run` — Execute Commands

Run a command on a server/site without opening a full SSH session.

**Commands:**

```
launchctl run --server <id> --site <id> "node migrate.js"
launchctl run --server <id> --site <id> --command "npm run build"
launchctl run --server <id> --site <id> --history           List previous commands
```

**API Endpoints Used:**

| Method | Endpoint | Purpose |
|--------|----------|---------|
| `POST` | `/servers/:serverId/sites/:id/commands` | Execute command |
| `GET` | `/servers/:serverId/sites/:id/commands` | List command history |
| `DELETE` | `/servers/:serverId/sites/:id/commands/:commandId` | Delete command |

**Request:**
```json
{
  "command": "node migrate.js"
}
```

**Response:**
```json
{
  "id": "abc123",
  "command": "node migrate.js",
  "status": "pending",
  "output": null,
  "exit_code": null
}
```

**Implementation Notes:**
- After submitting, poll the command status until it's `finished` or `failed`
- Show a spinner while waiting, then display output
- If WebSocket is available, subscribe to `task.output` events for live streaming
- Command statuses: `pending` → `running` → `finished`/`failed`

**Flags:**

| Flag | Description |
|------|-------------|
| `--server`, `-s` | Server ID (required) |
| `--site` | Site ID (required) |
| `--command` | Command string (alternative to positional arg) |
| `--history` | List previous commands instead of running one |

**Files to Create:**

| File | Purpose |
|------|---------|
| `cmd/run.go` | Run command |
| `internal/api/commands.go` | API methods for commands |

---

### 1.5 `launchctl deploy logs` — Deployment Logs

View output logs for the most recent or a specific deployment.

**Commands:**

```
launchctl deploy logs --server <id> --site <id>            Latest deployment log
launchctl deploy logs <deployment-id> --server <id> --site <id>   Specific deployment log
```

**Implementation Notes:**
- Fetches deployment details and displays the output/log content
- Useful for reviewing deployments after the fact (vs the live TUI during trigger)
- With `--follow`, stream logs in real-time for an in-progress deployment

**Files to Modify:**

| File | Change |
|------|--------|
| `cmd/deploy/deploy.go` | Add `logs` subcommand |

**Files to Create:**

| File | Purpose |
|------|---------|
| `cmd/deploy/logs.go` | Deployment log viewer |

---

### 1.6 CI/CD Mode

Non-interactive mode for automation pipelines (GitHub Actions, GitLab CI).

**Design:**
- `--ci` global flag disables all interactive prompts
- Auth via environment variable: `LAUNCHCTL_TOKEN` (API access token)
- Also support: `LAUNCHCTL_API_URL`, `LAUNCHCTL_TEAM_ID`
- Exit codes: 0 success, 1 error, 2 timeout
- All output goes to stderr except data (which goes to stdout for piping)

**Usage in GitHub Actions:**

```yaml
- name: Deploy
  env:
    LAUNCHCTL_TOKEN: ${{ secrets.LAUNCHCTL_TOKEN }}
    LAUNCHCTL_TEAM_ID: ${{ secrets.LAUNCHCTL_TEAM_ID }}
  run: |
    launchctl deploy trigger --server $SERVER_ID --site $SITE_ID --ci --wait
```

**Flags:**

| Flag | Description |
|------|-------------|
| `--ci` | Non-interactive mode, env var auth |
| `--wait` | Wait for operation to complete (deploy, provision) |
| `--timeout` | Max wait time (default: 10m) |

**Implementation Notes:**
- `--ci` flag sets a global context that all commands check
- `huh` form calls are wrapped: if `--ci` is set, use flag values or error
- `--wait` on deploy: poll deployment status until finished/failed, then exit with appropriate code

**Files to Modify:**

| File | Change |
|------|--------|
| `cmd/root.go` | Add `--ci`, `--wait`, `--timeout` flags; env var loading |
| `internal/config/config.go` | Support env var overrides |

---

## Tier 2: Server Management (High Value)

### 2.1 `launchctl ssh test` / `launchctl ssh configure` — SSH Key Setup

Test and configure SSH key authentication for server access.

**Commands:**

```
launchctl ssh test --server <id>                        Test SSH connectivity
launchctl ssh configure --server <id>                   Configure SSH key
launchctl ssh configure --server <id> --key ~/.ssh/id_ed25519.pub --name "My Laptop"
```

**Implementation Notes:**
- `ssh test` attempts an SSH connection to the server and reports success/failure
- `ssh configure` uploads the user's public key to the server via API
- Auto-detect default SSH key (`~/.ssh/id_ed25519.pub` or `~/.ssh/id_rsa.pub`)

**Files to Modify:**

| File | Change |
|------|--------|
| `cmd/servers/ssh.go` | Add `test` and `configure` subcommands or make ssh a top-level command group |

---

### 2.2 `launchctl services` — Service Status, Logs & Restart

Unified interface for managing all installed services on a server. Replaces individual daemon/database/webserver management with one consistent pattern.

**Commands:**

```
launchctl services list --server <id>                         List all services
launchctl services status <service> --server <id>             Check service status
launchctl services logs <service> --server <id>               View service logs
launchctl services logs <service> --server <id> --follow      Stream logs in realtime
launchctl services restart <service> --server <id>            Restart a service
launchctl services stop <service> --server <id>               Stop a service
launchctl services start <service> --server <id>              Start a service
```

Where `<service>` is a service name like `caddy`, `mysql`, `postgresql`, `redis`, `supervisor`, or a daemon name.

**API Endpoints:**

| Method | Endpoint | Purpose |
|--------|----------|---------|
| `GET` | `/servers/:id/services` | List installed services |
| `POST` | `/servers/:id/services/:id` | Service operation (start/stop/restart/status) |

**Service Types:** Web server, database, cache, process manager, runtime, custom daemons

**Table Output:**

```
Services (server: production-web)

  Name          Type        Version   Status    Default
  Caddy         webserver   2.7       running   -
  MySQL 8.0     database    8.0       running   -
  Redis         cache       7.2       running   -
  Supervisor    process     4.2       running   -
  Node.js       runtime     20        installed -
```

**Files to Create:**

| File | Purpose |
|------|---------|
| `cmd/services/services.go` | Parent command |
| `cmd/services/list.go` | List installed services |
| `cmd/services/status.go` | Service status |
| `cmd/services/logs.go` | Service logs with --follow |
| `cmd/services/restart.go` | Restart service |
| `cmd/services/stop.go` | Stop service |
| `cmd/services/start.go` | Start service |
| `internal/api/services.go` | API methods |

---

### 2.3 `launchctl databases` — Database Management

Manage databases, database users, and access database shells.

**Commands:**

```
launchctl databases list --server <id>                        List databases
launchctl databases shell --server <id>                       Open database shell (interactive picker)
launchctl databases shell --server <id> --database <name>     Open specific database shell
launchctl databases shell --server <id> --user <user>         Specify database user
launchctl databases proxy --server <id> --database <id>       SSH tunnel for local client access
```

**API Endpoints:**

| Method | Endpoint | Purpose |
|--------|----------|---------|
| `GET` | `/servers/:id/databases` | List databases |
| `GET` | `/servers/:id/database-users` | List database users |
| `GET` | `/servers/:id/ssh-credentials` | Get SSH credentials for tunnel |
| `GET` | `/servers/:id/database-users/:id/credentials` | Get DB user credentials |

**Database Shell:**
- Detect installed database type (MySQL, PostgreSQL)
- SSH into server and launch the appropriate client (`mysql`, `psql`)
- Support specifying database name and user

**Database Proxy (SSH Tunnel):**
- Establish SSH tunnel: `localhost:<auto-port>` → `server:3306/5432`
- Display connection string for local clients
- Keep tunnel alive until user exits

**Table Output:**

```
Databases (server: production-web)

  Name              Type         Status
  app_production    mysql        active
  app_staging       mysql        active
```

**Files to Create:**

| File | Purpose |
|------|---------|
| `cmd/databases/databases.go` | Parent command |
| `cmd/databases/list.go` | List databases |
| `cmd/databases/shell.go` | Open database shell |
| `cmd/databases/proxy.go` | SSH tunnel proxy |
| `internal/api/databases.go` | API methods |

---

### 2.4 `launchctl firewall` — Firewall Rules

**Commands:**

```
launchctl firewall list --server <id>
launchctl firewall add --server <id> --port 6379 --from 10.0.0.1 --action allow
launchctl firewall add --server <id>                    Interactive mode
launchctl firewall remove <rule-id> --server <id>
```

**API Endpoints:**

| Method | Endpoint | Purpose |
|--------|----------|---------|
| `GET` | `/servers/:id/firewall-rules` | List rules |
| `POST` | `/servers/:id/firewall-rules` | Create rule |
| `PUT` | `/servers/:id/firewall-rules/:ruleId` | Update rule |
| `DELETE` | `/servers/:id/firewall-rules/:ruleId` | Delete rule |

**Model Fields:**
- `Name`, `Action` (allow/deny/reject), `Port`, `FromIPv4` (CIDR), `Mask`, `Note`
- Status: `IsInstalled`, `IsPending`, `HasFailed`

**Table Output:**

```
Firewall Rules (server: production-web)

  Name          Action   Port    From          Status
  SSH           allow    22      0.0.0.0/0     installed
  HTTP          allow    80      0.0.0.0/0     installed
  HTTPS         allow    443     0.0.0.0/0     installed
  Redis         allow    6379    10.0.0.1      installed
  MySQL (LB)    allow    3306    10.0.0.5      pending
```

**Files to Create:**

| File | Purpose |
|------|---------|
| `cmd/firewall/firewall.go` | Parent command |
| `cmd/firewall/list.go` | List rules |
| `cmd/firewall/add.go` | Add rule (interactive + flag modes) |
| `cmd/firewall/remove.go` | Remove rule |
| `internal/api/firewall.go` | API methods |

---

### 2.5 `launchctl cron` — Cron Job Management

**Commands:**

```
launchctl cron list --server <id>
launchctl cron add --server <id>                         Interactive
launchctl cron add --server <id> --expression "0 * * * *" --command "backup.sh"
launchctl cron remove <cron-id> --server <id>
```

**API Endpoints:**

| Method | Endpoint | Purpose |
|--------|----------|---------|
| `GET` | `/servers/:id/crons` | List crons |
| `POST` | `/servers/:id/crons` | Create cron |
| `PUT` | `/servers/:id/crons/:cronId` | Update cron |
| `DELETE` | `/servers/:id/crons/:cronId` | Delete cron |

**Model Fields:**
- `User` (default: root), `Expression`, `Command` (encrypted), `Frequency`, `SiteID` (optional)
- Status: `IsInstalled`, `InstalledAt`

**Table Output:**

```
Cron Jobs (server: production-web)

  User    Expression      Command                  Status
  root    * * * * *       run-scheduler            installed
  root    0 2 * * *       /usr/local/bin/backup    installed
  app     0 */6 * * *     clear-cache              pending
```

**Files to Create:**

| File | Purpose |
|------|---------|
| `cmd/cron/cron.go` | Parent command |
| `cmd/cron/list.go` | List crons |
| `cmd/cron/add.go` | Add cron |
| `cmd/cron/remove.go` | Remove cron |
| `internal/api/crons.go` | API methods |

---

### 2.6 `launchctl ssl` — SSL Certificate Management

**Commands:**

```
launchctl ssl list --server <id> --site <id>
launchctl ssl install --server <id> --site <id>          Auto (Let's Encrypt)
launchctl ssl install --server <id> --site <id> --custom Upload custom cert
launchctl ssl off --server <id> --site <id>              Disable SSL
```

**API Endpoints:**

| Method | Endpoint | Purpose |
|--------|----------|---------|
| `GET` | `/servers/:serverId/sites/:id/certificates` | List certificates |
| `PUT` | `/servers/:serverId/sites/:id/ssl` | Update SSL settings |

**TLS Settings:** `auto` (Let's Encrypt), `custom` (upload), `internal` (self-signed), `off`

**Request (custom cert):**
```json
{
  "tls_setting": "custom",
  "private_key": "-----BEGIN PRIVATE KEY-----\n...",
  "certificate": "-----BEGIN CERTIFICATE-----\n..."
}
```

**Files to Create:**

| File | Purpose |
|------|---------|
| `cmd/ssl/ssl.go` | Parent command |
| `cmd/ssl/list.go` | List certs |
| `cmd/ssl/install.go` | Install SSL |
| `cmd/ssl/off.go` | Disable SSL |
| `internal/api/ssl.go` | API methods |

---

### 2.7 `launchctl ssh-keys` — SSH Key Management

**Commands:**

```
launchctl ssh-keys list                                  Team-wide keys
launchctl ssh-keys list --server <id>                    Keys on a server
launchctl ssh-keys add --name "My Key" --key ~/.ssh/id_ed25519.pub
launchctl ssh-keys add --generate --type ed25519         Generate new keypair
launchctl ssh-keys attach <key-id> --server <id>         Add key to server
launchctl ssh-keys detach <key-id> --server <id>         Remove key from server
launchctl ssh-keys remove <key-id>                       Delete key entirely
```

**API Endpoints:**

| Method | Endpoint | Purpose |
|--------|----------|---------|
| `GET` | `/ssh-keys` | List team SSH keys |
| `POST` | `/ssh-keys` | Create SSH key |
| `POST` | `/ssh-keys/generate` | Generate SSH keypair |
| `DELETE` | `/ssh-keys/:id` | Delete SSH key |
| `GET` | `/servers/:id/ssh-keys` | List keys on server |
| `POST` | `/servers/:id/ssh-keys` | Attach key to server |
| `DELETE` | `/servers/:id/ssh-keys/:id` | Detach key from server |

**Model Fields:**
- `Name`, `PublicKey`, `Fingerprint`, `Description`, `IsGlobal`

**Files to Create:**

| File | Purpose |
|------|---------|
| `cmd/sshkeys/sshkeys.go` | Parent command |
| `cmd/sshkeys/list.go` | List keys |
| `cmd/sshkeys/add.go` | Add/generate key |
| `cmd/sshkeys/attach.go` | Attach to server |
| `cmd/sshkeys/detach.go` | Detach from server |
| `cmd/sshkeys/remove.go` | Delete key |
| `internal/api/sshkeys.go` | API methods |

---

### 2.8 `launchctl daemons` — Background Process Management

**Commands:**

```
launchctl daemons list --server <id>
launchctl daemons add --server <id>                      Interactive
launchctl daemons add --server <id> --command "worker.js" --processes 3
launchctl daemons restart <id> --server <id>
launchctl daemons remove <id> --server <id>
launchctl daemons logs <id> --server <id>                View daemon output
launchctl daemons status <id> --server <id>              Check if running
```

**API Endpoints:**

| Method | Endpoint | Purpose |
|--------|----------|---------|
| `GET` | `/servers/:id/daemons` | List daemons |
| `POST` | `/servers/:id/daemons` | Create daemon |
| `PUT` | `/servers/:id/daemons/:id` | Update daemon |
| `POST` | `/servers/:id/daemons/:id/restart` | Restart daemon |
| `DELETE` | `/servers/:id/daemons/:id` | Delete daemon |

**Model Fields:**
- `User`, `Directory`, `Command`, `Processes` (default: 1), `StopWaitSeconds`, `StopSignal`
- Status: `IsInstalled`, `Running`

**Table Output:**

```
Daemons (server: production-web)

  User   Command                  Procs   Running   Status
  app    node worker.js           3       yes       installed
  app    python celery worker     1       yes       installed
  root   /usr/bin/redis-server    1       yes       installed
```

**Files to Create:**

| File | Purpose |
|------|---------|
| `cmd/daemons/daemons.go` | Parent command |
| `cmd/daemons/list.go` | List daemons |
| `cmd/daemons/add.go` | Add daemon |
| `cmd/daemons/restart.go` | Restart daemon |
| `cmd/daemons/remove.go` | Remove daemon |
| `cmd/daemons/logs.go` | View daemon logs |
| `cmd/daemons/status.go` | Check daemon status |
| `internal/api/daemons.go` | API methods |

---

### 2.9 `launchctl software` — Server Software Management

**Commands:**

```
launchctl software list --server <id>                    Installed software
launchctl software available --server <id>               Available to install
launchctl software install --server <id> node20
launchctl software remove --server <id> node18
```

**API Endpoints:**

| Method | Endpoint | Purpose |
|--------|----------|---------|
| `GET` | `/servers/:id/services` | List installed services |
| `GET` | `/servers/:id/services/create` | Available software |
| `POST` | `/servers/:id/services` | Install software |
| `POST` | `/servers/:id/services/:id` | Service operation (remove) |

**Table Output:**

```
Installed Software (server: production-web)

  Name         Type        Version   Status    Default
  Caddy        webserver   2.7       running   -
  MySQL 8.0    database    8.0       running   -
  Redis        cache       7.2       running   -
  Node.js      runtime     20        installed -
```

**Files to Create:**

| File | Purpose |
|------|---------|
| `cmd/software/software.go` | Parent command |
| `cmd/software/list.go` | List installed |
| `cmd/software/available.go` | Available to install |
| `cmd/software/install.go` | Install software |
| `cmd/software/remove.go` | Remove software |
| `internal/api/software.go` | API methods (reuses services endpoints) |

---

## Tier 3: Operations (Medium Value)

### 3.1 `launchctl backups` — Backup Management

**Commands:**

```
launchctl backups list --server <id>
launchctl backups show <id> --server <id>
launchctl backups create --server <id>                   Interactive
launchctl backups run <id> --server <id>                 Trigger manual backup
launchctl backups jobs <id> --server <id>                List backup job history
launchctl backups remove <id> --server <id>

launchctl backups storage list                           List storage providers
launchctl backups storage add                            Connect provider (S3/Dropbox)
launchctl backups storage remove <id>
```

**API Endpoints:**

| Method | Endpoint | Purpose |
|--------|----------|---------|
| `GET` | `/servers/:serverId/backups` | List backups |
| `POST` | `/servers/:serverId/backups` | Create backup config |
| `GET` | `/servers/:serverId/backups/:id` | Get backup |
| `PUT` | `/servers/:serverId/backups/:id` | Update backup |
| `DELETE` | `/servers/:serverId/backups/:id` | Delete backup |
| `POST` | `/servers/:serverId/backups/:id/run` | Run manual backup |
| `GET` | `/storage-providers` | List storage providers |
| `POST` | `/storage-providers/:provider/connect` | Connect provider |
| `DELETE` | `/storage-providers/:provider` | Delete provider |

**Model Fields:**
- Backup: `CronExpression`, `Path`, `IncludeFiles`, `ExcludeFiles`, `Retention`, `Enabled`
- BackupJob: `Status` (pending/running/finished/failed), `Size`, `Error`
- StorageProvider: `Provider` (s3/dropbox), `Label`, `Credentials` (encrypted), `Connected`

**Table Output:**

```
Backups (server: production-web)

  Name              Schedule      Storage    Retention   Enabled   Last Run
  Daily DB Backup   0 2 * * *     AWS S3     14 days     yes       2h ago (success)
  Weekly Files      0 0 * * 0     Dropbox    4 weeks     yes       3d ago (success)
```

**Files to Create:**

| File | Purpose |
|------|---------|
| `cmd/backups/backups.go` | Parent command |
| `cmd/backups/list.go` | List backups |
| `cmd/backups/show.go` | Show backup details |
| `cmd/backups/create.go` | Create backup (interactive) |
| `cmd/backups/run.go` | Trigger manual backup |
| `cmd/backups/jobs.go` | List backup job history |
| `cmd/backups/remove.go` | Remove backup |
| `cmd/backups/storage.go` | Storage provider subcommands |
| `internal/api/backups.go` | API methods |

---

### 3.2 `launchctl scripts` — Custom Script Execution

**Commands:**

```
launchctl scripts list
launchctl scripts show <id>
launchctl scripts create --name "Clear cache" --content "rm -rf /tmp/cache/*"
launchctl scripts run <id> --server <id> [--server <id2>]    Run on one or more servers
launchctl scripts executions <id>                             List execution history
launchctl scripts remove <id>
```

**API Endpoints:**

| Method | Endpoint | Purpose |
|--------|----------|---------|
| `GET` | `/scripts` | List scripts |
| `POST` | `/scripts` | Create script |
| `GET` | `/scripts/:id` | Get script |
| `PUT` | `/scripts/:id` | Update script |
| `DELETE` | `/scripts/:id` | Delete script |
| `POST` | `/scripts/:id/execute` | Execute on servers |
| `GET` | `/scripts/:id/executions` | Execution history |

**Model Fields:**
- Script: `Name`, `RunAs` (root/local), `Content` (bash script)
- Execution: `Status` (pending/running/finished/failed), `ExitCode`, `Output`, `BatchID`

**Execute Request:**
```json
{
  "server_ids": ["abc123", "def456"],
  "run_as": "root"
}
```

**Live Output:** Subscribe to `script.execution.output` WebSocket events for real-time streaming.

**Files to Create:**

| File | Purpose |
|------|---------|
| `cmd/scripts/scripts.go` | Parent command |
| `cmd/scripts/list.go` | List scripts |
| `cmd/scripts/show.go` | Show script content |
| `cmd/scripts/create.go` | Create script |
| `cmd/scripts/run.go` | Execute on servers (with live output) |
| `cmd/scripts/executions.go` | Execution history |
| `cmd/scripts/remove.go` | Delete script |
| `internal/api/scripts.go` | API methods |

---

### 3.3 `launchctl activity` — Activity/Audit Log

**Commands:**

```
launchctl activity                                       Recent team activity
launchctl activity --server <id>                         Activity on a server
launchctl activity --site <id> --server <id>             Activity on a site
launchctl activity --user <id>                           Activity by a user
```

**API Endpoint (new, requires backend):**

```
GET /activity?subject_type=server&subject_id=abc123&page=1&per_page=20
```

**Table Output:**

```
Recent Activity

  Time          User          Action                    Resource
  2m ago        john@acme     deployed site             api.example.com
  15m ago       jane@acme     added firewall rule       production-web
  1h ago        john@acme     restarted daemon          queue-worker
  3h ago        system        renewed SSL certificate   api.example.com
  1d ago        jane@acme     created server            staging-web
```

**Files to Create:**

| File | Purpose |
|------|---------|
| `cmd/activity.go` | Activity command |
| `internal/api/activity.go` | API methods |

---

### 3.4 `launchctl notifications` — Notification Channel Management

**Commands:**

```
launchctl notifications channels                         List notification channels
launchctl notifications channels add                     Add channel (interactive)
launchctl notifications channels test <id>               Test channel
launchctl notifications channels remove <id>
```

**API Endpoints:**

| Method | Endpoint | Purpose |
|--------|----------|---------|
| `GET` | `/settings/notifications` | List channels |
| `POST` | `/settings/notifications` | Create channel |
| `DELETE` | `/settings/notifications/:id` | Delete channel |
| `POST` | `/settings/notifications/:id/test` | Test channel |

**Channel Types:** email, slack, discord, telegram

**Table Output:**

```
Notification Channels

  Provider   Label              Default   Connected   Events
  slack      #deployments       yes       yes         deploy, backup
  email      team@acme.com      no        yes         deploy, backup
  discord    Server Alerts      no        yes         deploy
```

**Files to Create:**

| File | Purpose |
|------|---------|
| `cmd/notifications/notifications.go` | Parent command |
| `cmd/notifications/channels.go` | Channel management |
| `internal/api/notifications.go` | API methods |

---

### 3.5 `launchctl redirects` — URL Redirect Management

```
launchctl redirects list --server <id> --site <id>
launchctl redirects add --server <id> --site <id> --from /old --to /new --type 301
launchctl redirects remove <id> --server <id> --site <id>
```

**API Endpoints:**

| Method | Endpoint | Purpose |
|--------|----------|---------|
| `GET` | `/servers/:serverId/sites/:id/redirects` | List redirects |
| `POST` | `/servers/:serverId/sites/:id/redirects` | Create redirect |
| `DELETE` | `/servers/:serverId/sites/:id/redirects/:id` | Delete redirect |

**Redirect Types:** 301 (permanent), 302 (temporary), 307, 308

**Files to Create:**

| File | Purpose |
|------|---------|
| `cmd/redirects/redirects.go` | Parent command |
| `cmd/redirects/list.go` | List redirects |
| `cmd/redirects/add.go` | Add redirect |
| `cmd/redirects/remove.go` | Remove redirect |
| `internal/api/redirects.go` | API methods |

---

## Tier 4: Multi-Server / Advanced (Medium Value)

### 4.1 `launchctl lb` — Load Balancer Management

**Commands:**

```
launchctl lb list --server <id>
launchctl lb show <upstream-id> --server <id>
launchctl lb create --server <id>                        Interactive
launchctl lb remove <upstream-id> --server <id>

launchctl lb backends <upstream-id> --server <id>        List backends
launchctl lb add-backend <upstream-id> --server <id> --site <site-id>
launchctl lb remove-backend <backend-id> --server <id> --upstream <id>
launchctl lb toggle-down <backend-id> --server <id> --upstream <id>

launchctl lb health <upstream-id> --server <id>          Health check
```

**API Endpoints:**

| Method | Endpoint | Purpose |
|--------|----------|---------|
| `GET` | `/servers/:id/upstreams` | List upstreams |
| `POST` | `/servers/:id/upstreams` | Create upstream |
| `GET` | `/servers/:id/upstreams/:id` | Get upstream |
| `PUT` | `/servers/:id/upstreams/:id` | Update upstream |
| `DELETE` | `/servers/:id/upstreams/:id` | Delete upstream |
| `GET` | `/servers/:id/upstreams/:id/backends` | List backends |
| `POST` | `/servers/:id/upstreams/:id/backends` | Add backend |
| `DELETE` | `/servers/:id/upstreams/:id/backends/:id` | Remove backend |
| `POST` | `/servers/:id/upstreams/:id/backends/:id/toggle-down` | Toggle down |
| `GET` | `/servers/:id/upstreams/:id/health` | Health status |

**Health Output:**

```
Upstream Health: api.example.com

  Policy: round_robin
  Backends: 3 total, 2 healthy, 1 down

  Server            Site              Port   Health      Down
  web-1             api.example.com   8080   healthy     no
  web-2             api.example.com   8080   healthy     no
  web-3             api.example.com   8080   unhealthy   yes (manual)
```

**Files to Create:**

| File | Purpose |
|------|---------|
| `cmd/lb/lb.go` | Parent command |
| `cmd/lb/list.go` | List upstreams |
| `cmd/lb/show.go` | Show upstream details |
| `cmd/lb/create.go` | Create upstream |
| `cmd/lb/remove.go` | Remove upstream |
| `cmd/lb/backends.go` | Backend subcommands |
| `cmd/lb/health.go` | Health check display |
| `internal/api/loadbalancer.go` | API methods |

---

### 4.2 Profiles / Contexts

Support multiple API environments without re-logging in.

**Commands:**

```
launchctl config profiles                                List profiles
launchctl config profiles add staging                    Add profile (interactive login)
launchctl config profiles use staging                    Switch active profile
launchctl config profiles remove staging
```

**Config Structure:**

```json
{
  "active_profile": "production",
  "profiles": {
    "production": {
      "api_url": "https://api.launchctl.dev",
      "access_token": "...",
      "team_id": "...",
      "team_name": "Acme Corp"
    },
    "staging": {
      "api_url": "https://staging-api.launchctl.dev",
      "access_token": "...",
      "team_id": "...",
      "team_name": "Acme Staging"
    }
  }
}
```

**Files to Modify:**

| File | Change |
|------|--------|
| `internal/config/config.go` | Multi-profile support |
| `cmd/config.go` | Profile subcommands |

---

## Tier 5: Developer Experience (Nice to Have)

### 5.1 `launchctl whoami`

```bash
$ launchctl whoami

  User:   John Doe (john@acme.com)
  Team:   Acme Corp
  API:    https://api.launchctl.dev
```

**File:** `cmd/whoami.go` — calls `GET /auth/user`

---

### 5.2 `launchctl open`

Open the resource in the web dashboard browser.

```bash
launchctl open                                 Open dashboard
launchctl open --server <id>                   Open server page
launchctl open --server <id> --site <id>       Open site page
```

**File:** `cmd/open.go` — constructs URL and calls `open` (macOS) / `xdg-open` (Linux)

---

### 5.3 `launchctl diff`

Show what changed since last deploy.

```bash
launchctl diff --server <id> --site <id>
```

**How:**
1. Get latest deployment via API → extract `git_hash`
2. Run `git log --oneline <deployed_hash>..HEAD` locally
3. Show the commit list

**File:** `cmd/diff.go`

---

### 5.4 `launchctl health`

Quick health overview across all servers.

```bash
$ launchctl health

  Server            Status      Load    Memory   Disk    SSL Expiry
  production-web    connected   0.45    75%      52%     89 days
  staging-web       connected   0.12    30%      25%     89 days
  db-server         connected   2.10    85%      61%     -
  backup-server     offline     -       -        -       -
```

**How:** Fetch all servers, then parallel-fetch metrics for each. Highlight warnings (high load, high disk, SSL expiring soon, offline).

**File:** `cmd/health.go`

---

## Implementation Priority

| Phase | Features | Effort |
|-------|----------|--------|
| **Phase A** | `init`, `whoami`, CI/CD mode, profiles | Small — config changes only |
| **Phase B** | `env`, `run`, `logs`, `deploy logs` | Medium — 4 new command groups |
| **Phase C** | `services`, `databases`, `ssh test/configure` | Medium — unified service management |
| **Phase D** | `firewall`, `cron`, `ssl`, `ssh-keys`, `daemons` | Medium — 5 CRUD command groups |
| **Phase E** | `software`, `backups`, `scripts` | Medium — 3 command groups with complex flows |
| **Phase F** | `lb`, `notifications`, `activity`, `redirects` | Medium — 4 command groups |
| **Phase G** | `health`, `open`, `diff` | Small — utility commands |

---

## Shared Patterns

### Interactive + Flag Modes

Every `create`/`add` command supports both modes:

```go
// If required flags provided, use them directly
if serverID != "" && port != "" {
    return createDirect(serverID, port, action)
}

// Otherwise, show interactive huh form
return createInteractive(client)
```

### Project Config Integration

All commands that accept `--server` and `--site` check `.launchctl.yml`:

```go
func resolveServerID(cmd *cobra.Command) (string, error) {
    // 1. Check explicit flag
    if id, _ := cmd.Flags().GetString("server"); id != "" {
        return id, nil
    }
    // 2. Check .launchctl.yml
    if proj, err := config.LoadProject(); err == nil {
        return proj.ServerID, nil
    }
    // 3. Error
    return "", fmt.Errorf("no server specified, use --server or run 'launchctl init'")
}
```

### JSON Output

Every list/show command supports `--json`:

```go
if jsonOutput {
    data, _ := json.MarshalIndent(result, "", "  ")
    fmt.Println(string(data))
    return nil
}
// Otherwise render table
```
