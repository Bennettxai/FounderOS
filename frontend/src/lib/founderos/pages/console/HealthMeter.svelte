<!-- Ten fixed cells, lit while i < score/10 (FounderOS v1 app/page.tsx
     HealthMeter). The score is the Optimal Engine's health (G-Brain is
     retired): a graded check, so ten cells say so. Unlit cells stay on the
     border colour so the track reads without competing with the lit run. -->
<script lang="ts">
	import { healthCells } from './console';
	let { value }: { value: number | null } = $props();
	const cells = $derived(healthCells(value));
	const hue = $derived(cells.tone === 'accent' ? 'var(--bn-accent)' : cells.tone === 'warn' ? 'var(--bn-warn)' : cells.tone === 'err' ? 'var(--bn-err)' : 'var(--bn-text-3)');
</script>

<div class="flex items-end gap-[2px]" style:height="18px" aria-hidden="true" data-lit={cells.lit}>
	{#each Array.from({ length: 10 }, (_, i) => i) as i (i)}
		<span class="bn-cell flex-1" style:height="5px" style:background={i < cells.lit ? hue : 'var(--bn-border)'}></span>
	{/each}
</div>

<style>
	.bn-cell {
		transition: background-color var(--bn-dur-lens, 160ms) var(--bn-ease-lens, ease);
	}
</style>
