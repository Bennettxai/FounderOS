<!-- The process map (FounderOS v1 components/WorkflowTree.tsx): collapsed
     workflow cards that expand into the real tree, a step-detail drawer and
     the builder. Prod styles the card, its chips and the New workflow button
     with GladOS classes (glass-panel, glass-chip, c-btn) that its CSS never
     defined, so they render bare: no frame, logo-only chips, and a 16px
     button with the plus above the words. The port draws the same. -->
<script lang="ts">
	import { Pencil, Plus, X } from '$lib/founderos/icons';
	import ToolLogo from './ToolLogo.svelte';
	import StepDetail from './StepDetail.svelte';
	import TreeBranch from './TreeBranch.svelte';
	import WorkflowBuilder from './WorkflowBuilder.svelte';
	import type { AgentPresence, AgentRun, SimpleAgent, Workflow } from './types';
	import { buildWorkflowTree, stepTone, toolBrand, workflowStats, workflowStepParent, workflowToolIds } from './workflows';

	let {
		workflows,
		agentPresence,
		agents,
		runsByOwner,
		onchange
	}: { workflows: Workflow[]; agentPresence: Record<string, AgentPresence>; agents: SimpleAgent[]; runsByOwner: Record<string, AgentRun[]>; onchange?: () => void } = $props();

	let expandedId = $state<string | null>(null);
	let selectedStepId = $state<string | null>(null);
	let builder = $state<'new' | Workflow | null>(null);

	function open(id: string | null) {
		expandedId = id;
		selectedStepId = null;
	}

	$effect(() => {
		if (!expandedId) return;
		const onKey = (e: KeyboardEvent) => {
			if (e.key !== 'Escape' || builder) return;
			if (selectedStepId) selectedStepId = null;
			else expandedId = null;
		};
		// prod: a click anywhere outside the open card collapses it, in the
		// capture phase so clicking another card swaps the expansion cleanly
		const onClick = (e: MouseEvent) => {
			if (builder) return;
			const card = document.querySelector(`[data-workflow-card="${CSS.escape(expandedId!)}"]`);
			if (card && !card.contains(e.target as Node)) expandedId = null;
		};
		document.addEventListener('keydown', onKey);
		document.addEventListener('click', onClick, true);
		return () => {
			document.removeEventListener('keydown', onKey);
			document.removeEventListener('click', onClick, true);
		};
	});
</script>

