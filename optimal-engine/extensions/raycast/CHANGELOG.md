# Optimal Engine Raycast Extension — Changelog

## [0.1.0] — 2026-04-28

### Added
- **Search Memory** command: instant search with 200 ms debounce, detail panel, copy/open actions
- **Add Memory** command: form with content, audience, static flag, citation URI; auto-closes on success
- **Ask Engine** command: natural-language Q&A, Markdown detail view, sources panel, follow-up flow
- `lib/client.ts`: zero-dependency engine HTTP wrapper (native fetch)
- `lib/preferences.ts`: typed Raycast preference accessor
- `lib/types.ts`: branded ID types and all API response shapes
- `hooks/use-workspaces.ts`: SWR-style cached workspace list hook
