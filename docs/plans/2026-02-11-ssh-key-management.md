# SSH Key Management — Design Document

Secure passwordless SSH access from the CLI to managed servers, with safety guardrails to protect against token compromise.

---

## Problem

Currently, `launchctl servers ssh <id>` launches an SSH session that prompts for the server password every time. Users must manually manage SSH keys. This creates friction and encourages insecure practices (reusing passwords, skipping SSH keys entirely).

## Solution

Generate an SSH key pair locally on the CLI, upload only the public key to the server via API, and use the local private key for all SSH connections. The private key never leaves the client machine.

---

## Architecture

```
CLI (client machine)                    API                         Server
────────────────────                    ───                         ──────
1. ssh-keygen (ed25519)
   ~/.config/launchctl/keys/
   ├── id_launchctl         (private)
   └── id_launchctl.pub     (public)

2. POST /api/servers/{id}/ssh-keys ──→  3. Validate 2FA token
   {                                    4. Append public key to
     public_key: "ssh-ed25519...",         ~/.ssh/authorized_keys
     label: "macbook-karthick",            on the target server
     two_factor_code: "123456"          5. Return key metadata
   }

6. ssh -i ~/.config/launchctl/keys/id_launchctl
       -p {port} user@ip               ──→  Passwordless connection
```

---

## Security Model

### Threat: Stolen API Token

An attacker with a stolen API token should NOT be able to push an SSH key without additional verification.

### Mitigation: 2FA Gate

