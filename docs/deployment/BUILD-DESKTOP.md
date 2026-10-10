# Building the BusinessOS Desktop App

BusinessOS ships as an Electron desktop app with a **bundled OptimalEngine** (an
Elixir release that carries its own Erlang runtime). That bundled runtime is
**OS- and architecture-locked**: it embeds native ERTS binaries that only run on
the platform they were built on. A macOS engine cannot run on Windows, a Windows
engine cannot run on Linux, and so on.

**Consequence: you must build each installer on its own OS.** There is no
cross-compiling the desktop app. Build the Mac app on a Mac, the Windows app on
Windows, the Linux app on Linux. Each build stages the matching engine release
into `desktop/resources/engine/<platform>-<arch>/`, which electron-forge then
packs into the installer via `extraResource`.

All three flows share the same shape:

1. Install toolchains (Erlang, Elixir, Node).
2. Build the OptimalEngine prod release (`MIX_ENV=prod mix release optimal`).
3. Stage it into `desktop/resources/engine/<platform>-<arch>/`.
4. `npm ci && npm run build:all` in `desktop/`, then `electron-forge make`.
5. Optionally upload the artifacts to `gs://businessos-downloads`.

---

## macOS (automated locally)

The Mac build runs on Roberto's machine and is the day-to-day path.

- **Needs:** Erlang + Elixir + Node (via Homebrew or asdf), Xcode command line
  tools. Google Cloud SDK if uploading.
- **Command:**
  1. Build + stage the engine for the host arch (Apple Silicon = `arm64`):
     ```bash
     (cd optimal-engine && MIX_ENV=prod mix release optimal --overwrite)
     desktop/scripts/stage-engine.sh          # stages to resources/engine/darwin-<arch>
     ```
  2. Build the app and produce the DMG (in `desktop/`):
     ```bash
     npm ci && npm run build:all
     npx electron-forge package
     # The DMG is assembled from the packaged .app via hdiutil
     # (electron-forge's DMG maker is unreliable here, so we package + hdiutil).
     ```
- **Output:** the packaged `.app` under `desktop/out/`, and the `.dmg` produced by
  the `hdiutil` step. The DMG is currently **unsigned**.
- **Upload:** `gsutil cp <path>.dmg gs://businessos-downloads/` (needs
  `gcloud auth login`).

---

## Linux (Ubuntu/Debian)

Run `scripts/build-linux.sh` on an Ubuntu or Debian box.

- **Needs:** the script installs `erlang elixir nodejs npm rpm fakeroot dpkg
  rsync curl` via `apt-get`. If the distro's Elixir/Erlang are too old, use asdf
  or the Erlang Solutions repo. Google Cloud SDK if uploading.
- **Command** (from the repo root):
  ```bash
  ./scripts/build-linux.sh            # build only
  ./scripts/build-linux.sh --upload   # build + upload (needs: gcloud already logged in)
  ```
- **Output:** `.deb`, `.rpm`, and `.AppImage` installers in `desktop/out/make/`.
- **Upload:** pass `--upload`; it copies the artifacts to
  `gs://businessos-downloads` with `gsutil`.

---

## Windows

Run `scripts/build-windows.ps1` on a Windows machine, in PowerShell (Run as
Administrator so the toolchain install can succeed).

- **Needs:** the script installs **Erlang + Elixir + Node** via `winget` (falling
  back to Chocolatey), printing a clear manual-install link if neither package
  manager is present. Two extra tools are **not** auto-installed but are required
  by `npm run build:all` - the script warns if they are missing:
  - **Git for Windows** (provides `bash`), because `build:all` runs the
    `desktop/scripts/*.sh` helper scripts.
  - **Go** (https://go.dev/dl), because the backend step cross-compiles the Go
    server.
  Google Cloud SDK (`gsutil`) is needed only for `-Upload`.
- **Command** (from the repo root):
  ```powershell
  .\scripts\build-windows.ps1            # build only
  .\scripts\build-windows.ps1 -Upload    # build + upload (needs: gcloud already logged in)
  ```
- **Output:** a Squirrel installer (`BusinessOS ... Setup.exe`) plus its `.nupkg`
  in `desktop\out\make\`.
- **Upload:** pass `-Upload`; it copies the `.exe`/`.nupkg` to
  `gs://businessos-downloads` with `gsutil`.

---

## The download bucket

All platforms publish to the same place: **`gs://businessos-downloads`**. Uploads
require an authenticated Google Cloud SDK (`gcloud auth login`) with write access
to that bucket. Objects there are what the public download page serves.
