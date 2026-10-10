<!-- The pinned lead dossier (FounderOS v1 components/FunnelNodeCard.tsx): WHO,
     WHERE they came from, HOW to reach them, the status strip, the touch trail,
     and the last message fetched live when the card pins. -->
<script lang="ts">
	import { founderosFetch } from '$lib/founderos/api';
	import { safeHref } from '$lib/founderos/safe-href';
	import type { FunnelNode, LeadMessageBody, CommsItem } from './types';
	import { CHANNEL_GLYPHS, usd } from './viz';

	let {
		node,
		stageLabels,
		stallDays = 7,
		onclose
	}: { node: FunnelNode; stageLabels: Record<string, string>; stallDays?: number; onclose: () => void } = $props();

	type LastMsg = { kind: 'loading' } | { kind: 'unavailable' } | { kind: 'none' } | { kind: 'found'; item: CommsItem };
	let lastMsg = $state<LastMsg>({ kind: 'loading' });

	$effect(() => {
		const params = new URLSearchParams({ name: node.person ?? node.name });
		if (node.email) params.set('email', node.email);
		let alive = true;
		lastMsg = { kind: 'loading' };
		founderosFetch<LeadMessageBody>(`/pages/funnel/lead-message?${params}`)
			.then((body) => {
				if (!alive) return;
				if (!body || body.unavailable) lastMsg = { kind: 'unavailable' };
				else if (body.message) lastMsg = { kind: 'found', item: body.message };
				else lastMsg = { kind: 'none' };
			})
			.catch(() => {
				if (alive) lastMsg = { kind: 'unavailable' };
			});
		return () => {
			alive = false;
		};
	});

	const REL: Record<string, string> = { hot: 'var(--fn-hot)', warm: 'var(--fn-warm)', cold: 'var(--fn-cold)' };
	const who = $derived(node.person ?? node.name);
	const dealDiffers = $derived(node.person !== null && node.person !== node.name);
	const digits = $derived(node.phone?.replace(/[^\d]/g, '') ?? '');
	const isCall = $derived(Boolean(node.url?.includes('fathom')));
	const agoDays = (ts: string) => {
		const d = Math.max(0, Math.floor((Date.now() - Date.parse(ts)) / 86_400_000));
		return d === 0 ? 'today' : `${d}d ago`;
	};
	const quietStyle = $derived(
		node.decay > 0
			? `color: color-mix(in oklab, var(--bn-err) ${Math.round(Math.sqrt(node.decay) * 85)}%, var(--bn-text))`
			: node.state === 'stalled'
				? 'color: var(--bn-err)'
				: ''
	);
</script>

<div
	data-testid="funnel-dossier"
	class="absolute right-3 top-11 z-10 flex max-h-[min(520px,80vh)] w-[340px] flex-col overflow-y-auto rounded-[var(--bn-r-card)] border p-3"
	style="background: var(--bn-surface-2); border-color: var(--bn-border-strong)"
