<script lang="ts">
	import { onMount } from 'svelte';
	import { currentWorkspace } from '$lib/stores/workspaces';
	import { getDataSummary, type DataSummary } from '$lib/api/data';
	import {
		Database,
		Loader2,
		RefreshCw,
		Download,
		CheckSquare,
		FolderKanban,
		Users,
		Package,
		FileText,
		Megaphone,
		UserSquare,
		Image as ImageIcon,
		Library
	} from 'lucide-svelte';

	let summary = $state<DataSummary | null>(null);
	let loading = $state(true);
	let error = $state<string | null>(null);

	const ICONS: Record<string, typeof Database> = {
		tasks: CheckSquare,
		projects: FolderKanban,
		clients: Users,
		offers: Package,
		content: FileText,
		campaigns: Megaphone,
		personas: UserSquare,
		assets: ImageIcon,
		resources: Library
	};

	let wsId = $state<string | null | undefined>(undefined);
	$effect(() => {
		const id = $currentWorkspace?.id ?? null;
		if (id !== wsId) {
			wsId = id;
			load();
		}
	});
	onMount(load);

	async function load() {
		loading = true;
		error = null;
		try {
			summary = await getDataSummary();
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to load data summary';
		} finally {
			loading = false;
		}
	}
</script>

<svelte:head><title>Data - BusinessOS</title></svelte:head>

<div class="page">
	<header class="page-header">
		<div class="page-icon"><Database size={22} strokeWidth={1.8} /></div>
		<div class="head-text">
			<h1 class="page-title">Data</h1>
			<p class="page-desc">Every structured data entity in your workspace and how many records it holds.</p>
		</div>
		<button class="btn btn-ghost" onclick={load} disabled={loading}>
			{#if loading}<Loader2 size={15} class="spin" />{:else}<RefreshCw size={15} />{/if} Refresh
		</button>
	</header>

	{#if error}<div class="error-bar">{error}</div>{/if}

	{#if loading}
		<div class="loading"><Loader2 size={22} class="spin" /> Reading your workspace…</div>
	{:else if summary}
		<div class="note">
			<Download size={15} />
			<span>Export is coming soon. This view is read-only — no records are modified or deleted here.</span>
		</div>

		<div class="summary-bar">
			<span class="total-value">{summary.total.toLocaleString()}</span>
			<span class="total-label">total records across {summary.entities.length} entities</span>
		</div>

		<div class="rows">
			{#each summary.entities as e (e.key)}
				{@const Ico = ICONS[e.key] ?? Database}
				<div class="row">
					<div class="row-icon"><Ico size={17} strokeWidth={1.8} /></div>
					<div class="row-body">
						<span class="row-label">{e.label}</span>
						<span class="row-table">{e.table}</span>
					</div>
					<span class="row-count">{e.count.toLocaleString()}</span>
				</div>
			{/each}
		</div>
	{/if}
</div>

<style>
	.page { display: flex; flex-direction: column; gap: 18px; padding: 28px 32px; height: 100%; overflow-y: auto; }
	.page-header { display: flex; align-items: flex-start; gap: 14px; }
	.page-icon { width: 42px; height: 42px; border-radius: 10px; border: 1px solid var(--dbd); background: color-mix(in srgb, var(--dt) 5%, transparent); color: var(--dt2); display: flex; align-items: center; justify-content: center; flex-shrink: 0; }
	.head-text { flex: 1; min-width: 0; }
	.page-title { font-size: 1.25rem; font-weight: 650; color: var(--dt); letter-spacing: -0.02em; margin: 0; }
	.page-desc { font-size: 0.875rem; color: var(--dt3); margin: 2px 0 0; }

	.btn { display: inline-flex; align-items: center; gap: 6px; padding: 8px 14px; border-radius: 8px; font-size: 0.84rem; font-weight: 550; cursor: pointer; border: 1px solid transparent; white-space: nowrap; }
	.btn-ghost { background: transparent; border-color: var(--dbd); color: var(--dt2); }
	.btn-ghost:hover { background: color-mix(in srgb, var(--dt) 5%, transparent); }
	.btn:disabled { opacity: 0.6; cursor: default; }

	.error-bar { padding: 10px 14px; border-radius: 8px; background: color-mix(in srgb, #ef4444 12%, transparent); color: #ef4444; font-size: 0.83rem; }
	.loading { display: flex; align-items: center; gap: 8px; padding: 48px; justify-content: center; color: var(--dt3); font-size: 0.9rem; }

	.note { display: flex; align-items: center; gap: 9px; padding: 11px 14px; border-radius: 10px; border: 1px dashed var(--dbd); color: var(--dt3); font-size: 0.82rem; }
	.note :global(svg) { flex-shrink: 0; }

	.summary-bar { display: flex; align-items: baseline; gap: 10px; }
	.total-value { font-size: 1.6rem; font-weight: 680; color: var(--dt); letter-spacing: -0.02em; }
	.total-label { font-size: 0.83rem; color: var(--dt3); }

	.rows { display: flex; flex-direction: column; border: 1px solid var(--dbd); border-radius: 12px; overflow: hidden; }
	.row { display: flex; align-items: center; gap: 12px; padding: 12px 16px; border-bottom: 1px solid var(--dbd); }
	.row:last-child { border-bottom: none; }
	.row:hover { background: color-mix(in srgb, var(--dt) 3%, transparent); }
	.row-icon { width: 34px; height: 34px; border-radius: 8px; background: color-mix(in srgb, var(--dt) 6%, transparent); color: var(--dt2); display: flex; align-items: center; justify-content: center; flex-shrink: 0; }
	.row-body { flex: 1; min-width: 0; display: flex; flex-direction: column; gap: 1px; }
	.row-label { font-size: 0.88rem; font-weight: 550; color: var(--dt); }
	.row-table { font-size: 0.72rem; color: var(--dt3); font-family: ui-monospace, SFMono-Regular, Menlo, monospace; }
	.row-count { font-size: 1rem; font-weight: 600; color: var(--dt2); font-variant-numeric: tabular-nums; }

	:global(.spin) { animation: spin 0.8s linear infinite; }
	@keyframes spin { to { transform: rotate(360deg); } }
	@media (max-width: 768px) { .page { padding: 16px 18px; } }
	@media (max-width: 480px) { .page { padding: 12px 14px; } }
</style>
