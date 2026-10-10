import { createRawSnippet } from 'svelte';

/** A children/right snippet for component tests: `<span>…</span>`. */
export function snip(html: string) {
	return createRawSnippet(() => ({ render: () => `<span>${html}</span>` }));
}
