<script lang="ts">
	/**
	 * Settings Panel - Knowledge Base Settings
	 * Configuration options for the Pages/Knowledge Base module
	 */
	import { Moon, Sun, Monitor, Type, FileText, Trash2, Download, Upload } from 'lucide-svelte';
	import { Button, Separator } from '$lib/ui';
	import { kbSettings } from '../../stores/settings';

	interface Props {
		onClose?: () => void;
	}

	let { onClose }: Props = $props();

	// Bind to the persisted settings store
	let fontSize = $state<'small' | 'default' | 'large'>($kbSettings.fontSize);
	let pageWidth = $state<'narrow' | 'default' | 'wide' | 'full'>($kbSettings.pageWidth);
	let autoSaveDelay = $state($kbSettings.autoSaveDelay);
	let trashRetention = $state($kbSettings.trashRetention);

	// Sync local → store whenever values change
	$effect(() => { kbSettings.setFontSize(fontSize); });
	$effect(() => { kbSettings.setPageWidth(pageWidth); });
	$effect(() => { kbSettings.setAutoSaveDelay(autoSaveDelay); });
	$effect(() => { kbSettings.setTrashRetention(trashRetention); });

	const theme = 'system'; // Themes are managed at the app level

	// Theme options — display only, KB settings does not control app-level theme
	const themeOptions = [
		{ id: 'light', label: 'Light', icon: Sun },
		{ id: 'dark', label: 'Dark', icon: Moon },
		{ id: 'system', label: 'System', icon: Monitor }
	] as const;

	let activeTheme = $state<string>(theme);

	// Font size options
	const fontSizeOptions = [
		{ id: 'small', label: 'Small', size: '14px' },
		{ id: 'default', label: 'Default', size: '16px' },
		{ id: 'large', label: 'Large', size: '18px' }
	] as const;

	// Page width options
	const pageWidthOptions = [
		{ id: 'narrow', label: 'Narrow', width: '680px' },
		{ id: 'default', label: 'Default', width: '900px' },
		{ id: 'wide', label: 'Wide', width: '1200px' },
		{ id: 'full', label: 'Full Width', width: '100%' }
	] as const;

	function handleExport() {
		// TODO: Implement export functionality
		if (import.meta.env.DEV) console.log('Export data');
	}

	function handleImport() {
		// TODO: Implement import functionality
		if (import.meta.env.DEV) console.log('Import data');
	}

	function handleEmptyTrash() {
		if (confirm('Are you sure you want to permanently delete all items in trash?')) {
			// TODO: Implement empty trash
			if (import.meta.env.DEV) console.log('Empty trash');
		}
	}
</script>

