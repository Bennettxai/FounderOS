import { fireEvent, render, waitFor, within } from '@testing-library/svelte';
import { beforeEach, describe, expect, it, vi } from 'vitest';

vi.mock('$lib/founderos/api', async (orig) => ({ ...(await orig<typeof import('$lib/founderos/api')>()), founderosFetch: vi.fn(), founderosUrl: (p: string) => `/api/founderos${p}` }));

import { founderosFetch } from '$lib/founderos/api';
import ChatsPage from './ChatsPage.svelte';
import { initials, railAgo } from './chats';
import type { ChatsView } from './types';

const fetchMock = vi.mocked(founderosFetch);

const view = (over: Partial<ChatsView> = {}): ChatsView => ({
	summaries: [],
	roster: [
		{ id: 'social-agent', name: 'Social Agent', description: 'Drafts hooks and captions for every platform' },
		{ id: 'gmail-worker', name: 'Gmail Worker', description: 'Sorts the four inboxes' }
	],
	conductorModel: null,
	boardUrl: null,
	llm: { configured: false, detail: 'Set AI_GATEWAY_API_KEY in ~/.founderos/.env to enable agent chat via the Vercel AI Gateway.' },
	...over
});

function route(overrides: Record<string, (init?: { method?: string; json?: unknown }) => unknown> = {}) {
	fetchMock.mockImplementation(async (path: string, init?: { method?: string; json?: unknown }) => {
		const key = `${(init?.method ?? 'GET').toUpperCase()} ${path}`;
		if (overrides[key]) return overrides[key](init);
		if (key === 'GET /pages/chats') return view();
		if (key === 'GET /pages/conductor/chat') return { messages: [], configured: false };
		if (key.startsWith('GET /pages/chats/')) return { messages: [] };
		return {};
	});
}

beforeEach(() => {
	fetchMock.mockReset();
});

describe('chats helpers (v1 ChatHub)', () => {
	it('initials take the first letter of up to two words', () => {
		expect(initials('Social Agent')).toBe('SA');
		expect(initials('Conductor · CEO')).toBe('CC');
		expect(initials('Reelkit')).toBe('R');
	});
	it('rail ages are compact', () => {
		const now = Date.parse('2026-10-09T12:00:00Z');
		expect(railAgo('2026-10-09T11:59:40Z', now)).toBe('now');
		expect(railAgo('2026-10-09T11:55:00Z', now)).toBe('5m');
		expect(railAgo('2026-10-09T09:00:00Z', now)).toBe('3h');
		expect(railAgo('2026-10-07T12:00:00Z', now)).toBe('2d');
	});
});

