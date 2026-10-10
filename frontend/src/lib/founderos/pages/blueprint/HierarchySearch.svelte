<!-- The top-centre search pill that opens into a ⌘K palette. Typing dims the
     map live (the workspace owns `query`); Enter or a click selects the hit,
     which expands whatever contains it and flies the camera there. Port of
     FounderOS v1 components/blueprint/HierarchySearch.tsx. -->
<script lang="ts">
	import { Search } from '$lib/founderos/icons';
	import { KIND_ICON, KIND_LABEL, searchHierarchy, type HierarchyIndex, type HKind, type SearchHit } from './hierarchy';
	import { iconFor } from './icons';

	let {
		idx,
		open,
		query,
		onopen,
		onclose,
		onquery,
		onpick
	}: {
		idx: HierarchyIndex;
		open: boolean;
		query: string;
		onopen: () => void;
		onclose: () => void;
		onquery: (q: string) => void;
		onpick: (id: string) => void;
	} = $props();

	let inputEl: HTMLInputElement | undefined = $state();
	let rootEl: HTMLDivElement | undefined = $state();
	let cursor = $state(0);
	const hits = $derived(searchHierarchy(idx, query));

	$effect(() => {
		if (!open) return;
		const t = setTimeout(() => inputEl?.focus(), 60);
		const onDown = (ev: PointerEvent) => {
			if (rootEl && !rootEl.contains(ev.target as Node)) onclose();
		};
		document.addEventListener('pointerdown', onDown);
		return () => {
			clearTimeout(t);
			document.removeEventListener('pointerdown', onDown);
		};
	});

	function onkeydown(ev: KeyboardEvent) {
		if (ev.key === 'ArrowDown') {
			cursor = Math.min(hits.length - 1, cursor + 1);
			ev.preventDefault();
		} else if (ev.key === 'ArrowUp') {
			cursor = Math.max(0, cursor - 1);
			ev.preventDefault();
		} else if (ev.key === 'Enter') {
			const hit = hits[cursor];
			if (hit) onpick(hit.id);
		}
	}

	const groups = $derived.by(() => {
		const out: Array<[HKind, SearchHit[]]> = [];
		for (const hit of hits) {
			const g = out.find(([k]) => k === hit.kind);
			if (g) g[1].push(hit);
			else out.push([hit.kind, [hit]]);
		}
		return out;
	});
</script>

<div bind:this={rootEl} class="bh-search {open ? 'is-open' : ''}">
	<button type="button" class="bn-pressable bh-search-pill" onclick={onopen}>
		<Search />
		<span>Search the system</span>
		<kbd>⌘K</kbd>
	</button>
	<div class="bh-search-panel" aria-hidden={!open}>
		<div class="bh-search-row">
			<Search />
			<input
				bind:this={inputEl}
				type="text"
				value={query}
				placeholder="Agents, connectors, skills, daemons, stores, pages, models…"
				autocomplete="off"
				spellcheck={false}
				oninput={(e) => {
					cursor = 0;
					onquery((e.currentTarget as HTMLInputElement).value);
				}}
				{onkeydown}
			/>
			<kbd>esc</kbd>
		</div>
		<div class="bh-search-results">
			{#if hits.length === 0}
				<div class="bh-sr-empty">Nothing in the system matches that.</div>
			{:else}
				{#each groups as [kind, list] (kind)}
					<div>
						<div class="bh-sr-group">{KIND_LABEL[kind]}</div>
						{#each list as hit (hit.id)}
							{@const i = hits.indexOf(hit)}
							{@const it = idx.items.get(hit.id)}
							{@const Icon = iconFor(it?.icon || KIND_ICON[hit.kind])}
							<!-- svelte-ignore a11y_click_events_have_key_events -->
							<!-- svelte-ignore a11y_no_static_element_interactions -->
							<div class="bh-sr {i === cursor ? 'is-active' : ''}" onclick={() => onpick(hit.id)} onmouseenter={() => (cursor = i)}>
								<div class="bh-tile" style="--c: var(--bh-k-{hit.kind}, var(--text-2))"><Icon /></div>
								<div class="min-w-0">
									<div class="t">{hit.name}</div>
									<div class="s">{hit.sub}</div>
								</div>
								<span class="k">{hit.path}</span>
							</div>
						{/each}
					</div>
				{/each}
			{/if}
		</div>
	</div>
</div>
