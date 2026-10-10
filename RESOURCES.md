# BusinessOS Resources

Read these files before changing BusinessOS behavior:

- `AGENTS.md` - agent operating rules for this repo.
- `CLAUDE.md` - Claude-specific project instructions when present.
- `OPINIONS.md` - product and architecture opinions for BusinessOS.
- `desktop/` - Electron desktop shell, local terminal, IPC, desktop windows, and local engine wiring.
- `frontend/` - Svelte app modules, API client, and web UI.
- `desktop/src/main/ipc/index.ts` - local IPC bridge, including engine memory calls.
- `frontend/src/lib/api/base.ts` - browser API wrapper and memory mirroring behavior.
- `frontend/src/lib/optimal-engine/connect.ts` - Optimal Engine connection and sync helpers.

Public/customer Optimal Engine source-checkout CLI:

```bash
optimal-engine/bin/optimal
```

Use it for bundled/customer engine checks and public setup docs:

```bash
optimal-engine/bin/optimal doctor
optimal-engine/bin/optimal boot
optimal-engine/bin/optimal find "BusinessOS memory persistence" --workspace default:businessos
optimal-engine/bin/optimal aware "Important BusinessOS decision..." --workspace default:businessos
optimal-engine/bin/optimal close "What changed and what remains..." --workspace default:businessos
```

Roberto's private wrapper is only for Roberto-specific private context:

```bash
ROBERTO_OPTIMAL_ENGINE_CLI="${ROBERTO_OPTIMAL_ENGINE_CLI:?set private engine CLI path}"
$ROBERTO_OPTIMAL_ENGINE_CLI boot
$ROBERTO_OPTIMAL_ENGINE_CLI find "BusinessOS memory persistence" businessos
$ROBERTO_OPTIMAL_ENGINE_CLI aware "Important BusinessOS decision..." businessos
$ROBERTO_OPTIMAL_ENGINE_CLI close "What changed and what remains..." businessos
```

Do not commit runtime engine stores, user secrets, private workspace data, downloaded user data, local session state, or private connector credentials.
