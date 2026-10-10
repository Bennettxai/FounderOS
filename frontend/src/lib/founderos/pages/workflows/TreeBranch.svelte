<!-- One node of the expanded workflow tree, and its children (fans out on
     real branches). FounderOS v1 WorkflowTree.tsx TreeBranch: a rounded glass
     node, a round lucide glyph, the owner's avatar, logo tool chips and a
     status hairline. -->
<script lang="ts">
	import ToolLogo from './ToolLogo.svelte';
	import StepAvatar from './StepAvatar.svelte';
	import StepIcon from './StepIcon.svelte';
	import TreeBranch from './TreeBranch.svelte';
	import type { AgentPresence } from './types';
	import { stepTone, toolBrand, type WorkflowTreeNode } from './workflows';

	let {
		node,
		agentPresence,
		selectedStepId,
		onselect
	}: { node: WorkflowTreeNode; agentPresence: Record<string, AgentPresence>; selectedStepId: string | null; onselect: (id: string | null) => void } = $props();

	const step = $derived(node.step);
	const tone = $derived(stepTone(step));
	const presence = $derived(step.ownerKind === 'agent' ? agentPresence[step.owner] : undefined);
	const note = $derived(step.automation ? step.automation.title : step.ownerKind === 'human' ? `${step.hoursPerWeek}h/week, no automation attached` : 'no automation attached');
	const selected = $derived(selectedStepId === step.id);
	const fan = $derived(node.children.length > 1);
</script>

<div class="flex flex-col items-stretch">
	<button
		type="button"
		data-step={step.id}
		data-lens="r"
		aria-pressed={selected}
		aria-label="{step.title}: view step detail"
		class="bn-pressable wft-node flex w-full items-start gap-3 px-3.5 py-3 text-left"
		style={selected ? `border-color: ${tone.color}` : undefined}
		onclick={() => onselect(selected ? null : step.id)}
	>
		<span class="grid h-9 w-9 shrink-0 place-items-center rounded-full" style="background: color-mix(in oklab, {tone.color} 18%, transparent); border: 1px solid {tone.color}">
			<StepIcon {step} class="h-4 w-4" color={tone.color} />
		</span>
		<div class="min-w-0 flex-1">
			<span class="bn-text block truncate text-[13px] font-medium">{step.title}</span>
			<div class="mt-1.5 flex items-center gap-1.5">
				<StepAvatar size={16} />
				<span class="bn-dim truncate text-[11px]">{step.owner}</span>
				{#if presence}
					<span class="h-1.5 w-1.5 shrink-0 rounded-full" title={presence === 'active' ? 'agent live' : 'agent waiting on keys'} style="background: {presence === 'active' ? 'var(--bn-ok)' : 'var(--bn-warn)'}"></span>
				{/if}
			</div>
			<div class="mt-1.5 truncate text-[10.5px]"><span style="color: {tone.color}">{tone.word}</span><span class="bn-dim">{` · ${note}`}</span></div>
			{#if step.tools.length > 0}
				<div class="mt-2 flex flex-wrap items-center gap-1.5">
					{#each step.tools as t (t)}
						{@const b = toolBrand(t)}
						<span class="bn-dim flex items-center gap-1 px-1.5 py-[3px] text-[10px]">
							<span class="grid h-3 w-3 place-items-center"><ToolLogo slug={b.slug} name={b.name} size={12} /></span>
							{b.name}
						</span>
					{/each}
				</div>
			{/if}
		</div>
		<span aria-hidden="true" class="absolute inset-x-3 bottom-0 h-[2px] rounded-full" style="background: {tone.color}; opacity: 0.4"></span>
	</button>

	{#if node.children.length > 0}
		<span class="wft-line"></span>
		<div class={fan ? 'flex items-start gap-4' : 'flex flex-col items-stretch'}>
			{#each node.children as child (child.step.id)}
				<div class={fan ? 'relative min-w-0 flex-1' : 'flex flex-col items-stretch'}>
					{#if fan}<span class="wft-line"></span>{/if}
					{#if child.step.branch}
						<div class="mb-1.5 flex justify-center"><span class="bn-dim px-2 py-[2px] font-mono text-[9.5px]">if {child.step.branch.condition}</span></div>
					{/if}
					<TreeBranch node={child} {agentPresence} {selectedStepId} {onselect} />
				</div>
			{/each}
		</div>
	{/if}
</div>

<style>
	.wft-line {
		display: block;
		width: 1px;
		height: 16px;
		margin: 3px auto;
		background: rgba(255, 255, 255, 0.12);
	}
	.wft-node {
		position: relative;
		border-radius: 14px;
		border: 1px solid color-mix(in oklab, var(--bn-text) 10%, transparent);
		background: linear-gradient(
			165deg,
			color-mix(in oklab, var(--bn-surface-2) 55%, transparent) 0%,
			color-mix(in oklab, var(--bn-bg) 65%, transparent) 55%,
			color-mix(in oklab, var(--bn-bg) 82%, transparent) 100%
		);
		backdrop-filter: blur(10px);
	}
	.wft-node:hover,
	.wft-node:focus-visible {
		border-color: color-mix(in oklab, var(--bn-text) 28%, transparent);
		background: linear-gradient(
			165deg,
			color-mix(in oklab, var(--bn-surface-2) 70%, transparent) 0%,
			color-mix(in oklab, var(--bn-bg) 75%, transparent) 55%,
			color-mix(in oklab, var(--bn-bg) 90%, transparent) 100%
		);
	}
	.wft-node:focus-visible {
		outline: 2px solid var(--bn-accent);
		outline-offset: 2px;
	}
</style>
