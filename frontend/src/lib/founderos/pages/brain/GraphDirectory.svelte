<!-- The knowledge graph's everything-index (FounderOS v1 components/GraphDirectory.tsx):
     every AI agent, human, SOP and tool in labeled groups, each group header
     in its kind's colour. Clicking a row jumps the graph to that node.
     Collapsible to a slim right-edge rail. With every pillar switched off it
     says so and offers the way back. -->
<script lang="ts">
	import { ChevronLeft, ChevronRight } from '$lib/founderos/icons';
	import KindIcon from './KindIcon.svelte';
	import type { DirectoryGroup } from './types';

	let {
		groups,
		onPick,
		onHover,
		collapsed = false,
		onToggleCollapse,
		onShowAllPillars,
		class: className = ''
	}: {
		groups: DirectoryGroup[];
		onPick: (kind: DirectoryGroup['kind'], id: string) => void;
		/** hovering a row pre-lights its chain on the graph (null on leave) */
		onHover?: (kind: DirectoryGroup['kind'], id: string | null) => void;
		collapsed?: boolean;
		onToggleCollapse?: () => void;
		onShowAllPillars?: () => void;
		class?: string;
	} = $props();

	const GROUP_COLOR: Record<DirectoryGroup['kind'], string> = {
		employee: 'var(--bn-accent)',
		person: 'var(--bn-warn)',
		task: 'var(--bn-brain-1)',
		tool: 'var(--bn-kg-tool)'
	};
	const empty = $derived(groups.every((g) => g.rows.length === 0));
</script>

{#if collapsed}
	<button
		type="button"
		onclick={onToggleCollapse}
		title="Show directory"
		aria-label="Show directory"
		class="bn-pressable bn-dir-rail group flex w-9 shrink-0 flex-col items-center gap-3 rounded-[5px] border py-3 {className}"
		data-part="directory"
	>
		<ChevronLeft size={16} style="width: 16px; height: 16px" class="bn-dim" />
		<span class="bn-dim font-mono text-[10px] uppercase tracking-[0.18em] [writing-mode:vertical-rl]">Directory</span>
	</button>
{:else}
	<div class="bn-dir flex flex-col overflow-hidden rounded-[5px] border {className || 'h-full'}" data-part="directory">
		<div class="bn-dir-head flex items-center justify-between border-b px-3 py-2">
			<span class="bn-dim font-mono text-[10px] uppercase tracking-[0.16em]">Directory</span>
			{#if onToggleCollapse}
				<button type="button" onclick={onToggleCollapse} title="Hide directory" aria-label="Hide directory" class="bn-pressable bn-dim -mr-1 rounded-[5px] p-0.5">
					<ChevronRight size={16} style="width: 16px; height: 16px" />
				</button>
			{/if}
		</div>
		<div class="min-h-0 flex-1 overflow-y-auto px-1.5 pb-2">
			{#if empty}
				<div class="px-2 py-6 text-center">
					<p class="bn-muted font-mono text-[11px]">No pillars selected.</p>
					{#if onShowAllPillars}
						<p class="bn-dim mt-1 font-mono text-[10px]">turn a filter back on, or</p>
						<button
							type="button"
							onclick={onShowAllPillars}
							data-lens="c"
							class="bn-pressable bn-dir-show mt-2 inline-flex h-6 items-center rounded-[6px] border px-2.5 font-mono text-[10px] font-semibold">show all pillars</button
						>
					{/if}
				</div>
			{/if}
			{#each groups as g, gi (g.kind)}
				{#if g.rows.length > 0}
					<div class={gi > 0 ? 'bn-dir-sep mt-2 border-t pt-0.5' : undefined}>
						<div
							class="bn-dir-title sticky top-0 z-10 flex items-center gap-2 px-2 pb-1.5 pt-2.5 font-mono text-[11px] font-semibold uppercase tracking-[0.12em]"
							style="color: {GROUP_COLOR[g.kind]}"
						>
							<KindIcon kind={g.kind} size={14} />
							<span class="flex-1">{g.title}</span>
							<span class="bn-dim">{g.rows.length}</span>
						</div>
						{#each g.rows as r (r.id)}
							<button
								type="button"
								onclick={() => onPick(g.kind, r.id)}
								onmouseenter={() => onHover?.(g.kind, r.id)}
								onmouseleave={() => onHover?.(g.kind, null)}
								data-lens="r"
								title={`Show ${r.label} on the graph`}
								class="bn-pressable is-row bn-dir-row group relative flex w-full items-center gap-2 rounded-[5px] py-1.5 pl-3 pr-2 text-left"
							>
								<span aria-hidden="true" class="bn-dir-accent absolute inset-y-0 left-0 w-[3px]" style="background: {GROUP_COLOR[g.kind]}"></span>
								<span aria-hidden="true" class="h-1.5 w-1.5 shrink-0 rounded-full" style="background: {GROUP_COLOR[g.kind]}"></span>
								<span class="bn-text min-w-0 flex-1 truncate text-[13px] leading-snug">{r.label}</span>
								<span class="bn-dim shrink-0 font-mono text-[10px]">{r.sub}</span>
							</button>
						{/each}
					</div>
				{/if}
			{/each}
		</div>
	</div>
{/if}

<style>
	.bn-dir,
	.bn-dir-rail {
		border-color: var(--bn-border-strong);
		background: color-mix(in oklab, var(--bn-bg) 95%, transparent);
		backdrop-filter: blur(8px);
	}
	.bn-dir-head,
	.bn-dir-sep {
		border-color: var(--bn-border);
	}
	.bn-dir-title {
		background: var(--bn-bg);
	}
	.bn-dir-row:hover {
		background: var(--bn-surface-2);
	}
	.bn-dir-accent {
		opacity: 0;
		transition: opacity 0.15s ease;
	}
	.bn-dir-row:hover .bn-dir-accent {
		opacity: 1;
	}
	.bn-dir-show {
		border-color: var(--bn-border);
		background: var(--bn-bg);
		color: var(--bn-text-2);
	}
</style>
