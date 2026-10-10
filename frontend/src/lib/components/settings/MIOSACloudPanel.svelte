<script lang="ts">
  import { onMount } from 'svelte';
  import { Cloud, CloudOff, CheckCircle, AlertCircle, Loader2, RefreshCw, ExternalLink, KeyRound, Server } from 'lucide-svelte';
  import { currentWorkspace } from '$lib/stores/workspaces';
  import {
    getMIOSAStatus,
    pingMIOSACloud,
    syncToMIOSACloud,
    type MIOSAConnectionStatus,
    type SyncResult,
  } from '$lib/api/miosa';

  // ---------------------------------------------------------------------------
  // State
  // ---------------------------------------------------------------------------

  let status = $state<MIOSAConnectionStatus | null>(null);
  let isLoadingStatus = $state(true);
  let isSyncing = $state(false);
  let isPinging = $state(false);
  let lastSyncResult = $state<SyncResult | null>(null);
  let statusError = $state<string | null>(null);
  let syncError = $state<string | null>(null);

  // ---------------------------------------------------------------------------
  // Lifecycle
  // ---------------------------------------------------------------------------

  onMount(async () => {
    await loadStatus();
  });

  // ---------------------------------------------------------------------------
  // Actions
  // ---------------------------------------------------------------------------

  async function loadStatus() {
    isLoadingStatus = true;
    statusError = null;
    try {
      status = await getMIOSAStatus();
    } catch (err) {
      statusError = err instanceof Error ? err.message : 'Failed to load connection status';
    } finally {
      isLoadingStatus = false;
    }
  }

  async function handlePingCloud() {
    isPinging = true;
    syncError = null;
    try {
      const result = await pingMIOSACloud();
      if (!result.connected) {
        syncError = result.error ?? 'API key validation failed. Check your MIOSA_API_KEY.';
      }
      // Reload status to reflect updated connection state
      await loadStatus();
    } catch (err) {
      syncError = err instanceof Error ? err.message : 'Ping failed';
    } finally {
      isPinging = false;
    }
  }

  async function handleSync() {
    if (!$currentWorkspace?.id) {
      syncError = 'No workspace selected';
      return;
    }
    isSyncing = true;
    syncError = null;
    lastSyncResult = null;

    try {
      lastSyncResult = await syncToMIOSACloud($currentWorkspace.id);
      if (!lastSyncResult.success) {
        syncError = lastSyncResult.error ?? 'Sync failed';
      }
    } catch (err) {
      syncError = err instanceof Error ? err.message : 'Sync failed';
    } finally {
      isSyncing = false;
    }
  }

  // ---------------------------------------------------------------------------
  // Derived
  // ---------------------------------------------------------------------------

  const isConnected = $derived(status?.connected === true);
  const isCloud = $derived(status?.mode === 'cloud');
  const apiKeySet = $derived(status?.api_key_set === true);
  const capacityProvider = $derived(status?.capacity_provider ?? (apiKeySet ? 'user' : 'local'));

  function formatDate(iso?: string): string {
    if (!iso) return 'Never';
    return new Date(iso).toLocaleString();
  }

  function formatCapacityProvider(provider: string): string {
    if (provider === 'businessos') return 'BusinessOS MIOSA tenant';
    if (provider === 'user') return 'User MIOSA account';
    return 'Local CLI/environment';
  }
</script>

