import { error, redirect } from '@sveltejs/kit';
import { getLegacyApiBaseUrl } from '$lib/config/runtime';

export type SessionData = { user: Record<string, unknown>; session: Record<string, unknown> };

/**
 * Client-side session gate for a layout load: validates the HttpOnly session
 * cookie with the backend (browser JS cannot read it) and redirects to /login
 * otherwise, keeping the intended destination in `?next=` when the backend
 * simply says no. Shared by the (app) and (founderos) route groups.
 */
export async function requireSession(fetchFn: typeof fetch, url: URL, opts: { retryDelayMs?: number } = {}): Promise<SessionData> {
	if (typeof localStorage !== 'undefined' && localStorage.getItem('businessos_logged_out') === '1') {
		throw redirect(302, '/login');
	}

	try {
		const check = () => fetchFn(`${getLegacyApiBaseUrl()}/auth/session`, { method: 'GET', credentials: 'include' });
		// A rate limit (429) or a server error says nothing about the session:
		// retry once, then show an error page. Only an answer of "no" (401/403
		// and the like) is a trip to /login.
		const transient = (r: Response) => r.status === 429 || r.status >= 500;
		let response = await check();
		if (transient(response)) {
			await new Promise((r) => setTimeout(r, opts.retryDelayMs ?? 600));
			response = await check();
			if (transient(response)) throw error(503, `The session check is unavailable (HTTP ${response.status}). Try again in a moment.`);
		}

		if (!response.ok) {
			const returnTo = encodeURIComponent(url.pathname + url.search);
			throw redirect(302, `/login?next=${returnTo}`);
		}

		const contentType = response.headers.get('content-type') || '';
		if (!contentType.includes('application/json')) {
			throw redirect(302, '/login');
		}

		const data = await response.json();
		if (!data?.user) {
			throw redirect(302, '/login');
		}

		return { user: data.user, session: data.session || { id: 'active' } };
	} catch (err) {
		// Re-throw SvelteKit redirects so they are handled correctly.
		if (err instanceof Response || (err != null && typeof err === 'object' && 'status' in err)) {
			throw err;
		}
		// Network / parse error: redirect to login rather than crash.
		throw redirect(302, '/login');
	}
}
