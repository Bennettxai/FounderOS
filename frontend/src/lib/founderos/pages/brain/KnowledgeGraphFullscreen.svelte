<!-- The fullscreen department wheel (FounderOS v1 components/KnowledgeGraphFullscreen.tsx):
     the same graph over the whole screen, opened straight into a pillar's
     tree. Top-left pillar picker (name, n / N, one dot per pillar), big
     department title top-center, Exit top-right, a resizable directory docked
     right, the compact legend bottom-left, the ← / → stepper bottom-center and
     a floating detail card on the left. Escape backs out of a card, then the
     open core, then fullscreen; ← / → turn the wheel. Portalled to <body>. -->
<script lang="ts">
	import { onMount, type Snippet } from 'svelte';
	import { ArrowLeft, ChevronLeft, ChevronRight, Minimize2 } from '$lib/founderos/icons';

	type Dept = { teamId: string; deptId: string; name: string; color: string; tagline: string };
	let {
		deptList,
		currentTeamId,
		currentDept,
		hasDetail = false,
		coreOpen = false,
		onCollapseCore,
		hasSearch = false,
		searchSlot,
		legendSlot,
		directorySlot,
		directoryCollapsed = false,
		onNavDept,
		onBack,
		onClose,
		detail,
		children
	}: {
		deptList: Dept[];
		currentTeamId: string | null;
		currentDept: Dept | null;
		hasDetail?: boolean;
		coreOpen?: boolean;
		onCollapseCore?: () => void;
		/** the vault search chip is live (the core is open) */
		hasSearch?: boolean;
		searchSlot?: Snippet;
		legendSlot?: Snippet;
		directorySlot?: Snippet;
		directoryCollapsed?: boolean;
		onNavDept: (teamId: string) => void;
		onBack: () => void;
		onClose: () => void;
		detail?: Snippet;
		children: Snippet;
	} = $props();

	const idx = $derived(deptList.findIndex((d) => d.teamId === currentTeamId));
	const DIR_MIN = 288;
	const DIR_MAX = 680;
	let dirWidth = $state(380);
	let resize: { startX: number; startW: number } | null = null;
	function onResizeDown(e: PointerEvent) {
		e.preventDefault();
		(e.currentTarget as Element).setPointerCapture?.(e.pointerId);
		resize = { startX: e.clientX, startW: dirWidth };
	}
	function onResizeMove(e: PointerEvent) {
		if (!resize) return;
		dirWidth = Math.max(DIR_MIN, Math.min(DIR_MAX, resize.startW + (resize.startX - e.clientX)));
	}
	function onResizeUp(e: PointerEvent) {
		resize = null;
		(e.currentTarget as Element).releasePointerCapture?.(e.pointerId);
	}
	function step(dir: number) {
		if (deptList.length === 0) return;
		const next = idx < 0 ? (dir > 0 ? 0 : deptList.length - 1) : (idx + dir + deptList.length) % deptList.length;
		onNavDept(deptList[next].teamId);
	}
	function onKey(e: KeyboardEvent) {
		const t = e.target as HTMLElement | null;
		if (t?.tagName === 'INPUT' || t?.tagName === 'TEXTAREA' || t?.isContentEditable) return;
		if (e.key === 'Escape') {
			if (hasDetail) onBack();
			else if (coreOpen) onCollapseCore?.();
			else onClose();
		} else if (e.key === 'ArrowLeft') step(-1);
		else if (e.key === 'ArrowRight') step(1);
	}
	onMount(() => {
		const prev = document.body.style.overflow;
		document.body.style.overflow = 'hidden';
		return () => {
			document.body.style.overflow = prev;
		};
	});
	/** render into <body>, clear of every page stacking context */
	function portal(node: HTMLElement) {
		document.body.appendChild(node);
		return { destroy: () => node.remove() };
	}
</script>

<svelte:window onkeydown={onKey} />

