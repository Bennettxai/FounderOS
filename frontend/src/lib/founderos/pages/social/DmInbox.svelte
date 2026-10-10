<!-- The Instagram DM inbox (components/InstagramDmInbox.tsx): thread list +
     conversation + a reply box that sends through ManyChat via the guarded
     /pages/social/dm/reply. A refused or failed send shows inline and appends
     nothing. -->
<script lang="ts">
	import { Instagram, Send } from '$lib/founderos/icons';
	import { founderosFetch, FounderosApiError } from '$lib/founderos/api';
	import { initials, relativeTime } from './lib';
	import type { DmMessage, DmThread } from './types';

	let { threads: initial, nowMs }: { threads: DmThread[]; nowMs: number } = $props();

	// svelte-ignore state_referenced_locally
	let messages = $state<DmMessage[]>(initial.flatMap((t) => t.messages));
	// svelte-ignore state_referenced_locally
	let selected = $state<string | null>(initial[0]?.subscriberId ?? null);
	let draft = $state('');
	let sending = $state(false);
	let error = $state<string | null>(null);

	const threads = $derived.by(() => {
		const groups = new Map<string, DmMessage[]>();
		for (const m of messages) groups.set(m.subscriberId, [...(groups.get(m.subscriberId) ?? []), m]);
		const out: DmThread[] = [];
		for (const [subscriberId, list] of groups) {
			const chrono = [...list].sort((a, b) => (a.ts < b.ts ? -1 : a.ts > b.ts ? 1 : 0));
			const last = chrono[chrono.length - 1];
			out.push({ subscriberId, name: last.name, handle: last.handle, messages: chrono, last, unreplied: last.direction === 'in' });
		}
		return out.sort((a, b) => (a.last.ts < b.last.ts ? 1 : a.last.ts > b.last.ts ? -1 : 0));
	});
	const active = $derived(threads.find((t) => t.subscriberId === selected) ?? threads[0] ?? null);
	const isLive = $derived(messages.some((m) => m.source === 'manychat'));

	async function send() {
		if (!active || !draft.trim() || sending) return;
		sending = true;
		error = null;
		try {
			const body = await founderosFetch<{ ok: boolean; message: DmMessage; error?: string }>('/pages/social/dm/reply', {
				method: 'POST',
				json: { subscriberId: active.subscriberId, text: draft.trim() }
			});
			messages = [body.message, ...messages];
			draft = '';
		} catch (err) {
			const b = err instanceof FounderosApiError ? (err.body as { error?: string } | undefined) : undefined;
			// The server's reason when it gave one; otherwise what actually failed
			// (a network drop is not a missing key).
			error = typeof b?.error === 'string' ? b.error : `Send failed · ${err instanceof Error ? err.message : String(err)}`;
		} finally {
			sending = false;
		}
	}
</script>

