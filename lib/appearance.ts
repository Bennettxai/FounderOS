/**
 * Appearance model (SPEC S00 FR-1 to FR-7, §7; X3; X4).
 *
 * Three independent axes:
 *   ui      style:   classic (today's look) | glados (the new UI style). The
 *                    label users see is BRAND.name, never the id (§3.6).
 *   accent  palette: ice | lime | ember | mint. Used only under `glados`.
 *   mode    dark | light | system. Under `classic` the mode is derived from
 *           the Classic theme (White is light, every other theme dark).
 *
 * Storage: one cookie `glados-appearance` = `<ui>.<accent>.<mode>` (read by
 * the server so <html> carries the right attributes before any script), mirrored
 * in localStorage `glados-ui`, `glados-accent`, `glados-mode`. The Classic theme
 * keeps its own key `glados-theme` (lib/theme.ts), so switching style back and
 * forth never loses either side's choice.
 *
 * <html> attributes: data-ui, data-accent, data-mode-pref (the stored choice,
 * may be `system`), data-mode (always `dark` | `light`), data-theme (the
 * Classic theme, or under `glados` the one derived from accent and mode so
 * unconverted pages stay coherent, FR-3), data-ground (black | grey | white:
 * the ground that theme paints, which the glass light reads, see groundFor).
 *
 * This module is safe to import from server and client code. The server reader
 * is lib/appearance-server.ts (next/headers), the React hook lib/use-appearance.ts.
 */
import { DEFAULT_THEME, THEMES, THEME_STORAGE_KEY, isTheme, type Theme } from './theme';
import { isGReady } from './g-ready';
import { isLReady } from './l-ready';
import { BRAND, type Brand } from './brand';

export { G_READY, G_TREE_FILES, isGReady } from './g-ready';
export { L_READY, isLReady } from './l-ready';

/**
 * classic | glados | legacy. `legacy` is Bennett Legacy (FounderOS ui merge
 * SPEC §2.5): the FounderOS Monolith look, `--l-*` tokens, JetBrains Mono.
 */
export const UI_STYLES = ['classic', 'glados', 'legacy'] as const;
export type UiStyle = (typeof UI_STYLES)[number];

/** User-facing style names. Legacy's is fixed (T1); GladOS's is the product name (§3.6). */
export const STYLE_LABELS: Record<UiStyle, string> = { glados: BRAND.name, classic: 'Classic', legacy: 'Bennett Legacy' };

/**
 * Legacy skins (FO THEME_META). T2 default: Monolith only, which wears its
 * Daylight face in light mode. The cookie's middle field holds the skin under
 * `legacy`; Terminal, Clay, Midnight and Ember would join this list if T2 says so.
 */
export const LEGACY_SKINS = ['mono'] as const;
export type LegacySkin = (typeof LEGACY_SKINS)[number];
export const DEFAULT_LEGACY_SKIN: LegacySkin = 'mono';

/** The `data-theme` a Legacy page wears: Classic tokens remapped to the skin (SCAFFOLD §3.3). */
export const LEGACY_THEMES = ['monolith', 'daylight'] as const;
export type LegacyTheme = (typeof LEGACY_THEMES)[number];

/** Picker metadata for the two faces of the mono skin. Swatch fills are --l-swatch-* tokens. */
export const LEGACY_FACES: Record<LegacyTheme, { name: string; blurb: string; mode: Mode }> = {
  monolith: { name: 'Monolith', blurb: 'white on black, colour is status only', mode: 'dark' },
  daylight: { name: 'Daylight', blurb: 'brain blue on cool white', mode: 'light' },
};

export const ACCENT_IDS = ['ice', 'lime', 'ember', 'mint'] as const;
export type Accent = (typeof ACCENT_IDS)[number];

export const MODE_PREFS = ['dark', 'light', 'system'] as const;
export type ModePref = (typeof MODE_PREFS)[number];
export type Mode = 'dark' | 'light';

/**
 * The stored choice: what the cookie holds. `skin` is set only under `legacy`
 * (where the cookie's middle field is the skin, not the accent); the other
 * style's choice lives in localStorage (`glados-accent`, `glados-legacy-skin`).
 */
export type Appearance = { ui: UiStyle; accent: Accent; mode: ModePref; skin?: LegacySkin };

/** What <html> carries once resolved. */
export type AppliedAppearance = {
  ui: UiStyle;
  accent: Accent;
  modePref: ModePref;
  mode: Mode;
  theme: Theme | LegacyTheme;
};

