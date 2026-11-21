'use client';

/**
 * Notion-style agent dock: a slim expand tab on the right edge of every view
 * opens a vertical Wizard panel that knows what screen you're on. The
 * panel fetches /api/conductor/context for the current route, shows what it
 * sees, and sends that context with every message so the agent can talk
 * about "this screen" concretely. Chat itself is the existing conductor
 * pipeline (routes to the best-fit agent, @agent-id to force one).
 *
 * GP2 (docs/projects/glados-ui/SPEC.md): any page opens the dock with a
 * prefill through `openWizard()` (lib/wizard-dock.ts, the `wizard:open`
 * event), and the "Send to every agent" chip sends one message to every
 * agent (POST /api/agents/broadcast) with the replies grouped by agent.
 *
 * Two looks, one behaviour (S00 §5.5): the Classic tree below is unchanged;
 * under the GladOS style the dock, launcher and composer render from the G
 * kit (GWizardDock, GLauncher, GAsk). `ui` comes from the root layout's
 * cookie read, then follows html[data-ui] live (lib/use-ui-style.ts).
 */
import { useCallback, useEffect, useRef, useState } from 'react';
import { usePathname } from 'next/navigation';
import { AlertTriangle, ChevronLeft, ChevronRight, Send, X } from 'lucide-react';
import { ConductorEmblem } from '@/components/ConductorEmblem';
import { GAsk, GButton, GCardRaised, GChip, GErrorLine, GProgress, GTag } from '@/components/g';
import type { UiStyle } from '@/lib/appearance';
import { useUiStyle } from '@/lib/use-ui-style';
import {
  WIZARD_OPEN_EVENT,
  groupBroadcastReplies,
  mergeDraft,
  nextBroadcast,
  sendEndpoint,
  type BroadcastReplyView,
  type WizardOpenDetail,
} from '@/lib/wizard-dock';
import { WizardIcon } from '@/lib/nav';
import { pushEscape } from '@/lib/escape-stack';
import {
  CONDUCTOR_STATE_EVENT,
  CONDUCTOR_TOGGLE_EVENT,
  launcherCopy,
  launcherState,
  unreadBadge,
  type ConductorStateDetail,
} from '@/lib/conductor-launcher';

type Turn = {
  id: string;
  role: 'user' | 'assistant';
  content: string;
  routedTo?: string;
  /** A message sent to every agent (user turn), or the grouped replies to one (assistant turn). */
  toAll?: boolean;
  groups?: { agentId: string; replies: BroadcastReplyView[] }[];
};

/** The composer is one line, which cannot hold a line break: a prefill joins an existing draft after a space. */
const DRAFT_JOIN = ' ';
type ScreenCtx = { title: string; context: string };

/** Cross-component open signal: anything can fire it to open the dock. */
export const CONDUCTOR_OPEN_EVENT = 'conductor:open';
export { CONDUCTOR_STATE_EVENT, CONDUCTOR_TOGGLE_EVENT };

const WIDTH_KEY = 'glados-conductor-w';
const MIN_W = 300;
const MAX_W = 760;
const clampW = (w: number) => Math.min(MAX_W, Math.max(MIN_W, Math.round(w)));

