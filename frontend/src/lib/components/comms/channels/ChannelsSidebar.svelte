<script lang="ts">
	import { onMount } from 'svelte';
	import {
		ChevronDown,
		ChevronRight,
		Search,
		Plus,
		AtSign,
		Reply,
		Hash,
		MessageSquare,
		RefreshCw,
	} from 'lucide-svelte';
	import ChannelsList from './ChannelsList.svelte';
	import {
		filterByWorkspace,
		filterChannelsByQuery,
		partitionByType,
		providerLabel,
		workspaceLabel,
		workspaceStatusColor,
		type WorkspaceScope,
	} from './commsChannelsUtils';
	import type {
		ChannelProvider,
		CommsChannel,
		CommsWorkspace,
	} from '$lib/api/comms';

	interface Props {
		workspaces: CommsWorkspace[];
		channels: CommsChannel[];
		workspaceScope: WorkspaceScope;
		selectedChannelId: string | null;
		// Activity & Threads section badges (mentions count, threads-with-unread count).
		mentionsCount?: number;
		threadsCount?: number;
		onSelectChannel: (channel: CommsChannel) => void;
		onChangeWorkspaceScope: (scope: WorkspaceScope) => void;
		onAddWorkspace?: (provider: ChannelProvider) => void;
		// Activity / Threads sections — Wave 2 ships them dimmed if no data shape ready.
		onOpenActivity?: () => void;
		onOpenThreads?: () => void;
		// Wave 3 — wired to the page's sync handler so the "no channels synced
		// yet" empty state can offer a one-click recovery.
		onSync?: () => void;
		isSyncing?: boolean;
	}

	let {
		workspaces,
		channels,
		workspaceScope,
		selectedChannelId,
		mentionsCount = 0,
		threadsCount = 0,
		onSelectChannel,
		onChangeWorkspaceScope,
		onAddWorkspace,
		onOpenActivity,
		onOpenThreads,
		onSync,
		isSyncing = false,
	}: Props = $props();

	const SECTIONS_KEY = 'comms.channels.sidebarSections';
	type SectionKey = 'workspace' | 'channels' | 'dms' | 'activity' | 'threads';
	let collapsed = $state<Record<SectionKey, boolean>>({
		workspace: false,
		channels: false,
		dms: false,
		activity: false,
		threads: false,
	});

	let query = $state('');

	onMount(() => {
		try {
			const raw = localStorage.getItem(SECTIONS_KEY);
			if (raw) collapsed = { ...collapsed, ...JSON.parse(raw) };
		} catch {
			// localStorage may be unavailable; defaults are fine.
		}
	});

	function toggleSection(key: SectionKey) {
		collapsed = { ...collapsed, [key]: !collapsed[key] };
		try {
			localStorage.setItem(SECTIONS_KEY, JSON.stringify(collapsed));
		} catch {
			// Best-effort persistence.
		}
	}

	function workspaceStateLabel(workspace: CommsWorkspace): string {
		if (workspace.sync_status === 'history_synced') {
			return workspace.routing_status === 'configured'
				? 'Synced and routed'
				: 'History synced, not routed';
		}
		if (workspace.status === 'error') return 'Connection error';
		if (workspace.status === 'stale') return 'Sync is stale';
		return '';
	}

	const showWorkspaceSwitcher = $derived(
		workspaces.length >= 2 || workspaces.some((workspace) => !!workspace.sync_status),
	);

	const scopedChannels = $derived(
		filterByWorkspace(channels, workspaceScope),
	);

	const filtered = $derived(filterChannelsByQuery(scopedChannels, query));

	const partition = $derived(partitionByType(filtered));

	const showProviderTag = $derived(
		workspaceScope === 'all' && workspaces.length >= 2,
	);

	const missingProviders = $derived.by<ChannelProvider[]>(() => {
		const present = new Set(workspaces.map((w) => w.provider));
		return (['slack', 'teams'] as ChannelProvider[]).filter(
			(p) => !present.has(p),
		);
	});
</script>

