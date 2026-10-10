<script lang="ts">
	import { onMount } from 'svelte';
	import {
		deployedAppsStore,
		type DeployedApp,
		categoryIconMap,
		categoryColorMap
	} from '$lib/stores/deployedAppsStore';
	import * as Icons from 'lucide-svelte';
	import Loading from '$lib/ui/loading/Loading.svelte';

	// Category filter state
	let selectedCategory = $state<string>('all');
	let searchQuery = $state('');

	const { apps, loading, error } = $derived($deployedAppsStore);

	// Build icon component map from Lucide icons based on categoryIconMap
	const categoryIcons: Record<string, typeof Icons.AppWindow> = {};
	for (const [category, iconName] of Object.entries(categoryIconMap)) {
		const IconComponent = Icons[iconName as keyof typeof Icons] as typeof Icons.AppWindow;
		if (IconComponent) {
			categoryIcons[category] = IconComponent;
		} else {
			categoryIcons[category] = Icons.AppWindow;
		}
	}

	// Category colors (Tailwind classes) - derived from shared color map
	const categoryColors: Record<string, { bg: string; text: string }> = {};
	for (const [category, colors] of Object.entries(categoryColorMap)) {
		categoryColors[category] = {
			bg: `bg-[${colors.bg}]`,
			text: colors.text
		};
	}

	// Fallback Tailwind class colors for styling (since dynamic bg colors need special handling)
	const tailwindColors: Record<string, { bg: string; text: string }> = {
		finance: { bg: 'bg-green-500/10', text: 'text-green-400' },
		communication: { bg: 'bg-blue-500/10', text: 'text-blue-400' },
		productivity: { bg: 'bg-purple-500/10', text: 'text-purple-400' },
		analytics: { bg: 'bg-orange-500/10', text: 'text-orange-400' },
		ecommerce: { bg: 'bg-pink-500/10', text: 'text-pink-400' },
		crm: { bg: 'bg-cyan-500/10', text: 'text-cyan-400' },
		hr: { bg: 'bg-indigo-500/10', text: 'text-indigo-400' },
		inventory: { bg: 'bg-amber-500/10', text: 'text-amber-400' },
		marketing: { bg: 'bg-rose-500/10', text: 'text-rose-400' },
		project: { bg: 'bg-teal-500/10', text: 'text-teal-400' },
		general: { bg: 'bg-gray-500/10', text: 'text-gray-400' }
	};

	// Start discovery on mount
	onMount(() => {
		deployedAppsStore.startDiscovery();
		return () => {
			deployedAppsStore.stopDiscovery();
		};
	});

	// Get unique categories from deployed apps
	const categories = $derived.by(() => {
		const cats = new Set<string>();
		apps.forEach((app) => {
			if (app.metadata?.category) {
				cats.add(app.metadata.category);
			}
		});
		return ['all', ...Array.from(cats).sort()];
	});

	// Filter apps by category and search
	const filteredApps = $derived.by(() => {
		let filtered = apps;

		// Filter by category
		if (selectedCategory !== 'all') {
			filtered = filtered.filter((app) => app.metadata?.category === selectedCategory);
		}

		// Filter by search query
		if (searchQuery.trim()) {
			const query = searchQuery.toLowerCase();
			filtered = filtered.filter(
				(app) =>
					app.name.toLowerCase().includes(query) ||
					app.metadata?.name?.toLowerCase().includes(query) ||
					app.metadata?.description?.toLowerCase().includes(query) ||
					app.metadata?.keywords?.some((k) => k.toLowerCase().includes(query))
			);
		}

		return filtered;
	});

	function getCategoryIcon(category: string) {
		return categoryIcons[category?.toLowerCase()] || categoryIcons.general;
	}

	function getCategoryColorClasses(category: string) {
		return tailwindColors[category?.toLowerCase()] || tailwindColors.general;
	}

	function openApp(app: DeployedApp) {
		window.open(app.url, '_blank');
	}

	function formatDate(dateString: string | undefined): string {
		if (!dateString) return 'N/A';
		const date = new Date(dateString);
		return new Intl.DateTimeFormat('en-US', {
			month: 'short',
			day: 'numeric',
			hour: '2-digit',
			minute: '2-digit'
		}).format(date);
	}
