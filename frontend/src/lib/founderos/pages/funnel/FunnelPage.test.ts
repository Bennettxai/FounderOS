import { fireEvent, render, screen, waitFor, within } from '@testing-library/svelte';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import type { ChannelAnalyticsBody, FunnelBody, FunnelNode, FunnelVolume, Journey, VslSnapshot } from './types';

vi.mock('$lib/founderos/api', () => {
	class FounderosApiError extends Error {
		status: number;
		constructor(status: number, message: string) {
			super(message);
			this.status = status;
		}
	}
	return { founderosFetch: vi.fn(), FounderosApiError };
});

const { founderosFetch, FounderosApiError } = await import('$lib/founderos/api');
const fetchMock = vi.mocked(founderosFetch);
const FunnelPage = (await import('./FunnelPage.svelte')).default;
const FunnelChannelAnalytics = (await import('./FunnelChannelAnalytics.svelte')).default;
const FunnelNodeCard = (await import('./FunnelNodeCard.svelte')).default;
const VslAnalytics = (await import('./VslAnalytics.svelte')).default;
const FunnelGraph = (await import('./FunnelGraph.svelte')).default;

// FounderOS v1 lib/funnel.ts FUNNEL_STAGES
const STAGES = [
	{ id: 'first_touch', label: 'First touch' },
	{ id: 'engaged', label: 'Engaged' },
	{ id: 'nurtured', label: 'Nurtured' },
	{ id: 'opted_in', label: 'Opted in' },
	{ id: 'converted', label: 'Converted' }
] as FunnelBody['stages'];
const LABELS = { first_touch: 'First touch', engaged: 'Engaged', nurtured: 'Nurtured', opted_in: 'Opted in', converted: 'Converted' } as const;
const CONSTANTS = { stallDays: 7, decayDays: 90, decayFadeStart: 21 };

function journey(id: string, over: Partial<Journey> = {}): Journey {
	return {
		id,
		name: id,
		venture: 'vantage',
		status: 'engaged',
		product: null,
		amountUsd: null,
		relationship: 'warm',
		likelihood: 50,
		url: null,
		email: `${id}@x.com`,
		phone: null,
		person: null,
		company: null,
		role: null,
		linkedin: null,
		createdAt: '2026-09-20',
		touches: [{ id: `${id}-t1`, contactId: id, seq: 1, stage: 'engaged', channel: 'call', label: 'Call booked: strategy', source: 'calendar', at: '2026-09-28' }],
		...over
	};
}

function node(j: Journey, over: Partial<FunnelNode> = {}): FunnelNode {
	return {
		...j,
		state: 'active',
		daysSinceLastTouch: 2,
		hubs: [1],
		currentHub: 1,
		radius: 4,
		decay: 0,
		segment: 0,
		rings: [1],
		currentRing: 1,
		origin: { segment: 'Instagram', entry: 'IG reel', channel: 'organic', source: 'trakyo', at: '2026-09-01' },
		...over
	} as FunnelNode;
}

const dana = journey('dana', { person: 'Dana Reyes', likelihood: 80, relationship: 'hot' });
const won = journey('won', { status: 'converted', amountUsd: 997, product: 'Cohort', venture: 'launchpad-cohort' });

/** v1 funnelVolume: every active lead, touches over the last 30 days. */
function vol(over: Partial<FunnelVolume> = {}): FunnelVolume {
	return {
		revenueUsd: 997,
		chips: [{ tone: 'ok', text: '1 closed' }, { tone: 'err', text: '1 stalled' }, { text: '1 archived' }],
		caption: 'closed revenue across 2 active clients · 1 organic / 0 ads entry',
		meters: [{ label: 'First touch → Engaged (2/2)', frac: 1, display: '100%', hue: 'var(--ramp-1)' }],
		foot: 'each bar is a stage over the one before · 50% end to end',
		series: [{ label: 'Sep 30', count: 3 }],
		touchesInWindow: 26,
		insight: { value: 1, headline: '1 to push · 0 to save', body: 'Dana Reyes', frac: 0.5 },
		window: null,
		leads: 2,
		...over
	};
}

