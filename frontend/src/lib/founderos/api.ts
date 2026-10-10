/**
 * The one way the operator pages talk to the backend: `/api/founderos/*` on the same
 * base BusinessOS uses for its legacy (unversioned) routes, with the session
 * cookie. Mutating requests carry BusinessOS's double-submit CSRF token, using
 * the helpers in $lib/api/base (cookie first, else GET /auth/csrf once).
 *
 *   const { connections } = await founderosFetch<ConnectionsBody>('/connections');
 *   await founderosFetch('/agents/inbox/run', { method: 'POST', json: {} });
 */
import { goto } from '$app/navigation';
import { getCSRFToken, initCSRF } from '$lib/api/base';
import { getLegacyApiBaseUrl } from '$lib/config/runtime';

export class FounderosApiError extends Error {
	readonly status: number;
	readonly body: unknown;
	constructor(status: number, message: string, body?: unknown) {
		super(`${message} (HTTP ${status})`);
		this.name = 'FounderosApiError';
		this.status = status;
		this.body = body;
	}
}

/** True when the bridge guard (FOUNDEROS_WRITES=0) refused a write. Routes keep
 *  their FounderOS v1 status codes (403, 409 or 502), so match the body's
 *  guarded/refused flag, or the guard's reason in its error text. */
export function isGuardRefusal(err: unknown): boolean {
	if (!(err instanceof FounderosApiError)) return false;
	const b = (err.body ?? {}) as { guarded?: unknown; refused?: unknown; error?: unknown; detail?: unknown };
	if (b.guarded === true || b.refused === true) return true;
	return [b.error, b.detail].some((v) => typeof v === 'string' && v.includes('FOUNDEROS_WRITES=0'));
}

/** The session is gone; the viewer has already been sent to /login. */
export class FounderosAuthError extends FounderosApiError {
	constructor(body?: unknown) {
		super(401, 'Session expired', body);
		this.name = 'FounderosAuthError';
	}
}

export type FounderosInit = Omit<RequestInit, 'body'> & {
	body?: BodyInit | null;
	/** Sent as a JSON body with Content-Type: application/json. */
	json?: unknown;
};

const MUTATING = new Set(['POST', 'PUT', 'PATCH', 'DELETE']);

export function founderosUrl(path: string): string {
	return `${getLegacyApiBaseUrl()}/founderos${path.startsWith('/') ? path : `/${path}`}`;
}

/** X-CSRF-Token for a state-changing method, fetching a token first if none is readable. */
export async function csrfHeaders(method: string): Promise<Record<string, string>> {
	if (!MUTATING.has(method.toUpperCase())) return {};
	if (!getCSRFToken()) await initCSRF();
	const token = getCSRFToken();
	return token ? { 'X-CSRF-Token': token } : {};
}

export function loginRedirectHref(): string {
	const here = typeof window === 'undefined' ? '/os' : window.location.pathname + window.location.search;
	return `/login?next=${encodeURIComponent(here)}`;
}

function messageOf(body: unknown, fallback: string): string {
	if (body && typeof body === 'object') {
		const b = body as Record<string, unknown>;
		for (const k of ['error', 'message', 'detail']) if (typeof b[k] === 'string' && b[k]) return b[k] as string;
	}
	if (typeof body === 'string' && body.trim()) return body.trim().slice(0, 200);
	return fallback;
}

async function readBody(res: Response): Promise<unknown> {
	const text = await res.text().catch(() => '');
	if (!text) return undefined;
	try {
		return JSON.parse(text);
	} catch {
		return text;
	}
}

export async function founderosFetch<T>(path: string, init: FounderosInit = {}): Promise<T> {
	const { json, headers: given, ...rest } = init;
	const method = (rest.method ?? 'GET').toUpperCase();
	const headers = new Headers(given);
	let body = rest.body;
	if (json !== undefined) {
		body = JSON.stringify(json);
		if (!headers.has('Content-Type')) headers.set('Content-Type', 'application/json');
	}
	for (const [k, v] of Object.entries(await csrfHeaders(method))) headers.set(k, v);

	const res = await fetch(founderosUrl(path), { ...rest, method, headers, body, credentials: 'include' });
	const parsed = await readBody(res);

	if (res.status === 401) {
		goto(loginRedirectHref());
		throw new FounderosAuthError(parsed);
	}
	if (!res.ok) throw new FounderosApiError(res.status, messageOf(parsed, res.statusText || 'Request failed'), parsed);
	return parsed as T;
}
