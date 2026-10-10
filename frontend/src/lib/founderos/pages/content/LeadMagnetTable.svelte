<!-- Lead magnets as a Notion-style database (FounderOS v1 LeadMagnets +
     LeadMagnetRowActions): property columns, the real link on every row,
     copy-link, and row controls (status → PATCH, confirm-delete → DELETE). -->
<script lang="ts">
	import { ArrowUpRight, CalendarCheck, Mail, Minus, Trash2 } from '$lib/founderos/icons';
	import { founderosFetch } from '$lib/founderos/api';
	import CopyLink from './CopyLink.svelte';
	import { LEAD_MAGNET_STATUSES, dateLabel, host, statusPillStyle, type LeadMagnet } from './types';

	let {
		rows,
		showCopy = false,
		manage = false,
		onChanged
	}: { rows: LeadMagnet[]; showCopy?: boolean; manage?: boolean; onChanged?: () => void } = $props();

	const CAPTURES = {
		email: { Icon: Mail, label: 'Email' },
		booking: { Icon: CalendarCheck, label: 'Booking' },
		none: { Icon: Minus, label: 'None' }
	} as const;

	const headers = $derived(['Name', 'Status', 'Captures', 'Leads to', 'Source', 'Live', ...(showCopy ? ['Link'] : []), ...(manage ? ['Manage'] : [])]);

	let busy = $state<string | null>(null);
	let confirming = $state<string | null>(null);
	let failure = $state<{ id: string; msg: string } | null>(null);

	async function call(id: string, init: { method: string; json?: unknown }) {
		busy = id;
		failure = null;
		try {
			await founderosFetch(`/pages/lead-magnets/${encodeURIComponent(id)}`, init);
			onChanged?.();
		} catch (e) {
			failure = { id, msg: e instanceof Error ? e.message : String(e) };
		} finally {
			if (busy === id) busy = null;
			if (confirming === id) confirming = null;
		}
	}
</script>

{#if rows.length === 0}
	<p class="pc-hair bn-dim border-t px-6 py-8 text-center text-[12.5px]">No lead magnets yet. Every landing page we ship lands here.</p>
{:else}
	<div class="pc-hair overflow-x-auto border-t">
		<table class="w-full min-w-[760px] border-collapse">
			<thead>
				<tr class="pc-hair border-b">
					{#each headers as h (h)}
						<th class="bn-dim px-4 py-3 text-left font-mono text-[10px] uppercase tracking-[0.18em] first:pl-6 last:pr-6">{h}</th>
					{/each}
				</tr>
			</thead>
			<tbody>
				{#each rows as m (m.id)}
					{@const c = CAPTURES[m.captures] ?? CAPTURES.none}
					<tr class="pc-row pc-hair group border-b last:border-b-0 [&>td:last-child]:pr-6">
						<td class="py-4 pl-6 pr-4 align-top">
							<a href={m.url} target="_blank" rel="noreferrer" class="bn-text lm-name flex items-center gap-1.5 text-[13.5px] font-medium">
								{m.name}
								<ArrowUpRight size={12} class="shrink-0 opacity-0 transition-opacity group-hover:opacity-100" />
							</a>
							<div class="bn-muted mt-0.5 max-w-[380px] text-[11.5px] leading-snug">{m.offer}</div>
							<div class="bn-dim mt-1 font-mono text-[10px]">{host(m.url)}</div>
						</td>
						<td class="whitespace-nowrap px-4 py-4 align-top">
							<span class="rounded-full px-2.5 py-0.5 font-mono text-[10.5px] uppercase tracking-[0.12em]" style={statusPillStyle(m.status)}>{m.status}</span>
						</td>
						<td class="whitespace-nowrap px-4 py-4 align-top">
							<span class="pc-hair bn-muted inline-flex items-center gap-1.5 rounded-full border px-2.5 py-0.5 font-mono text-[10.5px]">
								<c.Icon size={12} class="shrink-0" />
								{c.label}
							</span>
						</td>
						<td class="bn-muted px-4 py-4 align-top text-[11.5px] leading-snug">{m.destination}</td>
						<td class="bn-muted px-4 py-4 align-top text-[11.5px] leading-snug">{m.source}</td>
						<td class="bn-dim whitespace-nowrap px-4 py-4 align-top font-mono text-[10.5px]">{dateLabel(m.launchedAt)}</td>
						{#if showCopy}
							<td class="whitespace-nowrap px-4 py-4 align-top"><CopyLink url={m.url} /></td>
						{/if}
						{#if manage}
							<td class="whitespace-nowrap px-4 py-4 pr-6 align-top">
								<span class="flex items-center gap-1.5">
									<select
										aria-label="Status"
										disabled={busy === m.id}
										value={m.status}
										onchange={(e) => call(m.id, { method: 'PATCH', json: { status: (e.currentTarget as HTMLSelectElement).value } })}
										class="lm-select rounded-[5px] border px-1.5 py-1 font-mono text-[10px] uppercase tracking-[0.12em] disabled:opacity-40"
									>
										{#each LEAD_MAGNET_STATUSES as s (s)}<option value={s}>{s}</option>{/each}
									</select>
									{#if confirming === m.id}
										<button
											type="button"
											onclick={() => call(m.id, { method: 'DELETE' })}
											disabled={busy === m.id}
											data-lens="c"
											class="bn-pressable pc-pill rounded-[5px] border px-1.5 py-1 font-mono text-[10px] uppercase tracking-widest disabled:opacity-40"
											style="color: var(--bn-err)">sure?</button
										>
										<button type="button" onclick={() => (confirming = null)} data-lens="c" class="bn-pressable lm-no bn-dim font-mono text-[10px] uppercase tracking-widest">no</button>
									{:else}
										<button
											type="button"
											onclick={() => (confirming = m.id)}
											aria-label="Delete lead magnet"
											title="Delete"
											data-lens="c"
											class="bn-pressable lm-del bn-dim opacity-0 transition-opacity group-hover:opacity-100 focus-visible:opacity-100"
										>
											<Trash2 size={14} />
										</button>
									{/if}
								</span>
								{#if failure?.id === m.id}
									<div class="mt-1 font-mono text-[10px]" style="color: var(--bn-err)">{failure.msg}</div>
								{/if}
							</td>
						{/if}
					</tr>
				{/each}
			</tbody>
		</table>
	</div>
{/if}

<style>
	.lm-name:hover {
		color: var(--bn-accent);
	}
	.lm-del:hover {
		color: var(--bn-err);
	}
	.lm-select {
		border-color: var(--bn-border);
		background: var(--bn-bg);
		color: var(--bn-text-2);
		outline: none;
	}
	.lm-select:focus {
		border-color: var(--bn-border-strong);
	}
	.lm-no:hover {
		color: var(--bn-text);
	}
</style>
