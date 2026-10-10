<!-- The Claude-style chat hub (FounderOS v1 components/ChatHub.tsx): ONE
     rounded shell holding a collapsible conversation rail and the open thread,
     equal height, everything scrolling inside. Collapse the rail and the strip
     keeps every conversation one click away as avatars. "New chat" flips the
     rail into the roster picker. The Conductor row renders the real cockpit
     thread (bare, so the shell does not double-frame it); agent rows use the
     stored agent_messages through GET/POST /pages/chats/:id. With no model key
     a send comes back configured:false and the hub says so; nothing is shown
     as sent. -->
<script lang="ts">
	import { tick } from 'svelte';
	import { Crown, PanelLeft, Plus, X } from '$lib/founderos/icons';
	import { founderosFetch } from '$lib/founderos/api';
	import Label from '$lib/founderos/kit/Label.svelte';
	import ConductorComposer from '$lib/founderos/chrome/ConductorComposer.svelte';
	import Synthesizing from '$lib/founderos/chrome/Synthesizing.svelte';
	import ConductorRail from '../agents/ConductorRail.svelte';
	import { CONDUCTOR, bumpSummary, initials, railAgo } from './chats';
	import type { AgentMessage, ChatSendResult, ConversationSummary, RosterAgent } from './types';

	let {
		initialSummaries,
		roster,
		conductorModel = null,
		boardUrl = null
	}: {
		initialSummaries: ConversationSummary[];
		roster: RosterAgent[];
		conductorModel?: string | null;
		boardUrl?: string | null;
	} = $props();

	// svelte-ignore state_referenced_locally
	let summaries = $state<ConversationSummary[]>(initialSummaries);
	let selected = $state<string>(CONDUCTOR);
	let picking = $state(false);
	let query = $state('');
	let railOpen = $state(true);
	let messages = $state<AgentMessage[]>([]);
	let sending = $state<number | null>(null);
	let error = $state<string | null>(null);
	let loadingThread = $state(false);
	let scroller: HTMLDivElement | undefined = $state();

	const nameOf = (id: string) => (id === CONDUCTOR ? 'Conductor · CEO' : (roster.find((a) => a.id === id)?.name ?? summaries.find((s) => s.agentId === id)?.agentName ?? id));
	const q = $derived(query.trim().toLowerCase());
	const direct = $derived(summaries.filter((c) => c.agentId !== 'conductor'));

	async function loadThread(agentId: string) {
		loadingThread = true;
		try {
			const body = await founderosFetch<{ messages: AgentMessage[] }>(`/pages/chats/${encodeURIComponent(agentId)}`);
			if (selected === agentId) messages = body.messages ?? [];
		} catch {
			/* keep the last view on a transient failure */
		} finally {
			loadingThread = false;
		}
	}

	function pick(id: string) {
		selected = id;
		picking = false;
		error = null;
		messages = [];
		if (id !== CONDUCTOR) void loadThread(id);
	}

	// keep the thread pinned to the newest message
	$effect(() => {
		void messages.length;
		void sending;
		void tick().then(() => scroller?.scrollTo?.({ top: scroller.scrollHeight }));
	});

	async function send(outbound: string, display: string) {
		if (selected === CONDUCTOR) return; // the cockpit rail owns its own send
		const target = selected;
		const optimistic: AgentMessage = { id: `local-${Date.now()}`, agentId: target, role: 'user', content: display, toolCalls: [], createdAt: new Date().toISOString() };
		messages = [...messages, optimistic];
		sending = Date.now();
		error = null;
		try {
			const body = await founderosFetch<ChatSendResult>(`/pages/chats/${encodeURIComponent(target)}`, { method: 'POST', json: { message: outbound } });
			if (selected !== target) return;
			if (!body.configured) {
				// Nothing was stored; keep what was typed in view (as v1 does) with the reason.
				error = body.error ?? 'Agent chat is not configured.';
				return;
			}
			messages = body.messages ?? [];
			const last = messages[messages.length - 1];
			if (last) summaries = bumpSummary(summaries, target, nameOf(target), last, messages.length);
		} catch (err) {
			// The message stays in view (as v1 does); the error says it did not land.
			error = err instanceof Error ? err.message : String(err);
		} finally {
			sending = null;
		}
	}
</script>

