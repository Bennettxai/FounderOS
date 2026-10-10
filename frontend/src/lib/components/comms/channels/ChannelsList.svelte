<script lang="ts">
	import { Hash, Lock, Archive } from 'lucide-svelte';
	import {
		dmDisplayName,
		initials,
		formatLastActivity,
	} from './commsChannelsUtils';
	import type { CommsChannel } from '$lib/api/comms';

	interface Props {
		channels: CommsChannel[];
		selectedChannelId?: string | null;
		onSelect: (channel: CommsChannel) => void;
		// Render hint — DM rows use an initial-avatar prefix instead of a Hash/Lock icon.
		variant: 'channel' | 'dm';
		// When the sidebar is showing "All conversations" mode, surface a small
		// provider tag in each row so users can tell Slack from Teams at a glance.
		showProviderTag?: boolean;
	}

	let {
		channels,
		selectedChannelId,
		onSelect,
		variant,
		showProviderTag = false,
	}: Props = $props();

	function rowAriaLabel(channel: CommsChannel): string {
		if (variant === 'dm') {
			const unreadSuffix =
				channel.unread_count > 0
					? `, ${channel.unread_count} unread`
					: '';
			return `Direct message with ${dmDisplayName(channel)}${unreadSuffix}`;
		}
		const privacy = channel.is_private ? 'private' : 'public';
		const archivedSuffix = channel.is_archived ? ', archived' : '';
		const unreadSuffix =
			channel.unread_count > 0 ? `, ${channel.unread_count} unread` : '';
		return `${privacy} channel ${channel.name}${archivedSuffix}${unreadSuffix}`;
	}
</script>

