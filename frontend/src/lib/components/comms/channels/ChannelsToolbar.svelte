<script lang="ts">
	import {
		Hash,
		Lock,
		Users,
		RefreshCw,
		MoreHorizontal,
		AlertCircle,
	} from 'lucide-svelte';
	import { dmDisplayName } from './commsChannelsUtils';
	import type { CommsChannel } from '$lib/api/comms';

	type SyncState = 'idle' | 'syncing' | 'error';

	interface Props {
		channel: CommsChannel | null;
		syncState: SyncState;
		syncError?: string | null;
		isScrolled?: boolean;
		onSync: () => void;
		onMore?: () => void;
	}

	let {
		channel,
		syncState,
		syncError = null,
		isScrolled = false,
		onSync,
		onMore,
	}: Props = $props();

	const isDm = $derived(channel?.is_dm ?? false);
	const isPrivate = $derived(channel?.is_private ?? false);
	const headerLabel = $derived(
		channel ? (channel.is_dm ? dmDisplayName(channel) : channel.name) : '',
	);
	const meta = $derived.by(() => {
		if (!channel) return '';
		if (channel.is_dm) return 'Direct message';
		const memberPart =
			channel.member_count > 0
				? `${channel.member_count} member${channel.member_count === 1 ? '' : 's'}`
				: '';
		return memberPart;
	});
</script>

<div class="cm-channels-toolbar" class:cm-channels-toolbar--elevated={isScrolled}>
	<div class="cm-channels-toolbar__title-block">
		<div class="cm-channels-toolbar__title-row">
			{#if channel}
				<span class="cm-channels-toolbar__icon" aria-hidden="true">
					{#if isDm}
						<span class="cm-channels-toolbar__dot"></span>
					{:else if isPrivate}
						<Lock size={14} />
					{:else}
						<Hash size={14} />
					{/if}
				</span>
			{/if}
			<h2 class="cm-channels-toolbar__title">
				{headerLabel || 'No channel selected'}
			</h2>
			{#if channel && !isDm && channel.member_count > 0}
				<span class="cm-channels-toolbar__sep" aria-hidden="true">·</span>
				<span class="cm-channels-toolbar__members">
					<Users size={13} />
					{channel.member_count} {channel.member_count === 1 ? 'member' : 'members'}
				</span>
			{:else if isDm}
				<span class="cm-channels-toolbar__sep" aria-hidden="true">·</span>
				<span class="cm-channels-toolbar__members">{meta}</span>
			{/if}
		</div>

		{#if channel && !isDm && (channel.topic || channel.description)}
			<p
				class="cm-channels-toolbar__topic"
				title={channel.description || channel.topic}
			>
				{channel.topic || channel.description}
			</p>
		{/if}
	</div>

	<div class="cm-channels-toolbar__actions">
		<button
			type="button"
			class="btn-compact btn-compact-ghost btn-compact-icon"
			aria-label={syncState === 'error' ? `Sync failed: ${syncError ?? 'unknown error'}` : 'Sync channel'}
			title={syncState === 'error' && syncError ? syncError : undefined}
			disabled={!channel || syncState === 'syncing'}
			onclick={onSync}
		>
			{#if syncState === 'error'}
				<AlertCircle size={16} class="cm-channels-toolbar__sync-error" />
			{:else}
				<RefreshCw size={16} class={syncState === 'syncing' ? 'cm-spin' : ''} />
			{/if}
		</button>
		{#if onMore}
			<button
				type="button"
				class="btn-compact btn-compact-ghost btn-compact-icon"
				aria-label="More options"
				disabled={!channel}
				onclick={onMore}
			>
				<MoreHorizontal size={16} />
			</button>
		{/if}
	</div>
</div>

<style>
	.cm-channels-toolbar {
		display: flex;
		align-items: flex-start;
		gap: var(--space-3);
		padding: var(--space-3) var(--space-5);
		background: var(--dbg);
		border-bottom: 1px solid var(--dbd);
		flex-shrink: 0;
		z-index: 1;
		transition: box-shadow 180ms ease;
	}

	.cm-channels-toolbar--elevated {
		box-shadow: 0 1px 2px rgba(0, 0, 0, 0.04), 0 4px 12px rgba(0, 0, 0, 0.04);
	}

	.cm-channels-toolbar__title-block {
		flex: 1;
		min-width: 0;
		display: flex;
		flex-direction: column;
		gap: 2px;
	}

	.cm-channels-toolbar__title-row {
		display: flex;
		align-items: center;
		gap: var(--space-2);
		min-width: 0;
	}

	.cm-channels-toolbar__icon {
		color: var(--dt3);
		display: inline-flex;
		flex-shrink: 0;
	}

	.cm-channels-toolbar__dot {
		width: 8px;
		height: 8px;
		border-radius: var(--radius-full);
		background: var(--bos-status-success);
		display: inline-block;
	}

	.cm-channels-toolbar__title {
		font-size: var(--text-base);
		font-weight: var(--font-semibold);
		color: var(--dt);
		margin: 0;
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}

	.cm-channels-toolbar__sep {
		color: var(--dt4);
		flex-shrink: 0;
	}

	.cm-channels-toolbar__members {
		display: inline-flex;
		align-items: center;
		gap: var(--space-1);
		font-size: var(--text-sm);
		color: var(--dt3);
		flex-shrink: 0;
	}

	.cm-channels-toolbar__topic {
		font-size: var(--text-xs);
		color: var(--dt3);
		margin: 0;
		max-width: 60vw;
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}

	.cm-channels-toolbar__actions {
		display: flex;
		align-items: center;
		gap: var(--space-1);
		flex-shrink: 0;
	}

	:global(.cm-channels-toolbar__sync-error) {
		color: var(--bos-status-error-text);
	}
</style>
