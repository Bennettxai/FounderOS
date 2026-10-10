import { get } from 'svelte/store';
import { tick } from 'svelte';
import { toasts } from '$lib/founderos/chrome/toast';
import { fireEvent, render, screen, waitFor, within } from '@testing-library/svelte';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import CommsRoute from '../../../../routes/(founderos)/os/comms/+page.svelte';
import type { CommsPage } from './model';

const json = (body: unknown, status = 200) => new Response(JSON.stringify(body), { status, headers: { 'content-type': 'application/json' } });

const NOW = '2026-09-30T15:00:00.000Z';
const ago = (h: number) => new Date(Date.parse(NOW) - h * 3_600_000).toISOString();

function fixture(over: Partial<CommsPage> = {}): CommsPage {
	return {
		now: NOW,
		sources: [
			{ id: 'email', name: 'Email', state: 'connected', detail: '4 inboxes · 3 unread' },
			{ id: 'whatsapp', name: 'WhatsApp', state: 'error', detail: 'stale: last push 3h ago' },
			{ id: 'slack', name: 'Slack', state: 'connected', detail: 'bot ok' },
			{ id: 'calendar', name: 'Calendar', state: 'connected', detail: '2 calendars' },
			{ id: 'plaud', name: 'Plaud', state: 'not_configured', detail: 'no token' }
		],
		lanes: [
			{
				id: 'inbox-1',
				name: 'Ops',
				source: 'email',
				state: 'connected',
				detail: '',
				unread: 1,
				items: [{ id: 'm1', sender: 'Acme Corp', preview: 'renewal terms', ts: ago(1), unread: 1, priority: 1, replyTo: 'ceo@acme.com' }]
			},
			{ id: 'whatsapp', name: 'WhatsApp', source: 'whatsapp', state: 'error', detail: 'stale', unread: 0, items: [] }
		],
		slackCards: [
			{ id: 's1', name: 'Vantage', channel: '#vantage', lastText: 'ship it', lastTs: ago(2), unread: 0, heat: 'hot', waiting: 'you', live: true },
			{ id: 's2', name: 'Quiet Co', channel: null, lastText: '', lastTs: null, unread: 0, heat: null, waiting: 'none', live: false }
		],
		slackRoster: { state: 'connected' },
		channels: [],
		channelsError: 'slack: invalid_auth',
		calendar: { accounts: [{ name: 'Ops', color: '#999' }], events: [], error: 'caldav: 401' },
		recordings: {
			recordings: [
				{ id: 'fathom-u', source: 'fathom', title: 'Discovery call', at: ago(3), durationMinutes: 42, url: 'https://fathom.video/calls/1' },
				{ id: 'plaud-p1', source: 'plaud', title: 'Site walk', at: ago(5), durationMinutes: 81, url: null, brain: 'store' }
			],
			sources: [
				{ id: 'plaud', name: 'Plaud', state: 'not_configured', detail: 'no token' },
				{ id: 'fathom', name: 'Fathom', state: 'connected', detail: 'key ok' }
			]
		},
		digest: {
			digest: {
				generatedAt: ago(6),
				windowHours: 24,
				total: 2,
				counts: { call: 1, client: 0, people: 1, branddeal: 0, group: 0, noise: 0 },
				unsubscribes: [{ sender: 'news@saas.io', count: 3, reason: 'bulk' }],
				entries: [
					{ tier: 'call', rank: 0, reason: 'call today', source: 'email', sender: 'Dana', title: 'Ops — Dana', preview: 'see you at 3', ts: ago(4), replyTo: 'dana@x.com' },
					{ tier: 'people', rank: 2, reason: 'family', source: 'whatsapp', sender: 'Mom', title: 'Mom', preview: 'dinner?', ts: ago(7), carried: true, firstSeenAt: ago(50) }
				]
			},
			sources: [
				{ source: 'email', ok: true, count: 1 },
				{ source: 'slack', ok: false, count: 0, error: 'invalid_auth' }
			],
			generatedAt: ago(6)
		},
		readKeys: [],
		volume: {
			headline: 1,
			chips: [{ tone: 'warn', text: '1 priority unread' }],
			caption: 'unread across 1 inbox + WhatsApp · 1 thread in view',
			meta: '1 unread · 3/5 sources connected · 0 meetings next 7 days · 2 recordings',
			meters: [{ label: 'Email (1)', frac: 1, display: '1 unread', hue: 'var(--bn-text-2)' }],
			foot: '0 meetings next 7 days · 2 recent recordings',
			series: [{ label: 'Sep 30', count: 1 }],
			seriesTotal: 1,
			meetings: [{ label: 'Wed', count: 0 }],
			meetingsTotal: 0,
			insight: { value: 2, headline: '1 client thread waiting on you · 1 priority unread.', body: 'Vantage · Acme Corp', frac: 0.66 }
		},
		feed: [],
		...over
	};
}