<ul class="cm-channels-list" role="list">
	{#each channels as channel (channel.id)}
		{@const isActive = channel.id === selectedChannelId}
		{@const hasUnread = channel.unread_count > 0}
		{@const showCountChip = channel.unread_count >= 5}
		{@const showDot = hasUnread && !showCountChip}
		<li class="cm-channels-list__item">
			<button
				type="button"
				class="cm-channels-list__row"
				class:cm-channels-list__row--active={isActive}
				class:cm-channels-list__row--unread={hasUnread}
				class:cm-channels-list__row--archived={channel.is_archived}
				onclick={() => onSelect(channel)}
				aria-label={rowAriaLabel(channel)}
				aria-current={isActive ? 'true' : undefined}
			>
				<span class="cm-channels-list__icon" aria-hidden="true">
					{#if variant === 'dm'}
						<span class="cm-channels-list__dm-avatar-wrap">
							<span class="cm-channels-list__dm-avatar">
								{initials(dmDisplayName(channel))}
							</span>
							{#if channel.presence && channel.presence !== 'unknown'}
								<span
									class="cm-channels-list__presence cm-channels-list__presence--{channel.presence}"
								></span>
							{/if}
						</span>
					{:else if channel.is_archived}
						<Archive size={13} />
					{:else if channel.is_private}
						<Lock size={13} />
					{:else}
						<Hash size={13} />
					{/if}
				</span>

				<span class="cm-channels-list__name">
					{variant === 'dm' ? dmDisplayName(channel) : channel.name}
				</span>

				{#if showProviderTag}
					<span class="cm-channels-list__provider-tag" aria-hidden="true">
						{channel.provider === 'slack' ? 'S' : channel.provider === 'whatsapp' ? 'W' : 'T'}
					</span>
				{/if}

				{#if showDot}
					<span class="cm-channels-list__dot" aria-hidden="true"></span>
				{:else if showCountChip}
					<span class="cm-channels-list__chip">{channel.unread_count}</span>
				{:else if !hasUnread && channel.last_message_at}
					<span class="cm-channels-list__activity" aria-hidden="true">
						{formatLastActivity(channel.last_message_at)}
					</span>
				{/if}
			</button>
		</li>
	{/each}
</ul>

<style>
	.cm-channels-list {
		list-style: none;
		margin: 0;
		padding: var(--space-1) 0 0;
		display: flex;
		flex-direction: column;
		gap: 1px;
	}

	.cm-channels-list__item {
		list-style: none;
	}

	.cm-channels-list__row {
		display: flex;
		align-items: center;
		gap: var(--space-2);
		width: 100%;
		padding: var(--space-1) var(--space-4);
		background: none;
		border: none;
		font-family: inherit;
		font-size: var(--text-sm);
		font-weight: var(--font-normal);
		color: var(--dt2);
		text-align: left;
		cursor: pointer;
		transition: background var(--bos-transition-fast), color var(--bos-transition-fast);
	}

	.cm-channels-list__row:hover {
		background: var(--dbg3);
	}

	.cm-channels-list__row--active {
		background: var(--dbg3);
		color: var(--dt);
		font-weight: var(--font-semibold);
	}

	.cm-channels-list__row--unread {
		color: var(--dt);
		font-weight: var(--font-semibold);
	}

	.cm-channels-list__row--archived {
		color: var(--dt4);
	}

	.cm-channels-list__icon {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		color: var(--dt3);
		flex-shrink: 0;
	}

	.cm-channels-list__row--active .cm-channels-list__icon,
	.cm-channels-list__row--unread .cm-channels-list__icon {
		color: var(--dt);
	}

	.cm-channels-list__dm-avatar-wrap {
		position: relative;
		display: inline-flex;
		align-items: center;
		justify-content: center;
	}

	.cm-channels-list__dm-avatar {
		width: 18px;
		height: 18px;
		border-radius: var(--radius-full);
		background: var(--bos-avatar-default);
		color: var(--bos-avatar-default-text);
		font-size: var(--text-xs);
		font-weight: var(--font-bold);
		display: inline-flex;
		align-items: center;
		justify-content: center;
	}

	.cm-channels-list__presence {
		position: absolute;
		bottom: -1px;
		right: -1px;
		width: 8px;
		height: 8px;
		border-radius: var(--radius-full);
		/* Border in the sidebar background lifts the dot off the avatar visually
		   so it reads as a separate element in both themes. */
		border: 2px solid var(--dbg2);
		box-sizing: content-box;
	}

	.cm-channels-list__presence--online {
		background: var(--bos-status-success);
	}

	.cm-channels-list__presence--away {
		background: var(--bos-status-warning);
	}

	.cm-channels-list__presence--offline {
		background: transparent;
		border-color: var(--dt4);
		border-width: 1.5px;
	}

	.cm-channels-list__name {
		flex: 1;
		min-width: 0;
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}

	.cm-channels-list__provider-tag {
		font-size: 9px;
		font-weight: var(--font-bold);
		color: var(--dt4);
		padding: 0 var(--space-1);
		border: 1px solid var(--dbd);
		border-radius: var(--radius-xs);
		flex-shrink: 0;
		line-height: 1.4;
	}

	.cm-channels-list__dot {
		width: 7px;
		height: 7px;
		border-radius: var(--radius-full);
		background: var(--bos-accent-blue);
		flex-shrink: 0;
	}

	.cm-channels-list__chip {
		min-width: 20px;
		padding: 1px 6px;
		background: var(--bos-accent-blue);
		color: var(--bos-surface-on-color);
		border-radius: var(--radius-full);
		font-size: 0.7rem;
		font-weight: 600;
		text-align: center;
		line-height: 1.5;
		font-feature-settings: 'tnum' 1;
		font-variant-numeric: tabular-nums;
		box-shadow: 0 1px 3px rgba(var(--bos-accent-blue-rgb), 0.3);
	}

	.cm-channels-list__activity {
		font-size: var(--text-xs);
		color: var(--dt4);
		flex-shrink: 0;
	}

	@media (prefers-reduced-motion: reduce) {
		.cm-channels-list__row {
			transition: none;
		}
	}
</style>
