<!--
	AppCard.svelte
	Card component for displaying an app in the grid

	Features:
	- Icon, name, description, version display
	- Status badge
	- Hover actions (Open, Edit, More)
	- Pin indicator
	- Generation progress bar
-->
<script lang="ts">
	import type { App } from '$lib/types/apps';
	import AppStatusBadge from './AppStatusBadge.svelte';
	import {
		Layers,
		CheckSquare,
		Users,
		Kanban,
		BookOpen,
		Calendar,
		BarChart3,
		Mail,
		Clock,
		FileText,
		Target,
		Briefcase,
		Wallet,
		MessageSquare,
		Settings,
		Pin,
		Play,
		Terminal,
		MoreHorizontal,
		AlertCircle,
		type Icon
	} from 'lucide-svelte';

	interface Props {
		app: App;
		onOpen?: (app: App) => void;
		onEdit?: (app: App) => void;
		onMore?: (app: App) => void;
		onContextMenu?: (app: App, x: number, y: number) => void;
		class?: string;
	}

	let { app, onOpen, onEdit, onMore, onContextMenu, class: className = '' }: Props = $props();

	// Icon and color mapping based on app name keywords
	type IconConfig = { icon: typeof Icon; gradient: string };
	const iconMap: Record<string, IconConfig> = {
		task: { icon: CheckSquare, gradient: 'from-blue-500 to-blue-600' },
		crm: { icon: Users, gradient: 'from-violet-500 to-purple-600' },
		client: { icon: Users, gradient: 'from-violet-500 to-purple-600' },
		project: { icon: Kanban, gradient: 'from-emerald-500 to-teal-600' },
		tracker: { icon: Kanban, gradient: 'from-emerald-500 to-teal-600' },
		journal: { icon: BookOpen, gradient: 'from-amber-400 to-orange-500' },
		calendar: { icon: Calendar, gradient: 'from-pink-500 to-rose-500' },
		analytics: { icon: BarChart3, gradient: 'from-orange-500 to-red-500' },
		report: { icon: BarChart3, gradient: 'from-orange-500 to-red-500' },
		dashboard: { icon: BarChart3, gradient: 'from-orange-500 to-red-500' },
		email: { icon: Mail, gradient: 'from-sky-500 to-blue-600' },
		inbox: { icon: Mail, gradient: 'from-sky-500 to-blue-600' },
		time: { icon: Clock, gradient: 'from-cyan-500 to-teal-500' },
		document: { icon: FileText, gradient: 'from-slate-500 to-gray-600' },
		docs: { icon: FileText, gradient: 'from-slate-500 to-gray-600' },
		goal: { icon: Target, gradient: 'from-red-500 to-rose-600' },
		okr: { icon: Target, gradient: 'from-red-500 to-rose-600' },
		work: { icon: Briefcase, gradient: 'from-indigo-500 to-violet-600' },
		finance: { icon: Wallet, gradient: 'from-green-500 to-emerald-600' },
		invoice: { icon: Wallet, gradient: 'from-green-500 to-emerald-600' },
		chat: { icon: MessageSquare, gradient: 'from-fuchsia-500 to-pink-600' },
		message: { icon: MessageSquare, gradient: 'from-fuchsia-500 to-pink-600' },
		setting: { icon: Settings, gradient: 'from-gray-500 to-slate-600' },
		workflow: { icon: Layers, gradient: 'from-indigo-500 to-blue-600' }
	};

	const defaultConfig: IconConfig = { icon: Layers, gradient: 'from-gray-600 to-gray-700' };

	function getAppConfig(name: string): IconConfig {
		const lowerName = name.toLowerCase();
		for (const [keyword, config] of Object.entries(iconMap)) {
			if (lowerName.includes(keyword)) {
				return config;
			}
		}
		return defaultConfig;
	}

	const appConfig = $derived(getAppConfig(app.name));
	const AppIcon = $derived(appConfig.icon);
	const iconGradient = $derived(appConfig.gradient);
	const isGenerating = $derived(app.status === 'generating');
	const isError = $derived(app.status === 'error');
	const isArchived = $derived(app.status === 'archived');

	let isHovered = $state(false);

	function handleCardClick() {
		if (!isGenerating && onOpen) {
			onOpen(app);
		}
	}

	function handleKeydown(e: KeyboardEvent) {
		if (e.key === 'Enter' && !isGenerating && onOpen) {
			onOpen(app);
		}
	}

	function handleContextMenu(e: MouseEvent) {
		if (onContextMenu) {
			e.preventDefault();
			onContextMenu(app, e.clientX, e.clientY);
		}
	}
