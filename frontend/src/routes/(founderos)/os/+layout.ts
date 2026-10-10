import type { LayoutLoad } from './$types';
import { requireSession } from '$lib/auth/requireSession';

// Same client-side session gate as the (app) group.
export const load: LayoutLoad = ({ fetch, url }) => requireSession(fetch, url);

export const ssr = false;
