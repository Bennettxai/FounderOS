# BusinessOS Workspace Container

Lightweight Alpine-based workspace container with essential development tools.

## Specifications

- **Base Image**: Alpine Linux 3.19
- **Image Size**: ~150MB (target < 200MB)
- **User**: `workspace` (UID 1000, non-root)
- **Working Directory**: `/workspace`
- **Default Shell**: `/bin/bash`

## Installed Tools

- **Shells**: bash, zsh
- **Version Control**: git
- **Network**: curl
- **Editors**: vim, nano
- **Languages**: python3 (with pip), nodejs (with npm)
- **Utilities**: ca-certificates, tzdata

## Building

```bash
# From this directory
./build.sh

# Or manually
docker build -t businessos-workspace:latest .
```

## Usage

### Basic Interactive Shell
```bash
docker run -it --rm businessos-workspace:latest
```

### Mount Current Directory
```bash
docker run -it --rm -v $(pwd):/workspace businessos-workspace:latest
```

### With Network Access
```bash
docker run -it --rm --network host -v $(pwd):/workspace businessos-workspace:latest
```

### Run Specific Command
```bash
docker run --rm -v $(pwd):/workspace businessos-workspace:latest python3 script.py
```

## Security Features

- Runs as non-root user (UID 1000)
- Minimal attack surface with Alpine base
- No unnecessary packages installed
- Read-only root filesystem compatible

## Environment

The container includes:
- Common bash aliases (ll, la, l)
- Colored prompt for better UX
- Standard PATH with python3 and node binaries

## Size Optimization

The image is optimized for size:
- Single-layer package installation
- APK cache cleanup
- No unnecessary documentation
- Minimal base image (Alpine)

Expected final size: ~150MB
