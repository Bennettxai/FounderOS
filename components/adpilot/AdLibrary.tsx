'use client';

import { useCallback, useEffect, useMemo, useRef, useState, type CSSProperties, type KeyboardEvent } from 'react';
import { useRouter, useSearchParams } from 'next/navigation';
import {
  Bookmark,
  Check,
  ChevronDown,
  ChevronRight,
  Library as LibraryBig,
  Copy,
  ExternalLink,
  Loader2,
  Plus,
  RefreshCw,
  Search,
  Send,
  TrendingDown,
  TrendingUp,
  Trophy,
  X,
  XCircle,
  Zap,
} from 'lucide-react';
import type { WallAd } from '@/lib/foreplay/wall';
import type { WatchEntry } from '@/lib/foreplay/watchlist';
import type { SavedAd } from '@/lib/foreplay/saved';
import type { Signal } from '@/lib/foreplay/signals';
import { forYou } from '@/lib/foreplay/personal';
import { dedupeSignals, parseBrandDomain, relativeAgo, rovingIndex } from '@/lib/adpilot';
import { SetupLink } from '@/components/SetupLink';
import { setupHrefFor } from '@/lib/integrations-catalog';
import { useToast } from '@/components/ui/Toast';
import { useDialog } from '@/components/ui/use-dialog';
import { patchUrlParams, readLocal, writeLocal } from '@/components/adpilot/url';
import { findLinkedAd } from '@/lib/adpilot-links';
import { GAsk, GBar, GButton, GCard, GEmpty, GHeroCard, GTag, GTickStrip } from '@/components/g';
import { useGStyle, useGc } from '@/components/system-g/style-context';
import { Layer } from '@/components/g/Overlays';

/** Set up link for the current style: the kit button under GladOS (GP4), the Classic SetupLink otherwise. */
function Setup({ href, hint }: { href: string; hint?: string }) {
  const g = useGStyle();
  return g ? <GButton size="sm" href={href} title={hint}>Set up</GButton> : <SetupLink href={href} hint={hint} />;
}

/**
 * The Ad library: ONE bounded module (viewport-height, internal scroll; the
 * page ends cleanly after it). Tabs: For you (personalized, honest, labeled
 * ranking) · Search (concept mining) · Saved (the swipe file) · Watchlist ·
 * Activity (the wire, in plain English). Ask Adscout is the prompt bar
 * pinned at the top. Every surface wears the page's theme-keyed glass
 * (.ap-panel and the .ap-* classes; no colour literal lives here).
 *
 * Deep links (SPEC G2): `?ad=<id>` opens that ad's dossier, `?brand=<id>`
 * opens the Watchlist on that brand. Anything that spends Foreplay credits
 * (Search, Refresh) asks once more before it fires (SPEC §3.1.7).
 */

type Props = {
  initialWall: WallAd[];
  /** The ad an ?ad= link names when it is not on the wall (server lookup, G2). */
  linkedAd?: WallAd | null;
  watchlist: WatchEntry[];
  saved: SavedAd[];
  /** The newest signals the page sends (capped). */
  signals: Signal[];
  /** How many signals exist in the store in total. */
  signalsTotal: number;
  credits: { remaining: number; total: number } | null;
  foreplayConfigured: boolean;
  lastSyncAt: string | null;
  /** Server render time, so relative times match on hydration. */
  now: number;
};

type Tab = 'foryou' | 'search' | 'saved' | 'watchlist' | 'activity';
const TAB_KEYS: Tab[] = ['foryou', 'search', 'saved', 'watchlist', 'activity'];
const isTab = (v: string | null): v is Tab => !!v && (TAB_KEYS as string[]).includes(v);

const LS_OPEN = 'adpilot-library-open';
const LS_TAB = 'adpilot-library-tab';
/** Undo window before a watchlist removal is sent (S01 F24, Q-AP-4). */
const UNDO_MS = 5000;
/** How long an armed spend button waits for its confirm click. */
const ARM_MS = 6000;

const nf = new Intl.NumberFormat('en-US');
const FOREPLAY_SETUP = setupHrefFor('foreplay');
const NOT_CONFIGURED = 'Foreplay is not configured';

/** A spend button that needs a second click within ARM_MS (SPEC §3.1.7). */
function useArm() {
  const [armed, setArmed] = useState(false);
  const timer = useRef<number | null>(null);
  const disarm = useCallback(() => {
    if (timer.current) window.clearTimeout(timer.current);
    timer.current = null;
    setArmed(false);
  }, []);
  const arm = useCallback(() => {
    if (timer.current) window.clearTimeout(timer.current);
    setArmed(true);
    timer.current = window.setTimeout(() => setArmed(false), ARM_MS);
  }, []);
  useEffect(() => () => {
    if (timer.current) window.clearTimeout(timer.current);
  }, []);
  return { armed, arm, disarm };
}

async function readJson(res: Response): Promise<Record<string, unknown>> {
  try {
    return (await res.json()) as Record<string, unknown>;
  } catch {
    return {};
  }
}

