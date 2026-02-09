# Database Proxy Feature

## Problem

Connecting to a remote database on a launchctl-managed server currently requires:

1. Obtaining the server's SSH credentials (IP, port, username, private key)
2. Configuring an SSH tunnel manually or setting up SSH keys in database clients like TablePlus
3. Finding the database credentials (username, password, database name, port)

This is tedious, error-prone, and exposes SSH keys to third-party applications.

## Solution

Add a `launchctl db proxy` command that creates a local SSH tunnel to the remote database. Users connect their database client to `localhost` — the CLI handles everything else.

```bash
launchctl db proxy --server <id> --database <id>
```

Output:

```
 Tunnel active

  Host:      127.0.0.1
  Port:      33060
  Database:  my_app_db
  Username:  my_db_user
  Password:  s3cretP@ss

  Press Ctrl+C to disconnect
```

The user pastes these details into TablePlus (or any client) and connects. No SSH keys needed.

---

## Architecture

### How It Works

```
TablePlus                launchctl CLI               Remote Server
─────────               ──────────────               ─────────────
                    ┌─── SSH Tunnel ────────────────┐
localhost:33060 ──► │  ssh -L 33060:localhost:3306  │ ──► MySQL :3306
                    └──────────────────────────────┘
```

1. CLI fetches server details (IP, SSH port, username) from the API
2. CLI fetches database + user credentials from a new API endpoint
3. CLI opens an SSH tunnel forwarding a local port to the database port on the server
4. CLI prints connection details and keeps the tunnel open until Ctrl+C

### Local Port Selection

- Default: use the database's remote port (3306 for MySQL, 5432 for PostgreSQL)
- If the port is in use, auto-increment (3307, 3308, ...) until a free port is found
- Override with `--local-port` flag

---

## API Changes (launch-go)

### New Endpoint: Get Database Credentials

