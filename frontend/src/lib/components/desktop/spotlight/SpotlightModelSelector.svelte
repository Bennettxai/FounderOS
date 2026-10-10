<script lang="ts">
	import { fly } from 'svelte/transition';
	import type { AIModel } from './spotlightSearch.ts';

	interface Props {
		availableModels: AIModel[];
		selectedModel: string;
		activeProvider: string;
		showModelDropdown: boolean;
		currentModelName: string;
		onModelSelect: (id: string) => void;
		onToggleModelDropdown: () => void;
	}

	let {
		availableModels,
		selectedModel,
		activeProvider,
		showModelDropdown,
		currentModelName,
		onModelSelect,
		onToggleModelDropdown
	}: Props = $props();
</script>

<div class="dropdown-wrapper">
	<button
		class="icon-btn"
		onclick={onToggleModelDropdown}
		title={currentModelName}
		aria-label="Select AI model: {currentModelName}"
		aria-expanded={showModelDropdown}
	>
		<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
			<rect x="4" y="4" width="16" height="16" rx="2" />
			<circle cx="9" cy="9" r="1.5" fill="currentColor" />
			<circle cx="15" cy="9" r="1.5" fill="currentColor" />
			<path d="M9 15h6" />
		</svg>
	</button>
	{#if showModelDropdown}
		<div class="dropdown-menu model-menu" transition:fly={{ y: 5, duration: 150 }}>
			<div class="dropdown-header">
				Provider: {activeProvider === 'ollama_local' ? 'Local' : activeProvider}
			</div>
			{#if availableModels.length === 0}
				<div class="dropdown-empty">No models available</div>
			{:else}
				{#each availableModels as model}
					<button
						class="dropdown-item"
						class:active={selectedModel === model.id}
						onclick={() => onModelSelect(model.id)}
					>
						<span class="model-name">{model.name}</span>
						{#if model.size}
							<span class="model-size">{model.size}</span>
						{/if}
					</button>
				{/each}
			{/if}
		</div>
	{/if}
</div>

<style>
	.dropdown-wrapper {
		position: relative;
	}

	.icon-btn {
		width: 36px;
		height: 36px;
		border: none;
		background: transparent;
		border-radius: 10px;
		cursor: pointer;
		display: flex;
		align-items: center;
		justify-content: center;
		color: #888;
		transition: all 0.15s;
	}

	.icon-btn:hover {
		background: #f3f3f3;
		color: #333;
	}

	.icon-btn:active {
		transform: scale(0.95);
	}

	.icon-btn svg {
		width: 18px;
		height: 18px;
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

	.dropdown-menu.model-menu {
		min-width: 220px;
	}

	.dropdown-header {
		padding: 8px 12px;
		font-size: 11px;
		color: #666;
		background: #f9f9f9;
		border-bottom: 1px solid #eee;
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

	.dropdown-item:hover {
		background: #f5f5f5;
	}

	.dropdown-item.active {
		background: #f0f0ff;
		color: #5b5bd6;
	}

	.dropdown-empty {
		padding: 16px;
		text-align: center;
		color: #999;
		font-size: 13px;
	}

	.model-name {
		flex: 1;
	}

	.model-size {
		font-size: 11px;
		color: #999;
	}

	/* Dark mode */
	:global(.dark) .icon-btn {
		color: #a1a1a6;
	}

	:global(.dark) .icon-btn:hover {
		background: #2c2c2e;
		color: #f5f5f7;
	}

	:global(.dark) .dropdown-menu {
		background: #2c2c2e;
		border-color: rgba(255, 255, 255, 0.12);
		box-shadow: 0 4px 16px rgba(0, 0, 0, 0.4);
	}

	:global(.dark) .dropdown-header {
		background: #1c1c1e;
		border-bottom-color: rgba(255, 255, 255, 0.1);
		color: #6e6e73;
	}

	:global(.dark) .dropdown-item {
		color: #f5f5f7;
	}

	:global(.dark) .dropdown-item:hover {
		background: #3a3a3c;
	}

	:global(.dark) .dropdown-item.active {
		background: rgba(10, 132, 255, 0.2);
		color: #0a84ff;
	}

	:global(.dark) .dropdown-empty {
		color: #6e6e73;
	}

	:global(.dark) .model-size {
		color: #6e6e73;
	}
</style>