export function AdLibrary({
  initialWall,
  linkedAd = null,
  watchlist: initialWatch,
  saved: initialSaved,
  signals,
  signalsTotal,
  credits,
  foreplayConfigured,
  lastSyncAt,
  now,
}: Props) {
  const router = useRouter();
  const params = useSearchParams();
  const { toast } = useToast();
  const [tab, setTabState] = useState<Tab>('foryou');
  const [watchlist, setWatchlist] = useState(initialWatch);
  const [saved, setSaved] = useState(initialSaved);
  const [busy, setBusy] = useState<string | null>(null);
  const [selected, setSelected] = useState<WallAd | null>(null);
  const [mineResults, setMineResults] = useState<WallAd[] | null>(null);
  const [libOpen, setLibOpenState] = useState(false);
  const [ringBrand, setRingBrand] = useState<string | null>(null);
  const [syncSetup, setSyncSetup] = useState(false);
  const sync = useArm();
  const g = useGStyle();
  const gc = useGc();

  const savedIds = useMemo(() => new Set(saved.map((s) => s.ad.id)), [saved]);
  const savedAds = useMemo(() => saved.map((s) => s.ad), [saved]);
  const watchedIds = useMemo(() => new Set(watchlist.map((w) => w.id)), [watchlist]);

  const setLibOpen = useCallback((open: boolean) => {
    setLibOpenState(open);
    writeLocal(LS_OPEN, open ? '1' : null);
  }, []);
  const setTab = useCallback((t: Tab) => {
    setTabState(t);
    writeLocal(LS_TAB, t);
    if (t !== 'watchlist') patchUrlParams({ brand: null });
  }, []);

  // Per-viewer conveniences: the library's open state and tab survive reloads.
  useEffect(() => {
    if (readLocal(LS_OPEN) === '1') setLibOpenState(true);
    // A ?brand= deep link picks the tab itself; the stored tab must not win.
    if (new URLSearchParams(window.location.search).has('brand')) return;
    const t = readLocal(LS_TAB);
    if (isTab(t)) setTabState(t);
  }, []);

  // For-you: watchlist store + saved snapshots + last mine results, scored
  // against niche keywords, longevity, and the taste of what the operator saved.
  const feed = useMemo(() => {
    const pool = new Map<string, WallAd>();
    for (const ad of initialWall) pool.set(ad.id, ad);
    for (const ad of mineResults ?? []) pool.set(ad.id, ad);
    return forYou([...pool.values()], savedAds).slice(0, 30);
  }, [initialWall, mineResults, savedAds]);

  const openAd = useCallback((ad: WallAd) => {
    setSelected(ad);
    patchUrlParams({ ad: ad.id });
  }, []);
  const closeAd = useCallback(() => {
    setSelected(null);
    patchUrlParams({ ad: null });
  }, []);

  /* ---------------------------- watchlist writes --------------------------- */

  const pending = useRef(new Map<string, { timer: number; entry: WatchEntry; index: number }>());

  const sendRemove = useCallback(async (brandId: string, keepalive = false) => {
    try {
      const res = await fetch('/api/adscout/watchlist', {
        method: 'DELETE',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ brandId }),
        keepalive,
      });
      const body = await readJson(res);
      if (keepalive) return;
      if (!res.ok || !body.ok) throw new Error(String(body.error ?? `HTTP ${res.status}`));
      const list = (body.watchlist as WatchEntry[]) ?? [];
      setWatchlist(list.filter((w) => !pending.current.has(w.id)));
    } catch (err) {
      if (keepalive) return;
      toast({ text: `Could not stop watching: ${err instanceof Error ? err.message : 'request failed'}`, tone: 'err' });
    }
  }, [toast]);

  const removeWatch = useCallback(
    (entry: WatchEntry) => {
      if (pending.current.has(entry.id)) return;
      const index = watchlist.findIndex((w) => w.id === entry.id);
      setWatchlist((list) => list.filter((w) => w.id !== entry.id));
      const timer = window.setTimeout(() => {
        pending.current.delete(entry.id);
        void sendRemove(entry.id);
      }, UNDO_MS);
      pending.current.set(entry.id, { timer, entry, index });
      toast({
        text: `Stopped watching ${entry.name}`,
        tone: 'info',
        ms: UNDO_MS,
        action: {
          label: 'Undo',
          onClick: () => {
            const p = pending.current.get(entry.id);
            if (!p) return;
            window.clearTimeout(p.timer);
            pending.current.delete(entry.id);
            setWatchlist((list) => {
              if (list.some((w) => w.id === entry.id)) return list;
              const next = [...list];
              next.splice(Math.min(p.index < 0 ? next.length : p.index, next.length), 0, p.entry);
              return next;
            });
          },
        },
      });
    },
    [watchlist, sendRemove, toast],
  );

  // Leaving the page inside the undo window still sends what was asked for.
  useEffect(() => {
    const map = pending.current;
    const flush = () => {
      map.forEach((p, id) => {
        window.clearTimeout(p.timer);
        void sendRemove(id, true);
      });
      map.clear();
    };
    window.addEventListener('pagehide', flush);
    return () => {
      window.removeEventListener('pagehide', flush);
      flush();
    };
  }, [sendRemove]);

  const watchBrand = useCallback(
    async (brand: { brandId: string; name: string; avatar: string | null }) => {
      try {
        const res = await fetch('/api/adscout/watchlist', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(brand),
        });
        const body = await readJson(res);
        if (!res.ok || !body.ok) throw new Error(String(body.error ?? `HTTP ${res.status}`));
        setWatchlist(body.watchlist as WatchEntry[]);
        toast({ text: `Watching ${brand.name}. Refresh data to pull its ads.`, tone: 'ok' });
      } catch (err) {
        toast({ text: `Could not track ${brand.name}: ${err instanceof Error ? err.message : 'request failed'}`, tone: 'err' });
      }
    },
    [toast],
  );

  const toggleSave = useCallback(
    async (ad: WallAd) => {
      const isSaved = savedIds.has(ad.id);
      try {
        const res = await fetch('/api/adscout/saved', {
          method: isSaved ? 'DELETE' : 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(isSaved ? { adId: ad.id } : ad),
        });
        const body = await readJson(res);
        if (!res.ok || !body.ok) throw new Error(String(body.error ?? `HTTP ${res.status}`));
        setSaved(body.saved as SavedAd[]);
      } catch (err) {
        toast({ text: `Could not ${isSaved ? 'unsave' : 'save'} the ad: ${err instanceof Error ? err.message : 'request failed'}`, tone: 'err' });
      }
    },
    [savedIds, toast],
  );

  /* ------------------------------- deep links ------------------------------ */

  const handled = useRef<{ ad: string | null; brand: string | null }>({ ad: null, brand: null });
  const adParam = params.get('ad');
  const brandParam = params.get('brand');

  useEffect(() => {
    if (!adParam || handled.current.ad === adParam) {
      if (!adParam) handled.current.ad = null;
      return;
    }
    handled.current.ad = adParam;
    const ad = findLinkedAd(adParam, [initialWall, savedAds, mineResults, linkedAd ? [linkedAd] : null]);
    setLibOpenState(true);
    if (ad) {
      setSelected(ad);
    } else {
      setTabState('foryou');
      patchUrlParams({ ad: null });
      toast({ text: 'That ad is no longer in the library.', tone: 'info' });
    }
  }, [adParam, initialWall, savedAds, mineResults, linkedAd, toast]);

  useEffect(() => {
    if (!brandParam || handled.current.brand === brandParam) {
      if (!brandParam) handled.current.brand = null;
      return;
    }
    handled.current.brand = brandParam;
    setLibOpenState(true);
    if (watchlist.some((w) => w.id === brandParam)) {
      setTabState('watchlist');
      setRingBrand(brandParam);
      window.setTimeout(() => setRingBrand(null), 1200);
      return;
    }
    patchUrlParams({ brand: null });
    const known = initialWall.find((a) => a.brandId === brandParam);
    toast({
      text: 'That brand is not on your watchlist.',
      tone: 'info',
      ms: known ? 8000 : undefined,
      action: known
        ? { label: 'Track brand', onClick: () => void watchBrand({ brandId: brandParam, name: known.brand, avatar: known.thumbnail }) }
        : undefined,
    });
    // Only a new ?brand= value re-runs this; the watchlist changing under it does not.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [brandParam]);

  /* --------------------------------- refresh -------------------------------- */

  const runSync = async () => {
    if (!sync.armed) {
      sync.arm();
      return;
    }
    sync.disarm();
    setBusy('sync');
    try {
      const res = await fetch('/api/adscout/sync', { method: 'POST' });
      const body = await readJson(res);
      if (res.status === 503) {
        setSyncSetup(true);
        throw new Error(NOT_CONFIGURED);
      }
      if (!res.ok || !body.ok) throw new Error(String(body.error ?? `HTTP ${res.status}`));
      const newSignals = Array.isArray(body.signals) ? body.signals.length : 0;
      toast({ text: `Refreshed: ${body.brands ?? 0} brands · ${newSignals} new signals`, tone: 'ok' });
      // Re-reads the server data in place: the open tab, scroll and selection stay.
      router.refresh();
    } catch (err) {
      toast({ text: `Refresh failed: ${err instanceof Error ? err.message : 'request failed'}`, tone: 'err' });
    } finally {
      setBusy(null);
    }
  };

  /* ---------------------------------- tabs ---------------------------------- */

  const TABS: { key: Tab; label: string; count?: number }[] = [
    { key: 'foryou', label: 'For you' },
    { key: 'search', label: 'Search' },
    { key: 'saved', label: 'Saved', count: saved.length },
    { key: 'watchlist', label: 'Watchlist', count: watchlist.length },
    { key: 'activity', label: 'Activity', count: signalsTotal },
  ];
  const tabRefs = useRef(new Map<Tab, HTMLButtonElement | null>());
  const onTabKey = (e: KeyboardEvent<HTMLDivElement>) => {
    const i = TAB_KEYS.indexOf(tab);
    const next = rovingIndex(e.key, i, TAB_KEYS.length);
    if (next === null || e.key === 'ArrowUp' || e.key === 'ArrowDown') return;
    e.preventDefault();
    setTab(TAB_KEYS[next]);
    tabRefs.current.get(TAB_KEYS[next])?.focus();
  };

  const syncDisabledReason = !foreplayConfigured ? NOT_CONFIGURED : busy === 'sync' ? 'Refreshing now' : null;

  return (
    <div className="ap-lib">
      {/* Ask Adscout: frosted glass with the orbiting glow */}
      <AskWidget
        wall={initialWall}
        onOpen={openAd}
        onOpenWatchlist={() => {
          setLibOpen(true);
          setTab('watchlist');
        }}
      />

      {/* The band: one loud element, two quiet ones. Under GladOS the slab is the
          page's one hero and Foreplay is one card (F32, F33). */}
      {g ? (
        <BandG
          libOpen={libOpen}
          onToggle={() => setLibOpen(!libOpen)}
          wallCount={initialWall.length}
          longevity={initialWall.slice(0, 60).map((a) => ({ days: a.daysRunning, title: `${a.brand}: ${a.daysRunning} days${a.live ? ', live' : ''}` }))}
          savedCount={saved.length}
          credits={credits}
          configured={foreplayConfigured}
          setupNeeded={!foreplayConfigured || syncSetup}
          lastSyncAt={lastSyncAt}
          now={now}
          onSync={runSync}
          onSyncBlur={sync.disarm}
          armed={sync.armed}
          syncing={busy === 'sync'}
          syncDisabledReason={syncDisabledReason}
        />
      ) : (
      <div className="ap-band mt-6">
        <button
          type="button"
          onClick={() => setLibOpen(!libOpen)}
          aria-expanded={libOpen}
          aria-controls="ap-library"
          className="ap-bright ap-lift ap-band-slab flex items-center justify-between p-5 text-left"
        >
          <span>
            <span className="ap-ink flex items-center gap-2 text-[16px] font-medium">
              <LibraryBig aria-hidden className="h-5 w-5" strokeWidth={1.7} />
              {libOpen ? 'Close ad library' : 'Open ad library'}
            </span>
            <span className="ap-ink-2 mt-1 block text-[12.5px]">
              {nf.format(initialWall.length)} tracked ads · {saved.length} saved · search by concept, save winners
            </span>
          </span>
          <ChevronDown aria-hidden className={`ap-ink-2 h-5 w-5 shrink-0 transition-transform ${libOpen ? 'rotate-180' : ''}`} strokeWidth={1.9} />
        </button>

        <CreditsPanel credits={credits} configured={foreplayConfigured} lastSyncAt={lastSyncAt} now={now} />

        <div className="ap-panel ap-band-sync flex flex-col justify-center p-5">
          <button
            type="button"
            onClick={runSync}
            onBlur={sync.disarm}
            disabled={syncDisabledReason !== null}
            title={syncDisabledReason ?? (sync.armed ? 'Click again to spend Foreplay credits on a refresh' : 'Pull new ads and signals for every watched brand')}
            className="c-btn c-btn-primary flex items-center justify-center gap-2 py-2.5 text-[13px]"
          >
            {busy === 'sync' ? <Loader2 aria-hidden className="h-4 w-4 animate-spin" strokeWidth={1.7} /> : <RefreshCw aria-hidden className="h-4 w-4" strokeWidth={1.7} />}
            {sync.armed ? 'Confirm refresh' : 'Refresh data'}
          </button>
          <div className="mt-2 text-center text-[12px] text-os-dim" aria-live="polite">
            {!foreplayConfigured || syncSetup ? (
              <span className="inline-flex flex-wrap items-center justify-center gap-1.5">
                Not configured <Setup href={FOREPLAY_SETUP} hint="Add a Foreplay API key" />
              </span>
            ) : sync.armed ? (
              'Uses Foreplay credits (one call per watched brand)'
            ) : (
              <>
                last synced {relativeAgo(lastSyncAt, now)} · uses Foreplay credits
              </>
            )}
          </div>
        </div>
      </div>

      )}

      {libOpen && (
        <div id="ap-library" className={gc('ap-g-library', 'ap-panel mt-4 flex h-[80vh] flex-col overflow-hidden')}>
          {/* Tab rail */}
          <div role="tablist" aria-label="Ad library" onKeyDown={onTabKey} className={gc('g-seg ap-g-tabs', 'ap-hair flex gap-1 overflow-x-auto border-b px-3')}>
            {TABS.map((t) => (
              <button
                key={t.key}
                ref={(el) => {
                  tabRefs.current.set(t.key, el);
                }}
                id={`ap-tab-${t.key}`}
                type="button"
                role="tab"
                aria-selected={tab === t.key}
                aria-controls="ap-tabpanel"
                tabIndex={tab === t.key ? 0 : -1}
                onClick={() => setTab(t.key)}
                className={gc(`g-seg-opt${tab === t.key ? ' is-on' : ''}`, `flex shrink-0 items-center gap-1.5 border-b-2 px-2.5 py-2.5 text-[12.5px] transition-colors ${
                  tab === t.key ? 'border-[var(--accent)] text-os-text' : 'border-transparent text-os-dim hover:text-os-muted'
                }`)}
              >
                {t.label}
                {t.count !== undefined && t.count > 0 && <span className="text-[10.5px] tabular-nums text-os-dim">{nf.format(t.count)}</span>}
              </button>
            ))}
          </div>

          {/* Tab content: internal scroll; the page never sprawls */}
          <div id="ap-tabpanel" role="tabpanel" aria-labelledby={`ap-tab-${tab}`} className="min-h-0 flex-1 overflow-y-auto p-4">
            {tab === 'foryou' && (
              <>
                <CardGrid ads={feed.map((f) => f.ad)} savedIds={savedIds} onOpen={openAd} onSave={toggleSave} />
                <p className="mt-3 text-center text-[11px] text-os-dim">
                  ranked by fit to your niche + your saves + proven longevity · saves teach this feed
                </p>
              </>
            )}
            {tab === 'search' && (
              <SearchTab
                savedIds={savedIds}
                onOpen={openAd}
                onSave={toggleSave}
                onResults={setMineResults}
                configured={foreplayConfigured}
              />
            )}
            {tab === 'saved' &&
              (saved.length === 0 ? (
                <Empty line1="Nothing saved yet." line2="The bookmark on any ad saves it here, and teaches the For-you feed." />
              ) : (
                <CardGrid ads={savedAds} savedIds={savedIds} onOpen={openAd} onSave={toggleSave} />
              ))}
            {tab === 'watchlist' && (
              <WatchlistTab
                watchlist={watchlist}
                wall={initialWall}
                setWatchlist={setWatchlist}
                onRemove={removeWatch}
                ringBrand={ringBrand}
                configured={foreplayConfigured}
              />
            )}
            {tab === 'activity' && (
              <ActivityTab signals={signals} total={signalsTotal} wall={initialWall} onOpen={openAd} />
            )}
          </div>
        </div>
      )}

      {selected && (
        <Dossier
          ad={selected}
          isSaved={savedIds.has(selected.id)}
          isWatched={!!selected.brandId && watchedIds.has(selected.brandId)}
          onClose={closeAd}
          onSave={toggleSave}
          onWatch={(ad) => ad.brandId && watchBrand({ brandId: ad.brandId, name: ad.brand, avatar: ad.thumbnail })}
        />
      )}
    </div>
  );
}

/* ------------------------------- credits panel ----------------------------- */

const METER = 20;

function CreditsPanel({
  credits,
  configured,
  lastSyncAt,
  now,
}: {
  credits: { remaining: number; total: number } | null;
  configured: boolean;
  lastSyncAt: string | null;
  now: number;
}) {
  const share = credits && credits.total > 0 ? credits.remaining / credits.total : null;
  const filled = share === null ? 0 : Math.round(share * METER);
  const low = share !== null && share < 0.1;
  return (
    <div className="ap-panel ap-band-credits p-5">
      <div className="flex items-baseline justify-between gap-2">
        <span className="text-[12.5px] text-os-dim">API credits</span>
        {configured && <span className="text-[11px] text-os-dim">as of {relativeAgo(lastSyncAt, now)}</span>}
      </div>
      <div className="mt-1 text-[25px] font-light tabular-nums text-os-text">
        {!configured ? (
          <span className="flex flex-wrap items-center gap-2 text-[15px] text-os-muted">
            Not configured <Setup href={FOREPLAY_SETUP} hint="Add a Foreplay API key" />
          </span>
        ) : credits ? (
          <>
            {nf.format(credits.remaining)} <span className="text-[14px] text-os-dim">/ {nf.format(credits.total)}</span>
          </>
        ) : (
          <span title="No usage reported yet: credits are read on each refresh">—</span>
        )}
      </div>
      <div
        className="mt-2.5 flex gap-[3px]"
        role="img"
        aria-label={share === null ? 'Credits unknown' : `${Math.round(share * 100)}% of credits remaining${low ? ', running low' : ''}`}
        title={low ? 'Under 10% of credits remain' : undefined}
      >
        {Array.from({ length: METER }, (_, i) => (
          <span
            key={i}
            data-warn={low && i < filled ? '' : undefined}
            className={`h-[10px] flex-1 rounded-[2px] ${i < filled ? 'ap-fill' : 'ap-track'}`}
          />
        ))}
      </div>
    </div>
  );
}

/* -------------------------------- Ask widget ------------------------------- */

/** Keyword-overlap match: the ads the answer is most plausibly about. */
function relatedAds(text: string, wall: WallAd[]): WallAd[] {
  const words = new Set(
    text
      .toLowerCase()
      .split(/[^a-z0-9$]+/)
      .filter((w) => w.length > 3),
  );
  return wall
    .map((ad) => {
      const hay = `${ad.hook ?? ''} ${ad.brand}`.toLowerCase().split(/[^a-z0-9$]+/);
      let score = 0;
      for (const w of hay) if (words.has(w)) score += 1;
      return { ad, score: score + (ad.live ? 0.5 : 0) + Math.min(ad.daysRunning / 200, 0.5) };
    })
    .filter((r) => r.score >= 1.5)
    .sort((a, b) => b.score - a.score)
    .slice(0, 3)
    .map((r) => r.ad);
}

type AskState =
  | { phase: 'idle' }
  | { phase: 'running' }
  | { phase: 'done'; text: string }
  | { phase: 'error'; text: string; status: number };

const ASK_MIN = 3;

function AskWidget({
  wall,
  onOpen,
  onOpenWatchlist,
}: {
  wall: WallAd[];
  onOpen: (ad: WallAd) => void;
  onOpenWatchlist: () => void;
}) {
  const [question, setQuestion] = useState('');
  const [focused, setFocused] = useState(false);
  const [state, setState] = useState<AskState>({ phase: 'idle' });
  const examples = useMemo(
    () => (state.phase === 'done' ? relatedAds(`${question} ${state.text}`, wall) : []),
    [state, question, wall],
  );
  const tooShort = question.trim().length < ASK_MIN;

  const ask = async () => {
    const q = question.trim();
    if (q.length < ASK_MIN || state.phase === 'running') return;
    setState({ phase: 'running' });
    try {
      const res = await fetch('/api/adscout/ask', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ question: q }),
      });
      const body = await readJson(res);
      if (!res.ok || !body.ok) {
        setState({ phase: 'error', text: String(body.error ?? `HTTP ${res.status}`), status: res.status });
        return;
      }
      setState({ phase: 'done', text: String(body.answer ?? '') });
    } catch (err) {
      setState({ phase: 'error', text: err instanceof Error ? err.message : 'ask failed', status: 0 });
    }
  };

  const spin = focused || state.phase === 'running';
  const g = useGStyle();

  if (g) {
    // F31: the kit's ask composer (orbit only while focused or asking).
    return (
      <div className="ap-g-ask">
        <GAsk
          value={question}
          onValueChange={setQuestion}
          onSubmit={() => void ask()}
          busy={state.phase === 'running'}
          minChars={ASK_MIN}
          label="Ask Adscout"
          placeholder="Ask Adscout what's working in your niche right now"
        />
        <div aria-live="polite">
          {state.phase === 'running' && <p className="soc-g-cap ap-g-ask-note">Checking the library: hooks, longevity, drivers, signals.</p>}
          {state.phase === 'error' && (
            <div className="ap-g-ask-note ap-g-inline">
              <span className="soc-g-cap">{state.status === 503 ? 'Analyst unavailable' : state.text}</span>
              {state.status === 503 && <Setup href="/setup" hint="The analyst runs on the local Claude CLI" />}
              {state.status === 409 && <GButton size="sm" onClick={onOpenWatchlist}>Open watchlist</GButton>}
            </div>
          )}
          {state.phase === 'done' && (
            <GCard p={20} className="ap-g-answer">
              <div className="ap-g-inline">
                <p className="ap-g-answer-text">{state.text}</p>
                <GButton variant="icon" size="sm" aria-label="Dismiss answer" title="Dismiss answer" onClick={() => setState({ phase: 'idle' })}><X aria-hidden size={13} strokeWidth={1.75} /></GButton>
              </div>
              {examples.length > 0 && (
                <div className="ap-g-examples">
                  {examples.map((ad) => (
                    <GCard key={ad.id} p={16} onClick={() => onOpen(ad)} className="ap-g-example" aria-label={`Open ${ad.brand} dossier`}>
                      <span className="soc-g-sub">{ad.brand}</span>
                      <span className="soc-g-cap">{ad.daysRunning}d {ad.live ? 'live' : 'off'}</span>
                    </GCard>
                  ))}
                </div>
              )}
            </GCard>
          )}
        </div>
      </div>
    );
  }

  return (
    <div className="ap-ask relative mx-auto max-w-3xl overflow-hidden rounded-[20px]">
      {/* the light circling behind the glass: it moves only while you ask */}
      <div className="ap-orbit" data-spin={spin ? '1' : undefined} aria-hidden />
      <div className="ap-ask-glass absolute inset-0 backdrop-blur-2xl" aria-hidden />
      <div className="relative px-6 py-5">
        <div className="flex items-center gap-3">
          <label htmlFor="ap-ask" className="shrink-0 text-[14px] font-medium text-[var(--accent-2)]">
            Ask Adscout
          </label>
          <input
            id="ap-ask"
            value={question}
            onChange={(e) => setQuestion(e.target.value)}
            onFocus={() => setFocused(true)}
            onBlur={() => setFocused(false)}
            onKeyDown={(e) => {
              if (e.key === 'Enter') void ask();
              if (e.key === 'Escape' && (state.phase === 'done' || state.phase === 'error')) {
                e.preventDefault();
                setState({ phase: 'idle' });
              }
            }}
            placeholder="what's working in my niche right now?"
            className="min-w-0 flex-1 bg-transparent text-[14.5px] text-os-text outline-none placeholder:text-os-dim"
          />
          <button
            type="button"
            onClick={() => void ask()}
            disabled={state.phase === 'running' || tooShort}
            aria-label="Ask Adscout"
            title={state.phase === 'running' ? 'Asking now' : tooShort ? `Type at least ${ASK_MIN} characters` : 'Ask (Enter)'}
            className="c-btn px-3 py-2"
          >
            {state.phase === 'running' ? <Loader2 aria-hidden className="h-4 w-4 animate-spin" strokeWidth={1.7} /> : <Send aria-hidden className="h-4 w-4" strokeWidth={1.7} />}
          </button>
        </div>

        {/* the widget grows downward for the answer */}
        <div aria-live="polite">
          {state.phase === 'running' && (
            <p className="mt-3 text-[13px] text-os-muted">Checking the library: hooks, longevity, drivers, signals…</p>
          )}
          {state.phase === 'error' && (
            <div className="mt-3 flex flex-wrap items-center gap-2 text-[13px]">
              <span className="ap-err">{state.status === 503 ? 'Analyst unavailable' : state.text}</span>
              {state.status === 503 && <Setup href="/setup" hint="The analyst runs on the local Claude CLI" />}
              {state.status === 409 && (
                <button type="button" onClick={onOpenWatchlist} className="c-btn px-2.5 py-1 text-[12px]">
                  Open watchlist
                </button>
              )}
            </div>
          )}
          {state.phase === 'done' && (
            <div className="ap-ask-rule mt-4 border-t pt-4">
              <div className="flex items-start gap-3">
                <p className="flex-1 whitespace-pre-wrap text-[13.5px] leading-relaxed text-os-text">{state.text}</p>
                <button
                  type="button"
                  onClick={() => setState({ phase: 'idle' })}
                  aria-label="Dismiss answer"
                  title="Dismiss answer (Esc in the field)"
                  className="shrink-0 text-os-dim hover:text-os-text"
                >
                  <X aria-hidden className="h-4 w-4" strokeWidth={1.7} />
                </button>
              </div>
              {examples.length > 0 && (
                <div className="mt-4">
                  <div className="text-[12px] text-os-dim">From the library</div>
                  <div className="mt-2 grid grid-cols-1 gap-2 sm:grid-cols-3">
                    {examples.map((ad) => (
                      <button
                        key={ad.id}
                        type="button"
                        onClick={() => onOpen(ad)}
                        className="ap-ask-tile flex items-center gap-2.5 rounded-[10px] p-2 text-left"
                      >
                        {ad.thumbnail || ad.image ? (
                          // eslint-disable-next-line @next/next/no-img-element
                          <img src={ad.thumbnail ?? ad.image ?? ''} alt="" className="h-11 w-14 shrink-0 rounded-[7px] object-cover" />
                        ) : (
                          <span className="ap-tile flex h-11 w-14 shrink-0 items-center justify-center rounded-[7px] text-[10px] text-os-dim">
                            {ad.format ?? 'ad'}
                          </span>
                        )}
                        <span className="min-w-0">
                          <span className="block truncate text-[12px] text-os-text">{ad.brand}</span>
                          <span className="block text-[11px] tabular-nums text-os-dim">
                            {ad.daysRunning}d {ad.live ? 'live' : 'off'}
                          </span>
                        </span>
                      </button>
                    ))}
                  </div>
                </div>
              )}
            </div>
          )}
        </div>
      </div>
    </div>
  );
}

