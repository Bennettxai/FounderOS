<!-- The three-pane messaging view (FounderOS v1 CommsThreePane.tsx): sources
     rail / message list / reader. Slack clients and channels are sources on
     the rail. Replies go out through POST /pages/comms/reply, which is a
     guarded write (502 with the refusal while FOUNDEROS_WRITES=0); archive and
     snooze are optimistic local removals with an undo (the previous lanes
     reference is the undo state). Keys: j/k move, e archive, s snooze,
     d delegate, r reply, z undo, 1-6 source, esc clear. -->
<script lang="ts">
	import './comms.css';
	import { Archive, Clock, CornerUpLeft, Hash, Inbox, Lock, Mail, MessageSquare, Send, UserPlus, Users } from '$lib/founderos/icons';
	import { untrack } from 'svelte';
	import { founderosFetch } from '$lib/founderos/api';
	import { toast } from '$lib/founderos/chrome/toast';
	import { Badge, Dot, Kbd } from '$lib/founderos/kit';
	import { agoCompact } from '$lib/founderos/pages/pc/ui';
	import SlackClientBoard from './SlackClientBoard.svelte';
	import { buildCommsSources, itemsForSource, removeItem, type Lane, type PaneRow, type SlackCard, type SlackChannel, type SourceKind } from './model';

	let {
		lanes,
		slackCards,
		slackRoster,
		channels,
		channelsError,
		nowMs
	}: {
		lanes: Lane[];
		slackCards: SlackCard[];
		slackRoster: { state: string; detail?: string };
		channels: SlackChannel[];
		channelsError?: string;
		nowMs: number;
	} = $props();

	const PRIORITY: Record<number, string> = { 1: 'var(--bn-err)', 2: 'var(--bn-warn)', 3: 'var(--bn-ok)' };
	const ICON: Record<SourceKind, typeof Mail> = { all: Inbox, email: Mail, whatsapp: MessageSquare, 'slack-clients': Users, 'slack-channels': Hash };

	let laneState = $state<Lane[]>(untrack(() => lanes));
	let sourceId = $state('all');
	let selId = $state<string | null>(null);
	let lastUndo = $state<(() => void) | null>(null);

	// composer
	let composing = $state(false);
	let draft = $state('');
	let sendState = $state<{ phase: 'sending' | 'sent' | 'error'; detail?: string } | null>(null);

	const sources = $derived(buildCommsSources(laneState, slackCards, channels));
	const source = $derived(sources.find((s) => s.id === sourceId) ?? sources[0]);
	const rows = $derived(itemsForSource(laneState, sourceId));
	const selected = $derived(rows.find((r) => r.id === selId) ?? null);
	const isBoard = $derived(source?.kind === 'slack-clients' || source?.kind === 'slack-channels');
	const lane = $derived(selected ? (laneState.find((l) => l.id === selected.laneId) ?? null) : null);
	const canReply = $derived(Boolean(selected?.replyTo && lane?.source === 'email'));

	function say(text: string, undo?: () => void) {
		toast.ok(text, undo);
	}

	function pick(id: string | null) {
		selId = id;
		composing = false;
		draft = '';
		sendState = null;
	}

	function setSource(id: string) {
		sourceId = id;
		pick(null);
	}

	function dismiss(row: PaneRow, verb: 'Archived' | 'Snoozed') {
		const prev = laneState;
		const idx = rows.findIndex((r) => r.id === row.id);
		const next = rows[idx + 1] ?? rows[idx - 1] ?? null;
		laneState = removeItem(prev, row.id);
		pick(next ? next.id : null);
		const undo = () => {
			laneState = prev;
			lastUndo = null;
		};
		lastUndo = undo;
		say(`${verb} — ${row.sender}`, undo);
	}

	function delegate(row: PaneRow) {
		say(`Delegated to Comms lead — ${row.sender}`);
	}

	function move(delta: 1 | -1) {
		if (rows.length === 0) return;
		const idx = selected ? rows.findIndex((r) => r.id === selected.id) : -1;
		const next = rows[Math.min(rows.length - 1, Math.max(0, idx + delta))] ?? rows[0];
		pick(next.id);
	}

	function onListKey(e: KeyboardEvent) {
		if (e.key === 'ArrowDown' || e.key === 'j') {
			e.preventDefault();
			move(1);
		} else if (e.key === 'ArrowUp' || e.key === 'k') {
			e.preventDefault();
			move(-1);
		} else if (e.key === 'e' && selected) {
			e.preventDefault();
			dismiss(selected, 'Archived');
		} else if (e.key === 's' && selected) {
			e.preventDefault();
			dismiss(selected, 'Snoozed');
		} else if (e.key === 'd' && selected) {
			e.preventDefault();
			delegate(selected);
		} else if (e.key === 'r' && selected) {
			e.preventDefault();
			if (canReply) composing = true;
		} else if (e.key === 'z') {
			e.preventDefault();
			lastUndo?.();
		} else if (/^[1-6]$/.test(e.key)) {
			// stopPropagation keeps the palette's global digit view-jumps out of it
			const s = sources[Number(e.key) - 1];
			if (s) {
				e.preventDefault();
				e.stopPropagation();
				setSource(s.id);
			}
		} else if (e.key === 'Escape') {
			e.preventDefault();
			e.stopPropagation();
			pick(null);
		}
	}

	async function send() {
		const row = selected;
		if (!row?.replyTo || !draft.trim() || sendState?.phase === 'sending') return;
		sendState = { phase: 'sending' };
		try {
			await founderosFetch('/pages/comms/reply', {
				method: 'POST',
				json: { source: 'email', account: row.laneId, to: row.replyTo, subject: `Re: ${row.preview}`, text: draft }
			});
			// Clear the draft so a second click cannot send the same email again.
			draft = '';
			sendState = { phase: 'sent' };
		} catch (e) {
			sendState = { phase: 'error', detail: e instanceof Error ? e.message : 'network error' };
		}
	}
