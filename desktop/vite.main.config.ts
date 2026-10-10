import { defineConfig } from "vite";
import path from "path";

// https://vitejs.dev/config
export default defineConfig({
  resolve: {
    alias: {
      "@": path.resolve(__dirname, "src"),
    },
  },
  build: {
    outDir: ".vite/build",
    rollupOptions: {
      external: ["electron", "better-sqlite3", "electron-store", "node-pty"],
      output: {
        entryFileNames: "index.js",
        format: "cjs",
      },
    },
  },
});
