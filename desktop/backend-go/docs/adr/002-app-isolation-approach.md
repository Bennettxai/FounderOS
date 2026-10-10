# ADR-002: App Isolation Approach

## Status
Accepted

## Date
January 2026

## Context

BusinessOS allows users to generate and deploy custom applications through:

1. **Template-Based Generation:** Pre-built templates (SaaS dashboard, API backend, etc.)
2. **OSA-Powered Generation:** AI agent generates complete apps from natural language
3. **Terminal Workspace:** Interactive development environment for users

These user-generated applications require isolated execution environments to ensure:

- **Security:** Prevent malicious code from affecting host or other users
- **Resource Control:** Limit CPU, memory, and network usage per container
- **Data Isolation:** Separate workspace volumes for each user
- **Network Isolation:** Control outbound connections and port exposure

### Options Considered

#### 1. E2B Sandboxes (Cloud Sandboxing Service)
- **Pros:**
  - Managed infrastructure (no Docker maintenance)
  - Built-in security policies
  - Fast provisioning (< 1s startup)
  - Automatic cleanup and resource limits
  - Designed for code execution

- **Cons:**
  - External dependency (SaaS vendor lock-in)
  - Monthly costs scale with usage (~$100-500/mo for 100 users)
  - Network latency for API calls
  - Limited customization of environment
  - Data leaves self-hosted infrastructure

#### 2. Docker Containers (Self-Hosted)
- **Pros:**
  - Full control over infrastructure
  - No external dependencies or vendor lock-in
  - One-time setup cost (no recurring fees)
  - Customizable security policies (seccomp, capabilities)
  - Works offline / in air-gapped environments
  - Mature ecosystem (Docker 24+, well-documented)

- **Cons:**
  - Requires Docker daemon on host
  - Container management complexity
  - Need to handle cleanup and resource limits manually
  - Slower startup than E2B (~2-5s cold start)

#### 3. Firecracker VMs (Micro-VMs)
- **Pros:**
  - Strong isolation (hardware virtualization)
  - Fast startup (~125ms)
  - Minimal overhead (5MB RAM)

- **Cons:**
  - Linux-only (no Windows/Mac support)
  - Complex setup (KVM, cgroups, networking)
  - Requires kernel modifications
  - Steep learning curve

#### 4. gVisor (User-Space Kernel)
- **Pros:**
  - Better isolation than Docker alone
  - Compatible with Docker/Kubernetes

- **Cons:**
  - Performance overhead (syscall interception)
  - Compatibility issues with some applications
  - Additional complexity

## Decision

**We chose Docker Containers (Option 2)** with custom security hardening.

### Rationale

1. **Self-Hosted Philosophy:** Aligns with BusinessOS's core principle of data ownership
2. **Cost Efficiency:** No recurring SaaS fees for small-medium deployments
3. **Flexibility:** Full control over container configuration and security policies
4. **Maturity:** Docker is battle-tested with extensive tooling and documentation
5. **Cross-Platform:** Works on Linux, macOS, Windows (Docker Desktop)

### Implementation Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│ User Browser (Frontend)                                         │
└────────────────────────┬────────────────────────────────────────┘
                         │ WebSocket (Terminal)
                         ↓
┌─────────────────────────────────────────────────────────────────┐
│ Go Backend (internal/terminal/manager.go)                       │
│  • Handles WebSocket connections                                │
│  • Validates user authentication                                │
│  • Manages session lifecycle                                    │
└────────────────────────┬────────────────────────────────────────┘
                         │
                         ↓
┌─────────────────────────────────────────────────────────────────┐
│ ContainerManager (internal/container/container.go)              │
│  • CreateContainer(userID, sessionID, image)                    │
│  • StartContainer(containerID)                                  │
│  • StopContainer(containerID, timeout)                          │
│  • RemoveContainer(containerID, force)                          │
└────────────────────────┬────────────────────────────────────────┘
                         │ Docker API
                         ↓
┌─────────────────────────────────────────────────────────────────┐
│ Docker Daemon                                                   │
│                                                                 │
│  ┌────────────────┐  ┌────────────────┐  ┌────────────────┐  │
│  │ Container 1    │  │ Container 2    │  │ Container 3    │  │
│  │ User: Alice    │  │ User: Bob      │  │ User: Carol    │  │
│  │ Volume: vol_A  │  │ Volume: vol_B  │  │ Volume: vol_C  │  │
│  │ Image: Alpine  │  │ Image: Alpine  │  │ Image: Alpine  │  │
│  └────────────────┘  └────────────────┘  └────────────────┘  │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

## Implementation Details

### Container Security Configuration

**File:** `desktop/backend-go/internal/container/container.go`

#### 1. Resource Limits
```go
Resources: container.Resources{
    Memory:    512 * 1024 * 1024,  // 512MB RAM limit
    CPUQuota:  50000,               // 50% of one CPU core
    CPUPeriod: 100000,              // Standard period: 100ms
    PidsLimit: 100,                 // Max 100 processes
}
```

