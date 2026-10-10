<!-- The memory core's overview (FounderOS v1 KnowledgeDetail MemoryCoreCard):
     the whole knowledge brain at a glance, with a live query bar. Searches go
     through the bridge's federated Optimal Engine retrieval. -->
<script lang="ts">
	import { Boxes, CornerDownLeft, Database, FileText, FolderTree, Loader2, Search } from '$lib/founderos/icons';
	import { founderosFetch } from '$lib/founderos/api';
	import PanelHeader from './PanelHeader.svelte';
	import SectionLabel from './SectionLabel.svelte';
	import type { MemoryGraph } from '../memory-core';
	import type { BrainHit, BrainQueryBody } from '../types';

	let { memory, color, onBack, onClose }: { memory: MemoryGraph | null | undefined; color: string; onBack?: () => void; onClose?: () => void } = $props();

	const fmt = (n: number): string => (n >= 1000 ? `${(n / 1000).toFixed(n >= 10_000 ? 0 : 1)}k` : String(n));
	const pages = $derived(memory ? memory.nodes.filter((n) => n.type === 'page') : []);
	const folders = $derived(memory ? memory.nodes.filter((n) => n.type === 'folder') : []);
	const links = $derived(memory ? memory.edges.filter((e) => e.type === 'wikilink').length : 0);
	const words = $derived(pages.reduce((s, p) => s + (p.wordCount ?? 0), 0));
	const clusters = $derived(new Set(pages.filter((p) => p.links > 0).map((p) => p.cluster)).size);
	const topFolders = $derived.by(() => {
		const byFolder = new Map<string, number>();
		for (const p of pages) byFolder.set(p.folder, (byFolder.get(p.folder) ?? 0) + 1);
		return [...byFolder.entries()].sort((a, b) => b[1] - a[1] || a[0].localeCompare(b[0])).slice(0, 8);
	});
	const maxFolder = $derived(topFolders[0]?.[1] ?? 1);
	const topNotes = $derived([...pages].sort((a, b) => b.links - a.links || (b.wordCount ?? 0) - (a.wordCount ?? 0)).slice(0, 12));
	const suggestions = $derived(topFolders.slice(0, 5).map(([f]) => f));

	let q = $state('');
	let results = $state<BrainHit[] | null>(null);
	let provider = $state<string | null>(null);
	let loading = $state(false);
	let error = $state<string | null>(null);

	async function runSearch(term: string) {
		const t = term.trim();
		if (!t) return;
		q = t;
		loading = true;
		error = null;
		try {
			const data = await founderosFetch<BrainQueryBody>(`/pages/brain/query?q=${encodeURIComponent(t)}`);
			results = data.results ?? [];
			provider = data.provider ?? null;
		} catch {
			error = 'search failed — the engines may be offline';
			results = [];
		} finally {
			loading = false;
		}
	}
	function clear() {
		results = null;
		error = null;
		q = '';
	}
</script>

