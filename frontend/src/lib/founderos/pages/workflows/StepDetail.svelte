<!-- The step-detail drawer: owner, presence, tools and the owner's real
     recent runs (FounderOS v1 WorkflowTree.tsx StepDetailPanel). Prod's
     glass-panel-violet / glass-label / glass-chip classes are undefined in
     its CSS, so the drawer has no frame and its labels read as plain 16px
     text; the port draws the same. -->
<script lang="ts">
	import { X } from '$lib/founderos/icons';
	import ToolLogo from './ToolLogo.svelte';
	import StepAvatar from './StepAvatar.svelte';
	import StepIcon from './StepIcon.svelte';
	import type { AgentPresence, AgentRun, WorkflowStep } from './types';
	import { relativeTime, stepTone, toolBrand } from './workflows';

	let {
		step,
		agentPresence,
		runsByOwner,
		onclose
	}: { step: WorkflowStep | null; agentPresence: Record<string, AgentPresence>; runsByOwner: Record<string, AgentRun[]>; onclose: () => void } = $props();

	const tone = $derived(step ? stepTone(step) : null);
	const presence = $derived(step && step.ownerKind === 'agent' ? agentPresence[step.owner] : undefined);
	const runs = $derived(step ? (runsByOwner[step.owner] ?? []) : []);
</script>

{#if !step || !tone}
	<div class="flex min-h-[220px] flex-col items-center justify-center gap-1 px-5 py-8 text-center">
		<span class="bn-dim text-[12px]">Select a step to see its detail.</span>
	</div>
{:else}
	<div data-part="step-detail" role="region" aria-label="{step.title} detail" class="flex min-h-[220px] flex-col gap-4 px-5 py-5">
		<div class="flex items-start justify-between gap-3">
			<div class="flex min-w-0 items-start gap-2.5">
				<span class="grid h-8 w-8 shrink-0 place-items-center rounded-full" style="background: color-mix(in oklab, {tone.color} 18%, transparent); border: 1px solid {tone.color}">
					<StepIcon {step} class="h-3.5 w-3.5" color={tone.color} />
				</span>
				<div class="min-w-0">
					<div class="bn-text text-[13.5px] font-medium leading-snug">{step.title}</div>
					<div class="mt-0.5 text-[11px]" style="color: {tone.color}">{tone.word}</div>
				</div>
			</div>
			<button type="button" aria-label="Close step detail" class="bn-pressable bn-dim grid h-6 w-6 shrink-0 place-items-center hover:text-[var(--bn-text)]" onclick={onclose}><X class="h-3 w-3" strokeWidth={1.8} /></button>
		</div>
		{#if step.branch}<div class="bn-dim inline-flex w-fit items-center gap-1.5 px-2.5 py-1 font-mono text-[10.5px]">runs when: {step.branch.condition}</div>{/if}
		{#if step.detail}<p class="bn-muted text-[12.5px] leading-relaxed">{step.detail}</p>{/if}
		<div>
			<span class="bn-text text-[16px] leading-6">Owner</span>
			<div class="mt-2 flex items-center gap-2">
				<StepAvatar size={22} />
				<span class="bn-text text-[12.5px]">{step.owner}</span>
				<span class="bn-dim text-[11px]">{step.ownerKind === 'human' ? 'human' : 'agent'}</span>
				{#if presence}
					<span class="bn-dim ml-auto flex shrink-0 items-center gap-1.5 text-[10.5px]"><span class="h-1.5 w-1.5 rounded-full" style="background: {presence === 'active' ? 'var(--bn-ok)' : 'var(--bn-warn)'}"></span>{presence === 'active' ? 'live' : 'waiting on keys'}</span>
				{/if}
			</div>
			<div class="bn-dim mt-1.5 font-mono text-[11px]">{step.hoursPerWeek}h / week</div>
		</div>
		{#if step.tools.length > 0}
			<div>
				<span class="bn-text text-[16px] leading-6">Tools</span>
				<div class="mt-2 flex flex-wrap gap-1.5">
					{#each step.tools as t (t)}
						{@const b = toolBrand(t)}
						<span class="bn-muted flex items-center gap-1.5 px-2 py-1 text-[11px]">
							<span class="grid h-3.5 w-3.5 place-items-center"><ToolLogo slug={b.slug} name={b.name} size={12} /></span>
							{b.name}
						</span>
					{/each}
				</div>
			</div>
		{/if}
		<div>
			<span class="bn-text text-[16px] leading-6">Recent runs</span>
			<div class="mt-2 flex flex-col gap-1.5">
				{#if step.ownerKind === 'human'}<div class="bn-dim text-[11.5px]">Human step: no run history.</div>{/if}
				{#if step.ownerKind === 'agent' && runs.length === 0}<div class="bn-dim text-[11.5px]">No runs recorded for {step.owner}.</div>{/if}
				{#each runs as run (run.id)}
					<div class="flex items-center gap-2.5 text-[11.5px]">
						<span class="h-1.5 w-1.5 shrink-0 rounded-full" style="background: {run.ok ? 'var(--bn-ok)' : 'var(--bn-err)'}"></span>
						<span class="bn-dim shrink-0 font-mono text-[10.5px]">{relativeTime(run.startedAt)}</span>
						<span class="bn-muted min-w-0 flex-1 truncate">{run.summary}</span>
					</div>
				{/each}
			</div>
		</div>
	</div>
{/if}
