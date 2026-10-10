// Theme Store - Manages light/dark mode
import { writable, get } from 'svelte/store';
import { browser } from '$app/environment';

export type Theme = 'light' | 'dark' | 'system';

interface ThemeState {
	theme: Theme;
	resolvedTheme: 'light' | 'dark'; // The actual applied theme (resolves 'system')
}

const defaultState: ThemeState = {
	theme: 'light',
	resolvedTheme: 'light'
};

function getSystemTheme(): 'light' | 'dark' {
	if (!browser) return 'light';
	return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
}

function applyTheme(resolvedTheme: 'light' | 'dark') {
	if (!browser) return;

	if (resolvedTheme === 'dark') {
		document.documentElement.classList.add('dark');
	} else {
		document.documentElement.classList.remove('dark');
	}

	// Update CSS variables for legacy support - Modern Apple-style dark mode
	document.documentElement.style.setProperty('--color-bg', resolvedTheme === 'dark' ? '#1c1c1e' : '#ffffff');
	document.documentElement.style.setProperty('--color-bg-secondary', resolvedTheme === 'dark' ? '#2c2c2e' : '#f9fafb');
	document.documentElement.style.setProperty('--color-bg-tertiary', resolvedTheme === 'dark' ? '#3a3a3c' : '#f3f4f6');
	document.documentElement.style.setProperty('--color-border', resolvedTheme === 'dark' ? 'rgba(255, 255, 255, 0.12)' : '#e5e7eb');
	document.documentElement.style.setProperty('--color-border-hover', resolvedTheme === 'dark' ? 'rgba(255, 255, 255, 0.2)' : '#d1d5db');
	document.documentElement.style.setProperty('--color-text', resolvedTheme === 'dark' ? '#f5f5f7' : '#111827');
	document.documentElement.style.setProperty('--color-text-secondary', resolvedTheme === 'dark' ? '#a1a1a6' : '#4b5563');
	document.documentElement.style.setProperty('--color-text-muted', resolvedTheme === 'dark' ? '#6e6e73' : '#9ca3af');
	document.documentElement.style.setProperty('--color-primary', resolvedTheme === 'dark' ? '#ffffff' : '#111827');
	document.documentElement.style.setProperty('--color-primary-hover', resolvedTheme === 'dark' ? '#e5e5ea' : '#374151');
}

function createThemeStore() {
	// Load from localStorage if available
	let initial: ThemeState = defaultState;

	if (browser) {
		const stored = localStorage.getItem('theme');
		if (stored === 'light' || stored === 'dark' || stored === 'system') {
			const resolvedTheme = stored === 'system' ? getSystemTheme() : stored;
			initial = { theme: stored, resolvedTheme };
		} else {
			// No stored preference - default to light and save it
			localStorage.setItem('theme', 'light');
		}

		// Apply theme immediately
		applyTheme(initial.resolvedTheme);
	}

	const { subscribe, set, update } = writable<ThemeState>(initial);

	// Listen for system theme changes
	if (browser) {
		const mediaQuery = window.matchMedia('(prefers-color-scheme: dark)');
		mediaQuery.addEventListener('change', (e) => {
			update(state => {
				if (state.theme === 'system') {
					const resolvedTheme = e.matches ? 'dark' : 'light';
					applyTheme(resolvedTheme);
					return { ...state, resolvedTheme };
				}
				return state;
			});
		});
	}

	return {
		subscribe,

		setTheme: (theme: Theme) => {
			const resolvedTheme = theme === 'system' ? getSystemTheme() : theme;

			if (browser) {
				localStorage.setItem('theme', theme);
				applyTheme(resolvedTheme);
			}

			set({ theme, resolvedTheme });
		},

		// Initialize from API settings (call this after loading user settings)
		initFromSettings: (theme: string) => {
			const validTheme: Theme = theme === 'dark' || theme === 'system' ? theme : 'light';
			const resolvedTheme = validTheme === 'system' ? getSystemTheme() : validTheme;

			if (browser) {
				localStorage.setItem('theme', validTheme);
				applyTheme(resolvedTheme);
			}

			set({ theme: validTheme, resolvedTheme });
		},

		// Get current resolved theme
		getResolvedTheme: (): 'light' | 'dark' => {
			return get({ subscribe }).resolvedTheme;
		},

		// Check if dark mode is active
		isDark: (): boolean => {
			return get({ subscribe }).resolvedTheme === 'dark';
		}
	};
}

export const themeStore = createThemeStore();