>
	<div class="flex items-start justify-between gap-2">
		<div class="min-w-0">
			<div class="bn-text truncate text-[14px] font-bold">{who}</div>
			{#if node.role || node.company}
				<div class="bn-muted mt-0.5 truncate text-[11px]">
					{node.role ?? ''}{node.role && node.company ? ' @ ' : ''}{node.company ?? ''}
				</div>
			{/if}
			<div class="bn-dim mt-0.5 truncate font-mono text-[9.5px] uppercase tracking-wide">
				{node.venture === 'vantage' ? 'Vantage' : 'Launchpad Cohort'} · {stageLabels[node.status] ?? node.status}{dealDiffers ? ` · deal: ${node.name}` : ''}
			</div>
		</div>
		<button onclick={onclose} class="bn-pressable bn-dim shrink-0 font-mono text-[12px] hover:text-[var(--bn-text)]" aria-label="Close">×</button>
	</div>

	<div class="mt-2.5 grid grid-cols-3 gap-2 border-t pt-2 font-mono text-[10px]" style="border-color: var(--bn-border)">
		<div><div class="bn-dim">likelihood</div><div class="bn-text text-[12px]">{node.likelihood}%</div></div>
		<div><div class="bn-dim">relationship</div><div class="text-[12px] capitalize" style="color: {REL[node.relationship]}">{node.relationship}</div></div>
		<div><div class="bn-dim">quiet for</div><div class="bn-text text-[12px]" style={quietStyle}>{node.daysSinceLastTouch}d</div></div>
	</div>
	{#if node.state === 'converted' && node.product}
		<div class="mt-2 rounded-[var(--bn-r-chip)] border px-2 py-1 font-mono text-[10px]" style="color: var(--bn-ok); border-color: color-mix(in oklab, var(--bn-ok) 35%, transparent); background: color-mix(in oklab, var(--bn-ok) 9%, transparent)">
			{node.product} · {node.amountUsd != null ? usd(node.amountUsd) : 'value unknown'}
		</div>
	{/if}
	{#if node.state !== 'converted' && (node.amountUsd ?? 0) > 0}
		<div class="bn-muted mt-2 font-mono text-[10px]">deal value · {usd(node.amountUsd ?? 0)}</div>
	{/if}
	{#if node.state === 'stalled'}
		<div class="mt-2 rounded-[var(--bn-r-chip)] border px-2 py-1 font-mono text-[10px]" style="color: var(--bn-err); border-color: color-mix(in oklab, var(--bn-err) 35%, transparent); background: color-mix(in oklab, var(--bn-err) 9%, transparent)">
			stalled — quiet past {stallDays}d before converting
		</div>
	{/if}

	<div class="mt-2.5 border-t pt-2" style="border-color: var(--bn-border)">
		<div class="bn-dim mb-1 font-mono text-[8.5px] uppercase tracking-[0.24em]">origin</div>
		<div class="flex items-baseline gap-2 text-[11px]">
			<span class="bn-text shrink-0 font-semibold">{node.origin.segment}</span>
			{#if node.origin.at}<span class="bn-dim shrink-0 font-mono text-[9.5px]">{node.origin.at}</span>{/if}
			{#if node.origin.source}<span class="bn-dim shrink-0 rounded-[var(--bn-r-chip)] border px-1 font-mono text-[8.5px] uppercase tracking-wide" style="border-color: var(--bn-border)">{node.origin.source}</span>{/if}
		</div>
		{#if node.origin.entry}<div class="bn-muted mt-0.5 truncate text-[10.5px]" title={node.origin.entry}>{node.origin.entry}</div>{/if}
	</div>

	<div class="mt-2.5 border-t pt-2" style="border-color: var(--bn-border)">
		<div class="bn-dim mb-1 font-mono text-[8.5px] uppercase tracking-[0.24em]">contact</div>
		<div class="flex flex-col gap-1">
			{#if node.email}
				<div class="flex items-baseline gap-2 font-mono text-[10.5px]"><span class="bn-dim w-10 shrink-0 uppercase tracking-wide">email</span><a class="bn-accent min-w-0 truncate hover:underline" href={`mailto:${node.email}`}>{node.email}</a></div>
			{/if}
			{#if node.phone}
				<div class="flex items-baseline gap-2 font-mono text-[10.5px]"><span class="bn-dim w-10 shrink-0 uppercase tracking-wide">phone</span><a class="bn-accent min-w-0 truncate hover:underline" href={`tel:${node.phone}`}>{node.phone}</a></div>
			{/if}
			{#if safeHref(node.linkedin)}
				<div class="flex items-baseline gap-2 font-mono text-[10.5px]"><span class="bn-dim w-10 shrink-0 uppercase tracking-wide">li</span><a class="bn-accent min-w-0 truncate hover:underline" href={safeHref(node.linkedin)} target="_blank" rel="noopener noreferrer">{node.linkedin?.replace(/^https?:\/\/(www\.)?/, '')}</a></div>
			{/if}
			{#if node.email || node.phone || node.url}
				<div class="mt-0.5 flex items-center gap-3 font-mono text-[9.5px] uppercase tracking-wide">
					{#if digits}<a class="bn-muted hover:text-[var(--bn-accent)]" href={`https://wa.me/${digits}`} target="_blank" rel="noopener noreferrer">whatsapp</a>{/if}
					{#if node.phone}<a class="bn-muted hover:text-[var(--bn-accent)]" href={`sms:${node.phone}`}>sms</a>{/if}
					{#if safeHref(node.url)}<a class="bn-muted hover:text-[var(--bn-accent)]" href={safeHref(node.url)} target="_blank" rel="noopener noreferrer">{isCall ? 'open the call ↗' : 'open ↗'}</a>{/if}
				</div>
			{/if}
			{#if !node.email && !node.phone && !node.linkedin && !node.url}
				<div class="bn-dim font-mono text-[10px]">no contact info on record</div>
			{/if}
		</div>
		<div class="mt-1.5 flex items-baseline gap-2 font-mono text-[9.5px]" data-testid="dossier-last-msg">
			<span class="bn-dim shrink-0 uppercase tracking-wide">last msg</span>
			{#if lastMsg.kind === 'loading'}<span class="bn-dim">checking comms…</span>
			{:else if lastMsg.kind === 'unavailable'}<span class="bn-dim">comms feed unavailable</span>
			{:else if lastMsg.kind === 'none'}<span class="bn-dim">no thread on record</span>
			{:else}<span class="bn-muted min-w-0 truncate" title={lastMsg.item.preview}>via {lastMsg.item.source} · {agoDays(lastMsg.item.ts)} · “{lastMsg.item.preview.slice(0, 60)}”</span>{/if}
		</div>
	</div>

	<div class="mt-2.5 border-t pt-2" style="border-color: var(--bn-border)">
		<div class="bn-dim mb-1 font-mono text-[8.5px] uppercase tracking-[0.24em]">journey</div>
		<ol class="flex flex-col gap-1">
			{#each node.touches as t (t.id)}
				<li class="bn-muted flex items-baseline gap-1.5 text-[11px]">
					<span class="bn-accent shrink-0 font-mono text-[10px]">{CHANNEL_GLYPHS[t.channel] ?? '·'}</span>
					<span class="min-w-0 flex-1 truncate" title={t.label}>{t.label}</span>
					<span class="bn-dim shrink-0 font-mono text-[9px]">{t.at.slice(5)}</span>
				</li>
			{/each}
		</ol>
	</div>
</div>
