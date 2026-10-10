<!-- A person in the process (FounderOS v1 KnowledgeDetail GraphHumanDetailCard): their one job and tools. -->
<script lang="ts">
	import { ClipboardList, Wrench } from '$lib/founderos/icons';
	import DetailRow from './DetailRow.svelte';
	import PanelHeader from './PanelHeader.svelte';
	import SectionLabel from './SectionLabel.svelte';
	import WikiLink from './WikiLink.svelte';
	import type { Person, SopTask } from '../types';

	let {
		person,
		deptName,
		color,
		task,
		tools,
		onBack,
		onClose,
		onTask,
		onTool
	}: {
		person: Person;
		deptName: string;
		color: string;
		task: SopTask | null;
		tools: { slug: string; name: string; mcp: boolean }[];
		onBack?: () => void;
		onClose?: () => void;
		onTask?: () => void;
		onTool?: (slug: string) => void;
	} = $props();
</script>

<div class="flex h-full flex-col" data-card="human">
	<PanelHeader title={person.name} sub={`${person.role} · human`} {color} {onBack} {onClose} />
	<div class="min-h-0 flex-1 overflow-y-auto overscroll-contain px-3 py-2.5">
		<p class="bn-muted mb-3 text-[11px] leading-relaxed">Human employee in <span class="font-semibold">{deptName}</span> — one job, done by them alone.</p>
		<SectionLabel icon={ClipboardList}>their job</SectionLabel>
		<div class="mb-4">
			{#if task}
				<DetailRow {color} title={task.title} sub={task.summary} badge="sop" onclick={onTask} />
			{:else}
				<p class="bn-dim font-mono text-[10.5px]">no task assigned</p>
			{/if}
		</div>
		<SectionLabel icon={Wrench}>works with ({tools.length})</SectionLabel>
		<div class="flex flex-wrap gap-x-3 gap-y-1.5">
			{#each tools as t (t.slug)}<WikiLink label={t.name} mcp={t.mcp} onclick={onTool ? () => onTool?.(t.slug) : undefined} />{/each}
		</div>
	</div>
</div>
