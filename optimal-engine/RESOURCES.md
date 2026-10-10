# Optimal Engine Resources

Start here:

- `CLAUDE.md` - Claude Code boot rules and the short agent memory loop.
- `AGENTS.md` - full agent boot, runtime boundaries, store model, and verification rules.
- `BOOT.md` - quick human and agent setup path.
- `OPTIONS.md` - supported CLI, environment, graph backend, and embedding options.
- `SYSTEM.md` - system map, layer ownership, and BusinessOS boundary.
- `docs/guides/getting-started.md` - local setup.
- `docs/guides/installation-and-deployment.md` - install and deploy path.
- `docs/guides/store-and-layer-reality.md` - stores and layer ownership.
- `docs/guides/agent-cli-sop.md` - agent CLI operating procedure.
- `docs/architecture/STORAGE-AND-PROJECTION-MAP.md` - canonical storage/projection map.
- `docs/reference/backend-readiness.md` - backend readiness checks.
- `skills/optimal-engine/README.md` - engine skill docs.

Core local commands:

```bash
bin/optimal install
bin/optimal bootstrap
bin/optimal dev
bin/optimal doctor
bin/optimal reality-check
```

Agent loop commands:

```bash
bin/optimal boot
bin/optimal find "query" --workspace default:my-workspace
bin/optimal capture "raw signal" --workspace default:my-workspace
bin/optimal aware "important correction" --workspace default:my-workspace
bin/optimal close "what changed and how verified"
```

Runtime data lives in `.optimal/`.
Do not commit `.optimal/index.db`, cache files, connector keys, RocksDB stores, workspace imports, or private user data.
