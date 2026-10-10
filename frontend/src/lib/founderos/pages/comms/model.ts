/**
 * /os/comms: the page model GET /api/founderos/pages/comms returns (built in
 * desktop/backend-go/internal/founderos/api/page_comms.go) and the pure
 * client-side derivations FounderOS v1 kept in lib/comms-panes.ts,
 * lib/calendar-layout.ts, lib/comms-digest.ts (entryKey, replyTarget) and
 * lib/recordings-format.ts. Pure, so archive/snooze stay optimistic (the
 * previous lanes reference IS the undo state) and every rule is unit-tested.
 */
import type { Meter, SeriesPoint, StatChip } from '$lib/founderos/kit';

export type ConnectorState = 'connected' | 'not_configured' | 'error' | string;
export type ConnectorStatus = { id: string; name: string; kind?: string; state: ConnectorState; detail: string };

export type LaneItem = {
	id: string;
	sender: string;
	preview: string;
	ts: string;
	unread: number;
	priority?: 1 | 2 | 3;
	replyTo?: string;
};
export type Lane = {
	id: string;
	name: string;
	source: 'email' | 'whatsapp';
	state: ConnectorState;
	detail: string;
	items: LaneItem[];
	unread: number;
};
export type SlackCard = {
	id: string;
	name: string;
	channel: string | null;
	lastText: string;
	lastTs: string | null;
	unread: number;
	heat: 'hot' | 'warm' | 'cold' | null;
	waiting: 'you' | 'them' | 'none';
	live: boolean;
};
export type SlackChannel = { id: string; name: string; isMember: boolean; isPrivate: boolean; members: number; topic: string };
export type CalEvent = {
	id: string;
	account: string;
	color: string;
	title: string;
	start: string;
	end: string | null;
	allDay: boolean;
	location: string | null;
	joinUrl: string | null;
};
export type Recording = {
	id: string;
	source: 'plaud' | 'fathom';
	title: string;
	at: string;
	durationMinutes: number | null;
	url: string | null;
	brain?: string | null;
};
export type DigestTier = 'call' | 'client' | 'people' | 'branddeal' | 'group' | 'noise';
export type DigestEntry = {
	tier: DigestTier;
	rank: number;
	reason: string;
	source: string;
	sender: string;
	title: string;
	preview: string;
	ts: string;
	replyTo?: string;
	account?: string;
	carried?: boolean;
	firstSeenAt?: string;
};
export type CommsDigest = {
	generatedAt: string;
	windowHours: number;
	entries: DigestEntry[];
	unsubscribes: Array<{ sender: string; count: number; reason: string }>;
	counts: Partial<Record<DigestTier, number>>;
	total: number;
};
export type DigestSourceState = { source: string; ok: boolean; count: number; error?: string };
export type DigestBody = {
	digest: CommsDigest | null;
	sources: DigestSourceState[];
	gaps?: string[];
	generatedAt: string | null;
	error?: string;
};
export type CommsVolume = {
	headline: number;
	chips: StatChip[];
	caption: string;
	meta: string;
	meters: Meter[];
	foot: string;
	series: SeriesPoint[];
	seriesTotal: number;
	meetings: SeriesPoint[];
	meetingsTotal: number;
	insight: { value: number; headline: string; body: string; frac: number };
};
export type CommsPage = {
	now: string;
	sources: ConnectorStatus[];
	lanes: Lane[];
	slackCards: SlackCard[];
	slackRoster: { state: ConnectorState; detail?: string };
	channels: SlackChannel[];
	channelsError?: string;
	calendar: { accounts: Array<{ name: string; color: string }>; events: CalEvent[]; error?: string };
	recordings: { recordings: Recording[]; sources: ConnectorStatus[]; errors?: string[] };
	digest: DigestBody;
	readKeys: string[];
	volume: CommsVolume;
	feed: unknown[];
	gaps?: string[];
};

// ---- lib/comms-panes.ts ------------------------------------------------------------

export type SourceKind = 'all' | 'email' | 'whatsapp' | 'slack-clients' | 'slack-channels';
export type PaneSource = { id: string; name: string; kind: SourceKind; unread: number; count: number; state?: ConnectorState };
export type PaneRow = LaneItem & { laneId: string; laneName: string };

