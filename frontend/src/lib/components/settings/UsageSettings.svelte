<script lang="ts">
	import { api, type UsageSummary, type ProviderUsage, type ModelUsage } from '$lib/api';

	let usageSummary = $state<UsageSummary | null>(null);
	let usageByProvider = $state<ProviderUsage[]>([]);
	let usageByModel = $state<ModelUsage[]>([]);
	let usagePeriod = $state<'today' | 'week' | 'month' | 'all'>('month');
	let isLoadingUsage = $state(false);

	$effect(() => {
		loadUsageData();
	});

	async function loadUsageData() {
		isLoadingUsage = true;
		try {
			const [summary, providers, models] = await Promise.all([
				api.getUsageSummary(usagePeriod),
				api.getUsageByProvider(usagePeriod === 'all' ? 'year' : usagePeriod),
				api.getUsageByModel(usagePeriod === 'all' ? 'year' : usagePeriod),
			]);
			usageSummary = summary;
			usageByProvider = providers;
			usageByModel = models;
		} catch (error) {
			console.error('Error loading usage data:', error);
		} finally {
			isLoadingUsage = false;
		}
	}

	function formatNumber(num: number): string {
		if (num >= 1000000) return (num / 1000000).toFixed(1) + 'M';
		if (num >= 1000) return (num / 1000).toFixed(1) + 'K';
		return num.toString();
	}

	function formatCost(cost: number): string {
		return '$' + cost.toFixed(4);
	}
</script>