function body(over: Partial<FunnelBody> = {}): FunnelBody {
	return {
		now: '2026-09-30T12:00:00Z',
		venture: null,
		stage: null,
		stages: STAGES,
		stageLabels: LABELS,
		acquisitions: [],
		summary: {
			clients: 2,
			converted: 1,
			revenueUsd: 997,
			stages: STAGES.map((s, i) => ({ stage: s.id, total: 2, organic: 1, ads: 0, conversionFromPrev: i === 0 ? null : 100 }))
		},
		journeys: [dana, won],
		archived: [journey('old', { name: 'Old Lead' })],
		table: [dana, won],
		stageCounts: { engaged: 1, converted: 1 },
		lastMessages: null,
		commsUnavailable: false,
		source: 'seed',
		isLive: false,
		liveLabel: '',
		laneErrors: { typeform: 'not configured' },
		seedError: null,
		radial: { nodes: [node(dana), node(won, { state: 'converted', currentHub: 4, hubs: [4] })], segments: [{ id: 'instagram', label: 'Instagram', count: 2, converted: 1 }] },
		attention: { pushNow: [dana], saveNow: [] },
		volume: vol(),
		// v1's control line (Attio (CRM) · GoHighLevel · Trakyo · Meta Ads), then any lane that is set up
		sources: [
			{ id: 'attio', name: 'Attio (CRM)', kind: 'crm', state: 'not_configured', detail: 'Attio is not connected · its past journeys sit in the archive', live: false, count: null },
			{ id: 'ghl', name: 'GoHighLevel', kind: 'crm', state: 'not_configured', detail: 'GoHighLevel is not connected · its past journeys sit in the archive', live: false, count: null },
			{ id: 'trakyo', name: 'Trakyo', state: 'not_configured', detail: 'Set TRAKYO_API_KEY', live: false, count: null },
			{ id: 'meta-ads', name: 'Meta Ads', state: 'not_configured', detail: 'Set META_ADS_TOKEN', live: false, count: null },
			{ id: 'calendar', name: 'Calendar', state: 'connected', detail: 'ok', live: true, count: 1 },
			{ id: 'stripe', name: 'Stripe', kind: 'payments', state: 'connected', detail: '50 successful payments in the last 90d', live: true, count: 50 },
			{ id: 'paykit', name: 'PayKit', kind: 'payments', state: 'connected', detail: '4 buyers paid in the last 90d', live: true, count: 4 }
		],
		constants: CONSTANTS,
		...over
	};
}

const analytics = (over: Partial<ChannelAnalyticsBody> = {}): ChannelAnalyticsBody => ({
	period: '30d',
	fetchedAt: '2026-09-30T12:00:00Z',
	content: { state: 'not_configured', rows: [], message: null },
	funnel: { state: 'not_configured', data: null, message: null },
	channels: [],
	...over
});

function route(map: Record<string, unknown | (() => unknown)>) {
	fetchMock.mockImplementation(async (path: string) => {
		for (const [prefix, value] of Object.entries(map)) {
			if (path.startsWith(prefix)) {
				const v = typeof value === 'function' ? (value as () => unknown)() : value;
				if (v instanceof Error) throw v;
				return v as never;
			}
		}
		throw new Error(`unmocked ${path}`);
	});
}

beforeEach(() => {
	fetchMock.mockReset();
	window.history.replaceState({}, '', '/os/funnel');
});
afterEach(() => vi.clearAllMocks());

