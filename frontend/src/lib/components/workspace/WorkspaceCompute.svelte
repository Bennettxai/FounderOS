<script lang="ts">
  import { request } from '$lib/api/base';
  import { onMount } from 'svelte';
  import {
    Server,
    Cpu,
    MemoryStick,
    HardDrive,
    Play,
    Square,
    RotateCcw,
    Loader2,
    AlertCircle,
    PlusCircle,
    ExternalLink,
    Activity,
  } from 'lucide-svelte';

  interface Props {
    workspaceId: string;
  }

  let { workspaceId }: Props = $props();

  // --- Types ---
  interface ComputeInstance {
    id: string;
    name: string;
    status: 'running' | 'stopped' | 'starting' | 'stopping' | 'error';
    plan: string;
    cpu_cores: number;
    memory_gb: number;
    disk_gb: number;
    created_at: string;
    region?: string;
  }

  interface ComputeResponse {
    instances: ComputeInstance[];
  }

  // --- State ---
  let instances = $state<ComputeInstance[]>([]);
  let isLoading = $state(true);
  let error = $state<string | null>(null);
  let actionInProgress = $state<string | null>(null);

  onMount(async () => {
    await loadInstances();
  });

  async function loadInstances() {
    try {
      isLoading = true;
      error = null;
      const data = await request<ComputeResponse>(
        `/workspaces/${workspaceId}/compute`,
        { skipAuthRedirect: false }
      );
      instances = data?.instances ?? [];
    } catch (err) {
      // If the endpoint 404s (not yet implemented), show empty state rather than error
      const msg = err instanceof Error ? err.message : '';
      if (msg.includes('404') || msg.includes('not found')) {
        instances = [];
      } else {
        error = msg || 'Failed to load compute instances';
      }
    } finally {
      isLoading = false;
    }
  }

  async function handleAction(instanceId: string, action: 'start' | 'stop' | 'restart') {
    try {
      actionInProgress = instanceId;
      await request(`/workspaces/${workspaceId}/compute/${instanceId}/${action}`, {
        method: 'POST',
      });
      await loadInstances();
    } catch (err) {
      console.error(`Failed to ${action} instance:`, err);
    } finally {
      actionInProgress = null;
    }
  }

  function getStatusColor(status: ComputeInstance['status']): string {
    const map: Record<string, string> = {
      running: 'var(--bos-status-success, #10b981)',
      stopped: 'var(--dt3)',
      starting: 'var(--bos-status-warning, #f59e0b)',
      stopping: 'var(--bos-status-warning, #f59e0b)',
      error: 'var(--bos-status-error, #ef4444)',
    };
    return map[status] ?? 'var(--dt3)';
  }

  function getStatusLabel(status: ComputeInstance['status']): string {
    const map: Record<string, string> = {
      running: 'Running',
      stopped: 'Stopped',
      starting: 'Starting',
      stopping: 'Stopping',
      error: 'Error',
    };
    return map[status] ?? status;
  }

  function isTransitioning(status: ComputeInstance['status']): boolean {
    return status === 'starting' || status === 'stopping';
  }
</script>

