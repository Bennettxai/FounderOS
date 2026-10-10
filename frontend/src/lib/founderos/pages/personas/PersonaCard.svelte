<!-- One persona as a card (FounderOS v1 PersonasViewer PersonaCard, split-view
     form: the knowledge graph beside it is the visual, so no cover chart). -->
<script lang="ts">
	import type { Component } from 'svelte';
	import { Boxes, Brain, ChartColumn, Plug, Star, Zap } from '$lib/founderos/icons';
	import { pad2 } from './graph';
	import type { Persona } from './types';

	let { p }: { p: Persona } = $props();

	// lucide-svelte ships legacy class components; the snippet takes them as plain components.
	type Icon = Component<{ class?: string }>;
	const icon = (c: unknown) => c as Icon;
</script>

{#snippet section(Icon: Icon, label: string)}
	<div class="bn-dim mb-2 flex items-center gap-1.5 font-mono text-[9.5px] uppercase tracking-[0.16em]">
		<Icon class="h-3 w-3" />
		{label}
	</div>
{/snippet}

<article class="bn-card flex flex-col overflow-hidden rounded-[var(--bn-r-panel)]" style={`border-top: 2px solid ${p.accent}`}>
	<div class="border-b px-5 py-4 bn-border">
		<div class="flex items-start justify-between gap-3">
			<div class="min-w-0">
				<div class="flex items-center gap-2">
					<span class="font-mono text-[11px] font-bold" style={`color: ${p.accent}`}>{pad2(p.order)}</span>
					<h2 class="bn-text text-[15px] font-bold">{p.name}</h2>
				</div>
				<div class="bn-muted mt-0.5 text-[12px] [text-wrap:pretty]">{p.tagline}</div>
			</div>
			<span class="shrink-0 rounded-[var(--bn-r-chip)] border px-2 py-0.5 font-mono text-[9.5px] uppercase tracking-wide" style={`border-color: ${p.accent}; color: ${p.accent}`}>{p.archetype}</span>
		</div>
		<p class="bn-dim mt-3 text-[12px] leading-relaxed [text-wrap:pretty]">{p.summary}</p>
		<div class="mt-3 flex items-start gap-2 rounded-[var(--bn-r-card)] border px-3 py-2 bn-border" style="background: var(--bn-surface-2)">
			<span class="mt-0.5 shrink-0" style={`color: ${p.accent}`}><Star class="h-3.5 w-3.5" /></span>
			<div>
				<span class="bn-dim font-mono text-[9px] uppercase tracking-[0.16em]">North star</span>
				<div class="bn-text text-[12.5px] font-semibold">{p.northStar}</div>
			</div>
		</div>
	</div>

	<div class="px-5 py-4">
		{@render section(icon(Boxes), `Pillars · ${p.pillars.length}`)}
		<div class="grid gap-2.5 lg:grid-cols-2">
			{#each p.pillars as pillar (pillar.name)}
				<div class="rounded-[var(--bn-r-card)] border px-3 py-2.5 bn-border" style="background: var(--bn-surface-2)">
					<div class="flex items-baseline justify-between gap-2">
						<span class="bn-text text-[12.5px] font-semibold">{pillar.name}</span>
						<span class="bn-dim shrink-0 truncate font-mono text-[10px]">{pillar.focus}</span>
					</div>
					<div class="mt-1.5 flex flex-wrap gap-1.5">
						{#each pillar.agents as a (a)}
							<span class="bn-accent rounded-[var(--bn-r-chip)] border px-1.5 py-0.5 font-mono text-[9.5px] bn-border bn-surface">{a}</span>
						{/each}
					</div>
				</div>
			{/each}
		</div>
	</div>

	<div class="grid gap-4 border-t px-5 py-4 bn-border sm:grid-cols-2">
		<div>
			{@render section(icon(Plug), 'Connectors')}
			<div class="flex flex-wrap gap-1.5">
				{#each p.connectors as c (c)}
					<span class="bn-muted rounded-[var(--bn-r-chip)] border px-2 py-0.5 font-mono text-[10px] bn-border" style="background: var(--bn-surface-2)">{c}</span>
				{/each}
			</div>
		</div>
		<div>
			{@render section(icon(ChartColumn), 'Tracks')}
			<ul class="flex flex-col gap-1">
				{#each p.metrics as m (m)}
					<li class="bn-muted flex items-baseline gap-1.5 text-[11.5px]">
						<span class="font-mono" style={`color: ${p.accent}`}>›</span>{m}
					</li>
				{/each}
			</ul>
		</div>
	</div>

	<div class="flex flex-col gap-2.5 border-t px-5 py-4 bn-border">
		<div class="flex items-start gap-2">
			<span class="bn-dim mt-0.5 shrink-0"><Brain class="h-3.5 w-3.5" /></span>
			<p class="bn-dim text-[11.5px] leading-relaxed [text-wrap:pretty]">{p.brainUse}</p>
		</div>
		<div class="flex items-start gap-2 rounded-[var(--bn-r-card)] border px-3 py-2.5" style={`border-color: ${p.accent}66; background: color-mix(in srgb, ${p.accent} 9%, var(--bn-surface))`}>
			<span class="mt-0.5 shrink-0" style={`color: ${p.accent}`}><Zap class="h-3.5 w-3.5" /></span>
			<div>
				<span class="font-mono text-[9px] uppercase tracking-[0.16em]" style={`color: ${p.accent}`}>Signature play</span>
				<p class="bn-text mt-0.5 text-[12px] leading-relaxed [text-wrap:pretty]">{p.signaturePlay}</p>
			</div>
		</div>
	</div>
</article>
