<!-- Read one piece of agent work, then decide on it (FounderOS v1
     components/TaskReviewPanel.tsx): a panel that slides in from the right over
     a dimmed page. Text comes back as JSON and renders as the document it is;
     images and PDFs render from the inline URL; binaries are download-only; a
     proposal is a live page. Esc or the backdrop closes it. Portalled to
     <body> so a lifted (transformed) card cannot pin it inside itself. -->
<script lang="ts">
	import { Check, Download, ExternalLink, Undo2, X } from '$lib/founderos/icons';
	import { founderosFetch, founderosUrl } from '$lib/founderos/api';
	import Label from '$lib/founderos/kit/Label.svelte';
	import Markdown from './Markdown.svelte';
	import { actionTextOf, jsonBodyOf } from './review';
	import type { Decision, DeliverableItem, DeliverableView } from './types';

	let {
		item,
		decision,
		onDecide,
		onClose
	}: {
		item: DeliverableItem | null;
		decision?: Decision;
		onDecide: (item: DeliverableItem, kind: Decision['decision'] | null) => void;
		onClose: () => void;
	} = $props();

	let preview = $state<DeliverableView | null>(null);
	let loading = $state(false);
	let failed = $state(false);

	$effect(() => {
		const it = item;
		preview = null;
		failed = false;
		loading = false;
		if (!it || it.kind !== 'file') return;
		loading = true;
		let cancelled = false;
		founderosFetch<DeliverableView>(`/pages/board/deliverables?file=${encodeURIComponent(it.id)}&mode=view`)
			.then((v) => {
				if (!cancelled) preview = v;
			})
			.catch(() => {
				if (!cancelled) failed = true;
			})
			.finally(() => {
				if (!cancelled) loading = false;
			});
		return () => {
			cancelled = true;
		};
	});

	// Esc closes, so reviewing a queue never traps him in a panel.
	$effect(() => {
		if (!item) return;
		const onKey = (e: KeyboardEvent) => e.key === 'Escape' && onClose();
		window.addEventListener('keydown', onKey);
		return () => window.removeEventListener('keydown', onKey);
	});

	function portal(node: HTMLElement) {
		document.body.appendChild(node);
		return { destroy: () => node.remove() };
	}

	const raw = $derived(item ? founderosUrl(`/pages/board/deliverables?file=${encodeURIComponent(item.id)}`) : '');
	const kind = $derived(!item ? null : item.kind === 'link' ? 'link' : (preview?.kind ?? null));
	const action = $derived(item ? ((item as { action?: string }).action || actionTextOf(item.name)) : '');
</script>

