<!-- Past newsletters (components/NewsletterList.tsx): one row per send with
     its open rate; click a row to unfold the full send analytics. -->
<script lang="ts">
	import { ChevronDown, ChevronRight, ExternalLink } from '$lib/founderos/icons';
	import type { Newsletter } from './types';

	let { newsletters }: { newsletters: Newsletter[] } = $props();
	let openId = $state<string | null>(null);

	const fmt = (n: number) => n.toLocaleString('en-US');
	const pct = (n: number) => `${n.toFixed(n < 10 ? 2 : 1)}%`;
	const dateLabel = (iso: string) => new Date(iso).toLocaleDateString('en-US', { month: 'short', day: 'numeric', year: 'numeric' });
</script>

{#snippet metric(label: string, value: string, sub?: string, tone?: 'ok' | 'warn')}
	<div class="rounded-[8px] border px-3 py-2.5" style="border-color: var(--bn-border); background: var(--bn-surface)">
		<div class="bn-dim font-mono text-[9px] uppercase tracking-[0.16em]">{label}</div>
		<div class="mt-1 font-mono text-[17px] font-semibold leading-none tracking-[-0.02em] {tone === 'ok' ? 'soc-ok' : tone === 'warn' ? 'soc-warn' : 'bn-text'}">{value}</div>
		{#if sub}<div class="bn-dim mt-1 font-mono text-[9.5px]">{sub}</div>{/if}
	</div>
{/snippet}

{#if newsletters.length === 0}
	<p class="bn-dim py-6 text-center text-[12.5px]">No newsletters yet.</p>
{:else}
	<div class="flex flex-col gap-2.5" data-part="newsletters">
		{#each newsletters as n (n.id)}
			{@const expanded = openId === n.id}
			<div class="pc-panel overflow-hidden rounded-[10px]">
				<button type="button" aria-expanded={expanded} class="bn-pressable pc-row flex w-full items-center gap-4 px-5 py-4 text-left" onclick={() => (openId = expanded ? null : n.id)}>
					<span class="bn-dim shrink-0">{#if expanded}<ChevronDown size={16} />{:else}<ChevronRight size={16} />{/if}</span>
					<span class="min-w-0 flex-1">
						<span class="bn-text block truncate text-[13.5px] font-medium">{n.title}</span>
						<span class="bn-dim mt-0.5 block font-mono text-[11px]">{dateLabel(n.publishedAt)} · {fmt(n.recipients)} sent</span>
					</span>
					<span class="bn-dim hidden shrink-0 font-mono text-[11px] tabular-nums sm:inline">{pct(n.clickRate)} click</span>
					<span class="shrink-0 rounded-full px-2.5 py-0.5 font-mono text-[11px] tabular-nums tracking-[0.04em]" style="background: var(--bn-accent-soft); color: var(--bn-accent)" title="open rate">{pct(n.openRate)} open</span>
				</button>
				{#if expanded}
					<div class="border-t px-5 pb-5 pt-4" style="border-color: var(--bn-border)" data-part="analytics">
						<div class="grid grid-cols-2 gap-2 sm:grid-cols-3 xl:grid-cols-4">
							{@render metric('Recipients', fmt(n.recipients))}
							{@render metric('Delivered', fmt(n.delivered), `${pct(n.deliveryRate)} delivery`, 'ok')}
							{@render metric('Open rate', pct(n.openRate), `${fmt(n.opens)} opens`, 'ok')}
							{@render metric('Click rate', pct(n.clickRate), `${fmt(n.clicks)} clicks`)}
							{@render metric('Unsubscribes', fmt(n.unsubscribes), pct(n.unsubscribeRate), n.unsubscribeRate > 1 ? 'warn' : undefined)}
							{@render metric('Spam reports', fmt(n.spamReports), undefined, n.spamReports > 0 ? 'warn' : undefined)}
							{@render metric('Web views', fmt(n.webViews))}
							{@render metric('Delivered %', pct(n.deliveryRate))}
						</div>
						{#if n.webUrl}
							<a href={n.webUrl} target="_blank" rel="noreferrer" data-lens="c" class="bn-pressable is-dark soc-view mt-4 inline-flex items-center gap-1.5 rounded-full px-3.5 py-1.5 text-[12.5px]">Read the issue <ExternalLink size={12} /></a>
						{/if}
					</div>
				{/if}
			</div>
		{/each}
	</div>
{/if}
