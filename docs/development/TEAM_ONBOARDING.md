# BusinessOS Team Onboarding

This guide covers team access after the source application passes local verification.

Start with [Developer Quick Start](DEVELOPER_QUICKSTART.md).

## 1. Run the Verified Local Profile

```bash
make onboard
```

Do not copy another developer's `.env`, database directory, or `.run` directory.
Each developer starts with isolated local infrastructure and receives shared credentials separately.

Confirm:

```bash
make dev-local-verify
```

Do not continue to shared workspace setup while verification is failing.

## 2. Create a Local Account

The Electron window opens automatically.
Create a local account or use an account already provisioned for the selected environment.

Local account creation does not require Google OAuth.

## 3. Configure Google Access When Required

Ask Roberto for the approved OAuth client credentials through a secure channel.
Add them to `desktop/backend-go/.env`.

```text
GOOGLE_CLIENT_ID=
GOOGLE_CLIENT_SECRET=
```

Google requires exact callback URLs.

Authentication:

```text
http://localhost:8801/api/auth/google/callback
```

Shared Google integration:

```text
http://localhost:8801/api/integrations/google/callback
```

Google Calendar:

```text
http://localhost:8801/api/integrations/google_calendar/callback
```

If local ports change, the OAuth client must authorize the matching exact callbacks.

## 4. Join the Team Workspace

Workspace membership comes from the configured BusinessOS environment.

1. Sign in with the invited account.
2. Open the workspace switcher.
3. Select the assigned workspace.
4. Confirm the organization, workspace name, role, and modules.
5. Refresh once and confirm the same workspace remains selected.

Do not create replacement workspaces when an expected shared workspace is absent.
Report the account email, organization, and expected workspace to the workspace owner.

## 5. Connect Optimal Engine

Each BusinessOS workspace maps to an Optimal Engine workspace.

1. Open **Settings**.
2. Select **Optimal Engine**.
3. Choose the built-in engine or enter the approved external engine URL.
4. Enter the API key only when the engine requires authentication.
5. Select the exact engine workspace slug.
6. Test and save the connection.

The built-in engine starts with clean local data.
Roberto's private OptimalOS data is not included in the repository or desktop package.

## 6. Configure Optional Providers

Provider credentials are optional until the assigned work requires them.

Examples:

- Anthropic or another model provider.
- Instagram or Meta.
- WhatsApp MCP.
- Google Workspace.
- MIOSA cloud sync.

Use variable names already present in `desktop/backend-go/.env.example`.
Do not invent aliases or commit credentials.

## 7. Acceptance Check

Before beginning assigned work:

```bash
make dev-local-verify
make test-dev-local-launcher
```

Then confirm in the UI:

- The assigned workspace loads.
- The correct modules appear for that workspace.
- Refresh preserves the active workspace.
- Optimal Engine reports the intended engine and workspace.
- No module shows an unexplained `HTTP 500` response.

Capture a screenshot and the relevant `.run` log when a check fails.

## Support Information

Provide these details with a setup issue:

```bash
git rev-parse --short HEAD
node --version
pnpm --version
go version
psql --version
redis-server --version
make dev-local-status
```

Never send environment files or full logs without checking them for credentials.
