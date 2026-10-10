import { FounderosApiError } from '$lib/founderos/api';
/** Shared bits of the /os chrome (Sidebar, Topbar, CommandPalette). */

/** Window event the Topbar fires to open the ⌘K palette (FounderOS v1 'founderos:palette'). */
export const PALETTE_EVENT = 'founderos:palette';

/** Sidebar widths (FounderOS v1 lib/sidebar-layout DEFAULT_W / RAIL_W). The
 *  Sidebar publishes the current one as --bn-sidebar-w for the shell margin. */
export const EXPANDED_W = 232;
export const RAIL_W = 56;

/** localStorage key for the sidebar's icon-rail state. */
export const RAIL_KEY = 'founderos-sidebar-rail';

export function openPalette(): void {
	window.dispatchEvent(new CustomEvent(PALETTE_EVENT));
}

// FounderOS v1 Topbar SEGMENT_LABELS, with the knowledge view renamed Brain.
const SEGMENT_LABELS: Record<string, string> = {
	'': 'home',
	org: 'org-chart',
	brain: 'brain',
	integrations: 'connections'
};

/** The breadcrumb's last part for a /os path: the first segment under /os. */
export function breadcrumbFor(pathname: string): string {
	const segment = pathname.replace(/^\/os\/?/, '').split('/')[0] ?? '';
	return SEGMENT_LABELS[segment] ?? segment;
}

/** Storage can be missing or throw (private window); the chrome renders the same either way. */
export function loadKey(key: string): string | null {
	try {
		return localStorage.getItem(key);
	} catch {
		return null;
	}
}
export function saveKey(key: string, value: string): void {
	try {
		localStorage.setItem(key, value);
	} catch {
		// a per-viewer convenience; losing it only costs the remembered shape
	}
}

/** True while focus is in a text field, so bare digit keys stay typing. */
export function isTyping(): boolean {
	const el = document.activeElement as HTMLElement | null;
	if (!el) return false;
	const tag = el.tagName;
	return tag === 'INPUT' || tag === 'TEXTAREA' || tag === 'SELECT' || el.isContentEditable;
}

/** The FounderOS operator views (/os and below), where BusinessOS's site chrome
 *  (the cookie notice) stays out of the way. */
export function isOperatorView(pathname: string): boolean {
	return pathname === '/os' || pathname.startsWith('/os/');
}

/** The owner gate's answer for a signed-in account that has no Founder OS yet
 *  (HTTP 403 with setup: true): the /os layout shows SetupNotice instead of a
 *  page full of failed cards. */
export function needsSetup(err: unknown): boolean {
	return err instanceof FounderosApiError && err.status === 403 && (err.body as { setup?: unknown } | undefined)?.setup === true;
}
