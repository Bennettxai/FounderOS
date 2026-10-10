/** A link from third-party data (ad libraries, funnel rows) is rendered only
 *  when it is an absolute http(s) URL; anything else (javascript:, data:,
 *  relative paths) is dropped rather than put in an href. */
export function safeHref(raw: string | null | undefined): string | null {
	const s = raw?.trim();
	if (!s || !/^https?:\/\//i.test(s)) return null;
	try {
		const u = new URL(s);
		return (u.protocol === 'http:' || u.protocol === 'https:') && u.hostname ? s : null;
	} catch {
		return null;
	}
}
