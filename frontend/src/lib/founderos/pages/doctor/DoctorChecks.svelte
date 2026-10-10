<!-- The health ring and the checks, as one thing that reruns (FounderOS v1
     components/DoctorChecks.tsx). While a rerun reads, shimmer rows replace
     the checks: a check that says "ok" during a read is a lie for as long as
     the read takes. The ring transitions its conic gradient, never a radius. -->
<script lang="ts">
	import { Dot } from '$lib/founderos/kit';
	import { groupChecks } from './core';
	import type { DoctorCheck } from './types';

	let {
		checks,
		healthScore,
		connected,
		detail,
		busy = false
	}: { checks: DoctorCheck[]; healthScore: number | null; connected: boolean; detail: string; busy?: boolean } = $props();

	const SKELETON_WIDTHS = [180, 210, 160, 200, 170];
	const arc = $derived(`${((healthScore ?? 0) / 100) * 360}deg`);
	const mask = 'radial-gradient(farthest-side,transparent calc(100% - 5px),#000 calc(100% - 5px))';
	const dotOf = (s: string) => (s === 'ok' ? 'connected' : s === 'warn' ? 'warn' : 'error');
	// one line per check across engines (prod lists one engine's six checks)
	const lines = $derived(groupChecks(checks));
	const spin = 'radial-gradient(farthest-side,transparent calc(100% - 3px),#000 calc(100% - 3px))';
</script>

<div>
	<div class="flex items-center gap-4">
		<div class="relative h-[84px] w-[84px] shrink-0">
			<div
				class="absolute inset-0 rounded-full"
				style="background: conic-gradient(var(--bn-text) {arc}, var(--bn-hairline) 0); -webkit-mask-image: {mask}; mask-image: {mask}; transition: background 900ms cubic-bezier(.22,.61,.36,1)"
			></div>
			{#if busy}
				<div
					class="dc-checks-spin absolute inset-0 rounded-full"
					style="background: conic-gradient(var(--bn-text) 30deg, transparent 0); -webkit-mask-image: {spin}; mask-image: {spin}"
				></div>
			{/if}
			<div class="absolute inset-0 grid place-items-center">
				<span class="bn-text font-mono text-[19px] font-bold tabular-nums">{healthScore ?? '—'}</span>
			</div>
		</div>
		<div class="min-w-0">
			<div class="bn-dim font-mono text-[11px]">/ 100 health{connected ? '' : ' · no engine answered'}</div>
			<div class="bn-muted mt-1 font-mono text-[10.5px]">{busy ? 'reading the engines…' : `${lines.length} checks`}</div>
		</div>
	</div>

	<ul class="mt-3 space-y-1.5" aria-busy={busy}>
		{#if busy}
			{#each SKELETON_WIDTHS as w, i (i)}
				<li class="py-1"><span class="bn-skeleton block" style="width: {w}px; max-width: 100%"></span></li>
			{/each}
		{:else}
			{#each lines as check, i (check.name)}
				<li class="bn-enter flex items-start gap-2 text-[11px]" style="animation-delay: {i * 40}ms">
					<span class="mt-1"><Dot state={dotOf(check.status)} /></span>
					<span class="bn-muted min-w-0 break-words">
						<span class="bn-text font-semibold">{check.name}</span>{#if check.message}{' '}· {check.message}{/if}
					</span>
				</li>
			{/each}
			{#if checks.length === 0}
				<li class="bn-dim rounded-[var(--bn-r-panel)] border border-dashed px-3 py-2 font-mono text-[11px]" style="border-color: var(--bn-border)">
					doctor offline · {detail}
				</li>
			{/if}
		{/if}
	</ul>
</div>

<style>
	.dc-checks-spin {
		animation: dc-checks-spin 1.1s linear infinite;
	}
	@keyframes dc-checks-spin {
		to {
			transform: rotate(360deg);
		}
	}
</style>
