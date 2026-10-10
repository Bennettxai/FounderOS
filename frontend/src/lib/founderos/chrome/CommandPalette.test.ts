// The ⌘K palette's Run and Ask groups (FounderOS v1 components/CommandPalette.tsx),
// driven through a mocked founderosFetch. The Go-to behaviour is in chrome.test.ts.
import { fireEvent, render, screen, waitFor } from '@testing-library/svelte';
import { tick } from 'svelte';
import { beforeEach, describe, expect, it, vi } from 'vitest';

vi.mock('../api', async (orig) => ({ ...(await orig<typeof import('../api')>()), founderosFetch: vi.fn() }));

import { FounderosApiError, founderosFetch } from '../api';
import CommandPalette from './CommandPalette.svelte';
import { get } from 'svelte/store';
import { REFUSED_NOTICE } from './conductor';
import { toasts } from './toast';

// Results land on the OS-wide toast stack (FounderOS v1 useToast): busy, then ok/err.
const last = () => get(toasts).at(-1);

const fetchMock = vi.mocked(founderosFetch);

const AGENTS = [
	{ id: 'inbox-triage', name: 'Inbox Triage', description: 'Sorts the inboxes', departmentId: 'communications', lastRun: null },
	{ id: 'markets', name: 'Markets Agent', description: 'Watches the sleeve', departmentId: 'finances', lastRun: null }
];
let post: (path: string, json: unknown) => unknown;
let agentsBody: unknown;

beforeEach(() => {
	toasts.set([]);
	agentsBody = { agents: AGENTS };
	post = () => ({});
	fetchMock.mockReset();
	fetchMock.mockImplementation(async (path: string, init?: { method?: string; json?: unknown }) => {
		if (init?.method === 'POST') return post(path, init.json) as never;
		if (path === '/agents') {
			if (agentsBody instanceof Error) throw agentsBody;
			return agentsBody as never;
		}
		throw new Error(`unexpected ${path}`);
	});
});

async function open() {
	await fireEvent.keyDown(window, { key: 'k', metaKey: true });
	await tick();
}

const input = () => screen.getByRole('combobox');
const groupTitles = () => [...document.querySelectorAll('[data-part="palette-group"]')].map((g) => g.textContent?.trim());

describe('CommandPalette: groups and scope', () => {
	it('loads the agent roster and shows Go to, Run and Ask groups', async () => {
		render(CommandPalette, { navigate: vi.fn() });
		await open();
		await waitFor(() => expect(screen.getByText('Inbox Triage')).toBeTruthy());
		expect(fetchMock).toHaveBeenCalledWith('/agents');
		expect(groupTitles()).toEqual(['Go to', 'Run', 'Ask']);
		expect(screen.getByText('Markets Agent')).toBeTruthy();
		expect(screen.getByText('What needs my attention?')).toBeTruthy();
	});

	it('fetches the roster once, not on every open', async () => {
		render(CommandPalette, { navigate: vi.fn() });
		await open();
		await waitFor(() => expect(screen.getByText('Inbox Triage')).toBeTruthy());
		await fireEvent.keyDown(window, { key: 'Escape' });
		await open();
		expect(fetchMock.mock.calls.filter(([p]) => p === '/agents')).toHaveLength(1);
	});

	it('scope chips narrow to one group; Tab cycles them', async () => {
		render(CommandPalette, { navigate: vi.fn() });
		await open();
		await waitFor(() => expect(screen.getByText('Inbox Triage')).toBeTruthy());
		await fireEvent.click(screen.getByRole('tab', { name: 'Run' }));
		expect(groupTitles()).toEqual(['Run']);
		expect(screen.getAllByRole('option')).toHaveLength(2);
		await fireEvent.keyDown(input(), { key: 'Tab' });
		expect(groupTitles()).toEqual(['Ask']);
		await fireEvent.keyDown(input(), { key: 'Tab' });
		expect(groupTitles()).toEqual(['Go to', 'Run', 'Ask']);
		expect(screen.getByRole('tab', { name: 'All' }).getAttribute('aria-selected')).toBe('true');
	});

	it('an unreachable roster says so in the Run scope instead of reading empty', async () => {
		agentsBody = new FounderosApiError(503, 'agent runtime unavailable');
		render(CommandPalette, { navigate: vi.fn() });
		await open();
		await fireEvent.click(screen.getByRole('tab', { name: 'Run' }));
		await waitFor(() => expect(screen.getByText(/agents unavailable: agent runtime unavailable/i)).toBeTruthy());
	});
});

