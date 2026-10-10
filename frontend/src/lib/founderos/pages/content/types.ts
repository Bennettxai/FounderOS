/** Payloads of GET /api/founderos/pages/content and /pages/content/lead-magnets
 *  (desktop/backend-go/internal/founderos/api/page_content.go). */
import type { Meter, SeriesPoint, StatChip } from '$lib/founderos/kit/format';

export type Insight = { value: number; display?: string; headline: string; body: string; frac: number };

export type CrewMember = {
	id: string;
	name: string;
	role: string;
	status: string;
	tier: string;
	description: string;
	model: string;
	tools: string[];
	parentId: string | null;
	departmentId: string;
	/** false when the bridge runtime has no implementation for this agent yet */
	runnable: boolean;
};

export type LeadMagnetStatus = 'live' | 'draft' | 'paused' | 'archived';
export type LeadMagnet = {
	id: string;
	name: string;
	offer: string;
	url: string;
	status: LeadMagnetStatus;
	captures: 'email' | 'booking' | 'none';
	destination: string;
	source: string;
	launchedAt: string;
	notes: string;
	origin: 'seed' | 'os';
};

export type ZernioPost = { platform: string; caption: string; url: string; publishedAt: string | null; status: string };

export type ContentVolume = {
	headline: number;
	chips: StatChip[];
	caption: string;
	meters: Meter[];
	foot: string;
	series: SeriesPoint[];
	postsInWindow: number;
	activeDays: number;
	crewRuns: SeriesPoint[];
	runsInWindow: number;
	magnets: { total: number; live: number; draft: number; paused: number; archived: number };
	insight: Insight;
};

export type ContentPage = {
	today: string;
	windowDays: number;
	crew: CrewMember[];
	leadMagnets: LeadMagnet[];
	posts: ZernioPost[];
	postsError: string | null;
	postsKnown: boolean;
	/** null when Zernio did not answer: unknown, not zero */
	pipelineActiveDays: number | null;
	volume: ContentVolume;
};

export type LeadMagnetVolume = {
	headline: number;
	total: number;
	counts: Record<LeadMagnetStatus, number>;
	chips: StatChip[];
	caption: string;
	meters: Meter[];
	foot: string;
	series: SeriesPoint[];
	shippedInWindow: number;
	captures: SeriesPoint[];
	destinations: SeriesPoint[];
	insight: Insight;
};

export type LeadMagnetsPage = {
	today: string;
	weeks: number;
	filter: 'all' | LeadMagnetStatus;
	filters: Array<'all' | LeadMagnetStatus>;
	rows: LeadMagnet[];
	volume: LeadMagnetVolume;
};

export const LEAD_MAGNET_STATUSES: LeadMagnetStatus[] = ['live', 'draft', 'paused', 'archived'];

/** "zernio-mcp" → "Zernio Mcp" (ContentAgentCard prettyTool). */
export function prettyTool(slug: string): string {
	return slug
		.split(/[-_]/)
		.map((w) => w.charAt(0).toUpperCase() + w.slice(1))
		.join(' ');
}

/** "https://www.x.com/a" → "x.com" (LeadMagnets host). */
export function host(url: string): string {
	try {
		return new URL(url).host.replace(/^www\./, '');
	} catch {
		return url;
	}
}

/** "2026-08-12" → "Aug 12" in UTC; anything unparseable prints as-is. */
export function dateLabel(iso: string): string {
	const d = new Date(`${iso}T00:00:00Z`);
	return Number.isNaN(d.getTime()) ? iso : d.toLocaleDateString('en-US', { month: 'short', day: 'numeric', timeZone: 'UTC' });
}

/** Status pill colors mean status only (Brand Deals style). */
const STATUS_HUE: Record<string, string> = {
	live: 'var(--bn-ok)',
	published: 'var(--bn-ok)',
	draft: 'var(--bn-text-2)',
	paused: 'var(--bn-warn)',
	scheduled: 'var(--bn-warn)',
	failed: 'var(--bn-err)',
	archived: 'var(--bn-text-3)'
};
export function statusPillStyle(status: string): string {
	const hue = STATUS_HUE[status] ?? 'var(--bn-text-2)';
	return `background: color-mix(in oklab, ${hue} 16%, transparent); color: ${hue}`;
}