const byTsDesc = (a: PaneRow, b: PaneRow) => Date.parse(b.ts) - Date.parse(a.ts);

export function buildCommsSources(lanes: Lane[], slackCards: SlackCard[], channels: SlackChannel[]): PaneSource[] {
	return [
		{
			id: 'all',
			name: 'All',
			kind: 'all',
			unread: lanes.reduce((n, l) => n + l.unread, 0),
			count: lanes.reduce((n, l) => n + l.items.length, 0)
		},
		...lanes.map((l) => ({ id: l.id, name: l.name, kind: l.source as SourceKind, unread: l.unread, count: l.items.length, state: l.state })),
		{ id: 'slack-clients', name: 'Slack · clients', kind: 'slack-clients', unread: slackCards.reduce((n, c) => n + c.unread, 0), count: slackCards.length },
		{ id: 'slack-channels', name: 'Slack · channels', kind: 'slack-channels', unread: 0, count: channels.length }
	];
}

export function itemsForSource(lanes: Lane[], sourceId: string): PaneRow[] {
	if (sourceId === 'all') {
		return lanes.flatMap((l) => l.items.map((it) => ({ ...it, laneId: l.id, laneName: l.name }))).sort(byTsDesc);
	}
	const lane = lanes.find((l) => l.id === sourceId);
	return lane ? lane.items.map((it) => ({ ...it, laneId: lane.id, laneName: lane.name })) : [];
}

export function removeItem(lanes: Lane[], itemId: string): Lane[] {
	return lanes.map((l) => {
		if (!l.items.some((it) => it.id === itemId)) return l;
		const items = l.items.filter((it) => it.id !== itemId);
		return { ...l, items, unread: items.reduce((n, it) => n + it.unread, 0) };
	});
}

// ---- lib/calendar-layout.ts --------------------------------------------------------

export type Interval = { startMin: number; endMin: number };
export type LaneSlot = { lane: number; lanes: number };

/** Side-by-side lanes for overlapping events (Google-Calendar style), aligned to input order. */
export function assignLanes(items: Interval[]): LaneSlot[] {
	const order = items.map((it, i) => ({ ...it, i })).sort((a, b) => a.startMin - b.startMin || a.endMin - b.endMin);
	const result: LaneSlot[] = new Array(items.length);
	let cluster: { i: number; startMin: number; endMin: number; lane: number }[] = [];
	let clusterEnd = -Infinity;
	const flush = () => {
		if (cluster.length === 0) return;
		const lanes = Math.max(...cluster.map((c) => c.lane)) + 1;
		for (const c of cluster) result[c.i] = { lane: c.lane, lanes };
		cluster = [];
	};
	for (const ev of order) {
		if (ev.startMin >= clusterEnd) {
			flush();
			clusterEnd = ev.endMin;
		} else {
			clusterEnd = Math.max(clusterEnd, ev.endMin);
		}
		const laneEnds: number[] = [];
		for (const c of cluster) laneEnds[c.lane] = Math.max(laneEnds[c.lane] ?? -Infinity, c.endMin);
		let lane = 0;
		while (laneEnds[lane] !== undefined && laneEnds[lane] > ev.startMin) lane++;
		cluster.push({ i: ev.i, startMin: ev.startMin, endMin: ev.endMin, lane });
	}
	flush();
	return result;
}

/** Visible hour window [loHour, hiHour) covering all events, min span enforced. */
export function hourBounds(items: Interval[], minSpanHours = 8, defaultLo = 8, defaultHi = 18): { loHour: number; hiHour: number } {
	if (items.length === 0) return { loHour: defaultLo, hiHour: defaultHi };
	let loHour = Math.max(0, Math.floor(Math.min(...items.map((i) => i.startMin)) / 60));
	let hiHour = Math.min(24, Math.ceil(Math.max(...items.map((i) => i.endMin)) / 60));
	if (hiHour - loHour < minSpanHours) {
		hiHour = Math.min(24, loHour + minSpanHours);
		loHour = Math.max(0, hiHour - minSpanHours);
	}
	return { loHour, hiHour };
}

export type Placed = { ev: CalEvent; startMin: number; endMin: number };
const dayKey = (d: Date) => `${d.getFullYear()}-${d.getMonth()}-${d.getDate()}`;