describe('CommandPalette: Run', () => {
	it('"run inbox" + Enter runs the agent and shows the run summary', async () => {
		post = (path) => {
			expect(path).toBe('/agents/inbox-triage/run');
			return { id: 'r1', agentId: 'inbox-triage', ok: true, summary: '3 threads triaged, 1 needs you' };
		};
		render(CommandPalette, { navigate: vi.fn() });
		await open();
		await waitFor(() => expect(screen.getByText('Inbox Triage')).toBeTruthy());
		await fireEvent.input(input(), { target: { value: 'run inbox' } });
		await fireEvent.keyDown(input(), { key: 'Enter' });
		expect(screen.queryByRole('dialog')).toBeNull();
		expect(fetchMock).toHaveBeenCalledWith('/agents/inbox-triage/run', { method: 'POST' });
		await waitFor(() => expect(last()).toMatchObject({ kind: 'ok', text: 'Inbox Triage · done · 3 threads triaged, 1 needs you' }));
		// one toast per run: the busy one is updated in place
		expect(get(toasts)).toHaveLength(1);
	});

	it('an agent already running is not started a second time', async () => {
		post = () => new Promise(() => {});
		render(CommandPalette, { navigate: vi.fn() });
		for (let n = 0; n < 2; n++) {
			await open();
			await waitFor(() => expect(screen.getByText('Inbox Triage')).toBeTruthy());
			await fireEvent.input(input(), { target: { value: 'run inbox' } });
			await fireEvent.keyDown(input(), { key: 'Enter' });
		}
		expect(fetchMock.mock.calls.filter((c) => c[0] === '/agents/inbox-triage/run')).toHaveLength(1);
		expect(last()).toMatchObject({ kind: 'warn', text: 'Inbox Triage is already running' });
	});

	it('a run that reports not-ok reads as failed, with its summary', async () => {
		post = () => ({ ok: false, summary: 'IMAP login refused' });
		render(CommandPalette, { navigate: vi.fn() });
		await open();
		await waitFor(() => expect(screen.getByText('Markets Agent')).toBeTruthy());
		await fireEvent.click(screen.getByText('Markets Agent'));
		await waitFor(() => expect(last()).toMatchObject({ kind: 'err', text: 'Markets Agent failed: IMAP login refused' }));
	});

	it('a run the server rejects shows the reason', async () => {
		post = () => {
			throw new FounderosApiError(500, 'run exploded');
		};
		render(CommandPalette, { navigate: vi.fn() });
		await open();
		await waitFor(() => expect(screen.getByText('Markets Agent')).toBeTruthy());
		await fireEvent.click(screen.getByText('Markets Agent'));
		await waitFor(() => expect(last()).toMatchObject({ kind: 'err', text: 'Markets Agent failed: run exploded (HTTP 500)' }));
	});
});

describe('CommandPalette: Ask', () => {
	it('an Ask row sends its prompt to the Conductor chat', async () => {
		render(CommandPalette, { navigate: vi.fn() });
		await open();
		await fireEvent.click(screen.getByText('Plan my day'));
		await waitFor(() =>
			expect(fetchMock).toHaveBeenCalledWith('/pages/conductor/chat', {
				method: 'POST',
				json: { message: 'Plan my day from the calendar and the open work on the board.' }
			})
		);
		await waitFor(() => expect(last()).toMatchObject({ kind: 'ok', text: 'Sent to the Conductor · "Plan my day"' }));
	});

	it('no match + Enter asks the Conductor the typed question', async () => {
		render(CommandPalette, { navigate: vi.fn() });
		await open();
		await fireEvent.input(input(), { target: { value: 'why did revenue dip zzq' } });
		expect(screen.getByText(/press ↵ to ask the Conductor instead/)).toBeTruthy();
		await fireEvent.keyDown(input(), { key: 'Enter' });
		await waitFor(() =>
			expect(fetchMock).toHaveBeenCalledWith('/pages/conductor/chat', { method: 'POST', json: { message: 'why did revenue dip zzq' } })
		);
	});

	it('an unreachable Conductor reads as such', async () => {
		post = () => {
			throw new FounderosApiError(502, 'board down');
		};
		render(CommandPalette, { navigate: vi.fn() });
		await open();
		await fireEvent.click(screen.getByText('Plan my day'));
		await waitFor(() => expect(last()).toMatchObject({ kind: 'err', text: 'Conductor unreachable · board down (HTTP 502)' }));
	});

	it('a 409 refusal says writes are off', async () => {
		post = () => {
			throw new FounderosApiError(409, 'bridge writes are disabled', { refused: true });
		};
		render(CommandPalette, { navigate: vi.fn() });
		await open();
		await fireEvent.click(screen.getByText('Agent status report'));
		// held by the FOUNDEROS_WRITES guard: an honest held state, never "sent"
		await waitFor(() => expect(last()).toMatchObject({ kind: 'warn', text: REFUSED_NOTICE }));
	});

	it('the Brain row is a jump, not a message', async () => {
		const navigate = vi.fn();
		render(CommandPalette, { navigate });
		await open();
		await fireEvent.click(screen.getByText('Search the Brain'));
		expect(navigate).toHaveBeenCalledWith('/os/brain');
		expect(fetchMock).not.toHaveBeenCalledWith('/pages/conductor/chat', expect.anything());
	});
});