/* --------------------------------- card grid ------------------------------- */

function CardGrid({
  ads,
  savedIds,
  onOpen,
  onSave,
}: {
  ads: WallAd[];
  savedIds: Set<string>;
  onOpen: (ad: WallAd) => void;
  onSave: (ad: WallAd) => void;
}) {
  // The 2x2 lead slot needs imagery: promote the first ad with a still.
  const display = useMemo(() => {
    const i = ads.findIndex((a) => a.thumbnail || a.image);
    if (i <= 0) return ads;
    return [ads[i], ...ads.slice(0, i), ...ads.slice(i + 1)];
  }, [ads]);

  const g = useGStyle();
  if (display.length === 0) {
    return <Empty line1="Nothing here yet." line2="Add brands to the watchlist or run a search. The library builds itself." />;
  }
  if (g) return <CardGridG ads={display} savedIds={savedIds} onOpen={onOpen} onSave={onSave} />;

  return (
    <div className="grid grid-cols-2 gap-2 sm:grid-cols-3 xl:grid-cols-4">
      {display.map((ad, i) => {
        const isSaved = savedIds.has(ad.id);
        return (
          <div key={ad.id} className={`group relative ${i === 0 ? 'col-span-2 row-span-2' : ''}`}>
            <button
              type="button"
              onClick={() => onOpen(ad)}
              aria-label={`${ad.brand}: ${ad.daysRunning} days${ad.live ? ', still running' : ''}. Open dossier`}
              className="ap-card hoverable block w-full overflow-hidden rounded-[12px] text-left"
            >
              <div className={`ap-tile relative w-full ${i === 0 ? 'aspect-[4/3.05]' : 'aspect-[4/3]'}`}>
                {ad.thumbnail || ad.image ? (
                  // eslint-disable-next-line @next/next/no-img-element
                  <img
                    src={ad.thumbnail ?? ad.image ?? ''}
                    alt={`${ad.brand} ad creative`}
                    className="h-full w-full object-cover opacity-90 transition-opacity group-hover:opacity-100"
                    loading="lazy"
                  />
                ) : (
                  <div className="flex h-full items-center justify-center font-mono text-[11px] text-os-dim">{ad.format ?? 'ad'}</div>
                )}
                <div className="ap-caption-scrim absolute inset-x-0 bottom-0 h-16" />
                <div className="absolute bottom-2 left-2.5 right-2.5 flex items-end justify-between gap-2">
                  <span className="truncate text-[11.5px] text-os-muted">{ad.brand}</span>
                  <span className="flex shrink-0 items-baseline gap-1">
                    <span className={`text-[19px] font-light tabular-nums ${ad.live ? 'text-[var(--accent)]' : 'text-os-dim'}`}>
                      {ad.daysRunning}
                    </span>
                    <span className="text-[10px] text-os-dim">days</span>
                  </span>
                </div>
                {ad.live && (
                  <span role="img" aria-label="live" title="Still running" className="ap-solid absolute left-2 top-2 h-2 w-2 rounded-full" />
                )}
              </div>
              {ad.hook && (
                <p className={`px-2.5 py-2 text-[12px] leading-snug text-os-muted ${i === 0 ? 'line-clamp-3' : 'line-clamp-2'}`}>
                  “{ad.hook}”
                </p>
              )}
            </button>
            {/* Save toggle: the swipe-file affordance on every card */}
            <button
              type="button"
              aria-label={isSaved ? `Unsave ${ad.brand} ad` : `Save ${ad.brand} ad`}
              aria-pressed={isSaved}
              title={isSaved ? 'Saved: click to remove from Saved' : 'Save to your swipe file'}
              onClick={(e) => {
                e.stopPropagation();
                onSave(ad);
              }}
              className="ap-save absolute right-2 top-2 rounded-full border p-1.5 text-os-muted backdrop-blur transition-[opacity,color] hover:text-os-text"
            >
              <Bookmark aria-hidden className="h-3.5 w-3.5" strokeWidth={1.7} fill={isSaved ? 'currentColor' : 'none'} />
            </button>
          </div>
        );
      })}
    </div>
  );
}

