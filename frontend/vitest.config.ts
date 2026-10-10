import { defineConfig } from 'vitest/config';
import { sveltekit } from '@sveltejs/kit/vite';
import tailwindcss from '@tailwindcss/vite';

export default defineConfig({
  plugins: [sveltekit(), tailwindcss()],
  test: {
    include: ['src/**/*.{test,spec}.{js,ts}'],
    globals: true,
    environment: 'jsdom',
    setupFiles: ['./src/test/setup.ts'],
    // CRITICAL: Force browser/client conditions for Svelte to enable client-side rendering
    // This prevents the "lifecycle_function_unavailable" error
    alias: {
      'svelte/internal/server': 'svelte/internal/client'
    },

    // Parallel test execution for faster runs (Vitest 4 syntax)
    pool: 'threads',
    poolMatchGlobs: [],
    singleThread: false,
    minThreads: 1,
    maxThreads: 4,
    // Note: cache.dir removed - Vitest 4 uses Vite's cacheDir automatically

    coverage: {
      provider: 'v8',
      reporter: ['text', 'json', 'html'],
      exclude: [
        'node_modules/',
        'src/test/',
        '**/*.d.ts',
        '**/*.config.*',
        '**/mockData',
        '**/*.test.{js,ts}',
        '**/*.spec.{js,ts}'
      ],
      all: true,
      lines: 80,
      functions: 80,
      branches: 80,
      statements: 80
    },
    testTimeout: 20000,
    hookTimeout: 20000
  },
  resolve: {
    alias: {
      $lib: '/src/lib',
      $app: '/node_modules/@sveltejs/kit/src/runtime/app'
    },
    // Force Svelte to resolve to browser build, not server build
    conditions: ['browser', 'default']
  }
});