<div class="usage-dashboard">
	<!-- Header with Period Selector -->
	<div class="usage-header">
		<div>
			<h2 class="usage-title">Usage Analytics</h2>
			<p class="usage-subtitle">Track your AI usage and costs</p>
		</div>
		<div class="period-selector">
			{#each ['today', 'week', 'month', 'all'] as period}
				<button
					onclick={() => { usagePeriod = period as typeof usagePeriod; loadUsageData(); }}
					class="period-btn"
					class:active={usagePeriod === period}
				>
					{period === 'all' ? 'All Time' : period.charAt(0).toUpperCase() + period.slice(1)}
				</button>
			{/each}
		</div>
	</div>

	{#if isLoadingUsage}
		<div class="usage-loading">
			<div class="usage-spinner"></div>
			<p>Loading analytics...</p>
		</div>
	{:else if (!usageSummary || usageSummary.total_requests === 0)}
		<div class="usage-empty">
			<div class="usage-empty-icon">
				<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
					<path stroke-linecap="round" stroke-linejoin="round" d="M3 13.125C3 12.504 3.504 12 4.125 12h2.25c.621 0 1.125.504 1.125 1.125v6.75C7.5 20.496 6.996 21 6.375 21h-2.25A1.125 1.125 0 013 19.875v-6.75zM9.75 8.625c0-.621.504-1.125 1.125-1.125h2.25c.621 0 1.125.504 1.125 1.125v11.25c0 .621-.504 1.125-1.125 1.125h-2.25a1.125 1.125 0 01-1.125-1.125V8.625zM16.5 4.125c0-.621.504-1.125 1.125-1.125h2.25C20.496 3 21 3.504 21 4.125v15.75c0 .621-.504 1.125-1.125 1.125h-2.25a1.125 1.125 0 01-1.125-1.125V4.125z" />
				</svg>
			</div>
			<h3>No Usage Data Yet</h3>
			<p>Start chatting with the AI to see your usage analytics here.</p>
		</div>
	{:else}
		<!-- Stats Grid -->
		<div class="stats-grid">
			<div class="stat-card requests">
				<div class="stat-icon">
					<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
						<path stroke-linecap="round" stroke-linejoin="round" d="M8 12h.01M12 12h.01M16 12h.01M21 12c0 4.418-4.03 8-9 8a9.863 9.863 0 01-4.255-.949L3 20l1.395-3.72C3.512 15.042 3 13.574 3 12c0-4.418 4.03-8 9-8s9 3.582 9 8z" />
					</svg>
				</div>
				<div class="stat-content">
					<span class="stat-value">{formatNumber(usageSummary?.total_requests || 0)}</span>
					<span class="stat-label">Requests</span>
				</div>
			</div>

			<div class="stat-card tokens">
				<div class="stat-icon">
					<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
						<path stroke-linecap="round" stroke-linejoin="round" d="M7 7h.01M7 3h5c.512 0 1.024.195 1.414.586l7 7a2 2 0 010 2.828l-7 7a2 2 0 01-2.828 0l-7-7A1.994 1.994 0 013 12V7a4 4 0 014-4z" />
					</svg>
				</div>
				<div class="stat-content">
					<span class="stat-value">{formatNumber(usageSummary?.total_tokens || 0)}</span>
					<span class="stat-label">Total Tokens</span>
				</div>
			</div>

			<div class="stat-card input">
				<div class="stat-icon">
					<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
						<path stroke-linecap="round" stroke-linejoin="round" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-8l-4-4m0 0L8 8m4-4v12" />
					</svg>
				</div>
				<div class="stat-content">
					<span class="stat-value">{formatNumber(usageSummary?.total_input_tokens || 0)}</span>
					<span class="stat-label">Input Tokens</span>
				</div>
			</div>

			<div class="stat-card output">
				<div class="stat-icon">
					<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
						<path stroke-linecap="round" stroke-linejoin="round" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4" />
					</svg>
				</div>
				<div class="stat-content">
					<span class="stat-value">{formatNumber(usageSummary?.total_output_tokens || 0)}</span>
					<span class="stat-label">Output Tokens</span>
				</div>
			</div>
		</div>

		<!-- Cost Analysis -->
		<div class="cost-analysis">
			<div class="cost-card spent">
				<div class="cost-header">
					<span class="cost-label">Estimated Cloud Cost</span>
					<span class="cost-badge">API Usage</span>
				</div>
				<span class="cost-value">{formatCost(usageSummary?.total_cost || 0)}</span>
				<p class="cost-note">Based on current provider pricing</p>
			</div>

			<div class="cost-card saved">
				<div class="cost-header">
					<span class="cost-label">Local Processing Savings</span>
					<span class="cost-badge saved">Saved</span>
				</div>
				<span class="cost-value">
					{formatCost((usageByProvider.find(p => p.provider === 'ollama')?.total_tokens || 0) * 0.00002)}
				</span>
				<p class="cost-note">Running {formatNumber(usageByProvider.find(p => p.provider === 'ollama')?.total_tokens || 0)} tokens locally</p>
			</div>
		</div>

		<!-- Provider Breakdown -->
		{#if usageByProvider.length > 0}
			<div class="usage-section">
				<h3 class="section-title">Usage by Provider</h3>
				<div class="provider-list">
					{#each usageByProvider as provider}
						{@const maxTokens = Math.max(...usageByProvider.map(p => p.total_tokens))}
						{@const percentage = maxTokens > 0 ? (provider.total_tokens / maxTokens) * 100 : 0}
						<div class="provider-item">
							<div class="provider-info">
								<div class="provider-icon" class:local={provider.provider === 'ollama'} class:anthropic={provider.provider === 'anthropic'} class:groq={provider.provider === 'groq'}>
									{#if provider.provider === 'ollama'}
										<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
											<rect x="2" y="3" width="20" height="14" rx="2"/>
											<path d="M8 21h8M12 17v4"/>
										</svg>
									{:else}
										<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
											<path d="M18 10h-1.26A8 8 0 1 0 9 20h9a5 5 0 0 0 0-10z"/>
										</svg>
									{/if}
								</div>
								<div class="provider-details">
									<span class="provider-name">{provider.provider}</span>
									<span class="provider-type">{provider.provider === 'ollama' ? 'Local' : 'Cloud'}</span>
								</div>
							</div>
							<div class="provider-stats">
								<div class="provider-bar-container">
									<div class="provider-bar" class:local={provider.provider === 'ollama'} style="width: {percentage}%"></div>
								</div>
								<div class="provider-numbers">
									<span class="provider-tokens">{formatNumber(provider.total_tokens)} tokens</span>
									<span class="provider-cost">{provider.provider === 'ollama' ? 'Free' : formatCost(provider.total_cost)}</span>
								</div>
							</div>
						</div>
					{/each}
				</div>
			</div>
		{/if}

		<!-- Model Usage -->
		{#if usageByModel.length > 0}
			<div class="usage-section">
				<h3 class="section-title">Model Usage</h3>
				<div class="model-grid">
					{#each usageByModel.slice(0, 6) as model}
						<div class="model-card">
							<div class="model-header">
								<span class="model-name">{model.model.split(':')[0]}</span>
								<span class="model-provider" class:local={model.provider === 'ollama'}>{model.provider}</span>
							</div>
							<div class="model-stats">
								<div class="model-stat">
									<span class="model-stat-value">{formatNumber(model.request_count)}</span>
									<span class="model-stat-label">requests</span>
								</div>
								<div class="model-stat">
									<span class="model-stat-value">{formatNumber(model.total_tokens)}</span>
									<span class="model-stat-label">tokens</span>
								</div>
							</div>
							<div class="model-cost">
								{model.provider === 'ollama' ? 'Free (Local)' : formatCost(model.total_cost)}
							</div>
						</div>
					{/each}
				</div>
			</div>
		{/if}
	{/if}
</div>

<style>
	.usage-dashboard {
		display: flex;
		flex-direction: column;
		gap: 24px;
	}

	.usage-header {
		display: flex;
		justify-content: space-between;
		align-items: flex-start;
		flex-wrap: wrap;
		gap: 16px;
	}

	.usage-title {
		font-size: 1.5rem;
		font-weight: 600;
		color: var(--color-text);
		margin: 0;
	}

	.usage-subtitle {
		font-size: 0.875rem;
		color: var(--color-text-secondary);
		margin-top: 4px;
	}

	.period-selector {
		display: flex;
		background: var(--color-bg-secondary);
		border-radius: 10px;
		padding: 4px;
		gap: 4px;
	}

	.period-btn {
		padding: 8px 16px;
		border: none;
		background: transparent;
		border-radius: 8px;
		font-size: 0.875rem;
		font-weight: 500;
		color: var(--color-text-secondary);
		cursor: pointer;
		transition: all 0.15s;
	}

	.period-btn:hover { color: var(--color-text); }

	.period-btn.active {
		background: var(--color-bg);
		color: var(--color-text);
		box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
	}

	:global(.dark) .period-btn.active {
		background: var(--bos-settings-card-bg);
		box-shadow: 0 1px 3px rgba(0, 0, 0, 0.3);
	}

	.usage-loading {
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		padding: 80px 20px;
		gap: 16px;
		color: var(--color-text-secondary);
	}

	.usage-spinner {
		width: 32px;
		height: 32px;
		border: 2px solid var(--color-border);
		border-top-color: var(--color-text);
		border-radius: 50%;
		animation: spin 0.8s linear infinite;
	}

	@keyframes spin { to { transform: rotate(360deg); } }

	.usage-empty {
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		padding: 80px 20px;
		text-align: center;
	}

	.usage-empty-icon {
		width: 64px;
		height: 64px;
		border-radius: 16px;
		background: var(--color-bg-secondary);
		display: flex;
		align-items: center;
		justify-content: center;
		margin-bottom: 16px;
	}

	.usage-empty-icon svg { width: 32px; height: 32px; color: var(--color-text-muted); }
	.usage-empty h3 { font-size: 1.125rem; font-weight: 600; color: var(--color-text); margin: 0 0 8px; }
	.usage-empty p { font-size: 0.875rem; color: var(--color-text-secondary); margin: 0; }

	.stats-grid {
		display: grid;
		grid-template-columns: repeat(4, 1fr);
		gap: 16px;
	}

	@media (max-width: 768px) { .stats-grid { grid-template-columns: repeat(2, 1fr); } }

	.stat-card {
		background: var(--color-bg);
		border: 1px solid var(--color-border);
		border-radius: 16px;
		padding: 20px;
		display: flex;
		align-items: center;
		gap: 16px;
	}

	:global(.dark) .stat-card { background: var(--bos-settings-card-bg); border-color: var(--bos-settings-card-border); }

	.stat-icon {
		width: 48px;
		height: 48px;
		border-radius: 12px;
		display: flex;
		align-items: center;
		justify-content: center;
		flex-shrink: 0;
	}

	.stat-icon svg { width: 24px; height: 24px; }

	.stat-card.requests .stat-icon { background: #dbeafe; color: #2563eb; }
	.stat-card.tokens .stat-icon { background: #f3e8ff; color: #9333ea; }
	.stat-card.input .stat-icon { background: #dcfce7; color: #16a34a; }
	.stat-card.output .stat-icon { background: #fef3c7; color: #d97706; }

	:global(.dark) .stat-card.requests .stat-icon { background: rgba(37, 99, 235, 0.2); }
	:global(.dark) .stat-card.tokens .stat-icon { background: rgba(147, 51, 234, 0.2); }
	:global(.dark) .stat-card.input .stat-icon { background: rgba(22, 163, 74, 0.2); }
	:global(.dark) .stat-card.output .stat-icon { background: rgba(217, 119, 6, 0.2); }

	.stat-content { display: flex; flex-direction: column; }
	.stat-value { font-size: 1.75rem; font-weight: 700; color: var(--color-text); line-height: 1; }
	.stat-label { font-size: 0.75rem; color: var(--color-text-muted); margin-top: 4px; text-transform: uppercase; letter-spacing: 0.5px; }

	.cost-analysis {
		display: grid;
		grid-template-columns: 1fr 1fr;
		gap: 16px;
	}

	@media (max-width: 640px) { .cost-analysis { grid-template-columns: 1fr; } }

	.cost-card {
		background: var(--color-bg);
		border: 1px solid var(--color-border);
		border-radius: 16px;
		padding: 24px;
	}

	:global(.dark) .cost-card { background: var(--bos-settings-card-bg); border-color: var(--bos-settings-card-border); }
	.cost-card.spent { border-left: 4px solid #6366f1; }
	.cost-card.saved { border-left: 4px solid #10b981; }

	.cost-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 12px; }
	.cost-label { font-size: 0.875rem; color: var(--color-text-secondary); }

	.cost-badge {
		font-size: 0.625rem;
		font-weight: 600;
		text-transform: uppercase;
		letter-spacing: 0.5px;
		padding: 4px 8px;
		border-radius: 4px;
		background: #e0e7ff;
		color: #4338ca;
	}

	.cost-badge.saved { background: #d1fae5; color: #047857; }
	:global(.dark) .cost-badge { background: color-mix(in srgb, var(--bos-category-productivity) 20%, transparent); color: var(--bos-category-productivity); }
	:global(.dark) .cost-badge.saved { background: var(--bos-status-success-bg); color: var(--bos-status-success); }

	.cost-value { font-size: 2.5rem; font-weight: 700; color: var(--color-text); display: block; }
	.cost-card.saved .cost-value { color: #10b981; }
	.cost-note { font-size: 0.75rem; color: var(--color-text-muted); margin-top: 8px; }

	.usage-section {
		background: var(--color-bg);
		border: 1px solid var(--color-border);
		border-radius: 16px;
		padding: 24px;
	}

	:global(.dark) .usage-section { background: var(--bos-settings-card-bg); border-color: var(--bos-settings-card-border); }
	.section-title { font-size: 1rem; font-weight: 600; color: var(--color-text); margin: 0 0 20px; }

	.provider-list { display: flex; flex-direction: column; gap: 16px; }

	.provider-item {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 24px;
	}

	.provider-info { display: flex; align-items: center; gap: 12px; min-width: 140px; }

	.provider-icon {
		width: 40px;
		height: 40px;
		border-radius: 10px;
		display: flex;
		align-items: center;
		justify-content: center;
		background: #e0e7ff;
		color: #4338ca;
	}

	.provider-icon.local { background: #d1fae5; color: #047857; }
	.provider-icon.anthropic { background: #fed7aa; color: #c2410c; }
	.provider-icon.groq { background: #dbeafe; color: #1d4ed8; }
	:global(.dark) .provider-icon { background: rgba(99, 102, 241, 0.2); }
	:global(.dark) .provider-icon.local { background: rgba(16, 185, 129, 0.2); }
	:global(.dark) .provider-icon.anthropic { background: rgba(194, 65, 12, 0.2); }
	.provider-icon svg { width: 20px; height: 20px; }

	.provider-details { display: flex; flex-direction: column; }
	.provider-name { font-weight: 600; color: var(--color-text); text-transform: capitalize; }
	.provider-type { font-size: 0.75rem; color: var(--color-text-muted); }

	.provider-stats { flex: 1; display: flex; flex-direction: column; gap: 8px; }

	.provider-bar-container {
		height: 8px;
		background: var(--color-bg-secondary);
		border-radius: 4px;
		overflow: hidden;
	}

	:global(.dark) .provider-bar-container { background: var(--bos-settings-input-bg); }

	.provider-bar {
		height: 100%;
		background: linear-gradient(90deg, #6366f1, #8b5cf6);
		border-radius: 4px;
		transition: width 0.3s ease;
	}

	.provider-bar.local { background: linear-gradient(90deg, #10b981, #34d399); }
	.provider-numbers { display: flex; justify-content: space-between; }
	.provider-tokens { font-size: 0.875rem; font-weight: 500; color: var(--color-text); }
	.provider-cost { font-size: 0.875rem; color: var(--color-text-secondary); }

	.model-grid {
		display: grid;
		grid-template-columns: repeat(3, 1fr);
		gap: 12px;
	}

	@media (max-width: 768px) { .model-grid { grid-template-columns: repeat(2, 1fr); } }
	@media (max-width: 480px) { .model-grid { grid-template-columns: 1fr; } }

	.model-card { background: var(--color-bg-secondary); border-radius: 12px; padding: 16px; }
	:global(.dark) .model-card { background: var(--bos-settings-card-bg); }

	.model-header { display: flex; justify-content: space-between; align-items: flex-start; margin-bottom: 12px; }
	.model-name { font-weight: 600; color: var(--color-text); font-size: 0.875rem; }

	.model-provider {
		font-size: 0.625rem;
		font-weight: 500;
		text-transform: uppercase;
		padding: 2px 6px;
		border-radius: 4px;
		background: #e0e7ff;
		color: #4338ca;
	}

	.model-provider.local { background: #d1fae5; color: #047857; }
	:global(.dark) .model-provider { background: color-mix(in srgb, var(--bos-category-productivity) 20%, transparent); color: var(--bos-category-productivity); }
	:global(.dark) .model-provider.local { background: var(--bos-status-success-bg); color: var(--bos-status-success); }

	.model-stats { display: flex; gap: 16px; margin-bottom: 8px; }
	.model-stat { display: flex; flex-direction: column; }
	.model-stat-value { font-size: 1.125rem; font-weight: 700; color: var(--color-text); }
	.model-stat-label { font-size: 0.625rem; color: var(--color-text-muted); text-transform: uppercase; }
	.model-cost { font-size: 0.75rem; color: var(--color-text-secondary); }
</style>