describe('/os/funnel page', () => {
	it('shows the loading skeleton, then every section with honest demo labelling', async () => {
		let resolve!: (v: unknown) => void;
		fetchMock.mockImplementation(async (path: string) => {
			if (path.startsWith('/pages/funnel/analytics')) return analytics() as never;
			return new Promise((r) => (resolve = r)) as never;
		});
		render(FunnelPage);
		expect(await screen.findByTestId('funnel-loading')).toBeTruthy();
		resolve(body());
		expect(await screen.findByText('demo data')).toBeTruthy();
		expect(screen.getByRole('heading', { level: 1, name: 'Funnel' })).toBeTruthy();
		expect(screen.getByText('// client journeys · leads → conversations → sales')).toBeTruthy();
		expect(screen.getByText('Funnel Volume')).toBeTruthy();
		expect(screen.getByText('First touch → Engaged (2/2)')).toBeTruthy();
		expect(screen.getAllByTestId('journey-row')).toHaveLength(2);
		expect(screen.getByText(/2 active clients · 1 closed · \$997 revenue · 1 archived/)).toBeTruthy();
		// v1's source checks, in v1's order; an unconfigured one is ○ with its why on hover
		const sources = screen.getByTestId('funnel-sources');
		expect([...sources.children].map((c) => c.textContent?.trim()).slice(0, 4)).toEqual(['○ Attio (CRM)', '○ GoHighLevel', '○ Trakyo', '○ Meta Ads']);
		expect(within(sources).getByText(/Attio \(CRM\)/).getAttribute('title')).toContain('not connected');
		expect(within(sources).getByText(/Calendar 1/)).toBeTruthy();
		expect(within(sources).getByText(/Stripe 50/)).toBeTruthy();
		expect(within(sources).getByText(/PayKit 4/).getAttribute('title')).toBe('4 buyers paid in the last 90d');
		expect(fetchMock).toHaveBeenCalledWith('/pages/funnel');
		// v1 has no Channel Funnels on /funnel (Trakyo analytics live on /analytics)
		expect(screen.queryByText('Channel Funnels')).toBeNull();
		expect(fetchMock.mock.calls.some(([p]) => String(p).startsWith('/pages/funnel/analytics'))).toBe(false);
	});

	it('hub rings breathe out of step, 0.6s apart, as in prod FunnelSpace', async () => {
		route({ '/pages/funnel/analytics': analytics(), '/pages/funnel': body() });
		const { container } = render(FunnelPage);
		await screen.findByText('Funnel Volume');
		await waitFor(() => expect(container.querySelectorAll('.fn-hub-ring').length).toBeGreaterThan(1));
		const delays = [...container.querySelectorAll('.fn-hub-ring')].map((r) => (r as SVGElement).style.animationDelay);
		expect(delays.slice(0, 3)).toEqual(['0s', '0.6s', '1.2s']);
	});

	it('reads live when a lane answered, with its label; badge and converted pill sit right of the title as in v1', async () => {
		route({ '/pages/funnel/analytics': analytics(), '/pages/funnel': body({ isLive: true, liveLabel: 'Typeform 3 + Stripe 1 + PayKit 4', source: 'typeform+stripe+paykit' }) });
		render(FunnelPage);
		const badge = await screen.findByText('live · Typeform 3 + Stripe 1 + PayKit 4');
		// v1 SlabTitle `right`: the badge row is beside the title block, not inside it
		const titleBlock = screen.getByRole('heading', { level: 1, name: 'Funnel' }).parentElement!;
		expect(titleBlock.contains(badge)).toBe(false);
		const right = screen.getByTestId('funnel-title-badges');
		expect(right.contains(badge)).toBe(true);
		expect(right.contains(screen.getByText('1/2 converted · $997'))).toBe(true);
	});

	it('v1 below the graph: Funnel Volume, Lead Activity and Needs you today, then Push Now / Save Now', async () => {
		const fading = journey('fading', { person: 'Fading Fay', status: 'nurtured', likelihood: 42, touches: [{ id: 'f1', contactId: 'fading', seq: 1, stage: 'nurtured', channel: 'email', label: 'x', source: 'manual', at: '2026-09-20' }] });
		route({
			'/pages/funnel/lead-message': { message: null, unavailable: false },
			'/pages/funnel': body({ attention: { pushNow: [dana], saveNow: [fading] } })
		});
		render(FunnelPage);
		const volume = (await screen.findByText('Funnel Volume')).closest('.bn-card') as HTMLElement;
		expect(within(volume).getByText('2 active')).toBeTruthy();
		expect(within(volume).getByText('1 closed')).toBeTruthy();
		expect(within(volume).getByText('1 archived')).toBeTruthy();
		expect(within(volume).getByText('closed revenue across 2 active clients · 1 organic / 0 ads entry')).toBeTruthy();
		expect(within(volume).getByText('First touch → Engaged (2/2)')).toBeTruthy();
		expect(within(volume).getByText('each bar is a stage over the one before · 50% end to end')).toBeTruthy();
		const activity = screen.getByText('Lead Activity').closest('.bn-card') as HTMLElement;
		expect(within(activity).getByText('last 30 days')).toBeTruthy();
		expect(within(activity).getByText('touches across every journey: forms, bookings, calls, payments')).toBeTruthy();
		expect(screen.getByText('Needs you today')).toBeTruthy();
		expect(screen.getByText('1 to push · 0 to save')).toBeTruthy();
		// no private-build window toggle
		expect(screen.queryByRole('button', { name: '60d' })).toBeNull();
		// the two rails, v1 copy
		const push = screen.getByText('Push Now').closest('.bn-card') as HTMLElement;
		expect(within(push).getByText('hot + moving · close them')).toBeTruthy();
		const save = screen.getByText('Save Now').closest('.bn-card') as HTMLElement;
		expect(within(save).getByText('fading toward the archive · highest likelihood first')).toBeTruthy();
		expect(within(save).getByText('Fading Fay')).toBeTruthy();
		expect(within(save).getByText('Nurtured')).toBeTruthy();
		expect(within(save).getByText('42% likely')).toBeTruthy();
		// a row pins that lead's dossier in the graph above (a read, never an outward action)
		await screen.findByTestId('funnel-space');
		await fireEvent.click(within(push).getByText('Dana Reyes'));
		await waitFor(() => expect(document.querySelector('[data-node="dana"]')?.getAttribute('aria-pressed')).toBe('true'));
		expect(fetchMock.mock.calls.every(([, init]) => !init)).toBe(true);
	});

	it('both rails empty: v1 hides the Push Now / Save Now row', async () => {
		route({ '/pages/funnel': body({ attention: { pushNow: [], saveNow: [] } }) });
		render(FunnelPage);
		await screen.findByText('Funnel Volume');
		expect(screen.queryByText('Push Now')).toBeNull();
		expect(screen.queryByText('Save Now')).toBeNull();
	});

	it('an unreachable backend is an error, not an empty funnel', async () => {
		route({ '/pages/funnel': new Error('HTTP 502') });
		render(FunnelPage);
		expect((await screen.findByRole('alert')).textContent).toContain('HTTP 502');
		expect(screen.queryByText('Funnel Volume')).toBeNull();
	});

	it('surfaces an unreadable seed instead of reading as zero clients', async () => {
		route({ '/pages/funnel/analytics': analytics(), '/pages/funnel': body({ seedError: 'founderos Postgres is not wired', journeys: [], table: [] }) });
		render(FunnelPage);
		expect((await screen.findByText(/seeded funnel could not be read/)).textContent).toContain('founderos Postgres is not wired');
	});

	it('venture tabs and stage chips refetch with the backend filters', async () => {
		route({ '/pages/funnel/analytics': analytics(), '/pages/funnel': body() });
		render(FunnelPage);
		await screen.findByText('Funnel Volume');
		await fireEvent.click(screen.getByRole('button', { name: /^Vantage$/ }));
		await waitFor(() => expect(fetchMock).toHaveBeenCalledWith('/pages/funnel?venture=vantage'));
		await fireEvent.click(screen.getByRole('button', { name: /Engaged 1/ }));
		await waitFor(() => expect(fetchMock).toHaveBeenCalledWith('/pages/funnel?venture=vantage&stage=engaged'));
	});

	it('a selected stage shows each lead’s last message, or says the feed is down', async () => {
		window.history.replaceState({}, '', '/os/funnel?stage=engaged');
		route({
			'/pages/funnel/analytics': analytics(),
			'/pages/funnel': body({
				stage: 'engaged',
				table: [dana, journey('quiet')],
				lastMessages: { dana: { message: { source: 'email', title: 'Re', preview: 'see you Friday', ts: '2026-09-29T12:00:00Z' } }, quiet: { message: null } }
			})
		});
		render(FunnelPage);
		expect(await screen.findByText(/see you Friday/)).toBeTruthy();
		expect(screen.getByText('no thread on record')).toBeTruthy();
		expect(fetchMock).toHaveBeenCalledWith('/pages/funnel?stage=engaged');
	});

	it('a closed row with no amount reads value unknown, never $0', async () => {
		const nx = journey('nx', { status: 'converted', product: 'Cohort', amountUsd: null });
		route({ '/pages/funnel/analytics': analytics(), '/pages/funnel': body({ journeys: [dana, nx], table: [dana, nx] }) });
		render(FunnelPage);
		const rows = await screen.findAllByTestId('journey-row');
		const row = rows.find((r) => r.textContent?.includes('nx'))!;
		expect(row.textContent).toContain('value unknown');
		expect(row.textContent).not.toContain('$0');
	});

	it('the archive tab lists what decayed and the retired CRM history', async () => {
		const attio = journey('attio-1', {
			name: 'Historic Attio',
			touches: [{ id: 'a-t1', contactId: 'attio-1', seq: 1, stage: 'engaged', channel: 'crm', label: 'Historic booked call', source: 'attio', at: '2026-06-01' }]
		});
		const ghl = journey('ghl-1', {
			name: 'Historic GHL',
			touches: [{ id: 'g-t1', contactId: 'ghl-1', seq: 1, stage: 'engaged', channel: 'crm', label: 'GHL stage', source: 'ghl', at: '2026-06-01' }]
		});
		route({ '/pages/funnel/analytics': analytics(), '/pages/funnel': body({ archived: [journey('old', { name: 'Old Lead' }), attio, ghl] }) });
		render(FunnelPage);
		await screen.findByText('Funnel Volume');
		const btn = screen.getByRole('button', { name: 'Archive (3)' });
		expect(btn.getAttribute('title')).toBe('Leads quiet past the decay window rest here');
		await fireEvent.click(btn);
		const archive = await screen.findByTestId('funnel-archive');
		expect(within(archive).getByText('Old Lead')).toBeTruthy();
		expect(within(archive).getByText('Attio · archived')).toBeTruthy();
		expect(within(archive).getByText('GoHighLevel · archived')).toBeTruthy();
		expect(screen.getByText('3 quiet past 90 days')).toBeTruthy();
		expect(screen.queryByTestId('funnel-graph')).toBeNull();
		expect(screen.queryByText('Lead Activity')).toBeNull();
	});
});

