<!-- The Conductor rail beside the board (FounderOS v1 components/ConductorChat.tsx):
     the REAL CEO on the company board, through the standing cockpit thread
     (GET/POST /pages/conductor/chat). The thread polls continuously (4s while a
     reply is pending, ~12s at idle, paused while the tab is hidden). Sending is
     an outward board write: with FOUNDEROS_WRITES=0 the bridge guard holds it and
     the rail says so; the message is never shown as sent. -->
<script lang="ts">
	import { onDestroy, onMount, tick } from 'svelte';
	import { ArrowUpRight } from '$lib/founderos/icons';
	import { founderosFetch, isGuardRefusal } from '$lib/founderos/api';
	import ConductorEmblem from '$lib/founderos/kit/ConductorEmblem.svelte';
	import ConductorComposer from '$lib/founderos/chrome/ConductorComposer.svelte';
	import Synthesizing from '$lib/founderos/chrome/Synthesizing.svelte';

	type WireMsg = { id: string; body: string; authorType: 'user' | 'agent'; createdAt: string };
	type Turn = { id: string; role: 'user' | 'assistant'; content: string };

	const WAIT_LIMIT_MS = 150_000;

	let {
		model = null,
		boardUrl = null,
		bare = false
	}: {
		model?: string | null;
		boardUrl?: string | null;
		/** Inside a shell that already frames it (the /chats hub): no border of its own. */
		bare?: boolean;
	} = $props();

	let turns = $state<Turn[]>([]);
	let sending = $state(false);
	let awaiting = $state(false);
	let awaitingSince = $state(0);
	let assistantBaseline = 0;
	let error = $state<string | null>(null);
	let scroller: HTMLDivElement | undefined = $state();
	let poll: ReturnType<typeof setInterval> | null = null;

	async function loadThread(): Promise<Turn[] | null> {
		try {
			const b = await founderosFetch<{ messages?: WireMsg[] | null }>('/pages/conductor/chat');
			return (b.messages ?? []).map((m) => ({ id: m.id, role: m.authorType === 'agent' ? 'assistant' : 'user', content: m.body }));
		} catch {
			// prod: a failed read is "nothing new"; the next poll tries again
			return null;
		}
	}

	function applyThread(thread: Turn[] | null) {
		if (!thread || thread.length === 0) return;
		turns = thread.slice(-14);
		const assistants = thread.filter((t) => t.role === 'assistant').length;
		if (awaiting && assistants > assistantBaseline) {
			awaiting = false;
			error = null;
		}
	}

	onMount(() => {
		void loadThread().then(applyThread);
		let n = 0;
		poll = setInterval(async () => {
			n++;
			if (typeof document !== 'undefined' && document.hidden) return;
			if (!awaiting && n % 3 !== 0) return;
			applyThread(await loadThread());
			if (awaiting && Date.now() - awaitingSince > WAIT_LIMIT_MS) {
				awaiting = false;
				error = 'The CEO run is taking a while. Its reply lands in this thread automatically.';
			}
		}, 4000);
	});
	onDestroy(() => poll && clearInterval(poll));

	// keep the newest message (or the thinking bubble) in view
	$effect(() => {
		void turns;
		void awaiting;
		void tick().then(() => {
			if (scroller) scroller.scrollTop = scroller.scrollHeight;
		});
	});

	async function send(message: string, display: string) {
		if (sending) return;
		sending = true;
		error = null;
		const optimistic: Turn = { id: `optimistic-${Date.now()}`, role: 'user', content: display };
		turns = [...turns, optimistic];
		try {
			await founderosFetch('/pages/conductor/chat', { method: 'POST', json: { message } });
			awaitingSince = Date.now();
			awaiting = true;
			// baseline from the FULL thread, so an old reply is not taken for this one
			assistantBaseline = ((await loadThread()) ?? []).filter((t) => t.role === 'assistant').length;
		} catch (err) {
			// never leave a held or failed message looking sent
			turns = turns.filter((t) => t.id !== optimistic.id);
			error = isGuardRefusal(err)
				? 'Message did not reach the board: writes are off (FOUNDEROS_WRITES=0), nothing was sent.'
				: `Message did not reach the board: ${err instanceof Error ? err.message : String(err)}`;
		} finally {
			sending = false;
		}
	}

	const lastReply = $derived([...turns].reverse().find((t) => t.role === 'assistant')?.content ?? null);
