# Real-Time WebSocket Streaming

## Problem

The launchctl CLI has a basic WebSocket client that only handles deployment log events. The launch-go backend broadcasts 100+ event types across servers, sites, deployments, tasks, databases, and more — but the CLI can't consume most of them.

Current gaps:
- **No server provisioning progress** — users can't watch provisioning live
- **No task output streaming** — SSH command output isn't visible
- **Dashboard uses polling** (30s interval) instead of real-time updates
- **No reconnection or heartbeat** — WebSocket drops silently
- **No general event watcher** — can't tail events for debugging
- **Deploy TUI is basic** — plain text, no step progress, no timestamps

## Solution

Enhance the WebSocket client and add real-time streaming to all major CLI workflows.

---

## Backend WebSocket Reference

### Connection

```
ws(s)://<host>/api/ws?token=<jwt>&team_id=<team_id>
```

Auto-subscribed to `team.{teamID}` on connect. Additional channels subscribed via:

```json
{"action": "subscribe", "channel": "deployment.abc123"}
```

### Message Format

```json
{
  "event": "deployment.progress",
  "channel": "deployment.abc123",
  "data": { ... }
}
```

### Channel Types

| Pattern | Use |
|---------|-----|
| `team.{id}` | Team-wide events (auto-subscribed) |
| `server.{id}` | Server-specific events |
| `site.{id}` | Site-specific events |
| `deployment.{id}` | Deployment log streaming |
| `task.{id}` | Task output streaming |
| `user.{id}` | User notifications |

### Event Categories (100+ events)

| Category | Key Events |
|----------|------------|
| **Server** | `server.created`, `server.provisioned`, `server.provision_progress`, `server.provision_step`, `server.provision_failed`, `server.provision_status`, `server.rebooting`, `server.rebooted`, `server.connectivity`, `server.metrics` |
| **Deployment** | `deployment.created`, `deployment.started`, `deployment.progress`, `deployment.log`, `deployment.finished`, `deployment.failed`, `deployment.timeout`, `deployment.cancelled` |
| **Site** | `site.created`, `site.updated`, `site.deleted`, `site.installed`, `site.installation_failed` |
| **Task** | `task.created`, `task.running`, `task.output`, `task.updated`, `task.finished`, `task.failed`, `task.timeout` |
| **Database** | `database.created`, `database.deleted`, `database.progress`, `database_user.created`, `database_user.deleted` |
| **Backup** | `backup.started`, `backup.finished`, `backup.failed` |
| **Certificate** | `certificate.created`, `certificate.renewed`, `certificate.expiring` |
| **Load Balancer** | `upstream.created`, `upstream.installed`, `backend.added`, `backend.health_changed` |
| **Script** | `script.execution.started`, `script.execution.output`, `script.execution.completed`, `script.execution.failed` |

---

## Architecture

### Enhanced WebSocket Client

Replace the current simple client with a robust one:

```
┌─────────────────────────────────────────────┐
│              WebSocket Manager               │
│                                              │
│  ┌──────────┐  ┌──────────┐  ┌───────────┐  │
│  │ Connect  │  │ Heartbeat│  │ Reconnect │  │
│  │ + Auth   │  │ (ping/   │  │ (exp.     │  │
│  │          │  │  pong)   │  │  backoff) │  │
│  └──────────┘  └──────────┘  └───────────┘  │
│                                              │
│  ┌──────────────────────────────────────┐    │
│  │         Event Dispatcher             │    │
│  │  channel/event → handler callback    │    │
│  └──────────────────────────────────────┘    │
└─────────────────────────────────────────────┘
         │                          │
    ┌────▼─────┐              ┌─────▼────┐
    │ Deploy   │              │ Server   │
    │ TUI      │              │ Provision│
    │ Model    │              │ TUI Model│
    └──────────┘              └──────────┘
```

**File:** `internal/api/websocket.go` (rewrite)

Key improvements:
- **Auto-reconnect** with exponential backoff (1s → 2s → 4s → ... → 30s max)
- **Ping/pong heartbeat** every 30s to detect dead connections
- **Event dispatcher** — register handlers by event name or channel
- **Thread-safe** subscription management
- **Connection state** callbacks (connected, disconnected, reconnecting)