/** Picker metadata (S00 §7). The swatch fill is each accent's hero-a token, owned by the token block. */
export const ACCENTS: Record<Accent, { name: string; blurb: string }> = {
  ice: { name: 'Ice', blurb: 'blue-white signal on the grey ground' },
  lime: { name: 'Lime', blurb: 'lime badge and green heroes' },
  ember: { name: 'Ember', blurb: 'red, orange and amber' },
  mint: { name: 'Mint', blurb: 'mint and teal' },
};

export const APPEARANCE_COOKIE = 'glados-appearance';
export const STORAGE_KEYS = { ui: 'glados-ui', accent: 'glados-accent', mode: 'glados-mode', skin: 'glados-legacy-skin' } as const;
/** The Creative Engine's old page-only mode (FR-10, Q-SYS-4): discarded, never promoted. */
export const LEGACY_PAPER_MODE_KEY = 'ae-mode';
export const COOKIE_MAX_AGE = 365 * 24 * 60 * 60;

/**
 * Defaults (FR-7, Q-SYS-5, applied at G-final 2026-10-09): a fresh install
 * starts on glados · ice · dark. An existing install (a stored Classic theme,
 * no appearance cookie yet) keeps Classic (FR-6, see the init script). Accent
 * Ice is Q-SYS-1. A stored choice always wins.
 */
export function defaultAppearanceFor(brand: Brand['id']): Appearance {
  // T1 (FounderOS ui merge §8): the client build (FounderOS brand) opens on Bennett Legacy · Monolith · Dark.
  return brand === 'founderos' ? { ui: 'legacy', accent: 'ice', mode: 'dark', skin: DEFAULT_LEGACY_SKIN } : { ui: 'glados', accent: 'ice', mode: 'dark' };
}
export const DEFAULT_APPEARANCE: Appearance = defaultAppearanceFor(BRAND.id);
const DEFAULT_ACCENT: Accent = 'ice';

export const isUiStyle = (v: unknown): v is UiStyle => typeof v === 'string' && (UI_STYLES as readonly string[]).includes(v);
export const isAccent = (v: unknown): v is Accent => typeof v === 'string' && (ACCENT_IDS as readonly string[]).includes(v);
export const isModePref = (v: unknown): v is ModePref => typeof v === 'string' && (MODE_PREFS as readonly string[]).includes(v);
export const isMode = (v: unknown): v is Mode => v === 'dark' || v === 'light';
export const isLegacySkin = (v: unknown): v is LegacySkin => typeof v === 'string' && (LEGACY_SKINS as readonly string[]).includes(v);
export const isLegacyTheme = (v: unknown): v is LegacyTheme => typeof v === 'string' && (LEGACY_THEMES as readonly string[]).includes(v);

/**
 * `<ui>.<accent-or-skin>.<mode>`; each field falls back to its default on its
 * own (edge 2). Under `legacy` the middle field is the skin (an unknown one
 * falls back to `mono`) and the accent is the default: the real GladOS accent
 * waits in localStorage `glados-accent` for the switch back (`withStoredChoices`).
 */
export function parseAppearanceCookie(raw: string | null | undefined): Appearance {
  const [ui, mid, mode] = (raw ?? '').split('.');
  const u = isUiStyle(ui) ? ui : DEFAULT_APPEARANCE.ui;
  const m = isModePref(mode) ? mode : DEFAULT_APPEARANCE.mode;
  if (u === 'legacy') return { ui: u, accent: DEFAULT_ACCENT, mode: m, skin: isLegacySkin(mid) ? mid : DEFAULT_LEGACY_SKIN };
  return { ui: u, accent: isAccent(mid) ? mid : DEFAULT_ACCENT, mode: m };
}

export function serializeAppearance(a: Appearance): string {
  return `${a.ui}.${a.ui === 'legacy' ? (a.skin ?? DEFAULT_LEGACY_SKIN) : a.accent}.${a.mode}`;
}

/**
 * The full choice on the client: the cookie's axes plus the other style's
 * choice from storage (the GladOS accent while on Legacy, the Legacy skin
 * while on GladOS or Classic). AC-5: switching styles never loses either side.
 */
export function withStoredChoices(a: Appearance, stored: { accent?: string | null; skin?: string | null }): Appearance & { skin: LegacySkin } {
  return {
    ...a,
    accent: a.ui === 'legacy' && isAccent(stored.accent) ? stored.accent : a.accent,
    skin: a.ui === 'legacy' ? (a.skin ?? DEFAULT_LEGACY_SKIN) : isLegacySkin(stored.skin) ? stored.skin : DEFAULT_LEGACY_SKIN,
  };
}

