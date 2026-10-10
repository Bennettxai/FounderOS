# CRIT-5: Shell Path Injection via Terminal WebSocket `?shell=` Parameter

**Severity:** CRITICAL | **Exploitability:** 9/10 | **CVSS:** 9.8

## Location
- `desktop/backend-go/internal/terminal/websocket.go:139`
- `desktop/backend-go/internal/terminal/pty.go:25-47`

## Description
The `shell` query parameter from the WebSocket URL is passed directly into `exec.Command(shell, ...)` with zero validation. Any authenticated user can execute any binary on the server.

## PoC
```
GET /api/terminal/ws?shell=/usr/bin/python3&cols=80&rows=24
```
Spawns a Python REPL as the server process user with full access to the host filesystem and environment.

```
GET /api/terminal/ws?shell=/tmp/malicious_binary&cols=80&rows=24
```
Executes an attacker-uploaded binary.

## Fix
```go
var allowedShells = map[string]bool{
    "zsh": true, "bash": true, "sh": true, "fish": true,
}

func sanitizeShell(requested string) string {
    base := filepath.Base(requested)
    if allowedShells[base] {
        return base
    }
    return "" // fall back to auto-detect
}
```
