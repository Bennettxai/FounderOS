/**
 * Sandbox Store — Generated Apps [id] page
 * Manages sandbox execution state: error messages, polling interval,
 * stop-confirmation dialog, and deployment progress.
 *
 * Uses the Svelte 5 singleton factory pattern with $state runes.
 */

import {
  deploySandbox,
  stopSandbox as stopSandboxAPI,
  getSandboxInfo,
} from "$lib/api/sandbox";
import { generatedAppsStore } from "$lib/stores/generatedAppsStore";
import type { GeneratedApp } from "./types";

function createSandboxStore() {
  let sandboxError = $state<string | null>(null);
  let sandboxPolling = $state<ReturnType<typeof setInterval> | null>(null);
  let showStopConfirm = $state(false);
  let sandboxDeployProgress = $state(0);

  // ── Private: polling implementation ──────────────────────────────────────

  function stopPolling() {
    if (sandboxPolling) {
      clearInterval(sandboxPolling);
      sandboxPolling = null;
    }
  }

  function startPolling(
    appId: string,
    onAppRefresh: (app: GeneratedApp | null) => void,
  ) {
    stopPolling();
    sandboxPolling = setInterval(async () => {
      try {
        const info = await getSandboxInfo(appId);
        if (info.status === "deploying" || info.status === "pending") {
          sandboxDeployProgress = Math.min(sandboxDeployProgress + 12, 90);
        }
        if (
          info.status === "running" ||
          info.status === "stopped" ||
          info.status === "failed"
        ) {
          sandboxDeployProgress = info.status === "running" ? 100 : 0;
          stopPolling();
          if (info.status === "failed") {
            sandboxError = "Sandbox deployment failed. Try again.";
          }
        }
        const refreshed = await generatedAppsStore.getAppById(appId);
        onAppRefresh(refreshed);
      } catch {
        // Silently retry on next interval
      }
    }, 3000);
  }

  return {
    get sandboxError() {
      return sandboxError;
    },
    set sandboxError(v: string | null) {
      sandboxError = v;
    },

    get showStopConfirm() {
      return showStopConfirm;
    },
    set showStopConfirm(v: boolean) {
      showStopConfirm = v;
    },

    get sandboxDeployProgress() {
      return sandboxDeployProgress;
    },
    set sandboxDeployProgress(v: number) {
      sandboxDeployProgress = v;
    },

    // ── Methods ─────────────────────────────────────────────────────────

    /**
     * Start the sandbox and begin polling for status.
     * `onAppRefresh` is called each time fresh app data arrives.
     */
    async start(
      app: GeneratedApp,
      onAppRefresh: (app: GeneratedApp | null) => void,
    ) {
      sandboxError = null;
      sandboxDeployProgress = 0;
      try {
        await deploySandbox(app.id, app.app_name);
        startPolling(app.id, onAppRefresh);
        const refreshed = await generatedAppsStore.getAppById(app.id);
        onAppRefresh(refreshed);
      } catch (err) {
        sandboxError =
          err instanceof Error ? err.message : "Failed to start sandbox";
      }
    },

    /**
     * Stop the sandbox and clean up polling.
     * `onAppRefresh` is called once with the updated app record.
     */
    async stop(
      app: GeneratedApp,
      onAppRefresh: (app: GeneratedApp | null) => void,
    ) {
      showStopConfirm = false;
      sandboxError = null;
      try {
        await stopSandboxAPI(app.id);
        stopPolling();
        sandboxDeployProgress = 0;
        const refreshed = await generatedAppsStore.getAppById(app.id);
        onAppRefresh(refreshed);
      } catch (err) {
        sandboxError =
          err instanceof Error ? err.message : "Failed to stop sandbox";
      }
    },

    /** Begin polling for an app that is already deploying/pending on mount. */
    beginPollingIfDeploying(
      app: GeneratedApp,
      onAppRefresh: (app: GeneratedApp | null) => void,
    ) {
      if (
        app.sandbox?.status === "deploying" ||
        app.sandbox?.status === "pending"
      ) {
        startPolling(app.id, onAppRefresh);
      }
    },

    /** Dismiss the sandbox error and retry start. */
    async retry(
      app: GeneratedApp,
      onAppRefresh: (app: GeneratedApp | null) => void,
    ) {
      sandboxError = null;
      await this.start(app, onAppRefresh);
    },

    /** Stop polling — call from onDestroy. */
    destroy() {
      stopPolling();
    },

    /** Clear the error banner. */
    clearError() {
      sandboxError = null;
    },
  };
}

export const sandboxStore = createSandboxStore();