/** The choice after picking a style: the other axes are kept as they are. */
export function switchStyle(a: Appearance & { skin: LegacySkin }, ui: UiStyle): Appearance & { skin: LegacySkin } {
  return { ...a, ui };
}

/** The `data-theme` under Legacy: Monolith in dark, Daylight in light (T2). */
export function legacyTheme(_skin: LegacySkin, mode: Mode): LegacyTheme {
  return mode === 'light' ? 'daylight' : 'monolith';
}

/** The cookie string a client writes (not HttpOnly: the head script and Settings write it). */
export function appearanceCookie(a: Appearance): string {
  return `${APPEARANCE_COOKIE}=${serializeAppearance(a)}; Path=/; Max-Age=${COOKIE_MAX_AGE}; SameSite=Lax`;
}

/**
 * FR-3: the Classic theme an unconverted page wears under `glados`.
 * light → White; dark + ice → Grey Frost; dark + ember → Red; dark + lime or mint → Console.
 */
export function deriveClassicTheme(accent: Accent, mode: Mode): Theme {
  if (mode === 'light') return 'light';
  if (accent === 'ice') return 'grey-frost';
  if (accent === 'ember') return 'red';
  return 'console';
}

/**
 * The ground a page paints under the glass (FounderOS Glass brief §2): the
 * groove, drop and shade are tuned per ground, and on a white ground the drop
 * carries the shape while the groove becomes a hairline (`[data-ground='white']`
 * in app/globals.css). Classic and Legacy paint their theme's ground; GladOS
 * paints its own (--g-bg): grey in dark for every accent (Q-SYS-3), white in light.
 */
export const GROUNDS = ['black', 'grey', 'white'] as const;
export type Ground = (typeof GROUNDS)[number];
export const GROUND_BY_THEME: Record<Theme | LegacyTheme, Ground> = {
  console: 'black',
  red: 'black',
  frost: 'black',
  'grey-frost': 'grey',
  light: 'white',
  monolith: 'black',
  daylight: 'white',
};

/**
 * `ui` is optional: without it the theme decides. A forced `?mode=` does not
 * repaint a Classic theme, so under Classic and Legacy the mode only breaks
 * the tie for a theme this table does not know.
 */
export function groundFor(theme: string, mode: Mode, ui?: UiStyle): Ground {
  if (ui === 'glados') return mode === 'light' ? 'white' : 'grey';
  return (GROUND_BY_THEME as Record<string, Ground>)[theme] ?? (mode === 'light' ? 'white' : 'black');
}

/** FR-4: under `classic` the mode follows the theme. */
export function classicMode(theme: Theme): Mode {
  return theme === 'light' ? 'light' : 'dark';
}

/** FR-6: what a stored Classic theme becomes when the user picks the new style. */
const LEGACY_MAP: Record<Theme, { accent: Accent; mode: Mode }> = {
  console: { accent: 'lime', mode: 'dark' },
  red: { accent: 'ember', mode: 'dark' },
  light: { accent: 'lime', mode: 'light' },
  frost: { accent: 'ice', mode: 'dark' },
  'grey-frost': { accent: 'ice', mode: 'dark' },
};

export function migrateLegacy(theme: string | null | undefined): { accent: Accent; mode: ModePref } {
  return isTheme(theme) ? { ...LEGACY_MAP[theme] } : { accent: DEFAULT_APPEARANCE.accent, mode: DEFAULT_APPEARANCE.mode };
}

/** `system` resolves from the OS; dark and light are themselves. */
export function resolveMode(pref: ModePref, prefersLight: boolean): Mode {
  return pref === 'system' ? (prefersLight ? 'light' : 'dark') : pref;
}

/** `?mode=dark|light` forces the mode for one load (FR-2, FR-11). Anything else is ignored. */
export function modeOverride(search: string | null | undefined): Mode | null {
  const q = new URLSearchParams(search ?? '').get('mode');
  return isMode(q) ? q : null;
}

/**
 * Everything <html> carries for a stored choice. `classicTheme` is the stored
 * `glados-theme` (null on the server, which cannot see localStorage: the head
 * script corrects it before paint). `prefersLight` is null on the server, so
 * `system` renders dark there (edge 4).
 */
