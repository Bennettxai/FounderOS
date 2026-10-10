import { error } from '@sveltejs/kit';
import type { PageLoad } from './$types';

// An unknown /os path 404s INSIDE the /os chrome, the way FounderOS v1's
// not-found renders inside its root layout (sidebar, topbar and all).
export const load: PageLoad = () => {
	error(404, 'This page could not be found.');
};
