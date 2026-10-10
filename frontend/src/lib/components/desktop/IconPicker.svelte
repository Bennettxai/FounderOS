<script lang="ts">
	import type { CustomIconConfig } from '$lib/stores/windowStore';

	interface Props {
		currentIcon?: CustomIconConfig;
		defaultColor?: string;
		defaultBgColor?: string;
		onSelect?: (config: CustomIconConfig | undefined) => void;
		onClose?: () => void;
	}

	let {
		currentIcon,
		defaultColor = '#333333',
		defaultBgColor = '#F5F5F5',
		onSelect,
		onClose
	}: Props = $props();

	// State
	let activeTab = $state<'icons' | 'custom'>('icons');
	let searchQuery = $state('');
	let selectedIcon = $state<string | undefined>(currentIcon?.lucideName);
	let customSvg = $state<string | undefined>(currentIcon?.customSvg);
	let foregroundColor = $state(currentIcon?.foregroundColor || defaultColor);
	let backgroundColor = $state(currentIcon?.backgroundColor || defaultBgColor);

	// Curated icon library with SVG paths (most commonly used icons)
	const iconLibrary: Record<string, string> = {
		// Apps & Files
		'Home': 'M3 9l9-7 9 7v11a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z M9 22V12h6v10',
		'Folder': 'M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z',
		'File': 'M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z M14 2v6h6',
		'FileText': 'M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z M14 2v6h6 M16 13H8 M16 17H8 M10 9H8',
		'Files': 'M15.5 2H8.6c-.4 0-.8.2-1.1.5-.3.3-.5.7-.5 1.1v12.8c0 .4.2.8.5 1.1.3.3.7.5 1.1.5h9.8c.4 0 .8-.2 1.1-.5.3-.3.5-.7.5-1.1V6.5L15.5 2z M3 7.6v12.8c0 .4.2.8.5 1.1.3.3.7.5 1.1.5h9.8 M15 2v5h5',
		'Archive': 'M21 8v13H3V8 M1 3h22v5H1z M10 12h4',
		'Box': 'M21 16V8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.73l7 4a2 2 0 0 0 2 0l7-4A2 2 0 0 0 21 16z M3.27 6.96 12 12.01l8.73-5.05 M12 22.08V12',
		'Package': 'M16.5 9.4 7.55 4.24 M21 16V8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.73l7 4a2 2 0 0 0 2 0l7-4A2 2 0 0 0 21 16z M3.27 6.96 12 12.01l8.73-5.05 M12 22.08V12',

		// Communication
		'Mail': 'M4 4h16c1.1 0 2 .9 2 2v12c0 1.1-.9 2-2 2H4c-1.1 0-2-.9-2-2V6c0-1.1.9-2 2-2z M22 6l-10 7L2 6',
		'MessageSquare': 'M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z',
		'MessageCircle': 'M21 11.5a8.38 8.38 0 0 1-.9 3.8 8.5 8.5 0 0 1-7.6 4.7 8.38 8.38 0 0 1-3.8-.9L3 21l1.9-5.7a8.38 8.38 0 0 1-.9-3.8 8.5 8.5 0 0 1 4.7-7.6 8.38 8.38 0 0 1 3.8-.9h.5a8.48 8.48 0 0 1 8 8v.5z',
		'Phone': 'M22 16.92v3a2 2 0 0 1-2.18 2 19.79 19.79 0 0 1-8.63-3.07 19.5 19.5 0 0 1-6-6 19.79 19.79 0 0 1-3.07-8.67A2 2 0 0 1 4.11 2h3a2 2 0 0 1 2 1.72 12.84 12.84 0 0 0 .7 2.81 2 2 0 0 1-.45 2.11L8.09 9.91a16 16 0 0 0 6 6l1.27-1.27a2 2 0 0 1 2.11-.45 12.84 12.84 0 0 0 2.81.7A2 2 0 0 1 22 16.92z',
		'Video': 'M23 7l-7 5 7 5V7z M14 5H3a2 2 0 0 0-2 2v10a2 2 0 0 0 2 2h11a2 2 0 0 0 2-2V7a2 2 0 0 0-2-2z',
		'Bell': 'M18 8A6 6 0 0 0 6 8c0 7-3 9-3 9h18s-3-2-3-9 M13.73 21a2 2 0 0 1-3.46 0',

		// Business
		'Calendar': 'M19 4H5a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2V6a2 2 0 0 0-2-2z M16 2v4 M8 2v4 M3 10h18',
		'Clock': 'M12 22c5.523 0 10-4.477 10-10S17.523 2 12 2 2 6.477 2 12s4.477 10 10 10z M12 6v6l4 2',
		'CheckSquare': 'M9 11l3 3L22 4 M21 12v7a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h11',
		'ClipboardList': 'M16 4h2a2 2 0 0 1 2 2v14a2 2 0 0 1-2 2H6a2 2 0 0 1-2-2V6a2 2 0 0 1 2-2h2 M15 2H9a1 1 0 0 0-1 1v2a1 1 0 0 0 1 1h6a1 1 0 0 0 1-1V3a1 1 0 0 0-1-1z M12 11h4 M12 16h4 M8 11h.01 M8 16h.01',
		'Users': 'M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2 M23 21v-2a4 4 0 0 0-3-3.87 M16 3.13a4 4 0 0 1 0 7.75 M9 7a4 4 0 1 0 0 8 4 4 0 0 0 0-8z',
		'User': 'M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2 M12 3a4 4 0 1 0 0 8 4 4 0 0 0 0-8z',
		'Briefcase': 'M20 7H4a2 2 0 0 0-2 2v10a2 2 0 0 0 2 2h16a2 2 0 0 0 2-2V9a2 2 0 0 0-2-2z M16 21V5a2 2 0 0 0-2-2h-4a2 2 0 0 0-2 2v16',
		'Building': 'M6 22V4a2 2 0 0 1 2-2h8a2 2 0 0 1 2 2v18Z M6 12H4a2 2 0 0 0-2 2v6a2 2 0 0 0 2 2h2 M18 9h2a2 2 0 0 1 2 2v9a2 2 0 0 1-2 2h-2 M10 6h4 M10 10h4 M10 14h4 M10 18h4',
		'DollarSign': 'M12 1v22 M17 5H9.5a3.5 3.5 0 0 0 0 7h5a3.5 3.5 0 0 1 0 7H6',
		'CreditCard': 'M21 4H3a2 2 0 0 0-2 2v12a2 2 0 0 0 2 2h18a2 2 0 0 0 2-2V6a2 2 0 0 0-2-2z M1 10h22',

		// Tech & Development
		'Settings': 'M12 15a3 3 0 1 0 0-6 3 3 0 0 0 0 6z M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1 0 2.83 2 2 0 0 1-2.83 0l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-2 2 2 2 0 0 1-2-2v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83 0 2 2 0 0 1 0-2.83l.06-.06a1.65 1.65 0 0 0 .33-1.82 1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1-2-2 2 2 0 0 1 2-2h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 0-2.83 2 2 0 0 1 2.83 0l.06.06a1.65 1.65 0 0 0 1.82.33H9a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 2-2 2 2 0 0 1 2 2v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 0 2 2 0 0 1 0 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82V9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 2 2 2 2 0 0 1-2 2h-.09a1.65 1.65 0 0 0-1.51 1z',
		'Code': 'M16 18l6-6-6-6 M8 6l-6 6 6 6',
		'Terminal': 'M4 17l6-6-6-6 M12 19h8',
		'Database': 'M21 5c0 1.657-4.03 3-9 3S3 6.657 3 5s4.03-3 9-3 9 1.343 9 3z M21 12c0 1.66-4 3-9 3s-9-1.34-9-3 M3 5v14c0 1.66 4 3 9 3s9-1.34 9-3V5',
		'Server': 'M22 9H2a1 1 0 0 0-1 1v4a1 1 0 0 0 1 1h20a1 1 0 0 0 1-1v-4a1 1 0 0 0-1-1z M6 12h.01 M22 17H2a1 1 0 0 0-1 1v4a1 1 0 0 0 1 1h20a1 1 0 0 0 1-1v-4a1 1 0 0 0-1-1z M6 20h.01 M22 1H2a1 1 0 0 0-1 1v4a1 1 0 0 0 1 1h20a1 1 0 0 0 1-1V2a1 1 0 0 0-1-1z M6 4h.01',
		'Cloud': 'M18 10h-1.26A8 8 0 1 0 9 20h9a5 5 0 0 0 0-10z',
		'Globe': 'M12 22c5.523 0 10-4.477 10-10S17.523 2 12 2 2 6.477 2 12s4.477 10 10 10z M2 12h20 M12 2a15.3 15.3 0 0 1 4 10 15.3 15.3 0 0 1-4 10 15.3 15.3 0 0 1-4-10 15.3 15.3 0 0 1 4-10z',
		'Wifi': 'M5 12.55a11 11 0 0 1 14.08 0 M1.42 9a16 16 0 0 1 21.16 0 M8.53 16.11a6 6 0 0 1 6.95 0 M12 20h.01',
		'Cpu': 'M18 4H6a2 2 0 0 0-2 2v12a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V6a2 2 0 0 0-2-2z M9 9h6v6H9z M9 1v3 M15 1v3 M9 20v3 M15 20v3 M20 9h3 M20 14h3 M1 9h3 M1 14h3',

		// Media & Creative
		'Image': 'M19 3H5a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2V5a2 2 0 0 0-2-2z M8.5 10a1.5 1.5 0 1 0 0-3 1.5 1.5 0 0 0 0 3z M21 15l-5-5L5 21',
		'Camera': 'M23 19a2 2 0 0 1-2 2H3a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h4l2-3h6l2 3h4a2 2 0 0 1 2 2z M12 17a4 4 0 1 0 0-8 4 4 0 0 0 0 8z',
		'Music': 'M9 18V5l12-2v13 M9 18a3 3 0 1 1-6 0 3 3 0 0 1 6 0z M21 16a3 3 0 1 1-6 0 3 3 0 0 1 6 0z',
		'Film': 'M19.82 2H4.18A2.18 2.18 0 0 0 2 4.18v15.64A2.18 2.18 0 0 0 4.18 22h15.64A2.18 2.18 0 0 0 22 19.82V4.18A2.18 2.18 0 0 0 19.82 2z M7 2v20 M17 2v20 M2 12h20 M2 7h5 M2 17h5 M17 17h5 M17 7h5',
		'Play': 'M5 3l14 9-14 9V3z',
		'Mic': 'M12 1a3 3 0 0 0-3 3v8a3 3 0 0 0 6 0V4a3 3 0 0 0-3-3z M19 10v2a7 7 0 0 1-14 0v-2 M12 19v4 M8 23h8',
		'Headphones': 'M3 18v-6a9 9 0 0 1 18 0v6 M21 19a2 2 0 0 1-2 2h-1a2 2 0 0 1-2-2v-3a2 2 0 0 1 2-2h3z M3 19a2 2 0 0 0 2 2h1a2 2 0 0 0 2-2v-3a2 2 0 0 0-2-2H3z',
		'Palette': 'M12 2.69l5.66 5.66a8 8 0 1 1-11.31 0L12 2.69z',

		// Navigation & Actions
		'Search': 'M21 21l-6-6m2-5a7 7 0 1 1-14 0 7 7 0 0 1 14 0z',
		'Plus': 'M12 5v14 M5 12h14',
		'Minus': 'M5 12h14',
		'X': 'M18 6L6 18 M6 6l12 12',
		'Check': 'M20 6L9 17l-5-5',
		'ChevronRight': 'M9 18l6-6-6-6',
		'ChevronDown': 'M6 9l6 6 6-6',
		'ArrowRight': 'M5 12h14 M12 5l7 7-7 7',
		'ArrowLeft': 'M19 12H5 M12 19l-7-7 7-7',
		'ArrowUp': 'M12 19V5 M5 12l7-7 7 7',
		'ArrowDown': 'M12 5v14 M19 12l-7 7-7-7',
		'ExternalLink': 'M18 13v6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h6 M15 3h6v6 M10 14L21 3',
		'Download': 'M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4 M7 10l5 5 5-5 M12 15V3',
		'Upload': 'M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4 M17 8l-5-5-5 5 M12 3v12',
		'Refresh': 'M23 4v6h-6 M1 20v-6h6 M3.51 9a9 9 0 0 1 14.85-3.36L23 10 M1 14l4.64 4.36A9 9 0 0 0 20.49 15',
		'MoreHorizontal': 'M12 13a1 1 0 1 0 0-2 1 1 0 0 0 0 2z M19 13a1 1 0 1 0 0-2 1 1 0 0 0 0 2z M5 13a1 1 0 1 0 0-2 1 1 0 0 0 0 2z',
		'Menu': 'M3 12h18 M3 6h18 M3 18h18',

		// Objects & Misc
		'Heart': 'M20.84 4.61a5.5 5.5 0 0 0-7.78 0L12 5.67l-1.06-1.06a5.5 5.5 0 0 0-7.78 7.78l1.06 1.06L12 21.23l7.78-7.78 1.06-1.06a5.5 5.5 0 0 0 0-7.78z',
		'Star': 'M12 2l3.09 6.26L22 9.27l-5 4.87 1.18 6.88L12 17.77l-6.18 3.25L7 14.14 2 9.27l6.91-1.01L12 2z',
		'Bookmark': 'M19 21l-7-5-7 5V5a2 2 0 0 1 2-2h10a2 2 0 0 1 2 2z',
		'Flag': 'M4 15s1-1 4-1 5 2 8 2 4-1 4-1V3s-1 1-4 1-5-2-8-2-4 1-4 1z M4 22v-7',
		'Award': 'M12 15a7 7 0 1 0 0-14 7 7 0 0 0 0 14z M8.21 13.89L7 23l5-3 5 3-1.21-9.12',
		'Gift': 'M20 12v10H4V12 M2 7h20v5H2z M12 22V7 M12 7H7.5a2.5 2.5 0 0 1 0-5C11 2 12 7 12 7z M12 7h4.5a2.5 2.5 0 0 0 0-5C13 2 12 7 12 7z',
		'Key': 'M21 2l-2 2m-7.61 7.61a5.5 5.5 0 1 1-7.778 7.778 5.5 5.5 0 0 1 7.777-7.777zm0 0L15.5 7.5m0 0l3 3L22 7l-3-3m-3.5 3.5L19 4',
		'Lock': 'M19 11H5a2 2 0 0 0-2 2v7a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7a2 2 0 0 0-2-2z M7 11V7a5 5 0 0 1 10 0v4',
		'Unlock': 'M19 11H5a2 2 0 0 0-2 2v7a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7a2 2 0 0 0-2-2z M7 11V7a5 5 0 0 1 9.9-1',
		'Shield': 'M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z',
		'Zap': 'M13 2L3 14h9l-1 8 10-12h-9l1-8z',
		'Target': 'M12 22c5.523 0 10-4.477 10-10S17.523 2 12 2 2 6.477 2 12s4.477 10 10 10z M12 18a6 6 0 1 0 0-12 6 6 0 0 0 0 12z M12 14a2 2 0 1 0 0-4 2 2 0 0 0 0 4z',
		'Compass': 'M12 22c5.523 0 10-4.477 10-10S17.523 2 12 2 2 6.477 2 12s4.477 10 10 10z M16.24 7.76l-2.12 6.36-6.36 2.12 2.12-6.36 6.36-2.12z',
		'Map': 'M1 6v16l7-4 8 4 7-4V2l-7 4-8-4-7 4z M8 2v16 M16 6v16',
		'Layers': 'M12 2L2 7l10 5 10-5-10-5z M2 17l10 5 10-5 M2 12l10 5 10-5',
		'Layout': 'M19 3H5a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2V5a2 2 0 0 0-2-2z M3 9h18 M9 21V9',
		'Grid': 'M10 3H3v7h7V3z M21 3h-7v7h7V3z M21 14h-7v7h7v-7z M10 14H3v7h7v-7z',
		'List': 'M8 6h13 M8 12h13 M8 18h13 M3 6h.01 M3 12h.01 M3 18h.01',
		'Activity': 'M22 12h-4l-3 9L9 3l-3 9H2',
		'TrendingUp': 'M23 6l-9.5 9.5-5-5L1 18 M17 6h6v6',
		'BarChart': 'M12 20V10 M18 20V4 M6 20v-4',
		'PieChart': 'M21.21 15.89A10 10 0 1 1 8 2.83 M22 12A10 10 0 0 0 12 2v10z',
		'Trash': 'M3 6h18 M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2',
		'Edit': 'M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7 M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z',
		'Copy': 'M20 9h-9a2 2 0 0 0-2 2v9a2 2 0 0 0 2 2h9a2 2 0 0 0 2-2v-9a2 2 0 0 0-2-2z M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1',
		'Save': 'M19 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h11l5 5v11a2 2 0 0 1-2 2z M17 21v-8H7v8 M7 3v5h8',
		'Printer': 'M6 9V2h12v7 M6 18H4a2 2 0 0 1-2-2v-5a2 2 0 0 1 2-2h16a2 2 0 0 1 2 2v5a2 2 0 0 1-2 2h-2 M6 14h12v8H6z',
		'Send': 'M22 2L11 13 M22 2l-7 20-4-9-9-4 20-7z',
		'Share': 'M4 12v8a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2v-8 M16 6l-4-4-4 4 M12 2v13',
		'Link': 'M10 13a5 5 0 0 0 7.54.54l3-3a5 5 0 0 0-7.07-7.07l-1.72 1.71 M14 11a5 5 0 0 0-7.54-.54l-3 3a5 5 0 0 0 7.07 7.07l1.71-1.71',
		'Eye': 'M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z M12 15a3 3 0 1 0 0-6 3 3 0 0 0 0 6z',
		'EyeOff': 'M17.94 17.94A10.07 10.07 0 0 1 12 20c-7 0-11-8-11-8a18.45 18.45 0 0 1 5.06-5.94M9.9 4.24A9.12 9.12 0 0 1 12 4c7 0 11 8 11 8a18.5 18.5 0 0 1-2.16 3.19m-6.72-1.07a3 3 0 1 1-4.24-4.24 M1 1l22 22',
		'Filter': 'M22 3H2l8 9.46V19l4 2v-8.54L22 3z',
		'Sliders': 'M4 21v-7 M4 10V3 M12 21v-9 M12 8V3 M20 21v-5 M20 12V3 M1 14h6 M9 8h6 M17 16h6',
		'Tool': 'M14.7 6.3a1 1 0 0 0 0 1.4l1.6 1.6a1 1 0 0 0 1.4 0l3.77-3.77a6 6 0 0 1-7.94 7.94l-6.91 6.91a2.12 2.12 0 0 1-3-3l6.91-6.91a6 6 0 0 1 7.94-7.94l-3.76 3.76z',
		'Wrench': 'M14.7 6.3a1 1 0 0 0 0 1.4l1.6 1.6a1 1 0 0 0 1.4 0l3.77-3.77a6 6 0 0 1-7.94 7.94l-6.91 6.91a2.12 2.12 0 0 1-3-3l6.91-6.91a6 6 0 0 1 7.94-7.94l-3.76 3.76z',
		'HelpCircle': 'M12 22c5.523 0 10-4.477 10-10S17.523 2 12 2 2 6.477 2 12s4.477 10 10 10z M9.09 9a3 3 0 0 1 5.83 1c0 2-3 3-3 3 M12 17h.01',
		'Info': 'M12 22c5.523 0 10-4.477 10-10S17.523 2 12 2 2 6.477 2 12s4.477 10 10 10z M12 16v-4 M12 8h.01',
		'AlertCircle': 'M12 22c5.523 0 10-4.477 10-10S17.523 2 12 2 2 6.477 2 12s4.477 10 10 10z M12 8v4 M12 16h.01',
		'AlertTriangle': 'M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z M12 9v4 M12 17h.01',
		'XCircle': 'M12 22c5.523 0 10-4.477 10-10S17.523 2 12 2 2 6.477 2 12s4.477 10 10 10z M15 9l-6 6 M9 9l6 6',
		'CheckCircle': 'M22 11.08V12a10 10 0 1 1-5.93-9.14 M22 4L12 14.01l-3-3',
		'Power': 'M18.36 6.64a9 9 0 1 1-12.73 0 M12 2v10',
		'Sun': 'M12 17a5 5 0 1 0 0-10 5 5 0 0 0 0 10z M12 1v2 M12 21v2 M4.22 4.22l1.42 1.42 M18.36 18.36l1.42 1.42 M1 12h2 M21 12h2 M4.22 19.78l1.42-1.42 M18.36 5.64l1.42-1.42',
		'Moon': 'M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z',
		'Coffee': 'M18 8h1a4 4 0 0 1 0 8h-1 M2 8h16v9a4 4 0 0 1-4 4H6a4 4 0 0 1-4-4V8z M6 1v3 M10 1v3 M14 1v3',
		'Smile': 'M12 22c5.523 0 10-4.477 10-10S17.523 2 12 2 2 6.477 2 12s4.477 10 10 10z M8 14s1.5 2 4 2 4-2 4-2 M9 9h.01 M15 9h.01',
		'Rocket': 'M4.5 16.5c-1.5 1.26-2 5-2 5s3.74-.5 5-2c.71-.84.7-2.13-.09-2.91a2.18 2.18 0 0 0-2.91-.09z M12 15l-3-3a22 22 0 0 1 2-3.95A12.88 12.88 0 0 1 22 2c0 2.72-.78 7.5-6 11a22.35 22.35 0 0 1-4 2z M9 12H4s.55-3.03 2-4c1.62-1.08 5 0 5 0 M12 15v5s3.03-.55 4-2c1.08-1.62 0-5 0-5',
		'Sparkles': 'M12 3l1.54 4.32L18 9l-4.46 1.68L12 15l-1.54-4.32L6 9l4.46-1.68L12 3z M5 19l.77 2.16L8 22l-2.23.84L5 25l-.77-2.16L2 22l2.23-.84L5 19z M19 17l.77 2.16L22 20l-2.23.84L19 23l-.77-2.16L16 20l2.23-.84L19 17z',
		'Bot': 'M12 8V4H8 M20 8v12a2 2 0 0 1-2 2H6a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h12a2 2 0 0 1 2 2z M14 8a2 2 0 0 0-4 0 M9 13h.01 M15 13h.01 M9 17h6',
		'Brain': 'M9.5 2A2.5 2.5 0 0 1 12 4.5v15a2.5 2.5 0 0 1-4.96.44 2.5 2.5 0 0 1-2.96-3.08 3 3 0 0 1-.34-5.58 2.5 2.5 0 0 1 1.32-4.24 2.5 2.5 0 0 1 1.98-3A2.5 2.5 0 0 1 9.5 2z M14.5 2A2.5 2.5 0 0 0 12 4.5v15a2.5 2.5 0 0 0 4.96.44 2.5 2.5 0 0 0 2.96-3.08 3 3 0 0 0 .34-5.58 2.5 2.5 0 0 0-1.32-4.24 2.5 2.5 0 0 0-1.98-3A2.5 2.5 0 0 0 14.5 2z',
	};

	// Get all icon names
	const allIconNames = Object.keys(iconLibrary).sort();

	// Filter icons based on search
	const filteredIcons = $derived(
		searchQuery
			? allIconNames.filter(name =>
				name.toLowerCase().includes(searchQuery.toLowerCase())
			)
			: allIconNames
	);

	// Handle icon selection
	function selectIcon(name: string) {
		selectedIcon = name;
		customSvg = undefined;
		activeTab = 'icons';
	}

	// Handle custom SVG upload
	function handleFileUpload(event: Event) {
		const input = event.target as HTMLInputElement;
		const file = input.files?.[0];
		if (!file) return;

		if (!file.type.includes('svg')) {
			alert('Please upload an SVG file');
			return;
		}

		const reader = new FileReader();
		reader.onload = (e) => {
			customSvg = e.target?.result as string;
			selectedIcon = undefined;
		};
		reader.readAsText(file);
	}

	// Handle paste SVG
	function handleSvgPaste(event: Event) {
		const textarea = event.target as HTMLTextAreaElement;
		const value = textarea.value.trim();
		if (value.startsWith('<svg') || value.startsWith('<?xml')) {
			customSvg = value;
			selectedIcon = undefined;
		}
	}

	// Apply selection
	function applySelection() {
		let config: CustomIconConfig | undefined;

		if (selectedIcon) {
			config = {
				type: 'lucide',
				lucideName: selectedIcon,
				foregroundColor,
				backgroundColor,
			};
		} else if (customSvg) {
			config = {
				type: 'custom',
				customSvg,
				foregroundColor,
				backgroundColor,
			};
		}

		onSelect?.(config);
	}

	// Reset to default
	function resetToDefault() {
		onSelect?.(undefined);
	}

	// Close picker
	function close() {
		onClose?.();
	}