describe('FunnelGraph: the heavy engines load lazily', () => {
	it('holds a same-aspect skeleton, then mounts the flow engine; radial swaps engines', async () => {
		const props = { nodes: body().radial.nodes, segments: body().radial.segments, summary: body().summary, stages: STAGES, stageLabels: LABELS, constants: CONSTANTS };
		const { rerender } = render(FunnelGraph, { layout: 'flow', ...props });
		const skeleton = screen.getByTestId('funnel-graph-skeleton');
		expect(skeleton.getAttribute('style')).toContain('1100 / 460');
		const space = await screen.findByTestId('funnel-space');
		expect(document.querySelectorAll('[data-node]')).toHaveLength(2);
		// v1's open space: 1100×460, each stage caption is reached · conversion from the stage before
		expect(space.querySelector('svg[role="img"]')?.getAttribute('viewBox')).toBe('0 0 1100 460');
		expect(within(space).getByText('First touch')).toBeTruthy();
		expect(within(space).getByText('2')).toBeTruthy(); // first touch: no conversion from a stage before
		expect(within(space).getAllByText('2 · 100%')).toHaveLength(4);
		// built to be filmed: a fullscreen button over the canvas
		expect(within(space).getByRole('button', { name: 'Fullscreen' })).toBeTruthy();
		await rerender({ layout: 'radial', ...props });
		const radial = await screen.findByTestId('funnel-radial');
		// bounded like prod's .funnel-radial-canvas (760px, centred), with its own fullscreen
		expect(radial.querySelector('svg[role="img"]')?.classList.contains('fn-radial-canvas')).toBe(true);
		expect(within(radial).getByRole('button', { name: 'Fullscreen' })).toBeTruthy();
	});
});

