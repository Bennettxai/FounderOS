<!-- Blueprint, hierarchy edition: the compiled graph as expandable buckets.
     Three disclosure levels, focus mode on click (everything unrelated blurs,
     the selection's relations light up in its kind's colour), ⌘K search that
     dims live, a kind legend that filters, a breadcrumb, the inspector, the
     ask bar, and "trace the chain" which runs a flare down the command spine.
     Port of FounderOS v1 components/blueprint/HierarchyWorkspace.tsx. -->
<script lang="ts">
	import { tick } from 'svelte';
	import { Layers, Maximize2, Minimize2, Play, Scan } from '$lib/founderos/icons';
	import type { BlueprintGraph } from './graph';
	import { buildHierarchy, expandedAtLevel, indexHierarchy, KIND_LABEL, LEGEND, matchesKind, matchesQuery, relationsOf, type HKind } from './hierarchy';
	import { focusEdges as computeFocusEdges, layoutHierarchy, spineEdges } from './hierarchy-layout';
	import AskBar from './AskBar.svelte';
	import HierarchyCanvas, { type FocusMode } from './HierarchyCanvas.svelte';
	import HierarchyInspector from './HierarchyInspector.svelte';
	import HierarchySearch from './HierarchySearch.svelte';

	let { graph, subline }: { graph: BlueprintGraph; subline: string } = $props();

	type Level = 1 | 2 | 3;
	const LEVELS: Array<{ n: Level; label: string; Icon: typeof Layers }> = [
		{ n: 1, label: 'Overview', Icon: Minimize2 },
		{ n: 2, label: 'Systems', Icon: Layers },
		{ n: 3, label: 'Everything', Icon: Maximize2 }
	];
	const wait = (ms: number) => new Promise((r) => setTimeout(r, ms));

	const h = $derived(buildHierarchy(graph));
	const idx = $derived(indexHierarchy(h));

	let level = $state<Level>(1);
	let expanded = $state<Set<string>>(new Set());
	let selected = $state<string | null>(null);
	let opened = $state<string[]>([]);
	let kindFilter = $state<HKind | null>(null);
	let query = $state('');
	let searchOpen = $state(false);
	let inspMin = $state(false);
	let tracing = $state(false);
	let lit = $state<Set<string>>(new Set());

	let canvas: HierarchyCanvas | undefined = $state();
	let ask: AskBar | undefined = $state();

	const layout = $derived(layoutHierarchy(h, expanded));
	const spine = $derived(spineEdges(h, idx, layout));

	const q = $derived(query.trim().toLowerCase());
	const focus = $derived.by<FocusMode>(() => {
		if (selected) return { mode: 'focus', selected, hot: computeFocusEdges(idx, layout, selected, relationsOf(h, idx, selected)).hot };
		if (q.length >= 2 || kindFilter) {
			const match = new Set<string>();
			for (const id of idx.items.keys()) if (kindFilter ? matchesKind(idx, id, kindFilter) : matchesQuery(idx, id, q)) match.add(id);
			return { mode: 'filter', match };
		}
		return { mode: 'none' };
	});
	const focusEdgeList = $derived(selected ? computeFocusEdges(idx, layout, selected, relationsOf(h, idx, selected)).edges : []);
	const selectedItem = $derived(selected ? (idx.items.get(selected) ?? null) : null);
	const focusColor = $derived(selectedItem ? `var(--bh-k-${selectedItem.kind}, var(--accent))` : null);

	// camera moves that must wait for the new layout to commit
	async function flyAfterLayout(id: string, s: number) {
		await tick();
		canvas?.flyTo(id, s);
	}
	async function fitAfterLayout() {
		await tick();
		canvas?.fitAll();
	}

	function select(id: string | null, fly = true) {
		if (id && !idx.items.has(id)) return;
		if (id) {
			const need = idx.chain(id).filter((p) => p !== id && idx.items.get(p)?.type === 'group' && !expanded.has(p));
			if (need.length) expanded = new Set([...expanded, ...need]);
			if (!opened.includes(id)) {
				const next = [...opened, id];
				opened = next.length > 5 ? next.slice(next.length - 5) : next;
			}
			kindFilter = null;
			if (fly) void flyAfterLayout(id, 0.95);
		}
		selected = id;
	}

	function toggle(id: string) {
		const open = expanded.has(id);
		const next = new Set(expanded);
		if (open) next.delete(id);
		else next.add(id);
		expanded = next;
		if (open && selected && idx.chain(selected).includes(id) && selected !== id) selected = id;
		if (!open) void flyAfterLayout(id, 0.85);
	}

	function setLevel(n: Level) {
		level = n;
		expanded = expandedAtLevel(idx, n);
		void fitAfterLayout();
	}

	const clear = () => (selected = null);
	function closeSearch() {
		searchOpen = false;
		query = '';
	}
	const openSearch = () => (searchOpen = true);

	function onkeydown(ev: KeyboardEvent) {
		const el = document.activeElement as HTMLElement | null;
		const typing = !!el && (el.tagName === 'INPUT' || el.tagName === 'TEXTAREA' || el.isContentEditable);
		if ((ev.metaKey || ev.ctrlKey) && ev.key.toLowerCase() === 'k') {
			// the page's own ⌘K: stop the /os palette opening over the map
			ev.preventDefault();
			ev.stopImmediatePropagation();
			if (searchOpen) closeSearch();
			else openSearch();
			return;
		}
		if (ev.key === 'Escape') {
			if (searchOpen) closeSearch();
			else if (!ask?.isFocused()) clear();
			return;
		}
		if (typing) return;
		if (ev.key === '/') {
			ev.preventDefault();
			ask?.focus();
		}
		if (ev.key === 'i' && !ev.metaKey && opened.length) inspMin = !inspMin;
	}

	// trace the chain: light the spine band by band, a flare running each structural edge
	async function trace() {
		if (tracing) return;
		tracing = true;
		selected = null;
		canvas?.fitAll();
		const litIds = new Set<string>();
		const light = (ids: string[]) => {
			ids.forEach((id) => litIds.add(id));
			lit = new Set(litIds);
		};
		await wait(400);
		for (let i = 0; i < h.spine.length; i++) {
			const e = h.spine[i];
			light([e.from]);
			await wait(180);
			light([`sp-${i}`]);
			await canvas?.runFlare(`sp-${i}`, 560);
			light(Array.isArray(e.to) ? e.to : [e.to]);
		}
		await wait(1800);
		lit = new Set();
		tracing = false;
	}

	function closeTab(id: string) {
		const next = opened.filter((o) => o !== id);
		opened = next;
		if (selected === id) selected = next[next.length - 1] ?? null;
	}

	const hasInspector = $derived(opened.length > 0);
	const crumb = $derived(selected ? idx.chain(selected) : []);