```go
type WSManager struct {
    conn       *websocket.Conn
    cfg        *config.Config
    handlers   map[string][]EventHandler  // event name → handlers
    channels   map[string]bool            // subscribed channels
    done       chan struct{}
    reconnect  bool
    mu         sync.RWMutex
}

type EventHandler func(event WSEvent)

type WSEvent struct {
    Event   string
    Channel string
    Data    json.RawMessage
}

// Subscribe to events
func (m *WSManager) On(event string, handler EventHandler)
func (m *WSManager) OnChannel(channel string, handler EventHandler)

// Channel management
func (m *WSManager) Subscribe(channel string) error
func (m *WSManager) Unsubscribe(channel string) error

// Lifecycle
func (m *WSManager) Connect() error
func (m *WSManager) Close()
func (m *WSManager) IsConnected() bool
```

---

## Feature 1: Live Deployment Logs (Enhance Existing)

The current deploy TUI works but needs improvement.

### Changes

**Current behavior:**
- Plain text log lines
- No step progress indicator
- No timestamps
- Subscribes to `team:{teamID}` (receives all team events)

**New behavior:**
- Subscribe to `deployment.{id}` channel for targeted events
- Show step progress bar (e.g., `[3/8] Installing dependencies...`)
- Timestamps on each log line
- Color-coded output (errors in red, steps in bold)
- Show elapsed time
- Copy-friendly output when piped (strip ANSI codes)

### Events Used

| Event | Action |
|-------|--------|
| `deployment.started` | Show "Deployment started" header |
| `deployment.progress` | Update step name + progress bar |
| `deployment.log` | Append log line to viewport |
| `deployment.finished` | Show success banner, elapsed time |
| `deployment.failed` | Show error banner with exit code |
| `deployment.timeout` | Show timeout warning |
| `deployment.cancelled` | Show cancellation notice |

### Files Modified

- `internal/tui/deploy/model.go` — enhanced TUI with progress bar, timestamps, colors
- `cmd/deploy/trigger.go` — subscribe to `deployment.{id}` instead of `team.{id}`

---

## Feature 2: Server Provisioning Progress

New TUI for watching server provisioning in real-time.

### Command

```bash
# After creating a server
launchctl servers create
# → automatically enters provisioning watch mode

# Or watch an existing provisioning
launchctl servers watch <id>
```

### TUI Layout

```
 Provisioning: production-web (DigitalOcean, nyc3)

  [████████████░░░░░░░░░░░░░] 48%

  ✓ Creating server on provider
  ✓ Waiting for SSH connection
  ✓ Running base provisioning
  ● Installing PHP 8.3...
  ○ Installing Caddy
  ○ Installing MySQL 8.0
  ○ Configuring firewall
  ○ Final cleanup

  Elapsed: 3m 42s

  Press q to exit (provisioning continues in background)
```

### Events Used

| Event | Action |
|-------|--------|
| `server.created_on_provider` | Mark "Creating server" as done |
| `server.waiting_for_connection` | Show "Waiting for SSH" step |
| `server.connection_attempt` | Show attempt count |
| `server.connected` | Mark SSH step as done |
| `server.provisioning` | Show provisioning started |
| `server.provision_step` | Mark step as complete, advance to next |
| `server.provision_progress` | Update progress bar percentage |
| `server.provision_status` | Show status message below progress |
| `server.provision_error` | Show warning (non-fatal) |
| `server.software_installed` | Mark software install as done |
| `server.provisioned` | Show success, all steps complete |
| `server.provision_failed` | Show failure with error details |
| `server.provision_timeout` | Show timeout |

### Files Created

- `internal/tui/provision/model.go` — provisioning TUI model
- `cmd/servers/watch.go` — watch command

---

## Feature 3: Live Task Output

Stream SSH command output in real-time when tasks run on servers.

### Command

```bash
# Watch a running task
launchctl tasks watch <task-id>

# List recent tasks
launchctl tasks list --server <id>
```

### TUI Layout

```
 Task: Update System Packages (server: production-web)

  Status: running
  Started: 2m ago

  ─────────────────────────────────
  Reading package lists...
  Building dependency tree...
  The following packages will be upgraded:
    libssl3 openssl
  2 upgraded, 0 newly installed, 0 to remove.
  Unpacking openssl (3.0.13-0ubuntu3.5) ...
  Setting up openssl (3.0.13-0ubuntu3.5) ...
  ▌
  ─────────────────────────────────

  Press q to exit (task continues in background)
```

### Events Used

| Event | Action |
|-------|--------|
| `task.created` | Show task header |
| `task.running` | Set status to running |
| `task.output` | Append output to viewport |
| `task.updated` | Update status/output |
| `task.finished` | Show success with exit code |
| `task.failed` | Show failure with exit code |
| `task.timeout` | Show timeout notice |