<div use:portal class="kg-bg fixed inset-0 z-[140] flex" data-part="kg-fullscreen" role="dialog" aria-label="Department wheel">
	<div class="kg-surf relative min-w-0 flex-1 overflow-hidden">
		{@render children()}

		{#if hasSearch && searchSlot}<div class="absolute left-5 top-5 z-10">{@render searchSlot()}</div>{/if}

		{#if !coreOpen && currentDept}
			<div class="pointer-events-none absolute left-1/2 top-4 z-30 -translate-x-1/2" data-part="fs-title">
				<span class="text-[24px] font-bold uppercase leading-none tracking-[0.08em]" style="color: #ffffff; text-shadow: 0 1px 8px rgba(0,0,0,0.75)">{currentDept.name}</span>
			</div>
		{/if}

		{#if !coreOpen && !hasDetail}
			<div class="kg-bs kg-bg85 kg-blur absolute left-5 top-5 z-20 flex items-center gap-2.5 rounded-[5px] border px-2.5 py-1.5" data-part="fs-picker">
				<div class="flex flex-col">
					<span class="kg-fade-state bn-text max-w-[150px] truncate text-[12.5px] font-bold leading-tight" style={currentDept ? `color: ${currentDept.color}` : undefined}>{currentDept?.name ?? 'Pick a pillar'}</span>
					<span class="bn-dim font-mono text-[8.5px] uppercase tracking-[0.14em]">{idx >= 0 ? `${idx + 1} / ${deptList.length}` : `${deptList.length} pillars`}</span>
				</div>
				<div class="kg-b flex items-center gap-0.5 border-l pl-2">
					{#each deptList as d (d.teamId)}
						{@const active = d.teamId === currentTeamId}
						<button type="button" onclick={() => onNavDept(d.teamId)} title={d.name} aria-label={d.name} aria-current={active ? 'true' : undefined} class="bn-pressable group grid h-6 w-5 place-items-center">
							<span
								class="kg-dot rounded-full {active ? 'h-3 w-3' : 'h-2.5 w-2.5 opacity-50'}"
								style="background: {active ? d.color : 'var(--bn-text-3)'}; {active ? `box-shadow: 0 0 8px ${d.color}` : ''}"
							></span>
						</button>
					{/each}
				</div>
			</div>
		{/if}

		{#if legendSlot}<div class="absolute bottom-5 left-5 z-10">{@render legendSlot()}</div>{/if}

		{#if directorySlot}
			<div class="absolute bottom-5 right-5 top-16 z-[5] flex" style="width: {directoryCollapsed ? 36 : dirWidth}px" data-part="fs-directory">
				{#if !directoryCollapsed}
					<div
						onpointerdown={onResizeDown}
						onpointermove={onResizeMove}
						onpointerup={onResizeUp}
						role="separator"
						aria-orientation="vertical"
						aria-label="Drag to resize the directory"
						aria-valuenow={dirWidth}
						tabindex="-1"
						title="Drag to resize"
						class="group absolute -left-2 top-0 z-10 flex h-full w-4 cursor-ew-resize touch-none items-center justify-center"
					>
						<span class="kg-handle kg-fade-state h-12 w-0.5 rounded-full"></span>
					</div>
				{/if}
				<div class="min-w-0 flex-1">{@render directorySlot()}</div>
			</div>
		{/if}

		<button type="button" onclick={onClose} class="bn-pressable kg-b kg-surf bn-muted kg-h-bs kg-h-text absolute right-5 top-5 z-20 flex items-center gap-1.5 rounded-[5px] border px-2.5 py-1 font-mono text-[11px]" data-part="fs-exit">
			<Minimize2 size={14} /> Exit
		</button>

		{#if !coreOpen}
			<div class="kg-bs kg-bg90 kg-blur absolute bottom-5 left-1/2 z-40 flex -translate-x-1/2 items-center gap-1 rounded-full border px-1.5 py-1.5" data-part="fs-stepper">
				<button type="button" onclick={() => step(-1)} aria-label="Previous department" title="Previous pillar (←)" class="bn-pressable bn-muted kg-h-surf kg-h-text flex h-10 w-10 items-center justify-center rounded-full"><ChevronLeft size={20} /></button>
				<span class="bn-text min-w-[96px] px-1 text-center font-mono text-[11px] font-semibold leading-none" style={currentDept ? `color: ${currentDept.color}` : undefined}>{currentDept?.name ?? 'All pillars'}</span>
				<button type="button" onclick={() => step(1)} aria-label="Next department" title="Next pillar (→)" class="bn-pressable bn-muted kg-h-surf kg-h-text flex h-10 w-10 items-center justify-center rounded-full"><ChevronRight size={20} /></button>
			</div>
		{/if}
	</div>

	{#if hasDetail}
		<aside class="kg-fs-detail kg-bs kg-bg95 kg-blur absolute bottom-6 left-4 top-16 z-30 flex w-[400px] flex-col overflow-hidden rounded-[10px] border" data-part="fs-detail">
			<button
				type="button"
				onclick={onBack}
				aria-label={`Back to the ${currentDept?.name ?? 'graph'} pillar`}
				class="bn-pressable kg-b bn-dim kg-h-text flex shrink-0 items-center gap-1.5 border-b px-3 py-2 text-left font-mono text-[10px] uppercase tracking-[0.14em]"
			>
				<span class="shrink-0"><ArrowLeft size={12} /></span>
				<span class="truncate">Back · <span style={currentDept ? `color: ${currentDept.color}` : undefined}>{currentDept?.name ?? 'graph'}</span></span>
			</button>
			<div class="min-h-0 flex-1 overflow-hidden">{@render detail?.()}</div>
		</aside>
	{/if}
</div>

<style>
	.kg-dot {
		transition:
			width 0.2s,
			height 0.2s,
			opacity 0.2s,
			transform 0.2s;
	}
	.group:hover .kg-dot {
		transform: scale(1.25);
		opacity: 1;
	}
	.kg-handle {
		background: var(--bn-border-strong);
	}
	.group:hover .kg-handle {
		background: var(--bn-accent);
	}
	.kg-fs-detail {
		box-shadow: var(--bn-shadow-lift);
	}
	@media (max-width: 860px) {
		.kg-fs-detail {
			left: 0.5rem;
			right: 0.5rem;
			bottom: 0.5rem;
			top: auto;
			height: 70vh;
			width: auto;
		}
	}
</style>
