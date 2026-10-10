<!-- The toast stack (FounderOS v1 components/Toaster.tsx): bottom-right, newest on
     top, max 4. The 200ms sweep doubles as the tick that advances each timed
     toast's progress hairline. Mounted once by the /os layout; anything calls
     `toast.*` from ./toast. -->
<script lang="ts">
	import { onMount } from 'svelte';
	import '../kit/kit.css';
	import { hold, sweep, toast, toasts, type ToastKind } from './toast';

	let now = $state(Date.now());

	onMount(() => {
		const iv = setInterval(() => {
			now = Date.now();
			sweep(now);
		}, 200);
		return () => clearInterval(iv);
	});

	const glyph = (k: ToastKind) => (k === 'ok' ? '✓' : k === 'warn' ? '!' : '✕');
</script>

<div data-part="toasts" class="pointer-events-none fixed bottom-4 right-4 z-[60] flex w-[300px] flex-col-reverse gap-1.5">
	{#each $toasts as t (t.id)}
		<!-- svelte-ignore a11y_no_static_element_interactions -->
		<div
			data-toast
			data-kind={t.kind}
			data-lens="r"
			role={t.kind === 'err' ? 'alert' : 'status'}
			onmouseenter={() => hold(t.id, true)}
			onmouseleave={() => hold(t.id, false)}
			class="bn-pressable bn-toast bn-enter pointer-events-auto relative flex items-center gap-2 rounded-[10px] border px-2.5 py-2 font-mono text-[10.5px]"
		>
			{#if t.kind === 'busy'}
				<span class="bn-toast-spin inline-block h-[10px] w-[10px] rounded-full border-[1.5px]" aria-hidden="true"></span>
			{:else}
				<span class="bn-pop bn-toast-glyph" aria-hidden="true">{glyph(t.kind)}</span>
			{/if}
			<span class="bn-text min-w-0 flex-1 truncate" title={t.text}>{t.text}</span>
			{#if t.undo}
				<button
					type="button"
					onclick={() => {
						t.undo?.();
						toast.close(t.id);
					}}
					class="bn-pressable bn-toast-undo text-[10px] underline">undo</button
				>
			{/if}
			<button type="button" aria-label="Dismiss" onclick={() => toast.close(t.id)} class="bn-pressable bn-toast-x px-0.5 text-[12px]">×</button>
			{#if t.ttl !== Infinity}
				<span
					data-part="ttl"
					aria-hidden="true"
					class="bn-toast-ttl absolute bottom-0 left-2 right-2 h-px origin-left opacity-50"
					style="transform: scaleX({t.held ? 1 : Math.max(0, 1 - (now - t.born) / t.ttl)})"
				></span>
			{/if}
		</div>
	{/each}
</div>

<style>
	.bn-toast {
		border-color: var(--bn-border-strong);
		background: var(--bn-bg);
		color: var(--bn-text);
		box-shadow: 0 12px 30px rgba(0, 0, 0, 0.7);
	}
	.bn-toast[data-kind='err'] {
		border-color: var(--bn-err);
	}
	/* status colours come from the theme, so every skin reads honestly */
	.bn-toast[data-kind='ok'] { --tone: var(--bn-ok); }
	.bn-toast[data-kind='warn'] { --tone: var(--bn-warn); }
	.bn-toast[data-kind='err'] { --tone: var(--bn-err); }
	.bn-toast[data-kind='busy'] { --tone: var(--bn-text); }
	.bn-toast-glyph {
		color: var(--tone);
	}
	.bn-toast-ttl {
		background: var(--tone);
		transition: transform 200ms linear;
	}
	.bn-toast-spin {
		border-color: var(--bn-text);
		border-right-color: transparent;
		animation: bn-om-spin 0.8s linear infinite;
	}
	.bn-toast-undo {
		color: var(--bn-text-2);
	}
	.bn-toast-undo:hover,
	.bn-toast-x:hover {
		color: var(--bn-text);
	}
	.bn-toast-x {
		color: var(--bn-text-3);
	}
	@media (prefers-reduced-motion: reduce) {
		.bn-toast-spin {
			animation-duration: 2.4s;
		}
	}
</style>
