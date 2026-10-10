<!-- Interject: throw a thought at the OS without picking an app first. Free
     text routes itself (task words → board, tell/ask → agent relay, else an
     Optimal Engine note); a chip pins the route. The receipt is honest: where
     it actually landed, or the error. -->
<script lang="ts">
	import { onDestroy } from 'svelte';
	import { FounderosApiError, founderosFetch } from '$lib/founderos/api';
	import { Label, Pressable, ToggleChip } from '$lib/founderos/kit';

	type Route = 'task' | 'agent' | 'note';
	type Receipt = { ok: boolean; route: Route; ref?: string; url?: string; slug?: string; error?: string };

	// Labelled by destination; no "auto" chip: auto is the unpinned state.
	const ROUTES: { id: Route; label: string }[] = [
		{ id: 'note', label: '→ Optimal Engine' },
		{ id: 'task', label: '→ Board' },
		{ id: 'agent', label: '→ Agent' }
	];
	const VERB: Record<Route, string> = { task: 'on the board', agent: 'relayed via the board', note: 'captured to the Optimal Engine' };

	let text = $state('');
	let route = $state<Route | null>(null);
	let receipt = $state<Receipt | null>(null);
	let busy = $state(false);
	/** prod AsyncButton: the disc shows ✓ for 1.4s after a send lands. */
	let done = $state(false);
	let timer: ReturnType<typeof setTimeout> | undefined;
	let doneTimer: ReturnType<typeof setTimeout> | undefined;
	onDestroy(() => {
		clearTimeout(timer);
		clearTimeout(doneTimer);
	});

	async function send() {
		const body = text.trim();
		if (!body || busy) return;
		busy = true;
		let r: Receipt;
		try {
			r = await founderosFetch<Receipt>('/pages/interject', { method: 'POST', json: { text: body, ...(route ? { route } : {}) } });
		} catch (e) {
			const b = e instanceof FounderosApiError ? (e.body as Partial<Receipt> | undefined) : undefined;
			r = { ok: false, route: b?.route ?? route ?? 'note', error: b?.error ?? (e instanceof Error ? e.message : 'network error') };
		}
		busy = false;
		done = true;
		clearTimeout(doneTimer);
		doneTimer = setTimeout(() => (done = false), 1400);
		receipt = r;
		if (r.ok) text = '';
		clearTimeout(timer);
		timer = setTimeout(() => (receipt = null), 5000);
	}

	function onKey(e: KeyboardEvent) {
		if (e.key === 'Enter' && !e.shiftKey) {
			e.preventDefault();
			void send();
		}
	}
</script>

<section data-part="card" class="bn-card">
	<div data-part="card-head" class="flex items-center gap-3 border-b px-4 py-2.5" style:border-color="var(--bn-border)">
		<Label>Interject</Label>
		<span class="bn-dim font-mono text-[10px]">task · agent · note — routed for you</span>
	</div>
	<div class="flex flex-col gap-2.5 p-4">
		<textarea
			bind:value={text}
			onkeydown={onKey}
			rows="3"
			data-part="input"
			placeholder="call Nick back re: Tampa scope…  (Enter sends, Shift+Enter breaks)"
			class="bn-interject-input bn-state-fade bn-text w-full resize-none rounded-[6px] border px-3 py-2.5 font-mono text-[12.5px] outline-none"
		></textarea>
		<div class="flex items-center gap-1.5">
			{#each ROUTES as r (r.id)}
				<!-- clicking the chip that is already on unpins it, back to auto -->
				<ToggleChip on={route === r.id} aria-pressed={route === r.id} onclick={() => (route = route === r.id ? null : r.id)}>{r.label}</ToggleChip>
			{/each}
			<span class="bn-dim ml-auto font-mono text-[9.5px]">
				{text.trim() ? (route ? 'route pinned' : 'auto-routed · Enter sends') : 'Enter sends · Shift+Enter breaks'}
			</span>
			<!-- the artboard's send is a 26px white disc carrying ↑, not a
			     labelled button: the label is already in the hint beside it -->
			<Pressable
				tone="primary"
				aria-label="send"
				aria-busy={busy}
				disabled={!text.trim() || busy}
				onclick={() => void send()}
				class="grid! h-[26px] w-[26px] place-items-center rounded-full! p-0! font-mono text-[12px] {text.trim() ? '' : 'opacity-40'}"
				>{#if busy}<span class="bn-send-spin inline-block h-[10px] w-[10px] rounded-full border-[1.5px]"></span>{:else if done}<span class="bn-pop">✓</span>{:else}↑{/if}</Pressable
			>
		</div>
		{#if receipt}
			<div class="bn-enter flex items-center gap-2 font-mono text-[11px]" style:color={receipt.ok ? 'var(--bn-text-2)' : 'var(--bn-err)'}>
				{#if receipt.ok}
					<span class="font-bold" style:color="var(--bn-ok)">✓</span>
					<span>{receipt.ref ? `${receipt.ref} · ` : ''}{receipt.slug ? `${receipt.slug} · ` : ''}{VERB[receipt.route]}</span>
					{#if receipt.url}<a href={receipt.url} target="_blank" rel="noreferrer" class="bn-accent">open →</a>{/if}
				{:else}
					<span>failed to land: {receipt.error ?? 'unknown error'}</span>
				{/if}
			</div>
		{/if}
	</div>
</section>

<style>
	.bn-interject-input {
		border-color: var(--bn-border);
		background: var(--bn-bg);
	}
	.bn-interject-input::placeholder {
		color: var(--bn-text-3);
	}
	.bn-interject-input:focus {
		border-color: var(--bn-border-strong);
	}
	.bn-send-spin {
		border-color: var(--bn-accent-ink);
		border-right-color: transparent;
		animation: bn-om-spin 0.8s linear infinite;
	}
</style>
