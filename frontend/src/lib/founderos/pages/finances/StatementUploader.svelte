<!-- Statement uploader (FounderOS v1 StatementUploader.tsx). The lane says what
     the file IS: a card statement (CSV or PDF) lands in the spend ledger under
     that card; a bank statement lands in the per-business income store. -->
<script lang="ts">
	import { Upload } from '$lib/founderos/icons';
	import { founderosFetch } from '$lib/founderos/api';
	import './fin.css';
	import { CARD_LANES, DEFAULT_CARD, monthName, type CardId } from './spend-report';
	import type { BankUploadResult, StatementUploadResult } from './types';

	type Target = CardId | 'bank';
	const TARGETS: { id: Target; label: string }[] = [...CARD_LANES.map((c) => ({ id: c.id as Target, label: c.label })), { id: 'bank', label: 'Bank statement · income' }];

	let { onuploaded, target = $bindable(DEFAULT_CARD) }: { onuploaded?: (months: string[]) => void; target?: Target } = $props();

	let status = $state<string | null>(null);
	let busy = $state(false);
	let dragOver = $state(false);

	async function upload(file: File) {
		const bank = target === 'bank';
		busy = true;
		status = `Parsing ${file.name}…`;
		try {
			const body = new FormData();
			body.append('file', file);
			if (!bank) body.append('card', target);
			if (bank) {
				const data = await founderosFetch<BankUploadResult>('/pages/finances/bank-statement', { method: 'POST', body });
				status = `✓ ${data.summary.business} ${data.summary.month}: ${'$' + (data.summary.creditsCents / 100).toLocaleString('en-US', { maximumFractionDigits: 0 })} in`;
				onuploaded?.([]);
			} else {
				const data = await founderosFetch<StatementUploadResult>('/pages/finances/statements', { method: 'POST', body });
				const landed = Array.isArray(data.uploadedMonths) ? data.uploadedMonths : [];
				const newest = landed.length > 0 ? landed[landed.length - 1] : null;
				status = `✓ ${data.inserted} new of ${data.parsed} parsed rows${newest ? ` · ${monthName(newest)}` : ''}`;
				onuploaded?.(landed);
			}
		} catch (err) {
			status = `✗ ${err instanceof Error ? err.message : 'upload failed'}`;
		} finally {
			busy = false;
		}
	}

	async function onFile(e: Event) {
		const input = e.currentTarget as HTMLInputElement;
		const file = input.files?.[0];
		if (file) await upload(file);
		input.value = '';
	}
</script>

<div
	role="region"
	aria-label="Upload statements"
	class="fin-drop flex h-full flex-col items-center justify-center gap-2 rounded-[10px] border border-dashed px-5 py-6 text-center"
	data-over={dragOver}
	ondragover={(e) => {
		e.preventDefault();
		dragOver = true;
	}}
	ondragleave={() => (dragOver = false)}
	ondrop={(e) => {
		e.preventDefault();
		dragOver = false;
		const file = e.dataTransfer?.files?.[0];
		if (file && !busy) void upload(file);
	}}
>
	<span class="bn-dim"><Upload size={20} strokeWidth={1.6} /></span>
	<div class="bn-muted text-[13px] font-semibold">Upload statements</div>

	<!-- the lane says what the file IS -->
	<div class="flex w-full max-w-[260px] flex-col gap-1">
		{#each TARGETS as t (t.id)}
			<button
				type="button"
				data-lens="c"
				class="fin-lane bn-pressable rounded-[5px] border px-2 py-1 font-mono text-[10px] uppercase tracking-[0.1em]"
				data-on={t.id === target}
				aria-pressed={t.id === target}
				onclick={() => (target = t.id)}>{t.label}</button
			>
		{/each}
	</div>

	<p class="bn-dim max-w-[260px] font-mono text-[10px] leading-relaxed">
		Drop a CSV or PDF here, or pick one below.
		{target === 'bank' ? 'Bank statement: per-business income and net.' : 'Card statement: categorized spend and subscriptions for this lane.'}
		Parsed locally, stored in the local database, never committed.
	</p>

	<label class="fin-primary bn-pressable rounded-[6px] px-3 py-1.5" data-lens="c">
		{busy ? 'Working…' : '↑ Upload statement'}
		<input type="file" accept=".csv,text/csv,.pdf,application/pdf,.txt,text/plain" class="hidden" onchange={onFile} disabled={busy} />
	</label>
	{#if status}<div class="bn-muted mt-1 max-w-[260px] font-mono text-[10px]" role="status">{status}</div>{/if}
</div>
