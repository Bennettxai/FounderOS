import { request } from "../base";
import type { SignalHealthResponse } from "./types";

export type {
  SignalHealthResponse,
  SignalEnvelope,
  SignalGenre,
  OsaMode,
} from "./types";
export { GENRE_LABELS, GENRE_DESCRIPTIONS, MODE_LABELS } from "./types";

/**
 * Get signal theory system health status.
 * GET /api/v1/signal/health
 */
export async function getSignalHealth(): Promise<SignalHealthResponse> {
  return request<SignalHealthResponse>("/signal/health");
}
