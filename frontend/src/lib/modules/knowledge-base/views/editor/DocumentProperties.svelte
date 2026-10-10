<script lang="ts">
	import { ChevronDown, ChevronRight, FileText, Tag, Hash, Link, Calendar, Layers } from 'lucide-svelte';
	import type { Document, Block } from '../../entities/types';

	interface Props {
		doc: Document;
		content: Block[];
	}

	let { doc, content }: Props = $props();

	let expanded = $state(false);

	interface Frontmatter {
		signal_mode?: string;
		genre?: string;
		type?: string;
		format?: string;
		sn_ratio?: string | number;
		node?: string;
		date?: string;
		tags?: string[] | string;
		[key: string]: unknown;
	}

	function parseFrontmatter(blocks: Block[]): Frontmatter | null {
		const firstBlock = blocks[0];
		if (!firstBlock) return null;

		let rawText = '';
		if (Array.isArray(firstBlock.content)) {
			rawText = firstBlock.content.map((rt) => rt.plain_text || '').join('');
		}

		const trimmed = rawText.trim();
		if (!trimmed.startsWith('---')) return null;

		const endIdx = trimmed.indexOf('---', 3);
		if (endIdx === -1) return null;

		const yamlBlock = trimmed.slice(3, endIdx).trim();
		const result: Frontmatter = {};

		for (const line of yamlBlock.split('\n')) {
			const colonIdx = line.indexOf(':');
			if (colonIdx === -1) continue;
			const key = line.slice(0, colonIdx).trim();
			const val = line.slice(colonIdx + 1).trim();
			if (!key) continue;

			if (val.startsWith('[') && val.endsWith(']')) {
				result[key] = val
					.slice(1, -1)
					.split(',')
					.map((s) => s.trim().replace(/^['"]|['"]$/g, ''))
					.filter(Boolean);
			} else {
				result[key] = val.replace(/^['"]|['"]$/g, '');
			}
		}

		return Object.keys(result).length > 0 ? result : null;
	}

	function computeStats(blocks: Block[]): { words: number; chars: number } {
		let words = 0;
		let chars = 0;
		for (const block of blocks) {
			if (Array.isArray(block.content)) {
				const t = block.content.map((rt) => rt.plain_text || '').join('');
				chars += t.length;
				words += t.trim() ? t.trim().split(/\s+/).length : 0;
			}
		}
		return { words, chars };
	}

	function formatDate(dateStr: string | undefined): string {
		if (!dateStr) return '—';
		try {
			return new Date(dateStr).toLocaleDateString('en-US', {
				month: 'short',
				day: 'numeric',
				year: 'numeric'
			});
		} catch {
			return dateStr;
		}
	}

	const frontmatter = $derived(parseFrontmatter(content));
	const stats = $derived(computeStats(content));

	const nodeFromId = $derived(doc.id.includes('/') ? doc.id.split('/')[0] : '');
	const relativePath = $derived(doc.id);

	const displayDate = $derived(formatDate((frontmatter?.date as string) || doc.created_at));
	const modifiedDate = $derived(formatDate(doc.updated_at));

	const tags = $derived.by((): string[] => {
		const t = frontmatter?.tags;
		if (Array.isArray(t)) return t as string[];
		if (typeof t === 'string' && t) return [t];
		return [];
	});

	const summaryParts = $derived.by(() => {
		const parts: string[] = [`${stats.words} words`];
		if (frontmatter?.genre) parts.push(frontmatter.genre as string);
		if (nodeFromId) parts.push(nodeFromId);
		return parts.join(' · ');
	});
</script>

<div class="dp" class:dp--expanded={expanded}>
	<!-- Toggle bar — always visible -->
	<button class="dp__toggle" onclick={() => (expanded = !expanded)} aria-label="Toggle document properties">
		{#if expanded}
			<ChevronDown class="dp__chevron" />
		{:else}
			<ChevronRight class="dp__chevron" />
		{/if}
		<span class="dp__toggle-label">Properties</span>
		{#if !expanded}
			<span class="dp__summary">{summaryParts}</span>
		{/if}
	</button>

	{#if expanded}
		<div class="dp__grid">
			<div class="dp__row">
				<span class="dp__key"><FileText class="dp__icon" /> Title</span>
				<span class="dp__val">{doc.title || '—'}</span>
			</div>

			{#if nodeFromId || frontmatter?.node}
				<div class="dp__row">
					<span class="dp__key"><Layers class="dp__icon" /> Node</span>
					<span class="dp__val dp__val--badge">{(frontmatter?.node as string) || nodeFromId}</span>
				</div>
			{/if}

			{#if frontmatter?.type || frontmatter?.genre}
				<div class="dp__row">
					<span class="dp__key"><Hash class="dp__icon" /> Type</span>
					<span class="dp__val">
						{#if frontmatter?.genre}<span class="dp__badge">{frontmatter.genre}</span>{/if}
						{#if frontmatter?.type}<span class="dp__badge">{frontmatter.type}</span>{/if}
					</span>
				</div>
			{/if}

			{#if frontmatter?.signal_mode}
				<div class="dp__row">
					<span class="dp__key"><Hash class="dp__icon" /> Mode</span>
					<span class="dp__val">{frontmatter.signal_mode}</span>
				</div>
			{/if}

			{#if frontmatter?.sn_ratio}
				<div class="dp__row">
					<span class="dp__key"><Hash class="dp__icon" /> S/N</span>
					<span class="dp__val">{frontmatter.sn_ratio}</span>
				</div>
			{/if}

			{#if tags.length > 0}
				<div class="dp__row">
					<span class="dp__key"><Tag class="dp__icon" /> Tags</span>
					<span class="dp__val dp__val--tags">
						{#each tags as tag}
							<span class="dp__tag">{tag}</span>
						{/each}
					</span>
				</div>
			{/if}

			<div class="dp__row">
				<span class="dp__key"><Calendar class="dp__icon" /> Created</span>
				<span class="dp__val">{displayDate}</span>
			</div>

			<div class="dp__row">
				<span class="dp__key"><Calendar class="dp__icon" /> Modified</span>
				<span class="dp__val">{modifiedDate}</span>
			</div>

			<div class="dp__row">
				<span class="dp__key"><Hash class="dp__icon" /> Stats</span>
				<span class="dp__val">{stats.words} words · {stats.chars} chars</span>
			</div>

			<div class="dp__row">
				<span class="dp__key"><Link class="dp__icon" /> Backlinks</span>
				<span class="dp__val dp__val--muted">—</span>
			</div>

			<div class="dp__row">
				<span class="dp__key"><FileText class="dp__icon" /> Path</span>
				<span class="dp__val dp__val--mono" title={relativePath}>{relativePath}</span>
			</div>
		</div>
	{/if}
</div>

<style>
	.dp {
		border-bottom: 1px solid var(--dbd);
		background: var(--dbg);
		font-size: 12px;
	}

	.dp__toggle {
		display: flex;
		align-items: center;
		gap: 6px;
		width: 100%;
		padding: 6px 2px;
		background: transparent;
		border: none;
		cursor: pointer;
		text-align: left;
		color: var(--dt3);
		transition: color 0.12s;
	}

	.dp__toggle:hover {
		color: var(--dt);
	}

	:global(.dp__chevron) {
		width: 13px;
		height: 13px;
		flex-shrink: 0;
	}

	.dp__toggle-label {
		font-size: 11px;
		font-weight: 600;
		letter-spacing: 0.04em;
		text-transform: uppercase;
		color: inherit;
	}

	.dp__summary {
		font-size: 11px;
		color: var(--dt4);
		margin-left: 2px;
		opacity: 0.75;
	}

	.dp__grid {
		padding: 4px 0 10px;
		display: flex;
		flex-direction: column;
		gap: 1px;
	}

	.dp__row {
		display: grid;
		grid-template-columns: 130px 1fr;
		align-items: baseline;
		gap: 8px;
		padding: 4px 2px;
		border-radius: 4px;
		transition: background 0.1s;
	}

	.dp__row:hover {
		background: var(--dbg2);
	}

	.dp__key {
		display: flex;
		align-items: center;
		gap: 5px;
		color: var(--dt3);
		font-size: 11.5px;
		font-weight: 500;
		white-space: nowrap;
		user-select: none;
	}

	:global(.dp__icon) {
		width: 12px;
		height: 12px;
		flex-shrink: 0;
		opacity: 0.65;
	}

	.dp__val {
		color: var(--dt);
		font-size: 11.5px;
		min-width: 0;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}

	.dp__val--muted {
		color: var(--dt4);
	}

	.dp__val--mono {
		font-family: ui-monospace, 'Cascadia Code', 'Source Code Pro', Menlo, Consolas, monospace;
		font-size: 10.5px;
		color: var(--dt3);
	}

	.dp__val--badge {
		font-weight: 500;
		color: var(--dt2);
	}

	.dp__val--tags {
		display: flex;
		flex-wrap: wrap;
		gap: 4px;
		white-space: normal;
	}

	.dp__badge {
		display: inline-block;
		padding: 1px 7px;
		border-radius: 4px;
		background: var(--dbg3);
		border: 1px solid var(--dbd);
		color: var(--dt2);
		font-size: 10.5px;
		font-weight: 500;
	}

	.dp__tag {
		display: inline-block;
		padding: 1px 7px;
		border-radius: 20px;
		background: var(--dbg3);
		border: 1px solid var(--dbd);
		color: var(--dt3);
		font-size: 10.5px;
	}
</style>
