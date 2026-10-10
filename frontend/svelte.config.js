import adapterNode from "@sveltejs/adapter-node";
import adapterStatic from "@sveltejs/adapter-static";
import { vitePreprocess } from "@sveltejs/vite-plugin-svelte";

// Determine which adapter to use based on environment
const isElectronBuild = process.env.ELECTRON_BUILD === "true";
// Cloudflare Pages: static adapter with SPA fallback (no SSR on edge)
const isCloudflareBuild = process.env.CLOUDFLARE_BUILD === "true";

/** @type {import('@sveltejs/kit').Config} */
const config = {
  // Consult https://svelte.dev/docs/kit/integrations
  // for more information about preprocessors
  preprocess: vitePreprocess(),

  kit: {
    // adapter-static  → Electron (file://) and Cloudflare Pages (SPA fallback)
    // adapter-node    → Cloud Run / Docker (SSR)
    adapter:
      isElectronBuild || isCloudflareBuild
        ? adapterStatic({
            pages: "build",
            assets: "build",
            fallback: "index.html", // SPA fallback for client-side routing
            precompress: false,
            strict: false,
          })
        : adapterNode(),
    // Disable prerendering for Electron and Cloudflare (pure SPA)
    prerender:
      isElectronBuild || isCloudflareBuild ? { entries: [] } : undefined,
    // Electron serves the build over the custom app://localhost protocol, which
    // has a real hostname. Absolute asset paths ("/_app/...") therefore resolve
    // the same way regardless of the current client-side route depth, so deep
    // routes (app://localhost/admin) load assets correctly. Do NOT use
    // { relative: true } here: relative ("./_app/...") refs are resolved against
    // the deep route path and 404 once the user navigates below root.
    paths: isElectronBuild ? { relative: false } : undefined,
  },
};

export default config;
