<script lang="ts">
  import { onMount } from 'svelte';
  import { CheckCircle, AlertCircle, CloudOff, Loader2, Eye, EyeOff, ExternalLink } from 'lucide-svelte';
  import { getApiBaseUrl, getCSRFToken } from '$lib/api/base';

  // ── Types ────────────────────────────────────────────────────────────────────

  type ConnectionStatus = 'idle' | 'connected' | 'invalid' | 'checking';

  interface CloudAccount {
    email: string;
    plan: string;
    credits_total: number;
    credits_used: number;
  }

  // ── State ────────────────────────────────────────────────────────────────────

  const LS_KEY = 'businessos_api_key';

  let apiKey = $state('');
  let showKey = $state(false);
  let status = $state<ConnectionStatus>('idle');
  let account = $state<CloudAccount | null>(null);
  let errorMessage = $state('');

  // ── Lifecycle ────────────────────────────────────────────────────────────────

  onMount(() => {
    const saved = localStorage.getItem(LS_KEY);
    if (saved) {
      apiKey = saved;
      // Silently validate the saved key on mount
      validateKey(saved);
    }
  });

  // ── Derived ──────────────────────────────────────────────────────────────────

  const isConnected = $derived(status === 'connected');
  const isChecking = $derived(status === 'checking');
  const hasKey = $derived(apiKey.trim().length > 0);
  const creditsRemaining = $derived(
    account ? account.credits_total - account.credits_used : 0
  );

  // ── Actions ──────────────────────────────────────────────────────────────────

  async function validateKey(key: string): Promise<void> {
    status = 'checking';
    errorMessage = '';

    try {
      const headers: Record<string, string> = {
        'X-BusinessOS-API-Key': key,
      };
      const csrf = getCSRFToken();
      if (csrf) headers['X-CSRF-Token'] = csrf;

      const res = await fetch(`${getApiBaseUrl()}/computer`, {
        method: 'GET',
        headers,
        credentials: 'include',
        signal: AbortSignal.timeout(10_000),
      });

      if (res.ok) {
        status = 'connected';
        // Try to load subscription data for account details
        await loadAccountInfo(key);
      } else {
        status = 'invalid';
        errorMessage = res.status === 401 || res.status === 403
          ? 'API key is invalid or has expired.'
          : `Connection failed (HTTP ${res.status}).`;
        account = null;
      }
    } catch {
      status = 'invalid';
      errorMessage = 'Could not reach BusinessOS Cloud. Check your network.';
      account = null;
    }
  }

  async function loadAccountInfo(key: string): Promise<void> {
    try {
      const headers: Record<string, string> = {
        'X-BusinessOS-API-Key': key,
      };
      const csrf = getCSRFToken();
      if (csrf) headers['X-CSRF-Token'] = csrf;

      const res = await fetch(`${getApiBaseUrl()}/billing/subscription`, {
        method: 'GET',
        headers,
        credentials: 'include',
        signal: AbortSignal.timeout(8_000),
      });

      if (res.ok) {
        const data = await res.json();
        // Map to our CloudAccount shape — billing endpoint returns Subscription type
        account = {
          email: data.email ?? '',
          plan: data.plan ?? 'Pro',
          credits_total: data.credits_total ?? 0,
          credits_used: data.credits_used ?? 0,
        };
      }
    } catch {
      // Non-fatal — account details are optional decoration
      account = null;
    }
  }

  async function handleConnect(): Promise<void> {
    const key = apiKey.trim();
    if (!key) return;

    localStorage.setItem(LS_KEY, key);
    await validateKey(key);
  }

  function handleDisconnect(): void {
    localStorage.removeItem(LS_KEY);
    apiKey = '';
    status = 'idle';
    account = null;
    errorMessage = '';
  }

  function handleKeydown(e: KeyboardEvent): void {
    if (e.key === 'Enter') handleConnect();
  }
</script>

