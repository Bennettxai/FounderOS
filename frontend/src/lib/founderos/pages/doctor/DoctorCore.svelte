<!-- The Doctor Core (FounderOS v1 components/BrainCore.tsx + BrainViz.tsx) on
     Optimal Engine: the radar sweep, the outer topology ring (dashed, sampled
     dots), the middle vector ring (eight diamonds), the inner store ring (one
     node per knowledge row, capped) with its cluster labels, and the pulsing
     health gauge. The gauge is a button: it opens the Doctor · Search pop-out
     with the flagged checks and a read-only brain query (GET /pages/brain/query). -->
<script lang="ts">
	import { founderosFetch } from '$lib/founderos/api';
	import type { BrainHit, BrainQueryBody } from '$lib/founderos/pages/brain/types';
	import { CX, CY, coreCallouts, groupChecks, knowledgeClusters, layoutCoreNodes, outerDots, polar } from './core';
	import type { DoctorCheck, EngineReading } from './types';

	let {
		engines,
		health,
		checks,
		connected,
		detail
	}: { engines: EngineReading[]; health: number | null; checks: DoctorCheck[]; connected: boolean; detail: string } = $props();

	const clusters = $derived(knowledgeClusters(engines));
	const layout = $derived(layoutCoreNodes(clusters));
	const callouts = $derived(coreCallouts(engines));
	const outer = outerDots();
	const diamonds = Array.from({ length: 8 }, (_, i) => polar(CX, CY, 158, i * 45));
	const arcR = 76;
	const C = 2 * Math.PI * arcR;
	const healthArc = $derived(((health ?? 0) / 100) * C);

	function anchor(angle: number): 'start' | 'end' | 'middle' {
		const norm = ((angle % 360) + 360) % 360;
		return norm < 88 || norm > 272 ? 'start' : Math.abs(norm - 180) < 88 ? 'end' : 'middle';
	}

	// ── the pop-out ────────────────────────────────────────────────────────
	let open = $state(false);
	const lines = $derived(groupChecks(checks));
	const flagged = $derived(lines.filter((l) => l.status !== 'ok'));
	const dot = (s: string) => (s === 'ok' ? 'var(--bn-ok)' : s === 'warn' ? 'var(--bn-warn)' : 'var(--bn-err)');

	$effect(() => {
		if (!open) return;
		const onKey = (e: KeyboardEvent) => e.key === 'Escape' && (open = false);
		window.addEventListener('keydown', onKey);
		const prev = document.body.style.overflow;
		document.body.style.overflow = 'hidden';
		return () => {
			window.removeEventListener('keydown', onKey);
			document.body.style.overflow = prev;
		};
	});

	type QueryState =
		| { phase: 'idle' }
		| { phase: 'busy'; query: string }
		| { phase: 'done'; query: string; hits: BrainHit[]; partial: string[] }
		| { phase: 'error'; query: string; message: string };
	let q = $state('');
	let qs = $state<QueryState>({ phase: 'idle' });
	let input = $state<HTMLInputElement | null>(null);
	$effect(() => {
		if (open && input) input.focus();
	});

	async function run() {
		const query = q.trim();
		if (!query || qs.phase === 'busy') return;
		qs = { phase: 'busy', query };
		try {
			const body = await founderosFetch<BrainQueryBody>(`/pages/brain/query?q=${encodeURIComponent(query)}`);
			qs = { phase: 'done', query, hits: body.results ?? [], partial: body.failedWorkspaces ?? [] };
			q = '';
		} catch (err) {
			qs = { phase: 'error', query, message: err instanceof Error ? err.message : String(err) };
		}
	}
	const statusLine = $derived(
		qs.phase === 'busy'
			? 'querying every workspace’s engine…'
			: qs.phase === 'done'
				? `${qs.hits.length} hits · "${qs.query}"${qs.partial.length ? ` · ${qs.partial.join(', ')} unreachable` : ''}`
				: qs.phase === 'error'
					? `query failed · ${qs.message}`
					: 'press ↵ to query the engines'
	);
</script>