<aside class="cm-channels-sidebar">
	<div class="cm-channels-sidebar__search">
		<span class="cm-channels-sidebar__search-icon" aria-hidden="true">
			<Search size={12} />
		</span>
		<input
			type="text"
			class="cm-channels-sidebar__search-input"
			placeholder="Filter channels & DMs…"
			bind:value={query}
			aria-label="Filter channels and direct messages"
		/>
	</div>

	{#if showWorkspaceSwitcher}
		<section class="cm-channels-sidebar__section">
			<button
				type="button"
				class="cm-channels-sidebar__section-header"
				onclick={() => toggleSection('workspace')}
				aria-expanded={!collapsed.workspace}
			>
				{#if collapsed.workspace}
					<ChevronRight size={12} />
				{:else}
					<ChevronDown size={12} />
				{/if}
				<span>Workspace</span>
			</button>
			{#if !collapsed.workspace}
				<ul class="cm-channels-sidebar__list" role="list">
					<li>
						<button
							type="button"
							class="cm-channels-sidebar__row"
							class:cm-channels-sidebar__row--active={workspaceScope === 'all'}
							onclick={() => onChangeWorkspaceScope('all')}
						>
							<span class="cm-channels-sidebar__radio"></span>
							<span class="cm-channels-sidebar__row-label">
								All conversations
							</span>
						</button>
					</li>
					{#each workspaces as workspace (`${workspace.provider}:${workspace.workspace_id}`)}
						<li>
							<button
								type="button"
								class="cm-channels-sidebar__row"
								class:cm-channels-sidebar__row--active={workspaceScope === workspace.workspace_id}
								onclick={() => onChangeWorkspaceScope(workspace.workspace_id)}
							>
								<span class="cm-channels-sidebar__radio"></span>
								<span class="cm-channels-sidebar__row-label">
									<span>{workspaceLabel(workspace)}</span>
									{#if workspaceStateLabel(workspace)}
										<small>{workspaceStateLabel(workspace)}</small>
									{/if}
								</span>
								<span
									class="cm-channels-sidebar__status-dot"
									style:background-color={workspaceStatusColor(workspace.status)}
									aria-hidden="true"
								></span>
							</button>
						</li>
					{/each}
				</ul>
			{/if}
		</section>
	{/if}

	<section class="cm-channels-sidebar__section">
		<button
			type="button"
			class="cm-channels-sidebar__section-header"
			onclick={() => toggleSection('channels')}
			aria-expanded={!collapsed.channels}
		>
			{#if collapsed.channels}
				<ChevronRight size={12} />
			{:else}
				<ChevronDown size={12} />
			{/if}
			<span>Channels</span>
			{#if partition.channels.length > 0}
				<span class="cm-channels-sidebar__count">
					{partition.channels.length}
				</span>
			{/if}
		</button>
		{#if !collapsed.channels}
			{#if partition.channels.length === 0}
				{#if query}
					<div class="cm-channels-sidebar__empty">
						<Search size={14} aria-hidden="true" />
						<p class="cm-channels-sidebar__empty-text">
							No channels match “{query}”
						</p>
					</div>
				{:else if onSync}
					<div class="cm-channels-sidebar__empty">
						<Hash size={14} aria-hidden="true" />
						<p class="cm-channels-sidebar__empty-text">
							No channels synced yet
						</p>
						<button
							type="button"
							class="cm-channels-sidebar__empty-action"
							onclick={() => onSync?.()}
							disabled={isSyncing}
						>
							<RefreshCw size={11} class={isSyncing ? 'cm-spin' : ''} />
							<span>{isSyncing ? 'Syncing…' : 'Sync now'}</span>
						</button>
					</div>
				{:else}
					<div class="cm-channels-sidebar__empty">
						<Hash size={14} aria-hidden="true" />
						<p class="cm-channels-sidebar__empty-text">No channels</p>
					</div>
				{/if}
			{:else}
				<ChannelsList
					channels={partition.channels}
					{selectedChannelId}
					onSelect={onSelectChannel}
					variant="channel"
					{showProviderTag}
				/>
			{/if}
		{/if}
	</section>

	<section class="cm-channels-sidebar__section">
		<button
			type="button"
			class="cm-channels-sidebar__section-header"
			onclick={() => toggleSection('dms')}
			aria-expanded={!collapsed.dms}
		>
			{#if collapsed.dms}
				<ChevronRight size={12} />
			{:else}
				<ChevronDown size={12} />
			{/if}
			<span>Direct Messages</span>
			{#if partition.dms.length > 0}
				<span class="cm-channels-sidebar__count">{partition.dms.length}</span>
			{/if}
		</button>
		{#if !collapsed.dms}
			{#if partition.dms.length === 0}
				{#if query}
					<div class="cm-channels-sidebar__empty">
						<Search size={14} aria-hidden="true" />
						<p class="cm-channels-sidebar__empty-text">
							No DMs match “{query}”
						</p>
					</div>
				{:else}
					<div class="cm-channels-sidebar__empty">
						<MessageSquare size={14} aria-hidden="true" />
						<p class="cm-channels-sidebar__empty-text">
							No direct messages
						</p>
					</div>
				{/if}
			{:else}
				<ChannelsList
					channels={partition.dms}
					{selectedChannelId}
					onSelect={onSelectChannel}
					variant="dm"
					{showProviderTag}
				/>
			{/if}
		{/if}
	</section>

	<section class="cm-channels-sidebar__section">
		<button
			type="button"
			class="cm-channels-sidebar__section-header cm-channels-sidebar__section-header--clickable"
			onclick={() => onOpenActivity?.()}
			disabled={!onOpenActivity}
		>
			<AtSign size={12} />
			<span>Activity</span>
			{#if mentionsCount > 0}
				<span class="cm-channels-sidebar__count">{mentionsCount}</span>
			{/if}
		</button>
	</section>

	<section class="cm-channels-sidebar__section">
		<button
			type="button"
			class="cm-channels-sidebar__section-header cm-channels-sidebar__section-header--clickable"
			onclick={() => onOpenThreads?.()}
			disabled={!onOpenThreads}
		>
			<Reply size={12} />
			<span>Threads</span>
			{#if threadsCount > 0}
				<span class="cm-channels-sidebar__count">{threadsCount}</span>
			{/if}
		</button>
	</section>

	{#if onAddWorkspace}
		<div class="cm-channels-sidebar__footer">
			{#each missingProviders as provider (provider)}
				<button
					type="button"
					class="cm-channels-sidebar__add"
					onclick={() => onAddWorkspace?.(provider)}
				>
					<Plus size={12} />
					<span>Add {providerLabel(provider)}</span>
				</button>
			{/each}
		</div>
	{/if}
</aside>

<style>
	.cm-channels-sidebar {
		width: 240px;
		flex-shrink: 0;
		display: flex;
		flex-direction: column;
		background: var(--dbg2);
		border-right: 1px solid var(--dbd);
		min-height: 0;
		overflow-y: auto;
	}

	.cm-channels-sidebar__search {
		display: flex;
		align-items: center;
		gap: var(--space-2);
		padding: var(--space-2) var(--space-3);
		border-bottom: 1px solid var(--dbd);
		background: var(--dbg2);
		position: sticky;
		top: 0;
		z-index: var(--bos-z-sticky);
	}

	.cm-channels-sidebar__search-icon {
		color: var(--dt3);
		flex-shrink: 0;
		display: inline-flex;
	}

	.cm-channels-sidebar__search-input {
		flex: 1;
		min-width: 0;
		border: none;
		background: transparent;
		font-family: inherit;
		font-size: var(--text-xs);
		color: var(--dt);
		outline: none;
		padding: var(--space-1) 0;
	}

	.cm-channels-sidebar__search-input::placeholder {
		color: var(--dt4);
	}

	.cm-channels-sidebar__section {
		padding: var(--space-3) 0;
		border-bottom: 1px solid var(--dbd);
	}

	.cm-channels-sidebar__section:last-of-type {
		border-bottom: none;
	}

	.cm-channels-sidebar__section-header {
		display: flex;
		align-items: center;
		gap: var(--space-2);
		width: 100%;
		padding: 0 var(--space-4);
		background: none;
		border: none;
		font-family: inherit;
		font-size: var(--text-xs);
		font-weight: var(--font-semibold);
		text-transform: uppercase;
		letter-spacing: 0.04em;
		color: var(--dt3);
		cursor: pointer;
		text-align: left;
	}

	.cm-channels-sidebar__section-header:hover {
		color: var(--dt2);
	}

	.cm-channels-sidebar__section-header:disabled {
		cursor: default;
		color: var(--dt4);
	}

	.cm-channels-sidebar__section-header--clickable {
		text-transform: none;
		letter-spacing: 0;
		font-size: var(--text-sm);
		font-weight: var(--font-medium);
		color: var(--dt2);
	}

	.cm-channels-sidebar__count {
		margin-left: auto;
		font-size: var(--text-xs);
		font-weight: var(--font-normal);
		color: var(--dt4);
	}

	.cm-channels-sidebar__list {
		list-style: none;
		padding: var(--space-1) 0 0;
		margin: 0;
		display: flex;
		flex-direction: column;
		gap: 1px;
	}

	.cm-channels-sidebar__row {
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
		cursor: pointer;
		text-align: left;
		transition: background var(--bos-transition-fast), color var(--bos-transition-fast);
	}

	.cm-channels-sidebar__row:hover {
		background: var(--dbg3);
	}

	.cm-channels-sidebar__row--active {
		background: var(--bos-nav-active-bg);
		color: var(--bos-nav-active);
		font-weight: var(--font-semibold);
	}

	.cm-channels-sidebar__row-label {
		flex: 1;
		min-width: 0;
		display: flex;
		flex-direction: column;
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}

	.cm-channels-sidebar__row-label > span,
	.cm-channels-sidebar__row-label > small {
		overflow: hidden;
		text-overflow: ellipsis;
	}

	.cm-channels-sidebar__row-label > small {
		color: var(--dt4);
		font-size: 10px;
		font-weight: var(--font-normal);
		line-height: 1.25;
	}

	.cm-channels-sidebar__radio {
		width: 10px;
		height: 10px;
		border-radius: var(--radius-full);
		border: 1.5px solid var(--dt4);
		flex-shrink: 0;
		transition: background var(--bos-transition-fast), border-color var(--bos-transition-fast);
	}

	.cm-channels-sidebar__row--active .cm-channels-sidebar__radio {
		background: var(--bos-accent-blue);
		border-color: var(--bos-accent-blue);
	}

	.cm-channels-sidebar__status-dot {
		width: 8px;
		height: 8px;
		border-radius: var(--radius-full);
		flex-shrink: 0;
	}

	.cm-channels-sidebar__empty {
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: var(--space-1);
		padding: var(--space-3) var(--space-4);
		text-align: center;
		color: var(--dt3);
	}

	.cm-channels-sidebar__empty :global(svg) {
		color: var(--dt4);
	}

	.cm-channels-sidebar__empty-text {
		margin: 0;
		font-size: var(--text-xs);
		color: var(--dt3);
		line-height: 1.4;
	}

	.cm-channels-sidebar__empty-action {
		display: inline-flex;
		align-items: center;
		gap: var(--space-1);
		padding: var(--space-1) var(--space-2);
		background: none;
		border: none;
		font-family: inherit;
		font-size: var(--text-xs);
		color: var(--bos-accent-blue);
		cursor: pointer;
		border-radius: var(--radius-sm);
		transition: background var(--bos-transition-fast);
	}

	.cm-channels-sidebar__empty-action:hover:not(:disabled) {
		background: var(--dbg3);
	}

	.cm-channels-sidebar__empty-action:disabled {
		color: var(--dt4);
		cursor: default;
	}

	.cm-channels-sidebar__footer {
		margin-top: auto;
		padding: var(--space-3) var(--space-4);
		border-top: 1px solid var(--dbd);
		background: var(--dbg2);
		display: flex;
		flex-direction: column;
		gap: var(--space-1);
	}

	.cm-channels-sidebar__add {
		display: inline-flex;
		align-items: center;
		gap: var(--space-2);
		padding: var(--space-1) 0;
		background: none;
		border: none;
		font-family: inherit;
		font-size: var(--text-xs);
		color: var(--bos-accent-blue);
		cursor: pointer;
		text-align: left;
	}

	.cm-channels-sidebar__add:hover {
		text-decoration: underline;
	}

	@media (prefers-reduced-motion: reduce) {
		.cm-channels-sidebar__row,
		.cm-channels-sidebar__radio {
			transition: none;
		}
	}
</style>
