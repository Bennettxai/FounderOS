<script lang="ts">
	import { onMount } from 'svelte';
	import { currentWorkspace } from '$lib/stores/workspaces';
	import {
		getIntelligence,
		synthesizeIntelligence,
		type Signal,
		type Insight
	} from '$lib/api/intelligence';
	import {
		Sparkles,
		AlertTriangle,
		Activity,
		Lightbulb,
		Loader2,
		RefreshCw,
		CheckSquare,
		Users,
		FolderKanban
	} from 'lucide-svelte';

	let signals = $state<Signal[]>([]);
	let insights = $state<Insight[]>([]);
	let aiModel = $state('');
	let aiGenerated = $state<string | null>(null);
	let aiAvailable = $state(false);
	let loading = $state(true);
	let synthesizing = $state(false);
	let error = $state<string | null>(null);

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
			const r = await getIntelligence();
			signals = r.signals ?? [];
			insights = r.insights ?? [];
			aiModel = r.ai_model ?? '';
			aiGenerated = r.ai_generated ?? null;
			aiAvailable = r.ai_available ?? false;
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to load intelligence';
		} finally {
			loading = false;
		}
	}

	async function synthesize() {
		synthesizing = true;
		error = null;
		try {
			const r = await synthesizeIntelligence();
			insights = r.insights ?? [];
			if (r.ai_model) aiModel = r.ai_model;
			aiGenerated = new Date().toISOString();
			if (!insights.length && r.note) error = r.note;
		} catch (e) {
			error = e instanceof Error ? e.message : 'AI synthesis failed';
		} finally {
			synthesizing = false;
		}
	}

	const risks = $derived(signals.filter((s) => s.kind === 'risk'));
	const plainSignals = $derived(signals.filter((s) => s.kind === 'signal'));
	const recommendations = $derived(insights.filter((i) => i.type === 'recommendation'));
	const aiObservations = $derived(insights.filter((i) => i.type !== 'recommendation'));

	function sevRank(s: string): number {
		return s === 'high' ? 0 : s === 'medium' ? 1 : 2;
	}
	const risksSorted = $derived([...risks].sort((a, b) => sevRank(a.severity) - sevRank(b.severity)));

	function srcIcon(t: string) {
		return t === 'task' ? CheckSquare : t === 'client' ? Users : FolderKanban;
	}
	function timeAgo(iso: string | null): string {
		if (!iso) return '';
		const secs = Math.max(0, (Date.now() - new Date(iso).getTime()) / 1000);
		if (secs < 60) return 'just now';
		if (secs < 3600) return `${Math.floor(secs / 60)}m ago`;
		if (secs < 86400) return `${Math.floor(secs / 3600)}h ago`;
		return `${Math.floor(secs / 86400)}d ago`;
	}
</script>

<svelte:head><title>Intelligence - BusinessOS</title></svelte:head>

