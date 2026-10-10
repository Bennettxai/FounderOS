// The operator's skins on the bridge, ported from FounderOS v1 lib/theme.ts: Monolith
// Signal is the default identity and the other five are full re-skins picked
// from the Topbar's theme menu. Production calls Terminal 'dark'; the bridge
// keeps its own id 'terminal' (pages key colours off it) and reads a stored
// 'dark' as Terminal. The token blocks live in monolith.css.
export const FOUNDEROS_THEMES = ['mono', 'mono-light', 'terminal', 'light', 'midnight', 'ember'] as const;
export type FounderosTheme = (typeof FOUNDEROS_THEMES)[number];
export const DEFAULT_FOUNDEROS_THEME: FounderosTheme = 'mono';
export const STORAGE_KEY = 'founderos-theme';

/** The skins painted on a light canvas. BusinessOS's own `.dark` styles must
 *  be off under them, or its components paint dark panels on a white page. */
const LIGHT_THEMES: readonly FounderosTheme[] = ['mono-light', 'light'];

/** Picker metadata (FounderOS v1 THEME_META): name, one-line feel, [bg, accent, text] swatch. */
export const THEME_META: Record<FounderosTheme, { name: string; blurb: string; swatch: [string, string, string] }> = {
	terminal: { name: 'Terminal', blurb: 'phosphor green on near-black', swatch: ['#050807', '#3df08c', '#e4efe6'] },
	light: { name: 'Clay', blurb: 'warm paper with clay orange', swatch: ['#ece3d2', '#c96442', '#2b2722'] },
	midnight: { name: 'Midnight', blurb: 'deep navy, signal blue', swatch: ['#070d1f', '#5ec9f8', '#e8ecf9'] },
	ember: { name: 'Ember', blurb: 'coal dark, vault orange', swatch: ['#0c0806', '#e35c35', '#f2e9e2'] },
	mono: { name: 'Monolith', blurb: 'white on black, color = status only', swatch: ['#0a0a0a', '#f2f2f2', '#2fd36f'] },
	'mono-light': { name: 'Daylight', blurb: 'brain blue on cool white', swatch: ['#f2f6f9', '#4db3de', '#16222c'] }
};

/** Production's id for a skin the bridge names differently. */
const ALIASES: Record<string, FounderosTheme> = { dark: 'terminal' };

export function isFounderosTheme(v: unknown): v is FounderosTheme {
	return typeof v === 'string' && (FOUNDEROS_THEMES as readonly string[]).includes(v);
}

export function isDarkTheme(theme: FounderosTheme): boolean {
	return !LIGHT_THEMES.includes(theme);
}

/** A stored or on-page value as a skin, or the default. */
export function resolveTheme(v: string | null | undefined): FounderosTheme {
	const id = v != null && v in ALIASES ? ALIASES[v] : v;
	return isFounderosTheme(id) ? id : DEFAULT_FOUNDEROS_THEME;
}

export function readStoredTheme(): FounderosTheme {
	try {
		return resolveTheme(localStorage.getItem(STORAGE_KEY));
	} catch {
		return DEFAULT_FOUNDEROS_THEME;
	}
}

export function applyFounderosTheme(theme: FounderosTheme): void {
	const root = document.documentElement;
	root.setAttribute('data-founderos-theme', theme);
	root.classList.toggle('dark', isDarkTheme(theme));
	try {
		localStorage.setItem(STORAGE_KEY, theme);
	} catch {
		// Private mode: the theme still applies for this session.
	}
}

/** Leaving /os for a BusinessOS page: drop the operator's skin and hand the root
 *  back to BusinessOS's own light/dark choice. */
export function leaveFounderosTheme(businessOsDark: boolean): void {
	const root = document.documentElement;
	root.removeAttribute('data-founderos-theme');
	root.classList.toggle('dark', businessOsDark);
}

// Inline in app.html so the first paint of an /os page is already skinned (no
// flash). Only /os is skinned, and BusinessOS's own 'theme' is never written:
// pinning it to dark here once flipped the whole desktop dark.
export const FOUNDEROS_THEME_INIT_SCRIPT = `(function(){var p=location.pathname;if(p!=='/os'&&p.indexOf('/os/')!==0)return;var k=${JSON.stringify([...FOUNDEROS_THEMES])},l=${JSON.stringify([...LIGHT_THEMES])},t;try{t=localStorage.getItem('${STORAGE_KEY}')}catch(e){}if(t==='dark')t='terminal';if(k.indexOf(t)<0)t='${DEFAULT_FOUNDEROS_THEME}';var r=document.documentElement;r.setAttribute('data-founderos-theme',t);if(l.indexOf(t)<0)r.classList.add('dark');else r.classList.remove('dark');})();`;
