/** Formatting for /os/analytics (FounderOS v1 components/SocialStats.tsx + the page's tileValue). */

export function formatFollowers(n: number | null): string {
	return n === null ? '—' : n.toLocaleString('en-US');
}

/** Growth percent; an em dash when history is too short, never a fake 0. */
export function formatPct(n: number | null): string {
	if (n === null) return '—';
	const rounded = Math.abs(n) < 10 ? n.toFixed(2) : n.toFixed(1);
	return `${n >= 0 ? '+' : ''}${rounded}%`;
}

/** A tile's value + small unit (compact for audience, $ for money). */
export function tileValue(value: number, unit: string): { main: string; small: string } {
	if (unit === 'usd') return { main: `$${value.toLocaleString('en-US')}`, small: '' };
	if (unit === 'followers') return { main: formatFollowers(value), small: '' };
	return { main: value.toLocaleString('en-US'), small: unit };
}

/** RunVolumeCard's axis label: "sep 24". */
export function fmtShort(iso: string): string {
	return new Date(`${iso}T00:00:00Z`).toLocaleDateString('en-US', { month: 'short', day: 'numeric', timeZone: 'UTC' }).toLowerCase();
}