function Empty({ line1, line2, children }: { line1: string; line2: string; children?: React.ReactNode }) {
  const g = useGStyle();
  if (g) {
    return (
      <div className="ap-g-empty">
        <GEmpty title={line1} body={line2} />
        {children}
      </div>
    );
  }
  return (
    <div className="flex h-full min-h-[200px] flex-col items-center justify-center text-center">
      <p className="text-[13px] text-os-muted">{line1}</p>
      <p className="mt-1 max-w-md text-[12px] text-os-dim">{line2}</p>
      {children}
    </div>
  );
}

/* --------------------------------- search --------------------------------- */

const SEARCH_COST = '~5 Foreplay credits';

function SearchTab({
  savedIds,
  onOpen,
  onSave,
  onResults,
  configured,
}: {
  savedIds: Set<string>;
  onOpen: (ad: WallAd) => void;
  onSave: (ad: WallAd) => void;
  onResults: (ads: WallAd[]) => void;
  configured: boolean;
}) {
  const [concept, setConcept] = useState('');
  const [minDays, setMinDays] = useState(21);
  const [format, setFormat] = useState<'all' | 'video' | 'image'>('all');
  const gate = useArm();
  const gc = useGc();
  const [state, setState] = useState<
    | { phase: 'idle' }
    | { phase: 'running' }
    | { phase: 'done'; probes: { query: string; results: number }[]; pooled: number; apiCalls: number; winners: WallAd[] }
    | { phase: 'error'; message: string; status: number }
  >({ phase: 'idle' });
  const tooShort = concept.trim().length < 3;
  const reason = !configured
    ? NOT_CONFIGURED
    : state.phase === 'running'
      ? 'Searching now'
      : tooShort
        ? 'Describe the concept in at least 3 characters'
        : null;

  const run = async () => {
    const q = concept.trim();
    if (reason) return;
    if (!gate.armed) {
      gate.arm();
      return;
    }
    gate.disarm();
    setState({ phase: 'running' });
    try {
      const res = await fetch('/api/adscout/mine', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ concept: q, minDays, ...(format !== 'all' ? { format } : {}) }),
      });
      const body = await readJson(res);
      if (!res.ok || !body.ok) {
        setState({ phase: 'error', message: String(body.error ?? `HTTP ${res.status}`), status: res.status });
        return;
      }
      const winners = (body.winners as WallAd[]) ?? [];
      setState({
        phase: 'done',
        probes: (body.probes as { query: string; results: number }[]) ?? [],
        pooled: Number(body.pooled ?? 0),
        apiCalls: Number(body.apiCalls ?? 0),
        winners,
      });
      onResults(winners);
    } catch (err) {
      setState({ phase: 'error', message: err instanceof Error ? err.message : 'search failed', status: 0 });
    }
  };

  const chip = (active: boolean, label: string, onClick: () => void) => (
    <button
      key={label}
      type="button"
      aria-pressed={active}
      onClick={onClick}
      className={gc('g-chip', `ap-pill rounded-full border px-2.5 py-1 text-[11.5px] transition-colors ${
        active ? 'text-[var(--accent)]' : 'text-os-dim hover:text-os-muted'
      }`)}
    >
      {label}
    </button>
  );

  return (
    <div>
      <label className="block text-[12.5px] text-os-muted" htmlFor="ap-concept">
        Describe a concept. Probes expand it and the library answers with proven ads.
      </label>
      <div className={gc('ap-g-searchbar', 'ap-glow ap-hair-strong ap-inset mt-2 flex items-center gap-2 rounded-[12px] border p-1.5')}>
        <Search aria-hidden className="ml-2 h-4 w-4 shrink-0 text-[var(--accent)]" strokeWidth={1.7} />
        <input
          id="ap-concept"
          value={concept}
          onChange={(e) => {
            setConcept(e.target.value);
            gate.disarm();
          }}
          onKeyDown={(e) => e.key === 'Enter' && void run()}
          placeholder="Describe a concept to search proven ads"
          className={gc('g-input ap-g-grow', 'min-w-0 flex-1 bg-transparent px-1 py-2 text-[14px] text-os-text outline-none placeholder:text-os-dim')}
        />
        <button
          type="button"
          onClick={() => void run()}
          onBlur={gate.disarm}
          disabled={reason !== null}
          title={reason ?? (gate.armed ? `Click again to spend ${SEARCH_COST}` : `Search (${SEARCH_COST})`)}
          className={gc('g-btn g-btn-ink', 'c-btn c-btn-primary flex items-center gap-1.5 text-[12.5px]')}
        >
          {state.phase === 'running' ? <Loader2 aria-hidden className="h-4 w-4 animate-spin" strokeWidth={1.7} /> : <Search aria-hidden className="h-4 w-4" strokeWidth={1.7} />}
          {gate.armed ? 'Confirm search' : 'Search ads'}
        </button>
      </div>
      <div className="mt-2 flex flex-wrap items-center gap-1.5">
        <span className="mr-1 text-[11px] text-os-dim">Proven for</span>
        {[21, 30, 60].map((d) => chip(minDays === d, `${d}+ days`, () => setMinDays(d)))}
        <span className="ml-3 mr-1 text-[11px] text-os-dim">Format</span>
        {(['all', 'video', 'image'] as const).map((f) => chip(format === f, f, () => setFormat(f)))}
        <span className={`ml-auto text-[11px] ${gate.armed ? 'text-os-muted' : 'text-os-dim'}`} aria-live="polite">
          {gate.armed ? `Spends ${SEARCH_COST}. Click Confirm search.` : `${SEARCH_COST} per search`}
        </span>
      </div>

      <div className="mt-4">
        {!configured && state.phase === 'idle' && (
          <Empty line1="Search needs Foreplay." line2="Concept search reads Foreplay's ad database. Add a key to turn it on.">
            <div className="mt-3">
              <Setup href={FOREPLAY_SETUP} hint="Add a Foreplay API key" />
            </div>
          </Empty>
        )}
        {configured && state.phase === 'idle' && (
          <Empty line1="Search the 100M-ad database by idea, not by brand." line2="Results are filtered to ads still paying off after your chosen threshold: the honest proof filter." />
        )}
        {state.phase === 'running' && (
          <p className="py-8 text-center text-[12.5px] text-os-dim">expanding concept → probing discovery → filtering to proven ads…</p>
        )}
        {state.phase === 'error' && (
          <div className="flex flex-col items-center gap-2 py-8 text-center">
            <p className="ap-err text-[12px]">{state.status === 503 ? NOT_CONFIGURED : state.message}</p>
            {state.status === 503 && <Setup href={FOREPLAY_SETUP} hint="Add a Foreplay API key" />}
          </div>
        )}
        {state.phase === 'done' && (
          <>
            <div className="mb-3 text-[11.5px] leading-relaxed text-os-dim">
              {state.probes.map((p) => (
                <span key={p.query} className="mr-3">
                  “{p.query}” → {p.results}
                </span>
              ))}
              <span className="text-os-muted">
                · {state.pooled} pooled · {state.apiCalls} credits
              </span>
            </div>
            <CardGrid ads={state.winners} savedIds={savedIds} onOpen={onOpen} onSave={onSave} />
          </>
        )}
      </div>
    </div>
  );
}

