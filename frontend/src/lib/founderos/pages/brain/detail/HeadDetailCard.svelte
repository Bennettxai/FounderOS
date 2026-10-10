<!-- Department-head card (FounderOS v1 KnowledgeDetail HeadDetailCard): the
     pillar's board seat, a Run heartbeat button and the SOP skills it presides
     over. Board seats reuse it without the SOP block. The Run goes through the
     bridge's guarded board route: with writes off it is held, and says so. -->
<script lang="ts">
	import { ClipboardList, Crown, Play, Server } from '$lib/founderos/icons';
	import { FounderosApiError, founderosFetch, isGuardRefusal } from '$lib/founderos/api';
	import DetailRow from './DetailRow.svelte';
	import PanelHeader from './PanelHeader.svelte';
	import SectionLabel from './SectionLabel.svelte';
	import type { Snippet } from 'svelte';

	let {
		title,
		deptName,
		color,
		boardLead,
		sops,
		roleLabel = 'department head',
		blurb,
		showSops = true,
		embed = null,
		onBack,
		onClose,
		onTask
	}: {
		title: string;
		deptName: string;
		color: string;
		boardLead: { id: string; name: string; status: string; model: string | null } | null;
		sops: { id: string; title: string; skillName: string }[];
		roleLabel?: string;
		blurb?: Snippet;
		showSops?: boolean;
		embed?: { url: string; title: string } | null;
		onBack?: () => void;
		onClose?: () => void;
		onTask?: (taskId: string) => void;
	} = $props();

	let running = $state(false);
	let runMsg = $state<string | null>(null);

	async function run() {
		if (!boardLead || running) return;
		running = true;
		runMsg = null;
		try {
			await founderosFetch(`/pages/board/agents/${boardLead.id}/run`, { method: 'POST' });
			runMsg = 'Run started on the board — watch the chips go green.';
		} catch (err) {
			if (isGuardRefusal(err)) runMsg = 'Run held: writes are off on this bridge (FOUNDEROS_WRITES=0).';
			else runMsg = `Run failed: ${err instanceof FounderosApiError ? err.message : err instanceof Error ? err.message : String(err)}`;
		} finally {
			running = false;
		}
	}
	const dot = $derived(
		boardLead?.status === 'running' ? 'animate-pulse bg-[var(--bn-ok)]' : boardLead?.status === 'error' ? 'bg-[var(--bn-err)]' : boardLead?.status === 'paused' ? 'bg-[var(--bn-warn)]' : 'bg-[var(--bn-text-2)]'
	);
</script>

<div class="flex h-full flex-col" data-card="head">
	<PanelHeader {title} sub={`${deptName} · ${roleLabel}`} {color} {onBack} {onClose} />
	<div class="min-h-0 flex-1 overflow-y-auto overscroll-contain px-3 py-2.5">
		<p class="bn-muted mb-3 text-[11px] leading-relaxed">
			{#if blurb}{@render blurb()}{:else}Executive seat of <span class="font-semibold">{deptName}</span> — every SOP below reports through this desk.{/if}
		</p>

		<SectionLabel icon={Crown}>board seat</SectionLabel>
		<div class="mb-3">
			{#if boardLead}
				<div class="kg-b kg-bg flex items-center gap-2 rounded border px-2.5 py-1.5" data-part="board-seat">
					<span class="h-1.5 w-1.5 shrink-0 rounded-full {dot}"></span>
					<span class="bn-text min-w-0 flex-1 truncate text-[11.5px] font-semibold">{boardLead.name}</span>
					<span class="bn-dim font-mono text-[9.5px] uppercase tracking-wider">{boardLead.status}</span>
					{#if boardLead.model}<span class="bn-dim font-mono text-[9px]">{boardLead.model}</span>{/if}
				</div>
			{:else}
				<p class="bn-dim font-mono text-[10.5px]" data-part="board-seat">no board seat yet — hire this head on the Paperclip board</p>
			{/if}
		</div>

		<button
			type="button"
			onclick={run}
			disabled={!boardLead || running}
			class="bn-pressable kg-bs kg-surf2 bn-text kg-h-dimb mb-1.5 flex w-full items-center justify-center gap-1.5 rounded border px-3 py-1.5 text-[11.5px] font-semibold disabled:opacity-40"
		>
			<Play size={12} />
			{running ? 'starting run…' : 'Run heartbeat on the board'}
		</button>
		{#if runMsg}<p class="bn-dim mb-3 font-mono text-[9.5px] leading-snug" data-part="run-msg">{runMsg}</p>{/if}

		{#if embed}
			<SectionLabel icon={Server}>{embed.title} — live</SectionLabel>
			<div class="kg-b kg-bg mb-1.5 overflow-hidden rounded border">
				<iframe src={embed.url} title={embed.title} class="h-80 w-full" style="border: 0"></iframe>
			</div>
			<a href={embed.url} target="_blank" rel="noreferrer" class="bn-dim mb-3 block font-mono text-[9.5px] underline-offset-2 hover:underline">open the full dashboard ↗</a>
		{/if}

		{#if showSops}
			<SectionLabel icon={ClipboardList}>presides over ({sops.length} skills)</SectionLabel>
			<div class="space-y-1">
				{#each sops as s (s.id)}
					<DetailRow {color} title={s.title} sub={s.skillName} badge="skill" onclick={onTask ? () => onTask?.(s.id) : undefined} />
				{/each}
				{#if sops.length === 0}<p class="bn-dim font-mono text-[10.5px]">no SOPs in this department yet</p>{/if}
			</div>
		{/if}
	</div>
</div>
