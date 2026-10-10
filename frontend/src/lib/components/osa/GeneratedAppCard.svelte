<script lang="ts">
	import type { GeneratedApp, AppStatus } from '$lib/stores/generatedAppsStore';

	interface Props {
		app: GeneratedApp;
		onView?: (app: GeneratedApp) => void;
		onDeploy?: (app: GeneratedApp) => void;
		onDelete?: (app: GeneratedApp) => void;
	}

	let { app, onView, onDeploy, onDelete }: Props = $props();

	let showMenu = $state(false);
	let showDeleteConfirm = $state(false);
	let isDeploying = $state(false);
	let isDeleting = $state(false);

	function getStatusColor(status: AppStatus): string {
		switch (status) {
			case 'deployed':
				return 'bg-green-50 dark:bg-green-900/30 text-green-700 dark:text-green-400';
			case 'generated':
				return 'bg-blue-50 dark:bg-blue-900/30 text-blue-700 dark:text-blue-400';
			case 'generating':
				return 'bg-yellow-50 dark:bg-yellow-900/30 text-yellow-700 dark:text-yellow-400';
			case 'failed':
				return 'bg-red-50 dark:bg-red-900/30 text-red-700 dark:text-red-400';
			default:
				return 'bg-gray-50 dark:bg-gray-900/30 text-gray-700 dark:text-gray-400';
		}
	}

	function getStatusIcon(status: AppStatus): string {
		switch (status) {
			case 'deployed':
				return 'M5 13l4 4L19 7'; // Check icon
			case 'generated':
				return 'M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z'; // Check circle
			case 'generating':
				return 'M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15'; // Refresh
			case 'failed':
				return 'M6 18L18 6M6 6l12 12'; // X icon
			default:
				return 'M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z'; // Alert circle
		}
	}

	function formatDate(dateString: string): string {
		const date = new Date(dateString);
		const now = new Date();
		const diffMs = now.getTime() - date.getTime();
		const diffMins = Math.floor(diffMs / 60000);
		const diffHours = Math.floor(diffMs / 3600000);
		const diffDays = Math.floor(diffMs / 86400000);

		if (diffMins < 1) return 'Just now';
		if (diffMins < 60) return `${diffMins} min ago`;
		if (diffHours < 24) return `${diffHours} hour${diffHours > 1 ? 's' : ''} ago`;
		if (diffDays < 7) return `${diffDays} day${diffDays > 1 ? 's' : ''} ago`;
		return date.toLocaleDateString();
	}

	function handleView() {
		if (onView) onView(app);
	}

	async function handleDeploy() {
		if (isDeploying) return; // Prevent duplicate clicks
		isDeploying = true;
		showMenu = false;

		try {
			if (onDeploy) await onDeploy(app);
		} finally {
			isDeploying = false;
		}
	}

	async function handleDelete() {
		if (isDeleting) return; // Prevent duplicate clicks

		if (!showDeleteConfirm) {
			showDeleteConfirm = true;
			return;
		}

		isDeleting = true;
		showMenu = false;
		showDeleteConfirm = false;

		try {
			if (onDelete) await onDelete(app);
		} finally {
			isDeleting = false;
		}
	}

	function handleMenuClick(e: MouseEvent) {
		e.stopPropagation();
		showMenu = !showMenu;
	}

	function handleMenuClose() {
		showMenu = false;
		showDeleteConfirm = false;
	}

	function handleCancelDelete(e: MouseEvent) {
		e.stopPropagation();
		showDeleteConfirm = false;
	}
</script>

<div
	class="group bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-700 rounded-xl p-5 hover:shadow-lg hover:border-gray-300 dark:hover:border-gray-600 transition-all duration-200 cursor-pointer"
	onclick={handleView}
	role="button"
	tabindex="0"
	onkeydown={(e) => {
		if (e.key === 'Enter' || e.key === ' ') {
			e.preventDefault();
			handleView();
		}
	}}
