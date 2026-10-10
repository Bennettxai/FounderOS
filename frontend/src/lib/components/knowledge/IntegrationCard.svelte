<script lang="ts">
	import { fade } from 'svelte/transition';

	interface Integration {
		id: string;
		name: string;
		icon: string;
		iconBg?: string;
		iconType?: 'text' | 'svg';
		logoPath?: string;
		description: string;
		autoLiveSync: boolean;
		status: 'connected' | 'available' | 'coming_soon';
		totalNodes?: number;
		estNodes?: string;
		dataSince?: string;
		initialSync?: string;
		tooltip?: string;
		category: 'productivity' | 'communication' | 'ai' | 'meetings' | 'crm' | 'storage' | 'project' | 'custom';
	}

	interface Props {
		integration: Integration;
		dynamicStatus: 'connected' | 'available' | 'coming_soon';
		isConnecting: boolean;
		isImporting: boolean;
		loadingStatuses: boolean;
		isFileImport: boolean;
		onConnect: () => void;
	}

	let {
		integration,
		dynamicStatus,
		isConnecting,
		isImporting,
		loadingStatuses,
		isFileImport,
		onConnect
	}: Props = $props();

	let hoveredId = $state<string | null>(null);
</script>

<div
	class="integration-card"
	class:connected={dynamicStatus === 'connected'}
	class:coming-soon={dynamicStatus === 'coming_soon'}
	class:connecting={isConnecting || isImporting}
	onmouseenter={() => hoveredId = integration.id}
	onmouseleave={() => hoveredId = null}