/** WeekCalendar's bucketing: 7 local-day columns from today, timed vs all-day. */
export function weekLayout(events: CalEvent[], nowISO: string) {
	const now = new Date(nowISO);
	const base = new Date(now);
	base.setHours(0, 0, 0, 0);
	const columns = Array.from({ length: 7 }, (_, i) => {
		const d = new Date(base);
		d.setDate(base.getDate() + i);
		return { date: d, key: dayKey(d) };
	});
	const colIndex = new Map(columns.map((c, i) => [c.key, i]));
	const timed: Placed[][] = columns.map(() => []);
	const allDay: CalEvent[][] = columns.map(() => []);
	for (const ev of events) {
		const s = new Date(ev.start);
		const idx = colIndex.get(dayKey(s));
		if (idx === undefined) continue;
		if (ev.allDay) {
			allDay[idx].push(ev);
			continue;
		}
		const e = ev.end ? new Date(ev.end) : new Date(s.getTime() + 30 * 60_000);
		const startMin = s.getHours() * 60 + s.getMinutes();
		let endMin = dayKey(e) !== dayKey(s) ? 24 * 60 : e.getHours() * 60 + e.getMinutes();
		if (endMin <= startMin) endMin = Math.min(24 * 60, startMin + 30);
		timed[idx].push({ ev, startMin, endMin });
	}
	const { loHour, hiHour } = hourBounds(timed.flat().map((p) => ({ startMin: p.startMin, endMin: p.endMin })));
	const nowMin = now.getHours() * 60 + now.getMinutes();
	return { columns, timed, allDay, loHour, hiHour, nowMin };
}

// ---- lib/comms-digest.ts -----------------------------------------------------------

export const TIER_ORDER: DigestTier[] = ['call', 'client', 'people', 'branddeal', 'group', 'noise'];

/** The stable id of one message (matches founderos_digest_reads keys). */
export function entryKey(e: Pick<DigestEntry, 'source' | 'sender' | 'ts'>): string {
	return `${e.source}|${e.sender}|${e.ts}`;
}

export type ReplyTarget = { kind: 'email' | 'whatsapp' | 'slack' | 'none'; href: string | null; label: string };

/** Where "reply" goes for an entry: a mailto, a wa.me deep link, a Slack channel. Never a fake link. */
export function replyTarget(e: Pick<DigestEntry, 'source' | 'sender' | 'replyTo'>): ReplyTarget {
	if (e.source === 'email') {
		const to = e.replyTo ?? (e.sender.includes('@') ? e.sender : null);
		return { kind: 'email', href: to ? `mailto:${to}` : null, label: 'Reply' };
	}
	if (e.source === 'whatsapp') {
		const digits = (e.replyTo ?? '').replace(/\D/g, '');
		return { kind: 'whatsapp', href: digits ? `https://wa.me/${digits}` : null, label: 'WhatsApp' };
	}
	if (e.source === 'slack') {
		const ch = (e.replyTo ?? '').replace(/^#/, '');
		return { kind: 'slack', href: ch ? `slack://channel?team=&id=${ch}` : null, label: 'Slack' };
	}
	return { kind: 'none', href: null, label: '' };
}

// ---- lib/recordings-format.ts ------------------------------------------------------

const PLAUD_SAMPLE_TITLES = new Set(['welcome to plaud.ai', 'how to use plaud', 'steve jobs & bill gates: a conversation that shaped technology']);

export function isPlaudSample(title: string): boolean {
	return PLAUD_SAMPLE_TITLES.has(title.trim().toLowerCase());
}

export function formatDuration(minutes: number | null): string {
	if (minutes === null || !Number.isFinite(minutes)) return '';
	if (minutes < 1) return '<1m';
	if (minutes < 60) return `${Math.round(minutes)}m`;
	const h = Math.floor(minutes / 60);
	const m = Math.round(minutes % 60);
	return m === 0 ? `${h}h` : `${h}h${String(m).padStart(2, '0')}m`;
}

/** RecordingsBoard's age label: "12m ago", "30h ago", "3d ago". */
export function relativeAge(ms: number): string {
	if (!Number.isFinite(ms) || ms < 0) return '';
	const m = Math.floor(ms / 60_000);
	if (m < 60) return `${m}m ago`;
	const h = Math.floor(m / 60);
	if (h < 48) return `${h}h ago`;
	return `${Math.floor(h / 24)}d ago`;
}