export function ConductorPanel({ ui = 'classic' }: { ui?: UiStyle } = {}) {
  const pathname = usePathname() ?? '/';
  const g = useUiStyle(ui) === 'glados';
  const [open, setOpen] = useState(false);
  // the operator controls the size: drag the left edge; the width persists
  const [width, setWidth] = useState(380);
  useEffect(() => {
    try {
      const stored = Number(localStorage.getItem(WIDTH_KEY));
      if (Number.isFinite(stored) && stored > 0) setWidth(clampW(stored));
    } catch {
      /* storage unavailable — default width stands */
    }
  }, []);
  const dragRef = useRef<{ startX: number; startW: number } | null>(null);
  // live width during a drag — state updates batch, so persisting from state
  // on pointerup can save a stale value on fast flicks
  const widthRef = useRef(380);
  useEffect(() => {
    widthRef.current = width;
  }, [width]);
  const onHandleDown = (e: React.PointerEvent<HTMLDivElement>) => {
    dragRef.current = { startX: e.clientX, startW: widthRef.current };
    // 1:1 tracking while dragging — the glide transition would lag the handle
    document.documentElement.classList.add('conductor-dragging');
    try {
      (e.target as HTMLElement).setPointerCapture(e.pointerId);
    } catch {
      /* synthetic/stale pointer — drag still tracks via move events */
    }
  };
  const onHandleMove = (e: React.PointerEvent<HTMLDivElement>) => {
    const d = dragRef.current;
    if (!d) return;
    const w = clampW(d.startW + (d.startX - e.clientX));
    widthRef.current = w;
    setWidth(w);
  };
  const onHandleUp = () => {
    if (!dragRef.current) return;
    dragRef.current = null;
    document.documentElement.classList.remove('conductor-dragging');
    try {
      localStorage.setItem(WIDTH_KEY, String(widthRef.current));
    } catch {
      /* fine */
    }
  };

  // Live `open` for async closures (a reply that lands after the dock was
  // closed counts as unread), and the element that opened the dock so
  // closing returns focus to it (S08 §6).
  const openRef = useRef(false);
  const openerRef = useRef<HTMLElement | null>(null);
  const launcherRef = useRef<HTMLButtonElement & HTMLAnchorElement>(null);
  const inputRef = useRef<HTMLInputElement>(null);
  const dockRef = useRef<HTMLElement>(null);
  /** The composer field in whichever tree is mounted (GAsk owns its own input). */
  const composer = useCallback(
    (): HTMLInputElement | null => inputRef.current ?? dockRef.current?.querySelector<HTMLInputElement>('[data-g="GAsk"] input') ?? null,
    [],
  );
  const [unread, setUnread] = useState(0);
  const [announce, setAnnounce] = useState('');

  const openDock = useCallback(() => {
    if (openRef.current) return;
    const el = document.activeElement;
    openerRef.current = el instanceof HTMLElement && el !== document.body ? el : null;
    openRef.current = true;
    setOpen(true);
    setUnread(0);
    setAnnounce('');
  }, []);
  const closeDock = useCallback(() => {
    if (!openRef.current) return;
    openRef.current = false;
    setOpen(false);
  }, []);

  // anything can open the dock (CONDUCTOR_OPEN_EVENT); the Topbar wand toggles it
  useEffect(() => {
    const onOpen = () => openDock();
    const onToggle = () => (openRef.current ? closeDock() : openDock());
    window.addEventListener(CONDUCTOR_OPEN_EVENT, onOpen);
    window.addEventListener(CONDUCTOR_TOGGLE_EVENT, onToggle);
    return () => {
      window.removeEventListener(CONDUCTOR_OPEN_EVENT, onOpen);
      window.removeEventListener(CONDUCTOR_TOGGLE_EVENT, onToggle);
    };
  }, [openDock, closeDock]);

  // Open: the cursor lands in the input so the user can type at once.
  // Close: focus returns to whichever launcher opened the dock (the corner
  // launcher remounts, so fall back to it when the opener left the DOM), but
  // only if focus was in the dock; a click elsewhere keeps its own focus.
  const wasOpen = useRef(false);
  useEffect(() => {
    if (open) {
      wasOpen.current = true;
      const t = window.setTimeout(() => composer()?.focus({ preventScroll: true }), 60);
      return () => window.clearTimeout(t);
    }
    if (!wasOpen.current) return;
    wasOpen.current = false;
    const active = document.activeElement;
    const inDock = !!active?.closest?.('[data-conductor-dock]');
    const opener = openerRef.current;
    openerRef.current = null;
    if (active && active !== document.body && !inDock) return;
    const target = opener && opener.isConnected ? opener : launcherRef.current;
    target?.focus({ preventScroll: true });
  }, [open, composer]);

  // Escape closes the dock only while it is open, through the one app-wide
  // stack (SPEC X1), so page-level Escape handlers are never raced.
  useEffect(() => {
    if (!open) return;
    return pushEscape(() => closeDock(), { id: 'conductor-dock' });
  }, [open, closeDock]);

  // the dock PUSHES the content instead of covering it: publish the width as
  // a CSS var the layout shell reads for its right margin, and tell the
  // Topbar wand (aria-expanded) and the sidebar (room for content, S09 FR-9)
  useEffect(() => {
    document.documentElement.style.setProperty('--conductor-w', open ? `${width}px` : '0px');
    const detail: ConductorStateDetail = { open, width: open ? width : 0 };
    window.dispatchEvent(new CustomEvent<ConductorStateDetail>(CONDUCTOR_STATE_EVENT, { detail }));
    return () => {
      document.documentElement.style.setProperty('--conductor-w', '0px');
    };
  }, [open, width]);
  const [ctx, setCtx] = useState<ScreenCtx | null>(null);
  const [turns, setTurns] = useState<Turn[]>([]);
  const [input, setInput] = useState('');
  // "Send to every agent" (GP2): off routes through the conductor; resets after each send
  const [broadcast, setBroadcast] = useState(false);
  const [sending, setSending] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const scrollRef = useRef<HTMLDivElement>(null);

  const loadContext = useCallback(async (path: string) => {
    setCtx(null);
    try {
      const res = await fetch(`/api/conductor/context?path=${encodeURIComponent(path)}`);
      if (res.ok) setCtx((await res.json()) as ScreenCtx);
    } catch {
      // panel still works without context — the chat just loses screen grounding
    }
  }, []);

  useEffect(() => {
    if (open) void loadContext(pathname);
  }, [open, pathname, loadContext]);

  useEffect(() => {
    scrollRef.current?.scrollTo({ top: scrollRef.current.scrollHeight });
  }, [turns, sending]);

  // GP2: openWizard({ text, broadcast }) opens the dock with the text in the
  // composer (never sent), an existing draft kept, the chip set, and the
  // cursor at the end of the text.
  const [focusTick, setFocusTick] = useState(0);
  useEffect(() => {
    const onWizard = (e: Event) => {
      const d = (e as CustomEvent<WizardOpenDetail>).detail ?? {};
      setInput((draft) => mergeDraft(draft, d.text, DRAFT_JOIN));
      setBroadcast((on) => nextBroadcast(on, d.broadcast));
      openDock();
      setFocusTick((n) => n + 1);
    };
    window.addEventListener(WIZARD_OPEN_EVENT, onWizard);
    return () => window.removeEventListener(WIZARD_OPEN_EVENT, onWizard);
  }, [openDock]);
  useEffect(() => {
    if (!focusTick) return;
    // after the open effect's own focus (60ms), so the caret lands at the end
    const t = window.setTimeout(() => {
      const el = composer();
      if (!el) return;
      el.focus({ preventScroll: true });
      const end = el.value.length;
      try {
        el.setSelectionRange(end, end);
      } catch {
        /* input types without a selection */
      }
    }, 80);
    return () => window.clearTimeout(t);
  }, [focusTick, composer]);

  async function send() {
    const text = input.trim();
    if (!text || sending) return;
    const toAll = broadcast;
    setSending(true);
    setError(null);
    setTurns((t) => [...t, { id: `u-${t.length}`, role: 'user', content: text, toAll }]);
    setInput('');
    setBroadcast(false); // the chip resets after each send (FR-GP2.2)
    try {
      const res = await fetch(sendEndpoint(toAll), {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        // a broadcast is the bare message: every agent answers it on its own terms
        body: JSON.stringify(
          toAll
            ? { message: text }
            : { message: text, context: ctx ? `Screen: ${ctx.title}\n${ctx.context}` : undefined },
        ),
      });
      if (!res.ok) {
        const body = (await res.json().catch(() => null)) as { error?: string } | null;
        throw new Error(body?.error ?? `${toAll ? 'broadcast' : 'conductor'} failed (${res.status})`);
      }
      if (toAll) {
        const body = (await res.json()) as { broadcast?: { replies?: unknown } };
        const groups = groupBroadcastReplies(body.broadcast?.replies);
        setTurns((t) => [...t, { id: `a-${t.length}`, role: 'assistant', content: '', toAll: true, groups }]);
      } else {
        const body = (await res.json()) as { routedTo: string; reply: string };
        setTurns((t) => [...t, { id: `a-${t.length}`, role: 'assistant', content: body.reply, routedTo: body.routedTo }]);
      }
      // a real reply that lands while the dock is closed stays unread until it reopens
      if (!openRef.current) {
        setUnread((n) => n + 1);
        setAnnounce('Wizard replied');
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
      if (!openRef.current) setAnnounce('Wizard message failed');
    } finally {
      setSending(false);
    }
  }

  const persistWidth = (w: number) => {
    const next = clampW(w);
    widthRef.current = next;
    setWidth(next);
    try {
      localStorage.setItem(WIDTH_KEY, String(next));
    } catch {
      /* fine */
    }
  };

  const state = launcherState({ open, sending, unread, error });
  const copy = launcherCopy(state, unread);
  const badge = unreadBadge(unread);
  const placeholder = broadcast ? 'Ask every agent…' : `Ask about ${ctx?.title ?? 'this screen'}…`;
  const broadcastTitle = broadcast
    ? 'On: this message goes to every agent, each answers on its own. Turns off after sending.'
    : 'Off: the Wizard routes this message to the best agent. Turn on to ask every agent.';

  if (g) {
    // ── GladOS tree (S00 §5.5): GLauncher, GWizardDock, GAsk composer ──
    return (
      <>
        {!open && (
          <div className="g-launch-wrap" data-g="GLauncher" data-state={state}>
            <GButton
              ref={launcherRef}
              variant="icon"
              onClick={openDock}
              aria-label={copy.aria}
              className={`g-launch${state === 'idle' ? '' : ` is-${state}`}`}
            >
              {state === 'thinking' && <span className="g-launch-orbit" aria-hidden />}
              <WizardIcon className="g-launch-wand" strokeWidth={1.5} aria-hidden />
              <span aria-hidden className="g-launch-label">
                {copy.pop}
              </span>
              {(state === 'unread' || state === 'error') && (
                <span aria-hidden className={`g-launch-dot${state === 'error' ? ' is-err' : ''}${state === 'unread' && badge ? ' has-count' : ''}`}>
                  {state === 'error' ? <AlertTriangle className="g-launch-err-icon" strokeWidth={2.2} /> : badge}
                </span>
              )}
            </GButton>
          </div>
        )}
        <span className="sr-only" aria-live="polite">
          {open ? '' : announce}
        </span>

        <aside
          ref={dockRef}
          aria-hidden={!open}
          aria-label="Wizard"
          data-conductor-dock=""
          data-g="GWizardDock"
          data-open={open || undefined}
          className="g-dock"
          style={{ width }}
        >
          <div
            onPointerDown={onHandleDown}
            onPointerMove={onHandleMove}
            onPointerUp={onHandleUp}
            title="Drag to resize"
            className="g-dock-handle"
          />
          <div className="g-dock-edge">
            <GButton
              variant="icon"
              size="sm"
              onClick={() => persistWidth(widthRef.current + 140)}
              disabled={width >= MAX_W}
              disabledReason="Already at the widest size"
              aria-label="Widen the panel"
              title="Wider"
            >
              <ChevronLeft size={14} strokeWidth={1.75} aria-hidden />
            </GButton>
            <GButton variant="icon" size="sm" onClick={closeDock} aria-label="Slide the panel away" title="Slide away">
              <ChevronRight size={14} strokeWidth={1.75} aria-hidden />
            </GButton>
          </div>
          <header className="g-dock-head">
            <ConductorEmblem size={32} thinking={sending} glyph="wand" className="shrink-0" />
            <div className="g-dock-head-text">
              <div className="g-dock-title">Wizard</div>
              <div className="g-dock-cap" title={ctx?.title ? `The Wizard can see ${ctx.title}` : undefined}>
                Seeing {ctx?.title ?? 'this screen'}
              </div>
            </div>
            <GButton variant="icon" onClick={closeDock} aria-label="Close Wizard">
              <X size={16} strokeWidth={1.75} aria-hidden />
            </GButton>
          </header>

          <div ref={scrollRef} className="g-dock-scroll" data-empty={turns.length === 0 && !sending && !error ? '' : undefined}>
            {turns.length === 0 && !sending && !error && (
              <div className="g-dock-empty">
                <ConductorEmblem size={44} glyph="wand" />
                <h2 className="g-dock-empty-title">What can I help with?</h2>
                <p className="g-dock-empty-body">
                  I can see {ctx?.title ?? 'this screen'}. Type @agent to ask one directly, or turn on Send to every agent.
                </p>
              </div>
            )}
            {turns.map((t) =>
              t.role === 'user' ? (
                <div key={t.id} className="g-dock-turn" data-role="user">
                  {t.toAll && (
                    <GTag size="xs" tone="accent">
                      To every agent
                    </GTag>
                  )}
                  <GCardRaised p={16} className="g-dock-bubble">
                    {t.content}
                  </GCardRaised>
                </div>
              ) : t.toAll ? (
                <div key={t.id} className="g-dock-turn" data-role="assistant">
                  {(t.groups ?? []).length === 0 && <p className="g-dock-text g-dock-quiet">No agent answered.</p>}
                  {(t.groups ?? []).map((grp) => (
                    <div key={grp.agentId} className="g-dock-group">
                      <GTag size="xs" tone={grp.replies.every((r) => r.ok) ? 'neutral' : 'err'}>
                        {grp.agentId}
                      </GTag>
                      {grp.replies.map((r, i) => (
                        <p key={i} className="g-dock-text" data-failed={r.ok ? undefined : ''}>
                          {r.ok ? r.reply || 'No reply text.' : r.reply ? `Failed: ${r.reply}` : 'Failed without a message.'}
                        </p>
                      ))}
                    </div>
                  ))}
                </div>
              ) : (
                <div key={t.id} className="g-dock-turn" data-role="assistant">
                  {t.routedTo && (
                    <GTag size="xs" title={`Routed to ${t.routedTo}`}>
                      {t.routedTo}
                    </GTag>
                  )}
                  <p className="g-dock-text">{t.content}</p>
                </div>
              ),
            )}
            {sending && <GProgress text={turns[turns.length - 1]?.toAll ? 'Asking every agent…' : 'Routing…'} />}
            {error && <GErrorLine align="left">{error}</GErrorLine>}
          </div>

          <div className="g-dock-compose">
            <div className="g-dock-chips">
              <GChip pressed={broadcast} onPressedChange={setBroadcast} size="compact" title={broadcastTitle} disabled={sending} disabledReason="Wait for the reply">
                Send to every agent
              </GChip>
            </div>
            <GAsk
              value={input}
              onValueChange={setInput}
              onSubmit={() => void send()}
              busy={sending}
              label="Message the Wizard"
              placeholder={placeholder}
              minChars={1}
            />
          </div>
        </aside>
      </>
    );
  }

  return (
    <>
      {/* Corner launcher (S08): the Wizard's wand, bottom-right on every
          route. While the dock is closed it reports the dock's state (working,
          new replies, a failed send). Its pop-out label is the tooltip, on
          hover and on keyboard focus. Click opens the dock, cursor in the input. */}
      {!open && (
        <button
          ref={launcherRef}
          type="button"
          onClick={openDock}
          aria-label={copy.aria}
          data-state={state}
          className={`wiz-launch group fixed bottom-5 right-5 z-40 flex items-center rounded-full border border-os-border-strong bg-os-surface/90 p-2.5 backdrop-blur hover:pr-3.5 focus-visible:pr-3.5${
            state === 'idle' ? '' : ` is-${state}`
          }`}
        >
          <WizardIcon className="wiz-launch-wand h-[18px] w-[18px] shrink-0" strokeWidth={1.5} aria-hidden />
          <span
            aria-hidden
            className="wiz-launch-label max-w-0 overflow-hidden whitespace-nowrap font-mono text-[12px] tracking-wide text-os-muted group-hover:ml-2 group-hover:max-w-[180px] group-focus-visible:ml-2 group-focus-visible:max-w-[180px]"
          >
            {copy.pop}
          </span>
          {(state === 'unread' || state === 'error') && (
            <span
              aria-hidden
              className={`wiz-launch-dot${state === 'error' ? ' is-err' : ''}${state === 'unread' && badge ? ' has-count' : ''}`}
            >
              {state === 'error' ? <AlertTriangle className="wiz-launch-err-icon" strokeWidth={2.2} /> : badge}
            </span>
          )}
        </button>
      )}
      {/* announces a reply or a failure that lands while the dock is closed */}
      <span className="sr-only" aria-live="polite">
        {open ? '' : announce}
      </span>

      {/* the dock — opened from the Topbar wand or the corner launcher,
          resizable from its left edge, width remembered across sessions */}
      <aside
        ref={dockRef}
        aria-hidden={!open}
        data-conductor-dock=""
        className={`c-pop fixed inset-y-0 right-0 !rounded-none z-50 flex max-w-[92vw] flex-col border-l border-os-border-strong bg-os-surface transition-transform duration-[420ms] ${
          // closed, the dock sits just off-screen: its popover shadow would bleed a
          // grey strip down the right edge of every page on lighter grounds (S06)
          open ? 'translate-x-0' : 'invisible translate-x-full !shadow-none'
        }`}
        style={{ transitionTimingFunction: 'var(--ease)', width, transitionProperty: 'transform, visibility' }}
      >
        {/* resize handle */}
        <div
          onPointerDown={onHandleDown}
          onPointerMove={onHandleMove}
          onPointerUp={onHandleUp}
          title="Drag to resize"
          className="absolute inset-y-0 left-0 z-10 w-1.5 cursor-col-resize hover:bg-os-accent/30"
          style={{ touchAction: 'none' }}
        />

        {/* edge arrows: ‹ widens the panel a step, › slides it away — hidden
            while closed so they don't poke past the off-screen edge */}
        <div
          className={`absolute -left-3 top-1/2 z-20 flex -translate-y-1/2 flex-col gap-2.5 transition-opacity duration-300 ${
            open ? 'opacity-100' : 'pointer-events-none opacity-0'
          }`}
        >
          <button
            onClick={() => persistWidth(widthRef.current + 140)}
            disabled={width >= MAX_W}
            aria-label="Widen the panel"
            title={width >= MAX_W ? 'Already at the widest size' : 'Wider'}
            className="grid h-7 w-7 place-items-center rounded-full border border-os-border-strong bg-os-surface text-os-dim shadow-sm transition-colors hover:text-os-accent disabled:opacity-30"
          >
            <ChevronLeft className="h-3.5 w-3.5" />
          </button>
          <button
            onClick={closeDock}
            aria-label="Slide the panel away"
            title="Slide away"
            className="grid h-7 w-7 place-items-center rounded-full border border-os-border-strong bg-os-surface text-os-dim shadow-sm transition-colors hover:text-os-text"
          >
            <ChevronRight className="h-3.5 w-3.5" />
          </button>
        </div>
        <header className="flex items-center gap-3.5 border-b border-os-border px-4 py-3">
          {/* the wand at the core, so the open dock matches its launcher (Q-SH-1) */}
          <ConductorEmblem size={32} thinking={sending} glyph="wand" className="shrink-0" />
          <div className="min-w-0 flex-1">
            <div className="text-[13.5px] font-medium">Wizard</div>
            <div className="truncate text-[11.5px] text-os-dim">
              seeing: {ctx?.title ?? '…'}
            </div>
          </div>
          <button
            onClick={closeDock}
            aria-label="Close Wizard"
            className="shrink-0 rounded-sm-t p-1 text-os-dim transition-colors hover:text-os-text"
          >
            <X className="h-4 w-4" />
          </button>
        </header>

        <div
          ref={scrollRef}
          className={`flex-1 overflow-y-auto px-4 py-3 ${turns.length === 0 && !sending && !error ? 'flex' : 'space-y-3'}`}
        >
          {/* empty: one quiet line in the middle, like a fresh chat */}
          {turns.length === 0 && !sending && !error && (
            <div className="wiz-empty m-auto flex flex-col items-center gap-3 px-6 text-center">
              <ConductorEmblem size={44} glyph="wand" />
              <h2 className="text-[19px] font-medium tracking-[-0.01em] text-os-text">What can I help with?</h2>
              <p className="max-w-[260px] text-[12.5px] leading-relaxed text-os-dim">
                I can see {ctx?.title ?? 'this screen'}. Type <span className="font-mono">@agent</span> to ask one directly.
              </p>
            </div>
          )}
          {turns.map((t) =>
            t.role === 'assistant' && t.toAll ? (
              <div key={t.id} className="space-y-2 text-left">
                {(t.groups ?? []).length === 0 && <p className="text-[12.5px] text-os-dim">No agent answered.</p>}
                {(t.groups ?? []).map((grp) => (
                  <div key={grp.agentId}>
                    <div className="mb-0.5 font-mono text-[10.5px] text-[var(--route-ink)]">→ {grp.agentId}</div>
                    {grp.replies.map((r, i) => (
                      <span
                        key={i}
                        className={`inline-block max-w-[92%] whitespace-pre-wrap break-words rounded-md border border-os-border bg-os-bg px-2.5 py-2.5 text-[13px] leading-relaxed ${r.ok ? 'text-os-muted' : 'text-os-err'}`}
                      >
                        {r.ok ? r.reply || 'No reply text.' : r.reply ? `Failed: ${r.reply}` : 'Failed without a message.'}
                      </span>
                    ))}
                  </div>
                ))}
              </div>
            ) : t.role === 'user' ? (
              <div key={t.id} className="text-right">
                <span className="glass-well gl-on-rim inline-block max-w-[88%] break-words px-2.5 py-2.5 text-left text-[13px] text-os-text">
                  {t.content}
                </span>
              </div>
            ) : (
              <div key={t.id} className="text-left">
                {t.routedTo && (
                  <div className="mb-0.5 font-mono text-[10.5px] text-[var(--route-ink)]">
                    → {t.routedTo}
                  </div>
                )}
                <span className="inline-block max-w-[92%] whitespace-pre-wrap break-words rounded-md border border-os-border bg-os-bg px-2.5 py-2.5 text-[13px] leading-relaxed text-os-muted">
                  {t.content}
                </span>
              </div>
            ),
          )}
          {sending && (
            <div className="flex items-center gap-3">
              <ConductorEmblem size={18} thinking />
              <span className="font-mono text-[11.5px] text-os-dim">{turns[turns.length - 1]?.toAll ? 'asking every agent…' : 'routing…'}</span>
            </div>
          )}
          {error && (
            <p className="flex items-start gap-1.5 font-mono text-[11.5px] text-os-err">
              <AlertTriangle aria-hidden className="mt-[2px] h-3 w-3 shrink-0" strokeWidth={2} />
              <span className="min-w-0 break-words">{error}</span>
            </p>
          )}
        </div>

        {/* composer: the same orbiting light behind frosted glass as the
            Blueprint and AdPilot ask bars, on the theme's accent */}
        <div className="p-3 pt-1">
          {/* GP2: one message to every agent instead of the routed one */}
          <div className="wiz-bc-row">
            <button
              type="button"
              className="wiz-bc-chip"
              aria-pressed={broadcast}
              onClick={() => setBroadcast((on) => !on)}
              disabled={sending}
              title={broadcastTitle}
            >
              Send to every agent
            </button>
          </div>
          <div className={`wiz-ask${sending ? ' is-busy' : ''}`}>
            <div className="wiz-ask-orbit" aria-hidden="true" />
            <div className="wiz-ask-glass" aria-hidden="true" />
            <div className="wiz-ask-row">
              <input
                ref={inputRef}
                value={input}
                onChange={(e) => setInput(e.target.value)}
                onKeyDown={(e) => e.key === 'Enter' && send()}
                placeholder={placeholder}
                aria-label="Message the Wizard"
                autoComplete="off"
                disabled={sending}
              />
              <button type="button" onClick={send} disabled={sending || !input.trim()} aria-label="Send" className="wiz-ask-send">
                <Send />
              </button>
            </div>
          </div>
        </div>
      </aside>
    </>
  );
}
