<!-- One drawn thing: a card, a compact card, a hero, the operator's start
     circle, a routing diamond, an external (cloud) card, or a collapsed group
     shown as its count card. Position comes in as a transform; the kind
     colour rides --c, so every glow, border and tile follows the legend.
     Port of FounderOS v1 components/blueprint/HierarchyNode.tsx. -->
<script lang="ts">
	import { ChevronDown } from '$lib/founderos/icons';
	import { KIND_ICON, type HGroup, type HItem } from './hierarchy';
	import type { Box } from './hierarchy-layout';
	import { iconFor } from './icons';

	let {
		box,
		className = '',
		enterDelay,
		onanimationend
	}: { box: Box; className?: string; enterDelay?: string; onanimationend?: (e: AnimationEvent) => void } = $props();

	const iconOf = (it: HItem) => iconFor(it.icon || KIND_ICON[it.kind] || 'box');
	const countLeaves = (g: HGroup): number => g.children.reduce((a, c) => a + (c.type === 'group' ? countLeaves(c) : 1), 0);

	const it = $derived(box.it as HItem);
	const shape = $derived(box.collapsed ? 'group' : it.type === 'node' ? it.shape : 'card');
	const status = $derived(it.type === 'node' ? it.status : (it.status ?? ''));
	const Icon = $derived(iconOf(it));
	const style = $derived(
		`width: ${box.w}px; height: ${box.h}px; transform: translate(${box.x}px, ${box.y}px); --tx: ${box.x}px; --ty: ${box.y}px; --c: var(--bh-k-${it.kind}, var(--text-2));` +
			(enterDelay ? ` --d: ${enterDelay};` : '')
	);
</script>

<div
	class="bh-node {shape} k-{it.kind} {className}"
	data-id={box.id}
	data-s={status}
	data-mode={box.collapsed ? 'g' : 'n'}
	{style}
	{onanimationend}
>
	{#if box.collapsed && it.type === 'group'}
		<div class="bh-card">
			<div class="bh-tile"><Icon /></div>
			<div class="bh-ttl">
				<div class="t">{it.name}</div>
				<div class="s">{it.sub}</div>
			</div>
			<span class="bh-stack">
				{#each it.children.slice(0, 4) as c (c.id)}
					{@const CI = iconOf(c)}
					<i><CI /></i>
				{/each}
			</span>
			<span class="bh-count">{countLeaves(it)}</span>
			<span class="bh-chev"><ChevronDown /></span>
		</div>
	{:else if shape === 'start' || shape === 'diamond'}
		<div class="bh-card"><Icon /></div>
		<div class="bh-label">
			<div>{it.name}</div>
			<div class="s">{it.sub}</div>
		</div>
	{:else}
		<div class="bh-card">
			<div class="bh-tile"><Icon /></div>
			<div class="bh-ttl">
				<div class="t">{it.name}</div>
				<div class="s">{it.sub}</div>
			</div>
			<span class="bh-st" data-s={status}></span>
		</div>
	{/if}
</div>