<div class="flex h-full flex-col" data-card="memory-core">
	<PanelHeader title="Optimal Engine" sub="the company knowledge brain" {color} {onBack} {onClose} />
	<div class="min-h-0 flex-1 overflow-y-auto overscroll-contain px-3 py-2.5">
		<form
			class="mb-2"
			onsubmit={(e) => {
				e.preventDefault();
				runSearch(q);
			}}
		>
			<div class="kg-b kg-surf flex items-center gap-1.5 rounded-[8px] border px-2.5 py-2 focus-within:border-[var(--bn-border-strong)]">
				<span class="bn-dim shrink-0"><Search size={14} /></span>
				<input bind:value={q} placeholder="query the whole brain…" aria-label="Search the knowledge base" class="bn-text kg-input min-w-0 flex-1 bg-transparent text-[12px] focus:outline-none" />
				{#if loading}
					<span class="bn-dim shrink-0 animate-spin"><Loader2 size={14} /></span>
				{:else}
					<button type="submit" aria-label="Search" class="bn-pressable bn-dim kg-h-accent shrink-0"><CornerDownLeft size={14} /></button>
				{/if}
			</div>
		</form>
		{#if suggestions.length > 0 && results === null}
			<div class="mb-4 flex flex-wrap gap-1.5">
				<span class="bn-dim font-mono text-[9px] uppercase tracking-[0.14em]">try</span>
				{#each suggestions as s (s)}
					<button type="button" onclick={() => runSearch(s)} class="bn-pressable kg-b bn-muted kg-h-bs kg-h-text rounded-[5px] border px-2 py-0.5 font-mono text-[10px]">{s}</button>
				{/each}
			</div>
		{/if}

		{#if results !== null}
			<div class="mb-4">
				<SectionLabel icon={Search}>
					{loading ? 'searching…' : `${results.length} result${results.length === 1 ? '' : 's'}`}{provider && !loading ? ` · ${provider}` : ''} ·
					<button type="button" onclick={clear} class="bn-pressable bn-dim kg-h-text">clear</button>
				</SectionLabel>
				{#if error}<p class="kg-warn mb-2 text-[10.5px]">{error}</p>{/if}
				{#if !loading && results.length === 0 && !error}<p class="bn-dim font-mono text-[10.5px]">nothing matched. try another phrasing.</p>{/if}
				<div class="flex flex-col gap-1.5">
					{#each results as r, i (i)}
						<div class="kg-b kg-surf rounded-[8px] border px-2.5 py-2">
							<div class="flex items-center justify-between gap-2">
								<span class="min-w-0 truncate text-[11.5px] font-semibold" style="color: {color}" title={r.title}>{r.title}</span>
								<span class="bn-dim shrink-0 font-mono text-[8.5px] uppercase tracking-wide">{r.source}</span>
							</div>
							<p class="bn-muted mt-1 text-[10.5px] leading-relaxed [text-wrap:pretty]">{r.snippet}</p>
						</div>
					{/each}
				</div>
			</div>
		{/if}

		{#if !memory}
			<p class="bn-dim font-mono text-[10.5px]">The brain is not loaded on this machine (the engines were unreadable). Numbers appear once the Optimal Engine workspaces can be read.</p>
		{:else}
			<p class="bn-muted mb-4 text-[11px] leading-relaxed">
				the operator's entire second brain, distilled: every page, workspace and link across the Optimal Engine, clustered into the domains the whole OS reasons over.
			</p>
			<SectionLabel icon={Boxes}>the brain in numbers</SectionLabel>
			<div class="mb-2 grid grid-cols-2 gap-2">
				{#each [['notes', pages.length], ['folders', folders.length], ['wiki-links', links], ['clusters', clusters]] as [label, value] (label)}
					<div class="kg-b kg-surf rounded-[8px] border px-2.5 py-2">
						<div class="text-[17px] font-bold leading-none" style="color: {color}">{fmt(value as number)}</div>
						<div class="bn-dim mt-1 font-mono text-[9px] uppercase tracking-[0.14em]">{label}</div>
					</div>
				{/each}
			</div>
			<p class="bn-dim mb-4 font-mono text-[9.5px]">{words > 0 ? `~${fmt(words)} words across ` : ''}{fmt(pages.length)} distilled notes</p>

			<SectionLabel icon={FolderTree}>knowledge domains</SectionLabel>
			<div class="mb-4 flex flex-col gap-1.5">
				{#each topFolders as [folder, count] (folder)}
					<button type="button" onclick={() => runSearch(folder)} class="bn-pressable group flex w-full items-center gap-2 text-left" title={`search "${folder}"`}>
						<span class="bn-muted kg-gh-text w-28 shrink-0 truncate text-[10.5px]" title={folder}>{folder}</span>
						<span class="kg-surf h-1.5 flex-1 overflow-hidden rounded-full"><span class="block h-full rounded-full" style="width: {(count / maxFolder) * 100}%; background: {color}"></span></span>
						<span class="bn-dim w-6 shrink-0 text-right font-mono text-[9.5px]">{count}</span>
					</button>
				{/each}
				{#if topFolders.length === 0}<span class="bn-dim font-mono text-[10px]">no folders</span>{/if}
			</div>

			{#if topNotes.length > 0}
				<SectionLabel icon={FileText}>most-linked notes</SectionLabel>
				<div class="mb-4 flex flex-col gap-1">
					{#each topNotes as note (note.id)}
						<button type="button" onclick={() => runSearch(note.label)} class="bn-pressable kg-b kg-surf kg-h-bs group flex w-full items-center gap-2 rounded-[8px] border px-2.5 py-1.5 text-left" title={`search "${note.label}"`}>
							<span class="bn-muted kg-gh-text min-w-0 flex-1 truncate text-[11px]">{note.label}</span>
							<span class="bn-dim shrink-0 font-mono text-[8.5px] uppercase tracking-wide">{note.folder}</span>
							<span class="bn-dim shrink-0 font-mono text-[8.5px]">{note.links}↔</span>
						</button>
					{/each}
				</div>
			{/if}

			<SectionLabel icon={Database}>where it comes from</SectionLabel>
			<p class="bn-muted text-[11px] leading-relaxed">
				Every Optimal Engine workspace on the hub and this machine: captures, call notes and pages, embedded and graphed by the engines. Every agent answers from these same notes, cited, never invented. Search above, or click any node in the graph to open its source.
			</p>
		{/if}
	</div>
</div>
