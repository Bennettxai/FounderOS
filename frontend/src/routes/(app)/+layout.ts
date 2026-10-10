import type { LayoutLoad } from "./$types";
import { requireSession } from "$lib/auth/requireSession";

// Client-side auth check for the (app) group.
// Replaces the deleted +layout.server.ts (which broke static/Cloudflare builds).
// The session check itself is shared with the (founderos) group.
export const load: LayoutLoad = async ({ fetch, url }) => {
  const isEmbed = url.searchParams.get("embed") === "true";

  if (isEmbed) {
    return { user: null, session: null };
  }

  return requireSession(fetch, url);
};

// This runs client-side, not server-side.
export const ssr = false;
