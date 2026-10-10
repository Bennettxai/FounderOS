/** JSON shapes of /api/founderos/pages/social* (internal/founderos/api/page_social.go). */
import type { Meter, SeriesPoint, StatChip } from '$lib/founderos/kit';

export type Growth = { d7: number | null; d30: number | null; d60: number | null; allTime: number | null };

export type PlatformStats = {
	platform: string;
	handle: string;
	url: string | null;
	followers: number | null;
	growth: Growth;
	series: { date: string; followers: number }[];
};

export type EmailList = {
	subscribers: number | null;
	asOf: string | null;
	growth: Growth;
	series: { date: string; subscribers: number }[];
};

export type DmMessage = {
	id: string;
	platform: string;
	subscriberId: string;
	name: string;
	handle: string | null;
	text: string;
	direction: 'in' | 'out';
	tag: string | null;
	ts: string;
	source: string;
};

export type DmThread = {
	subscriberId: string;
	name: string;
	handle: string | null;
	messages: DmMessage[];
	last: DmMessage;
	unreplied: boolean;
	/** Every message is seeded (seed-dummy), not a real DM. */
	demo?: boolean;
};

export type SocialPost = {
	id: string;
	caption: string;
	mediaUrl: string | null;
	platforms: string[];
	status: 'queued' | 'published' | 'failed';
	scheduledFor: string | null;
	createdAt: string;
};

export type ZernioPost = { platform: string; caption: string; url: string; publishedAt: string | null; status: string };
export type PostDay = { date: string; platforms: string[] };
export type SeriesValue = { date: string; value: number };
export type LabelledSeries = { key: string; label: string; color: string; points: SeriesValue[] };

export type Insight = { value: number; headline: string; body: string; frac: number };

export type SocialVolume = {
	headline: number;
	chips: StatChip[];
	caption: string;
	meters: Meter[];
	foot: string;
	series: SeriesPoint[];
	postsInWindow: number;
	mix: SeriesPoint[];
	insight: Insight;
};

export type SocialPage = {
	totalFollowers: number;
	asOf: string | null;
	platforms: PlatformStats[];
	emailList: EmailList;
	totalDms: number;
	audienceTotal: number;
	audienceGrowth: Growth;
	dmGrowth: Growth;
	today: string;
	growthLeader: string | null;
	dmThreads: DmThread[];
	audiencePoints: SeriesValue[];
	postDays: PostDay[] | null;
	postsKnown: boolean;
	postsError?: string;
	recentPosts: ZernioPost[] | null;
	recentError?: string;
	posts: SocialPost[];
	queued: number;
	sync: { source: string; recorded: number; error?: string };
	volume: SocialVolume;
};

export type Snapshot = { platform: string; capturedAt: string; followers: number; source: string };

export type PlatformPage = {
	account: { platform: string; handle: string; url: string | null; order: number };
	followers: number | null;
	growth: Growth;
	snapshots: Snapshot[];
	label: string;
	today: string;
	volume: {
		headline: number | null;
		chips: StatChip[];
		caption: string;
		meters: Meter[];
		foot: string;
		series: SeriesPoint[];
		intervals: number;
		gainedDays: number;
		dippedDays: number;
		trackedDays: number;
		net: number | null;
		insight: { value: number; display: string; headline: string; body: string; frac: number };
	};
};

export type Newsletter = {
	id: string;
	title: string;
	publishedAt: string;
	webUrl?: string;
	recipients: number;
	delivered: number;
	deliveryRate: number;
	opens: number;
	openRate: number;
	clicks: number;
	clickRate: number;
	unsubscribes: number;
	unsubscribeRate: number;
	spamReports: number;
	webViews: number;
};

export type BeehiivPage = {
	newsletters: Newsletter[];
	seeded: boolean;
	subscribers: number | null;
	live: boolean;
	fresh: boolean;
	volume: {
		headline: number | null;
		chips: StatChip[];
		caption: string;
		meters: Meter[];
		foot: string;
		sends: SeriesPoint[];
		openRates: SeriesPoint[];
		avgOpenRate: number | null;
		totalRecipients: number;
		totalClicks: number;
		unsubscribes: number;
		spamReports: number;
		insight: { display: string; headline: string; body: string; frac: number };
	};
};