describe('funnel nodes are keyboard reachable', () => {
	it.each(['flow', 'radial'] as const)('%s: each node is a focusable button that Enter opens and Space toggles', async (layout) => {
		route({ '/pages/funnel/lead-message': { message: null, unavailable: false } });
		const props = { nodes: body().radial.nodes, segments: body().radial.segments, summary: body().summary, stages: STAGES, stageLabels: LABELS, constants: CONSTANTS };
		render(FunnelGraph, { layout, ...props });
		await screen.findByTestId(layout === 'flow' ? 'funnel-space' : 'funnel-radial');
		const el = document.querySelector('[data-node="dana"]') as SVGGElement;
		expect(el.getAttribute('role')).toBe('button');
		expect(el.getAttribute('tabindex')).toBe('0');
		expect(el.getAttribute('aria-label')).toContain('Dana Reyes');
		await fireEvent.keyDown(el, { key: 'Enter' });
		expect(await screen.findByText(/contact/i)).toBeTruthy();
		expect(el.getAttribute('aria-pressed')).toBe('true');
		await fireEvent.keyDown(el, { key: ' ' });
		expect(el.getAttribute('aria-pressed')).toBe('false');
	});
});

describe('FunnelNodeCard (the dossier)', () => {
	it('fetches the last message with name + email when it pins', async () => {
		route({ '/pages/funnel/lead-message': { message: { source: 'whatsapp', title: 'Dana', preview: 'deal!', ts: new Date().toISOString() }, unavailable: false } });
		render(FunnelNodeCard, { node: node(dana), stageLabels: LABELS, onclose: () => {} });
		expect(await screen.findByText(/via whatsapp · today · “deal!”/)).toBeTruthy();
		expect(fetchMock).toHaveBeenCalledWith('/pages/funnel/lead-message?name=Dana+Reyes&email=dana%40x.com');
		expect(screen.getByText('Instagram')).toBeTruthy();
	});
	it('a converted deal with no amount reads value unknown, never $0', async () => {
		route({ '/pages/funnel/lead-message': { message: null, unavailable: false } });
		const { container } = render(FunnelNodeCard, { node: node(journey('nx', { status: 'converted', product: 'Cohort', amountUsd: null }), { state: 'converted' }), stageLabels: LABELS, onclose: () => {} });
		expect(await screen.findByText('Cohort · value unknown')).toBeTruthy();
		expect(container.textContent).not.toContain('$0');
	});
	it('links from lead data render only when they are http(s)', async () => {
		route({ '/pages/funnel/lead-message': { message: null, unavailable: false } });
		const { container } = render(FunnelNodeCard, { node: node(journey('bad', { linkedin: 'javascript:alert(1)', url: 'javascript:alert(2)' })), stageLabels: LABELS, onclose: () => {} });
		await screen.findByText(/contact/i);
		const hrefs = [...container.querySelectorAll('a')].map((a) => a.getAttribute('href') ?? '');
		expect(hrefs.some((h) => /^javascript:/i.test(h))).toBe(false);
	});
	it('an unavailable comms feed says so', async () => {
		route({ '/pages/funnel/lead-message': { message: null, unavailable: true } });
		render(FunnelNodeCard, { node: node(dana), stageLabels: LABELS, onclose: () => {} });
		expect(await screen.findByText('comms feed unavailable')).toBeTruthy();
	});
});

