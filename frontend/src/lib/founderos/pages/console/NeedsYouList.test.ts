import { fireEvent, render, screen, waitFor, within } from '@testing-library/svelte';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import NeedsYou from './NeedsYou.svelte';
import { SNOOZE_KEY } from './needs-queue';
import Toaster from '$lib/founderos/chrome/Toaster.svelte';
import { toasts } from '$lib/founderos/chrome/toast';

const json = (body: unknown, status = 200) =>
	new Response(JSON.stringify(body), { status, headers: { 'content-type': 'application/json' } });

const NOW = Date.now();
const iso = (msAgo: number) => new Date(NOW - msAgo).toISOString();

function classified(id: string, over: Record<string, unknown> = {}) {
	return {
		id,
		name: `${id}.md`,
		kind: 'file',
		url: null,
		meta: 'f7d1ec4a',
		modifiedAt: iso(8 * 86_400_000),
		sizeBytes: 10,
		accessCode: '',
		title: `Title ${id}`,
		summary: `Summary of ${id}`,
		revision: `file|${id}|10`,
		ask: 'draft',
		needsYou: true,
		deadline: null,
		overdue: false,
		label: 'draft to approve',
		action: '',
		person: false,
		glyph: '!',
		glyphTone: 'dim',
		why: 'a draft is waiting to go out',
		...over
	};
}

function payload(open: unknown[], extra: Record<string, unknown> = {}) {
	const files = Array.from({ length: 200 }, (_, i) => ({ ...classified(`f${i}`), kind: 'file' }));
	return {
		groups: [{ name: 'Agent files', items: files }],
		decisions: [],
		dir: '/w',
		available: true,
		reason: '',
		boardUrl: 'http://board.test:3100',
		needsYou: {
			open,
			decided: Array.from({ length: 24 }, (_, i) => ({ item: classified(`d${i}`), decision: { id: `d${i}`, decision: 'dismissed', decidedAt: '', decidedRevision: '', note: '' } }))
		},
		...extra
	};
}

const RETIRED = classified('ws/launch-draft.md', { title: 'RETIRED 2026-09-22: Do not send or revive this launch copy.' });
const NORTHWIND = classified('ws/fos-510-decision.md', {
	title: 'FOS-510 — Northwind/Hosting: the $1,050 decision, priced',
	ask: 'decision',
	label: 'your call',
	glyph: '!',
	glyphTone: 'warn',
	why: 'nobody else can make this call',
	modifiedAt: iso(11 * 86_400_000)
});

let fetchMock: ReturnType<typeof vi.fn>;
let posts: unknown[];
let decisionReply: () => Response;
function route(body: unknown, status = 200) {
	fetchMock.mockImplementation((url: string, init?: RequestInit) => {
		if (url.includes('/pages/board/deliverables/decision')) {
			posts.push(JSON.parse(String(init?.body)));
			return Promise.resolve(decisionReply());
		}
		if (url.includes('/pages/board/deliverables?file=')) return Promise.resolve(json({ id: 'x', name: 'x.md', kind: 'text', text: 'the draft', truncated: false }));
		if (url.includes('/pages/board/deliverables')) return Promise.resolve(json(body, status));
		return Promise.resolve(json({})); // the CSRF bootstrap
	});
}

beforeEach(() => {
	// the /os layout mounts the OS-wide toast stack; these tests mount it too
	render(Toaster);
	posts = [];
	decisionReply = () => json({ ok: true });
	fetchMock = vi.fn();
	vi.stubGlobal('fetch', fetchMock);
	try {
		window.localStorage.clear();
	} catch {
		/* no storage in this runner */
	}
});
afterEach(() => {
	vi.unstubAllGlobals();
	vi.restoreAllMocks();
	toasts.set([]);
});