#### 2. Capability Dropping (Privilege Restriction)
```go
// Drop ALL capabilities, add only essential ones
CapDrop: []string{"ALL"},
CapAdd: []string{
    "CHOWN",   // Change file ownership in /workspace
    "FOWNER",  // Operations on files regardless of owner
}

// REMOVED (security hardening):
// - DAC_OVERRIDE (file permission bypass)
// - SETUID/SETGID (privilege escalation)
// - SYS_ADMIN (mount, unmount, system modifications)
```

#### 3. Security Options
```go
SecurityOpt: []string{
    "no-new-privileges:true",     // Prevent privilege escalation
    "seccomp=" + SeccompProfile,  // Custom syscall filter
}
```

**Seccomp Profile** (Embedded JSON):
Blocks dangerous syscalls:
- `mount`, `umount2` - Filesystem modifications
- `pivot_root`, `chroot` - Container escape attempts
- `setns` - Namespace manipulation
- `ptrace` - Process debugging (container escape vector)
- `bpf` - Berkeley Packet Filter (network manipulation)
- `kernel_module` syscalls - Load/unload kernel modules

#### 4. Read-Only Root Filesystem
```go
ReadonlyRootfs: true,
Tmpfs: map[string]string{
    "/tmp":     "rw,noexec,nosuid,size=64m",  // Temp files, no execution
    "/var/tmp": "rw,noexec,nosuid,size=32m",  // Var temp
    "/run":     "rw,noexec,nosuid,size=16m",  // Runtime files (PID, sockets)
}
```

#### 5. Network Isolation
```go
// Terminal containers: Bridge mode for npm, git, curl
NetworkMode: "bridge",
DNS: []string{"8.8.8.8", "8.8.4.4", "1.1.1.1"},
ExtraHosts: []string{"host.docker.internal:host-gateway"},

// Filesystem containers: No network access
NetworkMode: "none",  // Complete isolation
```

#### 6. Volume Isolation (User Workspaces)
```go
volumeName := fmt.Sprintf("workspace_%s", userID)

Mounts: []mount.Mount{
    {
        Type:   mount.TypeVolume,
        Source: volumeName,           // Per-user Docker volume
        Target: "/workspace",         // Mounted at /workspace in container
    },
    {
        Type:     mount.TypeBind,
        Source:   getInitScriptPath(),  // Absolute path to init script
        Target:   "/etc/businessos/init.sh",
        ReadOnly: true,
    },
}
```

**Key Insight:** Each user gets a dedicated Docker volume (`workspace_userID`), preventing cross-user file access.

### Container Types

#### 1. Terminal Containers (Interactive)
- **Purpose:** Interactive development via WebSocket terminal
- **Image:** `businessos-workspace:latest` (Alpine Linux 3.19, 271MB)
- **Resources:** 512MB RAM, 50% CPU, 100 processes
- **Network:** Bridge mode (allows npm install, git clone, curl)
- **Lifecycle:** Created on-demand, stopped after idle timeout

#### 2. Filesystem Containers (Long-Lived)
- **Purpose:** File browsing and CRUD operations
- **Image:** `businessos-workspace:latest`
- **Resources:** 128MB RAM, 10% CPU, 10 processes
- **Network:** None (complete isolation)
- **Lifecycle:** Persistent, restarted if stopped

### Session Management

**File:** `desktop/backend-go/internal/terminal/session.go`

```go
type Session struct {
    ID            string
    UserID        string
    WorkspaceID   string          // Database workspace (not Docker workspace)

    // Container isolation
    ContainerID   string          // Docker container ID
    VolumeID      string          // Docker volume ID
    ExecID        string          // Docker exec instance ID

    // Security
    ClientIP      string          // IP binding for hijacking detection
    ExpiresAt     time.Time       // Hard expiration (8 hours max)
    Status        SessionStatus   // active, idle, closed
}

// Security features:
// - IP binding prevents session hijacking
// - Hard expiration after 8 hours
// - Idle timeout after 30 minutes
```

### Critical Fixes (Resolved Jan 2026)

**Problem:** Docker terminal system failing with multiple errors

**Symptoms:**
- `No such image: businessos-workspace:latest`
- `bash: $'\r': command not found` (CRLF line endings)
- `invalid mount path: must be absolute` (relative paths)

**Resolution:**
1. Built Docker image: `docker build -t businessos-workspace:latest .`
2. Fixed line endings: `sed -i 's/\r$//' businessos_init.sh`
3. Fixed absolute paths: Modified `getInitScriptPath()` to use `filepath.Abs()`

**Status:** Fully operational (Alpine Linux 3.19, 271MB)

## Consequences

### Positive

1. **Full Control:**
   - Custom security policies tailored to BusinessOS
   - No vendor lock-in or SaaS dependencies
   - Works in air-gapped/offline environments

2. **Cost Efficient:**
   - No monthly SaaS fees
   - Scales with hardware (add nodes as needed)
   - Predictable infrastructure costs

