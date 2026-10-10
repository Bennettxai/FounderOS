<!-- The Deliverables feed (FounderOS v1 components/DeliverablesList.tsx), from
     the GET /pages/board/deliverables payload AgentsTabs holds. ONE flat feed
     (2026-09-24): files, proposals and every board task that reached done,
     newest first, the folder surviving only as the row's source tag. Each
     file or proposal row carries its own unseen dot (2026-08-22): green for
     new since he last opened anything here, amber for rewritten since he read
     it. The dot runs off the OPENED store, not the tab's seen map, so looking
     at the tab does not wipe every dot on it; opening the row is what clears
     it. The review panel reads the work without leaving the tab. -->
<script lang="ts">
	import { founderosFetch, founderosUrl } from '$lib/founderos/api';
	import Label from '$lib/founderos/kit/Label.svelte';
	import { CircleCheck, Download, ExternalLink, FileText, RefreshCw } from '$lib/founderos/icons';
	import ReviewPanel from './ReviewPanel.svelte';
	import { ago } from './board';
	import { deliverablesFeed, feedCounts, type FeedRow } from './deliverables-feed';
	import { writeFailure } from './runerror';
	import type { UnseenState } from '../console/needs-queue';
	import type { Decision, DeliverableItem, DeliverablesBody, PaperclipIssue } from './types';

	let {
		data,
		error = null,
		issues = [],
		refreshing = false,
		unseenOf = () => null,
		onOpen = () => {},
		onReload
	}: {
		data: DeliverablesBody | null;
		error?: string | null;
		/** The live board's issues; the finished ones join the feed. */
		issues?: PaperclipIssue[];
		refreshing?: boolean;
		/** Per-row dot: new, rewritten since he read it, or nothing. */
		unseenOf?: (item: DeliverableItem) => UnseenState;
		/** Opening an item for review is what clears ITS dot. */
		onOpen?: (item: DeliverableItem) => void;
		onReload: () => void;
	} = $props();

	type Show = 'all' | 'files' | 'links' | 'done';
	let show = $state<Show>('all');
	let copied = $state<string | null>(null);
	const matchesShow = (r: FeedRow, f: Show) =>
		f === 'all' || (f === 'done' ? r.kind === 'done' : r.kind === 'item' && (f === 'links' ? r.item.kind === 'link' : r.item.kind === 'file'));
	const feed = $derived(deliverablesFeed(data?.groups ?? null, issues));
	const counts = $derived(feedCounts(feed));
	const rows = $derived(feed.filter((r) => matchesShow(r, show)));
	const SHOWS = $derived<Array<[Show, string, number]>>([
		['all', 'All', counts.all],
		['files', 'Files', counts.files],
		['links', 'Proposals', counts.links],
		['done', 'Done tasks', counts.done]
	]);
	const size = (b: number): string => (b < 1024 ? `${b} B` : b < 1024 * 1024 ? `${(b / 1024).toFixed(0)} KB` : `${(b / 1024 / 1024).toFixed(1)} MB`);

	// The operator sends the link and the code together, so the code is one tap away.
	async function copyCode(id: string, code: string) {
		try {
			await navigator.clipboard.writeText(code);
			copied = id;
			setTimeout(() => (copied = copied === id ? null : copied), 1400);
		} catch {
			/* clipboard blocked: the code is on screen anyway */
		}
	}

	let reviewing = $state<DeliverableItem | null>(null);
	let failure = $state<string | null>(null);
	// One decision write at a time, so a double click cannot record it twice.
	let busy = $state(false);

	const decisionOf = (id: string) => data?.decisions.find((d) => d.id === id);
	const fileUrl = (id: string) => founderosUrl(`/pages/board/deliverables?file=${encodeURIComponent(id)}`);

	function review(d: DeliverableItem) {
		onOpen(d);
		reviewing = d;
	}

	async function decide(item: DeliverableItem, kind: Decision['decision'] | null) {
		if (busy) return;
		busy = true;
		failure = null;
		try {
			await founderosFetch('/pages/board/deliverables/decision', {
				method: 'POST',
				json: kind === null ? { id: item.id, decision: null } : { id: item.id, decision: kind, decidedRevision: item.revision }
			});
			onReload();
		} catch (err) {
			failure = writeFailure(err, 'decision');
		} finally {
			busy = false;
		}
	}
</script>

