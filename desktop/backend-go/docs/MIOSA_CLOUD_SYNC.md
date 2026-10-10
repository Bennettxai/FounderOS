# MIOSA Cloud Sync — Design Reference

This document describes how the local-to-cloud sync system works across
all three BusinessOS deployment targets: Docker/web, bare-metal Go binary,
and Electron desktop app.

See ADR-003 for the architectural decision record.

---

## 1. Overview

```
User machine                         MIOSA Cloud
┌─────────────────────────────────┐  ┌──────────────────────┐
│  BusinessOS                     │  │  api.miosa.ai        │
│                                 │  │                      │
│  SvelteKit frontend             │  │  • Workspace config  │
│       │ /api/miosa/sync         │  │    backup            │
│       ▼                         │  │  • Agent marketplace │
│  Go backend (Gin)               │  │  • Template registry │
│  internal/integrations/miosa/   │  │  • Cross-device sync │
│       │ sdk-go CloudClient      │  │                      │
│       └─────────────────────────┼─►│                      │
│                                 │  └──────────────────────┘
│  OSA Agent (Elixir/OTP)         │
│  localhost:8089 (local mode)    │
│  OR api.miosa.ai (cloud mode)   │
└─────────────────────────────────┘
```

The Go backend is the **only** component that communicates with MIOSA Cloud.
The SvelteKit frontend never contacts api.miosa.ai directly. The MIOSA API
key is stored server-side and is never sent to the browser.

---

## 2. What syncs

The sync payload is a `WorkspaceManifest` (see
`internal/integrations/miosa/sync_service.go`).

### Included

| Category             | Details                                                    |
|----------------------|------------------------------------------------------------|
| Workspace settings   | Name, description, icon, theme, feature flags             |
| Agent configurations | Name, model, system prompt, tools, temperature (NO history)|
| App definitions      | Schema, layout, permissions (NO data rows)                 |
| Template definitions | Body, variables, metadata (NO documents from templates)    |

### Excluded (always stays local)

- Tasks, projects, contacts, deals
- Conversation and message history
- Emails, calendar events
- Uploaded files and document content
- Memory embeddings and vector store data
- OAuth tokens and user credentials
- Audit logs

---

## 3. Environment variables

```bash
# OSA_MODE: controls the backend runtime target
# local  (default) → direct HTTP adapter → localhost:8089
# cloud            → direct HTTP adapter → api.miosa.ai
OSA_MODE=local

# MIOSA_API_KEY: required when OSA_MODE=cloud
# Obtain from: https://app.miosa.ai/settings/api-keys
MIOSA_API_KEY=

# MIOSA_CLOUD_URL: override the cloud endpoint (rarely needed)
# Default: https://api.miosa.ai
MIOSA_CLOUD_URL=
```

### Priority order

In all three deployment modes the Go backend resolves config in this order:

1. OS environment variables (highest priority — used in Docker and CI)
2. `.env` file in the working directory (local dev and bare-metal)
3. Viper defaults (fallback, e.g. `OSA_MODE=local`)

---

## 4. Docker / web deployment

Set the variables in your `docker-compose.yml` or container environment:

```yaml
# docker-compose.yml
services:
  backend:
    environment:
      OSA_MODE: cloud
      MIOSA_API_KEY: ${MIOSA_API_KEY}   # from host .env or CI secret
```

The frontend container is unaffected; it does not need any MIOSA variables.

---

## 5. Bare-metal Go binary

Add to `.env` in the same directory as the binary:

```bash
OSA_MODE=cloud
MIOSA_API_KEY=msk_prod_xxxxxxxxxxxxxxxx
```

Then restart the server: `./businessos-server` or `go run ./cmd/server`.

---

## 6. Electron desktop app

The Electron main process is responsible for injecting `MIOSA_API_KEY` into
the spawned Go backend child process. The key is stored in the OS keychain
via `keytar`, not in a plaintext file.

### Key storage flow

```
User enters API key in Settings
        │
        ▼ IPC: ipcRenderer.invoke('miosa:set-api-key', key)
Electron main process
        │ keytar.setPassword('businessos', 'miosa-api-key', key)
        ▼
OS Keychain (macOS Keychain, Windows Credential Manager, libsecret on Linux)
        │
        ▼ On next backend process spawn
backend = spawn(serverBinary, [], {
  env: {
    ...process.env,
    MIOSA_API_KEY: await keytar.getPassword('businessos', 'miosa-api-key'),
    OSA_MODE: await store.get('osa-mode') ?? 'local',
  }
})
```

