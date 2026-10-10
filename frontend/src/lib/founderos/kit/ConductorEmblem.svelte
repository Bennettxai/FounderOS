<!-- The Conductor's living core (FounderOS v1 components/ConductorEmblem.tsx +
     the .conductor-emblem block of app/globals.css): a round core in a hairline
     ring that echoes the brain viz. Idle it stays quiet (machinery, not
     mascots); while an agent is being chatted with an accent arc orbits the
     ring, the core breathes and a halo pulses out. All motion is scoped to
     .thinking and off under prefers-reduced-motion. -->
<script lang="ts">
	import SparkIcon from './SparkIcon.svelte';
	import { conductorEmblemClasses } from './emblem';

	let {
		size = 32,
		shade = 'var(--bn-text)',
		thinking = false,
		class: className = ''
	}: { size?: number; shade?: string; thinking?: boolean; class?: string } = $props();

	// the silhouette sits at ~56% of the core so the ring and disc frame it
	const glyph = $derived(Math.round(size * 0.56));
</script>

<span class={conductorEmblemClasses(thinking, className)} style="width: {size}px; height: {size}px" data-thinking={thinking || undefined} aria-hidden="true">
	<span class="conductor-halo"></span>
	<span class="conductor-sweep"></span>
	<span class="conductor-ring"></span>
	<span class="conductor-core"><SparkIcon size={glyph} {shade} /></span>
</span>

<style>
	.conductor-emblem {
		position: relative;
		display: inline-grid;
		place-items: center;
		flex-shrink: 0;
		line-height: 0;
	}
	.conductor-emblem > * {
		grid-area: 1 / 1;
	}
	/* The one round mark in a square theme: Monolith forces every corner to 0
	   (theme/monolith.css); the emblem's disc and ring are the exception, as
	   they are in FounderOS v1. */
	:global(:root[data-founderos-theme]) .conductor-emblem > span {
		border-radius: 9999px !important;
	}
	.conductor-core {
		position: relative;
		z-index: 2;
		width: calc(100% - 7px);
		height: calc(100% - 7px);
		display: grid;
		place-items: center;
		border-radius: 9999px;
		background: radial-gradient(circle at 50% 40%, color-mix(in oklab, var(--bn-text) 12%, var(--bn-surface)), var(--bn-bg) 74%);
		box-shadow: inset 0 0 0 1px var(--bn-border-strong);
	}
	.conductor-ring {
		z-index: 1;
		width: 100%;
		height: 100%;
		border-radius: 9999px;
		border: 1px solid color-mix(in oklab, var(--bn-border-strong) 80%, transparent);
	}
	.conductor-sweep {
		z-index: 1;
		width: 100%;
		height: 100%;
		border-radius: 9999px;
		/* a comet: faint tail sharpening to a bright head, then gone */
		background: conic-gradient(
			from 0deg,
			transparent 0deg,
			color-mix(in oklab, var(--bn-accent) 35%, transparent) 34deg,
			var(--bn-accent) 82deg,
			transparent 104deg
		);
		-webkit-mask: radial-gradient(farthest-side, transparent calc(100% - 3px), black calc(100% - 3px));
		mask: radial-gradient(farthest-side, transparent calc(100% - 3px), black calc(100% - 3px));
		filter: drop-shadow(0 0 3px color-mix(in oklab, var(--bn-accent) 55%, transparent));
		opacity: 0;
	}
	.conductor-halo {
		z-index: 0;
		width: 100%;
		height: 100%;
		border-radius: 9999px;
		background: radial-gradient(circle, color-mix(in oklab, var(--bn-accent) 32%, transparent), transparent 68%);
		opacity: 0;
	}
	.conductor-emblem.thinking .conductor-sweep {
		opacity: 1;
		animation: conductor-sweep 1.1s linear infinite;
	}
	.conductor-emblem.thinking .conductor-halo {
		animation: conductor-halo 1.9s ease-out infinite;
	}
	.conductor-emblem.thinking .conductor-core {
		animation: conductor-breathe 1.9s ease-in-out infinite;
	}
	@keyframes conductor-sweep {
		to {
			transform: rotate(360deg);
		}
	}
	@keyframes conductor-halo {
		0% {
			transform: scale(0.7);
			opacity: 0.62;
		}
		70%,
		100% {
			transform: scale(2);
			opacity: 0;
		}
	}
	@keyframes conductor-breathe {
		0%,
		100% {
			transform: scale(1);
			box-shadow: inset 0 0 0 1px var(--bn-border-strong);
		}
		50% {
			transform: scale(1.05);
			box-shadow: inset 0 0 0 1px color-mix(in oklab, var(--bn-accent) 65%, var(--bn-border-strong));
		}
	}
	@media (prefers-reduced-motion: reduce) {
		.conductor-emblem.thinking .conductor-sweep {
			animation: none;
			opacity: 0.55;
		}
		.conductor-emblem.thinking .conductor-halo {
			animation: none;
			opacity: 0;
		}
		.conductor-emblem.thinking .conductor-core {
			animation: none;
		}
	}
</style>