The current `GET /servers/:serverId/database-users/:id` endpoint does **not** return the password (it's marked `json:"-"` for security). A new dedicated endpoint is needed.

```
GET /api/servers/:serverId/database-users/:id/credentials
```

**Response:**

```json
{
  "success": true,
  "data": {
    "username": "my_db_user",
    "password": "s3cretP@ss",
    "host": "localhost",
    "database_names": ["my_app_db", "other_db"]
  }
}
```

**Security:**
- Requires authentication (Bearer token)
- Requires team membership
- Requires server to be provisioned
- Audit log entry when credentials are accessed
- Rate limited (prevent credential scraping)

**Implementation location:**
- Route: `internal/modules/database/routes.go`
- Handler: `internal/modules/database/handlers/database_user_handler.go`
- DTO: `internal/modules/database/dto/responses.go` (new `DatabaseCredentialsResponse`)

### Server SSH Key Access

The CLI also needs the server's SSH private key to establish the tunnel. Currently `ServerResponse` does not include it.

**Option A — New endpoint (recommended):**

```
GET /api/servers/:serverId/ssh-credentials
```

Returns the SSH private key, username, port, and IP. This keeps sensitive data out of the standard server response.

```json
{
  "success": true,
  "data": {
    "host": "203.0.113.50",
    "port": 22,
    "username": "launchctl",
    "private_key": "-----BEGIN OPENSSH PRIVATE KEY-----\n..."
  }
}
```

**Option B — Use system SSH agent:**

If the user has the server's SSH key in their local `~/.ssh/`, the CLI can rely on the SSH agent instead of fetching the key from the API. This avoids transmitting private keys but requires local key setup.

**Recommendation:** Implement Option A with Option B as a fallback. Try the API key first; if unavailable, fall back to the local SSH agent.

---

## CLI Commands

### Command Structure

```
launchctl db
  ├── list       --server <id>                  List databases on a server
  ├── users      --server <id>                  List database users
  └── proxy      --server <id> --database <id>  Open database tunnel
```

### `launchctl db list`

```bash
launchctl db list --server abc123
```

Calls `GET /api/servers/:serverId/databases` and renders a table:

```
Databases (server: production-web)

  Name          Status     Users     Created
  my_app_db     installed  app_user  2026-01-15
  analytics     installed  app_user  2026-01-20
```

### `launchctl db users`

```bash
launchctl db users --server abc123
```

Calls `GET /api/servers/:serverId/database-users` and renders a table:

```
Database Users (server: production-web)

  Name       Host       Status     Databases
  app_user   localhost  installed  my_app_db, analytics
  readonly   localhost  installed  analytics
```

### `launchctl db proxy`

```bash
launchctl db proxy --server abc123 --database def456

# With options
launchctl db proxy --server abc123 --database def456 --user <db-user-id> --local-port 33060
```

**Flags:**

| Flag | Description | Default |
|------|-------------|---------|
| `--server` | Server ID (required) | — |
| `--database` | Database ID (required) | — |
| `--user` | Database user ID | First user linked to the database |
| `--local-port` | Local port to bind | Same as remote port (auto-increment if busy) |

**Flow:**

1. Fetch server details → get IP, SSH port, username
2. Fetch SSH credentials → get private key
3. Fetch database details → get database name, linked users
4. Fetch database user credentials → get username, password
5. Detect database type from server software → determine remote port (3306/5432)
6. Find available local port
7. Establish SSH tunnel
8. Display connection info
9. Keep alive until Ctrl+C, with reconnect on connection drop

### Interactive Mode

If `--server` or `--database` is not provided, use `huh` forms to let the user pick:

1. Select a server (from `GET /api/servers`)
2. Select a database (from `GET /api/servers/:id/databases`)
3. Select a database user (if multiple users are linked)

---

## Implementation Plan

### Phase 1: API Endpoints (launch-go)

1. **Add `GET /servers/:serverId/ssh-credentials`**
   - File: `internal/modules/server/routes.go`
   - Handler: `internal/modules/server/handlers/server_handler.go`
   - DTO: New `SSHCredentialsResponse` in `internal/modules/server/dto/responses.go`
   - Returns: host, port, username, private_key

2. **Add `GET /servers/:serverId/database-users/:id/credentials`**
   - File: `internal/modules/database/routes.go`
   - Handler: `internal/modules/database/handlers/database_user_handler.go`
   - DTO: New `DatabaseCredentialsResponse` in `internal/modules/database/dto/responses.go`
   - Returns: username, password, host, database_names
   - Add audit log entry

3. **Add database software type to server response**
   - Ensure `ServerResponse` includes the installed database software (mysql8.0, postgresql16, etc.)
   - This lets the CLI determine the correct port

### Phase 2: CLI API Client (launchctl)

4. **Add API methods** (`internal/api/databases.go`)
   - `ListDatabases(serverID string) ([]DatabaseResponse, error)`
   - `ListDatabaseUsers(serverID string) ([]DatabaseUserResponse, error)`
   - `GetDatabaseCredentials(serverID, userID string) (*DatabaseCredentials, error)`
   - `GetSSHCredentials(serverID string) (*SSHCredentials, error)`

5. **Add types** (`internal/api/types.go`)
   - `DatabaseResponse`, `DatabaseUserResponse`
   - `DatabaseCredentials`, `SSHCredentials`

### Phase 3: CLI Commands (launchctl)

6. **Add `cmd/db/` command group**
   - `db.go` — parent command
   - `list.go` — list databases
   - `users.go` — list database users
   - `proxy.go` — SSH tunnel proxy

7. **SSH tunnel implementation** (`internal/tunnel/tunnel.go`)
   - Accept SSH credentials + local/remote port
   - Open SSH connection using `golang.org/x/crypto/ssh`
   - Listen on local port, forward connections through SSH channel
   - Handle reconnection on disconnect
   - Graceful shutdown on SIGINT/SIGTERM

8. **Register commands** in `cmd/root.go`

### Phase 4: Polish

9. **Connection string output** — Show copy-paste connection strings:
   ```
   mysql -h 127.0.0.1 -P 33060 -u app_user -p my_app_db
   psql -h 127.0.0.1 -p 54320 -U app_user -d my_app_db
   ```

10. **Connection test** — After tunnel is up, verify the database is reachable before printing details

11. **Multiple tunnels** — Support proxying multiple databases simultaneously (future)

---

## Dependencies

### New Go dependencies (launchctl)

```
golang.org/x/crypto/ssh  — SSH client for tunnel
```

### Existing dependencies used

```
github.com/spf13/cobra     — CLI framework
github.com/charmbracelet/huh — Interactive forms
github.com/charmbracelet/lipgloss — Styled output
```

---

## Security Considerations

- SSH private keys are fetched over HTTPS and held in memory only — never written to disk
- Database passwords are displayed once in the terminal and not persisted
- The tunnel binds to `127.0.0.1` only (not `0.0.0.0`) to prevent network exposure
- API endpoints for credentials require full authentication and team authorization
- Credential access is audit-logged for compliance

---

## Database Port Reference

| Software | Default Port |
|----------|-------------|
| MySQL 8.0 | 3306 |
| PostgreSQL 16 | 5432 |
| MariaDB 10 | 3306 |

The port is determined by the server's installed database software type.