>
	<!-- Tooltip -->
	{#if hoveredId === integration.id && integration.tooltip && dynamicStatus !== 'coming_soon'}
		<div class="card-tooltip" transition:fade={{ duration: 150 }}>
			{integration.tooltip}
		</div>
	{/if}

	<div class="card-header">
		<div class="app-info">
			{#if integration.logoPath}
				<div class="app-icon logo">
					<img src={integration.logoPath} alt={integration.name} />
				</div>
			{:else}
				<div
					class="app-icon"
					style="background: {integration.iconBg}"
				>
					{integration.icon}
				</div>
			{/if}
			<span class="app-name">{integration.name}</span>
			{#if integration.autoLiveSync}
				<span class="auto-sync-badge">
					Auto Live-sync
					<svg width="12" height="12" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
					</svg>
				</span>
			{/if}
		</div>
		{#if dynamicStatus === 'connected'}
			<span class="status-badge connected">
				<span class="status-dot"></span>
				Live-Synced
			</span>
		{:else if dynamicStatus === 'coming_soon'}
			<span class="status-badge coming-soon">Soon</span>
		{:else if isConnecting || isImporting}
			<span class="status-badge connecting">
				<span class="spinner"></span>
				{isImporting ? 'Importing...' : 'Connecting...'}
			</span>
		{:else}
			<button
				class="btn-pill btn-pill-ghost connect-btn"
				onclick={onConnect}
				disabled={loadingStatuses}
			>
				{isFileImport ? 'Import' : 'Connect'}
			</button>
		{/if}
	</div>

	<p class="card-description">{integration.description}</p>

	<div class="card-stats">
		<div class="stat">
			<svg width="14" height="14" fill="none" stroke="currentColor" viewBox="0 0 24 24">
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10" />
			</svg>
			<span class="stat-label">
				{dynamicStatus === 'connected' ? 'Tot. nodes' : 'Est. nodes'}
			</span>
			<span class="stat-value">
				{integration.totalNodes ?? integration.estNodes ?? '--'}
			</span>
		</div>
		<div class="stat">
			<svg width="14" height="14" fill="none" stroke="currentColor" viewBox="0 0 24 24">
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
			</svg>
			<span class="stat-label">
				{dynamicStatus === 'connected' ? 'Data since' : 'Initial sync'}
			</span>
			<span class="stat-value">
				{integration.dataSince ?? integration.initialSync ?? '--'}
			</span>
		</div>
	</div>
</div>

<style>
	.integration-card {
		position: relative;
		background: white;
		border: 1px solid #e8e8e8;
		border-radius: 12px;
		padding: 18px;
		transition: all 0.2s;
	}

	.integration-card:hover:not(.coming-soon) {
		border-color: #d0d0d0;
		box-shadow: 0 4px 12px rgba(0, 0, 0, 0.06);
		z-index: 20;
	}

	.integration-card.coming-soon {
		opacity: 0.6;
	}

	.card-tooltip {
		position: absolute;
		bottom: -8px;
		left: 50%;
		transform: translateX(-50%) translateY(100%);
		background: #333;
		color: white;
		padding: 10px 14px;
		border-radius: 8px;
		font-size: 12px;
		max-width: 250px;
		text-align: center;
		z-index: 100;
		box-shadow: 0 4px 12px rgba(0, 0, 0, 0.2);
		pointer-events: none;
	}

	.card-tooltip::after {
		content: '';
		position: absolute;
		top: -6px;
		left: 50%;
		transform: translateX(-50%);
		border-left: 6px solid transparent;
		border-right: 6px solid transparent;
		border-bottom: 6px solid #333;
	}

	.card-header {
		display: flex;
		justify-content: space-between;
		align-items: center;
		margin-bottom: 10px;
	}

	.app-info {
		display: flex;
		align-items: center;
		gap: 8px;
		flex-wrap: wrap;
	}

	.app-icon {
		width: 28px;
		height: 28px;
		border-radius: 6px;
		display: flex;
		align-items: center;
		justify-content: center;
		font-size: 14px;
		font-weight: 600;
		color: white;
	}

	.app-icon.logo {
		background: transparent;
		padding: 2px;
	}

	.app-icon.logo img {
		width: 100%;
		height: 100%;
		object-fit: contain;
	}

	.app-name {
		font-size: 14px;
		font-weight: 600;
		color: #333;
	}

	.auto-sync-badge {
		display: flex;
		align-items: center;
		gap: 4px;
		padding: 3px 8px;
		background: #f5f5f5;
		border-radius: 4px;
		font-size: 11px;
		color: #666;
	}

	.status-badge {
		padding: 5px 12px;
		border-radius: 20px;
		font-size: 12px;
		font-weight: 500;
	}

	.status-badge.connected {
		display: flex;
		align-items: center;
		gap: 6px;
		background: white;
		color: #166534;
		border: 1px solid #e5e5e5;
	}

	.status-dot {
		width: 6px;
		height: 6px;
		background: #22c55e;
		border-radius: 50%;
	}

	.status-badge.coming-soon {
		background: #f0f0f0;
		color: #999;
	}

	.status-badge.connecting {
		display: flex;
		align-items: center;
		gap: 6px;
		background: #eef2ff;
		color: #4f46e5;
		border: 1px solid #e0e7ff;
	}

	.spinner {
		width: 12px;
		height: 12px;
		border: 2px solid #c7d2fe;
		border-top-color: #4f46e5;
		border-radius: 50%;
		animation: spin 0.8s linear infinite;
	}

	@keyframes spin {
		to { transform: rotate(360deg); }
	}

	.integration-card.connecting {
		border-color: #c7d2fe;
		background: linear-gradient(135deg, #fefefe 0%, #f5f8ff 100%);
	}

	.connect-btn:disabled {
		opacity: 0.5;
		cursor: not-allowed;
	}

	.connect-btn {
		padding: 6px 16px;
		background: #1a1a1a;
		color: white;
		border: none;
		border-radius: 20px;
		font-size: 13px;
		font-weight: 500;
		cursor: pointer;
		transition: all 0.15s;
	}

	.connect-btn:hover {
		background: #333;
	}

	.card-description {
		font-size: 13px;
		color: #666;
		line-height: 1.5;
		margin: 0 0 14px;
	}

	.card-stats {
		display: flex;
		flex-direction: column;
		gap: 6px;
		padding-top: 12px;
		border-top: 1px solid #f0f0f0;
	}

	.stat {
		display: flex;
		align-items: center;
		gap: 6px;
		font-size: 12px;
		color: #888;
	}

	.stat-label {
		flex: 1;
	}

	.stat-value {
		font-weight: 500;
		color: #555;
	}
</style>
