# BusinessOS And Optimal Engine Workspace Routing

## Invariant

One BusinessOS workspace maps to exactly one workspace inside its selected Optimal Engine.
The BusinessOS workspace UUID controls authorization and relational application data.
The engine workspace slug controls search, memory, context, and local knowledge files.
These identifiers are related explicitly through `workspaces.settings.optimal_engine` and must never be inferred from the currently selected UI workspace at ingestion time.

## Engine choices

Desktop users can use the Optimal Engine bundled with BusinessOS or another engine they run locally or host themselves.
Both choices use the same connection contract: `enabled`, `base_url`, optional `api_key`, and `workspace` slug.
The desktop bridge reaches local engines from the user's machine, so a cloud BusinessOS backend does not need network access to `localhost`.

The bundled engine may choose a free port when `4200` is unavailable.
BusinessOS reads the actual runtime URL from Electron before probing well-known local addresses.

## New workspace provisioning

Creating a BusinessOS workspace in the desktop app performs these steps:

1. Create the relational BusinessOS workspace and owner membership.
2. Detect the running local engine.
3. List engine workspaces and reuse an existing exact slug match.
4. Create the matching engine workspace only when the slug is absent.
5. Save the BusinessOS-to-engine connection in the backend.
6. Cache the same connection in Electron for local writes.
7. Switch the UI to the newly created BusinessOS workspace.

Engine provisioning is best effort so web and cloud workspace creation still succeeds when no local engine is available.
The workspace remains usable and can be connected later in Settings.

## Duplicate prevention

BusinessOS always lists engine workspaces before creating one.
An exact normalized slug match is reused.
Workspace detection does not create BusinessOS records automatically unless the user chooses the explicit create action.
Communication routing only accepts existing BusinessOS workspaces with active membership.

## Failure behavior

An unconfigured BusinessOS workspace remains usable for relational modules but has no engine search, memory, or context.
An unassigned communication source remains readable but is not indexed.
A stopped local engine does not redirect writes to another workspace.
An inaccessible workspace cannot be selected as a communication destination.
