/**
 * Minimal type shim for the ported SignalBadge + SignalDetailPanel
 * components. BusinessOS used a 5-mode cognitive-state atom
 * (BUILD / ASSIST / ANALYZE / EXECUTE / MAINTAIN); we keep the
 * same literal type so the components compile unchanged, and use
 * them here as presentation-only labels.
 */
export type OsaMode = "BUILD" | "ASSIST" | "ANALYZE" | "EXECUTE" | "MAINTAIN";

export interface OsaModeInfo {
  mode: OsaMode;
  description: string;
}

export const OSA_MODES: OsaModeInfo[] = [
  { mode: "BUILD", description: "Creating / drafting" },
  { mode: "ASSIST", description: "Helping / answering" },
  { mode: "ANALYZE", description: "Inspecting / reasoning" },
  { mode: "EXECUTE", description: "Running / acting" },
  { mode: "MAINTAIN", description: "Monitoring / upkeep" },
];