describe('Home: Needs you (NeedsYouList.tsx)', () => {
	it('is the ranked queue: title, counts, open board, clear all, refresh and the Conductor divider', async () => {
		route(payload([RETIRED, NORTHWIND]));
		const { container } = render(NeedsYou);
		await waitFor(() => expect(screen.getByText(RETIRED.title as string)).toBeTruthy());
		// prod's Label in the card head, the slab sets it at 19px
		expect(within(container.querySelector('[data-part="card-head"]') as HTMLElement).getByText('Needs you')).toBeTruthy();
		// Home: the queue fills its row-mate's box and scrolls inside it
		const card = container.querySelector('[data-part="card"]') as HTMLElement;
		expect(card.className).toContain('h-full');
		expect(container.querySelector('[data-part="queue"]')?.className).toContain('min-[1201px]:flex-1');
		expect(screen.getByText(/2 of 200 agent files are waiting on you/).textContent).toContain('· 24 handled');
		expect(screen.getByRole('link', { name: /open board/ }).getAttribute('href')).toBe('http://board.test:3100');
		expect(screen.getByRole('button', { name: /clear all/ })).toBeTruthy();
		expect(screen.getByRole('button', { name: /refresh/ })).toBeTruthy();
		expect(screen.getByText('ranked by the Conductor · people first, then deadlines')).toBeTruthy();

		const rows = container.querySelectorAll('[data-part="row"]');
		expect(rows.length).toBe(2);
		const first = within(rows[0] as HTMLElement);
		expect(first.getByText('f7d1ec4a')).toBeTruthy();
		expect(first.getByText('8d ago')).toBeTruthy();
		expect(first.getByText('Summary of ws/launch-draft.md')).toBeTruthy();
		expect(first.getByText(/why here · a draft is waiting to go out/)).toBeTruthy();
		expect(first.getByRole('button', { name: 'publish' })).toBeTruthy();
		expect(first.getByRole('button', { name: 'dismiss' })).toBeTruthy();
		expect(first.getByRole('button', { name: 'snooze' })).toBeTruthy();
		expect(first.getByText('draft to approve')).toBeTruthy();
		const second = within(rows[1] as HTMLElement);
		expect(second.getByRole('button', { name: 'go ahead' })).toBeTruthy();
		expect(second.getByText('your call')).toBeTruthy();
		expect(second.getByLabelText('your call').textContent).toBe('!');
	});

	it('no board URL on this box: no open board link', async () => {
		route(payload([RETIRED], { boardUrl: null }));
		render(NeedsYou);
		await waitFor(() => expect(screen.getByText(RETIRED.title as string)).toBeTruthy());
		expect(screen.queryByRole('link', { name: /open board/ })).toBeNull();
	});

	it('publish goes through the bridge guard: writes off holds it, nothing is sent and the row stays', async () => {
		route(payload([RETIRED]));
		decisionReply = () => json({ error: 'bridge: outbound writes are disabled (FOUNDEROS_WRITES=0)', guarded: true }, 403);
		render(NeedsYou);
		await waitFor(() => expect(screen.getByRole('button', { name: 'publish' })).toBeTruthy());
		await fireEvent.click(screen.getByRole('button', { name: 'publish' }));
		await waitFor(() => expect(screen.getByRole('button', { name: 'held: writes off' })).toBeTruthy());
		expect(posts).toEqual([{ id: RETIRED.id, decision: 'approved', decidedRevision: RETIRED.revision }]);
		expect(screen.getByText(RETIRED.title as string)).toBeTruthy();
		expect((screen.getByRole('button', { name: 'held: writes off' }) as HTMLButtonElement).disabled).toBe(true);
		// answering in place never opens the review
		expect(document.querySelector('[data-part="review"]')).toBeNull();
	});

	it('with writes on, go ahead records the call and the row leaves with a toast', async () => {
		route(payload([NORTHWIND]));
		render(NeedsYou);
		await waitFor(() => expect(screen.getByRole('button', { name: 'go ahead' })).toBeTruthy());
		await fireEvent.click(screen.getByRole('button', { name: 'go ahead' }));
		await waitFor(() => expect(screen.getByText(`called · ${NORTHWIND.title}`).closest('[data-toast]')).toBeTruthy());
		expect(posts[0]).toEqual({ id: NORTHWIND.id, decision: 'approved', decidedRevision: NORTHWIND.revision });
	});

	it('dismiss leaves at once, counts as handled, and undo puts it back', async () => {
		route(payload([RETIRED, NORTHWIND]));
		const { container } = render(NeedsYou);
		await waitFor(() => expect(container.querySelectorAll('[data-part="row"]').length).toBe(2));
		const row = within(container.querySelectorAll('[data-part="row"]')[0] as HTMLElement);
		await fireEvent.click(row.getByRole('button', { name: 'dismiss' }));
		await waitFor(() => expect(container.querySelectorAll('[data-part="row"]').length).toBe(1));
		expect(screen.getByText(/1 of 200 agent files/).textContent).toContain('· 25 handled');
		await waitFor(() => expect(posts[0]).toEqual({ id: RETIRED.id, decision: 'dismissed', decidedRevision: RETIRED.revision }));
		expect(screen.getByText(`dismissed · ${RETIRED.title}`).closest('[data-toast]')).toBeTruthy();
		await fireEvent.click(screen.getByRole('button', { name: 'undo' }));
		await waitFor(() => expect(posts[posts.length - 1]).toEqual({ id: RETIRED.id, decision: null }));
		await waitFor(() => expect(container.querySelectorAll('[data-part="row"]').length).toBe(2));
	});

	it('snooze is a two-hour hold on his view only: nothing is posted', async () => {
		route(payload([RETIRED, NORTHWIND]));
		const { container } = render(NeedsYou);
		await waitFor(() => expect(container.querySelectorAll('[data-part="row"]').length).toBe(2));
		await fireEvent.click(within(container.querySelectorAll('[data-part="row"]')[1] as HTMLElement).getByRole('button', { name: 'snooze' }));
		await waitFor(() => expect(container.querySelectorAll('[data-part="row"]').length).toBe(1));
		expect(screen.getByText(/1 of 200 agent files/).textContent).toContain('· 1 snoozed');
		expect(posts).toEqual([]);
		expect(screen.getByText(`snoozed 2h · ${NORTHWIND.title}`)).toBeTruthy();
		expect(JSON.parse(window.localStorage.getItem(SNOOZE_KEY) ?? '{}')[NORTHWIND.id as string]).toBeGreaterThan(Date.now());
	});

	it('clear all takes two clicks and dismisses in bulk; there is no bulk approve', async () => {
		route(payload([RETIRED, NORTHWIND]));
		const { container } = render(NeedsYou);
		await waitFor(() => expect(screen.getByRole('button', { name: /clear all/ })).toBeTruthy());
		await fireEvent.click(screen.getByRole('button', { name: /clear all/ }));
		expect(posts).toEqual([]);
		await fireEvent.click(screen.getByRole('button', { name: /dismiss all 2/ }));
		await waitFor(() =>
			expect(posts[0]).toEqual({
				decision: 'dismissed',
				items: [
					{ id: RETIRED.id, decidedRevision: RETIRED.revision },
					{ id: NORTHWIND.id, decidedRevision: NORTHWIND.revision }
				]
			})
		);
		await waitFor(() => expect(container.querySelectorAll('[data-part="row"]').length).toBe(0));
		expect(screen.getByText('Nothing needs you.')).toBeTruthy();
	});

	it('refresh re-reads the board', async () => {
		route(payload([RETIRED]));
		render(NeedsYou);
		await waitFor(() => expect(screen.getByText(RETIRED.title as string)).toBeTruthy());
		const before = fetchMock.mock.calls.filter((c) => String(c[0]).endsWith('/pages/board/deliverables')).length;
		await fireEvent.click(screen.getByRole('button', { name: /refresh/ }));
		await waitFor(() => expect(fetchMock.mock.calls.filter((c) => String(c[0]).endsWith('/pages/board/deliverables')).length).toBe(before + 1));
	});

	it('clicking a row opens the review panel for the full text', async () => {
		route(payload([RETIRED]));
		const { container } = render(NeedsYou);
		await waitFor(() => expect(container.querySelectorAll('[data-part="row"]').length).toBe(1));
		await fireEvent.click(container.querySelector('[data-part="row"]') as HTMLElement);
		await waitFor(() => expect(document.querySelector('[data-part="review"]')).toBeTruthy());
		await waitFor(() => expect(screen.getByText('the draft')).toBeTruthy());
		await fireEvent.click(screen.getByRole('button', { name: 'Close review' }));
		await waitFor(() => expect(document.querySelector('[data-part="review"]')).toBeNull());
	});

	it('an empty queue reads like prod: Nothing needs you, no extra lines', async () => {
		route({ ...payload([]), groups: [], needsYou: { open: [], decided: [] }, available: false, reason: 'the board workspaces directory is not on this machine' });
		render(NeedsYou);
		await waitFor(() => expect(screen.getByText('Nothing needs you.')).toBeTruthy());
		expect(screen.getByText(/0 of 0 agent files are waiting on you/)).toBeTruthy();
		expect(screen.getByText('the Conductor will surface the next thing here')).toBeTruthy();
		expect(screen.queryByText(/not on this machine/)).toBeNull();
		expect(screen.queryByRole('button', { name: /clear all/ })).toBeNull();
	});

	it('an unreachable queue says so instead of an empty one', async () => {
		route({ error: 'boom' }, 500);
		render(NeedsYou);
		await waitFor(() => expect(screen.getByText(/deliverables unreachable/)).toBeTruthy());
		expect(screen.queryByText('Nothing needs you.')).toBeNull();
	});
});
