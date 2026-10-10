<script lang="ts">
	import type { IntegrationProviderInfo, IntegrationCategory, UserIntegration } from '$lib/api/integrations';

	interface CategoryDef {
		id: IntegrationCategory | 'all';
		label: string;
		icon: string;
	}

	interface Props {
		providers: IntegrationProviderInfo[];
		connectedIntegrations: UserIntegration[];
		categories: CategoryDef[];
		selectedCategory: IntegrationCategory | 'all';
		searchQuery: string;
		connectingId: string | null;
		onselect: (category: IntegrationCategory | 'all') => void;
		onsearch: (query: string) => void;
		onconnect: (provider: IntegrationProviderInfo) => void;
		ondisconnect: (integrationId: string) => void;
		onviewdetail: (provider: IntegrationProviderInfo) => void;
	}

	let {
		providers,
		connectedIntegrations,
		categories,
		selectedCategory,
		searchQuery,
		connectingId,
		onselect,
		onsearch,
		onconnect,
		ondisconnect,
		onviewdetail
	}: Props = $props();

	function isProviderConnected(providerId: string) {
		return connectedIntegrations.some(
			(i) => i.provider_id === providerId && i.status === 'connected'
		);
	}

	function getConnectedIntegration(providerId: string) {
		return connectedIntegrations.find((i) => i.provider_id === providerId);
	}
</script>

<!-- Header Text -->
<div class="ih-section-intro">
	<h2 class="ih-section-intro__title">
		Let's bring all your data into a single place.
	</h2>
	<p class="ih-section-intro__text">
		When you connect your apps, we will process raw data and extract essential information and turn it into nodes.
	</p>
</div>