### Files Created

- `internal/tui/task/model.go` — task output TUI
- `cmd/tasks/tasks.go` — parent command
- `cmd/tasks/list.go` — list tasks
- `cmd/tasks/watch.go` — watch task output

---

## Feature 4: Real-Time Dashboard

Replace the 30-second polling dashboard with WebSocket-powered real-time updates.

### Changes

The dashboard already subscribes to `team.{teamID}` via the WebSocket. Instead of polling, process incoming events to update the dashboard state incrementally.

### Events Used

| Event | Dashboard Update |
|-------|-----------------|
| `server.created` | Add server to table |
| `server.updated` | Update server row |
| `server.deleted` | Remove server from table |
| `server.connectivity` | Update status dot |
| `server.metrics` | Update load/memory if shown |
| `deployment.created` | Add to recent deployments |
| `deployment.finished` | Update deployment status |
| `deployment.failed` | Update deployment status (red) |
| `site.created` | Increment server site count |
| `site.deleted` | Decrement server site count |

### Fallback

Keep the 30-second REST poll as a fallback for full state reconciliation (in case events were missed during a reconnection).

### Files Modified

- `internal/tui/dashboard/model.go` — add WebSocket event handling alongside existing polling

---

## Feature 5: Event Watcher

A general-purpose command for tailing all events — useful for debugging and monitoring.

### Command

```bash
# Watch all team events
launchctl events

# Filter by category
launchctl events --filter server
launchctl events --filter deployment
launchctl events --filter "server.provision*"

# Watch a specific server
launchctl events --server <id>

# JSON output for scripting
launchctl events --json
```

### Output

```
 Events (team: acme-corp)

  12:34:01  server.connectivity     production-web    connected
  12:34:15  deployment.started      api.example.com   deploying
  12:34:18  deployment.progress     api.example.com   [2/8] Installing deps
  12:34:45  deployment.finished     api.example.com   success (30s)
  12:35:00  server.metrics          production-web    load: 0.45
  12:35:12  backup.started          production-db     my_app_db

  Listening... Press q to exit
```

### Files Created

- `internal/tui/events/model.go` — event watcher TUI
- `cmd/events.go` — events command

---

## Implementation Plan

### Phase 1: WebSocket Client Rewrite

1. **Rewrite `internal/api/websocket.go`**
   - New `WSManager` with auto-reconnect, heartbeat, event dispatcher
   - Backward-compatible: existing `WSClient` can wrap `WSManager`
   - Add connection state callbacks

2. **Add `internal/api/websocket_helpers.go`**
   - Event type constants (all 100+ event names)
   - Typed event payload structs for common events
   - Helper to parse `WSEvent.Data` into typed structs

### Phase 2: Enhance Deploy TUI

3. **Update `internal/tui/deploy/model.go`**
   - Subscribe to `deployment.{id}` channel
   - Add step progress bar
   - Add timestamps to log lines
   - Color-coded output
   - Elapsed time display

4. **Update `cmd/deploy/trigger.go`**
   - Pass deployment ID to TUI for targeted subscription
   - Handle deploy trigger failure gracefully

### Phase 3: Server Provisioning TUI

5. **Create `internal/tui/provision/model.go`**
   - Step checklist with progress bar
   - Handle all `server.provision_*` events
   - Elapsed time, error display

6. **Create `cmd/servers/watch.go`**
   - `launchctl servers watch <id>` command
   - Auto-detect if server is provisioning

7. **Update `cmd/servers/create.go`**
   - After creation, automatically enter provisioning watch mode

### Phase 4: Task Output Streaming

8. **Create `internal/tui/task/model.go`**
   - Scrollable viewport for task output
   - Status header with elapsed time

9. **Create `cmd/tasks/` command group**
   - `tasks.go` — parent command
   - `list.go` — list recent tasks for a server
   - `watch.go` — stream task output

10. **Add task API methods to `internal/api/`**
    - `ListTasks(serverID)`, `GetTask(serverID, taskID)`

### Phase 5: Real-Time Dashboard

11. **Update `internal/tui/dashboard/model.go`**
    - Add WSManager integration
    - Handle server/deployment/site events for incremental updates
    - Keep 30s REST poll as reconciliation fallback

### Phase 6: Event Watcher