let fetchMock: ReturnType<typeof vi.fn>;
let calls: Array<{ url: string; method: string; body: unknown }>;
beforeEach(() => {
	calls = [];
	fetchMock = vi.fn(async (url: string, init: RequestInit = {}) => {
		const method = (init.method ?? 'GET').toUpperCase();
		calls.push({ url, method, body: init.body ? JSON.parse(String(init.body)) : undefined });
		if (url.endsWith('/pages/comms') && method === 'GET') return json(fixture());
		if (url.endsWith('/pages/comms/reply')) return json({ ok: false, error: 'bridge: outbound writes are disabled (FOUNDEROS_WRITES=0)' }, 502);
		if (url.endsWith('/pages/comms/digest/read')) return json({ ok: true });
		if (url.endsWith('/pages/comms/digest') && method === 'POST') return json({ ...fixture().digest, generatedAt: NOW });
		if (url.includes('/auth/csrf')) return json({ csrf_token: 't' });
		return json({ error: 'unexpected ' + url }, 500);
	});
	vi.stubGlobal('fetch', fetchMock);
});
afterEach(() => vi.unstubAllGlobals());

async function loaded() {
	const r = render(CommsRoute);
	await waitFor(() => expect(r.container.querySelectorAll('[data-part="source"]').length).toBe(5));
	return r;
}

