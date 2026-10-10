import { writable, get } from 'svelte/store';
import { browser } from '$app/environment';

export type KBSection = 'home' | 'recent' | 'favorites' | 'all' | 'memories';
export type KBViewMode = 'list' | 'graph';

interface KBPreferences {
	// Navigation
	activeSection: KBSection;
	viewMode: KBViewMode;

	// Sidebar state
	expandedSections: string[];
	expandedPages: string[];
	sidebarWidth: number;
	sidebarCollapsed: boolean;

	// Quick access
	favorites: string[];
	recentPages: string[];
}

const DEFAULT_PREFERENCES: KBPreferences = {
	activeSection: 'home',
	viewMode: 'list',
	expandedSections: ['favorites', 'recent', 'projects', 'people', 'business', 'documents', 'memories'],
	expandedPages: [],
	sidebarWidth: 280,
	sidebarCollapsed: false,
	favorites: [],
	recentPages: []
};

const STORAGE_KEY = 'kb-preferences';
const MAX_RECENT = 10;

function loadFromStorage(): KBPreferences {
	if (!browser) return DEFAULT_PREFERENCES;

	try {
		const stored = localStorage.getItem(STORAGE_KEY);
		if (stored) {
			const parsed = JSON.parse(stored);
			return { ...DEFAULT_PREFERENCES, ...parsed };
		}
	} catch (e) {
		console.error('Failed to load KB preferences:', e);
	}

	return DEFAULT_PREFERENCES;
}

function saveToStorage(prefs: KBPreferences) {
	if (!browser) return;

	try {
		localStorage.setItem(STORAGE_KEY, JSON.stringify(prefs));
	} catch (e) {
		console.error('Failed to save KB preferences:', e);
	}
}

function createKBPreferencesStore() {
	const { subscribe, update, set } = writable<KBPreferences>(loadFromStorage());

	// Auto-save on changes
	subscribe((prefs) => {
		saveToStorage(prefs);
	});

	return {
		subscribe,

		// Section navigation
		setActiveSection(section: KBSection) {
			update((p) => ({ ...p, activeSection: section }));
		},

		// View mode
		setViewMode(mode: KBViewMode) {
			update((p) => ({ ...p, viewMode: mode }));
		},

		// Sidebar
		setSidebarWidth(width: number) {
			update((p) => ({ ...p, sidebarWidth: width }));
		},

		toggleSidebarCollapsed() {
			update((p) => ({ ...p, sidebarCollapsed: !p.sidebarCollapsed }));
		},

		// Section expansion
		toggleSection(sectionId: string) {
			update((p) => {
				const sections = new Set(p.expandedSections);
				if (sections.has(sectionId)) {
					sections.delete(sectionId);
				} else {
					sections.add(sectionId);
				}
				return { ...p, expandedSections: Array.from(sections) };
			});
		},

		// Page expansion
		togglePageExpanded(pageId: string) {
			update((p) => {
				const pages = new Set(p.expandedPages);
				if (pages.has(pageId)) {
					pages.delete(pageId);
				} else {
					pages.add(pageId);
				}
				return { ...p, expandedPages: Array.from(pages) };
			});
		},

		// Expand a page (without toggling - only adds)
		expandPage(pageId: string) {
			update((p) => {
				const pages = new Set(p.expandedPages);
				if (!pages.has(pageId)) {
					pages.add(pageId);
				}
				return { ...p, expandedPages: Array.from(pages) };
			});
		},

		// Favorites
		addToFavorites(pageId: string) {
			update((p) => {
				if (p.favorites.includes(pageId)) return p;
				return { ...p, favorites: [...p.favorites, pageId] };
			});
		},

		removeFromFavorites(pageId: string) {
			update((p) => ({
				...p,
				favorites: p.favorites.filter((id) => id !== pageId)
			}));
		},

		toggleFavorite(pageId: string) {
			const prefs = get({ subscribe });
			if (prefs.favorites.includes(pageId)) {
				this.removeFromFavorites(pageId);
			} else {
				this.addToFavorites(pageId);
			}
		},

		isFavorite(pageId: string): boolean {
			const prefs = get({ subscribe });
			return prefs.favorites.includes(pageId);
		},

		// Recent pages
		addToRecent(pageId: string) {
			update((p) => {
				// Remove if already exists, then add to front
				const filtered = p.recentPages.filter((id) => id !== pageId);
				const recent = [pageId, ...filtered].slice(0, MAX_RECENT);
				return { ...p, recentPages: recent };
			});
		},

		clearRecent() {
			update((p) => ({ ...p, recentPages: [] }));
		},

		// Reset
		reset() {
			set(DEFAULT_PREFERENCES);
		}
	};
}

export const kbPreferences = createKBPreferencesStore();
