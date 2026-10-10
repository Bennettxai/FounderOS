<script lang="ts">
	import { desktopSettings } from '$lib/stores/desktopStore';
	import { onMount } from 'svelte';
	import { browser } from '$app/environment';

	import IconsTab from './settings/IconsTab.svelte';
	import BackgroundTab from './settings/BackgroundTab.svelte';
	import SoundsTab from './settings/SoundsTab.svelte';
	import AnimationsTab from './settings/AnimationsTab.svelte';
	import BootTab from './settings/BootTab.svelte';
	import ShortcutsTab from './settings/ShortcutsTab.svelte';
	import PermissionsTab from './settings/PermissionsTab.svelte';
	import DataTab from './settings/DataTab.svelte';
	import ModulesTab from './settings/ModulesTab.svelte';

	type SettingsTab = 'modules' | 'icons' | 'background' | 'sounds' | 'animations' | 'boot' | 'shortcuts' | 'permissions' | 'data';

	let selectedTab = $state<SettingsTab>('modules');
	let accessibilityGranted = $state(false);
	let isElectron = $state(false);

	// Shared accessibility request handler (used by both Shortcuts and Permissions tabs)
	async function requestAccessibility() {
		if (isElectron && browser) {
			try {
				const electron = (window as any).electron;
				await electron?.shortcuts?.requestAccessibility();
				setTimeout(async () => {
					const result = await electron?.shortcuts?.checkAccessibility();
					accessibilityGranted = result?.granted ?? false;
				}, 1000);
			} catch (e) {
				console.error('Failed to request accessibility:', e);
			}
		}
	}

	onMount(async () => {
		const electron = (window as any).electron;
		if (browser && electron) {
			isElectron = true;
			try {
				const accessResult = await electron.shortcuts?.checkAccessibility();
				accessibilityGranted = accessResult?.granted ?? false;
			} catch (e) {
				console.error('Failed to check accessibility:', e);
			}
		}
	});
</script>

