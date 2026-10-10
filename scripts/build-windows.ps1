<#
.SYNOPSIS
  Build the BusinessOS Windows desktop app ON a Windows machine.

.DESCRIPTION
  Run this on your Windows box (NOT on the Mac or Linux - a Windows app + its
  bundled OptimalEngine must be built on Windows, because the Elixir release
  ships its own architecture- and OS-locked Erlang runtime). It produces a
  Squirrel installer (Setup .exe) + .nupkg in desktop\out\make\ and (optionally)
  uploads them to the download bucket.

  Mirrors scripts/build-linux.sh step-for-step:
    1. toolchains (Erlang + Elixir + Node)
    2. build + stage the native Windows OptimalEngine release
    3. build the app + make the installer
    4. optional upload to gs://businessos-downloads

.PARAMETER Upload
  After a successful build, upload the installer(s) to gs://businessos-downloads.
  You must already be `gcloud auth login`'d.

.EXAMPLE
  .\scripts\build-windows.ps1            # build only

.EXAMPLE
  .\scripts\build-windows.ps1 -Upload    # build + upload (needs: gcloud already logged in)

.NOTES
  Run from an elevated (Administrator) PowerShell if you want the automatic
  toolchain install to succeed. Open "Windows PowerShell" -> Run as administrator.
#>
[CmdletBinding()]
param(
  [switch]$Upload
)

# Fail fast, like `set -euo pipefail` in the bash scripts.
$ErrorActionPreference = "Stop"

# --- helpers ----------------------------------------------------------------

# Repo root = parent of the scripts\ folder this file lives in.
$REPO = (Resolve-Path (Join-Path $PSScriptRoot "..")).Path

function Have($name) { return [bool](Get-Command $name -ErrorAction SilentlyContinue) }

# Run a native command and stop if it returns a non-zero exit code (robocopy is
# handled separately - it uses non-zero codes to mean success).
function Invoke-Native {
  param([Parameter(Mandatory)][scriptblock]$Cmd)
  & $Cmd
  if ($LASTEXITCODE -ne 0) {
    throw "command failed (exit $LASTEXITCODE): $($Cmd.ToString().Trim())"
  }
}

# Install a package, preferring winget, falling back to choco. If neither
# package manager is present we print a clear message and bail so the user
# knows exactly what to install by hand.
function Ensure-Tool {
  param(
    [Parameter(Mandatory)][string]$Command,   # the CLI to look for on PATH
    [string]$WingetId,                         # winget package id (optional)
    [string]$ChocoId,                          # choco package id (optional)
    [string]$Manual                            # manual-install hint
  )
  if (Have $Command) {
    Write-Host "    ok: $Command already installed"
    return
  }
  Write-Host "    missing: $Command - attempting install..."
  if ($WingetId -and (Have "winget")) {
    winget install --id $WingetId --accept-source-agreements --accept-package-agreements -e
  }
  elseif ($ChocoId -and (Have "choco")) {
    choco install $ChocoId -y
  }
  else {
    throw @"
Cannot auto-install '$Command': neither winget nor choco is available.
Install it manually, then re-run this script.
  $Manual
"@
  }
  # winget/choco update PATH for NEW shells; refresh this session so the very
  # next step can find the freshly installed binary.
  $env:Path = [System.Environment]::GetEnvironmentVariable("Path", "Machine") + ";" +
              [System.Environment]::GetEnvironmentVariable("Path", "User")
  if (-not (Have $Command)) {
    throw "installed a package for '$Command' but it is still not on PATH - open a new terminal and re-run."
  }
}

Write-Host "==> BusinessOS Windows build"
Write-Host "==> repo: $REPO"

# 1. Toolchains (Windows). Skips anything already installed.
Write-Host "==> Installing build toolchains (erlang, elixir, node)..."
# Elixir needs Erlang/OTP; install Erlang first so the Elixir install sees it.
Ensure-Tool -Command "erl"  -WingetId "Erlang.ErlangOTP" -ChocoId "erlang" `
  -Manual "Erlang/OTP: https://www.erlang.org/downloads"
Ensure-Tool -Command "elixir" -ChocoId "elixir" `
  -Manual "Elixir: https://elixir-lang.org/install.html#windows (the installer bundles a compatible Erlang)"