const snap = (id: string, plays: number, over: Partial<VslSnapshot> = {}): VslSnapshot => ({
	videoId: id,
	title: `Video ${id}`,
	duration: '10:00',
	dateFrom: '2025-08-10',
	dateTo: '2026-09-24',
	capturedAt: '2026-09-24T10:00:00.000Z',
	plays,
	uniqueViewers: plays,
	impressions: plays,
	playRate: 0.3,
	averageWatched: 0.4,
	unmuteRate: null,
	bounceRate: 0.1,
	recordedConversions: 0,
	recordedRevenue: 0,
	audience: [{ second: 0, viewers: 10 }, { second: 60, viewers: 5 }],
	trend: [],
	trendInterval: 'month',
	source: 'vidalytics-browser-export',
	...over
});

describe('FunnelChannelAnalytics', () => {
	it('not configured reads as a connect prompt, never as zero revenue', async () => {
		route({ '/pages/funnel/analytics': analytics() });
		render(FunnelChannelAnalytics, {});
		expect(await screen.findByText(/Connect Trakyo with a server-side API key/)).toBeTruthy();
		expect(screen.queryByText('$0.00')).toBeNull();
	});

	it('a ready answer shows first-touch revenue, hand-off meters and the channel content; IG shows the VSL', async () => {
		const row = { id: 'ci_1', type: 'custom', name: 'IG bio link', source: 'Instagram', views: null, published_at: null, clicks: 40, visits: 30, form_submissions: 4, bookings: 2, revenue: { first_touch: '1200', last_touch: '0' } };
		route({
			'/pages/funnel/analytics': analytics({
				content: { state: 'ready', rows: [row], message: null },
				funnel: { state: 'ready', data: { currency: 'USD', attribution: 'first_touch', range: { start: '2026-09-01T00:00:00Z', end: '2026-09-30T00:00:00Z', timezone: 'America/Chicago' }, counts: { clicks: 40, visits: 30, form_submissions: 4, bookings: 2, closes: 1 }, revenue: '1200' }, message: null },
				channels: [{ id: 'instagram_bio', label: 'IG bio', items: [row], clicks: 40, visits: 30, forms: 4, bookings: 2, views: null, videosWithViews: 0, revenue: { first_touch: 1200, last_touch: 0 } }]
			})
		});
		render(FunnelChannelAnalytics, { vslSnapshots: [snap('a', 50)], igVideo: null });
		expect((await screen.findAllByText('$1,200.00')).length).toBeGreaterThan(0);
		expect(screen.getByText('Visits → forms (4/30)')).toBeTruthy();
		expect(screen.getByText('IG bio link')).toBeTruthy();
		expect(screen.getByText('Inside the Instagram VSL')).toBeTruthy();
		await fireEvent.change(screen.getByLabelText('Analytics period'), { target: { value: '7d' } });
		await waitFor(() => expect(fetchMock).toHaveBeenCalledWith('/pages/funnel/analytics?period=7d'));
	});

	it('channels with tracked content are boxes; empty ones fold into one quiet line', async () => {
		const row = { id: 'ci_1', type: 'custom', name: 'IG bio link', source: 'Instagram', views: null, published_at: null, clicks: 1240, visits: 30, form_submissions: 4, bookings: 2, revenue: { first_touch: '0', last_touch: '0' } };
		const card = (id: string, label: string, items: (typeof row)[]) => ({ id, label, items, clicks: items.length ? 1240 : 0, visits: 0, forms: 0, bookings: 0, views: null, videosWithViews: 0, revenue: { first_touch: 0, last_touch: 0 } });
		route({
			'/pages/funnel/analytics': analytics({
				content: { state: 'ready', rows: [row], message: null },
				channels: [card('instagram_bio', 'IG bio', [row]), card('youtube', 'YouTube', [])]
			})
		});
		render(FunnelChannelAnalytics, { showVsl: false });
		const box = await screen.findByRole('button', { name: /IG bio.*1 item · 1,240 clicks/ });
		expect(box.getAttribute('aria-pressed')).toBe('true');
		const quiet = screen.getByText('no tracked content').parentElement!;
		const yt = within(quiet).getByRole('button', { name: 'YouTube' });
		await fireEvent.click(yt);
		expect(yt.getAttribute('aria-pressed')).toBe('true');
		expect(screen.getByText(/No content attributed to YouTube/)).toBeTruthy();
	});

	it('an unreachable analytics route is an alert', async () => {
		route({ '/pages/funnel/analytics': new FounderosApiError(500, 'boom') });
		render(FunnelChannelAnalytics, { showVsl: false });
		expect((await screen.findByRole('alert')).textContent).toContain('Unable to load Trakyo analytics');
	});
});

