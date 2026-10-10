import { writable, derived, get } from 'svelte/store';
import { browser } from '$app/environment';

export type ThemeMode = 'light' | 'dark' | 'system';
export type ResolvedTheme = 'light' | 'dark';

const STORAGE_KEY = 'businessos-theme';

function getInitialTheme(): ThemeMode {
  if (!browser) return 'system';
  const stored = localStorage.getItem(STORAGE_KEY);
  if (stored === 'light' || stored === 'dark' || stored === 'system') {
    return stored;
  }
  return 'system';
}

function getSystemTheme(): ResolvedTheme {
  if (!browser) return 'light';
  return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
}

function createThemeStore() {
  const mode = writable<ThemeMode>(getInitialTheme());

  const resolved = derived(mode, ($mode): ResolvedTheme => {
    if ($mode === 'system') {
      return getSystemTheme();
    }
    return $mode;
  });

  // Apply theme to document
  if (browser) {
    resolved.subscribe((theme) => {
      document.documentElement.setAttribute('data-theme', theme);
      document.documentElement.classList.remove('light', 'dark');
      document.documentElement.classList.add(theme);
    });

    // Listen for system theme changes
    window.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', () => {
      if (get(mode) === 'system') {
        mode.set('system'); // Trigger re-evaluation
      }
    });
  }

  return {
    mode,
    resolved,

    setMode(newMode: ThemeMode) {
      mode.set(newMode);
      if (browser) {
        localStorage.setItem(STORAGE_KEY, newMode);
      }
    },

    toggle() {
      mode.update((current) => {
        const resolvedCurrent = current === 'system' ? getSystemTheme() : current;
        const next = resolvedCurrent === 'dark' ? 'light' : 'dark';
        if (browser) {
          localStorage.setItem(STORAGE_KEY, next);
        }
        return next;
      });
    },

    // For components that need the current value without subscription
    getResolved(): ResolvedTheme {
      const currentMode = get(mode);
      return currentMode === 'system' ? getSystemTheme() : currentMode;
    }
  };
}

export const theme = createThemeStore();