export function resolveAppearance(
  a: Appearance,
  env: { classicTheme?: string | null; prefersLight?: boolean | null; forced?: Mode | null } = {},
): AppliedAppearance {
  if (a.ui === 'glados') {
    const mode = env.forced ?? resolveMode(a.mode, env.prefersLight === true);
    return { ui: a.ui, accent: a.accent, modePref: a.mode, mode, theme: deriveClassicTheme(a.accent, mode) };
  }
  if (a.ui === 'legacy') {
    const mode = env.forced ?? resolveMode(a.mode, env.prefersLight === true);
    return { ui: a.ui, accent: a.accent, modePref: a.mode, mode, theme: legacyTheme(a.skin ?? DEFAULT_LEGACY_SKIN, mode) };
  }
  const theme = isTheme(env.classicTheme) ? env.classicTheme : DEFAULT_THEME;
  const own = classicMode(theme);
  return { ui: a.ui, accent: a.accent, modePref: own, mode: env.forced ?? own, theme };
}

/** The attributes as a props object for <html>. */
export function htmlAttributes(r: AppliedAppearance) {
  return {
    'data-ui': r.ui,
    'data-accent': r.accent,
    'data-mode-pref': r.modePref,
    'data-mode': r.mode,
    'data-theme': r.theme,
    'data-ground': groundFor(r.theme, r.mode, r.ui),
  } as const;
}

/** Read the applied appearance off an element's attributes (client side). */
export function readAppliedAppearance(el: { getAttribute(name: string): string | null }): AppliedAppearance {
  const ui = el.getAttribute('data-ui');
  const accent = el.getAttribute('data-accent');
  const modePref = el.getAttribute('data-mode-pref');
  const mode = el.getAttribute('data-mode');
  const theme = el.getAttribute('data-theme');
  return {
    ui: isUiStyle(ui) ? ui : DEFAULT_APPEARANCE.ui,
    accent: isAccent(accent) ? accent : DEFAULT_APPEARANCE.accent,
    modePref: isModePref(modePref) ? modePref : DEFAULT_APPEARANCE.mode,
    mode: isMode(mode) ? mode : 'dark',
    theme: isTheme(theme) || isLegacyTheme(theme) ? theme : DEFAULT_THEME,
  };
}

export type Tree = 'glados' | 'legacy' | 'classic';

/**
 * Which tree a route renders for this appearance (X5, FounderOS ui merge §2.5).
 * `glados` when the style is GladOS and the route is in G_READY; `legacy` when
 * the style is Bennett Legacy and the route is in L_READY; otherwise `classic`
 * (under Legacy that Classic tree wears the Legacy skin, O1).
 */
export function treeFor(
  a: { ui: UiStyle },
  pathname: string,
  ready: { g?: readonly string[]; l?: readonly string[] } = {},
): Tree {
  if (a.ui === 'glados' && isGReady(pathname, ready.g)) return 'glados';
  if (a.ui === 'legacy' && isLReady(pathname, ready.l)) return 'legacy';
  return 'classic';
}

/** Does this route render its GladOS tree? Kept for callers outside the page forks (the Lab). */
export function usesGTree(a: { ui: UiStyle }, pathname: string, ready?: readonly string[]): boolean {
  return treeFor(a, pathname, { g: ready }) === 'glados';
}

/**
 * Pre-paint head script (replaces THEME_INIT_SCRIPT). Generated from the
 * registries so it cannot drift from the TS model; tests run it against a fake
 * document and compare with `resolveAppearance` for every combination.
 *
 * 1. Cookie present: it wins; storage is mirrored from it. Malformed fields
 *    fall back one by one and the cookie is rewritten (edge 2).
 * 2. Cookie missing: storage wins once (edge 3); with no stored axes, the
 *    accent and mode come from the old single theme (FR-6) and ui stays
 *    classic for that existing install; with no old theme either it is a
 *    fresh install and takes the FR-7 default. The cookie is then written so the server agrees next request.
 * 3. `ae-mode` is deleted (FR-10).
 * 4. Attributes applied: `system` resolved with matchMedia, `?mode=` applied
 *    for this load only, data-theme derived under glados, data-ground from groundFor.
 * 5. A live `change` listener on the OS scheme re-applies when the current
 *    preference (read from <html>, so a later Settings change counts) is system.
 * Storage or cookie failures are swallowed: attributes still apply (edge 1).
 */