</script>

<section data-part="conductor-rail" class="flex h-full min-h-[320px] flex-col p-4 {bare ? '' : 'bn-rail rounded-2xl border'}">
	<div class="flex shrink-0 items-center gap-2.5">
		<ConductorEmblem size={38} thinking={sending || awaiting} />
		<div class="min-w-0">
			<div class="flex items-center gap-2">
				<span aria-hidden="true" class="bn-live h-1.5 w-1.5 shrink-0"></span>
				<span class="bn-text text-[13px] font-bold tracking-[0.12em]">CONDUCTOR</span>
				<span class="bn-pill-sm bn-dim rounded-full border px-1.5 py-0.5 font-mono text-[8.5px]">board · {model ?? 'model unknown'}</span>
			</div>
			<div class="bn-dim font-mono text-[10px]">the real CEO on the company board · delegates, creates tasks, reads your data</div>
		</div>
		<div class="ml-auto flex shrink-0 items-center gap-2.5">
			{#if awaiting}<span class="font-mono text-[9.5px] uppercase tracking-wider" style:color="var(--bn-warn)">CEO working</span>{/if}
			{#if boardUrl}
				<a href={boardUrl} target="_blank" rel="noreferrer" class="bn-dim bn-linky flex items-center gap-1 font-mono text-[10px]">open on board <ArrowUpRight size={12} /></a>
			{/if}
		</div>
	</div>

	{#if turns.length > 0 || awaiting}
		<div bind:this={scroller} class="mt-3 min-h-0 flex-1 space-y-1.5 overflow-y-auto pr-1">
			{#each turns as t (t.id)}
				{#if t.role === 'user'}
					<div class="text-right">
						<span class="bn-user inline-block rounded-md max-w-[85%] break-words px-2.5 py-1 text-[11.5px]">{t.content}</span>
					</div>
				{:else}
					<div class="text-left">
						<div class="bn-accent mb-0.5 font-mono text-[9.5px] uppercase tracking-wider">→ Conductor · board</div>
						<span class="bn-reply inline-block rounded-md max-w-[85%] whitespace-pre-wrap break-words px-2.5 py-1 text-[11.5px]">{t.content}</span>
					</div>
				{/if}
			{/each}
			{#if awaiting}
				<div class="text-left" data-thinking>
					<div class="bn-accent mb-0.5 font-mono text-[9.5px] uppercase tracking-wider">→ Conductor · board</div>
					<span class="bn-reply inline-flex rounded-xl items-center px-3 py-2"><Synthesizing since={awaitingSince} /></span>
				</div>
			{/if}
		</div>
	{/if}

	<div class="mt-auto shrink-0 pt-3">
		<ConductorComposer onSend={send} disabled={sending} {model} {lastReply} placeholder="Message the CEO…" onError={(m) => (error = m)} />
	</div>
	{#if error}<p class="mt-1.5 shrink-0 font-mono text-[10px]" style:color="var(--bn-err)">⚠ {error}</p>{/if}
</section>

<style>
	.bn-rail {
		border-color: var(--bn-border-strong);
		background: var(--bn-surface);
	}
	.bn-live {
		background: var(--bn-ok);
		animation: bn-rail-pulse 2s ease-in-out infinite;
	}
	.bn-pill-sm {
		border-color: var(--bn-border);
	}
	.bn-accent {
		color: var(--bn-accent);
	}
	.bn-user {
		background: var(--bn-surface-2);
		color: var(--bn-text);
	}
	.bn-reply {
		background: var(--bn-surface-3);
		color: var(--bn-text-2);
	}
	@keyframes bn-rail-pulse {
		50% {
			opacity: 0.4;
		}
	}
	@media (prefers-reduced-motion: reduce) {
		.bn-live {
			animation: none;
		}
	}
</style>
