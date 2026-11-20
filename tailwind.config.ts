import type { Config } from 'tailwindcss';

// Terminal direction tokens — source of truth mirrored as CSS vars in globals.css
const config: Config = {
  content: ['./app/**/*.{ts,tsx}', './components/**/*.{ts,tsx}', './founderos/**/*.{ts,tsx}'],
  theme: {
    extend: {
      screens: {
        // Wide-screen tiers for the layout shell (see app/layout.tsx).
        // `wide` ≈ large desktop monitor, `ultra` ≈ 32" / ultrawide.
        wide: '1800px',
        ultra: '2200px',
      },
      colors: {
        // Tokens read CSS vars (defined in globals.css) so a single
        // data-theme flip on <html> re-themes every os.* class at once.
        os: {
          bg: 'var(--bg)',
          bg2: 'var(--bg-2)',
          surface: 'var(--surface)',
          // `raised` predates the revamp (= surface-2); /org still uses it
          raised: 'var(--surface-2)',
          surface2: 'var(--surface-2)',
          surface3: 'var(--surface-3)',
          border: 'var(--border)',
          // hairline row dividers inside lists/tables (Monolith handoff)
          hairline: 'var(--hairline)',
          // `border-bright` predates the revamp (= border-strong)
          'border-bright': 'var(--border-strong)',
          'border-strong': 'var(--border-strong)',
          text: 'var(--text)',
          muted: 'var(--text-2)',
          dim: 'var(--text-3)',
          accent: 'var(--accent)',
          accent2: 'var(--accent-2)',
          ink: 'var(--accent-ink)',
          ok: 'var(--ok)',
          warn: 'var(--warn)',
          err: 'var(--err)',
        },
      },
      fontFamily: {
        // Console two-voice lettering (2026-08-20): Geist leads everywhere —
        // headings, body, numerals — and Geist Mono does the system work
        // (ids, timestamps, code surfaces). Same family, so the voices never
        // clash. Brockmann + SF Pro retired with the Monolith skins.
        sans: ['var(--font-sans)', 'Geist', 'system-ui', 'sans-serif'],
        mono: ['var(--font-mono)', '"Geist Mono"', 'ui-monospace', 'SFMono-Regular', 'Menlo', 'monospace'],
      },
      borderRadius: {
        // Var-driven so radii are a THEME decision: every skin keeps its
        // sharp 0px analog edge except glados-glass, which restores the
        // rainforest radii (defined per-theme in globals.css).
        'sm-t': 'var(--r-input, 0px)',
        'md-t': 'var(--r-nav, 0px)',
        'lg-t': 'var(--r-card, 0px)',
      },
    },
  },
  plugins: [],
};

export default config;