{#if item}
	<!-- svelte-ignore a11y_click_events_have_key_events -->
	<div use:portal role="dialog" tabindex="-1" aria-modal="true" aria-label="Review {item.title || item.name}" data-part="review-overlay" class="bn-review-scrim fixed inset-0 z-50 flex justify-end" onclick={onClose}>
		<!-- svelte-ignore a11y_no_noninteractive_element_interactions -->
		<aside data-part="review" class="bn-review bn-panel-in flex h-full w-full max-w-2xl flex-col border-l" onclick={(e) => e.stopPropagation()}>
			<header class="bn-review-line flex shrink-0 items-start gap-3 border-b px-4 py-3">
				<div class="min-w-0 flex-1">
					<Label>Review</Label>
					<!-- The document's own title, not the filename. -->
					<div class="bn-text mt-1 break-words text-[13.5px] font-semibold leading-snug">{item.title || item.name}</div>
					<div class="bn-dim mt-0.5 font-mono text-[9.5px]" title={item.name}>
						{item.name} · {item.meta}{#if decision}<span style:color={decision.decision === 'approved' ? 'var(--bn-ok)' : 'var(--bn-text-2)'}>{' · '}{decision.decision}</span>{/if}
					</div>
				</div>
				<button type="button" aria-label="Close review" class="bn-pressable bn-review-x bn-dim shrink-0 border p-1" onclick={onClose}><X class="h-3.5 w-3.5" /></button>
			</header>

			{#if item.kind === 'file'}
				<p class="bn-review-line bn-muted shrink-0 border-b px-4 py-2 text-[11.5px]">{action}</p>
			{/if}

			<div class="min-h-0 flex-1 overflow-auto">
				{#if loading}<p class="bn-dim px-4 py-3 font-mono text-[10.5px]">reading…</p>{/if}
				{#if failed}
					<p class="px-4 py-3 font-mono text-[10.5px]" style:color="var(--bn-err)">Could not read this file. It may have been moved or the board host is unreachable.</p>
				{/if}
				{#if !loading && !failed && kind === 'text'}
					{#if preview?.truncated}
						<p class="bn-review-line bn-review-trunc border-b px-4 py-1.5 font-mono text-[9.5px]">Showing the first part only. Download for the whole file.</p>
					{/if}
					<div class="px-4 py-3"><Markdown text={jsonBodyOf(item.name, preview?.text ?? '')} /></div>
				{:else if !loading && !failed && kind === 'image'}
					<img src="{raw}&inline=1" alt={item.name} class="max-w-full p-4" />
				{:else if !loading && !failed && kind === 'pdf'}
					<iframe src="{raw}&inline=1" title={item.name} class="h-full min-h-[70vh] w-full"></iframe>
				{:else if !loading && !failed && kind === 'binary'}
					<p class="bn-dim px-4 py-3 font-mono text-[10.5px]">This format can't be shown here. Download it to review.</p>
				{/if}
				{#if kind === 'link'}
					<div class="bn-dim px-4 py-3 font-mono text-[10.5px]">
						This is a live proposal page rather than a file on the board.{#if item.accessCode}
							Gate code <span class="bn-text">{item.accessCode}</span>.{/if}
					</div>
				{/if}
			</div>

			<footer class="bn-review-line flex shrink-0 flex-wrap items-center gap-2 border-t px-4 py-3">
				{#if decision}
					<button type="button" class="bn-pressable bn-review-btn bn-muted flex items-center gap-1.5 border px-3 py-1.5 font-mono text-[10px] uppercase tracking-[0.12em]" onclick={() => onDecide(item, null)}>
						<Undo2 class="h-3 w-3" /> undo {decision.decision}
					</button>
				{:else}
					<button type="button" class="bn-pressable bn-review-btn bn-review-ok flex items-center gap-1.5 border px-3 py-1.5 font-mono text-[10px] font-bold uppercase tracking-[0.12em]" onclick={() => onDecide(item, 'approved')}>
						<Check class="h-3 w-3" /> approve
					</button>
					<button type="button" class="bn-pressable bn-review-btn bn-muted flex items-center gap-1.5 border px-3 py-1.5 font-mono text-[10px] uppercase tracking-[0.12em]" onclick={() => onDecide(item, 'dismissed')}>
						<X class="h-3 w-3" /> dismiss
					</button>
				{/if}
				<a href={item.kind === 'link' ? (item.url ?? raw) : raw} target="_blank" rel="noreferrer" class="bn-linky bn-dim ml-auto flex items-center gap-1 font-mono text-[10px]">
					{#if item.kind === 'link'}open page <ExternalLink class="h-3 w-3" />{:else}download <Download class="h-3 w-3" />{/if}
				</a>
			</footer>

			<p class="bn-review-line bn-dim shrink-0 border-t px-4 py-1.5 font-mono text-[9px]">
				Your call is recorded in the OS and clears this from the queue. If the agent rewrites the file, it comes back.
			</p>
		</aside>
	</div>
{/if}

<style>
	/* Prod's bg-os-bg/70, bg-os-text/[0.03], bg-os-warn/[0.08], border-os-ok/50
	   and hover:bg-os-ok/10 are opacity modifiers on var() colours, which
	   Tailwind v3 never generates: the scrim is clear, the action and
	   truncation lines carry no tint, and the approve border falls back to
	   preflight's gray-200. The port draws what prod shows. */
	.bn-review-scrim {
		background: transparent;
	}
	.bn-review {
		border-color: var(--bn-border-strong);
		background: var(--bn-surface);
	}
	.bn-review-line {
		border-color: var(--bn-border);
	}
	.bn-review-trunc {
		color: var(--bn-warn);
	}
	.bn-review-x,
	.bn-review-btn {
		border-color: var(--bn-border);
		border-radius: 8px;
	}
	.bn-review-x:hover,
	.bn-review-btn:hover {
		border-color: var(--bn-text-3);
		color: var(--bn-text);
	}
	.bn-review-ok {
		border-color: rgb(229 231 235);
		color: var(--bn-ok);
	}
	.bn-review-ok:hover {
		border-color: rgb(229 231 235);
		color: var(--bn-ok);
	}
</style>