<!-- Panel header -->
<div class="space-y-6">
  <div>
    <h3 class="text-lg font-semibold st-title">MIOSA Cloud</h3>
    <p class="mt-1 text-sm st-muted">
      Optionally sync your workspace configuration (settings, agents, app definitions,
      templates) to MIOSA Cloud. Raw business data stays local.
    </p>
  </div>

  <!-- Status banner -->
  {#if isLoadingStatus}
    <div class="flex items-center gap-2 rounded-lg st-cloud-card p-4">
      <Loader2 class="h-4 w-4 animate-spin st-icon" aria-hidden="true" />
      <span class="text-sm st-muted">Loading connection status...</span>
    </div>
  {:else if statusError}
    <div class="flex items-center gap-2 rounded-lg border border-red-200 bg-red-50 p-4 dark:border-red-800 dark:bg-red-900/20">
      <AlertCircle class="h-4 w-4 flex-shrink-0 text-red-500" aria-hidden="true" />
      <span class="text-sm text-red-700 dark:text-red-400">{statusError}</span>
    </div>
  {:else if status}
    <!-- Connected state -->
    {#if isConnected}
      <div class="flex items-start gap-3 rounded-lg border border-green-200 bg-green-50 p-4 dark:border-green-800 dark:bg-green-900/20">
        <CheckCircle class="mt-0.5 h-4 w-4 flex-shrink-0 text-green-600 dark:text-green-400" aria-hidden="true" />
        <div class="flex-1">
          <p class="text-sm font-medium text-green-800 dark:text-green-300">Connected to MIOSA Cloud</p>
          <p class="mt-0.5 text-xs text-green-600 dark:text-green-500">
            Last synced: {formatDate(status.last_sync)}
          </p>
        </div>
        <Cloud class="h-4 w-4 text-green-500" aria-hidden="true" />
      </div>

    <!-- API key set but not yet validated / not in cloud mode -->
    {:else if apiKeySet && !isCloud}
      <div class="flex items-start gap-3 rounded-lg border border-amber-200 bg-amber-50 p-4 dark:border-amber-800 dark:bg-amber-900/20">
        <AlertCircle class="mt-0.5 h-4 w-4 flex-shrink-0 text-amber-600" aria-hidden="true" />
        <div class="flex-1">
          <p class="text-sm font-medium text-amber-800 dark:text-amber-300">API key found but cloud mode is not active</p>
          <p class="mt-0.5 text-xs text-amber-600 dark:text-amber-500">
            Set <code class="rounded bg-amber-100 px-1 dark:bg-amber-900">OSA_MODE=cloud</code> in your .env to enable cloud sync.
          </p>
        </div>
      </div>

    <!-- No connection -->
    {:else}
      <div class="flex items-start gap-3 rounded-lg st-cloud-card p-4">
        <CloudOff class="mt-0.5 h-4 w-4 flex-shrink-0 st-icon" aria-hidden="true" />
        <div>
          <p class="text-sm font-medium st-label">Not connected to MIOSA Cloud</p>
          <p class="mt-0.5 text-xs st-muted">
            Running in local mode. Your data stays on your machine.
          </p>
        </div>
      </div>
    {/if}
  {/if}

  <!-- Configuration section -->
  <div class="space-y-4 rounded-lg st-cloud-border p-4">
    <h4 class="text-sm font-medium st-title">Configuration</h4>

    <div class="space-y-1">
      <label for="miosa-api-key" class="block text-xs font-medium st-label">
        MIOSA API Key
      </label>
      <p class="text-xs st-muted">
        Set via <code class="rounded st-cloud-code px-1">MIOSA_API_KEY</code> in your .env file,
        or in the Electron app via the system keychain. The key is never sent to the browser.
      </p>
      <div class="mt-2 flex items-center gap-2">
        <div
          class="flex-1 rounded-md st-cloud-card px-3 py-2 text-sm st-muted"
          aria-label="API key status"
        >
          {#if apiKeySet}
            <span class="flex items-center gap-1.5">
              <CheckCircle class="h-3.5 w-3.5 text-green-500" aria-hidden="true" />
              API key configured in backend
            </span>
          {:else}
            Not configured — add MIOSA_API_KEY to your .env
          {/if}
        </div>
        <a
          href="https://app.miosa.ai/settings/api-keys"
          target="_blank"
          rel="noopener noreferrer"
          class="flex items-center gap-1 text-xs text-blue-600 hover:text-blue-700 dark:text-blue-400"
          aria-label="Get your MIOSA API key (opens in new tab)"
        >
          Get key
          <ExternalLink class="h-3 w-3" aria-hidden="true" />
        </a>
      </div>
    </div>

    <div class="space-y-1">
      <label class="block text-xs font-medium st-label">
        OSA Mode
      </label>
      <p class="text-xs st-muted">
        Set <code class="rounded st-cloud-code px-1">OSA_MODE=cloud</code> to route OSA agent traffic
        through MIOSA Cloud. Default is <code class="rounded st-cloud-code px-1">local</code>
        (localhost:8089, SQLite, no cloud).
      </p>
      <div
        class="mt-1 inline-flex items-center rounded-md px-2.5 py-1 text-xs font-medium
          {isCloud
            ? 'bg-blue-100 text-blue-800 dark:bg-blue-900/30 dark:text-blue-300'
            : 'st-cloud-mode-local'}"
        aria-label="Current OSA mode"
      >
        Current mode: {status?.mode ?? 'local'}
      </div>
    </div>
  </div>

  <div class="space-y-4 rounded-lg st-cloud-border p-4">
    <div class="flex items-start justify-between gap-3">
      <div>
        <h4 class="text-sm font-medium st-title">Sandbox and computer capacity</h4>
        <p class="mt-1 text-xs st-muted">
          BusinessOS paid workspaces can use the BusinessOS MIOSA tenant.
          Local or self-hosted users can connect their own MIOSA account through the backend, keychain, or shell environment.
        </p>
      </div>
      <Server class="h-4 w-4 st-icon" aria-hidden="true" />
    </div>

    <div class="grid gap-3 md:grid-cols-2">
      <div class="rounded-md st-cloud-card p-3">
        <div class="flex items-center gap-2">
          <KeyRound class="h-3.5 w-3.5 st-icon" aria-hidden="true" />
          <p class="text-xs font-medium st-label">Current authority</p>
        </div>
        <p class="mt-1 text-sm st-title">{formatCapacityProvider(capacityProvider)}</p>
        <p class="mt-1 text-xs st-muted">
          Sandbox tabs pass this as terminal context so CLI tools and agents can choose the right MIOSA account.
        </p>
      </div>

      <div class="rounded-md st-cloud-card p-3">
        <p class="text-xs font-medium st-label">Workspace quota</p>
        <p class="mt-1 text-sm st-title">
          {status?.usage?.active_sandboxes ?? 0}/{status?.workspace_quota?.max_sandboxes ?? 'local'} sandboxes
        </p>
        <p class="mt-1 text-xs st-muted">
          Computers: {status?.usage?.active_computers ?? 0}/{status?.workspace_quota?.max_computers ?? 'local'}
          - Desktops: {status?.usage?.active_desktops ?? 0}/{status?.workspace_quota?.max_desktops ?? 'local'}
        </p>
      </div>
    </div>
  </div>

  <!-- Actions -->
  <div class="space-y-3">
    <!-- Validate API Key -->
    <div class="flex items-center justify-between">
      <div>
        <p class="text-sm font-medium st-title">Validate connection</p>
        <p class="text-xs st-muted">
          Confirm your API key is valid by pinging MIOSA Cloud.
        </p>
      </div>
      <button
        onclick={handlePingCloud}
        disabled={isPinging || !apiKeySet}
        class="flex items-center gap-2 btn-pill btn-pill-secondary disabled:opacity-50"
        aria-label="Validate MIOSA API key"
      >
        {#if isPinging}
          <Loader2 class="h-4 w-4 animate-spin" aria-hidden="true" />
          Validating...
        {:else}
          <RefreshCw class="h-4 w-4" aria-hidden="true" />
          Validate key
        {/if}
      </button>
    </div>

    <!-- Sync to MIOSA Cloud -->
    <div class="flex items-center justify-between">
      <div>
        <p class="text-sm font-medium st-title">Publish to MIOSA Cloud</p>
        <p class="text-xs st-muted">
          Syncs workspace config, agents, app definitions, and templates.
          No business data is included.
        </p>
      </div>
      <button
        onclick={handleSync}
        disabled={isSyncing || !isConnected}
        class="btn-pill btn-pill-primary btn-pill-sm flex items-center gap-2"
        aria-label="Publish workspace configuration to MIOSA Cloud"
      >
        {#if isSyncing}
          <Loader2 class="h-4 w-4 animate-spin" aria-hidden="true" />
          Syncing...
        {:else}
          <Cloud class="h-4 w-4" aria-hidden="true" />
          Publish to MIOSA Cloud
        {/if}
      </button>
    </div>
  </div>

  <!-- Feedback messages -->
  {#if syncError}
    <div
      class="flex items-center gap-2 rounded-lg border border-red-200 bg-red-50 p-3 dark:border-red-800 dark:bg-red-900/20"
      role="alert"
    >
      <AlertCircle class="h-4 w-4 flex-shrink-0 text-red-500" aria-hidden="true" />
      <span class="text-sm text-red-700 dark:text-red-400">{syncError}</span>
    </div>
  {/if}

  {#if lastSyncResult?.success}
    <div
      class="flex items-center gap-2 rounded-lg border border-green-200 bg-green-50 p-3 dark:border-green-800 dark:bg-green-900/20"
      role="status"
    >
      <CheckCircle class="h-4 w-4 flex-shrink-0 text-green-500" aria-hidden="true" />
      <span class="text-sm text-green-700 dark:text-green-400">
        Synced successfully at {formatDate(lastSyncResult.synced_at)}
        {#if lastSyncResult.manifest_id}
          &nbsp;(manifest {lastSyncResult.manifest_id.slice(0, 8)})
        {/if}
      </span>
    </div>
  {/if}

  <!-- Informational footer -->
  <div class="rounded-lg st-cloud-footer p-3">
    <p class="text-xs st-muted">
      <strong class="st-label">What syncs:</strong>
      workspace settings, agent configurations, custom app definitions, templates.
    </p>
    <p class="mt-1 text-xs st-muted">
      <strong class="st-label">What stays local:</strong>
      tasks, projects, contacts, deals, conversations, emails, files, embeddings.
    </p>
  </div>
</div>

<style>
  :global(.st-title) { color: var(--dt, var(--bos-text-primary, #111)); }
  :global(.st-label) { color: var(--dt2, var(--bos-text-secondary, #555)); }
  :global(.st-muted) { color: var(--dt3, var(--bos-text-tertiary, #888)); }
  :global(.st-icon) { color: var(--dt4, #bbb); }
  :global(.st-cloud-card) {
    border: 1px solid var(--bos-settings-card-border);
    background: var(--bos-settings-card-bg);
    border-radius: var(--bos-settings-card-radius);
    color: var(--dt3, var(--bos-text-tertiary, #888));
  }
  :global(.st-cloud-border) {
    border: 1px solid var(--bos-settings-card-border);
    border-radius: var(--bos-settings-card-radius);
  }
  :global(.st-cloud-code) {
    background: var(--dbg3, #eee);
  }
  :global(.st-cloud-mode-local) {
    background: var(--dbg3, #eee);
    color: var(--dt2, var(--bos-text-secondary, #555));
  }
  :global(.st-cloud-footer) {
    background: var(--bos-settings-card-bg);
    border: 1px solid var(--bos-settings-card-border);
    border-radius: var(--bos-settings-card-radius);
  }
</style>
