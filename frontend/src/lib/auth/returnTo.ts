/** Where /login sends you after signing in: ?redirect= (workspace invites)
 *  or ?next= (the session gate), same-site paths only. */
export function returnPath(params: URLSearchParams): string | null {
	for (const key of ['redirect', 'next']) {
		const r = params.get(key);
		if (r && r.startsWith('/') && !r.startsWith('//')) return r;
	}
	return null;
}