describe('VslAnalytics', () => {
	it('plays headline, per-video rows, an unexported rate reads N/A', async () => {
		render(VslAnalytics, { snapshots: [snap('a', 30), snap('b', 70)], igVideo: null });
		expect(await screen.findByText('VSL Performance')).toBeTruthy();
		expect(screen.getAllByText('Video b').length).toBeGreaterThan(0);
		expect(screen.getByText('70 · 70%')).toBeTruthy();
		// the unexported unmute rate is unknown on its meter (no bar), N/A in the row copy
		expect(screen.getByText('Unmute rate').parentElement?.textContent).toContain('unknown');
	});
	it('assigning the IG video posts the association', async () => {
		route({ '/pages/analytics/vsl/association': { videoId: 'a' } });
		render(VslAnalytics, { snapshots: [snap('a', 30)], igVideo: null, compact: true });
		await fireEvent.click(await screen.findByRole('button', { name: 'Use this video for IG' }));
		await waitFor(() => expect(fetchMock).toHaveBeenCalledWith('/pages/analytics/vsl/association', { method: 'POST', json: { videoId: 'a' } }));
		expect(await screen.findByText('Saved as the Instagram funnel video.')).toBeTruthy();
	});
	it('an unreadable snapshot store says so', async () => {
		render(VslAnalytics, { snapshots: [], error: 'founderos Postgres is not wired' });
		expect(await screen.findByText(/Saved video snapshots unavailable/)).toBeTruthy();
	});
});