Ensure-Tool -Command "node" -WingetId "OpenJS.NodeJS.LTS" -ChocoId "nodejs-lts" `
  -Manual "Node.js LTS: https://nodejs.org/en/download"

# `npm run build:all` shells out to the desktop\scripts\*.sh build scripts and
# the Go backend cross-compile. Those need bash (Git Bash) and Go on PATH; we
# don't auto-install them, but we warn loudly so the build doesn't die cryptically.
if (-not (Have "bash")) {
  Write-Warning "bash not found. 'npm run build:all' runs .sh helper scripts; install Git for Windows (https://git-scm.com/download/win) so npm can find sh."
}
if (-not (Have "go")) {
  Write-Warning "go not found. The backend build (build-backend.sh) cross-compiles the Go server; install Go (https://go.dev/dl) if build:all fails on the backend step."
}

# 2. Build + stage the native Windows OptimalEngine.
Write-Host "==> Building the OptimalEngine (Windows release)..."
Set-Location (Join-Path $REPO "optimal-engine")
$env:MIX_ENV = "prod"
# local.hex / local.rebar can fail harmlessly if already present - don't stop on them.
& mix local.hex --force 2>$null
& mix local.rebar --force 2>$null
Invoke-Native { mix deps.get }
Invoke-Native { mix release optimal --overwrite }

$SRC  = Join-Path $REPO "optimal-engine\_build\prod\rel\optimal"
$DEST = Join-Path $REPO "desktop\resources\engine\win32-x64"
if (Test-Path $DEST) { Remove-Item -Recurse -Force $DEST }
New-Item -ItemType Directory -Force -Path $DEST | Out-Null

# robocopy is the Windows rsync: mirror the release but EXCLUDE runtime cruft
# (tmp/, log/, Erlang FIFO pipes) so electron-forge's copy step never chokes.
#   /E    copy subdirs incl. empty      /XD  exclude dirs      /XF  exclude files
#   /NFL /NDL /NJH /NJS /NP             quiet output
robocopy $SRC $DEST /E /XD tmp log .optimal /XF "*.pipe.*" "erl_crash.dump" /NFL /NDL /NJH /NJS /NP | Out-Null
# robocopy exit codes 0-7 = success (8+ = failure). Normalize so $ErrorActionPreference
# doesn't treat a normal "files copied" (code 1) as a failure.
if ($LASTEXITCODE -ge 8) { throw "robocopy failed staging the engine (exit $LASTEXITCODE)" }
$global:LASTEXITCODE = 0
Write-Host "==> Engine staged to resources\engine\win32-x64"

# 3. Build the app + make the Windows installer.
Write-Host "==> Building the desktop app + installer..."
Set-Location (Join-Path $REPO "desktop")
Invoke-Native { npm ci }
Invoke-Native { npm run build:all }
Invoke-Native { npx electron-forge make --arch=x64 }

$MAKE = Join-Path $REPO "desktop\out\make"
Write-Host "==> Done. Installer(s) are in: $MAKE"
Get-ChildItem -Path $MAKE -Recurse -Include *.exe, *.nupkg -File -ErrorAction SilentlyContinue |
  ForEach-Object { Write-Host "    $($_.FullName)" }

# 4. Optional upload to the download bucket (you must already be `gcloud auth login`'d).
if ($Upload) {
  if (-not (Have "gsutil")) {
    throw "gsutil not found - install the Google Cloud SDK (https://cloud.google.com/sdk) and run 'gcloud auth login' before -Upload."
  }
  Write-Host "==> Uploading installer(s) to gs://businessos-downloads ..."
  Get-ChildItem -Path $MAKE -Recurse -Include *.exe, *.nupkg -File |
    ForEach-Object {
      Write-Host "    uploading $($_.Name)"
      & gsutil cp $_.FullName "gs://businessos-downloads/"
      if ($LASTEXITCODE -ne 0) { throw "gsutil upload failed for $($_.Name) (exit $LASTEXITCODE)" }
    }
  Write-Host "==> Uploaded. They are now downloadable."
}
