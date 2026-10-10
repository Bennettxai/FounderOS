import { isRedirect } from '@sveltejs/kit';
import { render, screen, waitFor } from '@testing-library/svelte';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { summarizeConnections, type Connection } from './connections';
import Placeholder from './chrome/PortingPlaceholder.svelte';
import { load as osLoad } from '../../routes/(founderos)/os/+layout';

const json = (body: unknown, status = 200) =>
	new Response(JSON.stringify(body), { status, headers: { 'content-type': 'application/json' } });

const conns: Connection[] = [
	{ id: 'email', name: 'Email', kind: 'email', state: 'connected', detail: '4 inboxes' },
	{ id: 'slack', name: 'Slack', kind: 'chat', state: 'connected', detail: '' },
	{ id: 'paperclip', name: 'Paperclip', kind: 'agents', state: 'error', detail: 'down' },
	{ id: 'typeform', name: 'Typeform', kind: 'forms', state: 'not_configured', detail: '' }
];

let fetchMock: ReturnType<typeof vi.fn>;
beforeEach(() => {
	fetchMock = vi.fn();
	vi.stubGlobal('fetch', fetchMock);
	localStorage.clear();
});
afterEach(() => vi.unstubAllGlobals());

describe('summarizeConnections', () => {
	it('counts connected, errors and not configured', () => {
		expect(summarizeConnections(conns)).toEqual({ up: 2, total: 4, errors: 1, notConfigured: 1 });
	});
});

describe('/os layout load', () => {
	it('uses the shared session check: no session → /login?next=<the /os page>', async () => {
		const err = (await (osLoad as (e: unknown) => Promise<unknown>)({
			fetch: vi.fn().mockResolvedValue(json({}, 401)),
			url: new URL('http://localhost/os/trading')
		}).catch((e: unknown) => e)) as { location: string };
		expect(isRedirect(err)).toBe(true);
		expect(err.location).toBe(`/login?next=${encodeURIComponent('/os/trading')}`);
	});
});

// The operator console itself is tested in $lib/founderos/pages/console.

describe('placeholder pages', () => {
	it('carry their PageHeader and say the port is in progress', () => {
		// Rendered directly: every page route is being replaced by its port.
		render(Placeholder, { href: '/os/comms' });
		expect(screen.getByRole('heading', { level: 1 }).textContent).toBe('Comms');
		expect(screen.getByText('Porting in progress')).toBeTruthy();
	});
});
