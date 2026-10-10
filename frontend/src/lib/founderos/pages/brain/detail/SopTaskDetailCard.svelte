<!-- SOP detail (FounderOS v1 KnowledgeDetail SopTaskDetailCard): the full
     playbook behind one SOP: breakdown, dependencies, what it replaces, the
     autonomy ladder, the human's role, the written steps, its tools, the
     runnable skill file and its build status. -->
<script lang="ts">
	import { ChevronDown, CircleDot, DollarSign, Download, GitBranch, HelpCircle, Layers, ListChecks, Puzzle, Sparkles, StickyNote, UserCheck, UserCog, Wrench } from '$lib/founderos/icons';
	import DetailRow from './DetailRow.svelte';
	import PanelHeader from './PanelHeader.svelte';
	import SectionLabel from './SectionLabel.svelte';
	import WikiLink from './WikiLink.svelte';
	import { AUTONOMY_LEVELS, autonomyLabel, playbookFor, statusLabel, type AutonomyLevel, type PlaybookStatus } from '../sop-playbooks';
	import type { SopTask } from '../types';

	let {
		task,
		assigneeName,
		assigneeKindLabel,
		assigneeColor,
		runtime = null,
		tools,
		onBack,
		onClose,
		onAssignee,
		onTool
	}: {
		task: SopTask;
		assigneeName: string;
		assigneeKindLabel: string;
		assigneeColor: string;
		runtime?: string | null;
		tools: { slug: string; name: string; mcp: boolean }[];
		onBack?: () => void;
		onClose?: () => void;
		onAssignee?: () => void;
		onTool?: (slug: string) => void;
	} = $props();

	const pb = $derived(playbookFor(task));
	let skillOpen = $state(false);

	const AUTONOMY_TONE: Record<AutonomyLevel, string> = {
		'human-led': 'var(--bn-warn)',
		'human-assisted': 'var(--bn-text)',
		'fully-autonomous': 'var(--bn-accent)'
	};
	const STATUS_TONE: Record<PlaybookStatus, string> = {
		'ready-to-run': 'var(--bn-ok)',
		'in-development': 'var(--bn-warn)',
		'not-started': 'var(--bn-text-3)'
	};
	const rung = (lvl: AutonomyLevel) => (lvl === 'human-led' ? pb.ladder.humanLed : lvl === 'human-assisted' ? pb.ladder.humanAssisted : pb.ladder.fullyAutonomous);

	function downloadSkill() {
		const blob = new Blob([pb.skillMarkdown], { type: 'text/markdown' });
		const url = URL.createObjectURL(blob);
		const a = document.createElement('a');
		a.href = url;
		a.download = `${pb.skill.slug}.md`;
		a.click();
		URL.revokeObjectURL(url);
	}
</script>

