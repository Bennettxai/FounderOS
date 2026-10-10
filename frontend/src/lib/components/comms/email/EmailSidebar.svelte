<script lang="ts">
	import { onMount } from 'svelte';
	import {
		Inbox,
		Send,
		FileEdit,
		Star,
		Archive,
		Trash2,
		Plus,
		ChevronDown,
		ChevronRight,
	} from 'lucide-svelte';
	import PillButton from '$lib/components/ui/PillButton.svelte';
	import { providerLabel } from './commsEmailUtils';
	import type {
		EmailAccount,
		EmailFolder,
		EmailProvider,
	} from '$lib/api/comms';

	export type ProviderScope = 'all' | EmailProvider;

	interface FolderItem {
		id: EmailFolder;
		label: string;
		count?: number;
	}

	interface Props {
		providerScope: ProviderScope;
		accounts: EmailAccount[];
		folders: FolderItem[];
		currentFolder: EmailFolder;
		onChangeProviderScope: (scope: ProviderScope) => void;
		onChangeFolder: (folder: EmailFolder) => void;
		onCompose: () => void;
		onAddAccount?: (provider: EmailProvider) => void;
	}

	let {
		providerScope,
		accounts,
		folders,
		currentFolder,
		onChangeProviderScope,
		onChangeFolder,
		onCompose,
		onAddAccount,
	}: Props = $props();

	const SECTIONS_KEY = 'comms.email.sidebarSections';
	type SectionKey = 'view' | 'folders' | 'accounts';
	let collapsed = $state<Record<SectionKey, boolean>>({
		view: false,
		folders: false,
		accounts: false,
	});

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

	const ICONS = {
		inbox: Inbox,
		sent: Send,
		drafts: FileEdit,
		starred: Star,
		archive: Archive,
		trash: Trash2,
	} as const;

	const connectedProviders = $derived(
		new Set(accounts.map((a) => a.provider)),
	);
	const showProviderSwitcher = $derived(connectedProviders.size >= 2);
	const missingProviders = $derived(
		(['gmail', 'outlook'] as EmailProvider[]).filter(
			(p) => !connectedProviders.has(p),
		),
	);

	function accountStatusColor(status?: EmailAccount['status']): string {
		if (status === 'reauth_required' || status === 'error') {
			return 'var(--bos-status-error)';
		}
		if (status === 'stale') return 'var(--bos-status-warning)';
		return 'var(--bos-status-success)';
	}
</script>