export const APPEARANCE_INIT_SCRIPT = `(function(){try{
var d=document.documentElement;
var U=${JSON.stringify(UI_STYLES)},A=${JSON.stringify(ACCENT_IDS)},M=${JSON.stringify(MODE_PREFS)},T=${JSON.stringify(THEMES)},SK=${JSON.stringify(LEGACY_SKINS)};
var D=${JSON.stringify(DEFAULT_APPEARANCE)},DA=${JSON.stringify(DEFAULT_ACCENT)},DS=${JSON.stringify(DEFAULT_LEGACY_SKIN)},DT=${JSON.stringify(DEFAULT_THEME)},L=${JSON.stringify(LEGACY_MAP)};
var K=${JSON.stringify(STORAGE_KEYS)},C=${JSON.stringify(APPEARANCE_COOKIE)},G=${JSON.stringify(GROUND_BY_THEME)};
function g(k){try{return localStorage.getItem(k)}catch(e){return null}}
function s(k,v){try{localStorage.setItem(k,v)}catch(e){}}
function ok(l,v){return l.indexOf(v)>=0}
var raw=null;try{var m=(document.cookie||'').match(new RegExp('(?:^|;\\\\s*)'+C+'=([^;]*)'));if(m)raw=decodeURIComponent(m[1]);}catch(e){}
var a;
if(raw!==null){var p=raw.split('.'),pu=ok(U,p[0])?p[0]:D.ui;a={ui:pu,accent:pu!=='legacy'&&ok(A,p[1])?p[1]:DA,mode:ok(M,p[2])?p[2]:D.mode,skin:pu==='legacy'&&ok(SK,p[1])?p[1]:DS};}
else{var lt=L[g(${JSON.stringify(THEME_STORAGE_KEY)})],lg=lt||D,su=g(K.ui),sa=g(K.accent),sm=g(K.mode),ss=g(K.skin);a={ui:ok(U,su)?su:lt?'classic':D.ui,accent:ok(A,sa)?sa:lg.accent,mode:ok(M,sm)?sm:lg.mode,skin:ok(SK,ss)?ss:DS};if(a.ui==='legacy')a.accent=DA;}
var ser=a.ui+'.'+(a.ui==='legacy'?a.skin:a.accent)+'.'+a.mode;
if(raw!==ser){try{document.cookie=C+'='+ser+'; Path=/; Max-Age=${COOKIE_MAX_AGE}; SameSite=Lax';}catch(e){}}
s(K.ui,a.ui);if(a.ui==='legacy')s(K.skin,a.skin);else s(K.accent,a.accent);s(K.mode,a.mode);
try{localStorage.removeItem(${JSON.stringify(LEGACY_PAPER_MODE_KEY)})}catch(e){}
var q=null;try{q=new URLSearchParams(location.search).get('mode');}catch(e){}
var f=(q==='dark'||q==='light')?q:null;
var mq=null;try{mq=window.matchMedia('(prefers-color-scheme: light)');}catch(e){}
var ct=g(${JSON.stringify(THEME_STORAGE_KEY)});if(!ok(T,ct))ct=DT;
function dv(ac,md){return md==='light'?'light':ac==='ice'?'grey-frost':ac==='ember'?'red':'console'}
function lv(md){return md==='light'?'daylight':'monolith'}
function gr(u,t,md){return u==='glados'?(md==='light'?'white':'grey'):(G[t]||(md==='light'?'white':'black'))}
var mode,pref,th;
if(a.ui==='glados'||a.ui==='legacy'){pref=a.mode;mode=f||(pref==='system'?(mq&&mq.matches?'light':'dark'):pref);th=a.ui==='legacy'?lv(mode):dv(a.accent,mode);}
else{th=ct;pref=ct==='light'?'light':'dark';mode=f||pref;}
d.setAttribute('data-ui',a.ui);d.setAttribute('data-accent',a.accent);d.setAttribute('data-mode-pref',pref);d.setAttribute('data-mode',mode);d.setAttribute('data-theme',th);d.setAttribute('data-ground',gr(a.ui,th,mode));
if(mq&&!f){var on=function(){var u=d.getAttribute('data-ui');if((u!=='glados'&&u!=='legacy')||d.getAttribute('data-mode-pref')!=='system')return;var md=mq.matches?'light':'dark',t=u==='legacy'?lv(md):dv(d.getAttribute('data-accent'),md);d.setAttribute('data-mode',md);d.setAttribute('data-theme',t);d.setAttribute('data-ground',gr(u,t,md));};if(mq.addEventListener)mq.addEventListener('change',on);else if(mq.addListener)mq.addListener(on);}
}catch(e){}})();`;

