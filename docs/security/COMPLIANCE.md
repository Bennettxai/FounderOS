# BusinessOS Data Handling & Compliance

This is a factual statement of how BusinessOS handles your data.
It describes the current architecture, not aspirational policy.
It makes no claims of formal certification (SOC 2, ISO 27001, HIPAA, etc.).

## Local-first by default

BusinessOS is a desktop application. By default, your data lives on your own machine:

- **Structured data** is stored in a local SQLite database on your computer.
- **Knowledge and documents** (workspace markdown files, notes) are plain files
  on your local disk, indexed by a local engine running on your machine.
- **Nothing leaves your machine unless you explicitly turn on cloud sync.**

If you never enable cloud sync, no workspace content is transmitted to or stored
on our servers.

## Cloud sync is opt-in

Cloud sync is a feature you choose to turn on, per workspace. It exists so that:

- Teammates without their own local engine can view a workspace's knowledge.
- Teammates with an engine can pull a shared source-of-truth copy into their own
  local engine (additive - a pull adds/updates docs, it never deletes yours).

When you enable sync for a workspace, the markdown documents you choose to sync
are copied to a cloud database (`knowledge_documents`). This copy is:

- **Encrypted at rest** in managed Cloud SQL (PostgreSQL).
- **Hosted in the `us-central1` region.**
- Tagged with the workspace slug, the document path, and the email of the user
  who performed the sync (for attribution).

A sync fully replaces that workspace's prior cloud copy - it does not accumulate
history or previous versions on the server.

## What is NOT collected

- No telemetry, analytics, or behavioral tracking of how you use the app.
- No content from workspaces you have not chosen to sync.
- No secrets, API keys, tokens, or passwords are written to application logs.
- No personally identifiable information beyond the account email needed to
  authenticate you and attribute a sync.

## Deleting your cloud data (right to be forgotten)

You can delete a workspace's cloud copy at any time. This removes every synced
row for that workspace from the cloud database and resets its storage usage to
zero.

- **In the app:** open the workspace's Knowledge module (or Settings) and use the
  "Delete cloud copy" action.
- **Via API:** `DELETE /api/knowledge/cloud?workspace=<slug>` (authenticated).
  The response reports how many documents were deleted: `{ "deleted": <count> }`.

Deleting the cloud copy does not touch your local files - your machine remains
the source of truth. It only removes the copy previously synced to the cloud.

## Data residency

- Cloud-synced data is stored in managed Cloud SQL in the `us-central1` region.
- Local data never leaves the region of your own machine.

## Logging

- Application logs record operational events (route registration, errors,
  request outcomes).
- Logs do not include secrets, credentials, tokens, or synced document bodies.
- Authentication uses the account email only; passwords and tokens are never logged.

## Questions

For data-handling questions or a deletion request that the in-app action does not
cover, contact the workspace owner who administers the cloud sync for your
workspace. The owner's machine is the only one that holds the cloud database
credentials required to push or purge synced data.
