/**
 * Wire types for /api/founderos/pages/funnel, /pages/funnel/analytics and
 * /pages/funnel/lead-message (desktop/backend-go/internal/founderos/api/page_funnel.go).
 * Nullable fields are unknowns: they read as unknown, never as zero.
 */
import type { Meter, SeriesPoint, StatChip } from '$lib/founderos/kit';

export type FunnelStage = 'first_touch' | 'engaged' | 'nurtured' | 'opted_in' | 'converted';
export type FunnelVenture = 'vantage' | 'launchpad-cohort';

export type Touch = {
	id: string;
	contactId: string;
	seq: number;
	stage: FunnelStage;
	channel: string;
	label: string;
	source: string;
	at: string;
	acquisition?: string;
};

export type Journey = {
	id: string;
	name: string;
	venture: FunnelVenture;
	status: FunnelStage;
	product: string | null;
	amountUsd: number | null;
	relationship: 'cold' | 'warm' | 'hot';
	likelihood: number;
	url: string | null;
	email: string | null;
	phone: string | null;
	person: string | null;
	company: string | null;
	role: string | null;
	linkedin: string | null;
	createdAt: string;
	touches: Touch[];
};

export type Origin = { segment: string; entry: string | null; channel: string | null; source: string | null; at: string | null };

export type FunnelNode = {
	id: string;
	name: string;
	venture: FunnelVenture;
	status: FunnelStage;
	relationship: Journey['relationship'];
	likelihood: number;
	state: 'converted' | 'stalled' | 'active' | 'decayed';
	daysSinceLastTouch: number;
	product: string | null;
	amountUsd: number | null;
	hubs: number[];
	currentHub: number;
	radius: number;
	decay: number;
	url: string | null;
	email: string | null;
	phone: string | null;
	person: string | null;
	company: string | null;
	role: string | null;
	linkedin: string | null;
	touches: Touch[];
	segment: number;
	rings: number[];
	currentRing: number;
	origin: Origin;
};

export type Segment = { id: string; label: string; count: number; converted: number };
export type StageRow = { stage: FunnelStage; total: number; organic: number; ads: number; conversionFromPrev: number | null };
export type Summary = { clients: number; converted: number; revenueUsd: number; stages: StageRow[] };

export type ConnectorStatus = { id: string; name: string; kind?: string; state: string; detail: string };
export type FunnelSource = ConnectorStatus & { live: boolean; count: number | null };

export type CommsItem = { source: string; title: string; sender?: string; replyTo?: string; preview: string; ts: string };

export type FunnelVolume = {
	revenueUsd: number;
	chips: StatChip[];
	caption: string;
	meters: Meter[];
	foot: string;
	series: SeriesPoint[];
	touchesInWindow: number;
	insight: { value: number; headline: string; body: string; frac: number };
	/** The entry window this volume covers (days); null on /funnel, which covers every active lead (v1). */
	window: number | null;
	/** Leads the headline, chips and meters are computed over. */
	leads: number;
};

export type VslSnapshot = {
	videoId: string;
	title: string;
	duration: string;
	dateFrom: string;
	dateTo: string;
	capturedAt: string;
	plays: number;
	uniqueViewers: number;
	impressions: number;
	playRate: number | null;
	averageWatched: number | null;
	unmuteRate: number | null;
	bounceRate: number | null;
	recordedConversions: number;
	recordedRevenue: number;
	audience: { second: number; viewers: number }[];
	trend: { date: string; plays: number }[];
	trendInterval: 'day' | 'month';
	source: string;
};

export type VslBody = { snapshots: VslSnapshot[]; igVideo: string | null; error: string | null };

export type FunnelBody = {
	now: string;
	venture: FunnelVenture | null;
	stage: FunnelStage | null;
	stages: { id: FunnelStage; label: string }[];
	stageLabels: Record<FunnelStage, string>;
	acquisitions: { id: string; label: string }[];
	summary: Summary;
	journeys: Journey[];
	archived: Journey[];
	table: Journey[];
	stageCounts: Partial<Record<FunnelStage, number>>;
	lastMessages: Record<string, { message: CommsItem | null }> | null;
	commsUnavailable: boolean;
	source: string;
	isLive: boolean;
	liveLabel: string;
	laneErrors: Record<string, string>;
	seedError: string | null;
	radial: { nodes: FunnelNode[]; segments: Segment[] };
	attention: { pushNow: Journey[]; saveNow: Journey[] };
	/** v1 funnelVolume: every active lead, touches over the last 30 days. */
	volume: FunnelVolume;
	sources: FunnelSource[];
	constants: { stallDays: number; decayDays: number; decayFadeStart: number };
	leads?: number;
	calls?: number;
	paykit?: number;
	total?: number;
};

export type TrakyoContent = {
	id: string | null;
	type: string;
	name: string;
	source: string;
	views: number | null;
	published_at: string | null;
	clicks: number;
	visits: number;
	form_submissions: number;
	bookings: number;
	revenue: { first_touch: string; last_touch: string };
};

export type TrakyoFunnel = {
	currency: string;
	attribution: string;
	range: { start: string; end: string; timezone: string };
	counts: { clicks: number; visits: number; form_submissions: number; bookings: number; closes: number };
	revenue: string;
};

export type ChannelCard = {
	id: string;
	label: string;
	items: TrakyoContent[];
	clicks: number;
	visits: number;
	forms: number;
	bookings: number;
	views: number | null;
	videosWithViews: number;
	revenue: { first_touch: number; last_touch: number };
};

type SectionState = 'ready' | 'partial' | 'error' | 'not_configured';
export type ChannelAnalyticsBody = {
	period: '7d' | '30d' | '90d';
	fetchedAt: string;
	content: { state: SectionState; rows: TrakyoContent[]; message: string | null };
	funnel: { state: SectionState; data: TrakyoFunnel | null; message: string | null };
	channels: ChannelCard[];
};

export type LeadMessageBody = { message: CommsItem | null; unavailable: boolean };