<aside class="cm-email-sidebar">
	{#if showProviderSwitcher}
		<section class="cm-email-sidebar__section">
			<button
				type="button"
				class="cm-email-sidebar__section-header"
				onclick={() => toggleSection('view')}
				aria-expanded={!collapsed.view}
			>
				{#if collapsed.view}
					<ChevronRight size={12} />
				{:else}
					<ChevronDown size={12} />
				{/if}
				<span>View</span>
			</button>
			{#if !collapsed.view}
				<ul class="cm-email-sidebar__list">
					<li>
						<button
							type="button"
							class="cm-email-sidebar__row"
							class:cm-email-sidebar__row--active={providerScope === 'all'}
							onclick={() => onChangeProviderScope('all')}
						>
							<span class="cm-email-sidebar__radio"></span>
							<span class="cm-email-sidebar__row-label">All inboxes</span>
						</button>
					</li>
					{#each Array.from(connectedProviders) as provider (provider)}
						<li>
							<button
								type="button"
								class="cm-email-sidebar__row"
								class:cm-email-sidebar__row--active={providerScope === provider}
								onclick={() => onChangeProviderScope(provider)}
							>
								<span class="cm-email-sidebar__radio"></span>
								<span class="cm-email-sidebar__row-label">{providerLabel(provider)}</span>
							</button>
						</li>
					{/each}
				</ul>
			{/if}
		</section>
	{/if}

	<section class="cm-email-sidebar__section">
		<button
			type="button"
			class="cm-email-sidebar__section-header"
			onclick={() => toggleSection('folders')}
			aria-expanded={!collapsed.folders}
		>
			{#if collapsed.folders}
				<ChevronRight size={12} />
			{:else}
				<ChevronDown size={12} />
			{/if}
			<span>Folders</span>
		</button>
		{#if !collapsed.folders}
			<ul class="cm-email-sidebar__list">
				{#each folders as folder (folder.id)}
					{@const Icon = ICONS[folder.id]}
					<li>
						<button
							type="button"
							class="cm-email-sidebar__row"
							class:cm-email-sidebar__row--active={currentFolder === folder.id}
							onclick={() => onChangeFolder(folder.id)}
							aria-label="{folder.label} folder"
						>
							<span class="cm-email-sidebar__row-icon">
								<Icon size={16} strokeWidth={1.75} />
							</span>
							<span class="cm-email-sidebar__row-label">{folder.label}</span>
							{#if folder.count && folder.count > 0}
								<span class="cm-email-sidebar__badge">{folder.count}</span>
							{/if}
						</button>
					</li>
				{/each}
			</ul>
		{/if}
	</section>

	<section class="cm-email-sidebar__section">
		<button
			type="button"
			class="cm-email-sidebar__section-header"
			onclick={() => toggleSection('accounts')}
			aria-expanded={!collapsed.accounts}
		>
			{#if collapsed.accounts}
				<ChevronRight size={12} />
			{:else}
				<ChevronDown size={12} />
			{/if}
			<span>Accounts</span>
		</button>
		{#if !collapsed.accounts}
			<ul class="cm-email-sidebar__list">
				{#each accounts as account (account.account_id || account.email)}
					<li class="cm-email-sidebar__account">
						<span
							class="cm-email-sidebar__account-dot"
							style:background-color={accountStatusColor(account.status)}
							aria-hidden="true"
						></span>
						<span class="cm-email-sidebar__account-email">{account.email}</span>
					</li>
				{/each}
				{#if onAddAccount}
					{#each missingProviders as provider (provider)}
						<li>
							<button
								type="button"
								class="cm-email-sidebar__add"
								onclick={() => onAddAccount?.(provider)}
							>
								<Plus size={12} />
								<span>Add {providerLabel(provider)}</span>
							</button>
						</li>
					{/each}
				{/if}
			</ul>
		{/if}
	</section>

	<div class="cm-email-sidebar__footer">
		<PillButton variant="cta" size="md" block onclick={onCompose} aria-label="Compose new email">
			<Plus size={16} />
			<span style="margin-left: var(--space-2);">Compose</span>
		</PillButton>
	</div>
</aside>

<style>
	.cm-email-sidebar {
		width: 220px;
		flex-shrink: 0;
		display: flex;
		flex-direction: column;
		background: var(--dbg2);
		border-right: 1px solid var(--dbd);
		min-height: 0;
	}

	.cm-email-sidebar__section {
		padding: var(--space-3) 0;
		border-bottom: 1px solid var(--dbd);
	}

	.cm-email-sidebar__section:last-of-type {
		border-bottom: none;
	}

	.cm-email-sidebar__section-header {
		display: flex;
		align-items: center;
		gap: var(--space-1);
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
	}

	.cm-email-sidebar__section-header:hover {
		color: var(--dt2);
	}

	.cm-email-sidebar__list {
		list-style: none;
		padding: var(--space-2) 0 0;
		margin: 0;
		display: flex;
		flex-direction: column;
		gap: 2px;
	}

	.cm-email-sidebar__row {
		display: flex;
		align-items: center;
		gap: var(--space-2);
		width: 100%;
		padding: var(--space-2) var(--space-4);
		background: none;
		border: none;
		font-family: inherit;
		font-size: var(--text-sm);
		font-weight: var(--font-medium);
		color: var(--dt2);
		cursor: pointer;
		text-align: left;
		transition: background var(--bos-transition-fast), color var(--bos-transition-fast);
	}

	.cm-email-sidebar__row:hover {
		background: var(--dbg3);
	}

	.cm-email-sidebar__row--active {
		background: var(--bos-nav-active-bg);
		color: var(--bos-nav-active);
		font-weight: var(--font-semibold);
	}

	.cm-email-sidebar__row-icon {
		display: inline-flex;
		align-items: center;
		flex-shrink: 0;
		color: inherit;
	}

	.cm-email-sidebar__row-label {
		flex: 1;
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}

	.cm-email-sidebar__radio {
		width: 10px;
		height: 10px;
		border-radius: var(--radius-full);
		border: 1.5px solid var(--dt4);
		flex-shrink: 0;
		transition: background var(--bos-transition-fast), border-color var(--bos-transition-fast);
	}

	.cm-email-sidebar__row--active .cm-email-sidebar__radio {
		background: var(--bos-accent-blue);
		border-color: var(--bos-accent-blue);
	}

	.cm-email-sidebar__badge {
		min-width: 20px;
		padding: 1px var(--space-2);
		background: var(--bos-accent-blue);
		color: var(--bos-surface-on-color);
		border-radius: var(--radius-full);
		font-size: var(--text-xs);
		font-weight: var(--font-bold);
		text-align: center;
		line-height: 1.4;
	}

	.cm-email-sidebar__account {
		display: flex;
		align-items: center;
		gap: var(--space-2);
		padding: var(--space-2) var(--space-4);
		font-size: var(--text-xs);
		color: var(--dt3);
	}

	.cm-email-sidebar__account-dot {
		width: 8px;
		height: 8px;
		border-radius: var(--radius-full);
		flex-shrink: 0;
	}

	.cm-email-sidebar__account-email {
		flex: 1;
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}

	.cm-email-sidebar__add {
		display: flex;
		align-items: center;
		gap: var(--space-2);
		width: 100%;
		padding: var(--space-2) var(--space-4);
		background: none;
		border: none;
		font-family: inherit;
		font-size: var(--text-xs);
		color: var(--bos-accent-blue);
		cursor: pointer;
		text-align: left;
	}

	.cm-email-sidebar__add:hover {
		text-decoration: underline;
	}

	.cm-email-sidebar__footer {
		margin-top: auto;
		padding: var(--space-3);
		border-top: 1px solid var(--dbd);
		background: var(--dbg2);
	}

	@media (prefers-reduced-motion: reduce) {
		.cm-email-sidebar__row,
		.cm-email-sidebar__radio {
			transition: none;
		}
	}
</style>
