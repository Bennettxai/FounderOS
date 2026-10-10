import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

// Fresh module per test: base.ts caches the CSRF token at module level.
// The $app/navigation mock is re-created by resetModules, so read goto after it.
let goto: ReturnType<typeof vi.fn>;
async function load() {
	vi.resetModules();
	const mod = await import('./api');
	goto = vi.mocked((await import('$app/navigation')).goto) as unknown as ReturnType<typeof vi.fn>;
	goto.mockClear();
	return mod;
}

const json = (body: unknown, status = 200) =>
	new Response(JSON.stringify(body), { status, headers: { 'content-type': 'application/json' } });

function clearCookies() {
	for (const c of document.cookie.split(';')) {
		const name = c.split('=')[0].trim();
		if (name) document.cookie = `${name}=; expires=Thu, 01 Jan 1970 00:00:00 GMT; path=/`;
	}
}

let fetchMock: ReturnType<typeof vi.fn>;

beforeEach(() => {
	clearCookies();
	fetchMock = vi.fn();
	vi.stubGlobal('fetch', fetchMock);
	window.history.replaceState({}, '', '/os/integrations?tab=all');
});

afterEach(() => {
	vi.unstubAllGlobals();
});

describe('founderosFetch', () => {
	it('GETs /api/founderos<path> with the session cookie and returns the parsed body', async () => {
		const { founderosFetch } = await load();
		fetchMock.mockResolvedValueOnce(json({ connections: [] }));
		const out = await founderosFetch<{ connections: unknown[] }>('/connections');
		expect(out).toEqual({ connections: [] });
		const [url, init] = fetchMock.mock.calls[0];
		expect(url).toBe('/api/founderos/connections');
		expect(init.credentials).toBe('include');
		expect(new Headers(init.headers).get('X-CSRF-Token')).toBeNull();
	});

	it('a GET never asks for a CSRF token', async () => {
		const { founderosFetch } = await load();
		fetchMock.mockResolvedValueOnce(json({}));
		await founderosFetch('/guard');
		expect(fetchMock).toHaveBeenCalledTimes(1);
	});

	it('401 sends the viewer to /login?next=<here> and throws', async () => {
		const { founderosFetch, FounderosAuthError } = await load();
		fetchMock.mockResolvedValueOnce(json({ error: 'unauthorized' }, 401));
		await expect(founderosFetch('/connections')).rejects.toBeInstanceOf(FounderosAuthError);
		expect(goto).toHaveBeenCalledWith(`/login?next=${encodeURIComponent('/os/integrations?tab=all')}`);
	});

	it('other failures throw a typed error carrying the status and the server message', async () => {
		const { founderosFetch, FounderosApiError } = await load();
		fetchMock.mockResolvedValueOnce(json({ error: 'engine down' }, 503));
		const err = (await founderosFetch('/brain').catch((e: unknown) => e)) as InstanceType<typeof FounderosApiError>;
		expect(err).toBeInstanceOf(FounderosApiError);
		expect(err.status).toBe(503);
		expect(err.message).toContain('engine down');
		expect(err.message).toContain('503');
		expect(goto).not.toHaveBeenCalled();
	});

	it('a failure with a non-JSON body still throws with its status', async () => {
		const { founderosFetch, FounderosApiError } = await load();
		fetchMock.mockResolvedValueOnce(new Response('bad gateway', { status: 502 }));
		const err = (await founderosFetch('/x').catch((e: unknown) => e)) as InstanceType<typeof FounderosApiError>;
		expect(err).toBeInstanceOf(FounderosApiError);
		expect(err.status).toBe(502);
	});

	it('204 resolves to undefined', async () => {
		document.cookie = 'csrf_token=t; path=/';
		const { founderosFetch } = await load();
		fetchMock.mockResolvedValueOnce(new Response(null, { status: 204 }));
		await expect(founderosFetch('/x', { method: 'DELETE' })).resolves.toBeUndefined();
	});
});

describe('CSRF on mutating requests (BusinessOS double-submit, reused from $lib/api/base)', () => {
	it('sends the readable csrf_token cookie back as X-CSRF-Token', async () => {
		document.cookie = 'csrf_token=cookie-token; path=/';
		const { founderosFetch } = await load();
		fetchMock.mockResolvedValueOnce(json({ ok: true }));
		await founderosFetch('/agents/x/run', { method: 'POST', json: { dry: true } });
		expect(fetchMock).toHaveBeenCalledTimes(1);
		const [, init] = fetchMock.mock.calls[0];
		const headers = new Headers(init.headers);
		expect(headers.get('X-CSRF-Token')).toBe('cookie-token');
		expect(headers.get('Content-Type')).toBe('application/json');
		expect(init.body).toBe(JSON.stringify({ dry: true }));
	});

	it('without a cookie it first GETs the auth csrf endpoint, then uses that token', async () => {
		const { founderosFetch } = await load();
		fetchMock.mockResolvedValueOnce(json({ csrf_token: 'fresh-token' })).mockResolvedValueOnce(json({ ok: true }));
		await founderosFetch('/agents/x/run', { method: 'POST' });
		expect(fetchMock).toHaveBeenCalledTimes(2);
		expect(String(fetchMock.mock.calls[0][0])).toMatch(/\/api(\/v1)?\/auth\/csrf$/);
		expect(fetchMock.mock.calls[0][1].credentials).toBe('include');
		const [url, init] = fetchMock.mock.calls[1];
		expect(url).toBe('/api/founderos/agents/x/run');
		expect(new Headers(init.headers).get('X-CSRF-Token')).toBe('fresh-token');
	});

	it('csrfHeaders() is the reusable helper for other mutating calls', async () => {
		document.cookie = 'csrf_token=abc; path=/';
		const { csrfHeaders } = await load();
		expect(await csrfHeaders('PUT')).toEqual({ 'X-CSRF-Token': 'abc' });
		expect(await csrfHeaders('GET')).toEqual({});
	});
});

describe('dev proxy', () => {
	it('vite proxies /api/founderos to the backend (otherwise SvelteKit answers 404)', () => {
		const cfg = readFileSync(resolve(__dirname, '../../../vite.config.ts'), 'utf8');
		expect(cfg).toMatch(/"\/api\/founderos": proxy\(\)/);
	});
});

describe('isGuardRefusal', () => {
	it('recognises a bridge-guard refusal by its flag or reason, whatever the status', async () => {
		const { FounderosApiError, isGuardRefusal } = await load();
		const reason = 'bridge: outbound writes are disabled (FOUNDEROS_WRITES=0)';
		expect(isGuardRefusal(new FounderosApiError(403, 'x', { guarded: true }))).toBe(true);
		expect(isGuardRefusal(new FounderosApiError(409, 'x', { refused: true, guarded: true }))).toBe(true);
		expect(isGuardRefusal(new FounderosApiError(502, 'x', { ok: false, detail: reason }))).toBe(true);
		expect(isGuardRefusal(new FounderosApiError(409, 'x', { ok: false, route: 'task', error: reason }))).toBe(true);
	});

	it('an upstream failure is not a refusal', async () => {
		const { FounderosApiError, isGuardRefusal } = await load();
		expect(isGuardRefusal(new FounderosApiError(502, 'x', { ok: false, error: 'SMTP 550', guarded: false }))).toBe(false);
		expect(isGuardRefusal(new FounderosApiError(403, 'x', { error: 'forbidden' }))).toBe(false);
		expect(isGuardRefusal(new Error('network down'))).toBe(false);
	});
});