/* -------------------------------- watchlist -------------------------------- */

function WatchlistTab({
  watchlist,
  wall,
  setWatchlist,
  onRemove,
  ringBrand,
  configured,
}: {
  watchlist: WatchEntry[];
  wall: WallAd[];
  setWatchlist: (w: WatchEntry[]) => void;
  onRemove: (w: WatchEntry) => void;
  ringBrand: string | null;
  configured: boolean;
}) {
  const { toast } = useToast();
  const [domain, setDomain] = useState('');
  const [error, setError] = useState<string | null>(null);
  const [adding, setAdding] = useState(false);
  const rowRefs = useRef(new Map<string, HTMLDivElement | null>());
  const gc = useGc();

  useEffect(() => {
    if (!ringBrand) return;
    rowRefs.current.get(ringBrand)?.scrollIntoView({ block: 'center' });
  }, [ringBrand]);

  const liveCounts = useMemo(() => {
    const counts = new Map<string, { live: number; top: number }>();
    for (const ad of wall) {
      if (!ad.brandId) continue;
      const cur = counts.get(ad.brandId) ?? { live: 0, top: 0 };
      if (ad.live) cur.live += 1;
      cur.top = Math.max(cur.top, ad.daysRunning);
      counts.set(ad.brandId, cur);
    }
    return counts;
  }, [wall]);

  const reason = !configured ? `${NOT_CONFIGURED}: finding a brand by its website needs a key` : adding ? 'Looking the brand up' : null;

  const add = async () => {
    const d = parseBrandDomain(domain);
    if (!d) {
      setError(domain.trim() ? 'Enter a website like brand.com' : 'Enter the brand website first');
      return;
    }
    if (reason) return;
    setError(null);
    setAdding(true);
    try {
      const res = await fetch('/api/adscout/watchlist', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ domain: d }),
      });
      const body = await readJson(res);
      if (!res.ok || !body.ok) throw new Error(String(body.error ?? `HTTP ${res.status}`));
      setWatchlist(body.watchlist as WatchEntry[]);
      setDomain('');
      const added = body.added as WatchEntry | undefined;
      toast({ text: `Watching ${added?.name ?? d}. Refresh data to pull its ads.`, tone: 'ok' });
    } catch (err) {
      toast({ text: `Could not track ${d}: ${err instanceof Error ? err.message : 'request failed'}`, tone: 'err' });
    } finally {
      setAdding(false);
    }
  };

  return (
    <div className="mx-auto max-w-2xl">
      <label className="block text-[12px] text-os-muted" htmlFor="ap-domain">
        Track a brand: enter its website and Adscout finds its ad pages
      </label>
      <div className="mt-1.5 flex items-center gap-2">
        <input
          id="ap-domain"
          value={domain}
          onChange={(e) => {
            setDomain(e.target.value);
            if (error) setError(null);
          }}
          onKeyDown={(e) => e.key === 'Enter' && void add()}
          aria-invalid={error ? true : undefined}
          aria-describedby={error ? 'ap-domain-err' : undefined}
          placeholder="brand.com"
          className={gc('g-input ap-g-grow', 'ap-hair ap-inset min-w-0 flex-1 rounded-[10px] border px-3 py-2.5 text-[13px] text-os-text outline-none placeholder:text-os-dim')}
        />
        <button
          type="button"
          onClick={() => void add()}
          disabled={reason !== null}
          title={reason ?? 'Track this brand (one Foreplay lookup)'}
          className={gc('g-btn', 'c-btn flex items-center gap-1.5 text-[12.5px]')}
        >
          {adding ? <Loader2 aria-hidden className="h-4 w-4 animate-spin" strokeWidth={1.7} /> : <Plus aria-hidden className="h-4 w-4" strokeWidth={1.7} />}
          Track brand
        </button>
      </div>
      {error && (
        <p id="ap-domain-err" className="ap-err mt-1.5 text-[11.5px]">
          {error}
        </p>
      )}
      {!configured && (
        <p className="mt-1.5 flex flex-wrap items-center gap-2 text-[11.5px] text-os-dim">
          Adding by website needs Foreplay. Track brand on any ad&apos;s dossier works without it.
          <Setup href={FOREPLAY_SETUP} hint="Add a Foreplay API key" />
        </p>
      )}

      <div className="mt-4 flex flex-col gap-2">
        {watchlist.length === 0 && (
          <Empty line1="No brands tracked." line2="Tracked brands get synced daily: new launches, longevity winners, kills, velocity moves." />
        )}
        {watchlist.map((w) => {
          const counts = liveCounts.get(w.id);
          return (
            <div
              key={w.id}
              ref={(el) => {
                rowRefs.current.set(w.id, el);
              }}
              data-brand-id={w.id}
              data-ring={ringBrand === w.id ? '1' : undefined}
              className={gc('ap-ring ap-g-watchrow', 'ap-ring ap-hair-soft ap-inset flex items-center gap-3 rounded-[12px] border px-3.5 py-2.5')}
            >
              {w.avatar ? (
                // eslint-disable-next-line @next/next/no-img-element
                <img src={w.avatar} alt="" className="h-8 w-8 rounded-full object-cover" />
              ) : (
                <span aria-hidden className="flex h-8 w-8 items-center justify-center rounded-full bg-[var(--accent-soft)] text-[12px] text-[var(--accent)]">
                  {w.name.slice(0, 1)}
                </span>
              )}
              <div className="min-w-0 flex-1">
                <div className="truncate text-[13px] text-os-text">{w.name}</div>
                <div className="text-[11px] text-os-dim">{w.domain ?? 'added from a result'}</div>
              </div>
              {counts && (
                <div className="text-right text-[11.5px] tabular-nums text-os-muted">
                  {counts.live} live · longest {counts.top}d
                </div>
              )}
              <button
                type="button"
                aria-label={`Stop watching ${w.name}`}
                title={`Stop watching ${w.name} (undo for ${UNDO_MS / 1000}s)`}
                onClick={() => onRemove(w)}
                className={gc('g-btn g-btn-icon g-btn-sm', 'c-btn p-1.5')}
              >
                <X aria-hidden className="h-3.5 w-3.5" strokeWidth={1.7} />
              </button>
            </div>
          );
        })}
      </div>
    </div>
  );
}

