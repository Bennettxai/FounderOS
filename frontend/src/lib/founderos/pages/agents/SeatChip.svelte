<!-- One board seat: glyph + name + model line, and a real Run button that
     fires a board heartbeat (FounderOS v1 components/BoardLive.tsx SeatChip).
     A click owns the control until its result has been seen, so the 4s poll
     cannot swap the button out mid-flight. -->
<script lang="ts">
	import { onDestroy } from 'svelte';
	import { Bot, Cpu, Crown, FileText, Hammer, Landmark, LoaderCircle, Megaphone, MessageSquare, Sparkles, TrendingUp, Zap } from '$lib/founderos/icons';
	import { founderosFetch } from '$lib/founderos/api';
	import { seatTone } from './board';
	import { writeFailure } from './runerror';
	import type { PaperclipAgent } from './types';

	let { agent, since = null }: { agent: PaperclipAgent; since?: string | null } = $props();

	// One glyph per seat, colour-coded with the department heads (prod
	// SEAT_GLYPHS); Bot in the muted tone is the honest fallback.
	const GLYPHS: [RegExp, typeof Bot, string][] = [
		[/conductor/i, Crown, '#e4efe6'],
		[/tech/i, Cpu, '#8b7cf6'],
		[/marketing|growth/i, Megaphone, '#ff9f43'],
		[/finance/i, Landmark, '#38bdf8'],
		[/sales/i, TrendingUp, '#ffd166'],
		[/comm/i, MessageSquare, '#4cc9f0'],
		[/forge/i, Hammer, '#f472b6'],
		[/hermes|worker/i, Zap, '#a78bfa'],
		[/summar/i, FileText, '#94a3b8'],
		[/reflect/i, Sparkles, '#facc15']
	];
	const glyph = $derived(GLYPHS.find(([re]) => re.test(agent.name)));
	const Icon = $derived(glyph?.[1] ?? Bot);
	const idleColor = $derived(glyph?.[2] ?? 'var(--bn-text-2)');

	let owned = $state(false);
	let busy = $state(false);
	let done = $state(false);
	let failure = $state<string | null>(null);
	let now = $state(Date.now());
	let release: ReturnType<typeof setTimeout> | null = null;
	const tick = setInterval(() => (now = Date.now()), 1000);
	onDestroy(() => {
		clearInterval(tick);
		if (release) clearTimeout(release);
	});

	async function run() {
		owned = true;
		busy = true;
		done = false;
		failure = null;
		if (release) clearTimeout(release);
		try {
			await founderosFetch(`/pages/board/agents/${agent.id}/run`, { method: 'POST' });
			done = true;
		} catch (err) {
			failure = writeFailure(err);
		} finally {
			busy = false;
			release = setTimeout(() => {
				owned = false;
				done = false;
			}, 1600);
		}
	}

	const running = $derived(agent.status === 'running');
	const live = $derived(running || owned);
	const tone = $derived(seatTone(agent, live));
	// status overrides the department colour: green while on, red when broken
	const toneVar = $derived(tone === 'idle' ? idleColor : `var(--bn-${tone})`);
	const modelLine = $derived(`${agent.model ?? agent.adapterType ?? 'unconfigured'}${agent.model && agent.adapterType ? ` · ${agent.adapterType}` : ''}`);
	const elapsed = $derived.by(() => {
		if (!since) return null;
		const s = Math.max(0, Math.floor((now - new Date(since).getTime()) / 1000));
		return s < 60 ? `${s}s` : `${Math.floor(s / 60)}m${String(s % 60).padStart(2, '0')}`;
	});
</script>

<div
	data-part="seat"
	data-lens="r"
	class="bn-pressable is-row bn-seat flex items-start gap-2 border px-2.5 py-2"
	class:is-live={live}
	class:bn-agent-live={live}
	style:border-color={live ? undefined : agent.status === 'error' ? 'color-mix(in oklab, var(--bn-err) 45%, var(--bn-border))' : 'var(--bn-border)'}
>
	<span class="mt-0.5 shrink-0" style:color={toneVar} class:animate-pulse={running}><Icon size={14} strokeWidth={1.8} /></span>
	<div class="min-w-0 flex-1">
		<div class="bn-text truncate text-[11.5px] font-semibold leading-tight" title={agent.name}>{agent.name}</div>
		<div class="bn-dim truncate font-mono text-[9px]" title={modelLine}>
			{modelLine}{#if failure}<span style:color="var(--bn-err)"> · {failure}</span>{/if}
		</div>
	</div>
	<div class="flex w-7 shrink-0 flex-col items-center gap-1 self-center">
		{#if running && !owned}
			<span class="animate-spin" style:color="var(--bn-ok)"><LoaderCircle size={14} /></span>
			{#if elapsed}<span class="font-mono text-[8.5px] tabular-nums leading-none" style:color="var(--bn-ok)">{elapsed}</span>{/if}
		{:else}
			<button
				type="button"
				title="Run {agent.name} heartbeat on the board"
				aria-label="Run {agent.name} heartbeat on the board"
				disabled={busy}
				onclick={run}
				data-lens="c"
				class="bn-pressable is-dark bn-run-btn inline-grid h-[26px] w-[26px] place-items-center border font-mono text-[10.5px] font-semibold"
			>
				{#if busy}<span class="bn-run-spin inline-block h-[10px] w-[10px] rounded-full border-[1.5px]"></span>{:else if done}<span class="bn-pop" style:color="var(--bn-ok)">✓</span>{:else if failure}<span class="bn-pop" style:color="var(--bn-err)">✗</span>{:else}▸{/if}
			</button>
		{/if}
	</div>
</div>

<style>
	.bn-seat {
		border-radius: var(--bn-r-ctl, 6px);
		background: var(--bn-bg);
	}
	/* prod AsyncButton tone="ghost": no border, dim, the dark lens on hover */
	.bn-run-btn {
		border-color: transparent;
		border-radius: 6px;
		background: transparent;
		color: var(--bn-text-3);
	}
	.bn-run-spin {
		border-color: var(--bn-ok);
		border-right-color: transparent;
		animation: bn-om-spin 0.8s linear infinite;
	}
</style>
