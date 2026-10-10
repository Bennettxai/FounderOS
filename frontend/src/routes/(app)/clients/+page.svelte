<script lang="ts">
  import { onMount } from "svelte";
  import { goto } from "$app/navigation";
  import { Plus, RefreshCw, Users } from "lucide-svelte";
  import type { ClientStatus, ClientType, CreateClientData } from "$lib/api";
  import {
    AddClientModal,
    ClientCardView,
    ClientKanbanView,
    ClientTableView,
    ClientViewSwitcher,
  } from "$lib/components/clients";
  import { clients } from "$lib/stores/clients";

  let addClientOpen = $state(false);
  let searchQuery = $state("");
  let statusFilter = $state<ClientStatus | null>(null);
  let typeFilter = $state<ClientType | null>(null);

  const visibleClients = $derived(
    $clients.clients.filter((client) => {
      const query = searchQuery.trim().toLowerCase();
      const matchesQuery = !query || [client.name, client.email, client.phone]
        .some((value) => value?.toLowerCase().includes(query));
      return matchesQuery
        && (!statusFilter || client.status === statusFilter)
        && (!typeFilter || client.type === typeFilter);
    }),
  );
  const activeCount = $derived($clients.clients.filter((client) => client.status === "active").length);
  const pipelineValue = $derived($clients.clients.reduce((total, client) => total + client.active_deals_value, 0));

  onMount(() => {
    clients.loadClients();
  });

  function openClient(id: string) {
    goto(`/clients/${id}`);
  }

  async function createClient(data: CreateClientData) {
    await clients.createClient(data);
  }

  async function changeStatus(id: string, status: ClientStatus) {
    await clients.updateClientStatus(id, status);
  }
</script>

<svelte:head><title>Clients | BusinessOS</title></svelte:head>

<div class="clients-page">
  <header class="page-header">
    <div>
      <h1>Clients</h1>
      <p>Workspace-scoped accounts, contacts, interactions, and delivery history.</p>
    </div>
    <div class="header-actions">
      <button class="icon-button" type="button" aria-label="Refresh clients" title="Refresh clients" onclick={() => clients.loadClients()}>
        <RefreshCw size={16} class={$clients.loading ? "spinning" : ""} />
      </button>
      <button class="primary-button" type="button" onclick={() => addClientOpen = true}>
        <Plus size={16} />
        Add client
      </button>
    </div>
  </header>

  <section class="metrics" aria-label="Client summary">
    <div><strong>{$clients.clients.length}</strong><span>Total clients</span></div>
    <div><strong>{activeCount}</strong><span>Active</span></div>
    <div><strong>{pipelineValue.toLocaleString("en-US", { style: "currency", currency: "USD", maximumFractionDigits: 0 })}</strong><span>Open deal value</span></div>
  </section>

  <ClientViewSwitcher
    view={$clients.viewMode}
    {searchQuery}
    {statusFilter}
    {typeFilter}
    onViewChange={(view) => clients.setViewMode(view)}
    onSearchChange={(query) => searchQuery = query}
    onStatusChange={(status) => statusFilter = status}
    onTypeChange={(type) => typeFilter = type}
  />

  {#if $clients.error}
    <div class="state state-error" role="alert">
      <span>{$clients.error}</span>
      <button type="button" onclick={() => clients.loadClients()}>Retry</button>
    </div>
  {:else if $clients.loading && !$clients.clients.length}
    <div class="state">Loading clients...</div>
  {:else if !visibleClients.length}
    <div class="state empty-state">
      <Users size={30} />
      <strong>No clients found</strong>
      <span>{searchQuery || statusFilter || typeFilter ? "Adjust the current filters." : "Add the first client for this workspace."}</span>
    </div>
  {:else if $clients.viewMode === "cards"}
    <ClientCardView clients={visibleClients} onClientClick={openClient} />
  {:else if $clients.viewMode === "kanban"}
    <ClientKanbanView clients={visibleClients} onClientClick={openClient} onStatusChange={changeStatus} />
  {:else}
    <ClientTableView clients={visibleClients} onClientClick={openClient} onStatusChange={changeStatus} />
  {/if}
</div>

<AddClientModal bind:open={addClientOpen} onCreate={createClient} />

<style>
  .clients-page { min-height: 100%; background: var(--dbg, #fff); color: var(--dt, #111); }
  .page-header { min-height: 88px; padding: 20px 24px; display: flex; align-items: center; justify-content: space-between; gap: 20px; border-bottom: 1px solid var(--dbd, #e5e5e5); }
  h1 { margin: 0; font-size: 22px; font-weight: 700; letter-spacing: 0; }
  p { margin: 5px 0 0; color: var(--dt3, #777); font-size: 13px; }
  .header-actions { display: flex; align-items: center; gap: 8px; }
  button { font: inherit; }
  .icon-button, .primary-button { height: 36px; border: 1px solid var(--dbd, #ddd); display: inline-flex; align-items: center; justify-content: center; gap: 7px; cursor: pointer; }
  .icon-button { width: 36px; background: var(--dbg, #fff); color: var(--dt2, #555); }
  .primary-button { padding: 0 14px; background: var(--dt, #111); color: var(--dbg, #fff); border-color: var(--dt, #111); font-weight: 650; }
  .metrics { display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); border-bottom: 1px solid var(--dbd, #e5e5e5); }
  .metrics div { padding: 14px 24px; display: flex; align-items: baseline; gap: 9px; border-right: 1px solid var(--dbd, #e5e5e5); }
  .metrics div:last-child { border-right: 0; }
  .metrics strong { font-size: 17px; }
  .metrics span { color: var(--dt3, #777); font-size: 12px; }
  .state { min-height: 260px; display: flex; align-items: center; justify-content: center; color: var(--dt3, #777); font-size: 13px; }
  .state-error { gap: 12px; color: var(--bos-status-error, #b91c1c); }
  .state-error button { border: 1px solid currentColor; background: transparent; color: inherit; padding: 6px 10px; cursor: pointer; }
  .empty-state { flex-direction: column; gap: 8px; }
  .empty-state strong { color: var(--dt, #111); font-size: 14px; }
  .spinning { animation: spin .8s linear infinite; }
  @keyframes spin { to { transform: rotate(360deg); } }
  @media (max-width: 720px) {
    .page-header { align-items: flex-start; padding: 16px; }
    .page-header p { max-width: 320px; }
    .metrics { grid-template-columns: 1fr; }
    .metrics div { border-right: 0; border-bottom: 1px solid var(--dbd, #e5e5e5); padding: 11px 16px; }
  }
</style>
