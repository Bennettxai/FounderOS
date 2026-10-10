<script lang="ts">
	import { onMount } from 'svelte';
	import * as integrationsApi from '$lib/api/integrations';
	import type { IntegrationProviderInfo, UserIntegration } from '$lib/api/integrations';

	// Props
	interface Props {
		moduleId: string;
		title?: string;
		compact?: boolean;
	}

	let { moduleId, title = 'Integrations', compact = false }: Props = $props();

	// State
	let isLoading = $state(true);
	let availableProviders = $state<IntegrationProviderInfo[]>([]);
	let connectedIntegrations = $state<UserIntegration[]>([]);
	let isExpanded = $state(!compact);

	onMount(async () => {
		await loadIntegrations();
	});

	async function loadIntegrations() {
		isLoading = true;
		try {
			const response = await integrationsApi.getModuleIntegrations(moduleId);
			availableProviders = response.available_providers || [];
			connectedIntegrations = response.connected_integrations || [];
		} catch (err) {
			console.error('Failed to load module integrations:', err);
		} finally {
			isLoading = false;
		}
	}

	async function handleConnect(providerId: string) {
		try {
			const response = await integrationsApi.initiateAuth(
				providerId as integrationsApi.IntegrationProvider
			);
			if (response.auth_url) {
				window.location.href = response.auth_url;
			}
		} catch (err) {
			console.error('Failed to initiate auth:', err);
		}
	}

	async function triggerSync(integrationId: string) {
		try {
			await integrationsApi.triggerIntegrationSync(integrationId, moduleId);
		} catch (err) {
			console.error('Failed to trigger sync:', err);
		}
	}

	function getStatusIndicator(status: string) {
		switch (status) {
			case 'connected':
				return 'ih-status-dot--green';
			case 'expired':
				return 'ih-status-dot--yellow';
			case 'error':
				return 'ih-status-dot--red';
			default:
				return 'ih-status-dot--gray';
		}
	}
</script>

