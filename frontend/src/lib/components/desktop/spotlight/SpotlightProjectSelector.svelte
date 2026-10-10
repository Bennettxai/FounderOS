<script lang="ts">
	import { fly } from 'svelte/transition';
	import type { Project } from './spotlightSearch.ts';

	interface Props {
		projectsList: Project[];
		selectedProjectId: string | null;
		showProjectDropdown: boolean;
		projectDropdownIndex: number;
		onProjectSelect: (id: string) => void;
		onToggleProjectDropdown: () => void;
		onProjectHover: (index: number) => void;
		onCloseAndNavigate: (appId: string) => void;
	}

	let {
		projectsList,
		selectedProjectId,
		showProjectDropdown,
		projectDropdownIndex,
		onProjectSelect,
		onToggleProjectDropdown,
		onProjectHover,
		onCloseAndNavigate
	}: Props = $props();

	let selectedProject = $derived(
		selectedProjectId ? projectsList.find((p) => p.id === selectedProjectId) : null
	);
</script>

<div class="dropdown-wrapper">
	<button
		class="selector-btn"
		class:selected={selectedProject}
		onclick={onToggleProjectDropdown}
		aria-label={selectedProject ? selectedProject.name : 'Select project'}
		aria-expanded={showProjectDropdown}
	>
		<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
			<path d="M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2z" />
		</svg>
		<span>{selectedProject ? selectedProject.name : 'Project'}</span>
		{#if !selectedProject}
			<span class="required-dot"></span>
		{/if}
		<svg class="chevron" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
			<path d="M19 9l-7 7-7-7" />
		</svg>
	</button>
	{#if showProjectDropdown}
		<div class="dropdown-menu" transition:fly={{ y: 5, duration: 150 }}>
			{#each projectsList as project, i}
				<button
					class="dropdown-item"
					class:active={selectedProjectId === project.id}
					class:highlighted={projectDropdownIndex === i}
					onclick={() => onProjectSelect(project.id)}
					onmouseenter={() => onProjectHover(i)}
				>
					{project.name}
				</button>
			{/each}
			<button
				class="dropdown-item create-new"
				class:highlighted={projectDropdownIndex === projectsList.length}
				onclick={() => onCloseAndNavigate('projects')}
				onmouseenter={() => onProjectHover(projectsList.length)}
			>
				<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
					<line x1="12" y1="5" x2="12" y2="19" /><line x1="5" y1="12" x2="19" y2="12" />
				</svg>
				New Project
			</button>
		</div>
	{/if}
</div>

<style>
	.dropdown-wrapper {
		position: relative;
	}

	.selector-btn {
		display: flex;
		align-items: center;
		gap: 6px;
		padding: 6px 10px;
		border: 1px solid #e5e5e5;
		background: white;
		border-radius: 8px;
		cursor: pointer;
		font-size: 13px;
		color: #666;
		transition: all 0.15s;
	}

	.selector-btn:hover {
		border-color: #ccc;
		color: #333;
	}

	.selector-btn.selected {
		background: #f0f0ff;
		border-color: #c7c7ff;
		color: #5b5bd6;
	}

	.required-dot {
		width: 5px;
		height: 5px;
		background: #ef4444;
		border-radius: 50%;
		flex-shrink: 0;
	}

	.selector-btn svg {
		width: 14px;
		height: 14px;
	}

	.selector-btn .chevron {
		width: 12px;
		height: 12px;
		opacity: 0.5;
	}

	.dropdown-menu {
		position: absolute;
		bottom: 100%;
		left: 0;
		margin-bottom: 6px;
		min-width: 180px;
		background: white;
		border: 1px solid #e5e5e5;
		border-radius: 12px;
		box-shadow: 0 4px 16px rgba(0, 0, 0, 0.12);
		overflow: hidden;
		z-index: 100;
	}

	.dropdown-item {
		width: 100%;
		padding: 10px 12px;
		border: none;
		background: none;
		text-align: left;
		font-size: 13px;
		color: #333;
		cursor: pointer;
		display: flex;
		align-items: center;
		justify-content: space-between;
		transition: background 0.1s;
	}

	.dropdown-item:hover,
	.dropdown-item.highlighted {
		background: #f5f5f5;
	}

	.dropdown-item.active {
		background: #f0f0ff;
		color: #5b5bd6;
	}

	.dropdown-item.active.highlighted {
		background: #e0e0ff;
	}

	.dropdown-item.create-new {
		display: flex;
		align-items: center;
		gap: 8px;
		color: #3b82f6;
		border-top: 1px solid #eee;
		margin-top: 4px;
		padding-top: 12px;
	}

	.dropdown-item.create-new:hover {
		background: #eff6ff;
	}

	.dropdown-item.create-new svg {
		width: 14px;
		height: 14px;
	}

	/* Dark mode */
	:global(.dark) .selector-btn {
		background: #2c2c2e;
		border-color: rgba(255, 255, 255, 0.12);
		color: #a1a1a6;
	}

	:global(.dark) .selector-btn:hover {
		border-color: rgba(255, 255, 255, 0.2);
		color: #f5f5f7;
	}

	:global(.dark) .selector-btn.selected {
		background: rgba(10, 132, 255, 0.2);
		border-color: rgba(10, 132, 255, 0.4);
		color: #0a84ff;
	}

	:global(.dark) .dropdown-menu {
		background: #2c2c2e;
		border-color: rgba(255, 255, 255, 0.12);
		box-shadow: 0 4px 16px rgba(0, 0, 0, 0.4);
	}

	:global(.dark) .dropdown-item {
		color: #f5f5f7;
	}

	:global(.dark) .dropdown-item:hover,
	:global(.dark) .dropdown-item.highlighted {
		background: #3a3a3c;
	}

	:global(.dark) .dropdown-item.active {
		background: rgba(10, 132, 255, 0.2);
		color: #0a84ff;
	}

	:global(.dark) .dropdown-item.create-new {
		border-top-color: rgba(255, 255, 255, 0.1);
		color: #0a84ff;
	}

	:global(.dark) .dropdown-item.create-new:hover {
		background: rgba(10, 132, 255, 0.1);
	}
</style>