/* -------------------------------- activity --------------------------------- */

const SIGNAL_META: Record<Signal['type'], { icon: typeof Zap; label: string }> = {
  new_launch: { icon: Zap, label: 'launched' },
  winner: { icon: Trophy, label: 'proven' },
  killed: { icon: XCircle, label: 'killed' },
  velocity_spike: { icon: TrendingUp, label: 'scaling up' },
  velocity_drop: { icon: TrendingDown, label: 'pulling back' },
};

const rowBase = 'ap-hair-soft flex w-full items-center gap-3 rounded-[10px] border px-3 py-2 text-left';

function ActivityRow({ s, ad, onOpen, sub = false }: { s: Signal; ad: WallAd | undefined; onOpen: (ad: WallAd) => void; sub?: boolean }) {
  const meta = SIGNAL_META[s.type];
  const Icon = meta.icon;
  const icon = <Icon aria-hidden className={`h-4 w-4 shrink-0 ${s.type === 'winner' ? 'text-[var(--accent)]' : 'text-os-dim'}`} strokeWidth={1.7} />;
  const text = <span className="min-w-0 flex-1 truncate text-[12.5px] text-os-muted">{sub ? s.at.slice(11, 16) + ' · ' + s.message : s.message}</span>;
  const gc = useGc();
  if (!ad) {
    // Nothing to open: a plain row, no hover, so it never reads as clickable.
    return (
      <div className={`${rowBase} opacity-80`} title="This ad is no longer in the library">
        {icon}
        {text}
      </div>
    );
  }
  return (
    <button type="button" onClick={() => onOpen(ad)} className={gc(`g-btn ap-g-rowbtn`, `${rowBase} ap-inset ap-row`)} title={`Open ${ad.brand} dossier`}>
      {icon}
      {text}
      {(ad.thumbnail || ad.image) && (
        // eslint-disable-next-line @next/next/no-img-element
        <img src={ad.thumbnail ?? ad.image ?? ''} alt="" className="h-8 w-11 shrink-0 rounded-[6px] object-cover" />
      )}
      <ChevronRight aria-hidden className="h-4 w-4 shrink-0 text-os-dim" strokeWidth={1.7} />
    </button>
  );
}

