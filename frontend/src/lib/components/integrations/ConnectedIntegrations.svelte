<script lang="ts">
	import type { UserIntegration } from '$lib/api/integrations';

	interface Props {
		integrations: UserIntegration[];
		syncingId: string | null;
		onsync: (integrationId: string) => void;
		ondisconnect: (integrationId: string) => void;
		onbrowse: () => void;
	}

	let {
		integrations,
		syncingId,
		onsync,
		ondisconnect,
		onbrowse
	}: Props = $props();

	function getStatusBadgeClass(status: string) {
		switch (status) {
			case 'connected': return 'ih-badge--connected';
			case 'available': return 'ih-badge--available';
			case 'coming_soon': return 'ih-badge--coming-soon';
			case 'error': return 'ih-badge--error';
			default: return 'ih-badge--default';
		}
	}
</script>

{#if integrations.length === 0}
	<div class="ih-empty">
		<svg
			class="ih-empty__icon"
			fill="none"
			viewBox="0 0 24 24"
			stroke="currentColor"
		>
			<path
				stroke-linecap="round"
				stroke-linejoin="round"
				stroke-width="2"
				d="M13.828 10.172a4 4 0 00-5.656 0l-4 4a4 4 0 105.656 5.656l1.102-1.101m-.758-4.899a4 4 0 005.656 0l4-4a4 4 0 00-5.656-5.656l-1.1 1.1"
			/>
		</svg>
		<h3 class="ih-empty__title">No integrations connected</h3>
		<p class="ih-empty__text">Connect your favorite tools to get started.</p>
		<button
			onclick={onbrowse}
			class="btn-pill btn-pill-primary btn-pill-sm mt-4"
		>
			Browse Available Integrations
		</button>
	</div>
{:else}
	<div class="ih-grid">
		{#each integrations as integration}
			<div class="ih-card">
				<div class="ih-card__header">
					<div class="ih-card__icon-wrap">
						{#if integration.icon_url}
							<img
								src={integration.icon_url}
								alt={integration.provider_name}
								class="w-6 h-6"
							/>
						{:else}
							<span class="ih-card__icon-letter">
								{integration.provider_name.charAt(0)}
							</span>
						{/if}
					</div>
					<div class="ih-card__info">
						<div class="ih-card__name-row">
							<h3 class="ih-card__name">{integration.provider_name}</h3>
							<span class="ih-badge {getStatusBadgeClass(integration.status)}">
								{integration.status}
							</span>
						</div>
						<p class="ih-card__meta">
							{integration.external_account_name ||
								integration.external_workspace_name ||
								'Connected'}
						</p>
						{#if integration.last_used_at}
							<p class="ih-card__sub-meta">
								Last used {new Date(integration.last_used_at).toLocaleDateString()}
							</p>
						{/if}
					</div>
				</div>
				<div class="ih-card__actions">
					<button
						onclick={() => onsync(integration.id)}
						disabled={syncingId === integration.id}
						class="ih-card__sync-btn"
						title="Sync now"
					>
						{#if syncingId === integration.id}
							<svg class="w-4 h-4 ih-spinner--inline" viewBox="0 0 24 24">
								<circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" fill="none" />
								<path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z" />
							</svg>
						{:else}
							<svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
							</svg>
						{/if}
					</button>
					<a
						href="/integrations/{integration.id}"
						class="ih-card__settings-link"
						title="Configure"
					>
						<svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z" />
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
						</svg>
					</a>
					<button
						onclick={() => ondisconnect(integration.id)}
						class="btn-pill btn-pill-danger btn-pill-sm ih-card__actions-btn"
					>
						Disconnect
					</button>
				</div>
			</div>
		{/each}
	</div>
{/if}

<style>
	.ih-empty {
		text-align: center;
		padding: 3rem 0;
	}
	.ih-empty__icon {
		width: 3rem;
		height: 3rem;
		margin: 0 auto;
		color: var(--dt4);
	}
	.ih-empty__title {
		margin-top: 1rem;
		font-size: 1.125rem;
		font-weight: 500;
		color: var(--dt);
	}
	.ih-empty__text {
		margin-top: 0.5rem;
		color: var(--dt3);
	}
	.ih-grid {
		display: grid;
		grid-template-columns: repeat(1, 1fr);
		gap: 0.75rem;
	}
	@media (min-width: 768px) {
		.ih-grid { grid-template-columns: repeat(2, 1fr); }
	}
	@media (min-width: 1024px) {
		.ih-grid { grid-template-columns: repeat(4, 1fr); }
	}
	@media (min-width: 1440px) {
		.ih-grid { grid-template-columns: repeat(5, 1fr); }
	}
	.ih-card {
		background: var(--dbg2);
		border: 1px solid var(--dbd);
		border-radius: 0.75rem;
		padding: 0.75rem;
		transition: box-shadow 0.15s;
	}
	.ih-card:hover {
		box-shadow: 0 4px 12px rgba(0, 0, 0, 0.08);
	}
	.ih-card__header {
		display: flex;
		align-items: flex-start;
		justify-content: space-between;
	}
	.ih-card__icon-wrap {
		width: 2rem;
		height: 2rem;
		border-radius: 0.5rem;
		background: var(--dbg3);
		display: flex;
		align-items: center;
		justify-content: center;
		overflow: hidden;
		flex-shrink: 0;
	}
	.ih-card__icon-wrap img {
		width: 1.25rem;
		height: 1.25rem;
		object-fit: contain;
	}
	.ih-card__icon-letter {
		font-size: 0.875rem;
		font-weight: 700;
		color: var(--dt3);
	}
	.ih-card__info {
		margin-left: 0.75rem;
		flex: 1;
		min-width: 0;
	}
	.ih-card__name-row {
		display: flex;
		align-items: center;
		gap: 0.5rem;
	}
	.ih-card__name {
		font-weight: 500;
		color: var(--dt);
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}
	.ih-card__meta {
		font-size: 0.75rem;
		color: var(--dt4);
		margin-top: 0.25rem;
	}
	.ih-card__sub-meta {
		font-size: 0.6875rem;
		color: var(--dt4);
		margin-top: 0.125rem;
	}
	.ih-card__actions {
		display: flex;
		align-items: center;
		gap: 0.25rem;
		margin-left: 0.5rem;
	}
	.ih-card__actions-btn {
		padding: 0.375rem;
		border-radius: 0.375rem;
		color: var(--dt4);
		background: none;
		border: none;
		cursor: pointer;
		transition: color 0.15s, background 0.15s;
	}
	.ih-card__actions-btn:hover {
		color: var(--dt2);
		background: var(--dbg3);
	}
	.ih-card__settings-link {
		font-size: 0.75rem;
		color: #3b82f6;
		text-decoration: none;
		display: inline-flex;
		align-items: center;
		gap: 0.25rem;
		margin-top: 0.5rem;
	}
	.ih-card__settings-link:hover {
		text-decoration: underline;
	}
	.ih-card__sync-btn {
		padding: 0.375rem;
		border-radius: 0.375rem;
		color: var(--dt4);
		background: none;
		border: 1px solid var(--dbd);
		cursor: pointer;
		transition: color 0.15s, background 0.15s;
		display: inline-flex;
		align-items: center;
	}
	.ih-card__sync-btn:hover:not(:disabled) {
		color: #3b82f6;
		background: rgba(59, 130, 246, 0.08);
		border-color: rgba(59, 130, 246, 0.3);
	}
	.ih-card__sync-btn:disabled {
		opacity: 0.5;
		cursor: not-allowed;
	}
	.ih-badge {
		display: inline-flex;
		padding: 0.125rem 0.5rem;
		border-radius: 9999px;
		font-size: 0.75rem;
		font-weight: 500;
		white-space: nowrap;
	}
	.ih-badge--connected { background: rgba(34, 197, 94, 0.1); color: #22c55e; }
	.ih-badge--available { background: rgba(59, 130, 246, 0.1); color: #3b82f6; }
	.ih-badge--coming-soon { background: rgba(156, 163, 175, 0.1); color: var(--dt4); }
	.ih-badge--error { background: rgba(239, 68, 68, 0.1); color: #ef4444; }
	.ih-badge--default { background: rgba(156, 163, 175, 0.1); color: var(--dt3); }
	.ih-spinner--inline {
		animation: ih-spin 0.7s linear infinite;
	}
	@keyframes ih-spin {
		to { transform: rotate(360deg); }
	}
</style>
