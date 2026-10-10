<!-- FounderOS v1 app/reference/page.tsx: one lens card per operating domain,
     its number, title and the systems it covers as item rows. -->
<script lang="ts">
	import type { Domain } from './types';

	let { domains }: { domains: Domain[] } = $props();
</script>

<div class="grid gap-3.5 sm:grid-cols-2 xl:grid-cols-4">
	{#each domains as domain (domain.id)}
		<div
			data-lens="r"
			data-domain={domain.id}
			class="bn-pressable is-row ref-card rounded-[var(--bn-r-panel)] border px-[17px] py-[15px]"
		>
			<div class="bn-accent font-mono text-[10px] tracking-[0.14em]">{String(domain.number).padStart(2, '0')}</div>
			<h2 class="bn-text mt-1.5 text-[13.5px] font-bold">{domain.title}</h2>
			<ul class="mt-2.5 flex flex-col gap-1.5">
				{#each domain.items as item (item)}
					<li class="ref-row bn-muted rounded-[var(--bn-r-chip)] border px-[9px] py-1.5 font-mono text-[10.5px]">{item}</li>
				{/each}
			</ul>
		</div>
	{/each}
</div>

<style>
	.ref-card {
		border-color: var(--bn-border);
		background: var(--bn-surface);
	}
	.ref-row {
		border-color: var(--bn-border);
		background: var(--bn-surface-2);
	}
</style>
