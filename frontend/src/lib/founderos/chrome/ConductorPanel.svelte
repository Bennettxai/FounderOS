<!-- The Conductor dock (FounderOS v1 components/ConductorPanel.tsx): a right-edge
     panel that knows which screen you are on. It loads the screen context
     (GET /pages/conductor/context), sends that context with every message to
     the real board thread (GET/POST /pages/conductor/chat), and `/ui <request>`
     dispatches a coding agent instead (POST /pages/conductor/dispatch). It
     PUSHES the page rather than covering it (--bn-conductor-w), resizes from
     its left edge with the width remembered, and closes on Escape. -->
<script lang="ts">
	import { onMount, tick } from 'svelte';
	import { ChevronRight, X } from '$lib/founderos/icons';
	import ConductorEmblem from '../kit/ConductorEmblem.svelte';
	import SparkIcon from '../kit/SparkIcon.svelte';
	import '../kit/kit.css';
	import { founderosFetch } from '../api';
	import { loadKey, saveKey } from './chrome';
	import ConductorComposer from './ConductorComposer.svelte';
	import Synthesizing from './Synthesizing.svelte';
	import {
		CONDUCTOR_DEFAULT_W,
		CONDUCTOR_OPEN_EVENT,
		CONDUCTOR_WIDTH_KEY,
		CONDUCTOR_W_VAR,
		FALLBACK_QUICK_ACTIONS,
		SLOW_AFTER_MS,
		SLOW_MESSAGE,
		assistantCount,
		clampConductorW,
		dispatchTurn,
		failureMessage,
		mergeTurns,
		threadToTurns,
		uiRequest,
		withScreen,
		type ScreenCtx,
		type ThreadMessage,
		type Turn
	} from './conductor';

	let { pathname }: { pathname: string } = $props();

	let open = $state(false);
	let width = $state(CONDUCTOR_DEFAULT_W);
	let ctx = $state<ScreenCtx | null>(null);
	let boardTurns = $state<Turn[]>([]);
	let localTurns = $state<Turn[]>([]);
	let sending = $state(false);
	let awaiting = $state(false);
	let awaitingSince = $state(0);
	let error = $state<string | null>(null);
	let threadError = $state<string | null>(null);
	let scroller: HTMLDivElement | undefined = $state();

	let baseline = 0;
	// A clear is local, not a board delete: the cockpit thread is the record.
	// Anything at or before it is dropped on every poll, or the refill would put
	// the transcript straight back.
	let clearedAt = 0;
	let drag: { startX: number; startW: number } | null = null;

	const turns = $derived(mergeTurns(boardTurns, localTurns));
	const quickActions = $derived(ctx?.quickActions?.length ? ctx.quickActions : FALLBACK_QUICK_ACTIONS);
	const lastReply = $derived([...turns].reverse().find((t) => t.role === 'assistant')?.content ?? null);

	onMount(() => {
		const stored = Number(loadKey(CONDUCTOR_WIDTH_KEY));
		if (Number.isFinite(stored) && stored > 0) width = clampConductorW(stored);
		const onOpen = () => (open = true);
		const onKey = (e: KeyboardEvent) => {
			if (e.key === 'Escape') open = false;
		};
		window.addEventListener(CONDUCTOR_OPEN_EVENT, onOpen);
		document.addEventListener('keydown', onKey);
		return () => {
			window.removeEventListener(CONDUCTOR_OPEN_EVENT, onOpen);
			document.removeEventListener('keydown', onKey);
			document.documentElement.style.setProperty(CONDUCTOR_W_VAR, '0px');
		};
	});

	// Push, don't cover: the /os shell reads this as its right margin.
	$effect(() => {
		document.documentElement.style.setProperty(CONDUCTOR_W_VAR, open ? `${width}px` : '0px');
	});

	// Only the latest request may land: navigate A→B and a slow answer for A
	// must not replace B's, or a send would carry the wrong screen.
	let ctxSeq = 0;
	async function loadContext(path: string) {
		const seq = ++ctxSeq;
		ctx = null;
		try {
			const body = await founderosFetch<ScreenCtx>(`/pages/conductor/context?path=${encodeURIComponent(path)}`);
			if (seq === ctxSeq) ctx = body;
		} catch {
			// the dock still works without context; the chat just loses screen grounding
		}
	}

	$effect(() => {
		if (open) void loadContext(pathname);
	});

	/** The board thread, or null when it could not be read (said so in the dock). */
	async function loadThread(): Promise<Turn[] | null> {
		try {
			const body = await founderosFetch<{ messages?: ThreadMessage[] | null }>('/pages/conductor/chat');
			threadError = null;
			return threadToTurns(body.messages ?? [], clearedAt);
		} catch (err) {
			threadError = `Thread unavailable: ${failureMessage(err)}`;
			return null;
		}
	}

	$effect(() => {
		if (!open) return;
		void loadThread().then((t) => {
			if (t && t.length > 0) boardTurns = t;
		});
	});

	// One poll while open: every 4s while a reply is pending, every ~12s at idle,
	// so a reply that lands late still shows without reopening the dock.
	$effect(() => {
		if (!open) return;
		let n = 0;
		const id = setInterval(async () => {
			n++;
			if (document.hidden) return;
			if (!awaiting && n % 3 !== 0) return;
			const thread = await loadThread();
			if (thread && thread.length > 0) {
				boardTurns = thread;
				if (awaiting && assistantCount(thread) > baseline) {
					awaiting = false;
					error = null;
				}
			}
			if (awaiting && Date.now() - awaitingSince > SLOW_AFTER_MS) {
				awaiting = false;
				error = SLOW_MESSAGE;
			}
		}, 4000);
		return () => clearInterval(id);
	});

	$effect(() => {
		void turns.length;
		void sending;
		void awaiting;
		void tick().then(() => {
			if (scroller) scroller.scrollTop = scroller.scrollHeight;
		});
	});

	function clearTurns() {
		clearedAt = Date.now();
		boardTurns = [];
		localTurns = [];
	}

	async function send(composed: string, display: string) {
		if (sending) return;
		sending = true;
		error = null;
		const now = Date.now();
		const bubble: Turn = { id: `optimistic-${now}`, role: 'user', content: display, createdAt: new Date(now).toISOString() };
		const request = uiRequest(display);
		try {
			if (request !== null) {
				// A coding agent on an isolated branch; never goes to the board thread.
				localTurns = [...localTurns, bubble];
				const res = await founderosFetch<{ workspaceId?: string; branch?: string }>('/pages/conductor/dispatch', {
					method: 'POST',
					json: { request }
				});
				localTurns = [...localTurns, dispatchTurn(request, res, Date.now())];
				return;
			}
			boardTurns = [...boardTurns, bubble];
			await founderosFetch('/pages/conductor/chat', { method: 'POST', json: { message: withScreen(composed, ctx) } });
			// The CEO answers asynchronously; the poll picks the reply up.
			baseline = assistantCount(boardTurns);
			awaitingSince = Date.now();
			awaiting = true;
		} catch (err) {
			error = failureMessage(err);
		} finally {
			sending = false;
		}
	}

	function onHandleDown(e: PointerEvent) {
		drag = { startX: e.clientX, startW: width };
		document.documentElement.classList.add('bn-conductor-dragging');
		try {
			(e.currentTarget as HTMLElement).setPointerCapture(e.pointerId);
		} catch {
			// synthetic or stale pointer; the move events still track
		}
	}
	function onHandleMove(e: PointerEvent) {
		if (!drag) return;
		width = clampConductorW(drag.startW + (drag.startX - e.clientX));
	}
	function onHandleUp() {
		if (!drag) return;
		drag = null;
		document.documentElement.classList.remove('bn-conductor-dragging');
		saveKey(CONDUCTOR_WIDTH_KEY, String(width));
	}