describe('/chats hub (v1 app/chats/page.tsx)', () => {
	it('is the Chats page: eyebrow, title, the Conductor pinned first and open, the empty-rail hint', async () => {
		route();
		const { getByText, findByText, container } = render(ChatsPage);
		expect(getByText('every conversation')).toBeTruthy();
		expect(getByText('Chats')).toBeTruthy();
		expect(await findByText('Conductor · CEO')).toBeTruthy();
		expect(getByText('pinned')).toBeTruthy();
		expect(getByText('the live board thread on the host')).toBeTruthy();
		expect(getByText('No direct chats yet. Hit New chat and pick an agent.')).toBeTruthy();
		expect(getByText('Conversations')).toBeTruthy();
		// the open thread is the board Conductor's cockpit rail, bare inside the hub shell
		const rail = container.querySelector('[data-part="conductor-rail"]') as HTMLElement;
		expect(rail).toBeTruthy();
		expect(rail.className).not.toContain('border');
		expect(within(rail).getByText('board · model unknown')).toBeTruthy();
		expect(within(rail).getByPlaceholderText('Message the CEO…')).toBeTruthy();
	});

	it('New chat flips the rail into the roster picker; search filters it; picking opens a fresh direct thread', async () => {
		route();
		const { findByTitle, getByText, getByPlaceholderText, queryByText, findByText } = render(ChatsPage);
		await fireEvent.click(await findByTitle('New chat'));
		expect(getByText('Social Agent')).toBeTruthy();
		expect(getByText('Gmail Worker')).toBeTruthy();
		await fireEvent.input(getByPlaceholderText('search threads'), { target: { value: 'gmail' } });
		expect(queryByText('Social Agent')).toBeNull();
		await fireEvent.click(getByText('Gmail Worker'));
		await waitFor(() => expect(fetchMock).toHaveBeenCalledWith('/pages/chats/gmail-worker'));
		expect(await findByText('Fresh thread with Gmail Worker. Say something.')).toBeTruthy();
		expect(getByText('direct · read-only tools')).toBeTruthy();
		expect(getByPlaceholderText('Message Gmail Worker…')).toBeTruthy();
	});

	it('with no model key a send says so honestly and keeps what you typed in the thread (as v1 does)', async () => {
		route({
			'POST /pages/chats/gmail-worker': () => ({ configured: false, error: 'AI_GATEWAY_API_KEY is not set — add it to ~/.founderos/.env to enable agent chat.', messages: [] })
		});
		const { findByTitle, getByText, getByPlaceholderText, findByText, queryByText } = render(ChatsPage);
		await fireEvent.click(await findByTitle('New chat'));
		await fireEvent.click(getByText('Gmail Worker'));
		const box = getByPlaceholderText('Message Gmail Worker…');
		await fireEvent.input(box, { target: { value: 'triage please' } });
		await fireEvent.keyDown(box, { key: 'Enter' });
		await waitFor(() => expect(fetchMock).toHaveBeenCalledWith('/pages/chats/gmail-worker', { method: 'POST', json: { message: 'triage please' } }));
		expect(await findByText(/AI_GATEWAY_API_KEY is not set/)).toBeTruthy();
		expect(queryByText('triage please')).toBeTruthy();
	});

	it('a configured reply renders the thread and moves the conversation to the top of the rail', async () => {
		route({
			'GET /pages/chats': () =>
				view({
					llm: { configured: true, detail: 'Vercel AI Gateway' },
					summaries: [{ agentId: 'social-agent', agentName: 'Social Agent', lastMessage: 'three hooks drafted', lastAt: '2026-10-09T10:00:00.000Z', messageCount: 2 }]
				}),
			'POST /pages/chats/gmail-worker': () => ({
				configured: true,
				reply: 'Four inboxes sorted.',
				messages: [
					{ id: 'm1', agentId: 'gmail-worker', role: 'user', content: 'triage please', toolCalls: [], createdAt: '2026-10-09T11:00:00.000Z' },
					{ id: 'm2', agentId: 'gmail-worker', role: 'assistant', content: 'Four inboxes sorted.', toolCalls: [], createdAt: '2026-10-09T11:00:02.000Z' }
				]
			})
		});
		const { findByText, findByTitle, getByText, getByPlaceholderText, container } = render(ChatsPage);
		expect(await findByText('three hooks drafted')).toBeTruthy();
		await fireEvent.click(await findByTitle('New chat'));
		await fireEvent.click(getByText('Gmail Worker'));
		const box = getByPlaceholderText('Message Gmail Worker…');
		await fireEvent.input(box, { target: { value: 'triage please' } });
		await fireEvent.keyDown(box, { key: 'Enter' });
		const thread = container.querySelector('[data-part="thread"]') as HTMLElement;
		expect(await within(thread).findByText('Four inboxes sorted.')).toBeTruthy();
		expect(within(thread).getByText('triage please')).toBeTruthy();
		const rows = [...container.querySelectorAll('[data-part="rail-row"]')].map((r) => r.getAttribute('data-agent'));
		expect(rows).toEqual(['__board_conductor__', 'gmail-worker', 'social-agent']);
	});

	it('the rail collapses to an avatar strip and expands back', async () => {
		route({
			'GET /pages/chats': () =>
				view({ summaries: [{ agentId: 'social-agent', agentName: 'Social Agent', lastMessage: 'hi', lastAt: '2026-10-09T10:00:00.000Z', messageCount: 1 }] })
		});
		const { findByTitle, queryByText, getByTitle } = render(ChatsPage);
		await fireEvent.click(await findByTitle('Collapse conversations'));
		expect(queryByText('Conversations')).toBeNull();
		expect(getByTitle('Social Agent').textContent?.trim()).toBe('SA');
		await fireEvent.click(getByTitle('Expand conversations'));
		expect(queryByText('Conversations')).toBeTruthy();
	});

	it('a failed view read is an honest line, not a blank', async () => {
		route({
			'GET /pages/chats': () => {
				throw new Error('founderos workspace missing');
			}
		});
		const { findByText } = render(ChatsPage);
		expect(await findByText(/Could not load the chats: founderos workspace missing/)).toBeTruthy();
	});
});
