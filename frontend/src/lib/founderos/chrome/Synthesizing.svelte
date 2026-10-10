<!-- The dock's liveness readout (FounderOS v1 Synthesizing, dock variant): a
     blinking square and "synthesizing · Ns" while a reply is out. -->
<script lang="ts">
	import { onMount } from 'svelte';

	let { since }: { since?: number } = $props();
	let now = $state(Date.now());

	onMount(() => {
		const id = setInterval(() => (now = Date.now()), 1000);
		return () => clearInterval(id);
	});

	const secs = $derived(since ? Math.max(0, Math.floor((now - since) / 1000)) : null);
</script>

<span class="inline-flex items-center gap-2 font-mono text-[10px]" data-synthesizing>
	<span aria-hidden="true" class="bn-sq h-[6px] w-[6px] shrink-0"></span>
	<span class="bn-synth-shimmer truncate">synthesizing{secs !== null ? ` · ${secs}s` : ''}</span>
</span>

<style>
	.bn-sq {
		background: var(--bn-text);
		animation: bn-synth-blink 1s steps(1) infinite;
	}
	/* production om-blink: a hard on/off, no fade, like a terminal caret */
	@keyframes bn-synth-blink {
		0%,
		49% {
			opacity: 1;
		}
		50%,
		100% {
			opacity: 0;
		}
	}
	/* production .synth-shimmer: dim → text → dim swept across the word */
	.bn-synth-shimmer {
		background: linear-gradient(90deg, var(--bn-text-3) 0%, var(--bn-text) 50%, var(--bn-text-3) 100%);
		background-size: 200% 100%;
		-webkit-background-clip: text;
		background-clip: text;
		color: transparent;
		animation: bn-synth-sweep 1.6s linear infinite;
	}
	@keyframes bn-synth-sweep {
		from {
			background-position: 180% 0;
		}
		to {
			background-position: -20% 0;
		}
	}
	@media (prefers-reduced-motion: reduce) {
		.bn-sq,
		.bn-synth-shimmer {
			animation: none;
		}
	}
</style>