function ActivityTab({
  signals,
  total,
  wall,
  onOpen,
}: {
  signals: Signal[];
  total: number;
  wall: WallAd[];
  onOpen: (ad: WallAd) => void;
}) {
  const byAd = useMemo(() => new Map(wall.map((a) => [a.id, a])), [wall]);
  const [open, setOpen] = useState<Set<string>>(new Set());
  const gc = useGc();
  const groups = useMemo(() => {
    const map = new Map<string, Signal[]>();
    for (const s of signals) {
      const day = s.at.slice(0, 10);
      map.set(day, [...(map.get(day) ?? []), s]);
    }
    // Within one calendar day, identical consecutive messages collapse to one
    // row with ×N (SPEC X5); `ids` index back into the day's rows.
    return [...map.entries()]
      .sort((a, b) => b[0].localeCompare(a[0]))
      .map(([day, rows]) => ({ day, rows, runs: dedupeSignals(rows, { idOf: (_s, i) => String(i) }) }));
  }, [signals]);

  if (signals.length === 0) {
    return <Empty line1="Quiet." line2="After each refresh this lists what changed across your watchlist: launches, ads crossing 21/30/60 days, kills, and spend-velocity moves." />;
  }

  return (
    <div className="mx-auto flex max-w-2xl flex-col gap-5">
      <p className="text-[12px] text-os-dim">
        What changed across your watchlist, newest first.
        {total > signals.length && ` Showing the latest ${signals.length} of ${nf.format(total)}.`}
      </p>
      {groups.map(({ day, rows, runs }) => (
        <div key={day}>
          <div className="mb-2 text-[11.5px] text-os-dim">{day}</div>
          <div className="flex flex-col gap-1.5">
            {runs.map((run) => {
              const members = run.ids.map((id) => rows[Number(id)]);
              const first = members[0];
              const key = `${day}-${run.ids[0]}`;
              if (run.count === 1) {
                return <ActivityRow key={key} s={first} ad={first.adId ? byAd.get(first.adId) : undefined} onOpen={onOpen} />;
              }
              const expanded = open.has(key);
              const meta = SIGNAL_META[first.type];
              const Icon = meta.icon;
              return (
                <div key={key} className="flex flex-col gap-1">
                  <button
                    type="button"
                    aria-expanded={expanded}
                    onClick={() =>
                      setOpen((cur) => {
                        const next = new Set(cur);
                        if (next.has(key)) next.delete(key);
                        else next.add(key);
                        return next;
                      })
                    }
                    title={expanded ? 'Collapse' : `Show all ${run.count}`}
                    className={gc('g-btn ap-g-rowbtn', `${rowBase} ap-inset ap-row`)}
                  >
                    <Icon aria-hidden className={`h-4 w-4 shrink-0 ${first.type === 'winner' ? 'text-[var(--accent)]' : 'text-os-dim'}`} strokeWidth={1.7} />
                    <span className="min-w-0 flex-1 truncate text-[12.5px] text-os-muted">{first.message}</span>
                    <span className="shrink-0 rounded-full border border-os-border px-2 text-[11px] tabular-nums text-os-muted">×{run.count}</span>
                    <ChevronDown aria-hidden className={`h-4 w-4 shrink-0 text-os-dim transition-transform ${expanded ? 'rotate-180' : ''}`} strokeWidth={1.7} />
                  </button>
                  {expanded && (
                    <div className="ml-6 flex flex-col gap-1">
                      {members.map((s, i) => (
                        <ActivityRow key={`${key}-${i}`} s={s} ad={s.adId ? byAd.get(s.adId) : undefined} onOpen={onOpen} sub />
                      ))}
                    </div>
                  )}
                </div>
              );
            })}
          </div>
        </div>
      ))}
    </div>
  );
}

/* ------------------------------ dossier drawer ----------------------------- */

const SWIPE_CLOSE_PX = 80;

function Dossier({
  ad,
  isSaved,
  isWatched,
  onClose,
  onSave,
  onWatch,
}: {
  ad: WallAd;
  isSaved: boolean;
  isWatched: boolean;
  onClose: () => void;
  onSave: (ad: WallAd) => void;
  onWatch: (ad: WallAd) => void;
}) {
  const videoRef = useRef<HTMLVideoElement>(null);
  const [copied, setCopied] = useState(false);
  const [dx, setDx] = useState(0);
  const touch = useRef<{ x: number; y: number } | null>(null);
  const dlg = useDialog({ open: true, onClose, labelledBy: 'ap-dossier-title', id: 'adpilot-dossier' });

  const topDrivers = useMemo(
    () => Object.entries(ad.drivers).sort((a, b) => b[1] - a[1]).slice(0, 6),
    [ad.drivers],
  );

  const copyHook = async () => {
    if (!ad.hook) return;
    try {
      await navigator.clipboard.writeText(ad.hook);
      setCopied(true);
      setTimeout(() => setCopied(false), 1500);
    } catch {
      /* clipboard blocked */
    }
  };

  const gc = useGc();
  const g = useGStyle();
  const swipeStyle: CSSProperties | undefined = dx > 0 ? { transform: `translateX(${dx}px)` } : undefined;

  // Portalled to <body>: a frosted ancestor would trap the fixed drawer. The page's token scope rides
  // along on a box-less wrapper: [data-adpilot] in Classic (app/adpilot/page.tsx), .ap-g under GladOS.
  return (
    <Layer>
      <div className={gc('ap-g contents', 'contents')} data-adpilot={g ? undefined : ''}>
        <div className="ap-backdrop fixed inset-0 z-50 flex justify-end" onClick={onClose}>
          <div
            {...dlg.props}
            data-ad-id={ad.id}
            className={gc('ap-g-dossier', 'ap-panel h-full w-full max-w-[460px] overflow-y-auto rounded-none border-y-0 border-r-0 p-5 outline-none')}
            style={swipeStyle}
            onClick={(e) => e.stopPropagation()}
            onTouchStart={(e) => {
              const t = e.touches[0];
              touch.current = { x: t.clientX, y: t.clientY };
            }}
            onTouchMove={(e) => {
              const start = touch.current;
              if (!start) return;
              const t = e.touches[0];
              const ddx = t.clientX - start.x;
              if (Math.abs(t.clientY - start.y) > Math.abs(ddx)) return;
              setDx(Math.max(0, ddx));
            }}
            onTouchEnd={() => {
              touch.current = null;
              if (dx >= SWIPE_CLOSE_PX) onClose();
              else setDx(0);
            }}
          >
            <div className="mb-3 flex items-start justify-between gap-3">
              <div>
                <h2 id="ap-dossier-title" className="text-[15px] font-normal text-os-text">
                  {ad.brand}
                </h2>
                <div className="mt-0.5 text-[12px] text-os-dim">
                  {ad.format ?? 'ad'} · {ad.daysRunning} days {ad.live ? '· live' : '· off'}
                  {ad.ctaType ? ` · ${ad.ctaType.toLowerCase().replaceAll('_', ' ')}` : ''}
                </div>
              </div>
              <button type="button" data-dialog-close onClick={onClose} aria-label="Close dossier" title="Close (Esc)" className={gc('g-btn g-btn-icon g-btn-sm', 'c-btn p-1.5')}>
                <X aria-hidden className="h-4 w-4" strokeWidth={1.7} />
              </button>
            </div>

            {ad.video ? (
              <video ref={videoRef} src={ad.video} controls playsInline preload="metadata" className="ap-media w-full rounded-[10px]" />
            ) : ad.thumbnail || ad.image ? (
              // eslint-disable-next-line @next/next/no-img-element
              <img src={ad.thumbnail ?? ad.image ?? ''} alt={`${ad.brand} creative`} className="ap-media w-full rounded-[10px]" />
            ) : null}

            <div className="mt-3 flex flex-wrap gap-1.5">
              <button type="button" onClick={() => onSave(ad)} aria-pressed={isSaved} className={gc('g-btn g-btn-sm', 'c-btn flex items-center gap-1.5 text-[12px]')}>
                <Bookmark aria-hidden className="h-3.5 w-3.5" strokeWidth={1.7} fill={isSaved ? 'currentColor' : 'none'} />
                {isSaved ? 'Saved' : 'Save'}
              </button>
              {ad.brandId &&
                (isWatched ? (
                  <button type="button" disabled title={`${ad.brand} is already on your watchlist`} className={gc('g-btn g-btn-sm', 'c-btn flex items-center gap-1.5 text-[12px]')}>
                    <Check aria-hidden className="h-3.5 w-3.5" strokeWidth={1.7} /> Tracking
                  </button>
                ) : (
                  <button type="button" onClick={() => onWatch(ad)} className={gc('g-btn g-btn-sm', 'c-btn flex items-center gap-1.5 text-[12px]')}>
                    <Plus aria-hidden className="h-3.5 w-3.5" strokeWidth={1.7} /> Track brand
                  </button>
                ))}
              {ad.hook && (
                <button type="button" onClick={copyHook} className={gc('g-btn g-btn-sm', 'c-btn flex items-center gap-1.5 text-[12px]')}>
                  <Copy aria-hidden className="h-3.5 w-3.5" strokeWidth={1.7} /> {copied ? 'Copied' : 'Copy hook'}
                </button>
              )}
              {ad.linkUrl && (
                <a href={ad.linkUrl} target="_blank" rel="noreferrer" className={gc('g-btn g-btn-sm', 'c-btn flex items-center gap-1.5 text-[12px]')}>
                  <ExternalLink aria-hidden className="h-3.5 w-3.5" strokeWidth={1.7} /> Landing page
                </a>
              )}
            </div>

            {ad.hook && (
              <div className="mt-4">
                <div className="text-[11px] text-os-dim">Hook ({ad.hookSource})</div>
                <p className="mt-1 text-[13.5px] leading-snug text-os-text">“{ad.hook}”</p>
              </div>
            )}

            {topDrivers.length > 0 && (
              <div className="mt-4">
                <div className="text-[11px] text-os-dim">Emotional drivers</div>
                <div className="mt-2 flex flex-col gap-1.5">
                  {topDrivers.map(([axis, score]) => (
                    <div key={axis} className="flex items-center gap-2">
                      <span className="w-24 shrink-0 text-[12px] text-os-muted">{axis}</span>
                      <div className="ap-inset h-[7px] flex-1 overflow-hidden rounded-full" role="img" aria-label={`${axis} ${score} of 10`}>
                        <div className="ap-solid h-full rounded-full" style={{ width: `${(score / 10) * 100}%`, opacity: 0.4 + 0.06 * score }} />
                      </div>
                      <span className="w-6 shrink-0 text-right text-[11.5px] tabular-nums text-os-dim">{score}</span>
                    </div>
                  ))}
                </div>
              </div>
            )}

            {ad.transcript.length > 0 && (
              <div className="mt-4">
                <div className="text-[11px] text-os-dim">{ad.video ? 'Transcript: click a line to seek' : 'Transcript'}</div>
                <div className="mt-2 flex flex-col">
                  {ad.transcript.map((line, i) => {
                    const stamp = `${Math.floor(line.t / 60)}:${String(Math.floor(line.t % 60)).padStart(2, '0')}`;
                    const body = (
                      <>
                        <span className="shrink-0 text-[11px] tabular-nums text-os-dim group-hover:text-[var(--accent)]">{stamp}</span>
                        <span className="text-[12.5px] leading-snug text-os-muted group-hover:text-os-text">{line.s}</span>
                      </>
                    );
                    // Without a video there is nothing to seek: plain lines, not buttons.
                    return ad.video ? (
                      <button
                        key={i}
                        type="button"
                        onClick={() => {
                          const v = videoRef.current;
                          if (v) {
                            v.currentTime = line.t;
                            v.play().catch(() => undefined);
                          }
                        }}
                        title={`Play from ${stamp}`}
                        className={gc('g-btn ap-g-rowbtn group', 'ap-inset-hover group flex gap-2.5 rounded-[7px] px-2 py-1.5 text-left')}
                      >
                        {body}
                      </button>
                    ) : (
                      <div key={i} className="flex gap-2.5 px-2 py-1.5">
                        {body}
                      </div>
                    );
                  })}
                </div>
              </div>
            )}
          </div>
        </div>
      </div>
    </Layer>
  );
}