<div data-part="process-map">
	<div class="mb-4 flex items-center justify-between gap-3">
		<span class="bn-dim text-[12.5px]">{workflows.length} workflow{workflows.length === 1 ? '' : 's'} mapped</span>
		<button type="button" class="bn-pressable bn-text block text-[16px] leading-6" onclick={() => (builder = 'new')}><Plus class="block h-[13px] w-[13px]" strokeWidth={2} /> New workflow</button>
	</div>

	{#if workflows.length === 0}
		<p class="bn-dim border px-4 py-6 font-mono text-[11px]" style="border-color: var(--bn-border)">No workflows mapped yet. Hit New workflow to map one.</p>
	{/if}

	<div class="grid grid-cols-1 gap-5 md:grid-cols-2 xl:grid-cols-3">
		{#each workflows as wf (wf.id)}
			{@const expanded = expandedId === wf.id}
			{@const stats = workflowStats(wf)}
			{@const toolIds = workflowToolIds(wf.steps)}
			{@const selected = expanded ? (wf.steps.find((s) => s.id === selectedStepId) ?? null) : null}
			<!-- svelte-ignore a11y_no_noninteractive_tabindex -->
			<div
				data-workflow-card={wf.id}
				aria-expanded={expanded}
				role={expanded ? undefined : 'button'}
				tabindex={expanded ? undefined : 0}
				data-lens={expanded ? undefined : 'r'}
				class="group relative flex flex-col overflow-hidden {expanded ? 'col-span-full' : 'bn-pressable is-row cursor-pointer'}"
				onclick={expanded ? undefined : () => open(wf.id)}
				onkeydown={expanded
					? undefined
					: (e) => {
							if (e.key === 'Enter' || e.key === ' ') {
								e.preventDefault();
								open(wf.id);
							}
						}}
			>
				<div class="flex items-start justify-between gap-3 px-5 pt-4">
					<div class="min-w-0">
						<div class="bn-text truncate text-[15px] font-medium">{wf.name}</div>
						<div class="bn-dim mt-1 truncate font-mono text-[11px]">trigger: {wf.steps[0]?.title ?? 'no steps recorded'}</div>
					</div>
					<div class="flex shrink-0 items-center gap-2">
						<button
							type="button"
							aria-label="Edit {wf.name}"
							class="bn-pressable bn-dim grid h-7 w-7 place-items-center transition-opacity hover:text-[var(--bn-text)] {expanded ? '' : 'opacity-0 focus-visible:opacity-100 group-hover:opacity-100'}"
							onclick={(e) => {
								e.stopPropagation();
								builder = wf;
							}}><Pencil class="h-3.5 w-3.5" strokeWidth={1.8} /></button
						>
						{#if expanded}<button
								type="button"
								aria-label="Collapse workflow"
								class="bn-pressable bn-dim grid h-7 w-7 place-items-center hover:text-[var(--bn-text)]"
								onclick={(e) => {
									e.stopPropagation();
									open(null);
								}}><X class="h-3.5 w-3.5" strokeWidth={1.8} /></button
							>{/if}
					</div>
				</div>

				{#if !expanded}
					<div class="mt-4 overflow-x-auto px-5">
						{#if wf.steps.length === 0}
							<div class="bn-dim pb-1 text-[11px]">no steps recorded</div>
						{:else}
							<div class="flex min-w-max items-center pb-1">
								{#each wf.steps as s, i (s.id)}
									{@const tone = stepTone(s)}
									{#if i > 0}<span class="h-px w-[14px] shrink-0" style="background: rgba(255,255,255,0.12)"></span>{/if}
									<span data-part="mini-dot" class="shrink-0 rounded-full" title={s.title} style="width: {i === 0 ? 9 : 6}px; height: {i === 0 ? 9 : 6}px; border: 1px solid {tone.color}; background: {s.ownerKind === 'agent' && s.automation?.state === 'live' ? tone.color : 'transparent'}"></span>
								{/each}
							</div>
						{/if}
					</div>
					<div class="mt-4 flex items-center gap-2 border-t px-5 py-3" style="border-color: var(--bn-border)">
						<span class="bn-dim shrink-0 font-mono text-[10.5px]">{wf.steps.length} step{wf.steps.length === 1 ? '' : 's'}</span>
						{#if toolIds.length > 0}
							<span class="shrink-0" style="color: var(--bn-border)">·</span>
							<div class="flex min-w-0 flex-1 items-center gap-1.5 overflow-hidden">
								{#each toolIds.slice(0, 3) as t (t)}
									{@const b = toolBrand(t)}
									<span data-part="tool-chip" class="flex shrink-0 items-center gap-1 px-1.5 py-[3px]" title={b.name}>
										<span class="grid h-3.5 w-3.5 place-items-center"><ToolLogo slug={b.slug} name={b.name} size={12} /></span>
									</span>
								{/each}
								{#if toolIds.length > 3}<span class="bn-dim shrink-0 px-1.5 py-[3px] font-mono text-[10px]">+{toolIds.length - 3}</span>{/if}
							</div>
						{/if}
					</div>
				{:else}
					{@const tree = buildWorkflowTree(wf.steps, workflowStepParent)}
					<div data-part="expanded" class="wft-expand flex flex-col gap-5 px-5 pb-5">
						<div class="bn-dim flex flex-wrap items-center gap-x-4 gap-y-1 font-mono text-[10.5px]">
							<span>{stats.agentSteps}/{wf.steps.length} agent-run</span><span>·</span>
							<span><span style="color: var(--bn-accent)">{stats.live} live</span>{#if stats.planned > 0}<span style="color: var(--bn-warn)">{` · ${stats.planned} planned`}</span>{/if}</span>
							{#if stats.manualHours + stats.agentHours > 0}<span>·</span><span>{stats.manualHours}h human / {stats.agentHours}h agent · wk</span>{/if}
						</div>
						<div class="grid gap-5 lg:grid-cols-[minmax(0,1fr)_320px]">
							<div class="overflow-x-auto pb-1">
								<div class="mx-auto min-w-[300px] max-w-[440px]">
									{#if tree.root}
										<TreeBranch node={tree.root} {agentPresence} {selectedStepId} onselect={(id) => (selectedStepId = id)} />
									{:else}
										<div class="bn-dim text-[11px]">no steps recorded</div>
									{/if}
								</div>
							</div>
							<StepDetail step={selected} {agentPresence} {runsByOwner} onclose={() => (selectedStepId = null)} />
						</div>
					</div>
				{/if}
			</div>
		{/each}
	</div>

	{#if builder}
		<WorkflowBuilder {agents} workflow={builder === 'new' ? null : builder} onclose={() => (builder = null)} onsaved={onchange} />
	{/if}
</div>

<style>
	.wft-expand {
		animation: wft-in 0.26s var(--bn-ease, ease-out) both;
	}
	@keyframes wft-in {
		from {
			opacity: 0;
			transform: translateY(-6px);
		}
	}
	@media (prefers-reduced-motion: reduce) {
		.wft-expand {
			animation: none;
		}
	}
</style>