</script>

{#if !open}
	<!-- corner agent: tucked bottom-right, its label pops out on hover -->
	<button
		type="button"
		onclick={() => (open = true)}
		aria-label="Open the Conductor agent panel"
		title="Ask the Conductor about this screen"
		class="bn-pressable bn-corner group fixed bottom-5 right-5 z-40 flex items-center rounded-full border p-2.5"
	>
		<SparkIcon size={17} shade="var(--bn-text)" />
		<span class="bn-corner-label bn-muted overflow-hidden whitespace-nowrap font-mono text-[10.5px] tracking-wide">Ask Conductor</span>
	</button>
{/if}

<aside
	data-part="conductor"
	aria-hidden={!open}
	aria-label="Conductor"
	inert={!open}
	class="bn-dock fixed inset-y-0 right-0 z-50 flex max-w-[92vw] flex-col border-l"
	class:bn-open={open}
	style="width: {width}px"
>
	<!-- svelte-ignore a11y_no_static_element_interactions -->
	<div
		onpointerdown={onHandleDown}
		onpointermove={onHandleMove}
		onpointerup={onHandleUp}
		title="Drag to resize"
		class="bn-handle absolute inset-y-0 left-0 z-10 w-1.5 cursor-col-resize"
	></div>

	<!-- one edge control: › slides the dock away; widening is the handle's job -->
	<div class="bn-edge absolute -left-3 top-1/2 z-20 -translate-y-1/2">
		<button type="button" onclick={() => (open = false)} aria-label="Slide the panel away" title="Slide away" data-lens="c" class="bn-pressable is-dark bn-edge-btn bn-dim grid h-7 w-7 place-items-center rounded-full border">
			<ChevronRight size={14} />
		</button>
	</div>

	<header class="bn-rule flex items-center gap-2.5 border-b px-4 py-3">
		<ConductorEmblem size={32} thinking={sending} class="shrink-0" />
		<div class="min-w-0 flex-1">
			<div class="flex items-center gap-2">
				<span class="bn-text text-[12.5px] font-bold tracking-[0.12em]">CONDUCTOR</span>
				<!-- whatever is in the seat on the board; honest when it can't be read -->
				<span data-part="model-chip" class="bn-rule bn-dim truncate rounded-[6px] border px-1.5 py-px font-mono text-[9px]">{ctx?.model ?? 'model unknown'}</span>
			</div>
			<div class="bn-dim truncate font-mono text-[9.5px] uppercase tracking-wide">seeing: {ctx?.title ?? '…'}</div>
		</div>
		<button type="button" onclick={clearTurns} aria-label="Clear the transcript" title="Clear transcript" class="bn-pressable is-dark bn-ctl bn-dim shrink-0 rounded-[6px] px-1.5 py-1 font-mono text-[9px] uppercase tracking-wider">clear</button>
		<button type="button" onclick={() => (open = false)} aria-label="Close Conductor" class="bn-pressable is-dark bn-ctl bn-dim shrink-0 rounded-[6px] p-1"><X size={16} /></button>
	</header>

	{#if ctx}
		<p class="bn-rule bn-dim border-b px-4 py-2 font-mono text-[10px] leading-relaxed" title={ctx.context}>{ctx.context.split('\n')[0]}</p>
	{/if}

	<!-- the band: openers for THIS screen, always up, so an empty transcript is never all there is -->
	<div class="bn-rule border-b px-4 py-2.5">
		<div class="bn-dim mb-1.5 font-mono text-[8.5px] uppercase tracking-[0.2em]">quick actions · this screen</div>
		<div class="flex flex-wrap gap-1.5">
			{#each quickActions as a (a.label)}
				<button type="button" onclick={() => void send(a.prompt, a.label)} disabled={sending} class="bn-pressable is-dark bn-qa bn-muted h-6 rounded-[6px] border px-2 font-mono text-[9.5px] disabled:opacity-40">{a.label}</button>
			{/each}
		</div>
	</div>

	<div bind:this={scroller} class="flex-1 space-y-2 overflow-y-auto px-4 py-3">
		{#if turns.length === 0}
			<p class="bn-dim pt-6 text-center font-mono text-[10.5px] leading-relaxed">
				Nothing yet.<br />ask about this screen, or pick a quick action above
			</p>
		{/if}
		{#each turns as t (t.id)}
			{#if t.role === 'user'}
				<div class="text-right">
					<span class="bn-bubble-user bn-text inline-block max-w-[88%] break-words rounded-[8px] px-2.5 py-1.5 text-left text-[11.5px]">{t.content}</span>
				</div>
			{:else}
				<div class="text-left">
					{#if t.routedTo}
						<div class="bn-accent mb-0.5 font-mono text-[9px] uppercase tracking-wider">→ {t.routedTo}</div>
					{/if}
					<span class="bn-bubble bn-muted inline-block max-w-[92%] whitespace-pre-wrap break-words rounded-[8px] border px-2.5 py-1.5 text-[11.5px] leading-relaxed">{t.content}</span>
					<!-- the proof, not the prose: only a turn that did something carries one -->
					{#if t.receipt}
						<div class="bn-receipt mt-1 flex items-center gap-2 rounded-[6px] border px-2 py-1 font-mono text-[9.5px]">
							<span class="bn-ok bn-pop shrink-0">✓</span>
							<span class="bn-muted min-w-0 flex-1 truncate">{t.receipt.text}</span>
							{#if t.receipt.href}
								<a href={t.receipt.href} class="bn-pressable is-dark bn-text shrink-0 rounded-[6px] px-1">open →</a>
							{/if}
						</div>
					{/if}
				</div>
			{/if}
		{/each}
		{#if sending || awaiting}
			<div class="flex items-center gap-2" data-thinking>
				<ConductorEmblem size={18} thinking />
				<Synthesizing since={awaiting ? awaitingSince : undefined} />
			</div>
		{/if}
		{#if error}
			<p class="bn-err font-mono text-[10px]" role="alert">⚠ {error}</p>
		{/if}
		{#if threadError}
			<p class="bn-err font-mono text-[10px]">⚠ {threadError}</p>
		{/if}
	</div>

	<div class="bn-rule border-t p-3">
		<ConductorComposer
			onSend={send}
			disabled={sending}
			placeholder="Ask about {ctx?.title ?? 'this screen'}… (/ui to dispatch a change)"
			{lastReply}
			model={ctx?.model ?? null}
			onError={(m) => (error = m)}
		/>
	</div>
</aside>

<style>
	.bn-dock {
		border-color: var(--bn-border-strong);
		background: var(--bn-surface);
		transform: translateX(100%);
		transition: transform 420ms var(--bn-ease);
	}
	.bn-dock.bn-open {
		transform: none;
	}
	.bn-rule {
		border-color: var(--bn-border);
	}
	.bn-handle:hover {
		background: color-mix(in oklab, var(--bn-accent) 30%, transparent);
	}
	.bn-edge {
		opacity: 0;
		pointer-events: none;
		transition: opacity 300ms;
	}
	.bn-open .bn-edge {
		opacity: 1;
		pointer-events: auto;
	}
	.bn-edge-btn {
		border-color: var(--bn-border-strong);
		background: var(--bn-surface);
	}
	/* rest colors; hover and press are the .bn-pressable lens (kit.css) */
	.bn-qa {
		border-color: var(--bn-border);
		background: var(--bn-bg);
	}
	.bn-bubble-user {
		background: var(--bn-surface-2);
	}
	.bn-bubble {
		border-color: var(--bn-border);
		background: var(--bn-bg);
	}
	.bn-receipt {
		border-color: var(--bn-border);
		background: var(--bn-surface-2);
	}
	.bn-ok {
		color: var(--bn-ok);
	}
	.bn-err {
		color: var(--bn-err);
	}
	/* the corner agent: a pressable lens with no drop, on the house ease */
	.bn-corner {
		border-color: var(--bn-border-strong);
		background: color-mix(in oklab, var(--bn-surface) 90%, transparent);
		color: var(--bn-text);
		opacity: 0.6;
		backdrop-filter: blur(8px);
		transition-timing-function: var(--bn-ease);
		box-shadow: none !important;
	}
	.bn-corner:hover {
		opacity: 1;
	}
	.bn-corner-label {
		max-width: 0;
		transition:
			max-width 300ms var(--bn-ease),
			margin-left 300ms var(--bn-ease);
	}
	.bn-corner:hover .bn-corner-label {
		max-width: 130px;
		margin-left: 8px;
	}
	:global(html.bn-conductor-dragging) .bn-dock {
		transition: none;
	}
	@media (prefers-reduced-motion: reduce) {
		.bn-dock,
		.bn-corner-label {
			transition: none;
		}
	}
</style>