<div data-part="deliverables" class="min-h-[240px]">
	<section class="bn-feed overflow-hidden rounded-[10px] border">
		<div class="bn-feed-head flex flex-wrap items-center gap-x-3 gap-y-2 border-b px-4 py-2.5">
			<Label>Agent deliverables</Label>
			<span class="bn-dim font-mono text-[10px]">{data === null ? (error ? 'files unreadable' : 'loading files…') : `${counts.all} delivered · newest first`}</span>
			<div class="flex flex-wrap items-center gap-1">
				{#each SHOWS as [id, label, n] (id)}
					<button type="button" aria-pressed={show === id} onclick={() => (show = id)} class="bn-pressable bn-show rounded-full px-2.5 py-0.5 font-mono text-[10px]" class:is-on={show === id}>{label} {n}</button>
				{/each}
			</div>
			<button type="button" onclick={onReload} title="Refresh" aria-label="Refresh" class="bn-pressable bn-dim bn-hover ml-auto rounded-full p-1.5"><RefreshCw class="h-3.5 w-3.5 {refreshing ? 'animate-spin' : ''}" /></button>
		</div>
		{#if error}
			<p class="px-4 py-3 font-mono text-[10.5px]" style:color="var(--bn-err)">Deliverables unreadable: {error}</p>
		{/if}
		{#if failure}<p class="px-4 pt-3 font-mono text-[10.5px]" style:color="var(--bn-err)">{failure}</p>{/if}
		{#if data !== null && feed.length === 0}
			<p class="bn-dim px-4 py-3 font-mono text-[10.5px]">
				Nothing in review yet. Proposals, finished board tasks and agent files land here when agents write into their workspace deliverables folders on the board host — a dev machine that does not host the board honestly shows none.
			</p>
		{/if}
		{#if feed.length > 0 && rows.length === 0}
			<p class="bn-dim px-4 py-3 font-mono text-[10.5px]">Nothing of that kind yet.</p>
		{/if}
		{#if rows.length > 0}
			<div class="bn-feed-rows max-h-[60vh] overflow-y-auto">
				{#each rows as r (r.id)}
					{#if r.kind === 'done'}
						<div data-part="feed-row" class="flex items-center gap-3 px-4 py-2">
							<CircleCheck class="h-4 w-4 shrink-0" style="color: var(--bn-ok)" />
							<div class="min-w-0 flex-1">
								<div class="bn-text truncate text-[12px] font-semibold" title={r.issue.title}>{r.issue.title}</div>
								<div class="bn-dim truncate font-mono text-[9.5px]">
									{[r.source, r.issue.identifier, r.issue.assigneeName, r.issue.updatedAt ? ago(r.issue.updatedAt) : null].filter(Boolean).join(' · ')}
								</div>
							</div>
						</div>
					{:else}
						{@const d = r.item}
						{@const unseen = unseenOf(d)}
						<div data-part="feed-row" class="flex items-center gap-3 px-4 py-2">
							{#if d.kind === 'link'}<ExternalLink class="bn-muted h-4 w-4 shrink-0" />{:else}<FileText class="bn-muted h-4 w-4 shrink-0" />{/if}
							<div class="min-w-0 flex-1">
								<div class="flex items-center gap-2">
									{#if unseen}
										<span
											class="bn-unseen h-1.5 w-1.5 shrink-0 animate-pulse rounded-full"
											title={unseen === 'new' ? 'New since you last looked' : 'Rewritten since you read it'}
											aria-label={unseen === 'new' ? 'new' : 'updated'}
											style:background={unseen === 'new' ? 'var(--bn-accent)' : 'var(--bn-warn)'}
										></span>
									{/if}
									<span class="bn-text truncate text-[12px] font-semibold" title={d.name}>{d.title || d.name}</span>
									{#if d.accessCode}
										<button type="button" onclick={() => copyCode(d.id, d.accessCode)} title="Copy the access code that opens this proposal" class="bn-pressable bn-code shrink-0 rounded-full border px-2 py-0.5 font-mono text-[9px] tracking-[0.08em]">{copied === d.id ? 'copied' : d.accessCode}</button>
									{/if}
								</div>
								<div class="bn-dim truncate font-mono text-[9.5px]">
									{[r.source, d.meta, d.sizeBytes !== null ? size(d.sizeBytes) : null, ago(d.modifiedAt)].filter(Boolean).join(' · ')}
								</div>
							</div>
							<button type="button" onclick={() => review(d)} title="Review this without leaving the OS" class="bn-pressable bn-ghost flex shrink-0 items-center gap-1.5 rounded-full border px-3 py-1.5 font-mono text-[10px]">review</button>
							{#if d.kind === 'link'}
								<a href={d.url ?? '#'} target="_blank" rel="noreferrer" data-lens="c" class="bn-pressable is-dark bn-act flex shrink-0 items-center gap-1.5 rounded-full border px-3 py-1.5 font-mono text-[10px]"><ExternalLink class="h-3 w-3" /> open</a>
							{:else}
								<a href={fileUrl(d.id)} download data-lens="c" class="bn-pressable is-dark bn-act flex shrink-0 items-center gap-1.5 rounded-full border px-3 py-1.5 font-mono text-[10px]"><Download class="h-3 w-3" /> download</a>
							{/if}
						</div>
					{/if}
				{/each}
			</div>
		{/if}
	</section>

	<ReviewPanel item={reviewing} decision={reviewing ? decisionOf(reviewing.id) : undefined} onDecide={(it, k) => decide(it, k)} onClose={() => (reviewing = null)} />
</div>

<style>
	.bn-feed,
	.bn-feed-head {
		border-color: var(--bn-border);
		background: var(--bn-surface);
	}
	.bn-feed-rows > :global(* + *) {
		border-top: 1px solid var(--bn-border);
	}
	.bn-show {
		border: 1px solid var(--bn-border);
		color: var(--bn-text-3);
	}
	.bn-show:hover,
	.bn-hover:hover {
		color: var(--bn-text);
	}
	.bn-show.is-on {
		border-color: var(--bn-accent);
		background: var(--bn-accent);
		color: var(--bn-accent-ink);
		font-weight: 600;
	}
	.bn-code {
		border-color: var(--bn-border-strong);
		color: var(--bn-text-2);
	}
	.bn-act {
		border-color: var(--bn-border-strong);
		background: var(--bn-surface-2);
		color: var(--bn-text);
	}
	.bn-ghost {
		border-color: var(--bn-border);
		color: var(--bn-text-3);
	}
	.bn-ghost:hover {
		border-color: var(--bn-text-3);
		color: var(--bn-text);
	}
	.bn-code:hover {
		border-color: var(--bn-text-3);
		color: var(--bn-text);
	}
</style>
