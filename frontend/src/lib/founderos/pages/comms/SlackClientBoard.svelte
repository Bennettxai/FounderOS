<!-- Slack seen through the client roster (FounderOS v1 SlackClientBoard.tsx):
     one card per current client. Live when a real Slack thread matched;
     otherwise it says plainly that no channel is linked. Heat is a status
     color; the waiting chip says whose court the ball is in. -->
<script lang="ts">
	import { Hash } from '$lib/founderos/icons';
	import { Badge } from '$lib/founderos/kit';
	import { agoCompact } from '$lib/founderos/pages/pc/ui';
	import type { SlackCard } from './model';

	let { cards, nowMs, roster }: { cards: SlackCard[]; nowMs: number; roster: { state: string; detail?: string } } = $props();

	const HEAT: Record<string, string> = { hot: 'var(--bn-err)', warm: 'var(--bn-warn)', cold: 'var(--bn-text-3)' };
	const WAIT: Record<SlackCard['waiting'], string> = { you: 'waiting on you', them: 'waiting on them', none: 'all clear' };
</script>

{#if cards.length === 0}
	<p data-part="slack-empty" class="bn-dim rounded-xl border border-dashed border-[color:var(--bn-border-strong)] px-3 py-3 text-center font-mono text-[10.5px]">
		{#if roster.state === 'connected'}
			No current clients on the roster. Anyone who paid through Stripe (and has a live Slack thread) lands here.
		{:else}
			Client roster unavailable ({roster.state}{roster.detail ? `: ${roster.detail}` : ''}), so current clients are unknown.
		{/if}
	</p>
{:else}
	<div class="grid gap-3 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
		{#each cards as card (card.id)}
			<div data-part="slack-card" data-lens="r" class="pc-surface bn-pressable is-row flex flex-col rounded-2xl px-3.5 py-3">
				<div class="flex items-center gap-2">
					<span
						class="h-2 w-2 shrink-0 rounded-full"
						style="background: {card.heat ? HEAT[card.heat] : 'var(--bn-border-strong)'}"
						title={card.heat ? `${card.heat} activity` : 'no Slack thread'}
					></span>
					<span class="truncate text-[12.5px] font-semibold">{card.name}</span>
					{#if card.unread > 0}<span class="ml-auto shrink-0"><Badge tone="accent">{card.unread}</Badge></span>{/if}
				</div>
				<div class="bn-dim mt-1 flex items-center gap-1.5 font-mono text-[9.5px]">
					<Hash size={12} class="shrink-0" />
					<span class="truncate">{card.channel ? card.channel.replace(/^#/, '') : 'no channel linked'}</span>
					{#if card.lastTs}<span class="ml-auto shrink-0">{agoCompact(card.lastTs, nowMs)}</span>{/if}
				</div>
				<p class="bn-muted mt-2 line-clamp-2 text-[11px] leading-snug">
					{#if card.lastText}{card.lastText}{:else}<span class="bn-dim">No Slack thread yet. Invite the bot to their channel.</span>{/if}
				</p>
				<div class="pc-hair mt-2.5 flex items-center gap-2 border-t pt-2">
					<span class="font-mono text-[9px] uppercase tracking-[0.14em]" style="color: {card.waiting === 'you' ? 'var(--bn-warn)' : 'var(--bn-text-3)'}">
						{WAIT[card.waiting]}
					</span>
					<span class="ml-auto shrink-0">
						{#if card.live}<Badge tone="ok">live</Badge>{:else}<Badge ghost>quiet</Badge>{/if}
					</span>
				</div>
			</div>
		{/each}
	</div>
{/if}