Every SSH key push requires a valid 2FA code from the user's authenticator app. This means an attacker needs both:
- The API token (something they have)
- The 2FA device (something they don't have)

This follows the same "sudo mode" pattern used by GitHub for sensitive operations.

---

## Safety Features

### 1. 2FA Confirmation for Key Push

Every `POST /api/servers/{id}/ssh-keys` request must include a valid `two_factor_code`. The API rejects the request without it.

**CLI flow:**
```
$ launchctl servers ssh <id>

No SSH key configured for this server.
Push your CLI SSH key? This requires 2FA verification.

Enter 2FA code: 123456
✓ SSH key pushed to server "Gigcodes Main"
✓ Connecting...
```

**API endpoint:**
```
POST /api/servers/{id}/ssh-keys
Authorization: Bearer {token}

{
  "public_key": "ssh-ed25519 AAAAC3... launchctl-cli",
  "label": "macbook-pro-karthick",
  "two_factor_code": "123456"
}

Response 201:
{
  "success": true,
  "data": {
    "id": "key_abc123",
    "label": "macbook-pro-karthick",
    "fingerprint": "SHA256:xyzabc...",
    "created_at": "2026-02-11T10:00:00Z"
  }
}

Response 403 (invalid 2FA):
{
  "success": false,
  "message": "Invalid two-factor authentication code"
}
```

### 2. Key Labeling

Each key is labeled with the device name so team admins can audit who has SSH access from which machine.

**Label format:** `{hostname}-{username}` (auto-detected, user can override)

**Example labels:**
- `macbook-pro-karthick`
- `ci-server-github-actions`
- `desktop-office`

### 3. Notifications

When an SSH key is added to a server, notify the team owner/admins:

**Email notification:**
```
Subject: SSH key added to server "Gigcodes Main"

A new SSH key was added to your server:

  Server:      Gigcodes Main (156.67.110.189)
  Key label:   macbook-pro-karthick
  Fingerprint: SHA256:xyzabc...
  Added by:    karthick@example.com
  Time:        2026-02-11 10:00:00 UTC

If you did not authorize this, revoke the key immediately:
  https://launchctl.io/servers/{id}/ssh-keys
```

**In-app notification** (dashboard + CLI `launchctl notifications`):
```
SSH key "macbook-pro-karthick" added to server "Gigcodes Main" by karthick@example.com
```

### 4. Key Revocation

Admins can revoke SSH keys via API or dashboard.

**API endpoint:**
```
DELETE /api/servers/{id}/ssh-keys/{key-id}
Authorization: Bearer {token}

Response 200:
{
  "success": true,
  "message": "SSH key revoked and removed from server"
}
```

**CLI command:**
```
$ launchctl ssh-keys revoke <server-id> <key-id>
✓ SSH key revoked from server "Gigcodes Main"
```

The API removes the public key from the server's `authorized_keys` file.

### 5. One Key Per Device Per Server

The CLI checks if it has already pushed a key to a server. If so, it reuses the existing key rather than pushing duplicates.

**Logic:**
```
1. Check local key exists at ~/.config/launchctl/keys/id_launchctl
   - If not, generate one
2. GET /api/servers/{id}/ssh-keys → list existing keys
3. Check if local key fingerprint matches any existing key
   - If match, skip push (already configured)
   - If no match, prompt 2FA and push
4. Connect with local key
```

### 6. Key Audit Log

Track all SSH key operations for compliance and security review.

**API endpoint:**
```
GET /api/servers/{id}/ssh-keys/audit

Response 200:
{
  "success": true,
  "data": [
    {
      "action": "added",
      "key_label": "macbook-pro-karthick",
      "fingerprint": "SHA256:xyzabc...",
      "user": "karthick@example.com",
      "ip_address": "203.0.113.50",
      "created_at": "2026-02-11T10:00:00Z"
    },
    {
      "action": "revoked",
      "key_label": "old-laptop",
      "fingerprint": "SHA256:defghi...",
      "user": "admin@example.com",
      "ip_address": "203.0.113.60",
      "created_at": "2026-02-10T08:00:00Z"
    }
  ]
}
```

---

## CLI Implementation

### Key Storage

```
~/.config/launchctl/
├── config.json              (existing)
└── keys/
    ├── id_launchctl         (private key, 0600 permissions)
    └── id_launchctl.pub     (public key)
```

- Key type: Ed25519 (modern, fast, secure)
- Key generated on first SSH attempt or via `launchctl ssh-keys generate`
- Private key file permissions: `0600` (owner read/write only)
- Keys directory permissions: `0700` (owner only)

### New Commands

```
launchctl ssh-keys generate          Generate local key pair (if not exists)
launchctl ssh-keys show              Show local public key and fingerprint
launchctl ssh-keys list <server-id>  List SSH keys on a server
launchctl ssh-keys push <server-id>  Push local key to server (requires 2FA)
launchctl ssh-keys revoke <server-id> <key-id>  Revoke a key from server
```

### Modified SSH Flow

The existing `launchctl servers ssh <id>` and interactive nav SSH option will be updated:

```go
func sshIntoServer(client *api.Client, cfg *config.Config, server api.ServerResponse) {
    // 1. Ensure local key pair exists
    keyPath, err := ensureKeyPair()
    if err != nil {
        tui.ShowError("Failed to generate SSH key")
        return
    }

    // 2. Check if key is pushed to this server
    pushed, err := isKeyPushedToServer(client, server.ID, keyPath+".pub")
    if err != nil {
        tui.ShowWarning("Could not verify SSH key status, trying anyway...")
    }

    // 3. If not pushed, offer to push with 2FA
    if !pushed {
        tui.ShowInfo("No SSH key configured for this server.")
        if !tui.GetConfirmation("Push your CLI SSH key? (requires 2FA)") {
            // Fall back to password-based SSH
            sshWithPassword(server)
            return
        }

        code, err := tui.GetInput("Enter 2FA code", "123456", false, nil)
        if err != nil {
            return
        }

        if err := client.PushSSHKey(server.ID, keyPath+".pub", code); err != nil {
            tui.ShowError(fmt.Sprintf("Failed to push key: %s", err))
            sshWithPassword(server)
            return
        }

        tui.ShowSuccess("SSH key pushed successfully")
    }

    // 4. Connect with key
    sshWithKey(server, keyPath)
}
```

### Key Generation

```go
func ensureKeyPair() (string, error) {
    keyDir := filepath.Join(configDir(), "keys")
    keyPath := filepath.Join(keyDir, "id_launchctl")

    if fileExists(keyPath) {
        return keyPath, nil
    }

    os.MkdirAll(keyDir, 0700)

    // Use ssh-keygen for reliable key generation
    cmd := exec.Command("ssh-keygen",
        "-t", "ed25519",
        "-f", keyPath,
        "-N", "",  // no passphrase
        "-C", "launchctl-cli",
    )
    return keyPath, cmd.Run()
}
```

---

## API Endpoints Summary

| Method | Endpoint | Auth | Description |
|--------|----------|------|-------------|
| `GET` | `/api/servers/{id}/ssh-keys` | Token | List SSH keys on server |
| `POST` | `/api/servers/{id}/ssh-keys` | Token + 2FA | Push public key to server |
| `DELETE` | `/api/servers/{id}/ssh-keys/{key-id}` | Token | Revoke/remove a key |
| `GET` | `/api/servers/{id}/ssh-keys/audit` | Token (admin) | View key audit log |

---

## Server-Side Implementation Notes

### Adding a Key

When the API receives a valid `POST /ssh-keys`:
1. Validate the 2FA code
2. Validate the public key format (must be valid SSH public key)
3. SSH into the server (using the platform's root access)
4. Append the public key to `/home/{user}/.ssh/authorized_keys`
5. Set correct permissions (`authorized_keys` 0600, `.ssh` 0700)
6. Store key metadata in the database (label, fingerprint, user, server)
7. Send notification to team owner/admins
8. Log the action in audit trail

### Revoking a Key

When the API receives a valid `DELETE /ssh-keys/{key-id}`:
1. Look up the key by ID
2. SSH into the server
3. Remove the matching line from `authorized_keys`
4. Delete key metadata from database
5. Log the action in audit trail

---

## Security Checklist

- [ ] Private key never leaves the client machine
- [ ] Public key push requires valid 2FA code
- [ ] Keys are labeled with device identifier
- [ ] Team owner notified on key add/remove
- [ ] Admin can revoke any key via API or dashboard
- [ ] One key per device per server (no duplicates)
- [ ] All key operations logged in audit trail
- [ ] Public key format validated server-side
- [ ] Key file permissions enforced (0600 private, 0700 directory)
- [ ] Rate limiting on key push endpoint (e.g., 5 per hour)
- [ ] Failed 2FA attempts on key push are logged
- [ ] Revoked keys are immediately removed from server authorized_keys