</script>

<div class="relative">
	<div class="flex flex-col gap-3 lg:grid lg:grid-cols-[200px_minmax(0,1fr)_380px]">
		<!-- Sources rail -->
		<div data-part="rail" class="flex flex-row gap-1 overflow-x-auto lg:flex-col lg:overflow-visible">
			{#each sources as s (s.id)}
				{@const Icon = ICON[s.kind]}
				{@const active = s.id === sourceId}
				<button
					type="button"
					data-source={s.id}
					aria-pressed={active}
					onclick={() => setSource(s.id)}
					data-lens="r"
					class="bn-pressable is-row flex shrink-0 items-center gap-2 rounded-[6px] px-2 py-1.5 text-left {active ? 'bn-text' : 'bn-muted hover:text-[color:var(--bn-text)]'}"
					style={active ? 'background: var(--bn-surface-2)' : ''}
				>
					<Icon size={12} class="shrink-0 {active ? 'bn-accent' : 'bn-dim'}" />
					<span class="truncate text-[11px] font-semibold">{s.name}</span>
					<span class="ml-auto flex shrink-0 items-center gap-1.5">
						{#if s.unread > 0}<Badge tone="accent">{s.unread}</Badge>{:else}<span class="bn-dim font-mono text-[9px]">{s.count}</span>{/if}
						{#if s.state && s.state !== 'connected'}<Dot state={s.state} />{/if}
					</span>
				</button>
			{/each}
			<div class="pc-hair mt-2 hidden border-t pt-2.5 lg:block">
				<p class="bn-dim font-mono text-[9px]"><Kbd size="sm">1</Kbd>-<Kbd size="sm">6</Kbd> switch source</p>
				<p class="bn-dim mt-2 font-mono text-[9px] leading-relaxed">Slack · channels moved under the Slack source. Recordings has its own tab.</p>
			</div>
		</div>

		{#if isBoard}
			<div class="min-w-0 lg:col-span-2">
				{#if source.kind === 'slack-clients'}
					<SlackClientBoard cards={slackCards} {nowMs} roster={slackRoster} />
				{:else if channels.length === 0}
					<p data-part="channels-empty" class="pc-hair bn-dim rounded-[12px] border border-dashed font-mono text-[10.5px] cm-tile">
						{#if channelsError}
							Slack channels unavailable: {channelsError}
						{:else}
							No channels imported — connect the Slack bot token to pull every current channel.
						{/if}
					</p>
				{:else}
					<div class="grid grid-cols-2 gap-2.5 sm:grid-cols-3 xl:grid-cols-4">
						{#each channels as c (c.id)}
							<div data-part="channel" data-lens="r" class="pc-surface bn-pressable is-row rounded-[12px] cm-tile">
								<div class="flex items-center gap-1.5">
									{#if c.isPrivate}<Lock size={13} class="bn-dim shrink-0" />{:else}<Hash size={13} class="bn-accent shrink-0" />{/if}
									<span class="min-w-0 flex-1 truncate font-mono text-[11.5px] font-semibold">{c.name}</span>
									<span class="bn-dim flex shrink-0 items-center gap-1 font-mono text-[9.5px]"><Users size={12} />{c.members}</span>
								</div>
								<p class="bn-dim mt-1 truncate text-[10px] leading-snug">{c.topic || (c.isMember ? 'member' : 'not a member')}</p>
							</div>
						{/each}
					</div>
				{/if}
			</div>
		{:else}
			<!-- Message list. Prod's list, reader and channel cards carry rounded-tile,
			     which its .os-slab rule pads to 22px all round; the port says so directly.
			     Rows are not data-part="row": the kit's slab rule would re-pad them. -->
			<div
				data-part="list"
				role="listbox"
				aria-label={source?.name ?? 'All sources'}
				tabindex="0"
				onkeydown={onListKey}
				class="pc-surface max-h-[calc(100dvh-17rem)] min-h-[320px] min-w-0 overflow-y-auto rounded-[12px] outline-none cm-tile focus-visible:[box-shadow:inset_0_0_0_1px_rgba(242,242,242,.18)]"
			>
				<div class="pc-hair sticky top-0 z-[1] flex items-baseline justify-between border-b px-3 py-1.5" style="background: var(--bn-surface)">
					<span class="bn-muted font-mono text-[9.5px] font-bold uppercase tracking-[0.18em]">{source?.name ?? 'All sources'}</span>
					<span class="bn-dim font-mono text-[9px]">newest first</span>
				</div>
				{#if rows.length === 0}
					<div data-part="list-empty" class="bn-dim px-4 py-5 font-mono text-[10.5px] leading-relaxed">
						{#if source?.state && source.state !== 'connected'}
							<p>{source.name} is not connected. Check /integrations.</p>
						{:else}
							<p class="bn-muted">{source?.name ?? 'This source'} is clear.</p>
							<p class="mt-1">Everything here was archived, snoozed or delegated away.</p>
							<div class="mt-3 flex items-center gap-2">
								{#if lastUndo}
									<button type="button" onclick={() => lastUndo?.()} class="bn-pressable pc-pill rounded-[6px] border px-2 py-1 text-[10px]">undo last</button>
								{/if}
								{#if sourceId !== 'all'}
									<button type="button" onclick={() => setSource('all')} class="bn-pressable pc-pill rounded-[6px] border px-2 py-1 text-[10px]">all sources →</button>
								{/if}
							</div>
						{/if}
					</div>
				{:else}
					<div class="flex flex-col">
						{#each rows as row (row.id)}
							{@const active = selected?.id === row.id}
							<div
								data-part="msg-row"
								role="option"
								aria-selected={active}
								tabindex="-1"
								onclick={() => pick(row.id)}
								onkeydown={(e) => e.key === 'Enter' && pick(row.id)}
								data-lens="r"
								class="bn-pressable is-row pc-hair relative cursor-pointer border-b px-3 py-1.5 last:border-b-0"
								style={active ? 'background: var(--bn-surface-2)' : ''}
							>
								{#if active}<span class="absolute inset-y-0 left-0 w-[2px]" style="background: var(--bn-accent)"></span>{/if}
								<div class="flex items-center gap-2">
									{#if row.priority}<span data-part="priority-dot" class="h-1.5 w-1.5 shrink-0 rounded-full" style="background: {PRIORITY[row.priority]}"></span>{/if}
									<span class="truncate text-[11.5px] font-semibold">{row.sender}</span>
									{#if sourceId === 'all'}<span class="bn-dim shrink-0 font-mono text-[8.5px] uppercase tracking-[0.12em]">{row.laneName}</span>{/if}
									<span class="bn-dim ml-auto shrink-0 font-mono text-[9px]">{agoCompact(row.ts, nowMs)}</span>
									{#if row.unread}<span data-part="unread-dot" class="bn-pop h-1.5 w-1.5 shrink-0 rounded-full" style="background: var(--bn-accent)" title="unread"></span>{/if}
								</div>
								<p class="bn-muted mt-0.5 line-clamp-1 text-[11px] leading-snug">{row.preview}</p>
							</div>
						{/each}
					</div>
				{/if}
			</div>

			<!-- Reader -->
			{#if !selected}
				<div data-part="reader-empty" class="pc-surface flex min-h-[320px] flex-col items-center justify-center rounded-[12px] text-center cm-tile">
					<Inbox size={20} class="bn-dim mb-2" />
					<p class="bn-dim font-mono text-[10.5px] leading-relaxed">Select a message to read it here.</p>
					<div class="bn-dim mt-3 flex max-w-[280px] flex-wrap items-center justify-center gap-x-3 gap-y-1.5 font-mono text-[9.5px]">
						<span><Kbd size="sm">j</Kbd> <Kbd size="sm">k</Kbd> move</span>
						<span><Kbd size="sm">e</Kbd> archive</span>
						<span><Kbd size="sm">s</Kbd> snooze</span>
						<span><Kbd size="sm">d</Kbd> delegate</span>
						<span><Kbd size="sm">r</Kbd> reply</span>
						<span><Kbd size="sm">z</Kbd> undo</span>
						<span><Kbd size="sm">1</Kbd>-<Kbd size="sm">6</Kbd> source</span>
						<span><Kbd size="sm">esc</Kbd> clear</span>
					</div>
				</div>
			{:else}
				<div data-part="reader" class="pc-surface bn-enter flex max-h-[calc(100dvh-17rem)] min-h-[320px] flex-col rounded-[12px] cm-tile">
					<div class="pc-hair shrink-0 border-b px-4 py-3">
						<div class="bn-dim flex items-center gap-2 font-mono text-[9px] uppercase tracking-[0.18em]">
							{#if selected.priority}<span class="h-1.5 w-1.5 shrink-0 rounded-full" style="background: {PRIORITY[selected.priority]}"></span>{/if}
							<span class="truncate">{selected.laneName} · {agoCompact(selected.ts, nowMs)}</span>
						</div>
						<p class="mt-1 truncate text-[13px] font-semibold">{selected.sender}</p>
						<p class="bn-dim mt-0.5 truncate font-mono text-[9.5px]">from {selected.laneName}{selected.replyTo ? ` · ${selected.replyTo}` : ''}</p>
					</div>
					<div class="min-h-0 flex-1 overflow-y-auto px-4 py-3">
						<p class="bn-muted text-[12px] leading-relaxed">{selected.preview}</p>
						{#if composing}
							<div data-part="composer" class="pc-panel mt-3 rounded-[12px] p-2.5">
								<p class="bn-dim mb-1.5 font-mono text-[9.5px] uppercase tracking-[0.15em]">to {selected.replyTo} · re: {selected.preview.slice(0, 60)}</p>
								<textarea bind:value={draft} rows="5" placeholder="Write the reply…" aria-label="Reply" class="pc-input w-full resize-y rounded-[6px] px-2.5 py-2 text-[12px] leading-relaxed"></textarea>
								<div class="mt-2 flex flex-wrap items-center gap-2">
									<button
										type="button"
										onclick={send}
										disabled={!draft.trim() || sendState?.phase === 'sending'}
										class="bn-pressable pc-pill bn-accent flex items-center gap-1.5 rounded-[6px] border px-3 py-1.5 font-mono text-[10.5px] font-semibold uppercase tracking-widest disabled:opacity-40"
									>
										<Send size={12} class="h-3 w-3" />
										{sendState?.phase === 'sending' ? 'sending…' : 'send'}
									</button>
									<button type="button" onclick={() => (composing = false)} class="bn-pressable bn-dim font-mono hover:text-[color:var(--bn-text)] text-[10.5px] uppercase tracking-widest">cancel</button>
									{#if sendState?.phase === 'sent'}<Badge tone="ok">sent</Badge>{/if}
									{#if sendState?.phase === 'error'}<Badge tone="err">{sendState.detail ?? 'failed'}</Badge>{/if}
								</div>
								{#if sendState?.phase === 'error' && selected.replyTo}
									<a class="bn-dim mt-2 inline-block font-mono text-[10px] underline" href={`mailto:${selected.replyTo}?subject=${encodeURIComponent(`Re: ${selected.preview}`)}&body=${encodeURIComponent(draft)}`}>
										open as a mailto: draft instead
									</a>
								{/if}
							</div>
						{/if}
					</div>
					<div class="pc-hair flex shrink-0 items-center gap-2 border-t px-3 py-2">
						<button
							type="button"
							onclick={() => (composing = true)}
							disabled={!canReply}
							title={canReply ? `Reply to ${selected.replyTo}` : 'No reply address on this message'}
							class="bn-pressable flex items-center gap-1.5 rounded-[6px] px-2.5 py-1 font-mono text-[9.5px] font-semibold uppercase tracking-widest transition-opacity hover:opacity-90 disabled:cursor-not-allowed disabled:opacity-40"
							style="background: var(--bn-text); color: var(--bn-bg)"
						>
							<CornerUpLeft size={12} class="h-3 w-3" /> Reply
						</button>
						<button type="button" onclick={() => dismiss(selected, 'Archived')} class="bn-pressable bn-dim flex items-center gap-1.5 rounded-[6px] px-2 py-1 font-mono text-[9.5px] uppercase tracking-widest hover:text-[color:var(--bn-text)]"><Archive size={12} class="h-3 w-3" /> archive</button>
						<button type="button" onclick={() => delegate(selected)} class="bn-pressable bn-dim flex items-center gap-1.5 rounded-[6px] px-2 py-1 font-mono text-[9.5px] uppercase tracking-widest hover:text-[color:var(--bn-text)]"><UserPlus size={12} class="h-3 w-3" /> delegate</button>
						<button type="button" onclick={() => dismiss(selected, 'Snoozed')} class="bn-pressable bn-dim flex items-center gap-1.5 rounded-[6px] px-2 py-1 font-mono text-[9.5px] uppercase tracking-widest hover:text-[color:var(--bn-text)]"><Clock size={12} class="h-3 w-3" /> snooze</button>
						{#if selected.unread}<Badge tone="accent">unread</Badge>{/if}
					</div>
				</div>
			{/if}
		{/if}
	</div>

</div>