3. **Data Sovereignty:**
   - All code execution happens on self-hosted infrastructure
   - User data never leaves controlled environment
   - Compliance-friendly (GDPR, HIPAA)

4. **Mature Ecosystem:**
   - Docker has extensive tooling (docker-compose, Portainer, etc.)
   - Well-documented API and best practices
   - Large community for troubleshooting

5. **Strong Isolation:**
   - Seccomp profiles block 90+ dangerous syscalls
   - Read-only root filesystem prevents tampering
   - Dropped capabilities eliminate privilege escalation vectors
   - Per-user volumes prevent cross-user file access

6. **Monitoring & Debugging:**
   - Full access to container logs
   - Docker stats API for resource monitoring
   - Easy to exec into containers for debugging

### Negative

1. **Operational Complexity:**
   - Requires Docker daemon on host system
   - Need to handle container cleanup (pruning stopped containers)
   - Must manage Docker image updates
   - **Mitigation:** Automated cleanup cron jobs, image versioning

2. **Cold Start Latency:**
   - Container creation: ~2-5 seconds
   - E2B is faster (~1 second or less)
   - **Mitigation:** Pre-warm containers for active users, keep filesystem containers persistent

3. **Resource Overhead:**
   - Docker daemon consumes ~200-500MB RAM
   - Each container adds ~50-100MB overhead
   - **Mitigation:** Lightweight Alpine base image (271MB), aggressive container pruning

4. **Security Maintenance:**
   - Must keep Docker updated (CVE patches)
   - Base image updates (Alpine security fixes)
   - Seccomp profile maintenance
   - **Mitigation:** Automated vulnerability scanning (Trivy), CI/CD image rebuilds

5. **Windows/Mac Limitations:**
   - Docker Desktop required (not native Linux)
   - Performance overhead on non-Linux hosts
   - **Mitigation:** Recommend Linux deployment for production

6. **Escape Risk (Theoretical):**
   - Container breakout vulnerabilities exist (rare)
   - Kernel-level exploits could affect all containers
   - **Mitigation:** Defense-in-depth (seccomp, capabilities, read-only root, AppArmor)

### Monitoring

**Key Metrics:**
- Container count per user (prevent resource abuse)
- CPU/Memory usage per container (identify runaway processes)
- Container creation/destruction rate (detect anomalies)
- Network egress per container (identify data exfiltration)

**Alerts:**
- Container resource limits exceeded
- Excessive container creation rate (> 10/min per user)
- Seccomp violations (blocked syscalls attempted)
- Unusual network activity patterns

### Files Involved

**Container Management:**
- `desktop/backend-go/internal/container/container.go` (586 lines)
- `desktop/backend-go/internal/container/manager.go`
- `desktop/backend-go/internal/container/exec.go`
- `desktop/backend-go/internal/container/filesystem.go`
- `desktop/backend-go/internal/container/monitor.go`
- `desktop/backend-go/internal/container/seccomp.go` (Seccomp profile)

**Terminal System:**
- `desktop/backend-go/internal/terminal/session.go` (215 lines)
- `desktop/backend-go/internal/terminal/manager.go`
- `desktop/backend-go/internal/terminal/osa_command.go`
- `desktop/backend-go/internal/terminal/sanitizer.go`

**Docker Image:**
- `desktop/backend-go/docker/workspace/Dockerfile` (Alpine 3.19 base)
- `desktop/backend-go/docker/workspace/businessos_init.sh` (Init script)

**Documentation:**
- `DOCKER_TERMINAL_FIX_SUMMARY.md` (Root directory - troubleshooting guide)

## Future Enhancements

1. **gVisor Integration (Optional):**
   - Add gVisor runtime for enhanced isolation
   - Trade-off: ~10-15% performance overhead
   - Benefit: Additional security layer against kernel exploits

2. **Kubernetes Support:**
   - Scale beyond single Docker host
   - Use Kubernetes Jobs/Pods for container management
   - Enable multi-node deployments

3. **Pre-Warming:**
   - Maintain pool of pre-started containers
   - Assign to users on-demand (< 500ms startup)
   - Trade-off: Higher idle resource usage

4. **Image Variants:**
   - Specialized images: Node.js, Python, Java
   - User-selectable base images
   - Pre-installed development tools

5. **GPU Support:**
   - Enable NVIDIA runtime for ML workloads
   - Allow users to run GPU-accelerated code

## Related Decisions

- **ADR-001:** Database Isolation Strategy (workspace-level row isolation)
- **ISR-4:** File Browsing Endpoint (path traversal protection)
- **CUS-94.1:** Terminal Session Authentication (fixed Jan 2026)

## References

- Docker Security Best Practices: https://docs.docker.com/engine/security/
- Seccomp Profiles: https://github.com/docker/labs/tree/master/security/seccomp
- Docker Container Isolation: https://docs.docker.com/engine/security/isolation/
- Alpine Linux Security: https://alpinelinux.org/about/
- BusinessOS Docker Fix: `DOCKER_TERMINAL_FIX_SUMMARY.md`
