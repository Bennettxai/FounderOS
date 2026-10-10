<script lang="ts" module>
	// SVG paths for commonly used page icons (Lucide icons)
	// Exported for use in other components
	export const iconLibrary: Record<string, string> = {
		// Documents & Files
		'FileText': 'M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z M14 2v6h6 M16 13H8 M16 17H8 M10 9H8',
		'File': 'M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z M14 2v6h6',
		'Folder': 'M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z',
		'FolderOpen': 'M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z M2 10h20',
		'Archive': 'M21 8v13H3V8 M1 3h22v5H1z M10 12h4',
		'Book': 'M4 19.5A2.5 2.5 0 0 1 6.5 17H20 M4 4.5A2.5 2.5 0 0 1 6.5 2H20v20H6.5A2.5 2.5 0 0 1 4 19.5z',
		'BookOpen': 'M2 3h6a4 4 0 0 1 4 4v14a3 3 0 0 0-3-3H2z M22 3h-6a4 4 0 0 0-4 4v14a3 3 0 0 1 3-3h7z',
		'Notebook': 'M2 3h6a4 4 0 0 1 4 4v14a3 3 0 0 0-3-3H2z',
		'ClipboardList': 'M16 4h2a2 2 0 0 1 2 2v14a2 2 0 0 1-2 2H6a2 2 0 0 1-2-2V6a2 2 0 0 1 2-2h2 M15 2H9a1 1 0 0 0-1 1v2a1 1 0 0 0 1 1h6a1 1 0 0 0 1-1V3a1 1 0 0 0-1-1z M12 11h4 M12 16h4 M8 11h.01 M8 16h.01',

		// Business & Work
		'Building': 'M6 22V4a2 2 0 0 1 2-2h8a2 2 0 0 1 2 2v18Z M6 12H4a2 2 0 0 0-2 2v6a2 2 0 0 0 2 2h2 M18 9h2a2 2 0 0 1 2 2v9a2 2 0 0 1-2 2h-2 M10 6h4 M10 10h4 M10 14h4 M10 18h4',
		'Building2': 'M6 22V4a2 2 0 0 1 2-2h8a2 2 0 0 1 2 2v18 M2 14h4 M2 18h4 M2 22h20 M18 14h4 M18 18h4 M10 6h4 M10 10h4 M10 14h4 M10 18h4',
		'Briefcase': 'M20 7H4a2 2 0 0 0-2 2v10a2 2 0 0 0 2 2h16a2 2 0 0 0 2-2V9a2 2 0 0 0-2-2z M16 21V5a2 2 0 0 0-2-2h-4a2 2 0 0 0-2 2v16',
		'Store': 'M3 9l9-7 9 7v11a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V9z M9 22V12h6v10',
		'Landmark': 'M3 22h18 M6 18v-7 M10 18v-7 M14 18v-7 M18 18v-7 M12 7l-9 4h18z M12 2v5',

		// People & Users
		'User': 'M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2 M12 3a4 4 0 1 0 0 8 4 4 0 0 0 0-8z',
		'Users': 'M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2 M23 21v-2a4 4 0 0 0-3-3.87 M16 3.13a4 4 0 0 1 0 7.75 M9 7a4 4 0 1 0 0 8 4 4 0 0 0 0-8z',
		'UserCircle': 'M18 20a6 6 0 0 0-12 0 M12 10a4 4 0 1 0 0-8 4 4 0 0 0 0 8z M12 22c5.523 0 10-4.477 10-10S17.523 2 12 2 2 6.477 2 12s4.477 10 10 10z',
		'Contact': 'M17 18a2 2 0 0 0-2-2H9a2 2 0 0 0-2 2 M12 14a4 4 0 1 0 0-8 4 4 0 0 0 0 8z M19 4H5a2 2 0 0 0-2 2v12a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2V6a2 2 0 0 0-2-2z',

		// Projects & Tasks
		'FolderKanban': 'M4 20h16a2 2 0 0 0 2-2V8a2 2 0 0 0-2-2h-7.93a2 2 0 0 1-1.66-.9l-.82-1.2A2 2 0 0 0 7.93 3H4a2 2 0 0 0-2 2v13c0 1.1.9 2 2 2z M8 10v4 M12 10v2 M16 10v6',
		'Kanban': 'M6 5v11 M6 5v11 M12 5v6 M12 14v2 M18 5v14',
		'CheckSquare': 'M9 11l3 3L22 4 M21 12v7a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h11',
		'ListTodo': 'M3 6h18 M3 12h18 M3 18h18 M6 6v.01 M6 12v.01 M6 18v.01',
		'Target': 'M12 22c5.523 0 10-4.477 10-10S17.523 2 12 2 2 6.477 2 12s4.477 10 10 10z M12 18a6 6 0 1 0 0-12 6 6 0 0 0 0 12z M12 14a2 2 0 1 0 0-4 2 2 0 0 0 0 4z',

		// Ideas & Creative
		'Lightbulb': 'M9 18h6 M10 22h4 M12 2v1 M12 6a4 4 0 0 0-4 4c0 2.5 2 4 2 6h4c0-2 2-3.5 2-6a4 4 0 0 0-4-4z',
		'Sparkles': 'M12 3l1.54 4.32L18 9l-4.46 1.68L12 15l-1.54-4.32L6 9l4.46-1.68L12 3z M5 19l.77 2.16L8 22l-2.23.84L5 25l-.77-2.16L2 22l2.23-.84L5 19z M19 17l.77 2.16L22 20l-2.23.84L19 23l-.77-2.16L16 20l2.23-.84L19 17z',
		'Wand': 'M15 4V2 M15 16v-2 M8 9h2 M20 9h2 M17.8 11.8L19 13 M17.8 6.2L19 5 M3 21l9-9 M12.2 6.2L11 5',
		'Palette': 'M12 2.69l5.66 5.66a8 8 0 1 1-11.31 0L12 2.69z',
		'PenTool': 'M12 19l7-7 3 3-7 7-3-3z M18 13l-1.5-7.5L2 2l3.5 14.5L13 18l5-5z M2 2l7.586 7.586 M11 11a2 2 0 1 0 0-4 2 2 0 0 0 0 4z',

		// Communication
		'MessageSquare': 'M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z',
		'MessageCircle': 'M21 11.5a8.38 8.38 0 0 1-.9 3.8 8.5 8.5 0 0 1-7.6 4.7 8.38 8.38 0 0 1-3.8-.9L3 21l1.9-5.7a8.38 8.38 0 0 1-.9-3.8 8.5 8.5 0 0 1 4.7-7.6 8.38 8.38 0 0 1 3.8-.9h.5a8.48 8.48 0 0 1 8 8v.5z',
		'Mail': 'M4 4h16c1.1 0 2 .9 2 2v12c0 1.1-.9 2-2 2H4c-1.1 0-2-.9-2-2V6c0-1.1.9-2 2-2z M22 6l-10 7L2 6',
		'AtSign': 'M16 8a6 6 0 0 1 .87 11.96A6 6 0 0 1 6 14a6 6 0 0 1 6-6v6 M22 12a10 10 0 1 0-5 8.66',

		// Calendar & Time
		'Calendar': 'M19 4H5a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2V6a2 2 0 0 0-2-2z M16 2v4 M8 2v4 M3 10h18',
		'CalendarDays': 'M8 2v4 M16 2v4 M3 10h18 M21 6v14a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V6a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z M8 14h.01 M12 14h.01 M16 14h.01 M8 18h.01 M12 18h.01 M16 18h.01',
		'Clock': 'M12 22c5.523 0 10-4.477 10-10S17.523 2 12 2 2 6.477 2 12s4.477 10 10 10z M12 6v6l4 2',
		'Timer': 'M10 2h4 M12 14a4 4 0 1 0 0-8 4 4 0 0 0 0 8z M12 14v-2 M12 22c4.97 0 9-4.03 9-9s-4.03-9-9-9-9 4.03-9 9 4.03 9 9 9z',

		// Data & Analytics
		'BarChart': 'M12 20V10 M18 20V4 M6 20v-4',
		'LineChart': 'M3 3v18h18 M18 9l-5 5-4-4-6 6',
		'PieChart': 'M21.21 15.89A10 10 0 1 1 8 2.83 M22 12A10 10 0 0 0 12 2v10z',
		'Activity': 'M22 12h-4l-3 9L9 3l-3 9H2',
		'TrendingUp': 'M23 6l-9.5 9.5-5-5L1 18 M17 6h6v6',
		'Database': 'M21 5c0 1.657-4.03 3-9 3S3 6.657 3 5s4.03-3 9-3 9 1.343 9 3z M21 12c0 1.66-4 3-9 3s-9-1.34-9-3 M3 5v14c0 1.66 4 3 9 3s9-1.34 9-3V5',

		// Tech & Code
		'Code': 'M16 18l6-6-6-6 M8 6l-6 6 6 6',
		'Terminal': 'M4 17l6-6-6-6 M12 19h8',
		'Cpu': 'M18 4H6a2 2 0 0 0-2 2v12a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V6a2 2 0 0 0-2-2z M9 9h6v6H9z M9 1v3 M15 1v3 M9 20v3 M15 20v3 M20 9h3 M20 14h3 M1 9h3 M1 14h3',
		'Globe': 'M12 22c5.523 0 10-4.477 10-10S17.523 2 12 2 2 6.477 2 12s4.477 10 10 10z M2 12h20 M12 2a15.3 15.3 0 0 1 4 10 15.3 15.3 0 0 1-4 10 15.3 15.3 0 0 1-4-10 15.3 15.3 0 0 1 4-10z',
		'Settings': 'M12 15a3 3 0 1 0 0-6 3 3 0 0 0 0 6z M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1-2.83 2.83l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-4 0v-.09a1.65 1.65 0 0 0-1-1.51 1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 1 1-2.83-2.83l.06-.06a1.65 1.65 0 0 0 .33-1.82 1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1 0-4h.09a1.65 1.65 0 0 0 1.51-1 1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 1 1 2.83-2.83l.06.06a1.65 1.65 0 0 0 1.82.33h.09a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 4 0v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82v.09a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 0 4h-.09a1.65 1.65 0 0 0-1.51 1z',

		// Money & Finance
		'DollarSign': 'M12 1v22 M17 5H9.5a3.5 3.5 0 0 0 0 7h5a3.5 3.5 0 0 1 0 7H6',
		'CreditCard': 'M21 4H3a2 2 0 0 0-2 2v12a2 2 0 0 0 2 2h18a2 2 0 0 0 2-2V6a2 2 0 0 0-2-2z M1 10h22',
		'Wallet': 'M17 14h.01 M21 8V6a2 2 0 0 0-2-2H5a2 2 0 0 0-2 2v12a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-2 M21 12h-6a2 2 0 0 0 0 4h6v-4z',
		'Receipt': 'M4 2v20l2-1 2 1 2-1 2 1 2-1 2 1 2-1 2 1V2l-2 1-2-1-2 1-2-1-2 1-2-1-2 1-2-1z M16 8h-6 M16 12H8 M12 16H8',

		// Objects & Misc
		'Star': 'M12 2l3.09 6.26L22 9.27l-5 4.87 1.18 6.88L12 17.77l-6.18 3.25L7 14.14 2 9.27l6.91-1.01L12 2z',
		'Heart': 'M20.84 4.61a5.5 5.5 0 0 0-7.78 0L12 5.67l-1.06-1.06a5.5 5.5 0 0 0-7.78 7.78l1.06 1.06L12 21.23l7.78-7.78 1.06-1.06a5.5 5.5 0 0 0 0-7.78z',
		'Bookmark': 'M19 21l-7-5-7 5V5a2 2 0 0 1 2-2h10a2 2 0 0 1 2 2z',
		'Flag': 'M4 15s1-1 4-1 5 2 8 2 4-1 4-1V3s-1 1-4 1-5-2-8-2-4 1-4 1z M4 22v-7',
		'Award': 'M12 15a7 7 0 1 0 0-14 7 7 0 0 0 0 14z M8.21 13.89L7 23l5-3 5 3-1.21-9.12',
		'Gift': 'M20 12v10H4V12 M2 7h20v5H2z M12 22V7 M12 7H7.5a2.5 2.5 0 0 1 0-5C11 2 12 7 12 7z M12 7h4.5a2.5 2.5 0 0 0 0-5C13 2 12 7 12 7z',
		'Key': 'M21 2l-2 2m-7.61 7.61a5.5 5.5 0 1 1-7.778 7.778 5.5 5.5 0 0 1 7.777-7.777zm0 0L15.5 7.5m0 0l3 3L22 7l-3-3m-3.5 3.5L19 4',
		'Lock': 'M19 11H5a2 2 0 0 0-2 2v7a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7a2 2 0 0 0-2-2z M7 11V7a5 5 0 0 1 10 0v4',
		'Rocket': 'M4.5 16.5c-1.5 1.26-2 5-2 5s3.74-.5 5-2c.71-.84.7-2.13-.09-2.91a2.18 2.18 0 0 0-2.91-.09z M12 15l-3-3a22 22 0 0 1 2-3.95A12.88 12.88 0 0 1 22 2c0 2.72-.78 7.5-6 11a22.35 22.35 0 0 1-4 2z M9 12H4s.55-3.03 2-4c1.62-1.08 5 0 5 0 M12 15v5s3.03-.55 4-2c1.08-1.62 0-5 0-5',
		'Zap': 'M13 2L3 14h9l-1 8 10-12h-9l1-8z',
		'Layers': 'M12 2L2 7l10 5 10-5-10-5z M2 17l10 5 10-5 M2 12l10 5 10-5',
		'Layout': 'M19 3H5a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2V5a2 2 0 0 0-2-2z M3 9h18 M9 21V9',
		'Grid': 'M10 3H3v7h7V3z M21 3h-7v7h7V3z M21 14h-7v7h7v-7z M10 14H3v7h7v-7z',
		'Table': 'M9 3H5a2 2 0 0 0-2 2v4m6-6h10a2 2 0 0 1 2 2v4M9 3v18m0 0h10a2 2 0 0 0 2-2V9M9 21H5a2 2 0 0 1-2-2V9m0 0h18',
		'Home': 'M3 9l9-7 9 7v11a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z M9 22V12h6v10',
		'Map': 'M1 6v16l7-4 8 4 7-4V2l-7 4-8-4-7 4z M8 2v16 M16 6v16',
		'Compass': 'M12 22c5.523 0 10-4.477 10-10S17.523 2 12 2 2 6.477 2 12s4.477 10 10 10z M16.24 7.76l-2.12 6.36-6.36 2.12 2.12-6.36 6.36-2.12z',
		'Image': 'M19 3H5a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2V5a2 2 0 0 0-2-2z M8.5 10a1.5 1.5 0 1 0 0-3 1.5 1.5 0 0 0 0 3z M21 15l-5-5L5 21',
		'Camera': 'M23 19a2 2 0 0 1-2 2H3a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h4l2-3h6l2 3h4a2 2 0 0 1 2 2z M12 17a4 4 0 1 0 0-8 4 4 0 0 0 0 8z',
		'Video': 'M23 7l-7 5 7 5V7z M14 5H3a2 2 0 0 0-2 2v10a2 2 0 0 0 2 2h11a2 2 0 0 0 2-2V7a2 2 0 0 0-2-2z',
		'Music': 'M9 18V5l12-2v13 M9 18a3 3 0 1 1-6 0 3 3 0 0 1 6 0z M21 16a3 3 0 1 1-6 0 3 3 0 0 1 6 0z',
		'Headphones': 'M3 18v-6a9 9 0 0 1 18 0v6 M21 19a2 2 0 0 1-2 2h-1a2 2 0 0 1-2-2v-3a2 2 0 0 1 2-2h3z M3 19a2 2 0 0 0 2 2h1a2 2 0 0 0 2-2v-3a2 2 0 0 0-2-2H3z',
		'Coffee': 'M18 8h1a4 4 0 0 1 0 8h-1 M2 8h16v9a4 4 0 0 1-4 4H6a4 4 0 0 1-4-4V8z M6 1v3 M10 1v3 M14 1v3',
	};

	// Helper function to get SVG path for an icon name
	export function getIconPath(name: string): string | undefined {
		return iconLibrary[name];
	}
