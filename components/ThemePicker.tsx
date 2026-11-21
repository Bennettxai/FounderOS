'use client';

import { useEffect, useState } from 'react';
import { Check } from 'lucide-react';
import { DEFAULT_THEME, THEMES, THEME_META, THEME_STORAGE_KEY, isTheme, type Theme } from '@/lib/theme';
import { classicMode, groundFor, isUiStyle } from '@/lib/appearance';

/**
 * OS theme picker (Settings → Appearance). Applies instantly by setting
 * `data-theme` on <html> and persists to localStorage — the same key the
 * pre-paint init script in layout reads, so the choice survives reloads
 * with no flash. Registry-driven: a new colorway needs zero changes here.
 */
export function ThemePicker() {
  const [current, setCurrent] = useState<Theme>(DEFAULT_THEME);

  useEffect(() => {
    const attr = document.documentElement.getAttribute('data-theme');
    if (isTheme(attr)) setCurrent(attr);
  }, []);

  const apply = (theme: Theme) => {
    const root = document.documentElement;
    root.setAttribute('data-theme', theme);
    // Under the Classic style the mode follows the theme (SPEC S00 FR-4).
    if (root.getAttribute('data-ui') !== 'glados') {
      root.setAttribute('data-mode-pref', classicMode(theme));
      root.setAttribute('data-mode', classicMode(theme));
    }
    // The glass light reads the ground the theme paints (lib/appearance.ts groundFor).
    const ui = root.getAttribute('data-ui');
    root.setAttribute('data-ground', groundFor(theme, root.getAttribute('data-mode') === 'light' ? 'light' : 'dark', isUiStyle(ui) ? ui : undefined));
    try {
      localStorage.setItem(THEME_STORAGE_KEY, theme);
    } catch {
      /* private mode — the attribute still applies for this session */
    }
    setCurrent(theme);
  };

  return (
    <div role="group" aria-label="Colour theme" className="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-3">
      {THEMES.map((theme) => {
        const meta = THEME_META[theme];
        const active = theme === current;
        return (
          <button
            key={theme}
            type="button"
            aria-pressed={active}
            data-theme-id={theme}
            onClick={() => apply(theme)}
            className={`glass-face hoverable rounded-lg-t p-4 text-left transition-colors ${active ? 'gl-on-rim' : ''}`}
          >
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-2.5">
                {/* swatch: ground · signal · ink */}
                <span className="flex overflow-hidden rounded-[7px] border border-[var(--border)]">
                  {meta.swatch.map((c, i) => (
                    <span key={i} className="h-6 w-5" style={{ background: c }} />
                  ))}
                </span>
                <span className="text-[13.5px] text-os-text">{meta.name}</span>
              </div>
              {active && <Check aria-hidden className="h-4 w-4 text-[var(--accent)]" strokeWidth={1.9} />}
            </div>
            <p className="mt-2 text-[12px] text-os-dim">{meta.blurb}</p>
          </button>
        );
      })}
    </div>
  );
}
