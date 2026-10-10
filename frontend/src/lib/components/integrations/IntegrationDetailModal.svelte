<script lang="ts">
	import { fade, slide } from 'svelte/transition';
	import type { IntegrationProviderInfo, UserIntegration } from '$lib/api/integrations';

	interface Props {
		provider: IntegrationProviderInfo;
		connectedIntegration: UserIntegration | undefined;
		isConnected: boolean;
		fileImportProviders: string[];
		categoryInfo: Record<string, { desc: string; features: string[] }>;
		onclose: () => void;
		onconnect: (provider: IntegrationProviderInfo) => void;
	}

	let {
		provider,
		connectedIntegration,
		isConnected,
		fileImportProviders,
		categoryInfo,
		onclose,
		onconnect
	}: Props = $props();
</script>

<div
	class="ih-modal-backdrop"
	role="presentation"
	onclick={onclose}
	onkeydown={(e) => { if (e.key === 'Escape') onclose(); }}
	transition:fade={{ duration: 200 }}
>
	<div
		class="ih-modal"
		onclick={(e) => e.stopPropagation()}
		transition:slide={{ duration: 200 }}
	>
		<!-- Header -->
		<div class="ih-modal__header">
			<div class="ih-modal__header-inner">
				<div class="ih-modal__provider">
					<div class="ih-modal__provider-icon">
						{#if provider.icon_url}
							<img src={provider.icon_url} alt={provider.name} class="ih-modal__provider-img" />
						{:else}
							<span class="ih-modal__provider-letter">{provider.name.charAt(0)}</span>
						{/if}
					</div>
					<div>
						<h2 class="ih-modal__title">{provider.name}</h2>
						<span class="ih-modal__category-badge">
							{provider.category}
						</span>
					</div>
				</div>
				<button
					onclick={onclose}
					class="btn-pill btn-pill-ghost btn-pill-icon"
					aria-label="Close"
				>
					<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
					</svg>
				</button>
			</div>
		</div>

		<!-- Body -->
		<div class="ih-modal__body">
			<p class="ih-modal__desc">
				{provider.description || `Connect your ${provider.name} account to sync data and enable powerful automations.`}
			</p>

			<!-- Category Features -->
			{#if categoryInfo[provider.category]}
				<div class="ih-modal__section">
					<h3 class="ih-modal__section-title">What you can do</h3>
					<ul class="ih-feature-list">
						{#each categoryInfo[provider.category].features as feature}
							<li class="ih-feature-item">
								<svg class="w-4 h-4 ih-feature-icon" fill="none" stroke="currentColor" viewBox="0 0 24 24">
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
								</svg>
								{feature}
							</li>
						{/each}
					</ul>
				</div>
			{/if}

			<!-- Sync Info -->
			<div class="ih-sync-panel">
				<h3 class="ih-modal__section-title">Sync details</h3>
				<div class="ih-sync-grid">
					{#if provider.auto_live_sync}
						<div class="ih-sync-item">
							<div class="ih-sync-icon ih-sync-icon--green">
								<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 10V3L4 14h7v7l9-11h-7z" />
								</svg>
							</div>
							<div>
								<div class="ih-sync-label">Sync type</div>
								<div class="ih-sync-value">Live sync</div>
							</div>
						</div>
					{:else}
						<div class="ih-sync-item">
							<div class="ih-sync-icon ih-sync-icon--blue">
								<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
								</svg>
							</div>
							<div>
								<div class="ih-sync-label">Sync type</div>
								<div class="ih-sync-value">Manual/Scheduled</div>
							</div>
						</div>
					{/if}

					{#if provider.est_nodes}
						<div class="ih-sync-item">
							<div class="ih-sync-icon ih-sync-icon--purple">
								<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10" />
								</svg>
							</div>
							<div>
								<div class="ih-sync-label">Est. nodes</div>
								<div class="ih-sync-value">{provider.est_nodes}</div>
							</div>
						</div>
					{/if}

					{#if provider.initial_sync}
						<div class="ih-sync-item">
							<div class="ih-sync-icon ih-sync-icon--amber">
								<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
								</svg>
							</div>
							<div>
								<div class="ih-sync-label">Initial sync</div>
								<div class="ih-sync-value">{provider.initial_sync}</div>
							</div>
						</div>
					{/if}
				</div>
			</div>

			<!-- Skills -->
			{#if provider.skills && provider.skills.length > 0}
				<div class="ih-modal__section">
					<h3 class="ih-modal__section-title">Available skills</h3>
					<div class="ih-tag-list">
						{#each provider.skills as skill}
							<span class="ih-skill-tag">{skill}</span>
						{/each}
					</div>
				</div>
			{/if}

			<!-- Modules -->
			{#if provider.modules && provider.modules.length > 0}
				<div class="ih-modal__section">
					<h3 class="ih-modal__section-title">Works with</h3>
					<div class="ih-tag-list">
						{#each provider.modules as module}
							<span class="ih-module-tag">
								<svg class="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 6h16M4 12h16M4 18h7" />
								</svg>
								{module.replace('_', ' ')}
							</span>
						{/each}
					</div>
				</div>
			{/if}
		</div>

		<!-- Footer -->
		<div class="ih-modal__footer">
			{#if isConnected}
				<div class="ih-modal__connected-row">
					<div class="ih-modal__connected-status">
						<span class="ih-status-dot--green"></span>
						<span class="ih-modal__connected-label">Connected</span>
						{#if connectedIntegration?.external_account_name}
							<span class="ih-modal__connected-account">as {connectedIntegration.external_account_name}</span>
						{/if}
					</div>
					<div class="ih-modal__connected-actions">
						{#if provider.auto_live_sync}
							<label class="ih-toggle-label">
								<span class="ih-toggle-text">Auto-sync</span>
								<div class="ih-toggle">
									<input type="checkbox" class="sr-only peer" checked />
									<div class="ih-toggle__track"></div>
									<div class="ih-toggle__thumb"></div>
								</div>
							</label>
						{/if}
						<a
							href="/integrations/{connectedIntegration?.id}"
							class="btn-pill btn-pill-ghost btn-pill-sm"
						>
							Settings
						</a>
					</div>
				</div>
			{:else if provider.status === 'coming_soon'}
				<button
					disabled
					class="btn-pill btn-pill-soft btn-pill-sm ih-modal__full-btn"
				>
					Coming Soon
				</button>
			{:else}
				<button
					onclick={() => { onclose(); onconnect(provider); }}
					class="btn-pill btn-pill-primary ih-modal__full-btn"
				>
					{fileImportProviders.includes(provider.id) ? 'Import Data' : 'Connect'}
				</button>
			{/if}
		</div>
	</div>
</div>

<style>
	.ih-modal-backdrop {
		position: fixed;
		inset: 0;
		z-index: 50;
		display: flex;
		align-items: center;
		justify-content: center;
		padding: 1rem;
		background: rgba(0, 0, 0, 0.5);
	}
	.ih-modal {
		position: relative;
		z-index: 10;
		background: var(--dbg2);
		border: 1px solid var(--dbd);
		border-radius: 1rem;
		box-shadow: 0 25px 50px rgba(0, 0, 0, 0.25);
		max-width: 32rem;
		width: 100%;
		max-height: 85vh;
		overflow: hidden;
		display: flex;
		flex-direction: column;
	}
	.ih-modal__header {
		padding: 1.5rem;
		border-bottom: 1px solid var(--dbd);
	}
	.ih-modal__header-inner {
		display: flex;
		align-items: flex-start;
		justify-content: space-between;
	}
	.ih-modal__provider {
		display: flex;
		align-items: center;
		gap: 1rem;
	}
	.ih-modal__provider-icon {
		width: 3.5rem;
		height: 3.5rem;
		border-radius: 0.75rem;
		background: var(--dbg3);
		display: flex;
		align-items: center;
		justify-content: center;
		overflow: hidden;
		flex-shrink: 0;
	}
	.ih-modal__provider-img {
		width: 2.25rem;
		height: 2.25rem;
		object-fit: contain;
	}
	.ih-modal__provider-letter {
		font-size: 1.25rem;
		font-weight: 700;
		color: var(--dt3);
	}
	.ih-modal__title {
		font-size: 1.25rem;
		font-weight: 700;
		color: var(--dt);
	}
	.ih-modal__category-badge {
		display: inline-flex;
		align-items: center;
		gap: 0.25rem;
		padding: 0.125rem 0.5rem;
		margin-top: 0.25rem;
		font-size: 0.75rem;
		font-weight: 500;
		border-radius: 9999px;
		background: var(--dbg3);
		color: var(--dt3);
		text-transform: capitalize;
	}
	.ih-modal__body {
		padding: 1.5rem;
		overflow-y: auto;
		flex: 1;
	}
	.ih-modal__desc {
		color: var(--dt3);
		margin-bottom: 1.5rem;
	}
	.ih-modal__section {
		margin-bottom: 1.5rem;
	}
	.ih-modal__section-title {
		font-size: 0.875rem;
		font-weight: 600;
		color: var(--dt);
		margin-bottom: 0.75rem;
	}
	.ih-modal__footer {
		padding: 1.5rem;
		border-top: 1px solid var(--dbd);
		background: var(--dbg);
		display: flex;
		align-items: center;
		justify-content: flex-end;
		gap: 0.75rem;
	}
	.ih-modal__full-btn {
		width: 100%;
	}
	.ih-modal__connected-row {
		display: flex;
		align-items: center;
		justify-content: space-between;
		width: 100%;
	}
	.ih-modal__connected-status {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		color: #22c55e;
	}
	.ih-modal__connected-label {
		font-size: 0.875rem;
		font-weight: 500;
	}
	.ih-modal__connected-account {
		font-size: 0.875rem;
		color: var(--dt3);
	}
	.ih-modal__connected-actions {
		display: flex;
		align-items: center;
		gap: 0.5rem;
	}
	.ih-feature-list {
		display: flex;
		flex-direction: column;
		gap: 0.5rem;
	}
	.ih-feature-item {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		font-size: 0.875rem;
		color: var(--dt3);
	}
	.ih-feature-icon {
		color: #22c55e;
		flex-shrink: 0;
	}
	.ih-sync-panel {
		background: var(--dbg);
		border-radius: 0.75rem;
		padding: 1rem;
		margin-bottom: 1.5rem;
	}
	.ih-sync-grid {
		display: grid;
		grid-template-columns: repeat(2, 1fr);
		gap: 1rem;
	}
	.ih-sync-item {
		display: flex;
		align-items: center;
		gap: 0.5rem;
	}
	.ih-sync-icon {
		width: 2rem;
		height: 2rem;
		border-radius: 0.5rem;
		display: flex;
		align-items: center;
		justify-content: center;
		flex-shrink: 0;
	}
	.ih-sync-icon--green { background: rgba(34, 197, 94, 0.1); color: #22c55e; }
	.ih-sync-icon--blue { background: rgba(59, 130, 246, 0.1); color: #3b82f6; }
	.ih-sync-icon--purple { background: rgba(168, 85, 247, 0.1); color: #a855f7; }
	.ih-sync-icon--amber { background: rgba(245, 158, 11, 0.1); color: #f59e0b; }
	.ih-sync-label {
		font-size: 0.75rem;
		color: var(--dt4);
	}
	.ih-sync-value {
		font-size: 0.875rem;
		font-weight: 500;
		color: var(--dt);
	}
	.ih-tag-list {
		display: flex;
		flex-wrap: wrap;
		gap: 0.5rem;
	}
	.ih-skill-tag {
		padding: 0.25rem 0.5rem;
		font-size: 0.75rem;
		font-family: monospace;
		background: var(--dbg3);
		color: var(--dt3);
		border-radius: 0.25rem;
	}
	.ih-module-tag {
		display: inline-flex;
		align-items: center;
		gap: 0.25rem;
		padding: 0.25rem 0.5rem;
		font-size: 0.75rem;
		background: rgba(59, 130, 246, 0.08);
		color: #3b82f6;
		border-radius: 0.25rem;
		text-transform: capitalize;
	}
	.ih-status-dot--green {
		width: 0.5rem;
		height: 0.5rem;
		border-radius: 50%;
		background: #22c55e;
		display: inline-block;
	}
	.ih-toggle-label {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		cursor: pointer;
	}
	.ih-toggle-text {
		font-size: 0.875rem;
		color: var(--dt3);
	}
	.ih-toggle {
		position: relative;
		display: inline-block;
		width: 36px;
		height: 20px;
		flex-shrink: 0;
	}
	.ih-toggle__track {
		width: 100%;
		height: 100%;
		border-radius: 9999px;
		background: var(--dbg3);
		transition: background 0.2s;
	}
	.ih-toggle__thumb {
		position: absolute;
		left: 0.125rem;
		top: 0.125rem;
		width: 1rem;
		height: 1rem;
		border-radius: 50%;
		background: white;
		transition: transform 0.2s;
	}
</style>