/* ── X3: one appearance observer ─────────────────────────────────────────── */

export type AppearanceListener = (a: AppliedAppearance) => void;

/** Just enough of the DOM for the observer; a fake satisfies it in tests. */
export type AppearanceEnv = {
  root: { getAttribute(name: string): string | null };
  MutationObserver: new (cb: () => void) => { observe(target: unknown, opts: MutationObserverInit): void; disconnect(): void };
  matchMedia?: (q: string) => {
    matches: boolean;
    addEventListener?(t: 'change', l: () => void): void;
    removeEventListener?(t: 'change', l: () => void): void;
  };
};

export const OBSERVED_ATTRIBUTES = ['data-ui', 'data-accent', 'data-mode', 'data-theme'] as const;

/**
 * Builds one shared observer: the first subscriber starts a single
 * MutationObserver on <html> (data-ui, data-accent, data-mode, data-theme) and
 * one System-scheme media listener; the last unsubscribe stops both. Every
 * callback gets the attributes as they are after the change.
 */
export function createAppearanceObserver(getEnv: () => AppearanceEnv | null) {
  const listeners = new Set<AppearanceListener>();
  let stop: (() => void) | null = null;

  const emit = (env: AppearanceEnv) => {
    const snapshot = readAppliedAppearance(env.root);
    for (const l of [...listeners]) l(snapshot);
  };

  const start = () => {
    const env = getEnv();
    if (!env) return;
    const fire = () => emit(env);
    const mo = new env.MutationObserver(fire);
    mo.observe(env.root, { attributes: true, attributeFilter: [...OBSERVED_ATTRIBUTES] });
    const mq = env.matchMedia?.('(prefers-color-scheme: light)');
    mq?.addEventListener?.('change', fire);
    stop = () => {
      mo.disconnect();
      mq?.removeEventListener?.('change', fire);
    };
  };

  return {
    subscribe(cb: AppearanceListener): () => void {
      listeners.add(cb);
      if (!stop) start();
      return () => {
        listeners.delete(cb);
        if (listeners.size === 0 && stop) {
          stop();
          stop = null;
        }
      };
    },
    /** How many subscribers are attached (tests). */
    size: () => listeners.size,
  };
}

const browserEnv = (): AppearanceEnv | null =>
  typeof document === 'undefined' || typeof MutationObserver === 'undefined'
    ? null
    : {
        root: document.documentElement,
        MutationObserver,
        matchMedia: typeof window !== 'undefined' && window.matchMedia ? (q) => window.matchMedia(q) : undefined,
      };

const shared = createAppearanceObserver(browserEnv);

/**
 * Subscribe to appearance changes on <html>. Returns the unsubscribe. Every
 * colour cache in JS (usePalette, canvases, minimaps, the globe) uses this one
 * observer (X3). On the server it is a no-op.
 */
export function onAppearanceChange(cb: AppearanceListener): () => void {
  return shared.subscribe(cb);
}

/** Current applied appearance on the client; the default on the server. */
export function currentAppearance(): AppliedAppearance {
  if (typeof document === 'undefined') return resolveAppearance(DEFAULT_APPEARANCE);
  return readAppliedAppearance(document.documentElement);
}

/**
 * Persist and apply a new choice on the client (Settings, lane 2a): cookie,
 * storage mirror, and the attributes. Colour and mode apply live; a style
 * change also needs `router.refresh()` from the caller so server trees re-render (FR-15).
 */
export function writeAppearance(next: Appearance): AppliedAppearance {
  const el = document.documentElement;
  try {
    document.cookie = appearanceCookie(next);
  } catch {}
  try {
    localStorage.setItem(STORAGE_KEYS.ui, next.ui);
    // Both style-specific choices are kept, so a switch back restores them (AC-5).
    localStorage.setItem(STORAGE_KEYS.accent, next.accent);
    if (next.skin) localStorage.setItem(STORAGE_KEYS.skin, next.skin);
    localStorage.setItem(STORAGE_KEYS.mode, next.mode);
  } catch {}
  let classicTheme: string | null = null;
  try {
    classicTheme = localStorage.getItem(THEME_STORAGE_KEY);
  } catch {}
  const prefersLight = typeof window !== 'undefined' && !!window.matchMedia?.('(prefers-color-scheme: light)').matches;
  const r = resolveAppearance(next, { classicTheme, prefersLight });
  for (const [k, v] of Object.entries(htmlAttributes(r))) el.setAttribute(k, v);
  return r;
}
