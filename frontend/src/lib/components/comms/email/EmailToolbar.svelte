<script lang="ts">
	import { Search, RefreshCw, MoreHorizontal, AlertCircle } from 'lucide-svelte';
	import type { EmailFolder } from '$lib/api/comms';

	type SyncState = 'idle' | 'syncing' | 'error';

	interface Props {
		folder: EmailFolder;
		folderLabel: string;
		unreadCount: number;
		searchQuery: string;
		searchScope?: 'folder' | 'all';
		syncState: SyncState;
		syncError?: string | null;
		onSearchChange: (value: string) => void;
		onSync: () => void;
		onMore?: () => void;
	}

	let {
		folder,
		folderLabel,
		unreadCount,
		searchQuery,
		searchScope = 'folder',
		syncState,
		syncError = null,
		onSearchChange,
		onSync,
		onMore,
	}: Props = $props();

	const placeholder = $derived(
		searchScope === 'folder' ? 'Search this folder…' : 'Search all email…',
	);

	function handleInput(event: Event) {
		const target = event.target as HTMLInputElement;
		onSearchChange(target.value);
	}
</script>

<div class="cm-email-toolbar">
	<div class="cm-email-toolbar__title">
		<span class="cm-email-toolbar__folder">{folderLabel}</span>
		{#if unreadCount > 0 && folder !== 'drafts'}
			<span class="cm-email-toolbar__sep" aria-hidden="true">·</span>
			<span class="cm-email-toolbar__unread">{unreadCount} unread</span>
		{/if}
	</div>

	<div class="cm-email-toolbar__search">
		<Search size={14} aria-hidden="true" />
		<input
			type="search"
			class="bos-input cm-email-toolbar__search-input"
			value={searchQuery}
			placeholder={placeholder}
			aria-label={placeholder}
			oninput={handleInput}
		/>
	</div>

	<div class="cm-email-toolbar__actions">
		<button
			type="button"
			class="btn-compact btn-compact-ghost btn-compact-icon"
			aria-label={syncState === 'error' ? `Sync failed: ${syncError ?? 'unknown error'}` : 'Sync mailbox'}
			title={syncState === 'error' && syncError ? syncError : undefined}
			disabled={syncState === 'syncing'}
			onclick={onSync}
		>
			{#if syncState === 'error'}
				<AlertCircle size={16} class="cm-email-toolbar__sync-error" />
			{:else}
				<RefreshCw size={16} class={syncState === 'syncing' ? 'cm-spin' : ''} />
			{/if}
		</button>
		{#if onMore}
			<button
				type="button"
				class="btn-compact btn-compact-ghost btn-compact-icon"
				aria-label="More options"
				onclick={onMore}
			>
				<MoreHorizontal size={16} />
			</button>
		{/if}
	</div>
</div>

<style>
	.cm-email-toolbar {
		display: flex;
		align-items: center;
		gap: var(--space-3);
		padding: var(--space-3) var(--space-4);
		background: var(--dbg);
		border-bottom: 1px solid var(--dbd);
		flex-shrink: 0;
	}

	.cm-email-toolbar__title {
		display: flex;
		align-items: baseline;
		gap: var(--space-2);
		flex-shrink: 0;
	}

	.cm-email-toolbar__folder {
		font-size: var(--text-base);
		font-weight: var(--font-semibold);
		color: var(--dt);
	}

	.cm-email-toolbar__sep {
		color: var(--dt4);
	}

	.cm-email-toolbar__unread {
		font-size: var(--text-sm);
		color: var(--dt3);
	}

	.cm-email-toolbar__search {
		position: relative;
		flex: 1;
		max-width: 280px;
		display: flex;
		align-items: center;
	}

	.cm-email-toolbar__search :global(svg) {
		position: absolute;
		left: var(--space-3);
		color: var(--dt4);
		pointer-events: none;
	}

	.cm-email-toolbar__search-input {
		width: 100%;
		padding-left: calc(var(--space-3) + 14px + var(--space-2));
	}

	.cm-email-toolbar__actions {
		display: flex;
		align-items: center;
		gap: var(--space-1);
		margin-left: auto;
	}

	:global(.cm-email-toolbar__sync-error) {
		color: var(--bos-status-error-text);
	}
</style>
