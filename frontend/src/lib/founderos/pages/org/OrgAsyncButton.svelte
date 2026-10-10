<!-- FounderOS v1 components/AsyncButton.tsx for the org board: idle → spinner
     (optionally "running" + elapsed seconds) → ✓ pop, back to idle after
     1.4s. A run that threw reads ✗ in the error color instead of claiming a
     success that never happened. On the kit Pressable tones. -->
<script lang="ts">
	import { onDestroy, type Snippet } from 'svelte';
	import { Pressable } from '$lib/founderos/kit';

	let {
		run,
		children,
		busyLabel = '',
		doneLabel = 'done',
		failLabel = 'failed',
		failed = false,
		showElapsed = false,
		disabled = false,
		tone = 'primary',
		class: className = ''
	}: {
		run: () => Promise<unknown>;
		children: Snippet;
		busyLabel?: string;
		doneLabel?: string;
		failLabel?: string;
		/** The caller knows the run failed (it caught its own error). */
		failed?: boolean;
		showElapsed?: boolean;
		disabled?: boolean;
		tone?: 'primary' | 'secondary' | 'ghost';
		class?: string;
	} = $props();

	let phase = $state<'idle' | 'busy' | 'done'>('idle');
	let threw = $state(false);
	let elapsed = $state(0);
	let tick: ReturnType<typeof setInterval> | null = null;
	let back: ReturnType<typeof setTimeout> | null = null;
	const dark = $derived(tone === 'primary');
	const bad = $derived(failed || threw);

	async function onclick() {
		if (disabled || phase !== 'idle') return;
		phase = 'busy';
		threw = false;
		elapsed = 0;
		const since = Date.now();
		if (showElapsed) tick = setInterval(() => (elapsed = Math.floor((Date.now() - since) / 1000)), 1000);
		try {
			await run();
		} catch {
			threw = true;
		} finally {
			if (tick) clearInterval(tick);
			tick = null;
			phase = 'done';
			back = setTimeout(() => (phase = 'idle'), 1400);
		}
	}

	onDestroy(() => {
		if (tick) clearInterval(tick);
		if (back) clearTimeout(back);
	});
</script>

<Pressable {tone} {onclick} {disabled} aria-busy={phase === 'busy'} class="{disabled ? 'disabled:opacity-30' : ''} {className}">
	{#if phase === 'idle'}
		{@render children()}
	{:else if phase === 'busy'}
		<span
			class="inline-block h-[10px] w-[10px] rounded-full border-[1.5px]"
			style="animation: bn-om-spin 0.8s linear infinite; border-color: {dark ? 'var(--bn-accent-ink)' : 'var(--bn-ok)'}; border-right-color: transparent"
		></span>
		{busyLabel}
		{#if showElapsed}<span class="tabular-nums">{elapsed}s</span>{/if}
	{:else}
		<span class="bn-pop" style:color={bad ? 'var(--bn-err)' : dark ? null : 'var(--bn-ok)'}>{bad ? '✗' : '✓'}</span>
		{bad ? failLabel : doneLabel}
	{/if}
</Pressable>
