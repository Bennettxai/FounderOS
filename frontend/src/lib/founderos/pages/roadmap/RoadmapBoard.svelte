<!-- FounderOS v1 components/RoadmapBoard.tsx: the roadmap as one board you can
     actually work. A phase card is a lens row that selects, and its bar is
     done/total of the roadmap rows that phase owns, so selecting a phase
     narrows the quarters to the same rows the percentage was counted from. The
     status chips filter on top of that. An item opens on click into its
     description and its two controls, and "mark done" is a real PATCH. -->
<script lang="ts">
	import { untrack } from 'svelte';
	import { founderosFetch } from '$lib/founderos/api';
	import { Badge, SectionHead, ToggleChip } from '$lib/founderos/kit';
	import OrgAsyncButton from '$lib/founderos/pages/org/OrgAsyncButton.svelte';
	import { FILTERS, STATUS_BADGE, groupRoadmapByQuarter, owned, pctOf, quarterLabel, visibleRows, type Filter } from './roadmap';
	import type { PhaseProgress, RoadmapItem, RoadmapPatchBody, RoadmapStatus } from './types';

	let {
		phases,
		items,
		departments
	}: { phases: PhaseProgress[]; items: RoadmapItem[]; departments: Record<string, string> } = $props();

	// The board is local from here on (v1 useState(items)): marking a row done
	// redraws the bars from the write, not from a refetch.
	let board = $state<RoadmapItem[]>(untrack(() => items));
	let phaseId = $state<string | null>(null);
	let filter = $state<Filter>('All');
	let open = $state<string | null>(null);

	const quarters = $derived(groupRoadmapByQuarter(visibleRows(board, phaseId, filter)));

	async function mark(item: RoadmapItem) {
		const next: RoadmapStatus = item.status === 'done' ? 'now' : 'done';
		const res = await founderosFetch<RoadmapPatchBody>('/pages/roadmap', { method: 'PATCH', json: { id: item.id, status: next } });
		const moved = res?.item ?? { ...item, status: next };
		board = board.map((r) => (r.id === item.id ? { ...r, status: moved.status } : r));
	}

	const togglePhase = (id: string) => (phaseId = phaseId === id ? null : id);
	const toggleOpen = (id: string) => (open = open === id ? null : id);
	const onKey = (e: KeyboardEvent, act: () => void) => {
		if (e.key === 'Enter' || e.key === ' ') {
			e.preventDefault();
			act();
		}
	};
</script>

