<!-- A persona's knowledge graph (FounderOS v1 PersonaBrainGraph): core →
     pillars → agents, pure deterministic SVG tinted with the persona's accent. -->
<script lang="ts">
	import { brainGraph, H, R_AGENT, R_PILLAR, trunc, W } from './graph';
	import type { Persona } from './types';

	let { persona }: { persona: Persona } = $props();

	const g = $derived(brainGraph(persona));
	const accent = $derived(persona.accent);
	const gid = $derived(`pbg-${persona.id}`);
</script>

<svg viewBox={`0 0 ${W} ${H}`} preserveAspectRatio="xMidYMid meet" class="block h-full w-full" style="height: 100%; max-width: none" role="img" aria-label={`${persona.name} knowledge graph`}>
	<defs>
		<radialGradient id={`${gid}-glow`} cx="50%" cy="50%" r="42%">
			<stop offset="0%" stop-color={accent} stop-opacity="0.18" />
			<stop offset="100%" stop-color={accent} stop-opacity="0" />
		</radialGradient>
		<pattern id={`${gid}-grid`} width="48" height="48" patternUnits="userSpaceOnUse">
			<path d="M48 0H0V48" fill="none" stroke="var(--bn-hairline)" stroke-width="1" />
		</pattern>
		<radialGradient id={`${gid}-core`} cx="50%" cy="50%" r="50%">
			<stop offset="0%" stop-color={accent} stop-opacity="0.9" />
			<stop offset="100%" stop-color={accent} stop-opacity="0.35" />
		</radialGradient>
	</defs>

	<rect x="0" y="0" width={W} height={H} fill={`url(#${gid}-grid)`} opacity="0.5" />
	<rect x="0" y="0" width={W} height={H} fill={`url(#${gid}-glow)`} />

	<circle cx={g.cx} cy={g.cy} r={R_PILLAR} fill="none" stroke="var(--bn-hairline)" stroke-width="1" opacity="0.7" />
	<circle cx={g.cx} cy={g.cy} r={R_AGENT} fill="none" stroke="var(--bn-hairline)" stroke-width="1" stroke-dasharray="2 6" opacity="0.6" />

	{#each g.pillars as pn, i (i)}
		{#each pn.agents as ag, j (j)}
			<line x1={pn.x} y1={pn.y} x2={ag.x} y2={ag.y} stroke={accent} stroke-width="1" opacity="0.28" />
		{/each}
	{/each}

	{#each g.pillars as pn, i (i)}
		<g>
			<line x1={pn.spokeStart.x} y1={pn.spokeStart.y} x2={pn.x} y2={pn.y} stroke={accent} stroke-width="1.6" opacity="0.45" />
			<path
				d={`M${pn.spokeStart.x} ${pn.spokeStart.y} L${pn.x} ${pn.y}`}
				fill="none"
				stroke={accent}
				stroke-width="1.8"
				stroke-linecap="round"
				pathLength="1"
				class="pbg-synapse"
				style={`animation-delay: ${(i % 6) * -0.8}s`}
			/>
		</g>
	{/each}

	{#each g.pillars as pn, i (i)}
		{#each pn.agents as ag, j (j)}
			<g>
				<circle cx={ag.x} cy={ag.y} r="5" fill="var(--bn-surface)" stroke={accent} stroke-width="1.4" />
				<text x={ag.x} y={ag.y + (ag.y < g.cy ? -9 : 15)} text-anchor="middle" font-family="var(--bn-font)" font-size="9" fill="var(--bn-text-2)">{trunc(ag.name, 16)}</text>
			</g>
		{/each}
	{/each}

	{#each g.pillars as pn, i (i)}
		<g>
			<circle cx={pn.x} cy={pn.y} r="21" fill="var(--bn-surface)" stroke={accent} stroke-width="2" />
			<circle cx={pn.x} cy={pn.y} r="6" fill={accent} />
			<text x={pn.x} y={pn.y + (pn.y < g.cy ? -30 : 40)} text-anchor="middle" font-family="var(--bn-font)" font-size="11" font-weight="700" fill="var(--bn-text)">{trunc(pn.pillar.name, 22)}</text>
		</g>
	{/each}

	<g>
		{#each g.coreDots as d, i (i)}
			<circle cx={d.x} cy={d.y} r={d.s} fill={accent} opacity="0.5" />
		{/each}
		<circle cx={g.cx} cy={g.cy} r="48" fill="none" stroke={accent} stroke-width="1.4" opacity="0.7" />
		<circle cx={g.cx} cy={g.cy} r="14" fill={`url(#${gid}-core)`} stroke={accent} stroke-width="1.6" />
		<text x={g.cx} y={g.cy + 72} text-anchor="middle" font-family="var(--bn-font)" font-size="10.5" font-weight="700" fill={accent}>{trunc(persona.name, 24)}</text>
		<text x={g.cx} y={g.cy + 88} text-anchor="middle" font-family="var(--bn-font)" font-size="8.5" letter-spacing="0.14em" fill="var(--bn-text-3)">KNOWLEDGE CORE</text>
	</g>
</svg>

<style>
	/* A travelling info-neuron along each core → pillar spoke (FounderOS v1
	   .kg-synapse: a calm sweep, 0.14 of the line, every 5s). */
	.pbg-synapse {
		stroke-dasharray: 0.14 0.86;
		animation: pbg-travel 5s linear infinite;
		opacity: 0.55;
	}
	@keyframes pbg-travel {
		from {
			stroke-dashoffset: 1;
		}
		to {
			stroke-dashoffset: 0;
		}
	}
	@media (prefers-reduced-motion: reduce) {
		.pbg-synapse {
			animation: none;
			opacity: 0.3;
		}
	}
</style>