</script>

<div class="icon-picker">
	<!-- Header -->
	<div class="picker-header">
		<h3>Choose Icon</h3>
		<button class="close-btn" onclick={close}>
			<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
				<path d="M18 6L6 18M6 6l12 12" />
			</svg>
		</button>
	</div>

	<!-- Tabs -->
	<div class="tabs">
		<button
			class="tab"
			class:active={activeTab === 'icons'}
			onclick={() => activeTab = 'icons'}
		>
			Icon Library ({allIconNames.length})
		</button>
		<button
			class="tab"
			class:active={activeTab === 'custom'}
			onclick={() => activeTab = 'custom'}
		>
			Custom SVG
		</button>
	</div>

	<!-- Content -->
	<div class="picker-content">
		{#if activeTab === 'icons'}
			<!-- Search -->
			<div class="search-container">
				<input
					type="text"
					placeholder="Search icons..."
					bind:value={searchQuery}
					class="search-input"
				/>
				<span class="icon-count">{filteredIcons.length} icons</span>
			</div>

			<!-- Icon Grid -->
			<div class="icon-grid">
				{#each filteredIcons as iconName (iconName)}
					<button
						class="icon-item"
						class:selected={selectedIcon === iconName}
						onclick={() => selectIcon(iconName)}
						title={iconName}
					>
						<svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
							<path d={iconLibrary[iconName]} />
						</svg>
					</button>
				{/each}
			</div>

			{#if selectedIcon}
				<div class="selected-info">
					Selected: <strong>{selectedIcon}</strong>
				</div>
			{/if}
		{:else}
			<!-- Custom SVG Upload -->
			<div class="custom-upload">
				<div class="upload-zone">
					<label class="upload-label">
						<input
							type="file"
							accept=".svg"
							onchange={handleFileUpload}
							class="file-input"
						/>
						<svg width="40" height="40" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
							<path d="M21 15v4a2 2 0 01-2 2H5a2 2 0 01-2-2v-4M17 8l-5-5-5 5M12 3v12" />
						</svg>
						<span>Click to upload SVG</span>
					</label>
				</div>

				<div class="divider">
					<span>or paste SVG code</span>
				</div>

				<textarea
					class="svg-textarea"
					placeholder="Paste SVG code here..."
					value={customSvg || ''}
					oninput={handleSvgPaste}
				></textarea>

				{#if customSvg}
					<div class="preview-custom">
						<span>Preview:</span>
						<div class="custom-preview-icon" style="background-color: {backgroundColor}">
							{@html customSvg}
						</div>
					</div>
				{/if}
			</div>
		{/if}
	</div>

	<!-- Color Pickers -->
	<div class="color-section">
		<div class="color-picker">
			<label>Icon Color</label>
			<div class="color-input-wrapper">
				<input
					type="color"
					bind:value={foregroundColor}
					class="color-input"
				/>
				<input
					type="text"
					bind:value={foregroundColor}
					class="color-text"
					placeholder="#333333"
				/>
			</div>
		</div>
		<div class="color-picker">
			<label>Background</label>
			<div class="color-input-wrapper">
				<input
					type="color"
					bind:value={backgroundColor}
					class="color-input"
				/>
				<input
					type="text"
					bind:value={backgroundColor}
					class="color-text"
					placeholder="#F5F5F5"
				/>
			</div>
		</div>
	</div>

	<!-- Preview -->
	{#if selectedIcon || customSvg}
		<div class="preview-section">
			<span>Preview:</span>
			<div class="preview-icon" style="background-color: {backgroundColor}">
				{#if selectedIcon && iconLibrary[selectedIcon]}
					<svg width="32" height="32" viewBox="0 0 24 24" fill="none" stroke={foregroundColor} stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
						<path d={iconLibrary[selectedIcon]} />
					</svg>
				{:else if customSvg}
					<div class="custom-svg-preview" style="color: {foregroundColor}">
						{@html customSvg}
					</div>
				{/if}
			</div>
		</div>
	{/if}

	<!-- Actions -->
	<div class="picker-actions">
		<button class="btn-secondary" onclick={resetToDefault}>
			Reset to Default
		</button>
		<button class="btn-secondary" onclick={close}>
			Cancel
		</button>
		<button
			class="btn-primary"
			onclick={applySelection}
			disabled={!selectedIcon && !customSvg}
		>
			Apply
		</button>
	</div>
</div>

<style>
	.icon-picker {
		background: white;
		border-radius: 12px;
		box-shadow: 0 20px 60px rgba(0, 0, 0, 0.3);
		width: 520px;
		max-height: 650px;
		display: flex;
		flex-direction: column;
		overflow: hidden;
		z-index: 1;
	}

	.picker-header {
		display: flex;
		justify-content: space-between;
		align-items: center;
		padding: 16px 20px;
		border-bottom: 1px solid #e5e7eb;
	}

	.picker-header h3 {
		margin: 0;
		font-size: 16px;
		font-weight: 600;
	}

	.close-btn {
		background: none;
		border: none;
		cursor: pointer;
		padding: 4px;
		color: #6b7280;
		border-radius: 4px;
	}

	.close-btn:hover {
		background: #f3f4f6;
		color: #111827;
	}

	.tabs {
		display: flex;
		border-bottom: 1px solid #e5e7eb;
	}

	.tab {
		flex: 1;
		padding: 12px;
		background: none;
		border: none;
		cursor: pointer;
		font-size: 14px;
		color: #6b7280;
		border-bottom: 2px solid transparent;
		transition: all 0.15s;
	}

	.tab:hover {
		color: #111827;
		background: #f9fafb;
	}

	.tab.active {
		color: #2563eb;
		border-bottom-color: #2563eb;
	}

	.picker-content {
		flex: 1;
		overflow-y: auto;
		padding: 16px;
	}

	.search-container {
		display: flex;
		align-items: center;
		gap: 12px;
		margin-bottom: 12px;
	}

	.search-input {
		flex: 1;
		padding: 8px 12px;
		border: 1px solid #d1d5db;
		border-radius: 8px;
		font-size: 14px;
	}

	.search-input:focus {
		outline: none;
		border-color: #2563eb;
		box-shadow: 0 0 0 3px rgba(37, 99, 235, 0.1);
	}

	.icon-count {
		font-size: 12px;
		color: #9ca3af;
		white-space: nowrap;
	}

	.icon-grid {
		display: grid;
		grid-template-columns: repeat(8, 1fr);
		gap: 4px;
		max-height: 280px;
		overflow-y: auto;
	}

	.icon-item {
		aspect-ratio: 1;
		display: flex;
		align-items: center;
		justify-content: center;
		background: none;
		border: 1px solid transparent;
		border-radius: 8px;
		cursor: pointer;
		color: #374151;
		transition: all 0.15s;
	}

	.icon-item:hover {
		background: #f3f4f6;
		border-color: #d1d5db;
	}

	.icon-item.selected {
		background: #dbeafe;
		border-color: #2563eb;
		color: #2563eb;
	}

	.selected-info {
		margin-top: 12px;
		padding: 8px 12px;
		background: #f3f4f6;
		border-radius: 6px;
		font-size: 13px;
		color: #4b5563;
	}

	.custom-upload {
		display: flex;
		flex-direction: column;
		gap: 16px;
	}

	.upload-zone {
		border: 2px dashed #d1d5db;
		border-radius: 12px;
		padding: 24px;
		text-align: center;
	}

	.upload-label {
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: 8px;
		cursor: pointer;
		color: #6b7280;
	}

	.upload-label:hover {
		color: #2563eb;
	}

	.file-input {
		display: none;
	}

	.divider {
		text-align: center;
		color: #9ca3af;
		font-size: 12px;
	}

	.svg-textarea {
		width: 100%;
		height: 100px;
		padding: 12px;
		border: 1px solid #d1d5db;
		border-radius: 8px;
		font-family: monospace;
		font-size: 12px;
		resize: vertical;
	}

	.svg-textarea:focus {
		outline: none;
		border-color: #2563eb;
	}

	.preview-custom {
		display: flex;
		align-items: center;
		gap: 12px;
		font-size: 13px;
		color: #6b7280;
	}

	.custom-preview-icon {
		width: 48px;
		height: 48px;
		border-radius: 8px;
		display: flex;
		align-items: center;
		justify-content: center;
	}

	.custom-preview-icon :global(svg) {
		width: 32px;
		height: 32px;
	}

	.color-section {
		display: flex;
		gap: 16px;
		padding: 16px;
		border-top: 1px solid #e5e7eb;
	}

	.color-picker {
		flex: 1;
	}

	.color-picker label {
		display: block;
		font-size: 12px;
		font-weight: 500;
		color: #6b7280;
		margin-bottom: 6px;
	}

	.color-input-wrapper {
		display: flex;
		gap: 8px;
		align-items: center;
	}

	.color-input {
		width: 36px;
		height: 36px;
		border: none;
		border-radius: 6px;
		cursor: pointer;
		padding: 0;
	}

	.color-text {
		flex: 1;
		padding: 8px;
		border: 1px solid #d1d5db;
		border-radius: 6px;
		font-size: 13px;
		font-family: monospace;
	}

	.preview-section {
		display: flex;
		align-items: center;
		gap: 12px;
		padding: 0 16px 16px;
		font-size: 13px;
		color: #6b7280;
	}

	.preview-icon {
		width: 56px;
		height: 56px;
		border-radius: 12px;
		display: flex;
		align-items: center;
		justify-content: center;
		box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
	}

	.custom-svg-preview :global(svg) {
		width: 32px;
		height: 32px;
	}

	.picker-actions {
		display: flex;
		gap: 8px;
		padding: 16px;
		border-top: 1px solid #e5e7eb;
		background: #f9fafb;
	}

	.btn-primary,
	.btn-secondary {
		padding: 10px 16px;
		border-radius: 8px;
		font-size: 14px;
		font-weight: 500;
		cursor: pointer;
		transition: all 0.15s;
	}

	.btn-primary {
		background: #2563eb;
		color: white;
		border: none;
		margin-left: auto;
	}

	.btn-primary:hover:not(:disabled) {
		background: #1d4ed8;
	}

	.btn-primary:disabled {
		opacity: 0.5;
		cursor: not-allowed;
	}

	.btn-secondary {
		background: white;
		border: 1px solid #d1d5db;
		color: #374151;
	}

	.btn-secondary:hover {
		background: #f3f4f6;
		border-color: #9ca3af;
	}
</style>