{#snippet railRow(id: string, name: string, sub: string, at?: string)}
	<button
		type="button"
		data-part="rail-row"
		data-agent={id}
		data-lens="r"
		onclick={() => pick(id)}
		class="bn-pressable is-row bn-row relative mx-1.5 my-0.5 flex w-[calc(100%-12px)] flex-col gap-0.5 px-2.5 py-2 text-left"
		class:is-selected={selected === id}
	>
		{#if selected === id}<span aria-hidden="true" class="bn-row-mark absolute bottom-1.5 left-0 top-1.5 w-[2px]"></span>{/if}
		<span class="flex items-baseline gap-2">
			<span aria-hidden="true" class="h-1.5 w-1.5 shrink-0 self-center {id === CONDUCTOR ? 'bn-dot-live' : 'bn-dot-dim'}"></span>
			{#if id === CONDUCTOR}<Crown class="bn-accent h-3 w-3 shrink-0 self-center" />{/if}
			<span class="bn-text min-w-0 flex-1 truncate text-[12px] font-semibold">{name}</span>
			<span class="bn-dim shrink-0 font-mono text-[9px]">{id === CONDUCTOR ? 'pinned' : at ? railAgo(at) : ''}</span>
		</span>
		<span class="bn-dim truncate font-mono text-[10px]">{sub}</span>
	</button>
{/snippet}

{#snippet stripDot(id: string, name: string)}
	<button
		type="button"
		title={name}
		onclick={() => pick(id)}
		class="bn-pressable bn-strip mx-auto flex h-8 w-8 items-center justify-center rounded-xl text-[9.5px] font-bold"
		class:is-selected={selected === id}
	>
		{#if id === CONDUCTOR}<Crown class="bn-accent h-3.5 w-3.5" />{:else}{initials(name)}{/if}
	</button>
{/snippet}

<div
	data-part="chat-hub"
	class="bn-hub grid min-h-[560px] flex-1 overflow-hidden rounded-2xl border"
	style="height: 100%; grid-template-columns: {railOpen ? '280px minmax(0,1fr)' : '46px minmax(0,1fr)'}"
>
	<!-- conversation rail ↔ collapsed avatar strip -->
	<div class="bn-railcol flex h-full min-h-0 flex-col border-r">
		<div class="bn-line flex shrink-0 items-center border-b py-2 {railOpen ? 'justify-between px-3' : 'justify-center px-1'}">
			{#if railOpen}<Label>Conversations</Label>{/if}
			<button
				type="button"
				onclick={() => (railOpen = !railOpen)}
				title={railOpen ? 'Collapse conversations' : 'Expand conversations'}
				class="bn-pressable bn-dim bn-hover-text"
			>
				<PanelLeft class="h-3.5 w-3.5" />
			</button>
		</div>
		{#if railOpen}
			<div class="bn-line flex shrink-0 items-center gap-1.5 border-b px-3 py-1.5">
				<input
					bind:value={query}
					placeholder="search threads"
					class="bn-search bn-text min-w-0 flex-1 border px-2 py-1 font-mono text-[10px] focus:outline-none"
				/>
				<button
					type="button"
					onclick={() => (picking = !picking)}
					title="New chat"
					class="bn-pressable bn-newchat bn-muted shrink-0 border p-1.5"
				>
					{#if picking}<X class="h-3 w-3" />{:else}<Plus class="h-3 w-3" />{/if}
				</button>
			</div>
			<div class="min-h-0 flex-1 overflow-y-auto overscroll-contain py-1">
				{#if picking}
					{#each roster.filter((a) => a.name.toLowerCase().includes(q)) as a (a.id)}
						{@render railRow(a.id, a.name, a.description.slice(0, 60))}
					{/each}
				{:else}
					{@render railRow(CONDUCTOR, 'Conductor · CEO', 'the live board thread on the host')}
					{#each direct.filter((c) => c.agentName.toLowerCase().includes(q)) as c (c.agentId)}
						{@render railRow(c.agentId, c.agentName, c.lastMessage, c.lastAt)}
					{/each}
					{#if summaries.length === 0}
						<p class="bn-dim px-3 py-3 font-mono text-[10px]">No direct chats yet. Hit New chat and pick an agent.</p>
					{/if}
				{/if}
			</div>
		{:else}
			<div class="min-h-0 flex-1 space-y-1 overflow-y-auto overscroll-contain py-2">
				{@render stripDot(CONDUCTOR, 'Conductor · CEO')}
				{#each direct as c (c.agentId)}
					{@render stripDot(c.agentId, c.agentName)}
				{/each}
			</div>
		{/if}
	</div>

	<!-- thread pane: same height as the rail, always -->
	{#if selected === CONDUCTOR}
		<div class="h-full min-h-0">
			<ConductorRail model={conductorModel} {boardUrl} bare />
		</div>
	{:else}
		<div class="flex h-full min-h-0 flex-col">
			<div class="bn-line flex shrink-0 items-center gap-2 border-b px-4 py-2">
				<span aria-hidden="true" class="bn-dot-ok h-1.5 w-1.5 shrink-0"></span>
				<span class="bn-text text-[12.5px] font-semibold">{nameOf(selected)}</span>
				<span class="bn-dim ml-2 font-mono text-[9.5px] uppercase tracking-[0.14em]">direct · read-only tools</span>
			</div>
			<div bind:this={scroller} data-part="thread" class="min-h-0 flex-1 space-y-3 overflow-y-auto overscroll-contain p-4">
				{#if loadingThread && messages.length === 0}
					<div class="space-y-3" aria-hidden="true">
						{#each [72, 46, 64, 38, 58, 50] as w, i (i)}
							<div class="bn-skeleton h-8 rounded-[10px]" style="width: {w}%; {i % 2 ? 'margin-left: auto;' : ''}"></div>
						{/each}
					</div>
				{/if}
				{#if messages.length === 0 && !sending && !loadingThread}
					<p class="bn-dim font-mono text-[10.5px]">Fresh thread with {nameOf(selected)}. Say something.</p>
				{/if}
				{#each messages.filter((m) => m.role !== 'tool') as m (m.id)}
					<div class={m.role === 'user' ? 'flex justify-end' : ''}>
						{#if m.role !== 'user'}
							<div class="bn-accent mb-0.5 font-mono text-[9.5px] uppercase tracking-wider">{nameOf(selected).toUpperCase()} · {railAgo(m.createdAt)}</div>
						{/if}
						<div class="max-w-[82%] whitespace-pre-wrap rounded-xl px-3 py-2 text-[12.5px] leading-relaxed {m.role === 'user' ? 'bn-msg-user' : 'bn-msg-agent inline-block'}">{m.content}</div>
					</div>
				{/each}
				{#if sending}
					<div class="flex justify-start">
						<span class="bn-msg-agent inline-flex items-center rounded-xl px-3 py-2"><Synthesizing since={sending} /></span>
					</div>
				{/if}
				{#if error}<p class="font-mono text-[10px]" style:color="var(--bn-err)">{error}</p>{/if}
			</div>
			<div class="shrink-0 p-3">
				<ConductorComposer onSend={send} disabled={sending !== null} placeholder={`Message ${nameOf(selected)}…`} onError={(m) => (error = m)} />
			</div>
		</div>
	{/if}
</div>

<style>
	.bn-hub {
		border-color: var(--bn-border-strong);
		background: var(--bn-surface);
		transition: grid-template-columns 200ms var(--bn-ease, ease);
	}
	.bn-railcol {
		border-color: var(--bn-border);
		background: var(--bn-bg-2);
	}
	.bn-line {
		border-color: var(--bn-border);
	}
	.bn-search {
		border-color: var(--bn-border);
		border-radius: var(--bn-r-ctl, 6px);
		background: var(--bn-bg);
	}
	.bn-search::placeholder {
		color: var(--bn-text-3);
	}
	.bn-search:focus {
		border-color: var(--bn-border-strong);
	}
	.bn-newchat {
		border-color: var(--bn-border);
		border-radius: var(--bn-r-ctl, 6px);
	}
	.bn-newchat:hover,
	.bn-hover-text:hover {
		color: var(--bn-text);
	}
	.bn-newchat:hover,
	.bn-row:hover,
	.bn-row.is-selected,
	.bn-strip:hover,
	.bn-strip.is-selected {
		background: var(--bn-surface-2);
	}
	.bn-row {
		border-radius: var(--bn-r-ctl, 6px);
	}
	.bn-row-mark {
		background: var(--bn-accent);
	}
	.bn-strip {
		color: var(--bn-text-2);
	}
	.bn-strip.is-selected {
		color: var(--bn-text);
	}
	.bn-dot-live {
		background: var(--bn-ok);
		animation: bn-hub-pulse 2s ease-in-out infinite;
	}
	.bn-dot-dim {
		background: var(--bn-text-3);
	}
	.bn-dot-ok {
		background: var(--bn-ok);
	}
	:global(.bn-accent) {
		color: var(--bn-accent);
	}
	.bn-accent {
		color: var(--bn-accent);
	}
	.bn-msg-user {
		background: var(--bn-surface-2);
		color: var(--bn-text);
	}
	.bn-msg-agent {
		background: var(--bn-surface-3);
		color: var(--bn-text-2);
	}
	@keyframes bn-hub-pulse {
		50% {
			opacity: 0.4;
		}
	}
	@media (prefers-reduced-motion: reduce) {
		.bn-hub {
			transition: none;
		}
		.bn-dot-live {
			animation: none;
		}
	}
</style>