describe('/os/comms', () => {
	it('fetches the page model once from /pages/comms', async () => {
		await loaded();
		expect(calls.filter((c) => c.url.endsWith('/founderos/pages/comms'))).toHaveLength(1);
	});

	it('hero: the five sources with honest states and the Message Volume card', async () => {
		const { container } = await loaded();
		expect(screen.getByRole('heading', { level: 1 }).textContent).toBe('Comms');
		expect(screen.getByText('1 unread · 3/5 sources connected · 0 meetings next 7 days · 2 recordings')).toBeTruthy();
		const wa = container.querySelector('[data-source="whatsapp"]')!;
		expect(wa.textContent).toContain('Error');
		expect(wa.textContent).toContain('stale: last push 3h ago');
		expect(container.querySelector('[data-source="plaud"]')!.textContent).toContain('Not configured');
		expect(screen.getByText('1 priority unread')).toBeTruthy();
		expect(screen.getByText('Email (1)')).toBeTruthy();
	});

	it('an unreachable calendar reads unknown, never 0 meetings', async () => {
		const { container } = await loaded();
		const meetings = container.querySelector('[data-part="meetings"]')!;
		expect(meetings.textContent).toContain('calendar unreachable · meetings unknown');
		expect(meetings.textContent).toContain('—');
	});

	it('morning report: tiers, held-over age, dead sources, and a tick persists read state', async () => {
		const { container } = await loaded();
		const digest = container.querySelector('[data-part="digest"]')!;
		expect(digest.textContent).toContain('2 left to clear');
		expect(digest.textContent).toContain('1 new · 1 held over');
		expect(digest.textContent).toContain('slack unavailable');
		expect(digest.textContent).toContain('You have a call with them');
		expect(digest.textContent).toMatch(/held 2d/);
		expect(within(digest as HTMLElement).getByText('news@saas.io')).toBeTruthy();

		await fireEvent.click(screen.getByLabelText('Mark Dana read'));
		await waitFor(() => expect(calls.some((c) => c.url.endsWith('/pages/comms/digest/read') && c.method === 'POST')).toBe(true));
		const post = calls.find((c) => c.url.endsWith('/pages/comms/digest/read'))!;
		expect(post.body).toEqual({ key: `email|Dana|${ago(4)}` });
		expect(digest.textContent).toContain('1 left to clear');
	});

	it('run now posts to /pages/comms/digest', async () => {
		await loaded();
		await fireEvent.click(screen.getByText('run now'));
		await waitFor(() => expect(calls.some((c) => c.url.endsWith('/pages/comms/digest') && c.method === 'POST')).toBe(true));
	});

	it('three-pane: select, read, and a reply refused by the guard shows the refusal', async () => {
		const { container } = await loaded();
		const row = container.querySelector('[data-part="msg-row"]')!;
		expect(row.textContent).toContain('Acme Corp');
		await fireEvent.click(row);
		const reader = container.querySelector('[data-part="reader"]')!;
		expect(reader.textContent).toContain('ceo@acme.com');
		await fireEvent.click(within(reader as HTMLElement).getByText('Reply'));
		await fireEvent.input(screen.getByLabelText('Reply', { selector: 'textarea' }), { target: { value: 'Sounds good' } });
		await fireEvent.click(screen.getByText('send'));
		await waitFor(() => expect(container.querySelector('[data-part="composer"]')!.textContent).toContain('FOUNDEROS_WRITES=0'));
		const sent = calls.find((c) => c.url.endsWith('/pages/comms/reply'))!;
		expect(sent.body).toEqual({ source: 'email', account: 'inbox-1', to: 'ceo@acme.com', subject: 'Re: renewal terms', text: 'Sounds good' });
	});

	it('a sent reply clears the draft so a second click cannot send it again', async () => {
		const base = fetchMock.getMockImplementation() as (url: string, init?: RequestInit) => Promise<Response>;
		fetchMock.mockImplementation(async (url: string, init: RequestInit = {}) => {
			if (url.endsWith('/pages/comms/reply')) {
				calls.push({ url, method: 'POST', body: JSON.parse(String(init.body)) });
				return json({ ok: true });
			}
			return base(url, init);
		});
		const { container } = await loaded();
		await fireEvent.click(container.querySelector('[data-part="msg-row"]')!);
		await fireEvent.click(within(container.querySelector('[data-part="reader"]') as HTMLElement).getByText('Reply'));
		const box = screen.getByLabelText('Reply', { selector: 'textarea' }) as HTMLTextAreaElement;
		await fireEvent.input(box, { target: { value: 'Sounds good' } });
		await fireEvent.click(screen.getByText('send'));
		await waitFor(() => expect(container.querySelector('[data-part="composer"]')!.textContent).toContain('sent'));
		expect(box.value).toBe('');
		const btn = screen.getByText('send').closest('button')!;
		expect(btn.disabled).toBe(true);
		await fireEvent.click(btn);
		expect(calls.filter((c) => c.url.endsWith('/pages/comms/reply'))).toHaveLength(1);
	});

	it('archive is an optimistic removal with undo', async () => {
		const { container } = await loaded();
		await fireEvent.click(container.querySelector('[data-part="msg-row"]')!);
		await fireEvent.click(screen.getByText('archive'));
		expect(container.querySelector('[data-part="msg-row"]')).toBeNull();
		// The OS-wide toaster (drawn by the /os layout) carries the confirmation.
		const t = get(toasts).at(-1)!;
		expect(t.text).toBe('Archived — Acme Corp');
		t.undo!();
		await tick();
		expect(container.querySelector('[data-part="msg-row"]')).toBeTruthy();
	});

	it('a disconnected lane says so instead of "clear"', async () => {
		const { container } = await loaded();
		await fireEvent.click(container.querySelector('[data-source="whatsapp"][aria-pressed]')!);
		expect(container.querySelector('[data-part="list-empty"]')!.textContent).toContain('WhatsApp is not connected');
	});

	it('Slack sources: client cards and an honest channels error', async () => {
		const { container } = await loaded();
		await fireEvent.click(container.querySelector('[data-source="slack-clients"]')!);
		const cards = container.querySelectorAll('[data-part="slack-card"]');
		expect(cards).toHaveLength(2);
		expect(cards[0].textContent).toContain('waiting on you');
		expect(cards[1].textContent).toContain('no channel linked');
		await fireEvent.click(container.querySelector('[data-source="slack-channels"]')!);
		expect(container.querySelector('[data-part="channels-empty"]')!.textContent).toContain('invalid_auth');
	});

	it('Recordings tab: newest first with the brain mark; Meetings tab names the calendar error', async () => {
		const { container } = await loaded();
		await fireEvent.click(screen.getByRole('tab', { name: /Recordings/ }));
		const rows = container.querySelectorAll('[data-part="recording"]');
		expect(rows).toHaveLength(2);
		expect(rows[0].textContent).toContain('Discovery call');
		expect(rows[1].textContent).toContain('in brain');
		expect(rows[1].textContent).toContain('1h21m');
		await fireEvent.click(screen.getByRole('tab', { name: /Meetings/ }));
		expect(container.querySelector('[data-part="calendar-error"]')!.textContent).toContain('caldav: 401');
	});

	describe('v1 layout (FounderOS v1 app/comms/page.tsx)', () => {
		afterEach(() => {
			try {
				window.localStorage.removeItem('founderos-os:comms:morning-report-open');
			} catch {
				/* ignore */
			}
		});

		it('slab title: eyebrow, 46px title, meta, unread badge and Open inbox on the right', async () => {
			const { container } = await loaded();
			expect(screen.getByText('// by source · inboxes · whatsapp · slack · calendar · recorder')).toBeTruthy();
			const h1 = screen.getByRole('heading', { level: 1 });
			expect(h1.textContent).toBe('Comms');
			expect(h1.className).toContain('text-[46px]');
			const title = h1.closest('.bn-rise')!;
			expect(title.textContent).toContain('1 unread · 3/5 sources connected');
			expect(title.textContent).toContain('1 unread');
			expect(title.contains(screen.getByText('Open inbox'))).toBe(true);
			expect(screen.getByText('Open inbox').getAttribute('href')).toBe('#inbox');
			expect(container.querySelector('[data-part="comms-head"]')).toBeNull();
		});

		it('hero row: a Sources card (N/5 connected) with full-detail source cards beside Message Volume', async () => {
			const { container } = await loaded();
			const hero = container.querySelector('[data-part="hero-row"]')!;
			const [sources, volume] = Array.from(hero.children);
			expect(sources.textContent).toContain('Sources');
			expect(sources.textContent).toContain('3/5 connected');
			expect(sources.querySelectorAll('[data-part="source"]')).toHaveLength(5);
			// the detail is printed on the card, not hidden in a tooltip
			const wa = sources.querySelector('[data-source="whatsapp"]')!;
			expect(wa.querySelector('p')!.textContent).toBe('stale: last push 3h ago');
			// connected sources wear a live dot and a Connected badge (v1)
			expect(sources.querySelector('[data-source="email"]')!.textContent).toContain('Connected');
			expect(volume.textContent).toContain('Message Volume');
			expect(volume.textContent).toContain('unread across 1 inbox + WhatsApp · 1 thread in view');
			expect(volume.textContent).toContain('0 meetings next 7 days · 2 recent recordings');
		});

		it('second row: Message Activity, Meetings This Week, Waiting on you; then the report and the inbox', async () => {
			const { container } = await loaded();
			const row = container.querySelector('[data-part="stat-row"]')!;
			const heads = Array.from(row.children).map((c) => c.textContent ?? '');
			expect(heads).toHaveLength(3);
			expect(heads[0]).toContain('Message Activity');
			expect(heads[0]).toContain('threads in view, last 14 days');
			expect(heads[1]).toContain('Meetings This Week');
			expect(heads[2]).toContain('Waiting on you');
			const digest = container.querySelector('[data-part="digest"]')!;
			const inbox = container.querySelector('#inbox')!;
			expect(row.compareDocumentPosition(digest) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
			expect(digest.compareDocumentPosition(inbox) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
			expect(container.querySelector('[data-part="footer-note"]')!.className).toContain('border-dashed');
		});

		it('the Inbox is a card titled "Inbox N unread", the tabs on their own row below', async () => {
			const { container } = await loaded();
			expect(screen.getByRole('heading', { name: 'Inbox' })).toBeTruthy();
			const inbox = container.querySelector('#inbox')!;
			expect(inbox.querySelector('[data-part="inbox-lead"]')).toBeNull();
			expect(inbox.querySelector('[data-part="tab-row"] [data-part="sliding-tabs"]')).toBeTruthy();
		});

		it('an unconfigured calendar and recorder read known-empty: 0 meetings, Recordings 0, no warning', async () => {
			const base = fetchMock.getMockImplementation() as (url: string, init?: RequestInit) => Promise<Response>;
			fetchMock.mockImplementation(async (url: string, init: RequestInit = {}) => {
				if (url.endsWith('/pages/comms') && (init.method ?? 'GET') === 'GET') {
					const f = fixture();
					return json({
						...f,
						calendar: { accounts: [], events: [] },
						recordings: { recordings: [], sources: f.recordings.sources },
						volume: { ...f.volume, meetings: [{ label: 'Fri', count: 0 }], meetingsTotal: 0 }
					});
				}
				return base(url, init);
			});
			const { container } = await loaded();
			const meetings = container.querySelector('[data-part="meetings"]')!;
			expect(meetings.textContent).toContain('meetings, next 7 days');
			expect(meetings.textContent).not.toContain('unknown');
			expect(container.querySelector('[data-part="recordings-chip"]')!.textContent).toMatch(/Recordings:\s*0/);
			expect(container.querySelector('[data-part="gaps"]')).toBeNull();
		});

		it('with no report yet, run now sits inline after the status, not pushed right (v1)', async () => {
			const base = fetchMock.getMockImplementation() as (url: string, init?: RequestInit) => Promise<Response>;
			fetchMock.mockImplementation(async (url: string, init: RequestInit = {}) => {
				if (url.endsWith('/pages/comms') && (init.method ?? 'GET') === 'GET') return json({ ...fixture(), digest: { digest: null, sources: [], generatedAt: null } });
				return base(url, init);
			});
			await loaded();
			expect(screen.getByText('no report yet — runs daily at 09:00')).toBeTruthy();
			expect(screen.getByText('run now').closest('button')!.className).not.toContain('ml-auto');
		});

		it('digest rows put title and preview on one line', async () => {
			const { container } = await loaded();
			const row = container.querySelector('[data-part="digest-row"]')!;
			const line = row.querySelector('[data-part="digest-line"]')!;
			expect(line.textContent).toBe('Ops — Dana · see you at 3');
		});

		it('a collapsed morning report stays collapsed across reloads', async () => {
			const first = await loaded();
			const toggle = screen.getByTestId('morning-report-toggle');
			expect(toggle.getAttribute('aria-expanded')).toBe('true');
			expect(toggle.getAttribute('aria-label')).toBe('Collapse morning report');
			await fireEvent.click(toggle);
			expect(window.localStorage.getItem('founderos-os:comms:morning-report-open')).toBe('0');
			first.unmount();

			const again = await loaded();
			await waitFor(() => expect(screen.getByTestId('morning-report-toggle').getAttribute('aria-expanded')).toBe('false'));
			expect(again.container.querySelector('[data-part="digest-row"]')).toBeNull();
		});
	});

	describe('prod shapes (rounded, pressable)', () => {
		it('header pill, source cards and the recordings chip round like v1', async () => {
			const { container } = await loaded();
			expect(screen.getByText('Open inbox').className).toContain('rounded-full');
			expect(container.querySelector('[data-part="recordings-chip"]')!.className).toContain('rounded-full');
			for (const c of container.querySelectorAll('[data-part="source"]')) {
				expect(c.className).toContain('rounded-[12px]');
				expect(c.className).toContain('bn-pressable');
			}
		});

		it('the tab row is prod SlidingTabs: a framed pill with a sliding marker and a solid count on the active tab', async () => {
			const { container } = await loaded();
			const tabs = container.querySelector('[data-part="sliding-tabs"]') as HTMLElement;
			expect(tabs.className).toContain('rounded-[8px]');
			expect((container.querySelector('[data-part="slider"]') as HTMLElement).style.transform).toBe('translateX(0px)');
			const active = screen.getByRole('tab', { name: /Messaging/ });
			expect(active.className).toContain('uppercase');
			expect(active.querySelector('[data-part="tab-count"]')!.getAttribute('style')).toContain('var(--bn-text)');
			await fireEvent.click(screen.getByRole('tab', { name: /Recordings/ }));
			expect((container.querySelector('[data-part="slider"]') as HTMLElement).style.transform).toBe('translateX(240px)');
		});

		it('morning report: a 2xl section, xl tier panels and tiles, round tier dots, a 6px tick and a pill reply link', async () => {
			const { container } = await loaded();
			expect(container.querySelector('[data-part="digest"]')!.className).toContain('rounded-2xl');
			for (const t of container.querySelectorAll('[data-part="tier-tile"]')) expect(t.className).toContain('rounded-xl');
			expect(container.querySelector('[data-tier="call"]')!.className).toContain('rounded-xl');
			expect(container.querySelector('[data-part="tier-dot"]')!.className).toContain('rounded-full');
			const row = container.querySelector('[data-part="digest-row"]')!;
			expect(row.querySelector('button')!.className).toContain('rounded-[6px]');
			expect(row.querySelector('a')!.className).toContain('rounded-full');
		});

		it('three-pane: 12px list and reader, 6px rail buttons, round dots', async () => {
			const { container } = await loaded();
			expect(container.querySelector('[data-part="list"]')!.className).toContain('rounded-[12px]');
			// prod's .os-slab .rounded-tile: 22px padding on the list and the reader
			expect(container.querySelector('[data-part="list"]')!.className).toContain('cm-tile');
			expect(container.querySelector('[data-part="reader-empty"]')!.className).toContain('cm-tile');
			// a message row must not wear the kit slab row hook (it would re-pad to 14px 24px)
			expect(container.querySelector('#inbox [data-part="row"]')).toBeNull();
			expect(container.querySelector('[data-part="reader-empty"]')!.className).toContain('rounded-[12px]');
			expect(container.querySelector('[data-part="rail"] button')!.className).toContain('rounded-[6px]');
			expect(container.querySelector('[data-part="priority-dot"]')!.className).toContain('rounded-full');
			expect(container.querySelector('[data-part="unread-dot"]')!.className).toContain('rounded-full');
			await fireEvent.click(container.querySelector('[data-part="msg-row"]')!);
			expect(container.querySelector('[data-part="reader"]')!.className).toContain('rounded-[12px]');
			// icons carry a CSS width, so the crowded action row cannot squeeze them to 0
			for (const svg of container.querySelectorAll('[data-part="reader"] button svg')) expect(svg.getAttribute('class')).toContain('w-3');
		});
	});

	it('an unreachable backend is an error, not an empty inbox', async () => {
		fetchMock.mockImplementation(async () => json({ error: 'backend down' }, 503));
		const { container } = render(CommsRoute);
		await waitFor(() => expect(container.querySelector('[data-part="page-error"]')).toBeTruthy());
		expect(container.textContent).toContain('backend down');
		expect(container.querySelector('[data-part="source"]')).toBeNull();
	});
});
