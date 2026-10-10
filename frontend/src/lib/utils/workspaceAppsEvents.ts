import { browser } from "$app/environment";

const CHANNEL = "businessos:workspace-apps-updated";
const STORAGE_KEY = "businessos_workspace_apps_updated_at";

export function notifyWorkspaceAppsUpdated() {
  if (!browser) return;

  window.dispatchEvent(new CustomEvent(CHANNEL));
  postWorkspaceAppsRefresh();

  try {
    const bc = new BroadcastChannel(CHANNEL);
    bc.postMessage({ type: CHANNEL, at: Date.now() });
    bc.close();
  } catch {
    /* BroadcastChannel can be unavailable in older embedded contexts. */
  }

  try {
    localStorage.setItem(STORAGE_KEY, String(Date.now()));
  } catch {
    /* localStorage can be unavailable in restricted contexts. */
  }
}

function postWorkspaceAppsRefresh() {
  try {
    const target = window.parent && window.parent !== window ? window.parent : window;
    target.postMessage(
      { type: "businessos:workspace-apps-refresh" },
      window.location.origin,
    );
  } catch {
    /* Parent messaging can be unavailable outside desktop windows. */
  }
}

export function onWorkspaceAppsUpdated(callback: () => void): () => void {
  if (!browser) return () => {};

  let bc: BroadcastChannel | null = null;
  const onEvent = () => callback();
  const onStorage = (event: StorageEvent) => {
    if (event.key === STORAGE_KEY) callback();
  };

  window.addEventListener(CHANNEL, onEvent);
  window.addEventListener("storage", onStorage);

  try {
    bc = new BroadcastChannel(CHANNEL);
    bc.onmessage = () => callback();
  } catch {
    bc = null;
  }

  return () => {
    window.removeEventListener(CHANNEL, onEvent);
    window.removeEventListener("storage", onStorage);
    bc?.close();
  };
}
