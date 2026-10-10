/**
 * Brand marks for the Connections board (FounderOS v1 lib/brand-logos.tsx).
 * Sources, in prod's order:
 *   1. HANDMADE  full-colour marks simple-icons dropped (Slack)
 *   2. VECTOR    real vendor marks (brand-marks.ts)
 *   3. ICONS     simple-icons glyph + official hex (brand-marks.ts)
 *   4. RASTER    vendored PNGs under static/logos/integrations
 *   5. LETTERMARK an intentional tinted initial for brands with no logo
 * A slug that matches none of them is a typo: brandMarkKind says 'none'.
 */
import { ICONS, VECTOR } from './brand-marks';

export type BrandMarkKind = 'handmade' | 'vector' | 'icon' | 'raster' | 'lettermark' | 'none';

/** Slack's four-colour mark (122.8 viewBox), as prod hand-authors it. */
const SLACK_SVG =
	'<path d="M25.8 77.6c0 7.1-5.8 12.9-12.9 12.9S0 84.7 0 77.6s5.8-12.9 12.9-12.9h12.9v12.9zm6.5 0c0-7.1 5.8-12.9 12.9-12.9s12.9 5.8 12.9 12.9v32.3c0 7.1-5.8 12.9-12.9 12.9s-12.9-5.8-12.9-12.9V77.6z" fill="#E01E5A"/>' +
	'<path d="M45.2 25.8c-7.1 0-12.9-5.8-12.9-12.9S38.1 0 45.2 0s12.9 5.8 12.9 12.9v12.9H45.2zm0 6.5c7.1 0 12.9 5.8 12.9 12.9s-5.8 12.9-12.9 12.9H12.9C5.8 58.1 0 52.3 0 45.2s5.8-12.9 12.9-12.9h32.3z" fill="#36C5F0"/>' +
	'<path d="M97 45.2c0-7.1 5.8-12.9 12.9-12.9s12.9 5.8 12.9 12.9-5.8 12.9-12.9 12.9H97V45.2zm-6.5 0c0 7.1-5.8 12.9-12.9 12.9s-12.9-5.8-12.9-12.9V12.9C64.7 5.8 70.5 0 77.6 0s12.9 5.8 12.9 12.9v32.3z" fill="#2EB67D"/>' +
	'<path d="M77.6 97c7.1 0 12.9 5.8 12.9 12.9s-5.8 12.9-12.9 12.9-12.9-5.8-12.9-12.9V97h12.9zm0-6.5c-7.1 0-12.9-5.8-12.9-12.9s5.8-12.9 12.9-12.9h32.3c7.1 0 12.9 5.8 12.9 12.9s-5.8 12.9-12.9 12.9H77.6z" fill="#ECB22E"/>';

const HANDMADE: Record<string, { viewBox: string; svg: string }> = {
	slack: { viewBox: '0 0 122.8 122.8', svg: SLACK_SVG }
};

const RASTER: Record<string, string> = {
	beehiiv: '/logos/integrations/beehiiv.png',
	docusign: '/logos/integrations/docusign.png'
};

/** Brands rendered as a tinted initial on purpose (no logo available). */
const LETTERMARK: Record<string, string> = {
	salesforce: '#00A1E0',
	arcads: '#FF6A3D',
	paykit: '#22C55E',
	trakyo: '#EAB308',
	flexpay: '#A855F7',
	vidalytics: '#3B82F6',
	skool: '#E4573D',
	'proposal-gen': '#00764f',
	wispr: '#FFA946',
	phantom: '#AB9FF2',
	// G-Brain's violet, worn by the Optimal Engine that replaced it here
	'optimal-engine': '#A78BFA',
	paperclip: '#E8E8E8',
	plaud: '#E8E8E8'
};

const FOREGROUND = '#e6e7ea';

/** Perceived luminance of #rrggbb, 0..1: dark glyphs need a lightened fill. */
function luminance(hex: string): number {
	const h = hex.replace('#', '');
	const r = parseInt(h.slice(0, 2), 16) / 255;
	const g = parseInt(h.slice(2, 4), 16) / 255;
	const b = parseInt(h.slice(4, 6), 16) / 255;
	return 0.2126 * r + 0.7152 * g + 0.0722 * b;
}

export type BrandMark =
	| { kind: 'handmade'; viewBox: string; svg: string; tile: string }
	| { kind: 'vector'; viewBox: string; svg: string; mono: boolean; tile: string }
	| { kind: 'icon'; path: string; fill: string; tile: string }
	| { kind: 'raster'; src: string; tile: string }
	| { kind: 'lettermark'; initial: string; color: string; tile: string };

export function brandMarkKind(slug: string): BrandMarkKind {
	if (HANDMADE[slug]) return 'handmade';
	if (VECTOR[slug]) return 'vector';
	if (ICONS[slug]) return 'icon';
	if (slug in RASTER) return 'raster';
	if (slug in LETTERMARK) return 'lettermark';
	return 'none';
}

/** What BrandLogo draws for a slug, with prod's tile tint. */
export function brandMark(slug: string, name: string): BrandMark {
	const handmade = HANDMADE[slug];
	if (handmade) return { kind: 'handmade', ...handmade, tile: 'rgba(255,255,255,0.05)' };
	const vector = VECTOR[slug];
	if (vector) return { kind: 'vector', ...vector, tile: 'rgba(255,255,255,0.06)' };
	const icon = ICONS[slug];
	if (icon) {
		const brand = `#${icon.hex}`;
		const dark = luminance(brand) < 0.22;
		return {
			kind: 'icon',
			path: icon.path,
			fill: dark ? FOREGROUND : brand,
			tile: dark ? 'rgba(255,255,255,0.06)' : `color-mix(in srgb, ${brand} 16%, transparent)`
		};
	}
	const raster = RASTER[slug];
	if (raster) return { kind: 'raster', src: raster, tile: 'rgba(255,255,255,0.06)' };
	const color = LETTERMARK[slug] ?? '#8a8f98';
	return {
		kind: 'lettermark',
		initial: (name || slug).trim().charAt(0).toUpperCase() || '?',
		color,
		tile: `color-mix(in srgb, ${color} 20%, transparent)`
	};
}