<!-- Category Filter -->
<div class="ih-category-filter">
	<div class="ih-search-wrap">
		<svg class="w-4 h-4 ih-search-icon" fill="none" viewBox="0 0 24 24" stroke="currentColor">
			<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
		</svg>
		<input
			type="text"
			placeholder="Search integrations..."
			value={searchQuery}
			oninput={(e) => onsearch((e.currentTarget as HTMLInputElement).value)}
			class="ih-search-input"
		/>
	</div>
	<div class="ih-cat-tabs">
		{#each categories as category}
			<button
				onclick={() => onselect(category.id)}
				class="ih-cat-tab {selectedCategory === category.id ? 'ih-cat-tab--active' : ''}"
			>
				{category.label}
			</button>
		{/each}
	</div>
</div>

<!-- Integrations Grid -->
<div class="ih-grid ih-grid--pb">
	{#each providers as provider}
		{@const isConnected = isProviderConnected(provider.id)}
		{@const isConnecting = connectingId === provider.id}
		{@const isComingSoon = provider.status === 'coming_soon'}
		<div class="ih-provider-card {isComingSoon ? 'ih-provider-card--soon' : ''} {isConnecting ? 'ih-provider-card--connecting' : ''}">
			<!-- Top row: icon + info + toggle -->
			<div class="ih-provider-card__row">
				<div class="ih-provider-card__icon">
					{#if provider.icon_url}
						<img
							src={provider.icon_url}
							alt={provider.name}
							class="w-5 h-5 object-contain"
							onerror={(e) => { const target = e.currentTarget as HTMLImageElement; target.style.display = 'none'; target.nextElementSibling?.classList.remove('hidden'); }}
						/>
						<span class="hidden ih-card__icon-letter--sm">{provider.name.charAt(0)}</span>
					{:else}
						<span class="ih-card__icon-letter--sm">{provider.name.charAt(0)}</span>
					{/if}
				</div>
				<div class="ih-provider-card__info">
					<span class="ih-provider-card__name">{provider.name}</span>
					<span class="ih-provider-card__author">By {provider.name}.com</span>
				</div>
				{#if isConnecting}
					<span class="ih-spinner ih-spinner--sm"></span>
				{:else}
					<label class="ih-toggle" aria-label="{isConnected ? 'Disconnect' : 'Connect'} {provider.name}">
						<input
							type="checkbox"
							checked={isConnected}
							onchange={() => isConnected ? ondisconnect(getConnectedIntegration(provider.id)?.id ?? '') : onconnect(provider)}
							disabled={isComingSoon}
						/>
						<span class="ih-toggle__slider"></span>
					</label>
				{/if}
			</div>

			<!-- Description -->
			<p class="ih-provider-card__desc">
				{provider.description || `Connect your ${provider.name} account`}
			</p>

			<!-- Footer: tags + view instruction -->
			<div class="ih-provider-card__footer">
				<div class="ih-provider-card__tags">
					<span class="ih-tag">{(provider.category || 'general').toUpperCase().replace('_', ' ')}</span>
					{#if provider.initial_sync}<span class="ih-tag">{provider.initial_sync}</span>{/if}
					{#if isComingSoon}<span class="ih-tag ih-tag--soon">SOON</span>{/if}
				</div>
				<button onclick={() => onviewdetail(provider)} class="ih-view-link">View instruction</button>
			</div>
		</div>
	{/each}
</div>

<style>
	.ih-section-intro {
		margin-bottom: 1.5rem;
	}
	.ih-section-intro__title {
		font-size: 1.125rem;
		font-weight: 600;
		color: var(--dt);
	}
	.ih-section-intro__text {
		font-size: 0.875rem;
		color: var(--dt3);
		margin-top: 0.25rem;
	}
	.ih-category-filter {
		display: flex;
		flex-direction: column;
		gap: 0.75rem;
		margin-bottom: 1.5rem;
	}
	.ih-cat-tabs {
		display: flex;
		gap: 1.5rem;
		padding-bottom: 0.75rem;
		border-bottom: 1px solid var(--dbd2);
		overflow-x: auto;
	}
	.ih-cat-tab {
		font-size: 0.8125rem;
		font-weight: 500;
		color: var(--dt3);
		background: none;
		border: none;
		cursor: pointer;
		padding: 0.25rem 0;
		border-bottom: 2px solid transparent;
		transition: color 0.15s, border-color 0.15s;
		white-space: nowrap;
	}
	.ih-cat-tab:hover {
		color: var(--dt);
	}
	.ih-cat-tab--active {
		color: var(--dt);
		border-bottom-color: var(--dt);
		font-weight: 600;
	}
	.ih-search-wrap {
		position: relative;
		flex-shrink: 0;
	}
	.ih-search-icon {
		position: absolute;
		left: 0.625rem;
		top: 50%;
		transform: translateY(-50%);
		color: var(--dt4);
		pointer-events: none;
	}
	.ih-search-input {
		padding: 0.375rem 0.75rem 0.375rem 2rem;
		font-size: 0.8125rem;
		border-radius: 0.5rem;
		border: 1px solid var(--dbd);
		background: var(--dbg);
		color: var(--dt);
		outline: none;
		width: 14rem;
		transition: border-color 0.15s;
	}
	.ih-search-input:focus {
		border-color: #3b82f6;
	}
	.ih-search-input::placeholder {
		color: var(--dt4);
	}
	.ih-grid--pb {
		display: grid;
		grid-template-columns: repeat(1, 1fr);
		gap: 0.625rem;
		padding-bottom: 2rem;
	}
	@media (min-width: 768px) {
		.ih-grid--pb { grid-template-columns: repeat(2, 1fr); }
	}
	@media (min-width: 1024px) {
		.ih-grid--pb { grid-template-columns: repeat(3, 1fr); }
	}
	.ih-provider-card {
		background: var(--dbg);
		border: 1px solid var(--dbd2);
		border-radius: 10px;
		padding: 0.875rem 1rem;
		transition: box-shadow 0.15s;
	}
	.ih-provider-card:hover {
		box-shadow: 0 2px 8px rgba(0, 0, 0, 0.06);
	}
	.ih-provider-card--soon {
		opacity: 0.6;
	}
	.ih-provider-card--connecting {
		border-color: rgba(59, 130, 246, 0.3);
	}
	.ih-provider-card__row {
		display: flex;
		align-items: center;
		gap: 0.75rem;
	}
	.ih-provider-card__icon {
		width: 2rem;
		height: 2rem;
		border-radius: 8px;
		display: flex;
		align-items: center;
		justify-content: center;
		background: var(--dbg2);
		flex-shrink: 0;
		overflow: hidden;
	}
	.ih-provider-card__icon img {
		width: 1.25rem;
		height: 1.25rem;
		object-fit: contain;
	}
	.ih-provider-card__info {
		flex: 1;
		min-width: 0;
	}
	.ih-provider-card__name {
		font-weight: 600;
		font-size: 0.875rem;
		color: var(--dt);
		display: block;
	}
	.ih-provider-card__author {
		font-size: 0.75rem;
		color: var(--dt3);
	}
	.ih-provider-card__desc {
		font-size: 0.75rem;
		color: var(--dt3);
		margin: 0.5rem 0;
		line-height: 1.4;
		display: -webkit-box;
		-webkit-line-clamp: 2;
		line-clamp: 2;
		-webkit-box-orient: vertical;
		overflow: hidden;
	}
	.ih-provider-card__footer {
		display: flex;
		align-items: center;
		justify-content: space-between;
		margin-top: 0.5rem;
	}
	.ih-provider-card__tags {
		display: flex;
		gap: 0.375rem;
		flex-wrap: wrap;
	}
	.ih-card__icon-letter--sm {
		font-size: 0.75rem;
		font-weight: 700;
		color: var(--dt3);
	}
	.ih-tag {
		font-size: 0.625rem;
		font-weight: 600;
		text-transform: uppercase;
		letter-spacing: 0.04em;
		padding: 0.125rem 0.375rem;
		border-radius: 4px;
		background: var(--dbg2);
		color: var(--dt3);
	}
	.ih-tag--soon {
		background: rgba(239, 68, 68, 0.1);
		color: #ef4444;
	}
	.ih-view-link {
		font-size: 0.75rem;
		color: var(--dt3);
		background: none;
		border: none;
		cursor: pointer;
		padding: 0;
		text-decoration: underline;
		text-underline-offset: 2px;
	}
	.ih-view-link:hover {
		color: var(--dt);
	}
	.ih-toggle {
		position: relative;
		display: inline-block;
		width: 36px;
		height: 20px;
		flex-shrink: 0;
	}
	.ih-toggle input {
		opacity: 0;
		width: 0;
		height: 0;
	}
	.ih-toggle__slider {
		position: absolute;
		cursor: pointer;
		inset: 0;
		background: var(--dbg3);
		border-radius: 20px;
		transition: 0.2s;
	}
	.ih-toggle__slider::before {
		content: '';
		position: absolute;
		height: 16px;
		width: 16px;
		left: 2px;
		bottom: 2px;
		background: white;
		border-radius: 50%;
		transition: 0.2s;
	}
	.ih-toggle input:checked + .ih-toggle__slider {
		background: #111;
	}
	.ih-toggle input:checked + .ih-toggle__slider::before {
		transform: translateX(16px);
	}
	.ih-toggle input:disabled + .ih-toggle__slider {
		opacity: 0.5;
		cursor: not-allowed;
	}
	.ih-spinner--sm {
		width: 1rem;
		height: 1rem;
		border-width: 2px;
		border-color: var(--dbd);
		border-top-color: #3b82f6;
		border-radius: 50%;
		animation: ih-spin 0.7s linear infinite;
	}
	@keyframes ih-spin {
		to { transform: rotate(360deg); }
	}
</style>