describe('production look (round 2): rounded controls, venture hues, the sliding view pill', () => {
	const ready = (over: Partial<FunnelBody> = {}) =>
		route({ '/pages/funnel/analytics': analytics(), '/pages/funnel/lead-message': { message: null, unavailable: false }, '/pages/funnel': body(over) });

	it('venture dots wear the brand hues (Vantage #00ffaa, Launchpad Cohort #d9263f) as round dots', async () => {
		ready();
		render(FunnelPage);
		await screen.findByText('Funnel Volume');
		const rows = screen.getAllByTestId('journey-row');
		const dot = (row: HTMLElement) => row.querySelector('[data-testid="venture-dot"]') as HTMLElement;
		expect(dot(rows[0]).style.background).toBe('rgb(0, 255, 170)'); // #00ffaa
		expect(dot(rows[1]).style.background).toBe('rgb(217, 38, 63)'); // #d9263f
		expect(dot(rows[0]).classList.contains('rounded-full')).toBe(true);
		const tab = screen.getByRole('button', { name: 'Launchpad Cohort' });
		const tabDot = tab.querySelector('span') as HTMLElement;
		expect(tabDot.style.background).toBe('rgb(217, 38, 63)');
		expect(tabDot.classList.contains('rounded-full')).toBe(true);
	});

	it('pills are round: relationship, stage, touch chips, the converted pill and the value pill', async () => {
		ready();
		render(FunnelPage);
		await screen.findByText('Funnel Volume');
		const row = screen.getAllByTestId('journey-row')[1];
		expect(within(row).getByText('Converted').classList.contains('rounded-full')).toBe(true);
		expect(within(row).getByText('warm').classList.contains('rounded-full')).toBe(true);
		expect(within(row).getByText('$997').classList.contains('rounded-full')).toBe(true);
		const chip = within(row).getByText('Call booked: strategy').parentElement as HTMLElement;
		expect(chip.classList.contains('rounded-full')).toBe(true);
		expect(screen.getByText('1/2 converted · $997').classList.contains('rounded-full')).toBe(true);
		// rows carry the row lens like prod's JourneyRow
		expect(row.getAttribute('data-lens')).toBe('r');
	});

	it('venture tabs and the archive toggle are rounded pressable controls; stage chips are the kit filter chips', async () => {
		ready();
		render(FunnelPage);
		await screen.findByText('Funnel Volume');
		for (const name of ['All clients', 'Vantage', 'Archive (1)']) {
			const b = screen.getByRole('button', { name });
			expect(b.classList.contains('bn-pressable')).toBe(true);
			expect(b.getAttribute('data-lens')).toBe('c');
			expect(b.className).toContain('rounded-[var(--bn-r-ctl)]');
		}
		const all = screen.getByRole('button', { name: 'All 2' });
		expect(all.classList.contains('bn-filter-chip')).toBe(true);
		expect(all.classList.contains('is-on')).toBe(true);
		expect(all.classList.contains('rounded-full')).toBe(true);
		const closed = screen.getByRole('button', { name: /Converted 1/ });
		expect(closed.classList.contains('is-on')).toBe(false);
		expect((closed.querySelector('span') as HTMLElement).classList.contains('rounded-full')).toBe(true);
	});

	it('Flow / Radial is one rounded nav with a sliding pill that dims in the archive', async () => {
		ready();
		render(FunnelPage);
		await screen.findByText('Funnel Volume');
		const nav = screen.getByRole('navigation', { name: 'Funnel graph view' });
		expect(nav.className).toContain('rounded-[var(--bn-r-ctl)]');
		const slider = within(nav).getByTestId('layout-slider');
		expect(slider.getAttribute('style')).toContain('translateX(0%)');
		await fireEvent.click(within(nav).getByRole('button', { name: 'Radial' }));
		expect(slider.getAttribute('style')).toContain('translateX(100%)');
		expect(within(nav).getByRole('button', { name: 'Radial' }).getAttribute('aria-current')).toBe('page');
		await fireEvent.click(screen.getByRole('button', { name: 'Archive (1)' }));
		expect(within(nav).getByTestId('layout-slider').classList.contains('opacity-40')).toBe(true);
		expect(within(nav).getByRole('button', { name: 'Radial' }).getAttribute('aria-current')).toBeNull();
	});

	it('the archive rows wear the round venture dot at 60% and a round stage pill', async () => {
		ready();
		render(FunnelPage);
		await screen.findByText('Funnel Volume');
		await fireEvent.click(screen.getByRole('button', { name: 'Archive (1)' }));
		const archive = await screen.findByTestId('funnel-archive');
		const dot = archive.querySelector('[data-testid="venture-dot"]') as HTMLElement;
		expect(dot.classList.contains('rounded-full')).toBe(true);
		expect(dot.classList.contains('opacity-60')).toBe(true);
		expect(dot.style.background).toBe('rgb(0, 255, 170)');
		expect(within(archive).getByText('Engaged').classList.contains('rounded-full')).toBe(true);
	});

	it('the attention rows are row-lens links with round stage pills', async () => {
		ready();
		render(FunnelPage);
		const push = (await screen.findByText('Push Now')).closest('.bn-card') as HTMLElement;
		const row = within(push).getByRole('button', { name: /Dana Reyes/ });
		expect(row.getAttribute('data-lens')).toBe('r');
		expect(within(row).getByText('Engaged').classList.contains('rounded-full')).toBe(true);
	});
});

describe('production look (round 2): graph chrome and the dossier', () => {
	it('legend dots are round and the fullscreen button is a rounded pressable', async () => {
		const props = { nodes: body().radial.nodes, segments: body().radial.segments, summary: body().summary, stages: STAGES, stageLabels: LABELS, constants: CONSTANTS };
		const { rerender } = render(FunnelGraph, { layout: 'flow', ...props });
		const space = await screen.findByTestId('funnel-space');
		const legend = within(space).getByText(/hue = its stage/).querySelector('span') as HTMLElement;
		expect(legend.classList.contains('rounded-full')).toBe(true);
		const fs = within(space).getByRole('button', { name: 'Fullscreen' });
		expect(fs.className).toContain('rounded-[var(--bn-r-chip)]');
		expect(fs.classList.contains('bn-pressable')).toBe(true);
		await rerender({ layout: 'radial', ...props });
		const radial = await screen.findByTestId('funnel-radial');
		expect((within(radial).getByText(/hue = where they came from/).querySelector('span') as HTMLElement).classList.contains('rounded-full')).toBe(true);
	});

	it('the dossier is an 8px card; a stalled lead reads prod’s copy', async () => {
		route({ '/pages/funnel/lead-message': { message: null, unavailable: false } });
		render(FunnelNodeCard, { node: node(dana, { state: 'stalled' }), stageLabels: LABELS, onclose: () => {} });
		const card = screen.getByTestId('funnel-dossier');
		expect(card.className).toContain('rounded-[var(--bn-r-card)]');
		const stalled = screen.getByText('stalled — quiet past 7d before converting');
		expect(stalled.className).toContain('rounded-[var(--bn-r-chip)]');
	});
});

