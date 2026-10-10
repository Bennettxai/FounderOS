<script lang="ts">
	import { AlertTriangle, RefreshCw, Info } from 'lucide-svelte';
	import { formatRelative, providerLabel } from './commsEmailUtils';
	import type { EmailAccount, EmailFolder, EmailProvider } from '$lib/api/comms';

	interface Props {
		accounts: EmailAccount[];
		currentFolder: EmailFolder;
		isSyncing: boolean;
		onReconnect: (provider: EmailProvider) => void;
		onSyncNow: () => void;
	}

	let { accounts, currentFolder, isSyncing, onReconnect, onSyncNow }: Props = $props();

	const STALE_THRESHOLD_MS = 60 * 60 * 1000;

	const reauth = $derived(
		accounts.filter((a) => a.status === 'reauth_required'),
	);

	const oldestSync = $derived.by(() => {
		const stamps = accounts
			.map((a) => (a.last_sync ? new Date(a.last_sync).getTime() : null))
			.filter((t): t is number => t !== null);
		if (!stamps.length) return null;
		return Math.min(...stamps);
	});

	const showStaleStrip = $derived(
		currentFolder === 'inbox' &&
			oldestSync !== null &&
			Date.now() - oldestSync > STALE_THRESHOLD_MS,
	);

	const oldestIso = $derived.by(() => {
		if (oldestSync === null) return null;
		return new Date(oldestSync).toISOString();
	});
</script>

{#if reauth.length > 0}
	{#each reauth as account (account.account_id || account.email)}
		<div class="cm-email-banner cm-email-banner--warning" role="status">
			<AlertTriangle size={16} aria-hidden="true" />
			<span class="cm-email-banner__text">
				{providerLabel(account.provider)} authorization expired for {account.email}
			</span>
			<button
				type="button"
				class="btn-compact btn-compact-ghost btn-compact-sm"
				onclick={() => onReconnect(account.provider)}
			>
				Reconnect
			</button>
		</div>
	{/each}
{:else if showStaleStrip}
	<div class="cm-email-banner cm-email-banner--info" role="status">
		<Info size={16} aria-hidden="true" />
		<span class="cm-email-banner__text">
			Last synced {formatRelative(oldestIso)}
		</span>
		<button
			type="button"
			class="btn-compact btn-compact-ghost btn-compact-sm"
			onclick={onSyncNow}
			disabled={isSyncing}
		>
			<RefreshCw size={14} class={isSyncing ? 'cm-spin' : ''} />
			Sync now
		</button>
	</div>
{/if}

<style>
	.cm-email-banner {
		display: flex;
		align-items: center;
		gap: var(--space-3);
		padding: var(--space-2) var(--space-4);
		border-bottom: 1px solid var(--dbd);
		font-size: var(--text-sm);
	}

	.cm-email-banner--warning {
		background: var(--bos-status-warning-bg);
		color: var(--bos-status-warning-text);
	}

	.cm-email-banner--info {
		background: var(--bos-status-info-bg);
		color: var(--bos-status-info-text);
	}

	.cm-email-banner__text {
		flex: 1;
	}

	:global(.cm-spin) {
		animation: cm-spin 1s linear infinite;
	}

	@keyframes cm-spin {
		from {
			transform: rotate(0deg);
		}
		to {
			transform: rotate(360deg);
		}
	}

	@media (prefers-reduced-motion: reduce) {
		:global(.cm-spin) {
			animation: none;
		}
	}
</style>