<div class="page">
	<header class="page-header">
		<div class="page-icon"><Sparkles size={22} strokeWidth={1.8} /></div>
		<div class="head-text">
			<h1 class="page-title">Intelligence</h1>
			<p class="page-desc">Signals, risks, and AI recommendations across your workspace.</p>
		</div>
		<button class="btn btn-primary" onclick={synthesize} disabled={synthesizing || !aiAvailable}>
			{#if synthesizing}<Loader2 size={15} class="spin" /> Analyzing…{:else}<Sparkles size={15} /> {insights.length ? 'Re-analyze' : 'Generate insights'}{/if}
		</button>
	</header>

	{#if error}<div class="error-bar">{error}</div>{/if}

	{#if loading}
		<div class="loading"><Loader2 size={22} class="spin" /> Reading your workspace…</div>
	{:else}
		<!-- Risks -->
		{#if risksSorted.length}
			<section>
				<h2 class="sec-title"><AlertTriangle size={16} /> Risks</h2>
				<div class="cards">
					{#each risksSorted as s}
						{@const Ico = srcIcon(s.source_type)}
						<div class="card sev-{s.severity}">
							<div class="card-top"><Ico size={15} /><span class="card-title">{s.title}</span><span class="sev-tag">{s.severity}</span></div>
							<p class="card-detail">{s.detail}</p>
						</div>
					{/each}
				</div>
			</section>
		{/if}

		<!-- Recommendations (AI) -->
		<section>
			<h2 class="sec-title"><Lightbulb size={16} /> Recommendations
				{#if aiGenerated}<span class="ai-meta">· {aiModel || 'AI'} · {timeAgo(aiGenerated)}</span>{/if}
			</h2>
			{#if recommendations.length}
				<div class="cards">
					{#each recommendations as i}
						<div class="card rec pri-{i.priority}">
							<div class="card-top"><Lightbulb size={15} /><span class="card-title">{i.title}</span><span class="sev-tag">{i.priority}</span></div>
							<p class="card-detail">{i.detail}</p>
						</div>
					{/each}
				</div>
			{:else}
				<div class="mini-empty">
					{#if aiAvailable}
						<p>No AI recommendations yet. Click <strong>Generate insights</strong> to analyze your workspace.</p>
					{:else}
						<p>AI is not configured for this workspace, so recommendations are unavailable.</p>
					{/if}
				</div>
			{/if}
		</section>

		<!-- AI observations (non-recommendation insights) -->
		{#if aiObservations.length}
			<section>
				<h2 class="sec-title"><Sparkles size={16} /> AI observations</h2>
				<div class="cards">
					{#each aiObservations as i}
						<div class="card pri-{i.priority}">
							<div class="card-top"><span class="type-tag">{i.type}</span><span class="card-title">{i.title}</span><span class="sev-tag">{i.priority}</span></div>
							<p class="card-detail">{i.detail}</p>
						</div>
					{/each}
				</div>
			</section>
		{/if}

		<!-- Deterministic signals -->
		{#if plainSignals.length}
			<section>
				<h2 class="sec-title"><Activity size={16} /> Signals</h2>
				<div class="cards">
					{#each plainSignals as s}
						{@const Ico = srcIcon(s.source_type)}
						<div class="card">
							<div class="card-top"><Ico size={15} /><span class="card-title">{s.title}</span></div>
							<p class="card-detail">{s.detail}</p>
						</div>
					{/each}
				</div>
			</section>
		{/if}

		{#if !signals.length && !insights.length}
			<div class="empty-state">
				<Sparkles size={40} strokeWidth={1.4} class="empty-icon" />
				<p class="empty-title">All clear</p>
				<p class="empty-body">No overdue tasks, stale clients, or stalled projects right now. Intelligence will surface signals as things change.</p>
			</div>
		{/if}
	{/if}
</div>

<style>
	.page { display: flex; flex-direction: column; gap: 24px; padding: 28px 32px; height: 100%; overflow-y: auto; }
	.page-header { display: flex; align-items: flex-start; gap: 14px; }
	.page-icon { width: 42px; height: 42px; border-radius: 10px; border: 1px solid var(--dbd); background: color-mix(in srgb, var(--dt) 5%, transparent); color: var(--dt2); display: flex; align-items: center; justify-content: center; flex-shrink: 0; }
	.head-text { flex: 1; min-width: 0; }
	.page-title { font-size: 1.25rem; font-weight: 650; color: var(--dt); letter-spacing: -0.02em; margin: 0; }
	.page-desc { font-size: 0.875rem; color: var(--dt3); margin: 2px 0 0; }

	.btn { display: inline-flex; align-items: center; gap: 6px; padding: 8px 14px; border-radius: 8px; font-size: 0.84rem; font-weight: 550; cursor: pointer; border: 1px solid transparent; white-space: nowrap; }
	.btn-primary { background: var(--bos-v2-button-primary); color: var(--bos-v2-button-pureWhiteText); }
	.btn-primary:hover { opacity: 0.9; }
	.btn:disabled { opacity: 0.5; cursor: default; }

	.error-bar { padding: 10px 14px; border-radius: 8px; background: color-mix(in srgb, #ef4444 12%, transparent); color: #ef4444; font-size: 0.83rem; }
	.loading { display: flex; align-items: center; gap: 8px; padding: 48px; justify-content: center; color: var(--dt3); font-size: 0.9rem; }

	section { display: flex; flex-direction: column; gap: 12px; }
	.sec-title { display: flex; align-items: center; gap: 7px; font-size: 0.82rem; font-weight: 650; text-transform: uppercase; letter-spacing: 0.05em; color: var(--dt2); margin: 0; }
	.ai-meta { text-transform: none; letter-spacing: 0; font-weight: 450; color: var(--dt3); font-size: 0.76rem; }

	.cards { display: grid; grid-template-columns: repeat(auto-fill, minmax(300px, 1fr)); gap: 12px; }
	.card { border: 1px solid var(--dbd); border-radius: 12px; padding: 14px 16px; display: flex; flex-direction: column; gap: 6px; background: color-mix(in srgb, var(--dt) 2%, transparent); border-left-width: 3px; }
	.card.sev-high, .card.pri-high { border-left-color: #ef4444; }
	.card.sev-medium, .card.pri-medium { border-left-color: #f59e0b; }
	.card.sev-low, .card.pri-low { border-left-color: var(--dbd); }
	.card.rec { background: color-mix(in srgb, #6366f1 6%, transparent); }
	.card-top { display: flex; align-items: center; gap: 8px; color: var(--dt2); }
	.card-title { font-size: 0.9rem; font-weight: 600; color: var(--dt); flex: 1; }
	.sev-tag { font-size: 0.66rem; text-transform: uppercase; letter-spacing: 0.04em; padding: 2px 6px; border-radius: 4px; background: color-mix(in srgb, var(--dt) 8%, transparent); color: var(--dt2); }
	.type-tag { font-size: 0.66rem; text-transform: uppercase; letter-spacing: 0.04em; padding: 2px 6px; border-radius: 4px; border: 1px solid var(--dbd); color: var(--dt3); }
	.card-detail { font-size: 0.85rem; color: var(--dt2); margin: 0; line-height: 1.45; }

	.mini-empty { padding: 16px; border: 1px dashed var(--dbd); border-radius: 10px; color: var(--dt3); font-size: 0.85rem; }
	.mini-empty strong { color: var(--dt2); }
	.empty-state { flex: 1; display: flex; flex-direction: column; align-items: center; justify-content: center; gap: 10px; padding: 48px 24px; text-align: center; }
	.empty-state :global(.empty-icon) { color: var(--dt4, var(--dt3)); opacity: 0.5; }
	.empty-title { font-size: 0.95rem; font-weight: 600; color: var(--dt2); margin: 4px 0 0; }
	.empty-body { font-size: 0.84rem; color: var(--dt3); max-width: 380px; margin: 0; }

	:global(.spin) { animation: spin 0.8s linear infinite; }
	@keyframes spin { to { transform: rotate(360deg); } }
	@media (max-width: 768px) { .page { padding: 16px 18px; } .cards { grid-template-columns: 1fr; } }
</style>