{#snippet chip(label: string, dashed: boolean)}
	<span class="inline-flex items-center rounded-[5px] border px-2 py-0.5 font-mono text-[10px] {dashed ? 'kg-b bn-muted border-dashed' : 'kg-bs bn-text'}">{label}</span>
{/snippet}

<div class="flex h-full flex-col" data-card="sop">
	<PanelHeader title={task.title} sub={pb.categoryPath} {onBack} {onClose} />
	<div class="min-h-0 flex-1 overflow-y-auto overscroll-contain px-3 py-2.5">
		<div class="mb-2 flex items-center gap-2">
			<span class="font-mono text-[9px] font-bold uppercase tracking-[0.18em]" style="color: {AUTONOMY_TONE[pb.autonomy]}">{autonomyLabel(pb.autonomy)}</span>
			<span class="ml-auto inline-flex items-center gap-1 font-mono text-[9px] uppercase tracking-[0.12em]" style="color: {STATUS_TONE[pb.status]}">
				<CircleDot size={10} />
				{statusLabel(pb.status)}
			</span>
		</div>

		<p class="bn-muted mb-3 text-[11px] leading-relaxed">{pb.description}</p>

		<button type="button" onclick={downloadSkill} class="bn-pressable kg-b kg-surf kg-h-bs mb-4 flex w-full items-center gap-2 rounded-[8px] border px-2.5 py-2 text-left">
			<span class="kg-accent shrink-0"><Download size={14} /></span>
			<span class="min-w-0 flex-1">
				<span class="bn-text block text-[11px] font-semibold">1 runnable skill file</span>
				<span class="bn-dim block font-mono text-[9px]">yours to download</span>
			</span>
		</button>

		<SectionLabel icon={Puzzle}>breaks into</SectionLabel>
		<div class="mb-3.5 flex flex-wrap gap-1.5">{#each pb.breaksInto as s (s)}{@render chip(s, false)}{/each}</div>

		<SectionLabel icon={Layers}>builds on</SectionLabel>
		<div class="mb-3.5 flex flex-wrap gap-1.5">
			{#each pb.buildsOn as s (s)}{@render chip(s, true)}{:else}<span class="bn-dim font-mono text-[10px]">nothing upstream</span>{/each}
		</div>

		<SectionLabel icon={DollarSign}>what it replaces</SectionLabel>
		<p class="kg-b kg-surf bn-muted mb-4 rounded-[8px] border px-2.5 py-2 text-[11px] leading-relaxed">{pb.replaces}</p>

		<SectionLabel icon={GitBranch}>the ladder</SectionLabel>
		<div class="mb-4 flex flex-col gap-1">
			{#each AUTONOMY_LEVELS as lvl (lvl)}
				{@const active = lvl === pb.autonomy}
				<div class="flex gap-2.5 border-l-2 py-1 pl-2.5" style="border-color: {active ? 'var(--bn-accent)' : 'var(--bn-hairline)'}">
					<span class="w-[104px] shrink-0 font-mono text-[9px] uppercase leading-relaxed tracking-[0.12em] {active ? 'kg-accent' : 'bn-dim'}">{autonomyLabel(lvl)}</span>
					<span class="flex-1 text-[10.5px] leading-relaxed {active ? 'bn-text' : 'bn-dim'}">{rung(lvl)}</span>
				</div>
			{/each}
		</div>

		<SectionLabel icon={UserCheck}>the human</SectionLabel>
		<p class="bn-muted mb-4 text-[11px] leading-relaxed">{pb.theHuman}</p>

		<SectionLabel icon={UserCog}>done by</SectionLabel>
		<div class="mb-4">
			<DetailRow color={assigneeColor} title={assigneeName} sub={assigneeKindLabel} badge="1:1" onclick={onAssignee} />
			{#if runtime}<p class="bn-dim mt-1 font-mono text-[9.5px]">runs on <span class="bn-muted">{runtime}</span></p>{/if}
		</div>

		<SectionLabel icon={ListChecks}>the SOP, written out</SectionLabel>
		<ol class="mb-4 flex flex-col gap-1.5">
			{#each task.steps as s, i (i)}
				<li class="bn-muted flex gap-2 text-[11px] leading-relaxed"><span class="bn-dim shrink-0 font-mono text-[10px]">{String(i + 1).padStart(2, '0')}</span>{s}</li>
			{/each}
		</ol>

		<SectionLabel icon={Wrench}>tools at the end of the chain ({tools.length})</SectionLabel>
		<div class="mb-4 flex flex-wrap gap-x-3 gap-y-1.5">
			{#each tools as t (t.slug)}<WikiLink label={t.name} mcp={t.mcp} onclick={onTool ? () => onTool?.(t.slug) : undefined} />{/each}
			{#if tools.length === 0}<span class="bn-dim font-mono text-[10.5px]">no tools wired</span>{/if}
		</div>

		<SectionLabel icon={StickyNote}>build notes</SectionLabel>
		<p class="bn-muted mb-4 text-[11px] leading-relaxed">{pb.buildNotes}</p>

		<SectionLabel icon={Sparkles}>take the skill</SectionLabel>
		<div class="kg-b kg-surf mb-2 rounded-[8px] border">
			<button type="button" onclick={() => (skillOpen = !skillOpen)} class="bn-pressable flex w-full items-center gap-2 px-2.5 py-2 text-left">
				<span class="kg-accent shrink-0"><Sparkles size={14} /></span>
				<span class="min-w-0 flex-1">
					<span class="bn-text block truncate text-[11px] font-semibold">{pb.skill.name}</span>
					<span class="bn-dim block truncate font-mono text-[9px]">{pb.skill.slug}.md</span>
				</span>
				<span class="bn-dim shrink-0 transition-transform" class:rotate-180={skillOpen}><ChevronDown size={14} /></span>
			</button>
			<p class="kg-b bn-muted border-t px-2.5 py-1.5 text-[10.5px] leading-relaxed">{pb.skill.blurb}</p>
			{#if skillOpen}
				<div class="kg-b border-t">
					<pre class="bn-muted max-h-64 overflow-auto px-2.5 py-2 font-mono text-[9.5px] leading-relaxed">{pb.skillMarkdown}</pre>
					<button type="button" onclick={downloadSkill} class="bn-pressable kg-b bn-dim kg-h-accent flex w-full items-center justify-center gap-1.5 border-t px-2.5 py-1.5 font-mono text-[9.5px] uppercase tracking-[0.14em]">
						<Download size={12} /> download {pb.skill.slug}.md
					</button>
				</div>
			{/if}
		</div>

		<button type="button" class="bn-pressable kg-b bn-dim kg-h-bs kg-h-text mt-1 flex w-full items-center justify-center gap-1.5 rounded-[8px] border border-dashed px-2.5 py-2 font-mono text-[10px] uppercase tracking-[0.12em]">
			<HelpCircle size={14} /> need help building this?
		</button>
	</div>
</div>
