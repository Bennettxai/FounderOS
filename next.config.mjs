/** @type {import('next').NextConfig} */
const nextConfig = {
  // Isolate the build output dir via env so a production build can run on its
  // own port without clobbering a concurrent `next dev` (which keeps `.next`).
  distDir: process.env.NEXT_DIST_DIR || '.next',
  // Which product this deployment shows (lib/brand.ts). Inlined at build so
  // client components read the same value the server does.
  env: { GLADOS_BRAND: process.env.GLADOS_BRAND ?? '' },
  // Ad Engine was renamed Creative Engine (S02). Old bookmarks, history and
  // links keep working: 308, query string carried. S12's /reference rule
  // joins this same function (SPEC §5 C30).
  async redirects() {
    return [
      { source: '/ad-engine', destination: '/creative-engine', permanent: true },
      { source: '/ad-engine/:path*', destination: '/creative-engine/:path*', permanent: true },
      // Reference Model was removed (S12); Blueprint is the live map of the
      // same idea. 307 on purpose so browsers never cache it, should the
      // URL be wanted for something else later.
      { source: '/reference', destination: '/blueprint', permanent: false },
      // FounderOS kept lead magnets under /content/lead-magnets; GladOS has no
      // /content page (archived 2026-09-17), so the old path 308s (SPEC O2).
      { source: '/content/lead-magnets', destination: '/lead-magnets', permanent: true },
    ];
  },
  // Pre-rename URLs, served by the renamed targets (S02 FR-5, FR-14). The
  // API alias stays until both Mac mini instances run this build (TASKS).
  // Rewrites, not redirects, so POST/PATCH/DELETE bodies pass untouched and a
  // tab still running the old bundle keeps working. Both run afterFiles: a
  // file still at public/content-media/ad-engine/ (migration not yet run, or
  // the rename failed) is served as is; only a missing one is rewritten.
  async rewrites() {
    return [
      { source: '/api/ad-engine/:path*', destination: '/api/creative-engine/:path*' },
      { source: '/content-media/ad-engine/:path*', destination: '/content-media/creative-engine/:path*' },
    ];
  },
  // Skip static generation for dev-only and dynamic pages that use notFound() or have dynamic params that can't be resolved at build time.
  staticPageGenerationTimeout: 120,
  experimental: {
    serverComponentsExternalPackages: ['better-sqlite3', 'node-ical', 'nodemailer'],
    // instrumentation.ts: warms the live-read caches at boot (lib/warm.ts) and starts the cron tick when GLADOS_CRON=1 (SPEC O7).
    instrumentationHook: true,
  },
};

export default nextConfig;
