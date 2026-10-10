<script lang="ts">
  import { onMount } from "svelte";
  import { goto } from "$app/navigation";
  import { Building2, RefreshCw } from "lucide-svelte";
  import { crm, formatCurrency } from "$lib/stores/crm";
  import type { Pipeline } from "$lib/api/crm";

  const openDeals = $derived($crm.deals.filter((deal) => deal.status !== "won" && deal.status !== "lost"));
  const openValue = $derived(openDeals.reduce((sum, deal) => sum + (deal.amount ?? 0), 0));
  const weightedValue = $derived(openDeals.reduce((sum, deal) => sum + ((deal.amount ?? 0) * (deal.probability ?? 0) / 100), 0));

  onMount(async () => {
    await Promise.all([crm.loadPipelines(), crm.loadCompanies()]);
  });

  function selectPipeline(pipeline: Pipeline) {
    crm.selectPipeline(pipeline);
  }

  function dealsForStage(stageId: string) {
    return $crm.deals.filter((deal) => deal.stage_id === stageId);
  }
</script>

<svelte:head><title>CRM | BusinessOS</title></svelte:head>

<div class="crm-page">
  <header class="page-header">
    <div>
      <h1>CRM</h1>
      <p>Companies, deals, pipeline stages, and sales activity for this workspace.</p>
    </div>
    <div class="header-actions">
      <button class="secondary-button" type="button" onclick={() => goto("/crm/companies")}>
        <Building2 size={16} />
        Companies
      </button>
      <button class="icon-button" type="button" aria-label="Refresh CRM" title="Refresh CRM" onclick={() => crm.loadPipelines()}>
        <RefreshCw size={16} class={$crm.loading ? "spinning" : ""} />
      </button>
    </div>
  </header>

  <section class="metrics" aria-label="Pipeline summary">
    <div><strong>{openDeals.length}</strong><span>Open deals</span></div>
    <div><strong>{formatCurrency(openValue)}</strong><span>Open value</span></div>
    <div><strong>{formatCurrency(weightedValue)}</strong><span>Weighted value</span></div>
    <div><strong>{$crm.companies.length}</strong><span>Companies</span></div>
  </section>

  <div class="pipeline-toolbar">
    <div class="pipeline-tabs" role="tablist" aria-label="Sales pipelines">
      {#each $crm.pipelines as pipeline (pipeline.id)}
        <button
          type="button"
          role="tab"
          aria-selected={$crm.currentPipeline?.id === pipeline.id}
          class:active={$crm.currentPipeline?.id === pipeline.id}
          onclick={() => selectPipeline(pipeline)}
        >{pipeline.name}</button>
      {/each}
    </div>
  </div>

  {#if $crm.error && !$crm.pipelines.length}
    <div class="state state-error" role="alert">
      <span>{$crm.error}</span>
      <button type="button" onclick={() => crm.loadPipelines()}>Retry</button>
    </div>
  {:else if $crm.loading && !$crm.pipelines.length}
    <div class="state">Loading pipeline...</div>
  {:else if !$crm.pipelines.length}
    <div class="state empty-state">
      <strong>No sales pipeline yet</strong>
      <span>Create a pipeline through the CRM API or connected sales system.</span>
    </div>
  {:else}
    <div class="pipeline-board">
      {#each $crm.stages as stage (stage.id)}
        <section class="stage-column">
          <header>
            <span class="stage-dot" style:background={stage.color || "#737373"}></span>
            <strong>{stage.name}</strong>
            <span>{dealsForStage(stage.id).length}</span>
          </header>
          <div class="stage-value">{formatCurrency(dealsForStage(stage.id).reduce((sum, deal) => sum + (deal.amount ?? 0), 0))}</div>
          <div class="deal-list">
            {#each dealsForStage(stage.id) as deal (deal.id)}
              <button class="deal-card" type="button" onclick={() => goto(`/crm/deals/${deal.id}`)}>
                <strong>{deal.name}</strong>
                <span>{deal.company_name || "No company linked"}</span>
                <div><b>{formatCurrency(deal.amount)}</b><small>{deal.probability ?? 0}%</small></div>
              </button>
            {:else}
              <div class="empty-stage">No deals</div>
            {/each}
          </div>
        </section>
      {/each}
    </div>
  {/if}
</div>

<style>
  .crm-page { min-height: 100%; background: var(--dbg, #fff); color: var(--dt, #111); }
  .page-header { min-height: 88px; padding: 20px 24px; display: flex; align-items: center; justify-content: space-between; gap: 20px; border-bottom: 1px solid var(--dbd, #e5e5e5); }
  h1 { margin: 0; font-size: 22px; font-weight: 700; letter-spacing: 0; }
  p { margin: 5px 0 0; color: var(--dt3, #777); font-size: 13px; }
  .header-actions { display: flex; gap: 8px; }
  button { font: inherit; }
  .icon-button, .secondary-button { height: 36px; border: 1px solid var(--dbd, #ddd); background: var(--dbg, #fff); color: var(--dt2, #555); display: inline-flex; align-items: center; justify-content: center; gap: 7px; cursor: pointer; }
  .icon-button { width: 36px; }
  .secondary-button { padding: 0 12px; font-weight: 600; }
  .metrics { display: grid; grid-template-columns: repeat(4, minmax(0, 1fr)); border-bottom: 1px solid var(--dbd, #e5e5e5); }
  .metrics div { padding: 14px 20px; display: flex; align-items: baseline; gap: 8px; border-right: 1px solid var(--dbd, #e5e5e5); }
  .metrics div:last-child { border-right: 0; }
  .metrics strong { font-size: 16px; }
  .metrics span, .stage-value { color: var(--dt3, #777); font-size: 12px; }
  .pipeline-toolbar { min-height: 50px; padding: 8px 20px; display: flex; align-items: center; border-bottom: 1px solid var(--dbd, #e5e5e5); overflow-x: auto; }
  .pipeline-tabs { display: flex; gap: 4px; }
  .pipeline-tabs button { height: 32px; padding: 0 11px; border: 0; background: transparent; color: var(--dt3, #777); cursor: pointer; }
  .pipeline-tabs button.active { background: var(--dbg2, #f3f3f3); color: var(--dt, #111); font-weight: 650; }
  .pipeline-board { display: grid; grid-auto-flow: column; grid-auto-columns: minmax(250px, 1fr); min-height: calc(100vh - 230px); overflow-x: auto; }
  .stage-column { min-width: 250px; padding: 14px; border-right: 1px solid var(--dbd, #e5e5e5); background: var(--dbg2, #fafafa); }
  .stage-column > header { display: flex; align-items: center; gap: 7px; font-size: 13px; }
  .stage-column > header > span:last-child { margin-left: auto; color: var(--dt3, #777); }
  .stage-dot { width: 7px; height: 7px; border-radius: 50%; }
  .stage-value { margin: 5px 0 12px 14px; }
  .deal-list { display: flex; flex-direction: column; gap: 8px; }
  .deal-card { width: 100%; padding: 12px; text-align: left; background: var(--dbg, #fff); border: 1px solid var(--dbd, #e5e5e5); cursor: pointer; }
  .deal-card:hover { border-color: var(--dt3, #777); }
  .deal-card strong, .deal-card span { display: block; }
  .deal-card strong { font-size: 13px; color: var(--dt, #111); }
  .deal-card span { margin-top: 4px; font-size: 12px; color: var(--dt3, #777); }
  .deal-card div { margin-top: 12px; display: flex; justify-content: space-between; align-items: center; }
  .deal-card b { font-size: 12px; }
  .deal-card small { color: var(--dt3, #777); }
  .empty-stage { min-height: 70px; display: grid; place-items: center; border: 1px dashed var(--dbd, #ddd); color: var(--dt3, #888); font-size: 12px; }
  .state { min-height: 260px; display: flex; align-items: center; justify-content: center; color: var(--dt3, #777); font-size: 13px; }
  .state-error { gap: 12px; color: var(--bos-status-error, #b91c1c); }
  .state-error button { border: 1px solid currentColor; background: transparent; color: inherit; padding: 6px 10px; cursor: pointer; }
  .empty-state { flex-direction: column; gap: 7px; }
  .empty-state strong { color: var(--dt, #111); }
  .spinning { animation: spin .8s linear infinite; }
  @keyframes spin { to { transform: rotate(360deg); } }
  @media (max-width: 760px) {
    .page-header { align-items: flex-start; padding: 16px; }
    .metrics { grid-template-columns: repeat(2, 1fr); }
    .metrics div:nth-child(2) { border-right: 0; }
    .metrics div:nth-child(-n+2) { border-bottom: 1px solid var(--dbd, #e5e5e5); }
  }
</style>
