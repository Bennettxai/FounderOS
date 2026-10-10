<!-- A labelled hand-off between two stages (FounderOS v1 app/doctor/page.tsx
     Arrow): a rule, the label set vertically and a down arrow on wide
     screens; a flat "label →" when the stages stack. -->
<script lang="ts">
	let { label }: { label: string } = $props();
</script>

<div data-part="arrow" class="flex shrink-0 items-center justify-center self-stretch px-1 py-2 xl:flex-col">
	<div class="flex items-center gap-1 xl:flex-col">
		<span class="hidden h-px w-6 xl:block xl:h-6 xl:w-px" style="background: var(--bn-border-strong)"></span>
		<span class="bn-dim font-mono text-[10px] uppercase tracking-widest xl:[writing-mode:vertical-rl]">{label}</span>
		<span class="bn-muted xl:rotate-90">→</span>
	</div>
</div>