/* ------------------------------ GladOS parts ------------------------------- */

/** The band under GladOS (F32, F33): the library slab as the page's one hero, and one Foreplay card. */
function BandG({
  libOpen, onToggle, wallCount, longevity, savedCount, credits, configured, setupNeeded, lastSyncAt, now, onSync, onSyncBlur, armed, syncing, syncDisabledReason,
}: {
  libOpen: boolean;
  onToggle: () => void;
  wallCount: number;
  /** One tick per tracked ad (up to 60): how many days it has run. */
  longevity: { days: number; title: string }[];
  savedCount: number;
  credits: { remaining: number; total: number } | null;
  configured: boolean;
  setupNeeded: boolean;
  lastSyncAt: string | null;
  now: number;
  onSync: () => void;
  onSyncBlur: () => void;
  armed: boolean;
  syncing: boolean;
  syncDisabledReason: string | null;
}) {
  const share = credits && credits.total > 0 ? credits.remaining / credits.total : null;
  const low = share !== null && share < 0.1;
  return (
    <div className="ap-g-band">
      <GHeroCard
        variant="b"
        size="sm"
        caption={libOpen ? 'Close ad library' : 'Open ad library'}
        value={nf.format(wallCount)}
        label={`${nf.format(wallCount)} tracked ads`}
        status={`tracked ads · ${savedCount} saved`}
        title="Search by concept, save winners, watch brands"
        strip={longevity.length > 1 ? <GTickStrip onHero values={longevity.map((l) => l.days)} titles={longevity.map((l) => l.title)} label="Days each tracked ad has run" /> : undefined}
        onClick={onToggle}
        aria-expanded={libOpen}
        aria-controls="ap-library"
        className="ap-g-slab"
      />
      <GCard p={24} className="ap-g-foreplay" aria-labelledby="ap-g-foreplay">
        <div className="ap-g-inline">
          <h3 id="ap-g-foreplay" className="ap-g-card-title">Foreplay</h3>
          {configured && <span className="soc-g-cap">as of {relativeAgo(lastSyncAt, now)}</span>}
        </div>
        {setupNeeded ? (
          <GEmpty kind="unconfigured" inline title="No Foreplay key yet" body="Credits and refresh both need a Foreplay API key." setupHref={FOREPLAY_SETUP} setupHint="Add a Foreplay API key" />
        ) : (
          <>
            <div className="ap-g-credits">
              <span className="soc-g-num">{credits ? nf.format(credits.remaining) : '—'}</span>
              <span className="soc-g-cap">{credits ? `of ${nf.format(credits.total)} credits left` : 'credits are read on each refresh'}</span>
            </div>
            <GBar label={share === null ? 'Credits unknown' : `${Math.round(share * 100)}% of credits remaining${low ? ', running low' : ''}`} value={credits?.remaining ?? null} max={credits?.total ?? 1} tone={low ? 'warn' : undefined} />
            <div className="ap-g-inline">
              <span className="soc-g-cap" aria-live="polite">{armed ? 'Uses Foreplay credits: one call per watched brand' : 'Pull new ads and signals for every watched brand'}</span>
              <GButton
                variant={armed ? 'ink' : 'default'}
                size="sm"
                onClick={onSync}
                onBlur={onSyncBlur}
                busy={syncing}
                disabled={syncDisabledReason !== null}
                disabledReason={syncDisabledReason ?? undefined}
                title={armed ? 'Click again to spend Foreplay credits on a refresh' : 'Refresh data'}
              >
                <RefreshCw aria-hidden size={13} strokeWidth={1.75} /> {armed ? 'Confirm refresh' : 'Refresh data'}
              </GButton>
            </div>
          </>
        )}
      </GCard>
    </div>
  );
}

/** Ad cards under GladOS (F34): a GCard with the creative in its media slot, save as an icon button. */
function CardGridG({ ads, savedIds, onOpen, onSave }: { ads: WallAd[]; savedIds: Set<string>; onOpen: (ad: WallAd) => void; onSave: (ad: WallAd) => void }) {
  return (
    <div className="ap-g-cards">
      {ads.map((ad) => {
        const isSaved = savedIds.has(ad.id);
        return (
          <div key={ad.id} className="ap-g-adcard">
            <GCard
              p={16}
              onClick={() => onOpen(ad)}
              aria-label={`${ad.brand}: ${ad.daysRunning} days${ad.live ? ', still running' : ''}. Open dossier`}
              media={ad.thumbnail || ad.image ? (
                // eslint-disable-next-line @next/next/no-img-element
                <img src={ad.thumbnail ?? ad.image ?? ''} alt={`${ad.brand} ad creative`} loading="lazy" />
              ) : (
                <span className="ap-g-noimg">{ad.format ?? 'ad'}</span>
              )}
            >
              <span className="ap-g-inline">
                <span className="soc-g-sub ap-g-trunc">{ad.brand}</span>
                <GTag tone={ad.live ? 'accent' : 'neutral'} dot={ad.live} title={ad.live ? 'Still running' : 'No longer running'}>{ad.daysRunning} days</GTag>
              </span>
              {ad.hook && <span className="ap-g-hook">“{ad.hook}”</span>}
            </GCard>
            <GButton
              variant="icon"
              size="sm"
              aria-label={isSaved ? `Unsave ${ad.brand} ad` : `Save ${ad.brand} ad`}
              aria-pressed={isSaved}
              title={isSaved ? 'Saved: click to remove from Saved' : 'Save to your swipe file'}
              onClick={() => onSave(ad)}
              className="ap-g-save"
            >
              <Bookmark aria-hidden size={13} strokeWidth={1.75} fill={isSaved ? 'currentColor' : 'none'} />
            </GButton>
          </div>
        );
      })}
    </div>
  );
}
