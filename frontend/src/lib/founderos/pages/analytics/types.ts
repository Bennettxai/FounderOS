/** Wire types for GET /api/founderos/pages/analytics (internal/founderos/api/page_analytics.go). */
import type { Meter, SeriesPoint, StatChip } from '$lib/founderos/kit';

export type MetricTile = {
	id: string;
	label: string;
	unit: string;
	source: string;
	value: number;
	delta: number;
	deltaPct: boolean;
	live: boolean;
};

export type LiveTile = MetricTile & { spark: number[]; sparkReal: boolean };

export type AnalyticsPlatform = {
	platform: string;
	label: string;
	handle: string;
	url: string | null;
	followers: number | null;
	share: number | null;
	d7: number | null;
	bars: number[] | null;
};

export type AnalyticsBody = {
	today: string;
	runVolume: { date: string; count: number }[];
	runs30d: number;
	runsKnown: boolean;
	volume: {
		reach: number;
		headline: string;
		chips: StatChip[];
		caption: string;
		meters: Meter[];
		foot: string;
		runs: { total: number; ok: number; failed: number; okPct: number | null; agents: number; meters: Meter[] };
		rhythm: SeriesPoint[];
		rhythmTotal: number;
		insight: { value: number; headline: string; body: string; frac: number };
	};
	live: LiveTile[];
	pending: MetricTile[];
	audience: { known: boolean; totalFollowers: number; asOf: string | null; growth7d: number | null; platforms: AnalyticsPlatform[] };
	errors: Record<string, string>;
};