</script>

<svelte:head>
	<title>Deployed Apps - BusinessOS</title>
</svelte:head>

<div class="apps-container">
	<!-- Header -->
	<div class="apps-header">
		<div class="header-content">
			<h1 class="header-title">Deployed Applications</h1>
			<p class="header-subtitle">Your running OSA-generated applications</p>
		</div>
		<button class="btn-pill btn-pill-ghost refresh-button" onclick={() => deployedAppsStore.refresh()} disabled={loading}>
			<Icons.RefreshCw size={16} class={loading ? 'animate-spin' : ''} />
			Refresh
		</button>
	</div>

	<!-- Filters -->
	<div class="filters-bar">
		<div class="search-box">
			<Icons.Search size={18} class="search-icon" />
			<input
				type="text"
				placeholder="Search apps..."
				bind:value={searchQuery}
				class="search-input"
			/>
		</div>

		<div class="category-filters">
			{#each categories as category}
				{@const Icon = getCategoryIcon(category)}
				{@const colors = getCategoryColorClasses(category)}
				<button
					class="category-btn"
					class:active={selectedCategory === category}
					onclick={() => (selectedCategory = category)}
				>
					{#if category !== 'all'}
						<svelte:component this={Icon} size={16} />
					{:else}
						<Icons.Grid3x3 size={16} />
					{/if}
					<span class="capitalize">{category}</span>
				</button>
			{/each}
		</div>
	</div>

	<!-- Apps Grid -->
	<div class="apps-body">
		{#if loading && apps.length === 0}
			<div class="loading-state">
				<Loading size="lg" />
				<p class="loading-text">Discovering deployed apps...</p>
			</div>
		{:else if error}
			<div class="error-state">
				<Icons.AlertCircle size={48} class="error-icon" />
				<h3 class="error-title">Failed to Load Apps</h3>
				<p class="error-message">{error}</p>
				<button class="btn-pill btn-pill-ghost retry-button" onclick={() => deployedAppsStore.refresh()}>
					Try Again
				</button>
			</div>
		{:else if filteredApps.length === 0}
			<div class="empty-state">
				<Icons.AppWindow size={64} class="empty-icon" />
				<h3 class="empty-title">
					{searchQuery || selectedCategory !== 'all' ? 'No matching apps' : 'No deployed apps yet'}
				</h3>
				<p class="empty-message">
					{searchQuery || selectedCategory !== 'all'
						? 'Try adjusting your filters or search query.'
						: 'Deploy your OSA-generated applications to see them here.'}
				</p>
			</div>
		{:else}
			<div class="apps-grid">
				{#each filteredApps as app (app.id)}
					{@const Icon = getCategoryIcon(app.metadata?.category || 'general')}
					{@const colors = getCategoryColorClasses(app.metadata?.category || 'general')}
					<div class="app-card" onclick={() => openApp(app)} role="button" tabindex="0">
						<!-- App Icon & Category Badge -->
						<div class="card-header">
							<div class="app-icon {colors.bg}">
								<svelte:component this={Icon} size={24} class={colors.text} />
							</div>
							{#if app.metadata?.category}
								<div class="category-badge {colors.bg} {colors.text}">
									{app.metadata.category}
								</div>
							{/if}
						</div>

						<!-- App Title & Description -->
						<div class="card-content">
							<h3 class="app-name">
								{app.metadata?.name || app.name}
							</h3>
							{#if app.metadata?.description}
								<p class="app-description">{app.metadata.description}</p>
							{/if}
						</div>

						<!-- App Meta -->
						<div class="card-meta">
							<div class="meta-item">
								<Icons.Globe size={14} class="meta-icon" />
								<span>:{app.port}</span>
							</div>
							<div class="meta-item">
								<Icons.Clock size={14} class="meta-icon" />
								<span>{formatDate(app.deployedAt)}</span>
							</div>
							<div class="status-badge" class:running={app.status === 'running'}>
								<span class="status-dot"></span>
								{app.status}
							</div>
						</div>

						<!-- Keywords -->
						{#if app.metadata?.keywords && app.metadata.keywords.length > 0}
							<div class="keywords">
								{#each app.metadata.keywords.slice(0, 3) as keyword}
									<span class="keyword-tag">{keyword}</span>
								{/each}
							</div>
						{/if}
					</div>
				{/each}
			</div>
		{/if}
	</div>
</div>

<style>
	.apps-container {
		display: flex;
		flex-direction: column;
		height: 100vh;
		background: #0f172a;
		color: #e2e8f0;
		overflow: hidden;
	}

	/* Header */
	.apps-header {
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 24px 32px;
		background: #1e293b;
		border-bottom: 1px solid #334155;
	}

	.header-content {
		flex: 1;
	}

	.header-title {
		font-size: 24px;
		font-weight: 700;
		color: #f1f5f9;
		margin: 0 0 4px 0;
	}

	.header-subtitle {
		font-size: 14px;
		color: #94a3b8;
		margin: 0;
	}

	.refresh-button {
		display: flex;
		align-items: center;
		gap: 8px;
		padding: 10px 20px;
		font-size: 14px;
		font-weight: 500;
		color: #e2e8f0;
		background: #0f172a;
		border: 1px solid #334155;
		border-radius: 8px;
		cursor: pointer;
		transition: all 0.15s;
	}

	.refresh-button:hover:not(:disabled) {
		background: #1e293b;
		border-color: #60a5fa;
	}

	.refresh-button:disabled {
		opacity: 0.5;
		cursor: not-allowed;
	}

	/* Filters Bar */
	.filters-bar {
		display: flex;
		gap: 16px;
		padding: 20px 32px;
		background: #1e293b;
		border-bottom: 1px solid #334155;
		flex-wrap: wrap;
	}

	.search-box {
		position: relative;
		flex: 1;
		min-width: 250px;
	}

	.search-icon {
		position: absolute;
		left: 12px;
		top: 50%;
		transform: translateY(-50%);
		color: #64748b;
	}

	.search-input {
		width: 100%;
		padding: 10px 12px 10px 40px;
		background: #0f172a;
		border: 1px solid #334155;
		border-radius: 8px;
		color: #e2e8f0;
		font-size: 14px;
		outline: none;
		transition: border-color 0.15s;
	}

	.search-input:focus {
		border-color: #60a5fa;
	}

	.search-input::placeholder {
		color: #64748b;
	}

	.category-filters {
		display: flex;
		gap: 8px;
		flex-wrap: wrap;
	}

	.category-btn {
		display: flex;
		align-items: center;
		gap: 6px;
		padding: 8px 14px;
		background: #0f172a;
		border: 1px solid #334155;
		border-radius: 6px;
		color: #94a3b8;
		font-size: 13px;
		cursor: pointer;
		transition: all 0.15s;
	}

	.category-btn:hover {
		background: #1e293b;
		border-color: #475569;
		color: #e2e8f0;
	}

	.category-btn.active {
		background: #1e3a8a;
		border-color: #3b82f6;
		color: #93c5fd;
	}

	.capitalize {
		text-transform: capitalize;
	}

	/* Apps Body */
	.apps-body {
		flex: 1;
		overflow-y: auto;
		padding: 32px;
	}

	/* Loading, Error, Empty States */
	.loading-state,
	.error-state,
	.empty-state {
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		gap: 16px;
		height: 100%;
		color: #94a3b8;
		text-align: center;
	}

	.error-icon {
		color: #ef4444;
	}

	.empty-icon {
		color: #475569;
	}

	.error-title,
	.empty-title {
		font-size: 18px;
		font-weight: 600;
		color: #f1f5f9;
		margin: 0;
	}

	.error-message,
	.empty-message,
	.loading-text {
		font-size: 14px;
		color: #94a3b8;
		margin: 0;
		max-width: 400px;
	}

	.retry-button {
		margin-top: 8px;
		padding: 10px 24px;
		font-size: 14px;
		font-weight: 500;
		color: #e2e8f0;
		background: #3b82f6;
		border: none;
		border-radius: 8px;
		cursor: pointer;
		transition: background 0.15s;
	}

	.retry-button:hover {
		background: #2563eb;
	}

	/* Apps Grid */
	.apps-grid {
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(320px, 1fr));
		gap: 20px;
	}

	/* App Card */
	.app-card {
		background: #1e293b;
		border: 1px solid #334155;
		border-radius: 12px;
		padding: 20px;
		cursor: pointer;
		transition: all 0.2s;
	}

	.app-card:hover {
		border-color: #60a5fa;
		box-shadow: 0 8px 24px rgba(59, 130, 246, 0.15);
		transform: translateY(-2px);
	}

	.card-header {
		display: flex;
		align-items: center;
		justify-content: space-between;
		margin-bottom: 16px;
	}

	.app-icon {
		display: flex;
		align-items: center;
		justify-content: center;
		width: 48px;
		height: 48px;
		border-radius: 12px;
	}

	.category-badge {
		padding: 4px 10px;
		border-radius: 6px;
		font-size: 11px;
		font-weight: 600;
		text-transform: uppercase;
		letter-spacing: 0.5px;
	}

	.card-content {
		margin-bottom: 16px;
	}

	.app-name {
		font-size: 18px;
		font-weight: 600;
		color: #f1f5f9;
		margin: 0 0 8px 0;
	}

	.app-description {
		font-size: 13px;
		color: #94a3b8;
		line-height: 1.5;
		margin: 0;
		display: -webkit-box;
		-webkit-line-clamp: 2;
		-webkit-box-orient: vertical;
		overflow: hidden;
	}

	.card-meta {
		display: flex;
		align-items: center;
		gap: 12px;
		padding-top: 12px;
		border-top: 1px solid #334155;
		flex-wrap: wrap;
	}

	.meta-item {
		display: flex;
		align-items: center;
		gap: 4px;
		font-size: 12px;
		color: #64748b;
	}

	.meta-icon {
		color: #475569;
	}

	.status-badge {
		display: flex;
		align-items: center;
		gap: 6px;
		padding: 4px 10px;
		background: #0f172a;
		border-radius: 6px;
		font-size: 11px;
		font-weight: 500;
		text-transform: capitalize;
		margin-left: auto;
	}

	.status-dot {
		width: 6px;
		height: 6px;
		border-radius: 50%;
		background: #64748b;
	}

	.status-badge.running .status-dot {
		background: #10b981;
		animation: pulse 2s ease-in-out infinite;
	}

	@keyframes pulse {
		0%,
		100% {
			opacity: 1;
		}
		50% {
			opacity: 0.5;
		}
	}

	.keywords {
		display: flex;
		gap: 6px;
		margin-top: 12px;
		flex-wrap: wrap;
	}

	.keyword-tag {
		padding: 3px 8px;
		background: #0f172a;
		border: 1px solid #334155;
		border-radius: 4px;
		font-size: 11px;
		color: #94a3b8;
	}

	/* Responsive */
	@media (max-width: 768px) {
		.apps-header {
			flex-direction: column;
			align-items: flex-start;
			gap: 16px;
			padding: 18px 20px;
		}

		.refresh-button {
			width: 100%;
			justify-content: center;
		}

		.filters-bar {
			flex-direction: column;
			padding: 16px 20px;
		}

		.apps-body {
			padding: 20px;
		}

		.apps-grid {
			grid-template-columns: 1fr;
		}

		.search-box {
			min-width: 0;
		}
	}

	@media (max-width: 480px) {
		.apps-header {
			padding: 14px 16px;
		}
		.filters-bar {
			padding: 12px 16px;
		}
		.apps-body {
			padding: 12px 16px;
		}
		.header-title {
			font-size: 20px;
		}
		.app-card {
			padding: 16px;
		}
		.card-meta {
			flex-wrap: wrap;
			gap: 8px;
		}
	}
</style>