<div>
	<section class="mb-9">
		<SectionHead label="Phases" count={phases.length} />
		<div class="grid gap-3 md:grid-cols-2 xl:grid-cols-4">
			{#each phases as { phase, items: seeded } (phase.id)}
				{@const rows = owned(board, phase.id)}
				{@const pct = pctOf(board, phase.id)}
				{@const on = phaseId === phase.id}
				{@const doneN = rows.filter((i) => i.status === 'done').length}
				{@const list = rows.length > 0 ? rows : seeded}
				<div
					data-lens="r"
					data-phase={phase.id}
					role="button"
					tabindex="0"
					aria-pressed={on}
					onclick={() => togglePhase(phase.id)}
					onkeydown={(e) => onKey(e, () => togglePhase(phase.id))}
					class="bn-pressable is-row rm-card cursor-pointer rounded-[var(--bn-r-panel)] border px-[17px] py-[15px]"
					class:is-on={on}
				>
					<div class="mb-[7px] flex items-baseline justify-between gap-2">
						<span class="bn-accent font-mono text-[10px] tracking-[0.18em]">PHASE {String(phase.number).padStart(2, '0')}</span>
						<span class="font-mono text-[9.5px] {pct === 100 ? 'rm-ok' : 'bn-dim'}">{pct}% · {doneN}/{rows.length}</span>
					</div>
					<h2 class="bn-text text-sm font-bold">{phase.title}</h2>
					<ul class="mt-2.5 flex flex-col gap-1.5">
						{#each list as it (it.id)}
							<li class="flex items-baseline gap-2 text-[11.5px] {it.status === 'done' ? 'bn-dim rm-strike line-through' : 'bn-muted'}">
								<span class="rm-bullet mt-1 h-1 w-1 shrink-0 rounded-full"></span>
								{it.title}
							</li>
						{/each}
					</ul>
					<div class="rm-track mt-2.5 h-[2px] overflow-hidden rounded-full">
						<span data-part="phase-bar" class="rm-fill block h-full" style="width: {pct}%"></span>
					</div>
				</div>
			{/each}
		</div>
	</section>

	<SectionHead label="Quarter by quarter">
		{#snippet right()}
			<div class="flex shrink-0 gap-1">
				{#each FILTERS as f (f)}
					<ToggleChip on={filter === f} onclick={() => (filter = f)}>{f}</ToggleChip>
				{/each}
			</div>
		{/snippet}
	</SectionHead>

	<div class="grid gap-3.5 md:grid-cols-2 xl:grid-cols-4">
		{#each quarters as { quarter, items: rows } (quarter)}
			{@const doneN = rows.filter((r) => r.status === 'done').length}
			<section>
				<div class="mb-3 flex items-center gap-2.5">
					<span class="bn-text font-mono text-xs font-semibold tracking-[0.12em]">{quarterLabel(quarter)}</span>
					<span data-part="quarter-tally" class="bn-dim font-mono text-[10px]">{doneN}/{rows.length} done</span>
					<span class="rm-rule h-px flex-1"></span>
				</div>
				<div class="flex flex-col gap-2.5">
					{#each rows as item (item.id)}
						{@const badge = STATUS_BADGE[item.status]}
						{@const dept = item.departmentId ? departments[item.departmentId] : null}
						{@const done = item.status === 'done'}
						{@const isOpen = open === item.id}
						<div
							data-lens="r"
							data-item={item.id}
							role="button"
							tabindex="0"
							aria-expanded={isOpen}
							onclick={() => toggleOpen(item.id)}
							onkeydown={(e) => onKey(e, () => toggleOpen(item.id))}
							class="bn-pressable is-row rm-card cursor-pointer rounded-[var(--bn-r-panel)] border px-[15px] py-3 {done ? 'opacity-[0.62]' : ''}"
							class:is-open={isOpen}
						>
							<div class="flex items-start justify-between gap-2.5">
								<div class="text-[12.5px] font-semibold leading-snug {done ? 'bn-muted rm-strike line-through' : 'bn-text'}">{item.title}</div>
								<Badge tone={badge.tone} ghost={badge.ghost}>{badge.label}</Badge>
							</div>
							{#if isOpen}
								<p class="bn-enter bn-dim mt-1.5 text-[11px] leading-relaxed [text-wrap:pretty]">{item.description}</p>
								<!-- svelte-ignore a11y_click_events_have_key_events, a11y_no_static_element_interactions -->
								<div class="bn-enter mt-2 flex gap-1.5" onclick={(e) => e.stopPropagation()}>
									<button
										type="button"
										data-lens="c"
										class="bn-pressable is-dark rm-ctl inline-flex h-[22px] items-center rounded-[var(--bn-r-ctl)] border px-2 font-mono text-[10px] font-semibold"
									>
										open task
									</button>
									<OrgAsyncButton run={() => mark(item)} tone="secondary" busyLabel="saving" doneLabel="saved">
										{done ? 'mark now' : 'mark done'}
									</OrgAsyncButton>
								</div>
							{/if}
							{#if dept}
								<div class="bn-muted mt-2.5 flex items-center gap-1.5 font-mono text-[9.5px]">
									<span class="rm-dept h-[5px] w-[5px] rounded-sm"></span>
									{dept}
								</div>
							{/if}
						</div>
					{/each}
				</div>
			</section>
		{/each}
	</div>
</div>

<style>
	.rm-card {
		border-color: var(--bn-border);
		background: var(--bn-surface);
	}
	.rm-card.is-on {
		border-color: var(--bn-accent-line);
		background: var(--bn-surface-2);
	}
	.rm-card.is-open {
		border-color: var(--bn-border-strong);
	}
	.rm-ok {
		color: var(--bn-ok);
	}
	.rm-strike {
		text-decoration-color: var(--bn-text-3);
	}
	.rm-bullet {
		background: var(--bn-text-3);
	}
	.rm-track,
	.rm-rule {
		background: var(--bn-border);
	}
	.rm-fill {
		background: var(--bn-text);
		transition: width 900ms cubic-bezier(0.22, 0.61, 0.36, 1);
	}
	.rm-ctl {
		border-color: var(--bn-border);
		background: var(--bn-bg);
		color: var(--bn-text-2);
	}
	.rm-dept {
		background: var(--bn-accent);
	}
</style>
