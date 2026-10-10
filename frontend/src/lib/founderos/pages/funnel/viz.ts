/**
 * Shared math for the two funnel canvases (FounderOS v1 lib/funnel-viz.ts).
 * Pure and deterministic: no Math.random, no Date.
 */

/** Deterministic per-node noise, quantized to 4 decimals. */
export const rnd = (i: number, salt: number): number => {
	const x = Math.sin(i * 127.1 + salt * 311.7) * 43758.5453;
	return Math.round((x - Math.floor(x)) * 10000) / 10000;
};

export const easeInOut = (u: number): number => (u < 0.5 ? 2 * u * u : 1 - (-2 * u + 2) ** 2 / 2);

export const usd = (n: number): string => `$${Math.round(n).toLocaleString('en-US')}`;

/** Frame-rate independent smoothing (1 − e^(−dt/τ)), dt clamped to 64ms. */
export const smoothK = (dtMs: number, tauMs = 110): number => 1 - Math.exp(-Math.min(Math.max(dtMs, 0), 64) / tauMs);

/** A node wears its cluster hue and fades toward red as contact ages; converted is green. */
export const decayedColor = (baseVar: string, decay: number, converted: boolean): string =>
	converted
		? 'var(--bn-ok)'
		: decay > 0
			? `color-mix(in oklab, var(--bn-err) ${Math.round(Math.sqrt(decay) * 85)}%, ${baseVar})`
			: baseVar;

export const decayedOpacity = (decay: number): number => 0.95 - decay * 0.45;

/** Crowded hubs breathe wider: orbit band scales with the cluster's population. */
export const orbitSpread = (clusterCount: number): number => Math.min(2.4, Math.max(1, Math.sqrt(clusterCount / 12)));

/** Channel glyphs for journey chips (lib/funnel.ts CHANNEL_GLYPHS). */
export const CHANNEL_GLYPHS: Record<string, string> = {
	organic: '◉',
	ads: '▣',
	dm: '✉',
	email: '@',
	webinar: '▶',
	call: '☎',
	checkout: '$',
	crm: '◈'
};
