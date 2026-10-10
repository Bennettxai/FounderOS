<!-- Agent harness card (FounderOS v1 KnowledgeDetail AgentHarnessCard): the
     agent's charter, the SOP it executes, where it sits in the harness, its
     tools and its last run. -->
<script lang="ts">
	import { FileText, ListChecks, Server, Wrench } from '$lib/founderos/icons';
	import PanelHeader from './PanelHeader.svelte';
	import SectionLabel from './SectionLabel.svelte';
	import WikiLink from './WikiLink.svelte';
	import { prettifySlug } from '../kg';
	import type { SopTask } from '../types';

	let {
		agent,
		task,
		parentName,
		parentAgentId,
		subAgents,
		lastRun,
		runLabel = null,
		headName = null,
		onBack,
		onClose,
		onTool,
		onAgent,
		onTask
	}: {
		agent: { name: string; role: string; model: string; tier: string; instance: string; description: string; tools: string[] };
		task: SopTask | null;
		parentName: string | null;
		parentAgentId: string | null;
		subAgents: { id: string; name: string }[];
		lastRun: { ok: boolean; summary: string } | null;
		runLabel?: string | null;
		headName?: string | null;
		onBack?: () => void;
		onClose?: () => void;
		onTool?: (slug: string) => void;
		onAgent?: (agentId: string) => void;
		onTask?: () => void;
	} = $props();
</script>

<div class="flex h-full flex-col" data-card="agent">
	<PanelHeader title={agent.name} sub={`${agent.role} · AI agent`} {onBack} {onClose} />
	<div class="min-h-0 flex-1 overflow-y-auto overscroll-contain px-3 py-2.5">
		<p class="bn-muted mb-3 text-[11px] leading-relaxed">{agent.description}</p>

		<SectionLabel icon={ListChecks}>instructions{task ? '' : ' · no SOP assigned'}</SectionLabel>
		{#if task}
			<button type="button" onclick={onTask} disabled={!onTask} class="bn-pressable mb-1 text-left {onTask ? 'kg-h-o80' : ''}">
				<span class="kg-accent font-mono text-[10.5px]">[[{task.title}]]</span>
			</button>
		{/if}
		<ol class="mb-4 flex flex-col gap-1.5">
			{#each task?.steps ?? [] as s, i (i)}
				<li class="bn-muted flex gap-2 text-[11px] leading-relaxed">
					<span class="bn-dim shrink-0 font-mono text-[10px]">{String(i + 1).padStart(2, '0')}</span>
					{s}
				</li>
			{/each}
		</ol>

		<SectionLabel icon={Server}>harness</SectionLabel>
		<div class="bn-muted mb-4 flex flex-col gap-1 font-mono text-[10.5px]">
			<span><span class="bn-dim">tier</span> {agent.tier} · <span class="bn-dim">runs on</span> {agent.instance} · {agent.model}</span>
			{#if parentName}
				<span>
					<span class="bn-dim">reports to</span>
					{#if parentAgentId && onAgent}
						<button type="button" onclick={() => onAgent?.(parentAgentId)} class="bn-pressable kg-accent kg-h-o80">{parentName}</button>
					{:else}{parentName}{/if}
				</span>
			{/if}
			{#if headName}<span><span class="bn-dim">human lead</span> {headName}</span>{/if}
			{#if subAgents.length > 0}
				<span class="flex flex-wrap items-center gap-x-2">
					<span class="bn-dim">sub-agents</span>
					{#each subAgents as s (s.id)}
						{#if onAgent}
							<button type="button" onclick={() => onAgent?.(s.id)} class="bn-pressable kg-accent kg-h-o80">{s.name}</button>
						{:else}<span>{s.name}</span>{/if}
					{/each}
				</span>
			{/if}
		</div>

		<SectionLabel icon={Wrench}>tools ({agent.tools.length})</SectionLabel>
		<div class="mb-4 flex flex-wrap gap-x-3 gap-y-1.5">
			{#each agent.tools as slug (slug)}
				<WikiLink label={prettifySlug(slug)} onclick={onTool ? () => onTool?.(slug) : undefined} />
			{/each}
			{#if agent.tools.length === 0}<span class="bn-dim font-mono text-[10.5px]">no tools wired</span>{/if}
		</div>

		<SectionLabel icon={FileText}>last run</SectionLabel>
		{#if lastRun}
			<div class="flex items-start gap-2" data-part="last-run">
				<span class="mt-1 h-2 w-2 shrink-0 rounded-full" style="background: {lastRun.ok ? 'var(--bn-ok)' : 'var(--bn-err)'}"></span>
				<p class="bn-muted text-[11px] leading-relaxed">
					{lastRun.summary}{#if runLabel}<span class="bn-dim font-mono text-[9.5px]"> · {runLabel}</span>{/if}
				</p>
			</div>
		{:else}
			<p class="bn-dim font-mono text-[10.5px]" data-part="last-run">never run — trigger it from /agents</p>
		{/if}
	</div>
</div>
