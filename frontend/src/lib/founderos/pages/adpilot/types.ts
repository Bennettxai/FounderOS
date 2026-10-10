/** GET /api/founderos/pages/adpilot (internal/founderos/api/page_adpilot.go). */

export type GeoPoint = { city: string; country: string; lat: number; lng: number; leads: number; bookings: number };
export type DailyPoint = { date: string; spend: number; leads: number; bookings: number };
export type Audience = { age: Record<string, number>; gender: Record<string, number>; placements: Record<string, number> };

export type Campaign = {
	id: string;
	name: string;
	objective: 'leads' | 'bookings' | 'purchases';
	status: 'active' | 'paused';
	platform: string;
	period: { from: string; to: string };
	spend: number;
	impressions: number;
	clicks: number;
	leads: number;
	bookings: number;
	purchases: number;
	revenue: number;
	audience: Audience;
	geo: GeoPoint[];
	daily?: DailyPoint[];
};

/** Ratios are null when their denominator is zero: never 0 or Infinity. */
export type Metrics = {
	spend: number;
	impressions: number;
	clicks: number;
	leads: number;
	bookings: number;
	purchases: number;
	revenue: number;
	ctr: number | null;
	cpl: number | null;
	costPerBooking: number | null;
	costPerResult: number | null;
	roas: number | null;
};

export type DeckView = { metrics: Metrics; geo: GeoPoint[]; audience: Audience; daily: DailyPoint[] };

export type Deck = {
	campaigns: Campaign[];
	syncedAt: string | null;
	liveCount: number;
	/** "all" plus one view per campaign id; null when no campaigns exist. */
	views: Record<string, DeckView> | null;
};

export type WallAd = {
	id: string;
	brand: string;
	brandId: string | null;
	thumbnail: string | null;
	video: string | null;
	image: string | null;
	format: string | null;
	live: boolean;
	daysRunning: number;
	hook: string | null;
	hookSource: 'spoken' | 'text' | null;
	transcript: { t: number; s: string }[];
	drivers: Record<string, number>;
	ctaType: string | null;
	linkUrl: string | null;
};

export type WatchEntry = { id: string; name: string; domain?: string | null; avatar?: string | null; addedAt: string };
export type SavedAd = { ad: WallAd; savedAt: string };
export type SignalType = 'new_launch' | 'winner' | 'killed' | 'velocity_spike' | 'velocity_drop';
export type Signal = { type: SignalType; brandId: string; brandName: string; adId?: string; message: string; at: string };

export type Library = {
	wall: WallAd[];
	watchlist: WatchEntry[];
	saved: SavedAd[];
	signals: Signal[];
	/** null until the first sync has read the credit meter. */
	credits: { remaining: number; total: number } | null;
	lastSyncAt: string | null;
	/** FOREPLAY_API_KEY is planted. */
	configured: boolean;
};

export type AdpilotPayload = { deck: Deck; library: Library; errors: { campaigns?: string; store?: string } };

export type MineResponse = {
	ok: true;
	concept: string;
	expandedBy: 'claude' | 'fallback';
	probes: { query: string; results: number }[];
	pooled: number;
	apiCalls: number;
	winners: WallAd[];
};