<div class="relative">
	<svg viewBox="0 0 520 520" class="block w-full" role="img" aria-label="Brain doctor core">
		<defs>
			<radialGradient id="dc-core">
				<stop offset="0%" stop-color="var(--bn-brain-1)" stop-opacity="0.22" />
				<stop offset="70%" stop-color="var(--bn-brain-1)" stop-opacity="0.07" />
				<stop offset="100%" stop-color="var(--bn-brain-1)" stop-opacity="0" />
			</radialGradient>
			<linearGradient id="dc-sweep" x1="0" y1="0" x2="1" y2="0">
				<stop offset="0%" stop-color="var(--bn-brain-2)" stop-opacity="0" />
				<stop offset="100%" stop-color="var(--bn-brain-2)" stop-opacity="0.13" />
			</linearGradient>
			<linearGradient id="dc-arc" x1="0" y1="0" x2="1" y2="1">
				<stop offset="0%" stop-color="var(--bn-brain-1)" />
				<stop offset="55%" stop-color="var(--bn-brain-2)" />
				<stop offset="100%" stop-color="var(--bn-brain-3)" />
			</linearGradient>
		</defs>

		<!-- radar sweep -->
		<g class="dc-spin dc-sweep">
			<path d="M260 260 L260 40 A220 220 0 0 1 369 69 Z" fill="url(#dc-sweep)" />
		</g>

		<!-- outer ring: the engine topology -->
		<g class="dc-spin dc-r3">
			<circle cx="260" cy="260" r="210" fill="none" stroke="var(--bn-border-strong)" stroke-dasharray="2 7" stroke-width="1" />
			{#each outer as p, i (i)}
				<circle cx={p.x} cy={p.y} r="1.6" fill="var(--bn-text-3)" opacity="0.55" />
			{/each}
		</g>

		<!-- middle ring: vector embeddings -->
		<g class="dc-spin dc-r2">
			<circle cx="260" cy="260" r="158" fill="none" stroke="var(--bn-border-strong)" stroke-width="1" opacity="0.8" />
			{#each diamonds as [x, y], i (i)}
				<rect x={x - 2.6} y={y - 2.6} width="5.2" height="5.2" transform="rotate(45 {x} {y})" fill="var(--bn-brain-2)" opacity="0.85" />
			{/each}
		</g>

		<!-- inner ring: the engine store -->
		<g class="dc-spin dc-r1">
			<circle cx="260" cy="260" r="108" fill="none" stroke="var(--bn-brain-1)" stroke-opacity="0.3" stroke-width="1" />
			{#each layout.nodes as n, i (i)}
				<line x1="260" y1="260" x2={n.x} y2={n.y} stroke="var(--bn-brain-1)" stroke-width="0.5" opacity="0.18" />
				<circle cx={n.x} cy={n.y} r="3.4" fill="var(--bn-brain-1)" />
				<circle cx={n.x} cy={n.y} r="6.5" fill="none" stroke="var(--bn-brain-1)" stroke-width="0.6" opacity="0.3" />
			{/each}
		</g>

		<!-- static cluster labels -->
		{#each layout.labels as l (l.label)}
			{@const p = polar(CX, CY, 134, l.angle)}
			<text x={p[0]} y={p[1]} text-anchor={anchor(l.angle)} dominant-baseline="middle" font-family="var(--bn-font)" font-size="8.5" letter-spacing="1.5" fill="var(--bn-text-3)"
				>{l.label.toUpperCase()} · {l.pages}</text
			>
		{/each}

		<!-- core glow + health gauge -->
		<circle cx="260" cy="260" r="120" fill="url(#dc-core)" />
		<g class="dc-pulse">
			<circle cx="260" cy="260" r={arcR} fill="none" stroke="var(--bn-border)" stroke-width="5" />
			{#if health != null}
				<circle
					cx="260"
					cy="260"
					r={arcR}
					fill="none"
					stroke="url(#dc-arc)"
					stroke-width="5"
					stroke-linecap="round"
					stroke-dasharray="{healthArc} {C}"
					transform="rotate(-90 260 260)"
				/>
			{/if}
			<circle cx="260" cy="260" r="58" fill="var(--bn-surface)" stroke="var(--bn-brain-1)" stroke-opacity="0.35" stroke-width="1" />
			<text x="260" y="252" text-anchor="middle" font-family="var(--bn-font)" font-size="30" font-weight="600" fill="var(--bn-text)">{health ?? '—'}</text>
			<text x="260" y="272" text-anchor="middle" font-family="var(--bn-font)" font-size="8" letter-spacing="2.5" fill="var(--bn-text-3)">HEALTH / 100</text>
			<text x="260" y="288" text-anchor="middle" font-family="var(--bn-font)" font-size="7.5" letter-spacing="1.5" fill="var(--bn-brain-2)">OPTIMAL ENGINE</text>
		</g>

		<!-- ring callouts -->
		<g font-family="var(--bn-font)" font-size="8.5" letter-spacing="1.2">
			<text x="260" y="146" text-anchor="middle" fill="var(--bn-brain-1)">{callouts.inner}</text>
			<text x="260" y="96" text-anchor="middle" fill="var(--bn-brain-2)">{callouts.middle}</text>
			<text x="260" y="44" text-anchor="middle" fill="var(--bn-text-3)">{callouts.outer}</text>
		</g>
	</svg>

	<!-- hotspot over the central gauge -->
	<button
		type="button"
		onclick={() => (open = true)}
		aria-label="Open Brain doctor and search"
		class="dc-hot group absolute left-1/2 top-1/2 grid h-[27%] w-[27%] -translate-x-1/2 -translate-y-1/2 place-items-end justify-center rounded-full pb-[6%] outline-none"
	>
		<span
			class="pointer-events-none flex translate-y-3 items-center gap-1 rounded-full border px-2 py-0.5 font-mono text-[8.5px] uppercase tracking-[0.12em] opacity-0 backdrop-blur transition-opacity duration-150 group-hover:opacity-100 group-focus-visible:opacity-100"
			style="border-color: color-mix(in oklab, var(--bn-accent) 40%, transparent); background: color-mix(in oklab, var(--bn-bg) 80%, transparent); color: var(--bn-accent)"
		>
			<svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true"><circle cx="11" cy="11" r="7" /><path d="m20 20-3.5-3.5" /></svg>
			doctor · search
		</span>
	</button>
</div>

{#if open}
	<!-- svelte-ignore a11y_click_events_have_key_events, a11y_no_static_element_interactions -->
	<div class="bn-overlay-in fixed inset-0 z-50 flex items-start justify-center overflow-y-auto p-4 backdrop-blur-sm sm:p-8" style="background: rgb(0 0 0 / 0.7)" onclick={() => (open = false)}>
		<div
			role="dialog"
			aria-modal="true"
			aria-label="Brain doctor and search"
			tabindex="-1"
			class="bn-panel-in mb-[6vh] mt-[5vh] w-full max-w-2xl rounded-[var(--bn-r-panel)] border"
			style="border-color: var(--bn-border); background: var(--bn-bg-2)"
			onclick={(e) => e.stopPropagation()}
		>
			<div class="sticky top-0 flex items-center justify-between border-b px-5 py-3.5" style="border-color: var(--bn-border); background: var(--bn-bg-2)">
				<div class="flex items-center gap-2">
					<span class="bn-dim font-mono text-[11px] uppercase tracking-[0.16em]">Brain</span>
					<span class="bn-text text-sm font-semibold">Doctor · Search</span>
				</div>
				<button
					type="button"
					onclick={() => (open = false)}
					aria-label="Close"
					data-lens="c"
					class="bn-pressable bn-muted grid h-7 w-7 place-items-center rounded-[var(--bn-r-ctl)] border"
					style="border-color: var(--bn-border)"
				>
					<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true"><path d="M18 6 6 18M6 6l12 12" /></svg>
				</button>
			</div>

			<div class="space-y-6 px-5 py-5">
				<div>
					<div class="mb-2 flex items-baseline justify-between">
						<span class="bn-dim font-mono text-[10px] uppercase tracking-[0.16em]">Doctor — warnings</span>
						<span class="bn-dim font-mono text-[10px]">{connected ? `${flagged.length} of ${lines.length} checks` : 'offline'}</span>
					</div>
					<div class="rounded-[var(--bn-r-card)] border px-3.5 py-3" style="border-color: var(--bn-border); background: var(--bn-surface)">
						{#if connected && flagged.length === 0 && lines.length > 0}
							<div class="bn-muted font-mono text-[11px]">all {lines.length} checks green · health {health ?? '—'}/100</div>
						{/if}
						{#if !connected}
							<div class="font-mono text-[11px]" style="color: var(--bn-err)">doctor offline — {detail}</div>
						{/if}
						{#if flagged.length > 0}
							<ul class="space-y-2">
								{#each flagged as check (check.name)}
									<li class="flex items-start gap-2.5 text-[11px]">
										<span class="mt-1 inline-block h-1.5 w-1.5 shrink-0 rounded-full" style="background: {dot(check.status)}"></span>
										<span class="bn-muted">
											<span class="bn-text font-mono font-semibold">{check.name}</span> — {check.message} <span class="bn-dim">({check.engines.join(', ')})</span>
										</span>
									</li>
								{/each}
							</ul>
						{/if}
					</div>
				</div>

				<div class="border-t pt-5" style="border-color: var(--bn-border)">
					<div class="bn-dim mb-2 font-mono text-[10px] uppercase tracking-[0.16em]">Hybrid search</div>
					<div class="flex min-h-[220px] flex-1 flex-col rounded-[var(--bn-r-panel)] border p-1" style="border-color: var(--bn-border); background: var(--bn-surface)">
						<div class="flex items-center gap-[9px] border-b px-3.5 py-[11px] font-mono text-xs" style="border-color: var(--bn-border)">
							<span class="font-bold" style="color: var(--bn-accent)">brain ›</span>
							<input
								bind:this={input}
								bind:value={q}
								onkeydown={(e) => e.key === 'Enter' && run()}
								placeholder="query the engines…"
								class="bn-text flex-1 bg-transparent font-mono text-xs outline-none"
							/>
							<kbd class="bn-muted rounded-[4px] border border-b-2 px-1.5 py-0.5 font-mono text-[10px]" style="border-color: var(--bn-border-strong); background: var(--bn-surface)">↵</kbd>
						</div>
						<div class="flex flex-col gap-1 overflow-y-auto p-2">
							<div class="px-[11px] py-1.5 font-mono text-[10px] tracking-[0.08em]" style="color: {qs.phase === 'error' ? 'var(--bn-err)' : 'var(--bn-text-3)'}">{statusLine}</div>
							{#if qs.phase === 'done'}
								{#each qs.hits as hit, i (hit.uri + i)}
									<div data-lens="r" class="bn-pressable is-row rounded-[var(--bn-r-ctl)] border border-transparent px-[11px] py-[9px]">
										<div class="bn-text flex items-baseline gap-2 font-mono text-[11.5px]">
											<span style="color: var(--bn-accent)">▸</span>
											<span class="min-w-0 flex-1 truncate">{hit.title}</span>
											<span class="shrink-0 font-mono text-[10px]" style="color: var(--bn-accent)">{hit.workspace || hit.source}</span>
										</div>
										<div class="bn-dim mt-[3px] text-[11px] leading-relaxed">{hit.snippet}</div>
									</div>
								{/each}
								{#if qs.hits.length === 0}
									<div class="bn-dim px-[11px] py-2 text-[11px]">nothing matched — try a broader phrase, or rerun the engine doctor</div>
								{/if}
							{/if}
						</div>
					</div>
				</div>
			</div>
		</div>
	</div>
{/if}

<style>
	.dc-spin,
	.dc-pulse {
		transform-origin: 260px 260px;
	}
	.dc-r1 {
		animation: dc-spin 80s linear infinite;
	}
	.dc-r2 {
		animation: dc-spin 140s linear infinite reverse;
	}
	.dc-r3 {
		animation: dc-spin 220s linear infinite;
	}
	.dc-sweep {
		animation: dc-spin 9s linear infinite;
	}
	.dc-pulse {
		animation: dc-pulse 3.2s ease-in-out infinite;
	}
	.dc-hot:focus-visible {
		box-shadow: 0 0 0 2px color-mix(in oklab, var(--bn-accent) 50%, transparent);
	}
	@keyframes dc-spin {
		to {
			transform: rotate(360deg);
		}
	}
	@keyframes dc-pulse {
		0%,
		100% {
			transform: scale(1);
		}
		50% {
			transform: scale(1.035);
		}
	}
	@media (prefers-reduced-motion: reduce) {
		.dc-spin,
		.dc-pulse {
			animation: none;
		}
	}
</style>