>
	<!-- Header -->
	<div class="flex items-start justify-between gap-4 mb-3">
		<div class="flex-1 min-w-0">
			<h3 class="font-semibold text-gray-900 dark:text-white truncate text-lg">
				{app.app_name}
			</h3>
			<p class="text-sm text-gray-500 dark:text-gray-400 mt-1">
				{formatDate(app.generated_at)}
			</p>
		</div>

		<!-- Status Badge -->
		<div class="flex-shrink-0">
			<div
				class="flex items-center gap-1.5 text-xs font-medium px-3 py-1.5 rounded-full {getStatusColor(
					app.status
				)}"
			>
				<svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path
						stroke-linecap="round"
						stroke-linejoin="round"
						stroke-width="2"
						d={getStatusIcon(app.status)}
					/>
				</svg>
				<span class="capitalize">{app.status}</span>
			</div>
		</div>
	</div>

	<!-- Description -->
	<p class="text-sm text-gray-600 dark:text-gray-400 line-clamp-2 mb-4">
		{app.description || 'No description provided'}
	</p>

	<!-- Progress Bar (for generating status) -->
	{#if app.status === 'generating' && app.progress !== undefined}
		{@const validProgress = Math.max(0, Math.min(100, app.progress || 0))}
		<div class="mb-4">
			<div class="flex items-center justify-between text-xs text-gray-600 dark:text-gray-400 mb-1">
				<span>{app.build_phase || 'Processing'}</span>
				<span>{validProgress}%</span>
			</div>
			<div class="w-full bg-gray-200 dark:bg-gray-700 rounded-full h-2 overflow-hidden">
				<div
					class="h-full bg-yellow-500 dark:bg-yellow-600 transition-all duration-500 rounded-full"
					style="width: {validProgress}%"
				></div>
			</div>
			{#if app.status_message}
				<p class="text-xs text-gray-500 dark:text-gray-400 mt-1 italic">
					{app.status_message}
				</p>
			{/if}
		</div>
	{/if}

	<!-- Error Message -->
	{#if app.status === 'failed' && app.error_message}
		<div class="bg-red-50 dark:bg-red-900/30 border border-red-200 dark:border-red-800 rounded-lg p-3 mb-4">
			<p class="text-sm text-red-700 dark:text-red-400 flex items-start gap-2">
				<svg class="w-4 h-4 flex-shrink-0 mt-0.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path
						stroke-linecap="round"
						stroke-linejoin="round"
						stroke-width="2"
						d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
					/>
				</svg>
				<span>{app.error_message}</span>
			</p>
		</div>
	{/if}

	<!-- Tags/Category -->
	{#if app.custom_config?.category || app.custom_config?.keywords}
		<div class="flex flex-wrap gap-2 mb-4">
			{#if app.custom_config.category}
				<span
					class="text-xs px-2.5 py-1 rounded-full bg-purple-50 dark:bg-purple-900/30 text-purple-700 dark:text-purple-400 font-medium"
				>
					{app.custom_config.category}
				</span>
			{/if}
			{#if app.custom_config.keywords}
				{#each app.custom_config.keywords.slice(0, 3) as keyword}
					<span
						class="text-xs px-2.5 py-1 rounded-full bg-gray-100 dark:bg-gray-700 text-gray-700 dark:text-gray-300"
					>
						{keyword}
					</span>
				{/each}
			{/if}
		</div>
	{/if}

	<!-- Divider -->
	<div class="border-t border-gray-100 dark:border-gray-700 mb-4"></div>

	<!-- Actions -->
	<div class="flex items-center justify-between gap-2">
		<button
			onclick={(e) => {
				e.stopPropagation();
				handleView();
			}}
			class="flex-1 px-4 py-2 text-sm font-medium text-gray-700 dark:text-gray-300 bg-gray-50 dark:bg-gray-700 hover:bg-gray-100 dark:hover:bg-gray-600 rounded-lg transition-colors"
		>
			View Details
		</button>

		<!-- Actions Menu -->
		<div class="relative">
			<button
				onclick={handleMenuClick}
				class="btn-pill btn-pill-ghost btn-pill-icon"
				aria-label="More actions"
			>
				<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path
						stroke-linecap="round"
						stroke-linejoin="round"
						stroke-width="2"
						d="M12 5v.01M12 12v.01M12 19v.01M12 6a1 1 0 110-2 1 1 0 010 2zm0 7a1 1 0 110-2 1 1 0 010 2zm0 7a1 1 0 110-2 1 1 0 010 2z"
					/>
				</svg>
			</button>

			{#if showMenu}
				<div
					role="menu"
					class="absolute right-0 bottom-full mb-2 w-48 bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-700 rounded-lg shadow-lg z-10"
					onclick={(e) => e.stopPropagation()}
				>
					{#if app.status === 'generated' && onDeploy}
						<button
							onclick={handleDeploy}
							disabled={isDeploying}
							class="btn-pill btn-pill-ghost btn-pill-sm w-full text-left flex items-center gap-2"
						>
							<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
								<path
									stroke-linecap="round"
									stroke-linejoin="round"
									stroke-width="2"
									d="M7 16a4 4 0 01-.88-7.903A5 5 0 1115.9 6L16 6a5 5 0 011 9.9M15 13l-3-3m0 0l-3 3m3-3v12"
								/>
							</svg>
							{isDeploying ? 'Deploying...' : 'Deploy'}
						</button>
					{/if}

					{#if onDelete}
						{#if !showDeleteConfirm}
							<button
								onclick={handleDelete}
								disabled={isDeleting}
								class="btn-pill btn-pill-ghost btn-pill-sm w-full text-left flex items-center gap-2"
							>
								<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
									<path
										stroke-linecap="round"
										stroke-linejoin="round"
										stroke-width="2"
										d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"
									/>
								</svg>
								Delete
							</button>
						{:else}
							<div class="p-3 bg-red-50 dark:bg-red-900/30 rounded-b-lg">
								<p class="text-xs text-red-600 dark:text-red-400 mb-2">
									Are you sure? This cannot be undone.
								</p>
								<div class="flex gap-2">
									<button
										onclick={handleDelete}
										disabled={isDeleting}
										class="btn-pill btn-pill-danger btn-pill-xs flex-1"
									>
										{isDeleting ? 'Deleting...' : 'Delete'}
									</button>
									<button
										onclick={handleCancelDelete}
										disabled={isDeleting}
										class="btn-pill btn-pill-soft btn-pill-xs flex-1"
									>
										Cancel
									</button>
								</div>
							</div>
						{/if}
					{/if}
				</div>
			{/if}
		</div>
	</div>
</div>

<!-- Click outside to close menu -->
{#if showMenu}
	<button
		class="fixed inset-0 z-0 bg-transparent"
		onclick={handleMenuClose}
		aria-hidden="true"
	></button>
{/if}

<style>
	.line-clamp-2 {
		display: -webkit-box;
		-webkit-line-clamp: 2;
		line-clamp: 2;
		-webkit-box-orient: vertical;
		overflow: hidden;
	}
</style>
