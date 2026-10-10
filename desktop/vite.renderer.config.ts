import { defineConfig } from "vite";
import path from "path";
import fs from "fs";

// https://vitejs.dev/config
// This config serves the pre-built SvelteKit files without transformation
export default defineConfig({
  root: path.resolve(__dirname, "src/renderer"),
  // Disable all processing - serve files as-is
  optimizeDeps: {
    exclude: ["**/*"],
  },
  server: {
    port: 5199, // Fixed port — avoids conflicting with SvelteKit on 5173
    strictPort: true,
    fs: {
      strict: false,
      allow: [path.resolve(__dirname, "src/renderer")],
    },
  },
  build: {
    outDir: path.resolve(__dirname, ".vite/renderer/main_window"),
    emptyOutDir: true,
    // Minimal build - Vite will create index.html entry but we'll overwrite
    rollupOptions: {
      // Use index.html as entry to satisfy Vite
      input: path.resolve(__dirname, "src/renderer/index.html"),
      // Disable code splitting which causes the ../chunks issue
      output: {
        manualChunks: undefined,
      },
    },
    // Disable minification to avoid transforming the already-built code
    minify: false,
    // Don't process CSS
    cssCodeSplit: false,
  },
  plugins: [
    {
      name: "serve-static-assets",
      configureServer(server) {
        // Serve static files directly without processing
        server.middlewares.use((req, res, next) => {
          // Let Vite handle the request normally for static files
          next();
        });
      },
    },
    {
      name: "copy-sveltekit-assets",
      closeBundle: async () => {
        // Copy the _app folder which contains all the pre-built SvelteKit assets
        const srcAppDir = path.resolve(__dirname, "src/renderer/_app");
        const destAppDir = path.resolve(
          __dirname,
          ".vite/renderer/main_window/_app",
        );

        const copyRecursive = (src: string, dest: string) => {
          if (!fs.existsSync(src)) return;

          if (fs.statSync(src).isDirectory()) {
            if (!fs.existsSync(dest)) {
              fs.mkdirSync(dest, { recursive: true });
            }
            for (const file of fs.readdirSync(src)) {
              copyRecursive(path.join(src, file), path.join(dest, file));
            }
          } else {
            fs.copyFileSync(src, dest);
          }
        };

        copyRecursive(srcAppDir, destAppDir);

        // Copy EVERYTHING else from the built SvelteKit output (images,
        // logos, icons, service worker, …). A hardcoded file whitelist here
        // silently dropped every static asset added to the frontend after
        // the list was written (app-logos/, logos/, logo.png, og-image, …),
        // shipping desktop builds with missing images on every platform.
        const rendererRoot = path.resolve(__dirname, "src/renderer");
        for (const entry of fs.readdirSync(rendererRoot)) {
          if (entry === "_app") continue; // already copied above
          copyRecursive(
            path.join(rendererRoot, entry),
            path.resolve(__dirname, ".vite/renderer/main_window", entry),
          );
        }

        console.log("Copied SvelteKit assets to renderer output");
      },
    },
  ],
});
