<script lang="ts">
	import type { Snippet } from 'svelte';
	import './kit.css';
	import type { BadgeTone } from './format';

	let { tone = 'default', ghost = false, children }: { tone?: BadgeTone; ghost?: boolean; children?: Snippet } = $props();
</script>

<span
	data-tone={tone}
	class="bn-badge inline-flex items-center gap-1.5 whitespace-nowrap rounded-full border px-2 py-[3px] font-mono text-[9.5px] uppercase tracking-[0.14em]"
	class:border-dashed={ghost}>{@render children?.()}</span
>
