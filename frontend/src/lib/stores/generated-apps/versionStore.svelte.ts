/**
 * Version Store — Generated Apps [id] page
 * Manages version history state: loaded versions, modal/panel open states,
 * the selected preview/restore version, diff targets, and async operations.
 *
 * Uses the Svelte 5 singleton factory pattern with $state runes.
 */

import type { Version, BackendVersionInfo } from "./types";
import { toVersionSummary } from "$lib/components/versioning";
import {
  listAppVersions,
  createAppSnapshot,
  restoreAppVersion,
} from "$lib/api/versions";
import { mapBackendVersionsList } from "$lib/api/versions/mappers";

function createVersionStore() {
  // ── Version data ──────────────────────────────────────────────────────────
  let versions = $state<Version[]>([]);
  let rawBackendVersions = $state<BackendVersionInfo[]>([]);
  let versionsLoading = $state(false);

  // ── Panel / modal open state ──────────────────────────────────────────────
  let timelinePanelOpen = $state(false);
  let saveModalOpen = $state(false);
  let previewModalOpen = $state(false);
  let restoreDialogOpen = $state(false);
  let diffModalOpen = $state(false);

  // ── Selected version state ────────────────────────────────────────────────
  let previewVersion = $state<Version | null>(null);
  let restoreVersion = $state<Version | null>(null);
  let diffFromVersion = $state<BackendVersionInfo | null>(null);
  let diffToVersion = $state<BackendVersionInfo | null>(null);

  // ── Async operation flags ─────────────────────────────────────────────────
  let isSavingVersion = $state(false);
  let isRestoring = $state(false);

  return {
    // Version data
    get versions() {
      return versions;
    },
    set versions(v: Version[]) {
      versions = v;
    },

    get rawBackendVersions() {
      return rawBackendVersions;
    },
    set rawBackendVersions(v: BackendVersionInfo[]) {
      rawBackendVersions = v;
    },

    get versionsLoading() {
      return versionsLoading;
    },
    set versionsLoading(v: boolean) {
      versionsLoading = v;
    },

    // Derived
    get currentVersion() {
      return versions.find((v) => v.isCurrent)?.versionNumber ?? 1;
    },

    get versionSummaries() {
      return versions.map(toVersionSummary);
    },

    // Panel / modal open state
    get timelinePanelOpen() {
      return timelinePanelOpen;
    },
    set timelinePanelOpen(v: boolean) {
      timelinePanelOpen = v;
    },

    get saveModalOpen() {
      return saveModalOpen;
    },
    set saveModalOpen(v: boolean) {
      saveModalOpen = v;
    },

    get previewModalOpen() {
      return previewModalOpen;
    },
    set previewModalOpen(v: boolean) {
      previewModalOpen = v;
    },

    get restoreDialogOpen() {
      return restoreDialogOpen;
    },
    set restoreDialogOpen(v: boolean) {
      restoreDialogOpen = v;
    },

    get diffModalOpen() {
      return diffModalOpen;
    },
    set diffModalOpen(v: boolean) {
      diffModalOpen = v;
    },

    // Selected versions
    get previewVersion() {
      return previewVersion;
    },
    set previewVersion(v: Version | null) {
      previewVersion = v;
    },

    get restoreVersion() {
      return restoreVersion;
    },
    set restoreVersion(v: Version | null) {
      restoreVersion = v;
    },

    get diffFromVersion() {
      return diffFromVersion;
    },
    set diffFromVersion(v: BackendVersionInfo | null) {
      diffFromVersion = v;
    },

    get diffToVersion() {
      return diffToVersion;
    },
    set diffToVersion(v: BackendVersionInfo | null) {
      diffToVersion = v;
    },

    // Async flags
    get isSavingVersion() {
      return isSavingVersion;
    },
    set isSavingVersion(v: boolean) {
      isSavingVersion = v;
    },

    get isRestoring() {
      return isRestoring;
    },
    set isRestoring(v: boolean) {
      isRestoring = v;
    },

    // ── Methods ─────────────────────────────────────────────────────────

    /** Fetch and map all versions for the given app. */
    async loadVersions(workspaceId: string, appId: string) {
      versionsLoading = true;
      try {
        rawBackendVersions = await listAppVersions(workspaceId, appId);
        versions = mapBackendVersionsList(rawBackendVersions);
      } catch (err) {
        console.error("Failed to load versions:", err);
        versions = [];
        rawBackendVersions = [];
      } finally {
        versionsLoading = false;
      }
    },

    /** Open the preview modal for the given version summary id. */
    selectVersionSummary(summaryId: string) {
      const version = versions.find((v) => v.id === summaryId);
      if (version) {
        previewVersion = version;
        previewModalOpen = true;
      }
    },

    /** Open the preview modal for a full Version object. */
    openPreview(version: Version) {
      previewVersion = version;
      previewModalOpen = true;
    },

    /** Open the restore confirmation dialog for the given version. */
    openRestore(version: Version) {
      restoreVersion = version;
      restoreDialogOpen = true;
    },

    /**
     * Execute a restore operation.
     * `onSuccess` is called after versions and files are refreshed.
     */
    async confirmRestore(
      workspaceId: string,
      appId: string,
      onSuccess: () => Promise<void>,
    ) {
      if (!restoreVersion) return;
      isRestoring = true;
      try {
        const backendVer =
          restoreVersion.backendVersion ??
          rawBackendVersions.find((v) => v.id === restoreVersion!.id)
            ?.version_number;
        if (!backendVer) throw new Error("Version not found");

        await restoreAppVersion(workspaceId, appId, backendVer);
        restoreDialogOpen = false;
        previewModalOpen = false;
        await onSuccess();
      } catch (err) {
        console.error("Failed to restore version:", err);
      } finally {
        isRestoring = false;
      }
    },

    /**
     * Create a snapshot and reload versions.
     * `onSuccess` is called after the new version list is fetched.
     */
    async saveVersion(
      workspaceId: string,
      appId: string,
      label: string | undefined,
      onSuccess: () => Promise<void>,
    ) {
      isSavingVersion = true;
      try {
        await createAppSnapshot(workspaceId, appId, label);
        saveModalOpen = false;
        await onSuccess();
      } catch (err) {
        console.error("Failed to save version:", err);
      } finally {
        isSavingVersion = false;
      }
    },

    /** Set up the diff modal to compare oldest vs newest. */
    openDiff() {
      if (rawBackendVersions.length < 2) return;
      diffFromVersion = rawBackendVersions[rawBackendVersions.length - 1]; // oldest
      diffToVersion = rawBackendVersions[0]; // newest
      diffModalOpen = true;
    },
  };
}

export const versionStore = createVersionStore();
