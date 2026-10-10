# Common Issues & Troubleshooting

This document covers common issues you may encounter when setting up BusinessOS for the first time.

---

## Table of Contents
- [Google OAuth "Client Not Found" Error](#google-oauth-client-not-found-error)
- [Terminal "zsh not found" Error (Linux)](#terminal-zsh-not-found-error-linux)
- [Database Connection Issues](#database-connection-issues)
- [Redis Connection Issues](#redis-connection-issues)
- [Port Already in Use](#port-already-in-use)

---

## Google OAuth "Client Not Found" Error

### Symptoms
- Google login shows "OAuth client was not found" error
- Google redirects to an error page saying the client doesn't exist
- The OAuth URL contains `client_id=your-client-id.apps.googleusercontent.com` (placeholder value)

### Root Cause
The Go backend's config loader has a function `applyDotenvOverrides()` that explicitly maps `.env` values to the config struct. If Google OAuth fields are missing from this function, the values from `.env` won't be applied even though the file is loaded correctly.

### How to Verify the Issue
Run this command to check what client_id the server is using:
```bash
curl -s "http://localhost:8001/api/auth/google?redirect=test" 2>&1 | grep -oE 'client_id=[^&"]*'
```

If it shows `client_id=your-client-id.apps.googleusercontent.com` instead of your actual Google Client ID, you have this issue.

### Fix

**Step 1:** Open `desktop/backend-go/internal/config/config.go`

**Step 2:** Find the `applyDotenvOverrides` function (around line 344)

**Step 3:** Ensure these lines are present at the end of the function (before the closing `}`):

```go
// Google OAuth
if v := strings.TrimSpace(vars["GOOGLE_CLIENT_ID"]); v != "" {
    cfg.GoogleClientID = v
}
if v := strings.TrimSpace(vars["GOOGLE_CLIENT_SECRET"]); v != "" {
    cfg.GoogleClientSecret = v
}
if v := strings.TrimSpace(vars["GOOGLE_REDIRECT_URI"]); v != "" {
    cfg.GoogleRedirectURI = v
}
```

**Step 4:** Rebuild and restart the backend:
```bash
cd desktop/backend-go
rm -f server
go build -o server cmd/server/main.go
./server
```

**Step 5:** Verify the fix:
```bash
curl -s "http://localhost:8001/api/auth/google?redirect=test" 2>&1 | grep -oE 'client_id=[^&"]*'
```

You should now see your actual Google Client ID.

### Prevention
When adding new OAuth providers or environment variables that need to be read from `.env` in development mode, always add them to the `applyDotenvOverrides()` function in `config.go`.

---

## Terminal "zsh not found" Error (Linux)

### Symptoms
- BusinessOS terminal shows: `failed to start PTY: exec: "zsh": executable file not found in $PATH`
- Terminal window shows error and disconnects immediately

### Root Cause
The terminal was defaulting to `zsh` shell, which is standard on macOS but not installed by default on Linux/Ubuntu.

### Fix Options

**Option 1: Install zsh (recommended)**
```bash
sudo apt install zsh
```

After installing, the first time you open the terminal you'll see a configuration wizard. Just type `0` and press Enter to create a basic config.

**Option 2: The code fix (already applied)**

If you're on an older version, update `desktop/backend-go/internal/terminal/websocket.go`:

The terminal now auto-detects the available shell:
- macOS: uses `zsh` if available, falls back to `bash`
- Linux: uses `bash` (since zsh isn't installed by default)

After updating, rebuild the backend:
```bash
cd desktop/backend-go
go build -o server cmd/server/main.go
./server
```

---

## Database Connection Issues

### Symptoms
- Backend fails to start with "failed to connect to database" error
- Error mentions "password authentication failed"
- Error mentions "connection refused"

### Common Causes & Fixes

#### 1. PostgreSQL not running
```bash
# Check if PostgreSQL is running
pg_isready -h localhost -p 5432

# Start PostgreSQL (Docker)
docker start businessos-postgres

# Or start PostgreSQL (systemd)
sudo systemctl start postgresql
```

#### 2. Wrong credentials in .env
Check `desktop/backend-go/.env` and ensure DATABASE_URL has correct credentials:
```bash
# For local Docker PostgreSQL (from docker-compose.yml)
DATABASE_URL=postgres://rhl:password@localhost:5432/business_os?sslmode=disable

# For Supabase
DATABASE_URL=postgres://postgres.[PROJECT]:[PASSWORD]@aws-0-us-east-1.pooler.supabase.com:6543/postgres?pgbouncer=true
```

#### 3. Database doesn't exist
```bash
# Create the database
psql -U postgres -c "CREATE DATABASE business_os;"

# Or via Docker
docker exec -it businessos-postgres psql -U rhl -c "CREATE DATABASE business_os;"
```

---

## Redis Connection Issues

### Symptoms
- Backend logs show Redis connection errors
- Session caching disabled

### Fix
```bash
# Start Redis via Docker
docker start businessos-redis

# Or install and start Redis locally
sudo apt install redis-server
sudo systemctl start redis

# Verify Redis is running
redis-cli ping
# Should return: PONG
```

---

## Port Already in Use

### Symptoms
- "address already in use" error when starting backend or frontend

### Fix
```bash
# Find what's using port 8001 (backend)
lsof -i :8001
# or
ss -tlnp | grep 8001

# Find what's using port 5173 (frontend)
lsof -i :5173

# Kill the process
kill -9 <PID>
```

---

## Docker Terminal System (Resolved Jan 2026)

### Symptoms
- `No such image: businessos-workspace:latest` when opening a terminal
- `bash: $'\r': command not found` (Windows-style line endings in init script)
- `invalid mount path: must be absolute` (relative bind mount path in container config)

### Fix

**Step 1: Build the workspace Docker image**
```bash
cd desktop/backend-go/docker/workspace/
docker build -t businessos-workspace:latest .
```

**Step 2: Fix line endings in init scripts (CRLF to LF)**
```bash
sed -i 's/\r$//' desktop/backend-go/docker/workspace/businessos_init.sh
sed -i 's/\r$//' desktop/backend-go/internal/terminal/businessos_init.sh
```

**Step 3: Fix absolute paths in `internal/container/container.go`**
- Add `"path/filepath"` to the import block
- Update `getInitScriptPath()` to use `filepath.Abs()` instead of relative paths
- Remove any relative paths from the `possiblePaths` slice

**Step 4: Rebuild the backend**
```bash
cd desktop/backend-go
go build -o bin/server-new ./cmd/server
```

### Verify the Fix
```bash
# Confirm the image exists
docker images businessos-workspace:latest

# Confirm correct line endings (should show "ASCII text executable", not "CRLF")
file desktop/backend-go/internal/terminal/businessos_init.sh
```

Open the terminal in the frontend — the prompt should show: `[docker] /workspace $`

### Files Affected
- `desktop/backend-go/internal/container/container.go`
- `desktop/backend-go/docker/workspace/businessos_init.sh`
- `desktop/backend-go/internal/terminal/businessos_init.sh`

---

## First-Time Setup Checklist

When setting up BusinessOS on a new machine, follow these steps:

1. **Clone the repository**
   ```bash
   git clone <repo-url>
   cd BusinessOS
   ```

2. **Copy environment files**
   ```bash
   cp desktop/backend-go/.env.example desktop/backend-go/.env
   cp frontend/.env.production.example frontend/.env
   ```

3. **Edit backend .env** with your actual credentials:
   - `DATABASE_URL` - PostgreSQL connection string
   - `GOOGLE_CLIENT_ID` - From Google Cloud Console
   - `GOOGLE_CLIENT_SECRET` - From Google Cloud Console
   - `GOOGLE_REDIRECT_URI` - Should be `http://localhost:8001/api/auth/google/callback`

4. **Start infrastructure**
   ```bash
   # Start PostgreSQL and Redis (via Docker)
   docker-compose up -d postgres redis

   # Or use ./dev.sh
   ./dev.sh start
   ```

5. **Verify Google OAuth is configured correctly**
   ```bash
   curl -s "http://localhost:8001/api/auth/google?redirect=test" 2>&1 | grep -oE 'client_id=[^&"]*'
   ```
   - Should show your actual Google Client ID, NOT `your-client-id.apps.googleusercontent.com`

6. **If OAuth shows placeholder values**, apply the fix described in the [Google OAuth section](#google-oauth-client-not-found-error)

---

## Getting Help

If you encounter issues not covered here:
1. Check the backend logs: `tail -f .startup-logs/backend.log`
2. Check the frontend logs: `tail -f .startup-logs/frontend.log`
3. Run `./dev.sh status` to see service health
4. Search existing docs in `/docs/` directory

---

*Last Updated: January 2026*
