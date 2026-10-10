# CRIT-6: Host Environment Secrets Leaked into Local PTY Sessions

**Severity:** CRITICAL | **Exploitability:** 9/10 | **CVSS:** 9.8

## Location
`desktop/backend-go/internal/terminal/pty.go:181`

## Description
```go
func buildEnvArray(envMap map[string]string) []string {
    env := os.Environ()  // Full host process environment
    for key, value := range envMap {
        env = append(env, fmt.Sprintf("%s=%s", key, value))
    }
    return env
}
```

Any authenticated user with a local PTY session (development mode, or when Docker is unavailable) receives the full Go server process environment including `DATABASE_URL`, `ANTHROPIC_API_KEY`, `SECRET_KEY`, `TOKEN_ENCRYPTION_KEY`, and all other secrets.

## PoC
In any local terminal session:
```bash
env | grep -E "KEY|SECRET|TOKEN|PASSWORD|DATABASE"
```
Returns all server secrets.

## Fix
```go
func buildEnvArray(envMap map[string]string) []string {
    allowed := map[string]bool{"TERM": true, "LANG": true, "COLORTERM": true, "HOME": true, "PATH": true}
    env := []string{}
    for _, e := range os.Environ() {
        key := strings.SplitN(e, "=", 2)[0]
        if allowed[key] {
            env = append(env, e)
        }
    }
    for key, value := range envMap {
        env = append(env, fmt.Sprintf("%s=%s", key, value))
    }
    return env
}
```