<div data-part="dm-inbox">
	<div class="mb-4 flex items-center gap-2 border-b pb-3" style="border-color: var(--bn-border)">
		<span class="soc-chip flex items-center gap-2 px-3.5 py-1.5 font-mono text-[12px] font-semibold" data-on="true">
			<Instagram size={14} /> Instagram
			<span class="soc-bubble-out px-1.5 text-[10px]">{threads.length}</span>
		</span>
		<span class="bn-dim ml-auto flex items-center gap-2 font-mono text-[10px] uppercase tracking-[0.15em]">
			<span class="h-1.5 w-1.5" style="background: {isLive ? 'var(--bn-ok)' : 'var(--bn-text-3)'}"></span>
			{isLive ? 'live · manychat' : 'seeded · live via manychat webhook'}
		</span>
	</div>

	{#if threads.length === 0}
		<div class="bn-dim bn-card px-4 py-8 text-center font-mono text-[12px]">No DMs yet. They appear here as ManyChat posts them to the webhook.</div>
	{:else}
		<div class="grid grid-cols-1 gap-px md:grid-cols-[minmax(0,320px)_minmax(0,1fr)]" style="background: var(--bn-border); border: 1px solid var(--bn-border)">
			<div class="flex max-h-[520px] flex-col overflow-y-auto" style="background: var(--bn-surface)">
				{#each threads as t (t.subscriberId)}
					<button
						type="button"
						class="pc-row flex items-center gap-3 border-b px-3.5 py-3 text-left {t.subscriberId === active?.subscriberId ? 'soc-sel' : ''}"
						style="border-color: var(--bn-border)"
						onclick={() => (selected = t.subscriberId)}
					>
						<span class="bn-muted grid h-9 w-9 shrink-0 place-items-center border font-mono text-[11px] font-bold" style="border-color: var(--bn-border-strong); background: var(--bn-bg)">{initials(t.name)}</span>
						<div class="min-w-0 flex-1">
							<div class="flex items-baseline gap-2">
								<span class="bn-text truncate text-[12.5px] font-semibold">{t.name}</span>
								<span class="bn-dim ml-auto shrink-0 font-mono text-[10px]">{relativeTime(t.last.ts, nowMs)}</span>
							</div>
							<div class="flex items-center gap-1.5">
								<span class="bn-dim truncate font-mono text-[11px]">{t.last.direction === 'out' ? 'You: ' : ''}{t.last.text}</span>
								{#if t.unreplied}<span class="ml-auto h-1.5 w-1.5 shrink-0" style="background: var(--bn-warn)" title="needs reply"></span>{/if}
							</div>
						</div>
					</button>
				{/each}
			</div>
			<div class="flex max-h-[520px] flex-col" style="background: var(--bn-surface)">
				{#if active}
					<div class="flex items-center gap-2 border-b px-4 py-3" style="border-color: var(--bn-border)">
						<Instagram size={14} class="bn-dim" />
						<span class="bn-text text-[12.5px] font-semibold">{active.name}</span>
						{#if active.handle}<span class="bn-dim font-mono text-[11px]">@{active.handle}</span>{/if}
						{#if active.unreplied}
							<span class="soc-warn ml-auto border px-1.5 font-mono text-[9.5px] uppercase tracking-[0.12em]" style="border-color: color-mix(in oklab, var(--bn-warn) 50%, transparent)">needs reply</span>
						{/if}
					</div>
					<div class="flex flex-1 flex-col gap-2.5 overflow-y-auto px-4 py-4">
						{#each active.messages as m (m.id)}
							<div class="flex flex-col {m.direction === 'out' ? 'items-end' : 'items-start'}">
								<div class="max-w-[78%] px-3 py-2 text-[12.5px] leading-relaxed {m.direction === 'out' ? 'soc-bubble-out' : 'soc-bubble-in'}">{m.text}</div>
								<div class="bn-dim mt-1 flex items-center gap-1.5 px-1 font-mono text-[9.5px]">
									{#if m.tag}<span class="bn-accent uppercase tracking-[0.1em]">{m.tag}</span>{/if}
									<span>{relativeTime(m.ts, nowMs)}</span>
								</div>
							</div>
						{/each}
					</div>
					<div class="border-t px-3 py-3" style="border-color: var(--bn-border)">
						{#if error}<p class="soc-err mb-2 font-mono text-[10.5px]" data-part="dm-error">{error}</p>{/if}
						<div class="flex items-end gap-2">
							<textarea
								bind:value={draft}
								rows="1"
								placeholder={`Reply to ${active.name}…  (⌘↵ to send)`}
								class="pc-input max-h-28 min-h-[38px] flex-1 resize-none px-3 py-2 font-mono text-[12px]"
								onkeydown={(e) => {
									if (e.key === 'Enter' && (e.metaKey || e.ctrlKey)) {
										e.preventDefault();
										void send();
									}
								}}
							></textarea>
							<button type="button" class="soc-primary flex h-[38px] items-center gap-1.5 px-3.5 font-mono text-[12px] font-semibold" disabled={sending || !draft.trim()} onclick={() => void send()}>
								<Send size={14} />{sending ? 'Sending' : 'Send'}
							</button>
						</div>
					</div>
				{/if}
			</div>
		</div>
	{/if}
</div>
