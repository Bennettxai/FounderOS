import type { Handle } from "@sveltejs/kit";
import { env } from "$env/dynamic/private";

function setSecurityHeaders(response: Response): Response {
  response.headers.set("X-Frame-Options", "SAMEORIGIN");
  response.headers.set("X-Content-Type-Options", "nosniff");
  response.headers.set("Referrer-Policy", "strict-origin-when-cross-origin");
  response.headers.set(
    "Permissions-Policy",
    "camera=(), microphone=(self), geolocation=()",
  );
  response.headers.set("X-XSS-Protection", "1; mode=block");
  return response;
}

export const handle: Handle = async ({ event, resolve }) => {
  // Same-origin /api/* proxy for the adapter-node (Docker) deployment.
  // The web frontend always calls its own origin (see lib/config/runtime.ts);
  // in production Cloudflare Pages proxies /api/* to the backend and in dev
  // the Vite server does. This is the equivalent for the containerized node
  // server. Unset BACKEND_INTERNAL_URL disables it (e.g. behind an external
  // proxy).
  const backend = env.BACKEND_INTERNAL_URL;
  if (backend && event.url.pathname.startsWith("/api/")) {
    const target = new URL(
      event.url.pathname + event.url.search,
      backend,
    ).toString();
    const headers = new Headers(event.request.headers);
    headers.delete("host");
    headers.delete("connection");
    const upstream = await fetch(target, {
      method: event.request.method,
      headers,
      body:
        event.request.method === "GET" || event.request.method === "HEAD"
          ? undefined
          : await event.request.arrayBuffer(),
      redirect: "manual",
    });
    // Copy: headers on a fetch() Response are immutable.
    return setSecurityHeaders(new Response(upstream.body, upstream));
  }

  return setSecurityHeaders(await resolve(event));
};
