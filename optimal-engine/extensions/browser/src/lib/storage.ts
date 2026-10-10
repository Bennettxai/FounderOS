import type { ExtensionSettings } from "./types";
import { DEFAULT_SETTINGS } from "./types";

const SETTINGS_KEY = "oe_settings";

/**
 * Read extension settings from chrome.storage.local.
 * Falls back to DEFAULT_SETTINGS for any missing keys.
 */
export async function getSettings(): Promise<ExtensionSettings> {
  return new Promise((resolve) => {
    chrome.storage.local.get(SETTINGS_KEY, (result) => {
      const stored = (result[SETTINGS_KEY] ?? {}) as Partial<ExtensionSettings>;
      resolve({ ...DEFAULT_SETTINGS, ...stored });
    });
  });
}

/**
 * Persist a partial settings update into chrome.storage.local.
 */
export async function saveSettings(
  patch: Partial<ExtensionSettings>,
): Promise<void> {
  const current = await getSettings();
  const next: ExtensionSettings = { ...current, ...patch };
  return new Promise((resolve, reject) => {
    chrome.storage.local.set({ [SETTINGS_KEY]: next }, () => {
      if (chrome.runtime.lastError) {
        reject(new Error(chrome.runtime.lastError.message));
      } else {
        resolve();
      }
    });
  });
}

/**
 * Clear all extension settings (reset to defaults).
 */
export async function clearSettings(): Promise<void> {
  return new Promise((resolve, reject) => {
    chrome.storage.local.remove(SETTINGS_KEY, () => {
      if (chrome.runtime.lastError) {
        reject(new Error(chrome.runtime.lastError.message));
      } else {
        resolve();
      }
    });
  });
}
