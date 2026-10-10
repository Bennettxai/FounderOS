/**
 * App Store — Generated Apps [id] page
 * Manages the core app data: the loaded GeneratedApp record, loading state,
 * and top-level error message.
 *
 * Uses the Svelte 5 singleton factory pattern with $state runes.
 */

import type { GeneratedApp } from "./types";

function createAppStore() {
  let app = $state<GeneratedApp | null>(null);
  let loading = $state(true);
  let error = $state<string | null>(null);
  let previewExpanded = $state(false);

  return {
    get app() {
      return app;
    },
    set app(v: GeneratedApp | null) {
      app = v;
    },

    get loading() {
      return loading;
    },
    set loading(v: boolean) {
      loading = v;
    },

    get error() {
      return error;
    },
    set error(v: string | null) {
      error = v;
    },

    get previewExpanded() {
      return previewExpanded;
    },
    set previewExpanded(v: boolean) {
      previewExpanded = v;
    },

    /** Toggle the expanded-preview state. */
    togglePreviewExpanded() {
      previewExpanded = !previewExpanded;
    },

    /** Clear the error banner. */
    clearError() {
      error = null;
    },
  };
}

export const appStore = createAppStore();