<div class="space-y-4">
  <!-- Section header -->
  <div class="card">
    <h2 class="text-xs font-semibold uppercase tracking-wide cc-label mb-1">BusinessOS Cloud</h2>
    <p class="text-sm cc-muted">
      Connect to BusinessOS Cloud to unlock Cloud Computer, data sync, team features, and AI credits.
    </p>

    <!-- Status banner -->
    <div class="mt-4">
      {#if status === 'checking'}
        <div class="flex items-center gap-2 rounded-lg cc-card p-3">
          <Loader2 class="h-4 w-4 animate-spin cc-icon-muted" aria-hidden="true" />
          <span class="text-sm cc-muted">Verifying connection...</span>
        </div>

      {:else if isConnected}
        <div class="flex items-start gap-3 rounded-lg cc-card-success p-3">
          <span class="mt-0.5 h-2 w-2 flex-shrink-0 rounded-full bg-green-500" aria-hidden="true"></span>
          <div class="flex-1 min-w-0">
            <p class="text-sm font-medium cc-success-text">Connected to BusinessOS Cloud</p>
            {#if account?.email}
              <p class="text-xs cc-muted mt-0.5 truncate">{account.email}</p>
            {/if}
          </div>
          <CheckCircle class="h-4 w-4 flex-shrink-0 text-green-500" aria-hidden="true" />
        </div>

      {:else if status === 'invalid'}
        <div class="flex items-center gap-2 rounded-lg cc-card-error p-3">
          <span class="h-2 w-2 flex-shrink-0 rounded-full bg-red-500" aria-hidden="true"></span>
          <span class="text-sm cc-error-text">Invalid key</span>
          <AlertCircle class="h-4 w-4 flex-shrink-0 text-red-500 ml-auto" aria-hidden="true" />
        </div>

      {:else}
        <div class="flex items-center gap-2 rounded-lg cc-card p-3">
          <span class="h-2 w-2 flex-shrink-0 rounded-full cc-dot-idle" aria-hidden="true"></span>
          <span class="text-sm cc-muted">Not connected</span>
          <CloudOff class="h-4 w-4 flex-shrink-0 cc-icon-muted ml-auto" aria-hidden="true" />
        </div>
      {/if}
    </div>

    <!-- Error detail -->
    {#if errorMessage}
      <p class="mt-2 text-xs cc-error-text" role="alert">{errorMessage}</p>
    {/if}
  </div>

  <!-- Connected state: account details -->
  {#if isConnected}
    <div class="card space-y-3">
      <h2 class="text-xs font-semibold uppercase tracking-wide cc-label">Account</h2>

      <div class="space-y-2">
        {#if account?.plan}
          <div class="flex items-center justify-between">
            <span class="text-sm cc-label">Plan</span>
            <span class="text-sm font-medium cc-title">{account.plan}</span>
          </div>
        {/if}

        {#if account && account.credits_total > 0}
          <div class="flex items-center justify-between">
            <span class="text-sm cc-label">AI Credits remaining</span>
            <span class="text-sm font-medium cc-title">{creditsRemaining.toLocaleString()} / {account.credits_total.toLocaleString()}</span>
          </div>
          <!-- Credits bar -->
          <div class="h-1.5 w-full rounded-full cc-progress-track overflow-hidden" aria-hidden="true">
            <div
              class="h-full rounded-full cc-progress-fill transition-all"
              style="width: {Math.max(0, Math.min(100, (creditsRemaining / account.credits_total) * 100))}%"
            ></div>
          </div>
        {/if}
      </div>

      <div class="flex items-center gap-3 pt-1">
        <a
          href="/computer"
          class="flex items-center gap-1.5 text-xs cc-link"
          aria-label="Manage subscription"
        >
          Manage Subscription
          <ExternalLink class="h-3 w-3" aria-hidden="true" />
        </a>
        <span class="cc-separator">|</span>
        <button
          onclick={handleDisconnect}
          class="text-xs cc-link-danger"
          aria-label="Disconnect from BusinessOS Cloud"
        >
          Disconnect
        </button>
      </div>
    </div>

  <!-- Disconnected state: feature list + input -->
  {:else}
    <div class="card space-y-4">
      <div>
        <h2 class="text-xs font-semibold uppercase tracking-wide cc-label mb-2">Connect to unlock</h2>
        <ul class="space-y-1.5">
          {#each ['Cloud Computer (VM)', 'Data Sync', 'Team Collaboration', 'AI Credits'] as feature}
            <li class="flex items-center gap-2 text-sm cc-muted">
              <span class="h-1.5 w-1.5 rounded-full cc-dot-feature flex-shrink-0" aria-hidden="true"></span>
              {feature}
            </li>
          {/each}
        </ul>
      </div>

      <!-- API key input -->
      <div class="space-y-1.5">
        <label for="bos-api-key" class="block text-xs font-medium cc-label">
          API Key
        </label>
        <div class="flex gap-2">
          <div class="relative flex-1">
            <input
              id="bos-api-key"
              type={showKey ? 'text' : 'password'}
              bind:value={apiKey}
              onkeydown={handleKeydown}
              placeholder="bos_live_..."
              autocomplete="off"
              spellcheck="false"
              class="cc-input w-full pr-9"
              aria-label="BusinessOS Cloud API key"
              aria-describedby="bos-api-key-hint"
            />
            <button
              type="button"
              onclick={() => (showKey = !showKey)}
              class="absolute inset-y-0 right-0 flex items-center px-2.5 cc-icon-muted hover:cc-icon"
              aria-label={showKey ? 'Hide API key' : 'Show API key'}
              tabindex="-1"
            >
              {#if showKey}
                <EyeOff class="h-3.5 w-3.5" aria-hidden="true" />
              {:else}
                <Eye class="h-3.5 w-3.5" aria-hidden="true" />
              {/if}
            </button>
          </div>
          <button
            onclick={handleConnect}
            disabled={!hasKey || isChecking}
            class="btn-pill btn-pill-primary btn-pill-sm disabled:opacity-50 disabled:cursor-not-allowed flex-shrink-0"
            aria-label="Connect to BusinessOS Cloud"
          >
            {#if isChecking}
              <Loader2 class="h-3.5 w-3.5 animate-spin" aria-hidden="true" />
            {:else}
              Connect
            {/if}
          </button>
        </div>
        <p id="bos-api-key-hint" class="text-xs cc-muted">
          Get your API key at
          <a
            href="https://businessos.com/settings/api-keys"
            target="_blank"
            rel="noopener noreferrer"
            class="cc-link"
            aria-label="Get your API key at businessos.com (opens in new tab)"
          >businessos.com</a>.
          Stored locally in your browser.
        </p>
      </div>
    </div>
  {/if}
</div>

<style>
  /* ── Color tokens ─────────────────────────────────────────────── */
  .cc-title  { color: var(--dt); }
  .cc-label  { color: var(--dt2); }
  .cc-muted  { color: var(--dt3); }
  .cc-icon-muted { color: var(--dt4); }

  /* ── Status cards ─────────────────────────────────────────────── */
  .cc-card {
    border: 1px solid var(--bos-settings-card-border, var(--dbd));
    background: var(--bos-settings-card-bg, var(--dbg2));
    border-radius: 8px;
  }
  .cc-card-success {
    border: 1px solid var(--bos-status-success-border, #bbf7d0);
    background: var(--bos-status-success-bg, #f0fdf4);
    border-radius: 8px;
  }
  .cc-card-error {
    border: 1px solid var(--bos-status-error-border, #fecaca);
    background: var(--bos-status-error-bg, #fef2f2);
    border-radius: 8px;
  }
  :global(.dark) .cc-card-success {
    border-color: #166534;
    background: rgba(20, 83, 45, 0.25);
  }
  :global(.dark) .cc-card-error {
    border-color: #991b1b;
    background: rgba(127, 29, 29, 0.25);
  }

  /* ── Semantic text ────────────────────────────────────────────── */
  .cc-success-text { color: var(--bos-status-success, #15803d); }
  .cc-error-text   { color: var(--bos-status-error,   #dc2626); }
  :global(.dark) .cc-success-text { color: #86efac; }
  :global(.dark) .cc-error-text   { color: #fca5a5; }

  /* ── Dots ─────────────────────────────────────────────────────── */
  .cc-dot-idle    { background: var(--dt4, #aaa); }
  .cc-dot-feature { background: var(--dt3, #888); }

  /* ── Links ────────────────────────────────────────────────────── */
  .cc-link {
    color: var(--bos-accent, #2563eb);
    transition: opacity 0.15s;
  }
  .cc-link:hover { opacity: 0.75; }
  .cc-link-danger {
    color: var(--bos-status-error, #dc2626);
    transition: opacity 0.15s;
  }
  .cc-link-danger:hover { opacity: 0.75; }
  .cc-separator { color: var(--dbd); }

  /* ── Input ────────────────────────────────────────────────────── */
  .cc-input {
    border: 1px solid var(--dbd);
    background: var(--dbg);
    color: var(--dt);
    border-radius: 6px;
    padding: 0.375rem 0.625rem;
    font-size: 0.8125rem;
    outline: none;
    transition: border-color 0.15s;
  }
  .cc-input:focus {
    border-color: var(--dt3);
    box-shadow: 0 0 0 2px color-mix(in srgb, var(--dt3) 20%, transparent);
  }
  .cc-input::placeholder { color: var(--dt4); }

  /* ── Progress bar ─────────────────────────────────────────────── */
  .cc-progress-track { background: var(--dbg3, #e5e7eb); }
  .cc-progress-fill  { background: var(--bos-accent, #2563eb); }
</style>