<div class="desktop-settings">
	<!-- Tabs with Icons -->
	<div class="tabs">
		<button
			class="tab"
			class:active={selectedTab === 'modules'}
			onclick={() => selectedTab = 'modules'}
			title="Desktop Modules"
		>
			<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
				<rect x="3" y="3" width="7" height="7" rx="1"/>
				<rect x="14" y="3" width="7" height="7" rx="1"/>
				<rect x="3" y="14" width="7" height="7" rx="1"/>
				<path d="M17.5 15v5M15 17.5h5"/>
			</svg>
			<span>Modules</span>
		</button>
		<button
			class="tab"
			class:active={selectedTab === 'icons'}
			onclick={() => selectedTab = 'icons'}
			title="Icons & Layout"
		>
			<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
				<rect x="3" y="3" width="7" height="7" rx="1"/>
				<rect x="14" y="3" width="7" height="7" rx="1"/>
				<rect x="14" y="14" width="7" height="7" rx="1"/>
				<rect x="3" y="14" width="7" height="7" rx="1"/>
			</svg>
			<span>Icons</span>
		</button>
		<button
			class="tab"
			class:active={selectedTab === 'background'}
			onclick={() => selectedTab = 'background'}
			title="Background & Wallpaper"
		>
			<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
				<rect x="3" y="3" width="18" height="18" rx="2"/>
				<circle cx="8.5" cy="8.5" r="1.5"/>
				<polyline points="21 15 16 10 5 21"/>
			</svg>
			<span>Wallpaper</span>
		</button>
		<button
			class="tab"
			class:active={selectedTab === 'sounds'}
			onclick={() => selectedTab = 'sounds'}
			title="System Sounds"
		>
			<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
				<polygon points="11 5 6 9 2 9 2 15 6 15 11 19 11 5"/>
				<path d="M15.54 8.46a5 5 0 0 1 0 7.07"/>
				<path d="M19.07 4.93a10 10 0 0 1 0 14.14"/>
			</svg>
			<span>Sounds</span>
		</button>
		<button
			class="tab"
			class:active={selectedTab === 'animations'}
			onclick={() => selectedTab = 'animations'}
			title="Effects & Animations"
		>
			<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
				<path d="M12 3v1m0 16v1m9-9h-1M4 12H3"/>
				<path d="M18.364 5.636l-.707.707M6.343 17.657l-.707.707"/>
				<path d="M5.636 5.636l.707.707M17.657 17.657l.707.707"/>
				<circle cx="12" cy="12" r="4"/>
			</svg>
			<span>Effects</span>
		</button>
		<button
			class="tab"
			class:active={selectedTab === 'boot'}
			onclick={() => selectedTab = 'boot'}
			title="Boot Screen"
		>
			<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
				<path d="M12 2v4M12 18v4M4.93 4.93l2.83 2.83M16.24 16.24l2.83 2.83"/>
				<path d="M2 12h4M18 12h4M4.93 19.07l2.83-2.83M16.24 7.76l2.83-2.83"/>
			</svg>
			<span>Boot</span>
		</button>
		<button
			class="tab"
			class:active={selectedTab === 'shortcuts'}
			onclick={() => selectedTab = 'shortcuts'}
			title="Keyboard Shortcuts"
		>
			<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
				<rect x="2" y="4" width="20" height="16" rx="2"/>
				<path d="M6 8h.01M10 8h.01M14 8h.01M18 8h.01"/>
				<path d="M8 12h8M6 16h12"/>
			</svg>
			<span>Shortcuts</span>
		</button>
		{#if isElectron}
			<button
				class="tab"
				class:active={selectedTab === 'permissions'}
				onclick={() => selectedTab = 'permissions'}
				title="System Permissions"
			>
				<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
					<path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"/>
				</svg>
				<span>Permissions</span>
			</button>
		{/if}
		<button
			class="tab"
			class:active={selectedTab === 'data'}
			onclick={() => selectedTab = 'data'}
			title="Import & Export"
		>
			<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
				<path d="M21 15v4a2 2 0 01-2 2H5a2 2 0 01-2-2v-4"/>
				<polyline points="7 10 12 15 17 10"/>
				<line x1="12" y1="15" x2="12" y2="3"/>
			</svg>
			<span>Data</span>
		</button>
	</div>

	<!-- Content -->
	<div class="content">
		{#if selectedTab === 'modules'}
			<ModulesTab />
		{:else if selectedTab === 'icons'}
			<IconsTab />
		{:else if selectedTab === 'background'}
			<BackgroundTab />
		{:else if selectedTab === 'sounds'}
			<SoundsTab />
		{:else if selectedTab === 'animations'}
			<AnimationsTab />
		{:else if selectedTab === 'boot'}
			<BootTab />
		{:else if selectedTab === 'shortcuts'}
			<ShortcutsTab
				{isElectron}
				{accessibilityGranted}
				onRequestAccessibility={requestAccessibility}
			/>
		{:else if selectedTab === 'permissions'}
			<PermissionsTab
				{isElectron}
				{accessibilityGranted}
				onAccessibilityChange={(granted) => (accessibilityGranted = granted)}
			/>
		{:else if selectedTab === 'data'}
			<DataTab />
		{/if}
	</div>

	<!-- Footer -->
	<div class="footer">
		<button class="reset-btn" onclick={() => desktopSettings.reset()}>
			Reset to Defaults
		</button>
	</div>
</div>

<style>
	.desktop-settings {
		display: flex;
		flex-direction: column;
		height: 100%;
		min-height: 0;
		background: #fafafa;
		overflow: hidden;
	}

	.tabs {
		display: flex;
		border-bottom: 1px solid #e5e5e5;
		background: white;
		flex-shrink: 0;
		padding: 0 8px;
		gap: 2px;
		overflow-x: auto;
		scrollbar-width: none;
	}

	.tabs::-webkit-scrollbar {
		display: none;
	}

	.tab {
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: 4px;
		padding: 10px 12px 8px;
		font-size: 10px;
		font-weight: 500;
		color: #888;
		background: none;
		border: none;
		cursor: pointer;
		border-bottom: 2px solid transparent;
		transition: all 0.15s ease;
		white-space: nowrap;
		min-width: fit-content;
	}

	.tab svg {
		width: 18px;
		height: 18px;
		stroke-width: 1.5;
		transition: all 0.15s ease;
	}

	.tab span {
		line-height: 1;
	}

	.tab:hover {
		color: #555;
		background: #f5f5f5;
		border-radius: 6px 6px 0 0;
	}

	.tab:hover svg {
		stroke-width: 2;
	}

	.tab.active {
		color: #111;
		border-bottom-color: #111;
	}

	.tab.active svg {
		stroke-width: 2;
	}

	.content {
		flex: 1;
		min-height: 0;
		overflow-y: auto;
		overscroll-behavior: contain;
		padding: 20px;
	}

	.footer {
		padding: 16px 20px;
		border-top: 1px solid #e5e5e5;
		background: white;
		flex-shrink: 0;
	}

	.reset-btn {
		font-size: 12px;
		color: #666;
		background: none;
		border: none;
		cursor: pointer;
		padding: 8px 12px;
		border-radius: 6px;
		transition: all 0.15s ease;
	}

	.reset-btn:hover {
		background: #f0f0f0;
		color: #333;
	}

</style>
