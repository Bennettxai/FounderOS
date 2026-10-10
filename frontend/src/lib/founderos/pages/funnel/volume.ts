/**
 * The channel-funnels and VSL cards' view-models in the Deal Volume shape
 * (FounderOS v1 lib/channel-volume.ts + lib/vsl-view.ts). Every meter is a real
 * fraction of its own whole. An empty Trakyo hand-off is an empty meter that
 * reads "none yet" (frac 0, as prod); an unexported VSL rate is unknown (null).
 */
import type { Meter, StatChip } from '$lib/founderos/kit';
import type { TrakyoFunnel, VslSnapshot } from './types';

const n = (v: number) => v.toLocaleString('en-US');

/** Trakyo counts events, not people: a ratio can pass 100%, the bar stops full. */
export function trakyoVolume(funnel: TrakyoFunnel) {
	const c = funnel.counts;
	const revenue = Number(funnel.revenue);
	const handoff = (label: string, num: number, den: number, hue: string): Meter => ({
		label: `${label} (${num}/${den})`,
		frac: den > 0 ? Math.min(1, num / den) : 0,
		display: den > 0 ? `${Math.round((num / den) * 100)}%` : 'none yet',
		hue
	});
	const chips: StatChip[] = [{ tone: 'ok', text: `${n(c.closes)} closes` }, { text: `${n(c.bookings)} bookings` }];
	return {
		revenue,
		headline: revenue.toLocaleString('en-US', { style: 'currency', currency: 'USD', minimumFractionDigits: 2, maximumFractionDigits: 2 }),
		chips,
		caption: `first-touch revenue · ${n(c.clicks)} clicks · ${n(c.visits)} visits`,
		meters: [
			handoff('Visits → forms', c.form_submissions, c.visits, 'var(--ramp-1)'),
			handoff('Forms → bookings', c.bookings, c.form_submissions, 'var(--ramp-2)'),
			handoff('Bookings → closes', c.closes, c.bookings, 'var(--bn-accent)')
		],
		foot: `${funnel.range.start.slice(0, 10)} to ${funnel.range.end.slice(0, 10)} · ${funnel.range.timezone} · events, not unique people`
	};
}

export function vslPeriods(rows: VslSnapshot[]): string[] {
	return [...new Set(rows.map((row) => `${row.dateFrom}/${row.dateTo}`))].sort().reverse();
}

export function vslForPeriod(rows: VslSnapshot[], period: string): VslSnapshot[] {
	return rows.filter((row) => `${row.dateFrom}/${row.dateTo}` === period);
}

const HUES = ['var(--ramp-1)', 'var(--ramp-2)', 'var(--ramp-3)', 'var(--ramp-4)'];

/** Plays in the saved period; one meter per video, the tail past four folded into Other. */
export function vslVolume(rows: VslSnapshot[]) {
	const plays = rows.reduce((s, r) => s + r.plays, 0);
	const sorted = [...rows].sort((a, b) => b.plays - a.plays).map((r) => ({ label: r.title, value: r.plays }));
	const shown =
		sorted.length <= 4
			? sorted
			: [...sorted.slice(0, 3), { label: `Other videos (${sorted.length - 3})`, value: sorted.slice(3).reduce((s, r) => s + r.value, 0) }];
	const meters: Meter[] = shown.map((r, i) => ({
		label: r.label,
		frac: plays > 0 ? r.value / plays : null,
		display: `${n(r.value)} · ${plays > 0 ? Math.round((r.value / plays) * 100) : 0}%`,
		hue: HUES[i % HUES.length]
	}));
	const captured = rows.map((r) => r.capturedAt.slice(0, 10)).sort().at(-1);
	const chips: StatChip[] = rows.length
		? [{ text: `${rows.length} video${rows.length === 1 ? '' : 's'}` }, { text: `${n(rows.reduce((s, r) => s + r.uniqueViewers, 0))} unique viewers` }]
		: [];
	return {
		plays,
		chips,
		caption: captured ? `plays in this saved period · captured ${captured}` : 'no video snapshots imported yet',
		meters
	};
}

/** The selected video's watch behavior; a rate Vidalytics did not export reads N/A. */
export function vslRateMeters(video: VslSnapshot): Meter[] {
	const rate = (label: string, v: number | null, hue: string): Meter => ({
		label,
		frac: v,
		display: v === null ? 'N/A' : `${(v * 100).toFixed(1)}%`,
		hue
	});
	return [
		rate('Average watched', video.averageWatched, 'var(--bn-accent)'),
		rate('Play rate', video.playRate, 'var(--ramp-1)'),
		rate('Unmute rate', video.unmuteRate, 'var(--ramp-2)'),
		rate('Bounce rate', video.bounceRate, 'var(--bn-warn)')
	];
}
