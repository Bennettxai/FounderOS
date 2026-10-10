<!-- Permanent footer CTA (FounderOS v1 components/CohortBanner.tsx): rendered
     once in the /os layout under the page, so it lands at the bottom of every
     view. Static markup, no state. -->
<script lang="ts">
	import { ArrowUpRight } from '$lib/founderos/icons';
	import { COHORT_CTA, COHORT_URL } from '$lib/founderos/cohort';
</script>

<footer class="cohort mt-14 pt-5">
	<div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
		<div class="min-w-0">
			<div class="bn-eyebrow bn-dim mb-1.5 flex items-center gap-2 font-mono text-[9.5px] uppercase tracking-[0.32em]">FounderOS Cohort</div>
			<p class="bn-muted max-w-[70ch] font-mono text-[11.5px] leading-relaxed">{COHORT_CTA}</p>
		</div>
		<a href={COHORT_URL} target="_blank" rel="noreferrer" class="cohort-link bn-text inline-flex shrink-0 items-center gap-1.5 self-start whitespace-nowrap px-3 py-2 font-mono text-[10px] font-bold uppercase tracking-[0.18em] sm:self-auto">
			founderos.sh
			<ArrowUpRight class="h-3 w-3" />
		</a>
	</div>
</footer>

<style>
	.cohort {
		border-top: 1px solid var(--bn-border);
	}
	.cohort-link {
		border: 1px solid var(--bn-border-strong);
		border-radius: var(--bn-r-chip);
		transition:
			background-color var(--bn-dur-press) var(--bn-ease),
			border-color var(--bn-dur-press) var(--bn-ease);
	}
	.cohort-link:hover {
		border-color: var(--bn-accent);
		background: var(--bn-surface-2);
	}
</style>