<div class="wpc-container">
  <!-- Header -->
  <div class="wpc-header">
    <div>
      <h2 class="wpc-title">Compute Instances</h2>
      <p class="wpc-subtitle">Manage dedicated compute for your workspace</p>
    </div>
    <button class="wpc-btn wpc-btn--primary" disabled title="Coming soon">
      <PlusCircle class="w-4 h-4" />
      Add Instance
    </button>
  </div>

  {#if isLoading}
    <div class="wpc-loading">
      <Loader2 class="w-6 h-6 wpc-spin" />
      <span>Loading compute instances...</span>
    </div>
  {:else if error}
    <div class="wpc-error">
      <AlertCircle class="w-5 h-5" />
      <span>{error}</span>
      <button class="wpc-retry-btn" onclick={loadInstances}>Retry</button>
    </div>
  {:else if instances.length === 0}
    <!-- Empty State -->
    <div class="wpc-empty">
      <div class="wpc-empty-icon">
        <Server class="w-10 h-10" />
      </div>
      <div class="wpc-empty-title">No compute instances</div>
      <p class="wpc-empty-desc">
        Dedicated compute instances give your workspace isolated CPU and memory resources
        for running AI agents, long-running tasks, and private workloads.
      </p>
      <div class="wpc-empty-actions">
        <button
          class="wpc-btn wpc-btn--primary"
          onclick={() => window.open('/register?plan=professional', '_self')}
        >
          <ExternalLink class="w-4 h-4" />
          Upgrade to unlock Compute
        </button>
      </div>

      <!-- Feature Overview -->
      <div class="wpc-features-grid">
        <div class="wpc-feature">
          <Cpu class="w-5 h-5 wpc-feature-icon" />
          <div class="wpc-feature-label">Dedicated CPU</div>
          <div class="wpc-feature-desc">Reserved cores for your workspace</div>
        </div>
        <div class="wpc-feature">
          <MemoryStick class="w-5 h-5 wpc-feature-icon" />
          <div class="wpc-feature-label">Isolated Memory</div>
          <div class="wpc-feature-desc">No shared memory with other tenants</div>
        </div>
        <div class="wpc-feature">
          <Activity class="w-5 h-5 wpc-feature-icon" />
          <div class="wpc-feature-label">AI Agent Runtime</div>
          <div class="wpc-feature-desc">Run agents 24/7 without limits</div>
        </div>
        <div class="wpc-feature">
          <HardDrive class="w-5 h-5 wpc-feature-icon" />
          <div class="wpc-feature-label">Persistent Storage</div>
          <div class="wpc-feature-desc">Data persists across restarts</div>
        </div>
      </div>
    </div>
  {:else}
    <!-- Instances Table -->
    <div class="wpc-table-wrap">
      <table class="wpc-table">
        <thead>
          <tr>
            <th>Instance</th>
            <th>Status</th>
            <th>Plan</th>
            <th>Resources</th>
            <th>Region</th>
            <th>Created</th>
            <th class="wpc-col-actions">Actions</th>
          </tr>
        </thead>
        <tbody>
          {#each instances as inst (inst.id)}
            {@const busy = actionInProgress === inst.id || isTransitioning(inst.status)}
            <tr>
              <td>
                <div class="wpc-instance-cell">
                  <div class="wpc-instance-icon">
                    <Server class="w-4 h-4" />
                  </div>
                  <span class="wpc-instance-name">{inst.name}</span>
                </div>
              </td>
              <td>
                <span class="wpc-status-pill" style="color: {getStatusColor(inst.status)}; background: {getStatusColor(inst.status)}18">
                  {#if isTransitioning(inst.status)}
                    <Loader2 class="w-3 h-3 wpc-spin" />
                  {:else}
                    <span class="wpc-status-dot" style="background: {getStatusColor(inst.status)}"></span>
                  {/if}
                  {getStatusLabel(inst.status)}
                </span>
              </td>
              <td>
                <span class="wpc-plan-text">{inst.plan}</span>
              </td>
              <td>
                <div class="wpc-resources">
                  <span class="wpc-resource-item">
                    <Cpu class="w-3 h-3" />
                    {inst.cpu_cores} CPU
                  </span>
                  <span class="wpc-resource-item">
                    <MemoryStick class="w-3 h-3" />
                    {inst.memory_gb} GB RAM
                  </span>
                  <span class="wpc-resource-item">
                    <HardDrive class="w-3 h-3" />
                    {inst.disk_gb} GB Disk
                  </span>
                </div>
              </td>
              <td>
                <span class="wpc-region">{inst.region ?? 'us-east-1'}</span>
              </td>
              <td>
                <span class="wpc-date">
                  {new Date(inst.created_at).toLocaleDateString()}
                </span>
              </td>
              <td class="wpc-col-actions">
                <div class="wpc-actions">
                  {#if inst.status === 'stopped'}
                    <button
                      class="wpc-action-btn wpc-action-btn--start"
                      onclick={() => handleAction(inst.id, 'start')}
                      disabled={busy}
                      title="Start instance"
                      aria-label="Start instance {inst.name}"
                    >
                      {#if busy}
                        <Loader2 class="w-3.5 h-3.5 wpc-spin" />
                      {:else}
                        <Play class="w-3.5 h-3.5" />
                      {/if}
                    </button>
                  {:else if inst.status === 'running'}
                    <button
                      class="wpc-action-btn"
                      onclick={() => handleAction(inst.id, 'restart')}
                      disabled={busy}
                      title="Restart instance"
                      aria-label="Restart instance {inst.name}"
                    >
                      {#if busy}
                        <Loader2 class="w-3.5 h-3.5 wpc-spin" />
                      {:else}
                        <RotateCcw class="w-3.5 h-3.5" />
                      {/if}
                    </button>
                    <button
                      class="wpc-action-btn wpc-action-btn--stop"
                      onclick={() => handleAction(inst.id, 'stop')}
                      disabled={busy}
                      title="Stop instance"
                      aria-label="Stop instance {inst.name}"
                    >
                      {#if busy}
                        <Loader2 class="w-3.5 h-3.5 wpc-spin" />
                      {:else}
                        <Square class="w-3.5 h-3.5" />
                      {/if}
                    </button>
                  {:else}
                    <span class="wpc-no-actions">—</span>
                  {/if}
                </div>
              </td>
            </tr>
          {/each}
        </tbody>
      </table>
    </div>
  {/if}
</div>

<style>
  .wpc-container {
    padding: 2rem;
    display: flex;
    flex-direction: column;
    gap: 1.5rem;
  }

  /* Header */
  .wpc-header {
    display: flex;
    align-items: flex-start;
    justify-content: space-between;
    gap: 1rem;
  }

  .wpc-title {
    font-size: 1.125rem;
    font-weight: 600;
    color: var(--dt);
    margin: 0 0 0.375rem 0;
  }

  .wpc-subtitle {
    font-size: 0.8125rem;
    color: var(--dt3);
    margin: 0;
  }

  /* Buttons */
  .wpc-btn {
    display: inline-flex;
    align-items: center;
    gap: 0.375rem;
    padding: 0.5rem 1rem;
    font-size: 0.8125rem;
    font-weight: 500;
    border-radius: 0.375rem;
    cursor: pointer;
    transition: all 0.15s;
    border: 1px solid transparent;
    white-space: nowrap;
  }

  .wpc-btn--primary {
    background: var(--dt);
    color: var(--dbg);
    border-color: var(--dt);
  }

  .wpc-btn--primary:hover:not(:disabled) {
    opacity: 0.88;
  }

  .wpc-btn--primary:disabled {
    opacity: 0.4;
    cursor: not-allowed;
  }

  /* Loading */
  .wpc-loading {
    display: flex;
    align-items: center;
    gap: 0.75rem;
    padding: 3rem 2rem;
    justify-content: center;
    color: var(--dt3);
    font-size: 0.875rem;
  }

  .wpc-spin {
    animation: wpc-spin 0.8s linear infinite;
  }

  @keyframes wpc-spin {
    from { transform: rotate(0deg); }
    to { transform: rotate(360deg); }
  }

  /* Error */
  .wpc-error {
    display: flex;
    align-items: center;
    gap: 0.75rem;
    padding: 1rem 1.25rem;
    background: color-mix(in srgb, var(--bos-status-error, #ef4444) 8%, transparent);
    border: 1px solid color-mix(in srgb, var(--bos-status-error, #ef4444) 25%, transparent);
    border-radius: 0.5rem;
    color: var(--bos-status-error, #ef4444);
    font-size: 0.875rem;
  }

  .wpc-retry-btn {
    margin-left: auto;
    padding: 0.25rem 0.75rem;
    font-size: 0.8125rem;
    font-weight: 500;
    border: 1px solid currentColor;
    border-radius: 0.25rem;
    background: transparent;
    color: inherit;
    cursor: pointer;
    flex-shrink: 0;
  }

  /* Empty State */
  .wpc-empty {
    display: flex;
    flex-direction: column;
    align-items: center;
    text-align: center;
    padding: 3rem 1.5rem;
    gap: 1rem;
  }

  .wpc-empty-icon {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 4rem;
    height: 4rem;
    background: var(--dbg2);
    border: 1px solid var(--dbd);
    border-radius: 1rem;
    color: var(--dt3);
  }

  .wpc-empty-title {
    font-size: 1rem;
    font-weight: 600;
    color: var(--dt);
  }

  .wpc-empty-desc {
    font-size: 0.875rem;
    color: var(--dt3);
    max-width: 440px;
    line-height: 1.6;
    margin: 0;
  }

  .wpc-empty-actions {
    margin-top: 0.5rem;
  }

  .wpc-features-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(180px, 1fr));
    gap: 1rem;
    width: 100%;
    max-width: 640px;
    margin-top: 1.5rem;
  }

  .wpc-feature {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 0.5rem;
    padding: 1rem;
    background: var(--dbg2);
    border: 1px solid var(--dbd);
    border-radius: 0.625rem;
    text-align: center;
  }

  .wpc-feature-icon {
    color: var(--dt3);
  }

  .wpc-feature-label {
    font-size: 0.8125rem;
    font-weight: 600;
    color: var(--dt);
  }

  .wpc-feature-desc {
    font-size: 0.75rem;
    color: var(--dt3);
    line-height: 1.4;
  }

  /* Table */
  .wpc-table-wrap {
    overflow-x: auto;
    border: 1px solid var(--dbd);
    border-radius: 0.5rem;
  }

  .wpc-table {
    width: 100%;
    border-collapse: collapse;
  }

  .wpc-table thead {
    background: var(--dbg2);
  }

  .wpc-table th {
    padding: 0.625rem 1rem;
    text-align: left;
    font-size: 0.6875rem;
    font-weight: 600;
    color: var(--dt3);
    text-transform: uppercase;
    letter-spacing: 0.05em;
    border-bottom: 1px solid var(--dbd);
    white-space: nowrap;
  }

  .wpc-table td {
    padding: 0.875rem 1rem;
    border-bottom: 1px solid var(--dbd);
    vertical-align: middle;
  }

  .wpc-table tbody tr:last-child td {
    border-bottom: none;
  }

  .wpc-table tbody tr:hover {
    background: var(--dbg2);
  }

  .wpc-col-actions {
    width: 100px;
    text-align: center;
  }

  .wpc-instance-cell {
    display: flex;
    align-items: center;
    gap: 0.625rem;
  }

  .wpc-instance-icon {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 1.875rem;
    height: 1.875rem;
    background: var(--dbg3);
    border: 1px solid var(--dbd);
    border-radius: 0.375rem;
    color: var(--dt3);
    flex-shrink: 0;
  }

  .wpc-instance-name {
    font-size: 0.875rem;
    font-weight: 500;
    color: var(--dt);
  }

  .wpc-status-pill {
    display: inline-flex;
    align-items: center;
    gap: 0.375rem;
    padding: 0.25rem 0.625rem;
    border-radius: 9999px;
    font-size: 0.75rem;
    font-weight: 600;
    white-space: nowrap;
  }

  .wpc-status-dot {
    width: 6px;
    height: 6px;
    border-radius: 50%;
    flex-shrink: 0;
  }

  .wpc-plan-text {
    font-size: 0.8125rem;
    color: var(--dt2);
    text-transform: capitalize;
  }

  .wpc-resources {
    display: flex;
    flex-direction: column;
    gap: 0.25rem;
  }

  .wpc-resource-item {
    display: flex;
    align-items: center;
    gap: 0.375rem;
    font-size: 0.75rem;
    color: var(--dt3);
  }

  .wpc-region {
    font-size: 0.8125rem;
    color: var(--dt3);
    font-family: ui-monospace, 'Cascadia Code', monospace;
  }

  .wpc-date {
    font-size: 0.8125rem;
    color: var(--dt3);
  }

  .wpc-actions {
    display: flex;
    align-items: center;
    gap: 0.375rem;
    justify-content: center;
  }

  .wpc-action-btn {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 1.875rem;
    height: 1.875rem;
    background: transparent;
    border: 1px solid var(--dbd);
    border-radius: 0.375rem;
    color: var(--dt3);
    cursor: pointer;
    transition: all 0.15s;
  }

  .wpc-action-btn:hover:not(:disabled) {
    background: var(--dbg2);
    color: var(--dt);
    border-color: var(--dbd2);
  }

  .wpc-action-btn:disabled {
    opacity: 0.4;
    cursor: not-allowed;
  }

  .wpc-action-btn--start:hover:not(:disabled) {
    color: var(--bos-status-success, #10b981);
    border-color: var(--bos-status-success, #10b981);
  }

  .wpc-action-btn--stop:hover:not(:disabled) {
    color: var(--bos-status-error, #ef4444);
    border-color: var(--bos-status-error, #ef4444);
  }

  .wpc-no-actions {
    color: var(--dbd);
    font-size: 0.875rem;
  }
</style>