</script>

<script lang="ts">
	/**
	 * PageIconPicker - SVG icon selector for pages (NO emojis)
	 * Uses Lucide SVG icons only
	 */
	import { Search, X } from 'lucide-svelte';

	interface Props {
		currentIcon?: string | null;
		onSelect?: (iconName: string | null) => void;
	}

	let { currentIcon, onSelect }: Props = $props();

	let searchQuery = $state('');

	const allIconNames = Object.keys(iconLibrary).sort();

	const filteredIcons = $derived(
		searchQuery
			? allIconNames.filter(name => name.toLowerCase().includes(searchQuery.toLowerCase()))
			: allIconNames
	);

	function selectIcon(name: string) {
		onSelect?.(name);
	}

	function removeIcon() {
		onSelect?.(null);
	}
</script>

<div class="page-icon-picker">
	{#if currentIcon}
		<div class="page-icon-picker__header">
			<button class="btn-pill btn-pill-ghost page-icon-picker__remove" onclick={removeIcon}>
				<X class="h-3 w-3" />
				<span>Remove icon</span>
			</button>
		</div>
	{/if}

	<div class="page-icon-picker__search">
		<Search class="page-icon-picker__search-icon" />
		<input
			type="text"
			bind:value={searchQuery}
			placeholder="Search icons..."
			class="page-icon-picker__search-input"
		/>
	</div>

	<div class="page-icon-picker__grid">
		{#each filteredIcons as iconName}
			<button
				class="btn-pill btn-pill-ghost page-icon-picker__icon"
				class:page-icon-picker__icon--selected={currentIcon === iconName}
				onclick={() => selectIcon(iconName)}
				title={iconName}
			>
				<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
					<path d={iconLibrary[iconName]} />
				</svg>
			</button>
		{/each}
	</div>

	{#if filteredIcons.length === 0}
		<div class="page-icon-picker__empty">
			No icons found for "{searchQuery}"
		</div>
	{/if}
</div>

<style>
	.page-icon-picker {
		width: 320px;
		max-height: 400px;
		display: flex;
		flex-direction: column;
	}

	.page-icon-picker__header {
		display: flex;
		justify-content: flex-end;
		padding-bottom: 0.5rem;
		border-bottom: 1px solid hsl(var(--border));
		margin-bottom: 0.5rem;
	}

	.page-icon-picker__remove {
		display: flex;
		align-items: center;
		gap: 0.25rem;
		padding: 0.25rem 0.5rem;
		background: transparent;
		border: none;
		color: hsl(var(--destructive));
		font-size: 0.75rem;
		cursor: pointer;
		border-radius: 0.25rem;
		transition: background-color 0.15s;
	}

	.page-icon-picker__remove:hover {
		background-color: hsl(var(--destructive) / 0.1);
	}

	.page-icon-picker__search {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		padding: 0.5rem;
		margin-bottom: 0.5rem;
		background-color: hsl(var(--muted));
		border-radius: 0.375rem;
	}

	.page-icon-picker__search-icon {
		width: 16px;
		height: 16px;
		color: hsl(var(--muted-foreground));
		flex-shrink: 0;
	}

	.page-icon-picker__search-input {
		flex: 1;
		border: none;
		background: transparent;
		font-size: 0.875rem;
		color: hsl(var(--foreground));
		outline: none;
	}

	.page-icon-picker__search-input::placeholder {
		color: hsl(var(--muted-foreground));
	}

	.page-icon-picker__grid {
		display: grid;
		grid-template-columns: repeat(8, 1fr);
		gap: 0.25rem;
		max-height: 280px;
		overflow-y: auto;
		padding: 0.25rem;
	}

	.page-icon-picker__icon {
		display: flex;
		align-items: center;
		justify-content: center;
		width: 36px;
		height: 36px;
		background: transparent;
		border: none;
		border-radius: 0.375rem;
		color: hsl(var(--foreground));
		cursor: pointer;
		transition: background-color 0.1s;
	}

	.page-icon-picker__icon:hover {
		background-color: hsl(var(--muted));
	}

	.page-icon-picker__icon--selected {
		background-color: hsl(var(--primary) / 0.15);
		color: hsl(var(--primary));
	}

	.page-icon-picker__empty {
		padding: 1rem;
		text-align: center;
		color: hsl(var(--muted-foreground));
		font-size: 0.875rem;
	}
</style>
