<!-- First-run welcome pop-up (FounderOS v1 components/CohortModal.tsx). The
     first time someone runs FounderOS and lands on Home it points at the
     cohort; dismissal persists to localStorage, so it shows once per browser.
     CohortBanner carries the same invite on every view. -->
<script lang="ts">
	import { onMount } from 'svelte';
	import { ArrowUpRight, X } from '$lib/founderos/icons';
	import { COHORT_SEEN, COHORT_STORAGE_KEY, COHORT_URL, shouldShowCohortModal } from '$lib/founderos/cohort';

	let { pathname }: { pathname: string } = $props();
	let open = $state(false);

	function stored(): string | null {
		try {
			return localStorage.getItem(COHORT_STORAGE_KEY);
		} catch {
			return null; // storage blocked: treat as a fresh visit
		}
	}

	// Storage is read after mount (and again on navigation), never during SSR.
	let mounted = $state(false);
	onMount(() => (mounted = true));
	$effect(() => {
		if (mounted) open = shouldShowCohortModal({ pathname, stored: stored() });
	});

	function dismiss() {
		open = false;
		try {
			localStorage.setItem(COHORT_STORAGE_KEY, COHORT_SEEN);
		} catch {
			/* storage blocked: it greets them again next run, no harm */
		}
	}

	function onKey(e: KeyboardEvent) {
		if (open && e.key === 'Escape') dismiss();
	}
</script>

<svelte:window onkeydown={onKey} />

{#if open}
	<div class="bn-overlay-in fixed inset-0 z-[150] flex items-center justify-center bg-black/75 p-4 backdrop-blur-sm sm:p-6" onclick={dismiss} role="presentation">
		<div
			class="cohort-panel bn-panel-in w-full max-w-lg"
			onclick={(e) => e.stopPropagation()}
			onkeydown={() => {}}
			role="dialog"
			tabindex="-1"
			aria-modal="true"
			aria-labelledby="cohort-modal-title"
		>
			<div class="cohort-head flex items-start justify-between px-5 py-3.5">
				<div class="bn-eyebrow bn-dim flex items-center gap-2 pt-1 font-mono text-[9.5px] uppercase tracking-[0.32em]">FounderOS Cohort</div>
				<button type="button" onclick={dismiss} aria-label="Close" class="cohort-btn bn-pressable bn-muted grid h-7 w-7 shrink-0 place-items-center">
					<X class="h-3.5 w-3.5" />
				</button>
			</div>
			<div class="px-5 py-5">
				<h2 id="cohort-modal-title" class="bn-text text-[20px] font-bold uppercase leading-[1.15] tracking-[0.06em]">Build this for real</h2>
				<p class="bn-muted mt-3 font-mono text-[12px] leading-relaxed">
					You're running the FounderOS demo locally — the operator console, the agent roster, the knowledge core, all of it.
				</p>
				<p class="bn-muted mt-3 font-mono text-[12px] leading-relaxed">
					Want help setting this up? Go to the FounderOS cohort to learn how to build the entire thing end-to-end and get it into production.
				</p>
				<div class="mt-6 flex flex-col gap-2 sm:flex-row sm:items-center">
					<a href={COHORT_URL} target="_blank" rel="noreferrer" onclick={dismiss} class="cohort-cta inline-flex items-center justify-center gap-1.5 px-4 py-2.5 font-mono text-[10px] font-bold uppercase tracking-[0.18em]">
						Join the cohort — founderos.sh
						<ArrowUpRight class="h-3 w-3" />
					</a>
					<button type="button" onclick={dismiss} class="cohort-btn bn-pressable bn-muted px-4 py-2.5 font-mono text-[10px] font-bold uppercase tracking-[0.18em]">Keep exploring</button>
				</div>
			</div>
		</div>
	</div>
{/if}

<style>
	.cohort-panel {
		background: var(--bn-bg);
		border: 1px solid var(--bn-border-strong);
		border-radius: var(--bn-r-panel);
	}
	.cohort-head {
		border-bottom: 1px solid var(--bn-border);
	}
	.cohort-btn {
		border: 1px solid var(--bn-border);
		border-radius: var(--bn-r-chip);
		transition:
			border-color var(--bn-dur-press) var(--bn-ease),
			color var(--bn-dur-press) var(--bn-ease);
	}
	.cohort-btn:hover {
		border-color: var(--bn-border-strong);
		color: var(--bn-text);
	}
	.cohort-cta {
		border: 1px solid var(--bn-accent);
		border-radius: var(--bn-r-chip);
		background: var(--bn-accent);
		color: var(--bn-accent-ink);
		transition: opacity var(--bn-dur-press) var(--bn-ease);
	}
	.cohort-cta:hover {
		opacity: 0.9;
	}
</style>
