<script lang="ts">
	import EmailListRow from './EmailListRow.svelte';
	import EmailEmptyState from './EmailEmptyState.svelte';
	import { emptyFolderCopy } from './commsEmailUtils';
	import type { EmailFolder, EmailThread } from '$lib/api/comms';

	interface Props {
		threads: EmailThread[];
		selectedThreadId: string | null;
		isLoading: boolean;
		isRefreshing: boolean;
		showProviderBadge: boolean;
		searchQuery: string;
		folder: EmailFolder;
		folderLabel: string;
		errorMessage?: string | null;
		onSelectThread: (thread: EmailThread) => void;
		onToggleStar?: (thread: EmailThread) => void;
		onRetry?: () => void;
		onClearSearch?: () => void;
		onSync?: () => void;
	}

	let {
		threads,
		selectedThreadId,
		isLoading,
		isRefreshing,
		showProviderBadge,
		searchQuery,
		folder,
		folderLabel,
		errorMessage = null,
		onSelectThread,
		onToggleStar,
		onRetry,
		onClearSearch,
		onSync,
	}: Props = $props();

	const folderEmpty = $derived(emptyFolderCopy(folder));

	const SKELETON_ROWS = 5;
	const skeletons = Array.from({ length: SKELETON_ROWS }, (_, i) => i);

	const isSearching = $derived(searchQuery.trim().length > 0);
	const isEmpty = $derived(!isLoading && !errorMessage && threads.length === 0);
</script>

<div class="cm-email-list">
	{#if isRefreshing && !isLoading}
		<div class="cm-email-list__progress" role="progressbar" aria-label="Refreshing"></div>
	{/if}

	{#if errorMessage}
		<EmailEmptyState
			variant="error"
			title="Couldn't load this folder"
			description={errorMessage}
			actionLabel={onRetry ? 'Try again' : undefined}
			onAction={onRetry}
		/>
	{:else if isLoading}
		<div class="cm-email-list__skeletons" aria-busy="true">
			{#each skeletons as i (i)}
				<div class="cm-email-list__skeleton-row">
					<span class="cm-email-list__skeleton-dot"></span>
					<span class="cm-email-list__skeleton-avatar"></span>
					<span class="cm-email-list__skeleton-body">
						<span class="cm-email-list__skeleton-bar cm-email-list__skeleton-bar--top"></span>
						<span class="cm-email-list__skeleton-bar cm-email-list__skeleton-bar--mid"></span>
						<span class="cm-email-list__skeleton-bar cm-email-list__skeleton-bar--bot"></span>
					</span>
				</div>
			{/each}
		</div>
	{:else if isEmpty && isSearching}
		<EmailEmptyState
			variant="no-match"
			title="No matches in {folderLabel}"
			description="Try different words, or clear search to see everything."
			actionLabel="Clear search"
			onAction={onClearSearch}
		/>
	{:else if isEmpty}
		<EmailEmptyState
			variant="empty-folder"
			title={folderEmpty.title}
			description={folderEmpty.description}
			actionLabel={folder === 'inbox' && onSync ? folderEmpty.actionLabel : undefined}
			onAction={folder === 'inbox' ? onSync : undefined}
		/>
	{:else}
		<ul class="cm-email-list__rows">
			{#each threads as thread (thread.id)}
				<li>
					<EmailListRow
						{thread}
						isSelected={thread.id === selectedThreadId}
						{showProviderBadge}
						onSelect={onSelectThread}
						{onToggleStar}
					/>
				</li>
			{/each}
		</ul>
	{/if}
</div>

<style>
	.cm-email-list {
		position: relative;
		flex: 1;
		min-height: 0;
		overflow-y: auto;
		background: var(--dbg);
	}

	.cm-email-list__progress {
		position: sticky;
		top: 0;
		left: 0;
		right: 0;
		height: 2px;
		background: linear-gradient(
			90deg,
			transparent 0%,
			var(--bos-accent-blue) 50%,
			transparent 100%
		);
		background-size: 200% 100%;
		animation: cm-email-progress 1.4s linear infinite;
		z-index: var(--bos-z-sticky);
	}

	@keyframes cm-email-progress {
		from { background-position: 100% 0; }
		to { background-position: -100% 0; }
	}

	.cm-email-list__rows {
		list-style: none;
		margin: 0;
		padding: 0;
	}

	.cm-email-list__skeletons {
		display: flex;
		flex-direction: column;
	}

	.cm-email-list__skeleton-row {
		display: grid;
		grid-template-columns: 16px 32px 1fr;
		gap: var(--space-3);
		padding: var(--space-3) var(--space-4);
		border-bottom: 1px solid var(--dbd);
	}

	.cm-email-list__skeleton-dot,
	.cm-email-list__skeleton-avatar,
	.cm-email-list__skeleton-bar {
		background: linear-gradient(
			90deg,
			var(--dbg2) 0%,
			var(--dbg3) 50%,
			var(--dbg2) 100%
		);
		background-size: 200% 100%;
		animation: cm-email-shimmer 1.6s ease-in-out infinite;
	}

	.cm-email-list__skeleton-dot {
		width: 7px;
		height: 7px;
		border-radius: var(--radius-full);
		margin-top: 10px;
		justify-self: center;
	}

	.cm-email-list__skeleton-avatar {
		width: 32px;
		height: 32px;
		border-radius: var(--radius-full);
	}

	.cm-email-list__skeleton-body {
		display: flex;
		flex-direction: column;
		gap: 6px;
		padding-top: var(--space-1);
	}

	.cm-email-list__skeleton-bar {
		height: 12px;
		border-radius: var(--radius-xs);
	}

	.cm-email-list__skeleton-bar--top { width: 60%; }
	.cm-email-list__skeleton-bar--mid { width: 90%; }
	.cm-email-list__skeleton-bar--bot { width: 70%; }

	@keyframes cm-email-shimmer {
		0% { background-position: 200% 0; }
		100% { background-position: -200% 0; }
	}

	@media (prefers-reduced-motion: reduce) {
		.cm-email-list__progress,
		.cm-email-list__skeleton-dot,
		.cm-email-list__skeleton-avatar,
		.cm-email-list__skeleton-bar {
			animation: none;
		}
	}
</style>
