<script lang="ts">
	/**
	 * AppShell - Main wrapper component for generated apps
	 * Provides header, toolbar, and content layout
	 */

	import type { AppBranding, ToolbarConfig } from '../types';
	import { TemplateButton, TemplateInput, TemplateSelect } from '../primitives';

	interface Props {
		branding: AppBranding;
		toolbar?: ToolbarConfig;
		selectedCount?: number;
		currentView?: string;
		views?: { id: string; name: string; type: string }[];
		onsearch?: (query: string) => void;
		onviewchange?: (viewId: string) => void;
		oncreate?: () => void;
	}

	let {
		branding,
		toolbar = {},
		selectedCount = 0,
		currentView,
		views = [],
		onsearch,
		onviewchange,
		oncreate,
		children
	}: Props & { children?: any } = $props();

	let searchQuery = $state('');

	function handleSearch(e: Event) {
		const input = e.target as HTMLInputElement;
		searchQuery = input.value;
		onsearch?.(searchQuery);
	}

	const viewOptions = $derived(views.map((v) => ({ value: v.id, label: v.name })));
</script>

<div class="tpl-app-shell">
	<!-- Header -->
	<header class="tpl-app-header">
		<div class="tpl-app-header-left">
			{#if branding.icon}
				<span class="tpl-app-icon">{branding.icon}</span>
			{/if}
			<h1 class="tpl-app-title">{branding.name}</h1>
			{#if branding.description}
				<span class="tpl-app-desc">{branding.description}</span>
			{/if}
		</div>
		<div class="tpl-app-header-right">
			{#if toolbar.actions}
				{#each toolbar.actions as action}
					<TemplateButton
						variant={action.variant || 'secondary'}
						size="sm"
					>
						{action.label}
					</TemplateButton>
				{/each}
			{/if}
			<TemplateButton variant="primary" size="sm" onclick={oncreate}>
				<svg viewBox="0 0 20 20" fill="currentColor" style="width: 16px; height: 16px; margin-right: 4px;">
					<path d="M10 5a1 1 0 011 1v3h3a1 1 0 110 2h-3v3a1 1 0 11-2 0v-3H6a1 1 0 110-2h3V6a1 1 0 011-1z" />
				</svg>
				New
			</TemplateButton>
		</div>
	</header>

	<!-- Toolbar -->
	<div class="tpl-app-toolbar">
		<div class="tpl-app-toolbar-left">
			{#if toolbar.showSearch !== false}
				<div class="tpl-app-search">
					<svg class="tpl-app-search-icon" viewBox="0 0 20 20" fill="currentColor">
						<path fill-rule="evenodd" d="M8 4a4 4 0 100 8 4 4 0 000-8zM2 8a6 6 0 1110.89 3.476l4.817 4.817a1 1 0 01-1.414 1.414l-4.816-4.816A6 6 0 012 8z" clip-rule="evenodd" />
					</svg>
					<input
						type="search"
						class="tpl-app-search-input"
						placeholder={toolbar.searchPlaceholder || 'Search...'}
						value={searchQuery}
						oninput={handleSearch}
					/>
				</div>
			{/if}

			{#if toolbar.showViewSwitcher !== false && views.length > 1}
				<div class="tpl-app-view-switcher">
					{#each views as view}
						<button
							type="button"
							class="tpl-app-view-btn"
							class:active={currentView === view.id}
							onclick={() => onviewchange?.(view.id)}
							title={view.name}
						>
							{#if view.type === 'table'}
								<svg viewBox="0 0 20 20" fill="currentColor">
									<path fill-rule="evenodd" d="M5 4a3 3 0 00-3 3v6a3 3 0 003 3h10a3 3 0 003-3V7a3 3 0 00-3-3H5zm-1 9v-1h5v2H5a1 1 0 01-1-1zm7 1h4a1 1 0 001-1v-1h-5v2zm0-4h5V8h-5v2zM9 8H4v2h5V8z" clip-rule="evenodd" />
								</svg>
							{:else if view.type === 'card'}
								<svg viewBox="0 0 20 20" fill="currentColor">
									<path d="M5 3a2 2 0 00-2 2v2a2 2 0 002 2h2a2 2 0 002-2V5a2 2 0 00-2-2H5zM5 11a2 2 0 00-2 2v2a2 2 0 002 2h2a2 2 0 002-2v-2a2 2 0 00-2-2H5zM11 5a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2h-2a2 2 0 01-2-2V5zM11 13a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2h-2a2 2 0 01-2-2v-2z" />
								</svg>
							{:else if view.type === 'kanban'}
								<svg viewBox="0 0 20 20" fill="currentColor">
									<path d="M2 4.5A2.5 2.5 0 014.5 2h11A2.5 2.5 0 0118 4.5v11a2.5 2.5 0 01-2.5 2.5h-11A2.5 2.5 0 012 15.5v-11zM4.5 4A.5.5 0 004 4.5v11a.5.5 0 00.5.5H7V4H4.5zM9 16h2V4H9v12zm4 0h2.5a.5.5 0 00.5-.5v-11a.5.5 0 00-.5-.5H13v12z" />
								</svg>
							{:else}
								<svg viewBox="0 0 20 20" fill="currentColor">
									<path fill-rule="evenodd" d="M3 5a2 2 0 012-2h10a2 2 0 012 2v10a2 2 0 01-2 2H5a2 2 0 01-2-2V5zm11 1H6v8l4-2 4 2V6z" clip-rule="evenodd" />
								</svg>
							{/if}
						</button>
					{/each}
				</div>
			{/if}

			{#if toolbar.showFilter !== false}
				<TemplateButton variant="ghost" size="sm">
					<svg viewBox="0 0 20 20" fill="currentColor" style="width: 16px; height: 16px; margin-right: 4px;">
						<path fill-rule="evenodd" d="M3 3a1 1 0 011-1h12a1 1 0 011 1v3a1 1 0 01-.293.707L12 11.414V15a1 1 0 01-.293.707l-2 2A1 1 0 018 17v-5.586L3.293 6.707A1 1 0 013 6V3z" clip-rule="evenodd" />
					</svg>
					Filter
				</TemplateButton>
			{/if}

			{#if toolbar.showSort !== false}
				<TemplateButton variant="ghost" size="sm">
					<svg viewBox="0 0 20 20" fill="currentColor" style="width: 16px; height: 16px; margin-right: 4px;">
						<path d="M3 3a1 1 0 000 2h11a1 1 0 100-2H3zM3 7a1 1 0 000 2h7a1 1 0 100-2H3zM3 11a1 1 0 100 2h4a1 1 0 100-2H3zM15 8a1 1 0 10-2 0v5.586l-1.293-1.293a1 1 0 00-1.414 1.414l3 3a1 1 0 001.414 0l3-3a1 1 0 00-1.414-1.414L15 13.586V8z" />
					</svg>
					Sort
				</TemplateButton>
			{/if}
		</div>

		<div class="tpl-app-toolbar-right">
			{#if selectedCount > 0}
				<span class="tpl-app-selection-count">
					{selectedCount} selected
				</span>
			{/if}

			{#if toolbar.showExport !== false}
				<TemplateButton variant="ghost" size="sm">
					<svg viewBox="0 0 20 20" fill="currentColor" style="width: 16px; height: 16px;">
						<path fill-rule="evenodd" d="M3 17a1 1 0 011-1h12a1 1 0 110 2H4a1 1 0 01-1-1zm3.293-7.707a1 1 0 011.414 0L9 10.586V3a1 1 0 112 0v7.586l1.293-1.293a1 1 0 111.414 1.414l-3 3a1 1 0 01-1.414 0l-3-3a1 1 0 010-1.414z" clip-rule="evenodd" />
					</svg>
				</TemplateButton>
			{/if}
		</div>
	</div>

	<!-- Content -->
	<main class="tpl-app-content">
		{@render children?.()}
	</main>
</div>

<style>
	.tpl-app-shell {
		display: flex;
		flex-direction: column;
		height: 100%;
		background: var(--tpl-bg-secondary);
		font-family: var(--tpl-font-sans);
	}

	.tpl-app-header {
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: var(--tpl-space-4) var(--tpl-space-6);
		background: var(--tpl-bg-primary);
		border-bottom: 1px solid var(--tpl-border-default);
	}

	.tpl-app-header-left {
		display: flex;
		align-items: center;
		gap: var(--tpl-space-3);
	}

	.tpl-app-icon {
		font-size: 24px;
	}

	.tpl-app-title {
		margin: 0;
		font-size: var(--tpl-text-xl);
		font-weight: var(--tpl-font-semibold);
		color: var(--tpl-text-primary);
	}

	.tpl-app-desc {
		font-size: var(--tpl-text-sm);
		color: var(--tpl-text-muted);
	}

	.tpl-app-header-right {
		display: flex;
		align-items: center;
		gap: var(--tpl-space-2);
	}

	.tpl-app-toolbar {
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: var(--tpl-space-3) var(--tpl-space-6);
		background: var(--tpl-bg-primary);
		border-bottom: 1px solid var(--tpl-border-default);
	}

	.tpl-app-toolbar-left {
		display: flex;
		align-items: center;
		gap: var(--tpl-space-2);
	}

	.tpl-app-toolbar-right {
		display: flex;
		align-items: center;
		gap: var(--tpl-space-3);
	}

	.tpl-app-search {
		position: relative;
		display: flex;
		align-items: center;
	}

	.tpl-app-search-icon {
		position: absolute;
		left: var(--tpl-space-2-5);
		width: var(--tpl-icon-sm);
		height: var(--tpl-icon-sm);
		color: var(--tpl-text-muted);
		pointer-events: none;
	}

	.tpl-app-search-input {
		width: 220px;
		height: var(--tpl-size-sm);
		padding: 0 var(--tpl-space-3) 0 var(--tpl-space-8);
		background: var(--tpl-bg-secondary);
		border: 1px solid var(--tpl-border-default);
		border-radius: var(--tpl-radius-md);
		font-family: var(--tpl-font-sans);
		font-size: var(--tpl-text-xs);
		color: var(--tpl-text-primary);
		transition: all var(--tpl-transition-fast);
	}

	.tpl-app-search-input:focus {
		outline: none;
		border-color: var(--tpl-border-focus);
		box-shadow: var(--tpl-shadow-focus);
	}

	.tpl-app-search-input::placeholder {
		color: var(--tpl-text-muted);
	}

	.tpl-app-view-switcher {
		display: flex;
		align-items: center;
		padding: 2px;
		background: var(--tpl-bg-secondary);
		border-radius: var(--tpl-radius-md);
	}

	.tpl-app-view-btn {
		display: flex;
		align-items: center;
		justify-content: center;
		width: 26px;
		height: 26px;
		padding: 0;
		background: transparent;
		border: none;
		border-radius: var(--tpl-radius-sm);
		color: var(--tpl-text-muted);
		cursor: pointer;
		transition: all var(--tpl-transition-fast);
	}

	.tpl-app-view-btn:hover {
		color: var(--tpl-text-primary);
		background: var(--tpl-bg-hover);
	}

	.tpl-app-view-btn:focus-visible {
		outline: none;
		box-shadow: var(--tpl-shadow-focus);
	}

	.tpl-app-view-btn.active {
		color: var(--tpl-accent-primary);
		background: var(--tpl-bg-primary);
		box-shadow: var(--tpl-shadow-xs);
	}

	.tpl-app-view-btn svg {
		width: var(--tpl-icon-md);
		height: var(--tpl-icon-md);
	}

	.tpl-app-selection-count {
		font-size: var(--tpl-text-xs);
		font-weight: var(--tpl-font-medium);
		color: var(--tpl-text-secondary);
		padding: var(--tpl-space-1) var(--tpl-space-2);
		background: var(--tpl-bg-selected);
		border-radius: var(--tpl-radius-sm);
	}

	.tpl-app-content {
		flex: 1;
		padding: var(--tpl-space-6);
		overflow-y: auto;
	}
</style>