<div class="settings-panel">
	<!-- Appearance Section -->
	<section class="settings-section">
		<h3 class="settings-section__title">Appearance</h3>

		<div class="settings-item">
			<div class="settings-item__info">
				<span class="settings-item__label">Theme</span>
				<span class="settings-item__description">Choose your preferred color scheme</span>
			</div>
			<div class="settings-item__control">
				<div class="settings-toggle-group">
					{#each themeOptions as option}
						<button
							class="settings-toggle"
							class:settings-toggle--active={activeTheme === option.id}
							onclick={() => activeTheme = option.id}
						>
							{#if option.id === 'light'}
								<Sun class="h-4 w-4" />
							{:else if option.id === 'dark'}
								<Moon class="h-4 w-4" />
							{:else}
								<Monitor class="h-4 w-4" />
							{/if}
							<span>{option.label}</span>
						</button>
					{/each}
				</div>
			</div>
		</div>

		<div class="settings-item">
			<div class="settings-item__info">
				<span class="settings-item__label">Font Size</span>
				<span class="settings-item__description">Adjust the default text size</span>
			</div>
			<div class="settings-item__control">
				<div class="settings-toggle-group">
					{#each fontSizeOptions as option}
						<button
							class="settings-toggle settings-toggle--compact"
							class:settings-toggle--active={fontSize === option.id}
							onclick={() => fontSize = option.id}
						>
							<span style="font-size: {option.size}">{option.label}</span>
						</button>
					{/each}
				</div>
			</div>
		</div>
	</section>

	<Separator />

	<!-- Editor Section -->
	<section class="settings-section">
		<h3 class="settings-section__title">Editor</h3>

		<div class="settings-item">
			<div class="settings-item__info">
				<span class="settings-item__label">Page Width</span>
				<span class="settings-item__description">Default width for new pages</span>
			</div>
			<div class="settings-item__control">
				<select
					class="settings-select"
					bind:value={pageWidth}
				>
					{#each pageWidthOptions as option}
						<option value={option.id}>{option.label} ({option.width})</option>
					{/each}
				</select>
			</div>
		</div>

		<div class="settings-item">
			<div class="settings-item__info">
				<span class="settings-item__label">Auto-save Delay</span>
				<span class="settings-item__description">Time before changes are saved (ms)</span>
			</div>
			<div class="settings-item__control">
				<input
					type="number"
					class="settings-input"
					bind:value={autoSaveDelay}
					min="500"
					max="10000"
					step="500"
				/>
			</div>
		</div>
	</section>

	<Separator />

	<!-- Data Management Section -->
	<section class="settings-section">
		<h3 class="settings-section__title">Data Management</h3>

		<div class="settings-item">
			<div class="settings-item__info">
				<span class="settings-item__label">Trash Retention</span>
				<span class="settings-item__description">Days before auto-deletion (0 = never)</span>
			</div>
			<div class="settings-item__control">
				<input
					type="number"
					class="settings-input"
					bind:value={trashRetention}
					min="0"
					max="365"
				/>
			</div>
		</div>

		<div class="settings-item settings-item--actions">
			<Button variant="secondary" onclick={handleExport}>
				{#snippet prefix()}
					<Download class="h-4 w-4" />
				{/snippet}
				Export All Data
			</Button>
			<Button variant="secondary" onclick={handleImport}>
				{#snippet prefix()}
					<Upload class="h-4 w-4" />
				{/snippet}
				Import Data
			</Button>
			<Button variant="error" onclick={handleEmptyTrash}>
				{#snippet prefix()}
					<Trash2 class="h-4 w-4" />
				{/snippet}
				Empty Trash
			</Button>
		</div>
	</section>
</div>

<style>
	.settings-panel {
		display: flex;
		flex-direction: column;
		gap: 1.5rem;
		padding: 0.5rem 0;
	}

	.settings-section {
		display: flex;
		flex-direction: column;
		gap: 1rem;
	}

	.settings-section__title {
		font-size: 0.875rem;
		font-weight: 600;
		color: var(--dt);
		margin: 0;
	}

	.settings-item {
		display: flex;
		align-items: flex-start;
		justify-content: space-between;
		gap: 1rem;
	}

	.settings-item--actions {
		flex-direction: column;
		align-items: stretch;
		gap: 0.5rem;
	}

	.settings-item__info {
		display: flex;
		flex-direction: column;
		gap: 0.25rem;
	}

	.settings-item__label {
		font-size: 0.875rem;
		font-weight: 500;
		color: var(--dt);
	}

	.settings-item__description {
		font-size: 0.75rem;
		color: var(--dt3);
	}

	.settings-item__control {
		flex-shrink: 0;
	}

	/* Toggle Group */
	.settings-toggle-group {
		display: flex;
		background: var(--dbg2);
		border-radius: 0.5rem;
		padding: 0.25rem;
		gap: 0.25rem;
	}

	.settings-toggle {
		display: flex;
		align-items: center;
		gap: 0.375rem;
		padding: 0.375rem 0.75rem;
		border: none;
		border-radius: 0.375rem;
		background: transparent;
		color: var(--dt3);
		font-size: 0.75rem;
		font-weight: 500;
		cursor: pointer;
		transition: all 0.15s;
	}

	.settings-toggle:hover {
		color: var(--dt);
	}

	.settings-toggle--active {
		background: var(--dbg);
		color: var(--dt);
		box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
	}

	.settings-toggle--compact {
		padding: 0.375rem 0.5rem;
	}

	/* Select */
	.settings-select {
		min-width: 180px;
		padding: 0.5rem 0.75rem;
		border: 1px solid var(--dbd);
		border-radius: 0.375rem;
		background: var(--dbg);
		color: var(--dt);
		font-size: 0.875rem;
		cursor: pointer;
	}

	.settings-select:focus {
		outline: none;
		border-color: #1e96eb;
		box-shadow: 0 0 0 2px rgba(30, 150, 235, 0.2);
	}

	/* Input */
	.settings-input {
		width: 100px;
		padding: 0.5rem 0.75rem;
		border: 1px solid var(--dbd);
		border-radius: 0.375rem;
		background: var(--dbg);
		color: var(--dt);
		font-size: 0.875rem;
		text-align: right;
	}

	.settings-input:focus {
		outline: none;
		border-color: #1e96eb;
		box-shadow: 0 0 0 2px rgba(30, 150, 235, 0.2);
	}
</style>
