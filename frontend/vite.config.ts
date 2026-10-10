import { sveltekit } from "@sveltejs/kit/vite";
import tailwindcss from "@tailwindcss/vite";
import { defineConfig, loadEnv } from "vite";

export default defineConfig(({ mode }) => {
  // Load .env, .env.development, .env.development.local — pull port overrides
  // even though they don't have the VITE_ prefix.
  const env = loadEnv(mode, process.cwd(), ["VITE_", "BACKEND_", "FRONTEND_"]);
  const BACKEND_PORT = env.BACKEND_PORT || "8001";
  const BACKEND_TARGET =
    env.BACKEND_TARGET || `http://localhost:${BACKEND_PORT}`;
  const FRONTEND_PORT = Number(env.FRONTEND_PORT || 5173);

  const proxy = (extra: Record<string, unknown> = {}) => ({
    target: BACKEND_TARGET,
    changeOrigin: true,
    ...extra,
  });

  return {
    plugins: [sveltekit(), tailwindcss()],
    esbuild: {
      drop: mode === "production" ? ["console", "debugger"] : [],
    },
    ssr: {
      // Force these CommonJS modules to be bundled for SSR
      noExternal: ["ms"],
    },
    optimizeDeps: {
      include: ["ms"],
    },
    server: {
      port: FRONTEND_PORT,
      strictPort: true,
      // Extra hostnames the dev server answers to (e.g. the Mac mini's
      // Tailscale name behind `tailscale serve`); unset keeps Vite's default.
      ...(env.FRONTEND_ALLOWED_HOSTS
        ? { allowedHosts: env.FRONTEND_ALLOWED_HOSTS.split(",").map((h) => h.trim()).filter(Boolean) }
        : {}),
      proxy: {
        // Versioned API (v1) - catch-all for /api/v1/*
        "/api/v1": proxy(),
        // Auth routes - proxied to Go backend
        "/api/auth": proxy(),
        // Terminal WebSocket - preserve Origin header for CORS
        "/api/terminal": {
          target: BACKEND_TARGET,
          changeOrigin: false,
          ws: true,
          configure: (p) => {
            p.on("error", (err) => console.log("[Vite Proxy] Error:", err));
            p.on("proxyReq", (_, req) =>
              console.log("[Vite Proxy] Request:", req.method, req.url),
            );
            p.on("proxyReqWs", (_, req) =>
              console.log("[Vite Proxy] WebSocket upgrade:", req.url),
            );
          },
        },
        "/api/chat": proxy(),
        "/api/projects": proxy(),
        "/api/contexts": proxy(),
        "/api/team": proxy(),
        "/api/dashboard": proxy(),
        "/api/mcp": proxy(),
        "/api/daily": proxy(),
        "/api/settings": proxy(),
        "/api/artifacts": proxy(),
        "/api/nodes": proxy(),
        "/api/clients": proxy(),
        "/api/deals": proxy(),
        "/api/transcribe": proxy(),
        "/api/voice-notes": proxy(),
        "/api/ai": proxy(),
        "/api/calendar": proxy(),
        "/api/integrations": proxy(),
        "/api/profile": proxy(),
        "/api/filesystem": proxy(),
        "/api/usage": proxy(),
        "/api/knowledge": proxy(),
        // OSA SSE streams: longer timeout, no buffering
        "/api/osa": proxy({
          timeout: 600000,
          configure: (p: any) => {
            p.on("proxyRes", (proxyRes: any) => {
              if (
                proxyRes.headers["content-type"]?.includes("text/event-stream")
              ) {
                proxyRes.headers["x-accel-buffering"] = "no";
                proxyRes.headers["cache-control"] = "no-cache";
              }
            });
          },
        }),
        "/api/apps": proxy(),
        "/api/documents": proxy(),
        "/api/memories": proxy(),
        "/api/learning": proxy(),
        "/api/app-profiles": proxy(),
        "/api/intelligence": proxy(),
        "/api/onboarding": proxy(),
        "/api/users": proxy(),
        // The operator bridge surface (desktop/backend-go/internal/founderos/api)
        "/api/founderos": proxy(),
        "/health": proxy(),
      },
    },
  };
});
