# Release Process

> **Status:** ACTIVE
> **Owner:** Roberto
> **Priority:** P0

---

## Versioning

We follow [Semantic Versioning](https://semver.org/):

```
MAJOR.MINOR.PATCH

MAJOR: Breaking changes (API, database schema, config format)
MINOR: New features, backward-compatible
PATCH: Bug fixes, backward-compatible
```

**Current version:** `0.1.0` (pre-production, initial development)

Pre-1.0 versions indicate the API is not yet stable. Breaking changes may occur in MINOR bumps during this phase. Version 1.0.0 will be tagged at the first production-ready release.

## Version Bump Files

When bumping the version, update these files:

| File | Field | Example |
|------|-------|---------|
| `desktop/package.json` | `"version"` | `"0.2.0"` |
| `CHANGELOG.md` | New section header | `## [0.2.0] - 2026-03-01` |

**Note:** The Go backend does not currently embed a version string. When it does, add `desktop/backend-go/cmd/server/main.go` to this list.

## Release Types

| Type | When | Branch | Approval |
|------|------|--------|----------|
| **Regular** | Bi-weekly (every other Tuesday) | `main` | 1 reviewer |
| **Hotfix** | Critical bug in production | `hotfix/*` → `main` | Fast-track, 1 reviewer |
| **Major** | Breaking changes | `release/*` → `main` | All 3 operators |

## Release Schedule

```
Regular releases: Bi-weekly (every other Tuesday)
Hotfixes:         As needed (no schedule, deploy immediately after review)
```

Adjust cadence based on sprint velocity. If a sprint has no user-facing changes, skip the release.

## Regular Release Process

### 1. Prepare

```bash
# Ensure main is clean and all tests pass
git checkout main
git pull origin main
cd desktop/backend-go && go build ./cmd/server && go test ./...
cd frontend && npm run build && npm run check
```

### 2. Version Bump

```bash
# Update version in:
# - desktop/package.json → "version": "X.Y.Z"
# - CHANGELOG.md → add new section
```

### 3. Changelog

Update `CHANGELOG.md` using [Keep a Changelog](https://keepachangelog.com/) format:

```markdown
## [X.Y.Z] - YYYY-MM-DD

### Added
- Feature description (#PR)

### Fixed
- Bug fix description (#PR)

### Changed
- Change description (#PR)

### Removed
- Removed feature description (#PR)
```

Changelog entries should reference PR numbers. Write entries as you merge PRs, not at release time.

### 4. Tag and Push

```bash
git add desktop/package.json CHANGELOG.md
git commit -m "chore(release): bump version to vX.Y.Z"
git tag -a vX.Y.Z -m "Release vX.Y.Z"
git push origin main --tags
```

### 5. Deploy

```
Web:     Tag push triggers CI/CD → Cloud Run auto-deploy
Desktop: Tag push triggers CI/CD → Build artifacts → GitHub Release (draft)
```

### 6. Verify

- [ ] Backend health check passes
- [ ] Frontend loads
- [ ] Auth flow works
- [ ] Generate an app (E2E smoke test)
- [ ] Check Sentry for new errors

### 7. Publish

- Edit GitHub Release (add changelog notes, unmark draft)
- Notify team in chat channel
- For major releases: update any public-facing documentation

## Hotfix Process

```
1. Create branch: git checkout -b hotfix/description main
2. Fix the bug (minimal change)
3. Test: go test ./... && npm run build
4. PR → fast-track review → merge to main
5. Tag: git tag -a vX.Y.(Z+1) -m "Hotfix: description"
6. Deploy immediately
7. Backport to any active release branches if applicable
```

## Release Checklist

- [ ] All tests pass (`go test ./...`, `npm run build && npm run check`)
- [ ] No P0/P1 bugs open
- [ ] Changelog updated
- [ ] Version bumped in all relevant files
- [ ] Tagged in git
- [ ] Deployed to staging, smoke tested
- [ ] Deployed to production, smoke tested
- [ ] Desktop builds created (if desktop changes)
- [ ] GitHub Release published
- [ ] Team notified

---

**Last Updated:** 2026-02-23