<div class="ih-mod-panel">
	<!-- Header -->
	<button
		onclick={() => (isExpanded = !isExpanded)}
		class="ih-mod-header"
	>
		<div class="ih-mod-header__left">
			<svg class="w-5 h-5" style="color: var(--dt3)" fill="none" viewBox="0 0 24 24" stroke="currentColor">
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13.828 10.172a4 4 0 00-5.656 0l-4 4a4 4 0 105.656 5.656l1.102-1.101m-.758-4.899a4 4 0 005.656 0l4-4a4 4 0 00-5.656-5.656l-1.1 1.1" />
			</svg>
			<span class="ih-mod-header__title">{title}</span>
			{#if connectedIntegrations.length > 0}
				<span class="ih-mod-count-badge">
					{connectedIntegrations.length} connected
				</span>
			{/if}
		</div>
		<svg
			class="w-5 h-5 ih-mod-chevron {isExpanded ? 'ih-mod-chevron--open' : ''}"
			style="color: var(--dt4)"
			fill="none"
			viewBox="0 0 24 24"
			stroke="currentColor"
		>
			<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7" />
		</svg>
	</button>

	{#if isExpanded}
		<div class="ih-mod-body">
			{#if isLoading}
				<div class="ih-mod-spinner-wrap">
					<div class="ih-spinner"></div>
				</div>
			{:else if connectedIntegrations.length === 0 && availableProviders.length === 0}
				<p class="ih-mod-empty">
					No integrations available for this module.
				</p>
			{:else}
				<!-- Connected Integrations -->
				{#if connectedIntegrations.length > 0}
					<div class="ih-mod-section">
						<h4 class="ih-mod-section-label">
							Connected
						</h4>
						<div class="ih-mod-list">
							{#each connectedIntegrations as integration}
								<div class="ih-mod-row">
									<div class="ih-mod-row__left">
										<div class="ih-mod-icon-wrap">
											<div class="ih-mod-icon">
												{#if integration.icon_url}
													<img src={integration.icon_url} alt={integration.provider_name} class="w-5 h-5" />
												{:else}
													<span class="ih-mod-icon__letter">
														{integration.provider_name.charAt(0)}
													</span>
												{/if}
											</div>
											<div class="ih-mod-status-dot {getStatusIndicator(integration.status)}"></div>
										</div>
										<div>
											<p class="ih-mod-name">
												{integration.provider_name}
											</p>
											<p class="ih-mod-meta">
												{integration.external_account_name || integration.external_workspace_name || 'Connected'}
											</p>
										</div>
									</div>
									<div class="ih-mod-row__actions">
										<button
											onclick={() => triggerSync(integration.id)}
											class="ih-mod-action-btn"
											title="Sync now"
										>
											<svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
												<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
											</svg>
										</button>
										<a
											href="/integrations/{integration.id}"
											class="ih-mod-action-btn"
											title="Settings"
										>
											<svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
												<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z" />
												<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
											</svg>
										</a>
									</div>
								</div>
							{/each}
						</div>
					</div>
				{/if}

				<!-- Available Integrations -->
				{#if availableProviders.length > 0}
					<div>
						<h4 class="ih-mod-section-label">
							Available
						</h4>
						<div class="ih-mod-avail-grid">
							{#each availableProviders as provider}
								{@const isConnected = connectedIntegrations.some(i => i.provider_id === provider.id)}
								{#if !isConnected}
									<button
										onclick={() => handleConnect(provider.id)}
										disabled={provider.status !== 'available'}
										class="ih-mod-avail-btn"
									>
										<div class="ih-mod-icon">
											{#if provider.icon_url}
												<img src={provider.icon_url} alt={provider.name} class="w-5 h-5" />
											{:else}
												<span class="ih-mod-icon__letter">
													{provider.name.charAt(0)}
												</span>
											{/if}
										</div>
										<div class="ih-mod-avail-info">
											<p class="ih-mod-name ih-mod-name--truncate">
												{provider.name}
											</p>
											{#if provider.status !== 'available'}
												<p class="ih-mod-meta">
													{provider.status.replace('_', ' ')}
												</p>
											{/if}
										</div>
									</button>
								{/if}
							{/each}
						</div>
					</div>
				{/if}

				<!-- Link to full integrations page -->
				<div class="ih-mod-footer">
					<a
						href="/integrations"
						class="ih-mod-link"
					>
						Manage all integrations
					</a>
				</div>
			{/if}
		</div>
	{/if}
</div>

<style>
	.ih-mod-panel {
		background: var(--dbg2);
		border-radius: 0.75rem;
		border: 1px solid var(--dbd);
	}
	.ih-mod-header {
		width: 100%;
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 1rem;
		text-align: left;
		background: none;
		border: none;
		cursor: pointer;
		transition: background 0.15s;
	}
	.ih-mod-header:hover { background: var(--dbg3); }
	.ih-mod-header__left {
		display: flex;
		align-items: center;
		gap: 0.75rem;
	}
	.ih-mod-header__title {
		font-weight: 500;
		color: var(--dt);
	}
	.ih-mod-count-badge {
		padding: 0.125rem 0.5rem;
		font-size: 0.75rem;
		background: rgba(34, 197, 94, 0.1);
		color: #22c55e;
		border-radius: 9999px;
	}
	.ih-mod-chevron { transition: transform 0.15s; }
	.ih-mod-chevron--open { transform: rotate(180deg); }

	.ih-mod-body {
		border-top: 1px solid var(--dbd);
		padding: 1rem;
	}
	.ih-mod-spinner-wrap {
		display: flex;
		align-items: center;
		justify-content: center;
		padding: 2rem 0;
	}
	.ih-spinner {
		width: 1.5rem;
		height: 1.5rem;
		border: 2px solid var(--dbd);
		border-top-color: #3b82f6;
		border-radius: 9999px;
		animation: ih-spin 0.6s linear infinite;
	}
	@keyframes ih-spin { to { transform: rotate(360deg); } }

	.ih-mod-empty {
		font-size: 0.875rem;
		color: var(--dt3);
		text-align: center;
		padding: 1rem 0;
	}

	.ih-mod-section { margin-bottom: 1rem; }
	.ih-mod-section-label {
		font-size: 0.75rem;
		font-weight: 600;
		color: var(--dt3);
		text-transform: uppercase;
		letter-spacing: 0.05em;
		margin-bottom: 0.5rem;
	}
	.ih-mod-list {
		display: flex;
		flex-direction: column;
		gap: 0.5rem;
	}
	.ih-mod-row {
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 0.5rem;
		background: var(--dbg3);
		border-radius: 0.5rem;
	}
	.ih-mod-row__left {
		display: flex;
		align-items: center;
		gap: 0.5rem;
	}
	.ih-mod-icon-wrap { position: relative; }
	.ih-mod-icon {
		width: 2rem;
		height: 2rem;
		border-radius: 0.375rem;
		background: var(--dbg);
		display: flex;
		align-items: center;
		justify-content: center;
		flex-shrink: 0;
	}
	.ih-mod-icon__letter {
		font-size: 0.875rem;
		font-weight: 500;
		color: var(--dt2);
	}
	.ih-mod-status-dot {
		position: absolute;
		bottom: -2px;
		right: -2px;
		width: 0.625rem;
		height: 0.625rem;
		border-radius: 9999px;
		border: 2px solid var(--dbg2);
	}
	.ih-status-dot--green { background: #22c55e; }
	.ih-status-dot--yellow { background: #f59e0b; }
	.ih-status-dot--red { background: #ef4444; }
	.ih-status-dot--gray { background: var(--dt4); }

	.ih-mod-name {
		font-size: 0.875rem;
		font-weight: 500;
		color: var(--dt);
	}
	.ih-mod-name--truncate {
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}
	.ih-mod-meta {
		font-size: 0.75rem;
		color: var(--dt3);
	}
	.ih-mod-row__actions {
		display: flex;
		align-items: center;
		gap: 0.25rem;
	}
	.ih-mod-action-btn {
		padding: 0.375rem;
		color: var(--dt4);
		background: none;
		border: none;
		border-radius: 0.375rem;
		cursor: pointer;
		transition: color 0.15s, background 0.15s;
		text-decoration: none;
		display: flex;
		align-items: center;
		justify-content: center;
	}
	.ih-mod-action-btn:hover {
		color: var(--dt2);
		background: var(--dbg);
	}

	.ih-mod-avail-grid {
		display: grid;
		grid-template-columns: repeat(2, 1fr);
		gap: 0.5rem;
	}
	@media (min-width: 640px) {
		.ih-mod-avail-grid { grid-template-columns: repeat(3, 1fr); }
	}
	.ih-mod-avail-btn {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		padding: 0.5rem;
		background: var(--dbg3);
		border: none;
		border-radius: 0.5rem;
		cursor: pointer;
		text-align: left;
		transition: background 0.15s;
	}
	.ih-mod-avail-btn:hover { background: var(--dbg); }
	.ih-mod-avail-btn:disabled {
		opacity: 0.5;
		cursor: not-allowed;
	}
	.ih-mod-avail-info { min-width: 0; }

	.ih-mod-footer {
		margin-top: 1rem;
		padding-top: 1rem;
		border-top: 1px solid var(--dbd);
	}
	.ih-mod-link {
		font-size: 0.875rem;
		color: #3b82f6;
		text-decoration: none;
	}
	.ih-mod-link:hover { text-decoration: underline; }
</style>
