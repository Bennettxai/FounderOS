import { isHttpError, isRedirect } from '@sveltejs/kit';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { requireSession } from './requireSession';

const url = new URL('http://localhost/os/integrations?tab=all');
const json = (body: unknown, status = 200) =>
	new Response(JSON.stringify(body), { status, headers: { 'content-type': 'application/json' } });

async function redirectOf(p: Promise<unknown>): Promise<{ status: number; location: string }> {
	const err = await p.then(
		() => null,
		(e) => e
	);
	expect(isRedirect(err), `expected a redirect, got ${err}`).toBe(true);
	return err;
}

beforeEach(() => localStorage.clear());

describe('requireSession (shared by the (app) and (founderos) groups)', () => {
	it('validates the session cookie against /api/auth/session and returns the user', async () => {
		const fetchFn = vi.fn().mockResolvedValue(json({ user: { id: 'u1' }, session: { id: 's1' } }));
		const out = await requireSession(fetchFn, url);
		expect(out).toEqual({ user: { id: 'u1' }, session: { id: 's1' } });
		expect(fetchFn).toHaveBeenCalledWith('/api/auth/session', { method: 'GET', credentials: 'include' });
	});

	it('a session without an id still reads as active', async () => {
		const out = await requireSession(vi.fn().mockResolvedValue(json({ user: { id: 'u1' } })), url);
		expect(out.session).toEqual({ id: 'active' });
	});

	it('no session: 302 to /login keeping the destination in ?next=', async () => {
		const r = await redirectOf(requireSession(vi.fn().mockResolvedValue(json({}, 401)), url));
		expect(r.status).toBe(302);
		expect(r.location).toBe(`/login?next=${encodeURIComponent('/os/integrations?tab=all')}`);
	});

	it('an explicit logout skips the network and goes to /login', async () => {
		localStorage.setItem('businessos_logged_out', '1');
		const fetchFn = vi.fn();
		const r = await redirectOf(requireSession(fetchFn, url));
		expect(r.location).toBe('/login');
		expect(fetchFn).not.toHaveBeenCalled();
	});

	it('non-JSON, a missing user, or a network error all go to /login', async () => {
		const html = new Response('<html>', { status: 200, headers: { 'content-type': 'text/html' } });
		expect((await redirectOf(requireSession(vi.fn().mockResolvedValue(html), url))).location).toBe('/login');
		expect((await redirectOf(requireSession(vi.fn().mockResolvedValue(json({ user: null })), url))).location).toBe('/login');
		expect((await redirectOf(requireSession(vi.fn().mockRejectedValue(new TypeError('offline')), url))).location).toBe('/login');
	});

	it('a rate-limited check (429) is retried, not read as logged out', async () => {
		const fetchFn = vi.fn().mockResolvedValueOnce(json({ error: 'slow down' }, 429)).mockResolvedValueOnce(json({ user: { id: 'u1' } }));
		const out = await requireSession(fetchFn, url, { retryDelayMs: 0 });
		expect(out.user).toEqual({ id: 'u1' });
		expect(fetchFn).toHaveBeenCalledTimes(2);
	});

	it('a check that keeps failing server-side is an error page, never the login screen', async () => {
		for (const status of [429, 502, 503]) {
			const err = await requireSession(vi.fn().mockResolvedValue(json({}, status)), url, { retryDelayMs: 0 }).then(
				() => null,
				(e) => e
			);
			expect(isRedirect(err), `HTTP ${status} must not log you out`).toBe(false);
			expect(isHttpError(err) && err.status === 503, `HTTP ${status}`).toBe(true);
		}
	});
});