</script>

<article
	class="app-card {className}"
	class:generating={isGenerating}
	class:error={isError}
	class:archived={isArchived}
	class:hovered={isHovered}
	role="article"
	aria-label="{app.name} app, version {app.version}, {app.status}"
	tabindex="0"
	onclick={handleCardClick}
	onkeydown={handleKeydown}
	oncontextmenu={handleContextMenu}
	onmouseenter={() => (isHovered = true)}
	onmouseleave={() => (isHovered = false)}
>
	<!-- Header Row -->
	<div class="card-header">
		<div class="app-icon bg-gradient-to-br {isError ? 'from-red-500 to-rose-600' : iconGradient}">
			{#if isError}
				<AlertCircle size={24} strokeWidth={1.5} />
			{:else}
				<svelte:component this={AppIcon} size={24} strokeWidth={1.5} />
			{/if}
		</div>
		<h3 class="app-name">{app.name}</h3>
		{#if app.isPinned}
			<div class="pin-indicator" title="Pinned" role="img" aria-label="Pinned app">
				<Pin size={16} strokeWidth={2} aria-hidden="true" />
			</div>
		{/if}
	</div>

	<!-- Description -->
	<p class="app-description">
		{#if isError && app.errorMessage}
			{app.errorMessage}
		{:else}
			{app.description}
		{/if}
	</p>

	<!-- Generation Progress -->
	{#if isGenerating && app.generationProgress !== undefined}
		<div class="progress-container">
			<div class="progress-bar" style="width: {app.generationProgress}%"></div>
		</div>
		<p class="progress-text">Generating... {app.generationProgress}%</p>
	{/if}

	<!-- Footer: Version + Status -->
	<div class="card-footer">
		<span class="version">
			v{app.version}
			{#if isHovered && app.versionCount > 1}
				<span class="version-count"> · {app.versionCount} versions</span>
			{/if}
		</span>
		<AppStatusBadge status={app.status} />
	</div>

	<!-- Action Bar (on hover) -->
	{#if isHovered && !isGenerating}
		<div class="action-bar">
			{#if isError}
				<button
					class="btn-pill btn-pill-ghost action-btn primary"
					onclick={(e) => {
						e.stopPropagation();
						onEdit?.(app);
					}}
				>
					Retry
				</button>
			{:else}
				<button
					class="btn-pill btn-pill-ghost action-btn"
					onclick={(e) => {
						e.stopPropagation();
						onOpen?.(app);
					}}
					aria-label="Open {app.name}"
				>
					<Play size={16} strokeWidth={2} />
					Open
				</button>
				<button
					class="btn-pill btn-pill-ghost action-btn"
					onclick={(e) => {
						e.stopPropagation();
						onEdit?.(app);
					}}
					aria-label="Edit {app.name} in terminal"
				>
					<Terminal size={16} strokeWidth={2} />
					Edit
				</button>
			{/if}
			<button
				class="btn-pill btn-pill-ghost action-btn icon-only"
				onclick={(e) => {
					e.stopPropagation();
					onMore?.(app);
				}}
				aria-label="More actions for {app.name}"
			>
				<MoreHorizontal size={16} strokeWidth={2} />
			</button>
		</div>
	{/if}
</article>

<style>
	.app-card {
		position: relative;
		display: flex;
		flex-direction: column;
		padding: 16px;
		background: white;
		border: 1px solid #e5e7eb;
		border-radius: 12px;
		cursor: pointer;
		transition: all 150ms ease;
	}

	:global(.dark) .app-card {
		background: #1f2937;
		border-color: #374151;
	}

	.app-card:hover,
	.app-card.hovered {
		transform: translateY(-2px);
		box-shadow: 0 8px 16px -4px rgba(0, 0, 0, 0.1), 0 4px 8px -4px rgba(0, 0, 0, 0.06);
		border-color: #d1d5db;
	}

	:global(.dark) .app-card:hover,
	:global(.dark) .app-card.hovered {
		border-color: #4b5563;
		box-shadow: 0 8px 16px -4px rgba(0, 0, 0, 0.3), 0 4px 8px -4px rgba(0, 0, 0, 0.2);
	}

	.app-card:active {
		transform: scale(0.98);
		transition: transform 100ms ease;
	}

	.app-card:focus {
		outline: none;
		box-shadow: 0 0 0 3px rgba(59, 130, 246, 0.5);
	}

	.app-card.generating {
		animation: pulse 2s ease-in-out infinite;
	}

	.app-card.error {
		border-left: 3px solid #ef4444;
	}

	.app-card.archived {
		opacity: 0.6;
	}

	@keyframes pulse {
		0%,
		100% {
			opacity: 1;
		}
		50% {
			opacity: 0.85;
		}
	}

	/* Header */
	.card-header {
		display: flex;
		align-items: center;
		gap: 12px;
		margin-bottom: 8px;
	}

	.app-icon {
		width: 40px;
		height: 40px;
		display: flex;
		align-items: center;
		justify-content: center;
		border-radius: 10px;
		color: white;
		flex-shrink: 0;
		box-shadow: 0 2px 4px rgba(0, 0, 0, 0.1);
	}

	.app-name {
		flex: 1;
		font-size: 16px;
		font-weight: 600;
		color: #111827;
		margin: 0;
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}

	:global(.dark) .app-name {
		color: #f3f4f6;
	}

	.pin-indicator {
		color: #9ca3af;
		flex-shrink: 0;
	}

	/* Description */
	.app-description {
		font-size: 14px;
		color: #6b7280;
		margin: 0 0 auto;
		line-height: 1.4;
		display: -webkit-box;
		-webkit-line-clamp: 2;
		-webkit-box-orient: vertical;
		overflow: hidden;
		min-height: 40px;
	}

	:global(.dark) .app-description {
		color: #9ca3af;
	}

	/* Progress bar */
	.progress-container {
		height: 6px;
		background: #e5e7eb;
		border-radius: 3px;
		overflow: hidden;
		margin: 8px 0 4px;
	}

	:global(.dark) .progress-container {
		background: #374151;
	}

	.progress-bar {
		height: 100%;
		background: #3b82f6;
		border-radius: 3px;
		transition: width 300ms ease;
	}

	.progress-text {
		font-size: 12px;
		color: #6b7280;
		margin: 0;
	}

	/* Footer */
	.card-footer {
		display: flex;
		align-items: center;
		justify-content: space-between;
		margin-top: 12px;
		padding-top: 12px;
		border-top: 1px solid #f3f4f6;
	}

	:global(.dark) .card-footer {
		border-top-color: #374151;
	}

	.version {
		font-size: 12px;
		font-weight: 500;
		color: #9ca3af;
	}

	.version-count {
		color: #6b7280;
	}

	/* Action bar */
	.action-bar {
		position: absolute;
		bottom: 0;
		left: 0;
		right: 0;
		display: flex;
		align-items: center;
		gap: 8px;
		padding: 12px 16px;
		background: #f9fafb;
		border-top: 1px solid #e5e7eb;
		border-radius: 0 0 12px 12px;
		animation: slideIn 150ms ease-out;
	}

	:global(.dark) .action-bar {
		background: #111827;
		border-top-color: #374151;
	}

	@keyframes slideIn {
		from {
			opacity: 0;
			transform: translateY(4px);
		}
		to {
			opacity: 1;
			transform: translateY(0);
		}
	}

	.action-btn {
		display: inline-flex;
		align-items: center;
		gap: 4px;
		padding: 6px 12px;
		font-size: 13px;
		font-weight: 500;
		color: #374151;
		background: white;
		border: 1px solid #e5e7eb;
		border-radius: 6px;
		cursor: pointer;
		transition: all 150ms ease;
		font-family: inherit;
	}

	:global(.dark) .action-btn {
		color: #d1d5db;
		background: #1f2937;
		border-color: #374151;
	}

	.action-btn:hover {
		background: #f3f4f6;
		border-color: #d1d5db;
	}

	:global(.dark) .action-btn:hover {
		background: #374151;
	}

	.action-btn.primary {
		background: #1a1a1a;
		color: white;
		border-color: #1a1a1a;
	}

	.action-btn.primary:hover {
		background: #333;
	}

	.action-btn.icon-only {
		padding: 6px;
		margin-left: auto;
	}

	/* Responsive */
	@media (max-width: 768px) {
		.action-bar {
			position: static;
			margin-top: 12px;
			border-radius: 8px;
		}
	}
</style>