</script>

<svelte:window onkeydowncapture={onkeydown} />

<div class="bh-page">
	<header class="bh-top">
		<div class="bh-title">
			<div class="bh-h1">Blueprint</div>
			<div class="bh-sub">{subline}</div>
		</div>
		<div class="bh-levels">
			{#each LEVELS as { n, label, Icon } (n)}
				<button type="button" class="bn-pressable bh-lvl {level === n ? 'is-on' : ''}" onclick={() => setLevel(n)}>
					<Icon />
					<span>{label}</span>
				</button>
			{/each}
		</div>
		<HierarchySearch
			{idx}
			open={searchOpen}
			{query}
			onopen={openSearch}
			onclose={closeSearch}
			onquery={(v) => {
				query = v;
				if (selected) selected = null;
			}}
			onpick={(id) => {
				closeSearch();
				select(id);
			}}
		/>
		<div class="bh-actions">
			<button type="button" class="bn-pressable bh-btn" onclick={() => canvas?.fitAll()}><Scan /><span>Fit</span></button>
			<button type="button" class="bn-pressable bh-btn bh-btn-primary {tracing ? 'is-busy' : ''}" onclick={() => void trace()}><Play /><span>Trace the chain</span></button>
		</div>
	</header>

	<div class="bh-kinds">
		{#each LEGEND as k (k)}
			<button
				type="button"
				class="bn-pressable bh-kind {kindFilter === k ? 'is-on' : ''}"
				style="--c: var(--bh-k-{k})"
				onclick={() => {
					const next = kindFilter === k ? null : k;
					kindFilter = next;
					if (next) selected = null;
				}}
			>
				<i class="sw"></i>
				{KIND_LABEL[k]}
			</button>
		{/each}
	</div>

	<section class="bh-stage {hasInspector && !inspMin ? 'has-inspector' : ''} {hasInspector && inspMin ? 'inspector-min' : ''}">
		<HierarchyCanvas
			bind:this={canvas}
			{h}
			{idx}
			{layout}
			{spine}
			focusEdges={focusEdgeList}
			{focusColor}
			{focus}
			{lit}
			onselect={(id) => select(id === selected ? null : id)}
			ontoggle={toggle}
			onfly={(id) => canvas?.flyTo(id, 1.3)}
		/>

		<AskBar bind:this={ask} scope={selectedItem} onclearscope={clear} />

		<div class="bh-crumb {crumb.length ? 'is-on' : ''}">
			{#each crumb as p, i (p)}
				<span>
					<!-- svelte-ignore a11y_click_events_have_key_events -->
					<b role="button" tabindex="-1" onclick={() => select(p)}>{idx.items.get(p)?.name ?? p}</b>
					{#if i < crumb.length - 1}<i class="sep">›</i>{/if}
				</span>
			{/each}
		</div>

		<HierarchyInspector
			{h}
			{idx}
			{opened}
			active={selected}
			{expanded}
			minimized={inspMin}
			onselect={(id) => select(id)}
			onclosetab={closeTab}
			onminimize={(v) => (inspMin = v)}
			onfocus={() => selected && canvas?.flyTo(selected, 1.3)}
			onfit={() => canvas?.fitAll()}
			ontoggle={toggle}
		/>
	</section>
</div>
