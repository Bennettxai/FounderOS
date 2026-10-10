import { getPreferenceValues } from "@raycast/api";
import type { Preferences } from "./types";

/**
 * Returns typed preference values, stripping a trailing slash from engineUrl
 * so callers never need to think about it.
 */
export function getPreferences(): Preferences {
  const raw = getPreferenceValues<Preferences>();
  return {
    ...raw,
    engineUrl: raw.engineUrl.replace(/\/$/, ""),
  };
}
