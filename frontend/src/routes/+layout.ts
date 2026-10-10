// Disable SSR for Electron builds (pure SPA mode)
// This prevents server-side code from running during static build
export const ssr = false;
export const prerender = false;