### Settings UI flow in Electron

The Settings page is the same Svelte component (`MIOSACloudPanel.svelte`) in
both web and Electron. In Electron, when the user types a new API key:

1. The UI sends the key to the backend via `POST /api/miosa/ping`
   — this validates the key against MIOSA Cloud.
2. If valid, the UI calls an Electron-specific IPC handler
   `miosa:set-api-key` via `window.electronAPI.setMIOSAApiKey(key)`.
3. The Electron main process stores the key in the OS keychain.
4. The main process sends `SIGTERM` to the Go backend child process and
   restarts it with the new environment variable injected.
5. The frontend polls `GET /api/miosa/status` until the backend reports
   `connected: true`.

This restart approach is simpler than live config reloading and is
acceptable for a settings change that users perform rarely.

### Electron preload bridge (pseudocode)

```typescript
// preload.ts
contextBridge.exposeInMainWorld('electronAPI', {
  setMIOSAApiKey: (key: string) =>
    ipcRenderer.invoke('miosa:set-api-key', key),
  getMIOSAMode: () =>
    ipcRenderer.invoke('miosa:get-mode'),
  setMIOSAMode: (mode: 'local' | 'cloud') =>
    ipcRenderer.invoke('miosa:set-mode', mode),
});

// main.ts
ipcMain.handle('miosa:set-api-key', async (_, key: string) => {
  await keytar.setPassword('businessos', 'miosa-api-key', key);
  restartBackendProcess();
});

ipcMain.handle('miosa:set-mode', async (_, mode: string) => {
  await store.set('osa-mode', mode);
  restartBackendProcess();
});
```

---

## 7. API endpoints

All endpoints require a valid user session (auth middleware applied).

| Method | Path               | Description                                          |
|--------|--------------------|------------------------------------------------------|
| GET    | /api/miosa/status  | Connection status. No external call. Safe to poll.   |
| POST   | /api/miosa/ping    | Validates API key against MIOSA Cloud.               |
| POST   | /api/miosa/sync    | Pushes WorkspaceManifest to MIOSA Cloud.             |

### GET /api/miosa/status — response

```json
{
  "mode": "cloud",
  "connected": true,
  "api_key_set": true,
  "last_sync": "2026-02-24T14:30:00Z"
}
```

### POST /api/miosa/sync — request body

```json
{ "workspace_id": "uuid" }
```

Response:

```json
{
  "success": true,
  "synced_at": "2026-02-24T14:31:00Z",
  "manifest_id": "mfst_abc123"
}
```

---

## 8. "Publish to MIOSA Cloud" button — step by step

When the user clicks the button in `MIOSACloudPanel.svelte`:

1. **Frontend** calls `POST /api/miosa/sync` with the current workspace ID.
2. **Go backend** (`MIOSASyncHandler.Sync`) validates the workspace ID.
3. **SyncService.Sync** calls `ManifestProvider.BuildManifest(workspaceID)`.
4. **ManifestProvider** (implemented by the workspace service) queries the
   local database for workspace settings, agent configs, app definitions,
   and templates. Business data tables are not touched.
5. The manifest is JSON-serialized and sent by the backend's direct MIOSA
   HTTP adapter.
6. The backend calls the configured `MIOSA_CLOUD_URL` with the
   `Authorization: Bearer {MIOSA_API_KEY}` header.
7. MIOSA Cloud stores the manifest and returns a `manifest_id`.
8. **Go backend** returns `{ success: true, manifest_id: "..." }`.
9. **Frontend** shows the success banner and updates `last_sync`.

In local mode (OSA_MODE=local), step 2 detects the mode and immediately
returns `{ success: true }` without any network call.

---

## 9. Security considerations

- The MIOSA API key is stored in the Go process environment, not in the
  database and not in the browser.
- The `/api/miosa/*` endpoints require an authenticated session; anonymous
  requests are rejected by the existing auth middleware.
- The manifest payload is serialized to JSON by the Go backend before being
  sent; there is no raw SQL or binary data in the sync payload.
- In Electron, the key is stored in the OS keychain and injected at process
  spawn time; it is never written to disk in plaintext.
- Rate limiting: the existing Gin rate-limiter middleware applies to
  `/api/miosa/sync`; users cannot flood MIOSA Cloud through the backend.