12. **Create `internal/tui/events/model.go`**
    - Scrollable event log with timestamps
    - Event filtering by category/glob pattern
    - Color-coded by event type

13. **Create `cmd/events.go`**
    - `launchctl events` command
    - `--filter`, `--server`, `--json` flags

---

## WebSocket Client API (Phase 1 Detail)

### New `WSManager` Interface

```go
// Connect and start listening
manager := api.NewWSManager(cfg)
manager.Connect()
defer manager.Close()

// Subscribe to channels
manager.Subscribe("deployment.abc123")
manager.Subscribe("server.xyz789")

// Register event handlers
manager.On("deployment.log", func(e WSEvent) {
    var log DeploymentLogEvent
    json.Unmarshal(e.Data, &log)
    fmt.Println(log.Output)
})

manager.On("server.provision_progress", func(e WSEvent) {
    var progress ProvisionProgressEvent
    json.Unmarshal(e.Data, &progress)
    fmt.Printf("Progress: %d%%\n", progress.Progress)
})

// Bubble Tea integration
func listenWS(manager *WSManager) tea.Cmd {
    return func() tea.Msg {
        event := <-manager.Events()  // channel-based for Bubble Tea
        return wsEventMsg{event}
    }
}
```

### Reconnection Strategy

```
Attempt 1: wait 1s
Attempt 2: wait 2s
Attempt 3: wait 4s
Attempt 4: wait 8s
Attempt 5: wait 16s
Attempt 6+: wait 30s (max)
Reset backoff on successful connection
```

### Heartbeat

- Send WebSocket ping every 30 seconds
- If no pong received within 10 seconds, trigger reconnect
- Server-side pong handler already exists in launch-go

---

## Event Payload Types (Phase 1 Detail)

```go
// Deployment events
type DeploymentLogEvent struct {
    DeploymentID string `json:"deployment_id"`
    Output       string `json:"output"`
    LineNumber   int    `json:"line_number"`
}

type DeploymentProgressEvent struct {
    DeploymentID string `json:"deployment_id"`
    SiteID       string `json:"site_id"`
    ServerID     string `json:"server_id"`
    Step         string `json:"step"`
    Progress     int    `json:"progress"`
    Message      string `json:"message"`
    Status       string `json:"status"`
}

// Server provisioning events
type ProvisionProgressEvent struct {
    ServerID string `json:"server_id"`
    Progress int    `json:"progress"`
}

type ProvisionStepEvent struct {
    ServerID string `json:"server_id"`
    Step     string `json:"step"`
}

type ProvisionStatusEvent struct {
    ServerID string `json:"server_id"`
    Message  string `json:"message"`
}

type SoftwareInstalledEvent struct {
    ServerID string `json:"server_id"`
    Software string `json:"software"`
}

// Task events
type TaskOutputEvent struct {
    TaskID   string `json:"task_id"`
    ServerID string `json:"server_id"`
    Output   string `json:"output"`
}

type TaskStatusEvent struct {
    TaskID   string `json:"task_id"`
    ServerID string `json:"server_id"`
    Name     string `json:"name"`
    Status   string `json:"status"`
    ExitCode int    `json:"exit_code"`
}

// Server events
type ServerConnectivityEvent struct {
    ServerID  string `json:"server_id"`
    Connected bool   `json:"connected"`
}

type ServerMetricsEvent struct {
    ServerID string  `json:"server_id"`
    Load     float64 `json:"load"`
    Memory   struct {
        Total int64 `json:"total"`
        Used  int64 `json:"used"`
        Free  int64 `json:"free"`
    } `json:"memory"`
    Disk struct {
        Total int64 `json:"total"`
        Used  int64 `json:"used"`
        Free  int64 `json:"free"`
    } `json:"disk"`
}
```

---

## Dependencies

### Existing (no new deps needed)

```
github.com/gorilla/websocket       — WebSocket client (already used)
github.com/charmbracelet/bubbletea — TUI framework (already used)
github.com/charmbracelet/bubbles   — Spinner, viewport (already used)
github.com/charmbracelet/lipgloss  — Styling (already used)
```

### Optional

```
github.com/charmbracelet/bubbles/progress — Progress bar widget (for provisioning)
```

---

## Testing

- **Unit tests**: Mock WebSocket server to test reconnection, event dispatch, heartbeat
- **Integration tests**: Connect to running launch-go instance, trigger deployment, verify events received
- **Manual testing**: Each TUI can be tested by triggering the corresponding action in the web UI and watching the CLI
