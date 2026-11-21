'use client';

import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { Activity, AlertTriangle, Box, Camera, Check, ChevronDown, Copy, Eye, EyeOff, FileText, Film, Images, Keyboard, Layers, Lock, LockOpen, MessageSquare, Palette, RotateCcw, Settings2, SkipBack, SkipForward, Square, Target, Trash2, UserRound, Wand2, X } from 'lucide-react';
import type { Campaign, ConsoleControls, VideoModel } from '@/lib/creative-engine/schemas';
import { CONSOLE_LIMITS, ConsoleControlsSchema, FRAMINGS, HEIGHTS, LENSES, LIGHTING, MOVEMENTS, SETTINGS, STYLES, TEXT_REVEALS, VIDEO_MODELS } from '@/lib/creative-engine/schemas';
import { CONTRIBUTIONS, INTRO_OPTIONS, TONE_OPTIONS, composePrompt } from '@/lib/creative-engine/compose';
import { MODEL_RULES, UNVERIFIED_MODELS, referencePaths, stripIgnoredRefs, validateRequest, type RefCheck } from '@/lib/arcads/validate';
import { DURATION_RULES, durationForScript } from '@/lib/creative-engine/duration';
import type { ResolvedRun } from '@/lib/creative-engine/runs';
import { api, ApiError, type CampaignSummary, type HealthSnapshot } from '@/components/creative-engine/api';
import { fmtMs } from '@/components/creative-engine/primitives';
import { CheckRow, LinkRow, SelectRow, SliderRow, SwatchRow, TextRow, type ResetProps } from '@/components/creative-engine/octane-rows';
import { WorkflowCanvas } from '@/components/creative-engine/WorkflowCanvas';
import { SpendGate } from '@/components/creative-engine/SpendGate';
import { OBtn, OInput, OSelect, OTextarea, SetupLink } from '@/components/creative-engine/skin';
import { ConsoleErrorBoundary } from '@/components/creative-engine/ConsoleErrorBoundary';
import { Layer } from '@/components/g/Overlays';
import { ToastProvider, useToast } from '@/components/ui/Toast';
import { lockBodyScroll } from '@/components/ui/use-dialog';
import { pushEscape } from '@/lib/escape-stack';
import { fmtCredits, plural } from '@/lib/creative-engine/format';
import {
  COLUMN_KEYS, CONSOLE_SHORTCUTS, chipWidths, clampControls, clampReport, controlsIssue, defaultFor, draftKey, generateBlockReason,
  isModified, isPresetModified, parseDraft, shortcutFor, titleCaseStatus, trimNotice, type BlockReason, type ConsoleDraft, type DraftPart,
} from '@/lib/creative-engine/console-ui';

/**
 * Motion console (DESIGN.md §2). Built to the reference's measurements: a
 * near black window floating on a grey ground; a top bar; a tool strip,
 * transport and readout; a scene tree whose eye hides a prompt part and
 * whose lock protects it from "Describe it"; a full-height stage with a
 * drawer for the active tool panel; three parameter columns of 34px rows;
 * a node canvas of the approved script's beats. Every colour is a token.
 *
 * Every element does something (S05a): pills, readout lines, HUD rows,
 * carets and value boxes are real buttons or selects; a control that cannot
 * act says why in its title and, for Generate, in a caption.
 */

type Part = keyof typeof CONTRIBUTIONS;
type Panel = 'prompt' | 'presets' | 'dialogue' | 'lints' | 'describe' | 'shortcuts' | null;
type Preset = { id: string; name: string; builtin: boolean; controls: Partial<ConsoleControls> };
type Column = keyof typeof COLUMN_KEYS;
type Menu = 'console' | 'abandon' | `column:${Column}` | null;
/** Sheets and drawers the console hands over to; the Dashboard returns here when they close (S05a FR-15). */
export type ConsoleSheet = 'runs' | 'credits' | 'connect' | 'scripter' | 'generator';

const DEFAULTS: ConsoleControls = ConsoleControlsSchema.parse({});
const PART_ORDER: Part[] = ['subject', 'action', 'camera', 'style', 'constraints'];
const PART_META: Record<Part, { label: string; colour: string; icon: React.ReactNode }> = {
  subject: { label: 'Subject', colour: 'var(--o-part-1)', icon: <UserRound className="h-4 w-4" strokeWidth={1.6} /> },
  action: { label: 'Action', colour: 'var(--o-part-2)', icon: <Film className="h-4 w-4" strokeWidth={1.6} /> },
  camera: { label: 'Camera', colour: 'var(--o-part-3)', icon: <Camera className="h-4 w-4" strokeWidth={1.6} /> },
  style: { label: 'Style', colour: 'var(--o-part-4)', icon: <Palette className="h-4 w-4" strokeWidth={1.6} /> },
  constraints: { label: 'Constraints', colour: 'var(--o-part-5)', icon: <Lock className="h-4 w-4" strokeWidth={1.6} /> },
};
const COLUMN_PART: Record<Column, Part | null> = { shot: 'camera', look: 'style', output: null };
const COLUMN_NAME: Record<Column, string> = { shot: 'Shot', look: 'Look', output: 'Output' };
const ASPECTS = ['9:16', '16:9', '1:1', 'auto'] as const;
const RESOLUTIONS = ['480p', '720p', '1080p', '4K'] as const;
const INTENSITY_WORDS = ['slowly', 'gently', 'casually', 'quickly', 'deliberately'] as const;
const TABS = [
  { key: 'scene', label: 'Scene', icon: <Layers className="h-3.5 w-3.5" strokeWidth={2} /> },
  { key: 'presets', label: 'Presets', icon: <Settings2 className="h-3.5 w-3.5" strokeWidth={2} /> },
  { key: 'references', label: 'Refs', icon: <Images className="h-3.5 w-3.5" strokeWidth={2} /> },
] as const;
type Tab = (typeof TABS)[number]['key'];
/** Errors raised by `set` for a field limit; a later valid edit clears them. */
const LIMIT_ERROR = / (characters )?(max|min)$/;
const PANEL_TITLE: Record<Exclude<Panel, null>, string> = { prompt: 'Composed prompt', presets: 'Presets', dialogue: 'Dialogue', lints: 'Lints', describe: 'Describe it', shortcuts: 'Keyboard shortcuts' };
const SAMPLE_OFF = 'Sample data: actions are off';
/** Fields sent to Arcads as request fields (the Output column's "Show in composed prompt"). */
const BOUND_FIELDS: { key: keyof ConsoleControls; label: string }[] = [
  { key: 'model', label: 'Model' }, { key: 'duration', label: 'Duration' }, { key: 'aspectRatio', label: 'Aspect' }, { key: 'resolution', label: 'Resolution' },
  { key: 'variations', label: 'Variations' }, { key: 'audioEnabled', label: 'Audio' }, { key: 'enhance', label: 'Enhance prompt' }, { key: 'bitrateMode', label: 'Bitrate' },
];

type ConsoleProps = {
  campaign: Campaign | null;
  health: HealthSnapshot;
  runs: ResolvedRun[];
  readOnly?: boolean;
  onClose: () => void;
  onConnect: () => void;
  onCampaign: (c: Campaign) => void;
  onRuns: (r: ResolvedRun[]) => void;
  /** Campaign picker (S05a T4, FR-5): the Dashboard's list and its loader. */
  campaigns?: CampaignSummary[];
  onSwitchCampaign?: (id: string) => Promise<void> | void;
  /** Hand over to a Dashboard sheet; the Dashboard reopens the console when it closes (FR-15). */
  onOpenSheet?: (sheet: ConsoleSheet, opts?: { runsKind?: 'motion' }) => void;
  /** Open with this asset on stage (G3). Unknown ids leave the console on its default. */
  initialAssetId?: string | null;
};

function readStoredDraft(campaignId: string | null | undefined, readOnly: boolean): ConsoleDraft | null {
  if (readOnly || typeof window === 'undefined') return null;
  try { return parseDraft(window.localStorage.getItem(draftKey(campaignId)), ConsoleControlsSchema); } catch { return null; }
}
function clearStoredDraft(campaignId: string | null | undefined) {
  try { window.localStorage.removeItem(draftKey(campaignId)); } catch { /* storage blocked */ }
}

/** The console inside its error boundary and its own toast stack: a render error shows a Reset controls / Close card, never a white screen (S05a FR-2). */
export function MotionConsole(props: ConsoleProps) {
  // The console is a full-viewport layer, portalled to <body> (a frosted ancestor would trap it). It is
  // its own scope root (.ae-octane): the workflow's dialogs portal into it.
  return (
    <ConsoleErrorBoundary onClose={props.onClose} onReset={() => clearStoredDraft(props.campaign?.id)}>
      <Layer>
        <div className="ae-octane ae-o-ground fixed inset-0 z-[130] overflow-y-auto">
          <ToastProvider>
            <MotionConsoleBody {...props} />
          </ToastProvider>
        </div>
      </Layer>
    </ConsoleErrorBoundary>
  );
}

/** Merge a patch into the controls without ever throwing: clamp to the limits, validate, keep `cur` if it still fails. */
function mergeControls(cur: ConsoleControls, patch: Partial<ConsoleControls>): ConsoleControls {
  const r = ConsoleControlsSchema.safeParse({ ...cur, ...clampControls(patch) });
  return r.success ? r.data : cur;
}

function valueText(key: keyof ConsoleControls, v: unknown): string {
  if (typeof v === 'boolean') return v ? 'on' : 'off';
  if (key === 'duration') return `${String(v)}s`;
  if (key === 'lensMm') return `${String(v)} mm`;
  if (key === 'intensity' && typeof v === 'number') return INTENSITY_WORDS[v - 1] ?? String(v);
  if (typeof v === 'string') return v || 'empty';
  return String(v);
}

/** Copy text: the async clipboard on a secure origin, else a hidden textarea and execCommand (plain-http hosts). */
async function copyText(text: string): Promise<boolean> {
  try {
    if (window.isSecureContext && navigator.clipboard) { await navigator.clipboard.writeText(text); return true; }
  } catch { /* fall through */ }
  try {
    const ta = document.createElement('textarea');
    ta.value = text;
    ta.setAttribute('readonly', '');
    ta.style.position = 'fixed';
    ta.style.top = '0';
    ta.style.opacity = '0';
    document.body.appendChild(ta);
    ta.select();
    const ok = document.execCommand('copy');
    ta.remove();
    return ok;
  } catch { return false; }
}

/** A popover menu on the Escape stack: Esc or an outside click closes it, focus returns to its anchor (S05a §6). */
function Popover({ id, label, onClose, className = '', children }: { id: string; label: string; onClose: () => void; className?: string; children: React.ReactNode }) {
  const ref = useRef<HTMLDivElement>(null);
  const close = useRef(onClose);
  close.current = onClose;
  useEffect(() => {
    const opener = document.activeElement as HTMLElement | null;
    const first = ref.current?.querySelector<HTMLElement>('button:not([disabled]), input:not([disabled])');
    first?.focus({ preventScroll: true });
    const off = pushEscape(() => close.current(), { id });
    const onDown = (e: MouseEvent) => {
      const t = e.target as Node;
      if (ref.current?.contains(t) || ref.current?.parentElement?.contains(t)) return;
      close.current();
    };
    document.addEventListener('mousedown', onDown);
    return () => {
      off();
      document.removeEventListener('mousedown', onDown);
      queueMicrotask(() => { if (opener?.isConnected) opener.focus({ preventScroll: true }); });
    };
  }, [id]);
  const onKey = (e: React.KeyboardEvent) => {
    if (e.key !== 'ArrowDown' && e.key !== 'ArrowUp') return;
    const items = Array.from(ref.current?.querySelectorAll<HTMLElement>('[role="menuitem"]:not([disabled]), [role="menuitemradio"]') ?? []);
    if (!items.length) return;
    e.preventDefault();
    const i = items.indexOf(document.activeElement as HTMLElement);
    items[(i + (e.key === 'ArrowDown' ? 1 : -1) + items.length) % items.length].focus();
  };
  return <div ref={ref} className={`ae-o-pop ${className}`} role="menu" aria-label={label} onKeyDown={onKey}>{children}</div>;
}

function MotionConsoleBody({ campaign, health, runs, readOnly = false, onClose, onConnect, onCampaign, onRuns, campaigns = [], onSwitchCampaign, onOpenSheet, initialAssetId = null }: ConsoleProps) {
  const { toast } = useToast();
  const notify = useCallback((text: string, tone: 'info' | 'ok' | 'err' = 'info') => { toast({ text, tone }); }, [toast]);
  const [initial] = useState(() => readStoredDraft(campaign?.id, readOnly));
  const [scriptPick, setScriptPick] = useState<string | null>(null);
  const approved = (campaign?.scripts ?? []).filter((s) => s.approvedVersion !== null);
  const firstScript = approved.find((s) => s.id === scriptPick) ?? approved[0];
  const firstVersion = firstScript?.versions.find((v) => v.n === firstScript.approvedVersion);
  const scriptKey = firstScript ? `${firstScript.id}:${firstScript.approvedVersion}` : null;
  const [c, setC] = useState<ConsoleControls>(() => initial?.c ?? DEFAULTS);
  const [locked, setLocked] = useState<Set<Part>>(() => new Set(initial?.locked ?? []));
  const [presets, setPresets] = useState<Preset[]>([]);
  const [presetsError, setPresetsError] = useState<string | null>(null);
  const [presetName, setPresetName] = useState('');
  const [replaceAsk, setReplaceAsk] = useState<Preset | null>(null);
  const [activePreset, setActivePreset] = useState<string | null>(initial?.activePreset ?? null);
  const [panel, setPanel] = useState<Panel>(null);
  const [promptFocus, setPromptFocus] = useState<Part | 'output' | null>(null);
  const [describe, setDescribe] = useState('');
  const [gate, setGate] = useState(false);
  const [gateError, setGateError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [assetIndex, setAssetIndex] = useState(0);
  const [capAsk, setCapAsk] = useState<string | null>(null);
  const [flash, setFlash] = useState(0);
  const [ring, setRing] = useState(0);
  const [, setTick] = useState(0);
  const [probeAsk, setProbeAsk] = useState(false);
  const [tab, setTab] = useState<Tab>(initial?.tab ?? 'scene');
  const [open, setOpen] = useState<Set<Part>>(() => new Set(initial?.open.length ? initial.open : ['action']));
  const [beatPick, setBeatPick] = useState<number | null>(initial?.beatPick ?? null);
  const [menu, setMenu] = useState<Menu>(null);
  const [abandonId, setAbandonId] = useState<string | null>(null);
  const [abandonError, setAbandonError] = useState<string | null>(null);
  const [switching, setSwitching] = useState(false);
  const [stageError, setStageError] = useState<string | null>(null);
  const [refChecks, setRefChecks] = useState<Record<string, RefCheck | 'checking' | 'error'>>({});
  const [focusRefs, setFocusRefs] = useState(0);
  // the campaign the state above belongs to; the draft is only written when it matches (no cross-campaign writes)
  const [stateFor, setStateFor] = useState<string | null>(campaign?.id ?? null);
  // the latest controls, so `set` validates against them synchronously instead of throwing inside a state updater
  const cRef = useRef(c);
  cRef.current = c;
  const seededKey = useRef<string | null>(initial?.scriptKey ?? null);
  const titleRef = useRef<HTMLSpanElement>(null);
  const pickerRef = useRef<HTMLSelectElement>(null);
  const refsRef = useRef<HTMLDivElement>(null);
  const promptRef = useRef<HTMLDivElement>(null);

  const loadPresets = useCallback(() => {
    setPresetsError(null);
    api.presets().then((r) => setPresets(r.presets)).catch((err) => setPresetsError(err instanceof Error ? err.message : 'failed'));
  }, []);
  useEffect(() => { loadPresets(); }, [loadPresets]);

  // focus the title on open; restore the previous focus on close; one ref-counted body scroll lock
  useEffect(() => {
    const prev = document.activeElement as HTMLElement | null;
    titleRef.current?.focus({ preventScroll: true });
    const unlock = lockBodyScroll();
    return () => { unlock(); queueMicrotask(() => { if (prev?.isConnected) prev.focus({ preventScroll: true }); }); };
  }, []);

  // Escape (X1): popovers and dialogs sit above this layer on the stack. The builder still runs its own
  // Escape chain (palette, node sheet, expanded builder), so this layer waits while one of those is open.
  const panelRef = useRef(panel);
  panelRef.current = panel;
  const closeRef = useRef(onClose);
  closeRef.current = onClose;
  useEffect(() => pushEscape(() => {
    if (document.querySelector('.ae-o-wf-palette, .ae-o-wf-sheet, .ae-o-graph.expanded')) return;
    if (panelRef.current) setPanel(null); else closeRef.current();
  }, { id: 'motion-console' }), []);

  // a campaign switch while open: load that campaign's draft (or defaults)
  useEffect(() => {
    const id = campaign?.id ?? null;
    if (id === stateFor) return;
    const d = readStoredDraft(id, readOnly);
    const next = d?.c ?? DEFAULTS;
    cRef.current = next;
    seededKey.current = d?.scriptKey ?? null;
    setC(next);
    setLocked(new Set(d?.locked ?? []));
    setOpen(new Set(d?.open.length ? d.open : ['action']));
    setBeatPick(d?.beatPick ?? null);
    setTab(d?.tab ?? 'scene');
    setActivePreset(d?.activePreset ?? null);
    setScriptPick(null);
    setAssetIndex(0);
    setStateFor(id);
  }, [campaign?.id, stateFor, readOnly]);

  // seed the dialogue from the approved script once per script version (a restored draft keeps its own)
  useEffect(() => {
    if (!firstVersion || !scriptKey || scriptKey === seededKey.current) return;
    seededKey.current = scriptKey;
    setBeatPick(null);
    if (cRef.current.dialogue.trim()) return;
    const { controls, trimmed } = clampReport({ dialogue: firstVersion.fullScript, duration: Math.min(15, Math.max(4, Math.ceil(firstVersion.wordCount / 2.5))) });
    setC((cur) => (cur.dialogue.trim() ? cur : mergeControls(cur, controls)));
    const note = trimNotice(trimmed);
    if (note) notify(note);
  }, [firstVersion, scriptKey, notify]);

  // draft persistence (FR-1, Q-MC-5: per browser): every change, per campaign; never in sample mode
  useEffect(() => {
    if (readOnly || stateFor !== (campaign?.id ?? null)) return;
    const draft: ConsoleDraft = { v: 1, c, locked: [...locked] as DraftPart[], open: [...open] as DraftPart[], beatPick, tab, activePreset, scriptKey: seededKey.current };
    try { window.localStorage.setItem(draftKey(stateFor), JSON.stringify(draft)); } catch { /* storage blocked: works, no persistence */ }
  }, [c, locked, open, beatPick, tab, activePreset, stateFor, campaign?.id, readOnly]);

  const liveRuns = runs.filter((r) => r.status === 'running' && (r.kind === 'motion' || r.kind === 'image'));
  const liveCount = liveRuns.length;
  useEffect(() => {
    if (!liveCount) return;
    const t = setInterval(() => setTick((n) => n + 1), 1000);
    return () => clearInterval(t);
  }, [liveCount]);

  // what the stage can show: delivered motion and image assets of this campaign
  const stageAssets = useMemo(() => (campaign?.assets ?? []).filter((a) => {
    const kind = campaign?.generations.find((g) => g.id === a.generationId)?.kind;
    return kind === 'motion' || kind === 'image';
  }), [campaign]);
  // the arrival pop fires only when the count grows for the SAME campaign after first paint, never on load or switch
  const assetCount = stageAssets.length;
  const seenAssets = useRef<{ id: string | null; n: number } | null>(null);
  useEffect(() => {
    const id = campaign?.id ?? null;
    const prev = seenAssets.current;
    if (prev && prev.id === id && assetCount > prev.n) { setFlash((f) => f + 1); setAssetIndex(assetCount - 1); }
    seenAssets.current = { id, n: assetCount };
  }, [assetCount, campaign?.id]);
  // G3: open on a given asset once it is loaded; an unknown id leaves the default
  const appliedAsset = useRef<string | null>(null);
  useEffect(() => {
    if (!initialAssetId || appliedAsset.current === initialAssetId) return;
    const i = stageAssets.findIndex((a) => a.id === initialAssetId);
    if (i < 0) return;
    appliedAsset.current = initialAssetId;
    setAssetIndex(i);
    setRing((n) => n + 1);
  }, [initialAssetId, stageAssets]);

  // never throws: an out-of-range value keeps the previous state and names the field's limit
  const set = useCallback(<K extends keyof ConsoleControls>(k: K, v: ConsoleControls[K]) => {
    const r = ConsoleControlsSchema.safeParse({ ...cRef.current, [k]: v });
    if (!r.success) { setError(controlsIssue(r.error)); return; }
    cRef.current = r.data;
    setC(r.data);
    setError((e) => (e && LIMIT_ERROR.test(e) ? null : e));
  }, []);

  // reference path checks (FR-10): debounced, stat only, each path once
  const refList = useMemo(() => referencePaths(c), [c]);
  const refSig = refList.map((r) => r.path).join('\n');
  useEffect(() => {
    if (readOnly) return;
    const todo = refList.map((r) => r.path).filter((p) => !(p in refChecks));
    if (!todo.length) return;
    const t = setTimeout(() => {
      setRefChecks((m) => ({ ...m, ...Object.fromEntries(todo.map((p) => [p, 'checking' as const])) }));
      for (const p of todo) {
        api.checkAsset(p)
          .then((r) => setRefChecks((m) => ({ ...m, [p]: { exists: r.exists, kind: r.kind } })))
          .catch(() => setRefChecks((m) => ({ ...m, [p]: 'error' })));
      }
    }, 400);
    return () => clearTimeout(t);
  }, [refSig, readOnly]);
  const resolvedChecks = useMemo(() => Object.fromEntries(Object.entries(refChecks).filter(([, v]) => typeof v === 'object')) as Record<string, RefCheck>, [refChecks]);

  const composed = useMemo(() => composePrompt(c), [c]);
  const lints = useMemo(() => validateRequest(c, resolvedChecks), [c, resolvedChecks]);
  const allLints = [...composed.lints, ...lints];
  const blocking = allLints.filter((l) => l.level === 'error');
  const rule = MODEL_RULES[c.model];
  const dRule = DURATION_RULES[c.model];
  const dialogueOn = !!c.dialogue.trim() && !c.omitDialogue;
  const dialogueFit = dialogueOn ? durationForScript(c.model, c.dialogue) : null;
  const modelHealth = health.arcads.health[c.model];
  const requests = c.variations;
  const asset = stageAssets.length ? stageAssets[Math.min(assetIndex, stageAssets.length - 1)] : null;
  useEffect(() => { setStageError(null); }, [asset?.id]);
  const running = liveRuns.map((r) => ({ ...r, elapsedMs: r.finishedAt ? r.elapsedMs : Math.max(r.elapsedMs, Date.now() - Date.parse(r.startedAt)) }));
  const delivered = runs.filter((r) => r.status === 'delivered' && (r.kind === 'motion' || r.kind === 'image')).length;
  const chips = chipWidths(health.arcads.sessionSpent, health.arcads.sessionCap);
  const capState: 'ok' | 'warn' | 'cap' = chips.used >= 100 ? 'cap' : chips.used >= 80 ? 'warn' : 'ok';
  const connected = health.arcads.configured;
  const block: BlockReason | null = generateBlockReason({ campaign: !!campaign, readOnly, connected, blocking: blocking.length, busy });
  const presetObj = presets.find((p) => p.name === activePreset) ?? null;
  const presetEdited = presetObj ? isPresetModified(presetObj.controls, c) : false;
  const sent = stripIgnoredRefs(c);
  const uploads = sent.referenceImages.length + sent.referenceVideos.length + sent.referenceAudios.length + (sent.startFrame ? 1 : 0) + (sent.endFrame ? 1 : 0);

  const isHidden = (part: Part) => c.omit.includes(part as never);
  const toggleOmit = (part: Part | null) => { if (!part || part === 'constraints') return; set('omit', isHidden(part) ? c.omit.filter((x) => x !== part) : [...c.omit, part as never]); };
  const toggleLock = (part: Part | null) => { if (!part) return; setLocked((s) => { const n = new Set(s); if (n.has(part)) n.delete(part); else n.add(part); return n; }); };
  const toggleOpen = (part: Part) => setOpen((s) => { const n = new Set(s); if (n.has(part)) n.delete(part); else n.add(part); return n; });
  const scriptSeconds = firstVersion ? firstVersion.beats.reduce((n, b) => n + b.seconds, 0) : 0;
  // a beat row is a real selection: the dialogue and the duration follow it; the whole-script row restores them
  const pickBeat = (i: number | null) => {
    if (!firstVersion) return;
    const beat = i === null ? null : firstVersion.beats[i];
    const seconds = beat ? beat.seconds : scriptSeconds;
    const duration = dRule.kind === 'range' ? Math.min(dRule.max, Math.max(dRule.min, Math.round(seconds))) : dRule.kind === 'enum' ? dRule.values.reduce((best, v) => (Math.abs(v - seconds) < Math.abs(best - seconds) ? v : best), dRule.values[0]) : c.duration;
    const { controls, trimmed } = clampReport({ dialogue: beat ? beat.spoken : firstVersion.fullScript, ...(beat ? { action: beat.action } : {}), duration });
    setC((cur) => mergeControls(cur, controls));
    setBeatPick(i);
    notify(trimNotice(trimmed) ?? (beat ? `${beat.name}: dialogue and duration follow this beat.` : 'Whole script: dialogue and duration restored.'));
  };
  const applyPreset = (p: Preset) => { setC((cur) => mergeControls(cur, p.controls)); setActivePreset(p.name); };

  // P7 caret reset: back to the applied preset's value, else the default
  const preset = presetObj?.controls ?? null;
  const reset = <K extends keyof ConsoleControls>(key: K): ResetProps => ({
    modified: isModified(key, c, DEFAULTS, preset),
    resetTo: valueText(key, defaultFor(key, DEFAULTS, preset)),
    onReset: () => set(key, defaultFor(key, DEFAULTS, preset)),
  });
  const resetColumn = (col: Column) => {
    const patch: Partial<ConsoleControls> = {};
    for (const k of COLUMN_KEYS[col]) (patch as Record<string, unknown>)[k] = defaultFor(k, DEFAULTS, preset);
    setC((cur) => mergeControls(cur, patch));
    notify(`${COLUMN_NAME[col]} reset to ${presetObj ? presetObj.name : 'defaults'}.`, 'ok');
  };
  const seedDialogue = (): Partial<ConsoleControls> => (firstVersion ? clampControls({ dialogue: firstVersion.fullScript }) : {});
  const resetAll = () => {
    const next = mergeControls(DEFAULTS, seedDialogue());
    cRef.current = next;
    setC(next);
    setLocked(new Set());
    setOpen(new Set(['action']));
    setBeatPick(null);
    setActivePreset(null);
    setTab('scene');
    if (!readOnly) clearStoredDraft(campaign?.id);
    notify('Every control is back to its default.', 'ok');
  };
  const resetToPreset = () => {
    if (!presetObj) return;
    setC((cur) => mergeControls(mergeControls(DEFAULTS, presetObj.controls), { dialogue: cur.dialogue, omitDialogue: cur.omitDialogue, referenceImages: cur.referenceImages, referenceVideos: cur.referenceVideos, referenceAudios: cur.referenceAudios, startFrame: cur.startFrame, endFrame: cur.endFrame }));
    notify(`Back to ${presetObj.name}.`, 'ok');
  };

  const applyDescribe = () => {
    if (!describe.trim()) return;
    const t = describe.toLowerCase();
    const next: Partial<ConsoleControls> = {};
    const want = (part: Part) => !locked.has(part);
    if (want('camera')) { for (const f of FRAMINGS) if (t.includes(f)) next.framing = f; for (const m of MOVEMENTS) if (t.includes(m)) next.movement = m; for (const h of HEIGHTS) if (t.includes(h)) next.height = h; if (/slow/.test(t)) next.intensity = 1; if (/fast|quick/.test(t)) next.intensity = 4; }
    if (want('style')) { for (const s of SETTINGS) if (t.includes(s)) next.setting = s; for (const l of LIGHTING) if (t.includes(l.split(' ')[0])) next.lighting = l; if (/selfie|phone|handheld/.test(t)) next.style = 'handheld phone'; if (/documentary/.test(t)) next.style = 'documentary'; if (/commercial|polished/.test(t)) next.style = 'commercial'; }
    const m = t.match(/(\d{1,2})\s*s(ec|econds)?\b/); if (m) next.duration = Number(m[1]);
    for (const v of VIDEO_MODELS) if (t.includes(v.split('-')[0])) next.model = v;
    if (Object.keys(next).length === 0 && want('subject')) next.subject = describe.trim().slice(0, CONSOLE_LIMITS.subject);
    setC((cur) => mergeControls(cur, next));
    notify(Object.keys(next).length ? `Set ${Object.keys(next).join(', ')}.` : 'Nothing recognised.');
  };

  const refreshRuns = async () => { const rr = await api.runs(campaign?.id ?? null).catch(() => null); if (rr) onRuns(rr.runs); };
  const submit = async (override = false) => {
    if (block) return;
    if (!campaign) return;
    setBusy(true); setGateError(null); setCapAsk(null);
    try {
      const res = await api.motion({ campaignId: campaign.id, controls: c, scriptId: firstScript?.id ?? null, override });
      const ok = res.submitted.length;
      if (res.failed.length) notify(`${ok} of ${ok + res.failed.length} sent. ${res.failed[0].reason}`, 'err');
      else notify(`${plural(ok, 'request')} sent to Arcads.`, 'ok');
      setGate(false);
      await refreshRuns();
      onCampaign((await api.campaign(campaign.id)).campaign);
    } catch (err) {
      if (err instanceof ApiError && err.kind === 'cap') setCapAsk(err.message);
      else setGateError(err instanceof Error ? err.message : 'Submit failed.');
    } finally { setBusy(false); }
  };
  const abandon = async () => {
    const r = running.find((x) => x.id === abandonId) ?? running[0];
    if (!r || readOnly) return;
    setAbandonError(null);
    try {
      await api.abandon(r.id);
      notify(`Abandoned ${r.title}.`, 'ok');
      setMenu(null);
      await refreshRuns();
    } catch (err) { setAbandonError(err instanceof Error ? err.message : 'Abandon failed.'); }
  };
  const copyPrompt = async () => {
    if (await copyText(composed.prompt)) { notify('Prompt copied.', 'ok'); return; }
    setPromptFocus(null);
    setPanel('prompt');
    notify('Copy blocked. The prompt is open; select it with Cmd+A.', 'err');
    requestAnimationFrame(() => {
      const el = promptRef.current; if (!el) return;
      const range = document.createRange(); range.selectNodeContents(el);
      const sel = window.getSelection(); sel?.removeAllRanges(); sel?.addRange(range);
    });
  };
  const savePreset = async (replace?: Preset) => {
    const name = presetName.trim();
    if (!name || readOnly) return;
    const clash = presets.find((p) => p.name.toLowerCase() === name.toLowerCase());
    if (clash?.builtin) { notify("Built-in presets can't be replaced. Pick another name.", 'err'); return; }
    if (clash && !replace) { setReplaceAsk(clash); return; }
    try {
      if (replace) await api.deletePreset(replace.id);
      const r = await api.savePreset(name, c);
      setPresets((p) => [{ id: r.preset.id, name, builtin: false, controls: c }, ...p.filter((x) => x.id !== replace?.id)]);
      setActivePreset(name);
      setReplaceAsk(null);
      notify(`Preset “${name}” ${replace ? 'replaced' : 'saved'}.`, 'ok');
      setPresetName('');
    } catch (err) { notify(err instanceof Error ? err.message : 'Save failed.', 'err'); }
  };
  const deletePreset = async (p: Preset) => {
    if (p.builtin || readOnly) return;
    try {
      await api.deletePreset(p.id);
      setPresets((list) => list.filter((x) => x.id !== p.id));
      if (activePreset === p.name) setActivePreset(null);
      notify(`Preset “${p.name}” deleted.`, 'ok');
    } catch (err) { notify(err instanceof Error ? err.message : 'Delete failed.', 'err'); }
  };
  const probeThisModel = async (override = false) => {
    if (readOnly) return;
    setBusy(true); setGateError(null);
    try {
      const r = await api.probeModel(c.model, override);
      setProbeAsk(false); setCapAsk(null);
      notify(`Probe for ${c.model} submitted as ${r.runId}. Its verdict lands in Credits & health.`, 'ok');
      await refreshRuns();
    } catch (err) {
      if (err instanceof ApiError && err.kind === 'cap') setCapAsk(err.message);
      else setGateError(err instanceof Error ? err.message : 'Probe failed.');
    } finally { setBusy(false); }
  };

  const openSheet = (sheet: ConsoleSheet, opts?: { runsKind?: 'motion' }) => {
    if (sheet === 'connect') { onConnect(); return; }
    onOpenSheet?.(sheet, opts);
  };
  const openCampaignPicker = () => {
    setGate(false);
    if (readOnly) return;
    if (campaigns.length === 0) { openSheet('generator'); return; }
    const el = pickerRef.current; if (!el) return;
    el.focus();
    try { (el as HTMLSelectElement & { showPicker?: () => void }).showPicker?.(); } catch { /* focus is enough */ }
  };
  const fixBlock = (action: BlockReason['action']) => {
    if (action === 'campaign') openCampaignPicker();
    else if (action === 'lints') { setGate(false); setPanel('lints'); }
    else if (action === 'connect') { setGate(false); onConnect(); }
  };
  const switchCampaign = async (id: string) => {
    if (!onSwitchCampaign || id === campaign?.id) return;
    setSwitching(true);
    try { await onSwitchCampaign(id); } finally { setSwitching(false); }
  };
  const goRefs = () => { setTab('references'); setFocusRefs((n) => n + 1); };
  useEffect(() => {
    if (!focusRefs) return;
    refsRef.current?.querySelector<HTMLElement>('textarea:not([disabled]), input:not([disabled])')?.focus();
  }, [focusRefs]);
  const showPrompt = (focus: Part | 'output' | null) => { setPromptFocus(focus); setPanel('prompt'); };
  const step = (d: 1 | -1) => { if (stageAssets.length < 2) return; setAssetIndex((i) => (i + d + stageAssets.length) % stageAssets.length); };

  // keyboard shortcuts (FR-3): never while typing, never under a dialog or menu, never inside the builder
  const shortcutRef = useRef<(a: ReturnType<typeof shortcutFor>) => void>(() => {});
  shortcutRef.current = (a) => {
    if (a === 'prev') step(-1);
    else if (a === 'next') step(1);
    else if (a === 'generate') { setGateError(null); setGate(true); }
    else if (a === 'prompt') { setPromptFocus(null); setPanel((p) => (p === 'prompt' ? null : 'prompt')); }
    else if (a === 'lints') setPanel((p) => (p === 'lints' ? null : 'lints'));
    else if (a === 'dialogue') setPanel((p) => (p === 'dialogue' ? null : 'dialogue'));
    else if (a === 'describe') setPanel((p) => (p === 'describe' ? null : 'describe'));
  };
  const layered = gate || probeAsk || menu !== null;
  useEffect(() => {
    if (layered) return;
    const onKey = (e: KeyboardEvent) => {
      const t = e.target as HTMLElement | null;
      if (t?.closest('.ae-o-graph, .ae-o-gate, [role="dialog"], [role="menu"]')) return;
      const a = shortcutFor(e, t?.tagName ?? null, !!t?.isContentEditable);
      if (!a) return;
      e.preventDefault();
      shortcutRef.current(a);
    };
    window.addEventListener('keydown', onKey);
    return () => window.removeEventListener('keydown', onKey);
  }, [layered]);

  const toggle = (key: Exclude<Panel, null>) => { if (key === 'prompt') setPromptFocus(null); setPanel(panel === key ? null : key); };
  const tool = (key: Exclude<Panel, null>, icon: React.ReactNode, title: string, small = false, badge?: number) => (
    <OBtn kind={small ? 'icon-sm' : 'icon'} cls={`ae-o-icon ${small ? 'sm' : ''}`}  data-on={panel === key} aria-pressed={panel === key} title={title} aria-label={title} onClick={() => toggle(key)}>{icon}{badge ? <span className="ae-o-icon-badge" aria-hidden>{badge}</span> : null}</OBtn>
  );
  const refsTool = (icon: React.ReactNode, small = false) => (
    <OBtn kind={small ? 'icon-sm' : 'icon'} cls={`ae-o-icon ${small ? 'sm' : ''}`}  data-on={tab === 'references'} aria-pressed={tab === 'references'} title="References: open the Refs tab" aria-label="References" onClick={goRefs}>{icon}</OBtn>
  );

  const statusText = running.length ? `Generating ${running.length}` : health.dry ? 'Dry run' : 'Ready';
  const modeWord = readOnly ? 'Sample' : !connected ? 'Not configured' : health.dry ? 'Dry run' : 'Live';
  const presetLabel = activePreset ? `${activePreset}${presetEdited ? ' · edited' : ''}` : 'Custom';
  const runLine = running.length
    ? `${running.length} running · ${fmtMs(running[0].elapsedMs)} · ${titleCaseStatus(running[0].arcads?.lastStatus) || 'Waiting'}${running.length > 1 ? ` · +${running.length - 1}` : ''}`
    : delivered ? `${delivered} delivered` : 'No requests yet';
  const balance = health.arcads.balance?.creditsRemaining ?? null;
  const creditLine = balance != null
    ? `${fmtCredits(balance)} left · session ${fmtCredits(health.arcads.sessionSpent)} / ${fmtCredits(health.arcads.sessionCap)}`
    : `session ${fmtCredits(health.arcads.sessionSpent)} / ${fmtCredits(health.arcads.sessionCap)} cap`;
  const ringPct = running.length ? Math.max(4, Math.min(100, (running[0].elapsedMs / running[0].timeoutMs) * 100)) : 0;
  const ringTitle = running.length ? `${running[0].title} · ${fmtMs(running[0].elapsedMs)} · ${titleCaseStatus(running[0].arcads?.lastStatus) || 'Waiting'}` : 'Nothing running';
  const assetsWord = plural(stageAssets.length, 'asset');
  const words = plural(composed.wordCount, 'word');
  const lintWord = plural(allLints.length, 'lint');
  const abandonTarget = running.find((x) => x.id === abandonId) ?? running[0] ?? null;
  const scriptTitle = (s: (typeof approved)[number]) => `${campaign?.angles.find((x) => x.id === s.angleId)?.title ?? 'Script'} v${s.approvedVersion}`;
  const blockLabel = (a: BlockReason['action'], inGate: boolean) => a === 'campaign' ? (campaigns.length ? (inGate ? 'Open campaigns' : 'Pick one') : 'Create one') : a === 'lints' ? 'Show lints' : a === 'connect' ? 'Connect Arcads' : undefined;
  const sortedCampaigns = [...campaigns].sort((a, b) => b.updatedAt.localeCompare(a.updatedAt));
  const refCount = (n: number, max: number) => <em className="not-italic" style={{ color: n > max && max > 0 ? 'var(--o-red-text)' : 'var(--o-dim)' }}>{max ? (n ? `${n} of ${max}` : `up to ${max}`) : n ? `${n} kept, not sent` : 'none for this model'}</em>;
  const refStatus = (paths: string[]) => paths.length > 0 && (
    <ul className="ae-o-refstat">
      {paths.map((p) => {
        const k = refChecks[p];
        const text = k === undefined || k === 'checking' ? 'checking' : k === 'error' ? 'check failed' : !k.exists ? 'missing' : k.kind ?? 'unknown type';
        const tone = typeof k === 'object' ? (k.exists ? 'ok' : 'bad') : k === 'error' ? 'bad' : 'wait';
        return <li key={p} data-tone={tone} title={p}><span className="truncate">{p.split('/').pop()}</span><span>{text}</span></li>;
      })}
    </ul>
  );
  const refField = (field: 'referenceImages' | 'referenceVideos' | 'referenceAudios', label: string, max: number, rows: number, ph: string) => (
    <>
      <div className="mb-1 mt-2 flex justify-between gap-2"><span>{label}</span>{refCount(c[field].length, max)}</div>
      <OTextarea cls="ae-o-text w-full resize-none" keep="w-full resize-none" value={c[field].join('\n')} onChange={(e) => set(field, e.target.value.split('\n').map((s) => s.trim()).filter(Boolean))} rows={rows} placeholder={max ? ph : `${c.model} takes no ${label.toLowerCase()}`} disabled={readOnly || (max === 0 && c[field].length === 0)} title={readOnly ? SAMPLE_OFF : max === 0 ? `${c.model} takes no ${label.toLowerCase()}` : undefined}  aria-label={`${label} reference paths`} />
      {refStatus(c[field])}
    </>
  );

  return (
    <>
      <div className="ae-o-window flex min-h-full w-full flex-col">
        {/* top bar */}
        <div className="ae-o-top">
          <span className="ae-o-mark" aria-hidden><Film className="h-4 w-4" strokeWidth={2.2} /></span>
          <span ref={titleRef} tabIndex={-1} className="ae-o-title" id="ae-console-title">Motion console</span>
          <span className="relative inline-flex">
            <OBtn kind="icon-sm" cls="ae-o-icon sm"  title="Console menu" aria-label="Console menu" aria-haspopup="menu" aria-expanded={menu === 'console'} data-on={menu === 'console'} onClick={() => setMenu(menu === 'console' ? null : 'console')}><ChevronDown className="h-3.5 w-3.5" strokeWidth={2} /></OBtn>
            {menu === 'console' && (
              <Popover id="console-menu" label="Console menu" onClose={() => setMenu(null)} className="left-0 top-[36px]">
                <OBtn kind="row" role="menuitem" onClick={() => { setMenu(null); resetAll(); }}><RotateCcw className="h-3.5 w-3.5" strokeWidth={2} />Reset all controls to defaults</OBtn>
                <OBtn kind="row" role="menuitem" disabled={!presetObj} title={presetObj ? undefined : 'No preset applied'} onClick={() => { setMenu(null); resetToPreset(); }}><Layers className="h-3.5 w-3.5" strokeWidth={2} />{presetObj ? `Reset to ${presetObj.name}` : 'Reset to preset'}</OBtn>
                <OBtn kind="row" role="menuitem" onClick={() => { setMenu(null); openSheet('runs'); }}><Activity className="h-3.5 w-3.5" strokeWidth={2} />Open Runs</OBtn>
                <OBtn kind="row" role="menuitem" onClick={() => { setMenu(null); openSheet('credits'); }}><Settings2 className="h-3.5 w-3.5" strokeWidth={2} />Open Credits & health</OBtn>
                <OBtn kind="row" role="menuitem" onClick={() => { setMenu(null); setPanel('shortcuts'); }}><Keyboard className="h-3.5 w-3.5" strokeWidth={2} />Keyboard shortcuts</OBtn>
              </Popover>
            )}
          </span>
          <OBtn kind="pill" cls="ae-o-pill"  title={readOnly ? 'Sample data: the bridge is not used' : connected ? `Arcads credentials present on this host (${modeWord}). Open Credits & health` : 'No Arcads credentials yet. Connect Arcads'} onClick={() => (connected ? openSheet('credits') : onConnect())}>
            <Activity className="h-3.5 w-3.5 text-[var(--o-muted)]" strokeWidth={1.8} />Arcads bridge<span className={`ae-o-badge ${connected ? '' : 'off'}`}>{connected ? <Check className="h-2.5 w-2.5" strokeWidth={3.5} /> : <X className="h-2.5 w-2.5" strokeWidth={3} />}</span><span className="ae-o-mode" data-mode={readOnly ? 'sample' : !connected ? 'off' : health.dry ? 'dry' : 'live'}>{modeWord}</span>
          </OBtn>
          {readOnly ? (
            <OBtn kind="pill" cls="ae-o-pill center"  disabled title="Sample campaign: switching is off in the sample">{campaign ? campaign.name.slice(0, 60) : 'Sample campaign'}</OBtn>
          ) : campaigns.length === 0 ? (
            <OBtn kind="pill" cls="ae-o-pill center"  title="No campaign yet. Create one in the Creative generator" onClick={() => openSheet('generator')}>No campaign · Create one</OBtn>
          ) : (
            <OSelect cls="ae-o-pill center ae-o-picker" ref={pickerRef}  value={switching ? '__loading' : campaign?.id ?? ''} disabled={switching} onChange={(e) => void switchCampaign(e.target.value)} aria-label="Campaign" title="Switch campaign">
              {switching && <option value="__loading">Loading…</option>}
              {!campaign && <option value="">No campaign</option>}
              {sortedCampaigns.map((s) => (<option key={s.id} value={s.id}>{s.name.slice(0, 60)}</option>))}
            </OSelect>
          )}
          <span className="ml-auto flex items-center gap-3">
            <OBtn kind="pill" cls="ae-o-pill dark"  title={activePreset ? `Preset ${activePreset}${presetEdited ? ', edited since applied' : ''}. Open presets` : 'No preset applied. Open presets'} data-on={panel === 'presets'} onClick={() => toggle('presets')}><Layers className="h-4 w-4 text-[var(--o-muted)]" strokeWidth={1.7} /><span className="ae-o-ellipsis">{presetLabel}</span></OBtn>
            <span className="ae-o-label-strong">Arcads status</span>
            {connected
              ? <OBtn kind="pill" cls="ae-o-pill green"  title={`${statusText}. Open Runs`} onClick={() => openSheet('runs')}><span className={`ae-o-dot ${running.length ? 'live' : ''}`} style={{ background: health.dry && !running.length ? 'var(--o-dim)' : 'var(--o-green)' }} />{statusText}</OBtn>
              : <OBtn kind="pill" cls="ae-o-pill"  title="Connect Arcads" onClick={onConnect}><span className="ae-o-dot" style={{ background: 'var(--o-dim)' }} />Connect Arcads</OBtn>}
            <OBtn kind="icon-sm" cls="ae-o-icon sm"  title="Engine settings: session cap, UGC path, new session" aria-label="Engine settings" onClick={() => openSheet('credits')}><Settings2 className="h-4 w-4" strokeWidth={1.8} /></OBtn>
            <OBtn kind="icon-sm" cls="ae-o-icon sm"  title="Exit (Esc). Your controls are kept" aria-label="Exit" onClick={onClose}><X className="h-4 w-4" strokeWidth={2} /></OBtn>
          </span>
        </div>

        {/* tool strip · transport · readout */}
        <div className="ae-o-tools">
          <span className="ae-o-strip">
            {tool('presets', <Layers className="h-[18px] w-[18px]" strokeWidth={1.8} />, 'Presets')}
            {refsTool(<Images className="h-[18px] w-[18px]" strokeWidth={1.8} />)}
            {tool('dialogue', <MessageSquare className="h-[18px] w-[18px]" strokeWidth={1.8} />, 'Dialogue')}
            {tool('lints', <AlertTriangle className="h-[18px] w-[18px]" strokeWidth={1.8} />, blocking.length ? `${lintWord}, ${blocking.length} blocking` : lintWord, false, blocking.length)}
            {tool('describe', <Wand2 className="h-[18px] w-[18px]" strokeWidth={1.8} />, 'Describe it')}
            {tool('prompt', <FileText className="h-[18px] w-[18px]" strokeWidth={1.8} />, 'Composed prompt')}
            <OBtn kind="icon" cls="ae-o-icon"  title="Copy the prompt" aria-label="Copy the prompt" onClick={copyPrompt}><Copy className="h-[18px] w-[18px]" strokeWidth={1.8} /></OBtn>
            <OBtn kind="icon" cls="ae-o-icon"  title={readOnly ? SAMPLE_OFF : !connected ? 'Connect Arcads first' : `Probe ${c.model} with one cheapest request (charges credits)`} aria-label={`Probe ${c.model}`} disabled={!connected || readOnly} data-on={probeAsk} onClick={() => { setGateError(null); setCapAsk(null); setProbeAsk(true); }}><Activity className="h-[18px] w-[18px]" strokeWidth={1.8} /></OBtn>
          </span>
          <span className="ae-o-strip">
            <OBtn kind="icon" cls="ae-o-icon"  title={stageAssets.length < 2 ? (stageAssets.length ? '1 motion asset: nothing to step to' : 'No motion assets yet') : 'Previous asset ( [ )'} aria-label="Previous asset" disabled={stageAssets.length < 2} onClick={() => step(-1)}><SkipBack className="h-[18px] w-[18px]" strokeWidth={1.8} /></OBtn>
            <span className="relative inline-flex">
              <OBtn kind="icon" cls="ae-o-icon"  title={readOnly ? SAMPLE_OFF : running.length ? 'Abandon a running job' : 'Nothing running'} aria-label="Abandon a running job" aria-haspopup="menu" aria-expanded={menu === 'abandon'} disabled={running.length === 0 || readOnly} data-on={menu === 'abandon'} onClick={() => { setAbandonError(null); setAbandonId(running[0]?.id ?? null); setMenu(menu === 'abandon' ? null : 'abandon'); }}><Square className="h-[16px] w-[16px]" strokeWidth={1.8} /></OBtn>
              {menu === 'abandon' && abandonTarget && (
                <Popover id="abandon-confirm" label="Abandon a running job" onClose={() => setMenu(null)} className="left-0 top-[40px] w-[300px]">
                  {running.length > 1 && (
                    <div role="radiogroup" aria-label="Running jobs" className="ae-o-pop-list">
                      {running.map((r) => (
                        <OBtn kind="row" key={r.id} role="menuitemradio" aria-checked={abandonTarget.id === r.id} onClick={() => setAbandonId(r.id)}><span className="ae-o-radio" data-on={abandonTarget.id === r.id} /><span className="min-w-0 flex-1 truncate">{r.title}</span><span className="text-[var(--o-dim)]">{fmtMs(r.elapsedMs)}</span></OBtn>
                      ))}
                    </div>
                  )}
                  <p className="ae-o-pop-text">Abandon <b>{abandonTarget.title}</b>? Arcads already charged; this stops tracking it.</p>
                  {abandonError && <p className="ae-o-pop-text" style={{ color: 'var(--o-red-text)' }}>{abandonError}</p>}
                  <div className="ae-o-pop-actions">
                    <OBtn kind="pill" cls="ae-o-pill" role="menuitem"  onClick={() => setMenu(null)}>Keep</OBtn>
                    <OBtn kind="danger" cls="ae-o-pill ae-o-danger" role="menuitem"  onClick={() => void abandon()}>Abandon</OBtn>
                  </div>
                </Popover>
              )}
            </span>
            {/* the elapsed ring of the running job (display only; Generate lives in the parameters, Q-MC-1) */}
            <span className="ae-o-ringslot" data-display-only="run progress" title={ringTitle} aria-label={ringTitle} role="img">
              <svg viewBox="0 0 20 20" width="18" height="18" aria-hidden><circle cx="10" cy="10" r="8" className="ae-o-ring-track" /><circle cx="10" cy="10" r="8" className="ae-o-ring-fill" data-live={running.length > 0} pathLength={100} strokeDasharray={`${ringPct} 100`} transform="rotate(-90 10 10)" /></svg>
            </span>
            <OBtn kind="icon" cls="ae-o-icon"  title={block ? `Cost preview: ${block.text}` : 'Cost preview: what Generate will send and charge'} aria-label="Cost preview" onClick={() => { setGateError(null); setGate(true); }}><Target className="h-[18px] w-[18px]" strokeWidth={1.8} /></OBtn>
            <OBtn kind="icon" cls="ae-o-icon"  title={stageAssets.length < 2 ? (stageAssets.length ? '1 motion asset: nothing to step to' : 'No motion assets yet') : 'Next asset ( ] )'} aria-label="Next asset" disabled={stageAssets.length < 2} onClick={() => step(1)}><SkipForward className="h-[18px] w-[18px]" strokeWidth={1.8} /></OBtn>
          </span>
          <span className="ae-o-readout">
            <OBtn kind="row" cls="ae-o-readline" keep="ae-o-readline"  title="Open Runs" onClick={() => openSheet('runs')}>
              <span className="ae-tick" data-mono-ok>{runLine}</span>
              <span className="ae-o-chips" data-idle={!running.length}>{running.length > 0 && <span className="ae-o-chip pink live" style={{ width: `${Math.max(12, Math.min(100, (running[0].elapsedMs / running[0].timeoutMs) * 100))}%` }} />}</span>
            </OBtn>
            <OBtn kind="row" cls="ae-o-readline" keep="ae-o-readline"  title={`Session spend ${fmtCredits(health.arcads.sessionSpent)} of ${fmtCredits(health.arcads.sessionCap)} cap. Open Credits & health`} onClick={() => openSheet('credits')}>
              <span data-mono-ok>{creditLine}</span>
              <span className="ae-o-chips" data-cap={capState}>
                {chips.used > 0 && <span className="ae-o-chip pink" data-chip="used" style={{ flexBasis: `calc((100% - ${chips.left > 0 ? 6 : 0}px) * ${chips.used / 100})` }} />}
                {chips.left > 0 && <span className="ae-o-chip green" data-chip="left" style={{ flexBasis: `calc((100% - ${chips.used > 0 ? 6 : 0}px) * ${chips.left / 100})` }} />}
              </span>
            </OBtn>
          </span>
        </div>

        {/* body */}
        <div className="ae-o-body">
          {/* scene tree */}
          <div className="ae-o-panel flex flex-col">
            <div className="ae-o-tabs" role="tablist" aria-label="Scene tree">
              {TABS.map((t, i) => (
                <OBtn kind="row" cls="ae-o-tab" keep="ae-o-tab" key={t.key} role="tab" id={`ae-o-tab-${t.key}`} aria-selected={tab === t.key} tabIndex={tab === t.key ? 0 : -1}  data-on={tab === t.key} onClick={() => setTab(t.key)}
                  onKeyDown={(e) => {
                    if (e.key !== 'ArrowRight' && e.key !== 'ArrowLeft') return;
                    e.preventDefault();
                    const n = TABS[(i + (e.key === 'ArrowRight' ? 1 : -1) + TABS.length) % TABS.length];
                    setTab(n.key);
                    requestAnimationFrame(() => document.getElementById(`ae-o-tab-${n.key}`)?.focus());
                  }}>{t.icon}{t.label}</OBtn>
              ))}
            </div>
            {tab === 'scene' && (
              <div className="ae-o-tree" role="tabpanel" aria-labelledby="ae-o-tab-scene">
                {PART_ORDER.map((part) => (
                  <div key={part}>
                    <div className="ae-o-tree-row" style={{ opacity: isHidden(part) ? 0.45 : 1 }}>
                      <OBtn kind="row" cls="ae-o-knot" keep="ae-o-knot"  data-open={open.has(part)} aria-expanded={open.has(part)} title={open.has(part) ? `Collapse ${PART_META[part].label}` : `Expand ${PART_META[part].label}`} onClick={() => toggleOpen(part)}><i /></OBtn>
                      <span style={{ color: PART_META[part].colour }}>{PART_META[part].icon}</span>
                      <OBtn kind="row" cls="truncate text-left" keep="truncate text-left"  aria-expanded={open.has(part)} title={open.has(part) ? 'Collapse' : part === 'action' ? 'Show the beats of the approved script' : 'Show what this part puts in the prompt'} onClick={() => toggleOpen(part)}>{PART_META[part].label}</OBtn>
                      <span className="ae-o-tools-mini">
                        <OBtn kind="icon-sm" title={part === 'constraints' ? 'Constraints always apply' : isHidden(part) ? 'Include in the prompt' : 'Hide from the prompt'} aria-label={`${isHidden(part) ? 'Include' : 'Hide'} ${PART_META[part].label}`} disabled={part === 'constraints'} data-on={!isHidden(part)} onClick={() => toggleOmit(part)}>{isHidden(part) ? <EyeOff className="h-3.5 w-3.5" strokeWidth={1.8} /> : <Eye className="h-3.5 w-3.5" strokeWidth={1.8} />}</OBtn>
                        <OBtn kind="icon-sm" title={locked.has(part) ? 'Unlock: Describe it may change this part' : 'Protect: Describe it will not change this part'} aria-label={`${locked.has(part) ? 'Unlock' : 'Lock'} ${PART_META[part].label}`} aria-pressed={locked.has(part)} data-on={locked.has(part)} onClick={() => toggleLock(part)}>{locked.has(part) ? <Lock className="h-3.5 w-3.5" strokeWidth={1.8} /> : <LockOpen className="h-3.5 w-3.5" strokeWidth={1.8} />}</OBtn>
                      </span>
                    </div>
                    {open.has(part) && part === 'action' && firstVersion && (<>
                      <OBtn kind="row" cls="ae-o-tree-row w-full text-left" keep="ae-o-tree-row w-full text-left"  data-sub="true" data-on={beatPick === null} title="Dialogue and duration follow the whole approved script" onClick={() => pickBeat(null)}><span /><span className="text-[var(--o-dim)]"><FileText className="h-3.5 w-3.5" strokeWidth={1.6} /></span><span className="truncate">Whole script · {scriptSeconds}s</span><span /></OBtn>
                      {firstVersion.beats.map((b, i) => (
                        <OBtn kind="row" cls="ae-o-tree-row w-full text-left" keep="ae-o-tree-row w-full text-left" key={i}  data-sub="true" data-on={beatPick === i} title={b.spoken ? `${b.spoken.slice(0, 140)}${b.spoken.length > 140 ? '…' : ''}` : 'No spoken line in this beat'} onClick={() => pickBeat(i)}><span /><span className="text-[var(--o-dim)]"><FileText className="h-3.5 w-3.5" strokeWidth={1.6} /></span><span className="truncate">{b.name} · {b.seconds}s</span><span /></OBtn>
                      ))}
                    </>)}
                    {open.has(part) && (part !== 'action' || !firstVersion) && (
                      <OBtn kind="row" cls="ae-o-tree-note" keep="ae-o-tree-note"  title={`${composed.parts[part] || 'Nothing composed for this part yet.'}\n\nClick to see it in the composed prompt`} onClick={() => showPrompt(part)}>
                        <span>{part === 'action' && !firstVersion ? composed.parts.action || 'No approved script yet. Approve one in the Ad scripter to pick its beats here.' : composed.parts[part] || 'Nothing composed for this part yet.'}</span>
                      </OBtn>
                    )}
                  </div>
                ))}
              </div>
            )}
            {tab === 'presets' && (
              <div className="ae-o-tree" role="tabpanel" aria-labelledby="ae-o-tab-presets">
                {presetsError && <div className="ae-o-inline-err">Couldn&apos;t load presets · <OBtn kind="link" cls="ae-o-linkbtn"  onClick={loadPresets}>Retry</OBtn></div>}
                {presets.map((p) => (
                  <div key={p.id} className="ae-o-preset-row">
                    <OBtn kind="row" cls="ae-o-tree-row w-full text-left" keep="ae-o-tree-row w-full text-left"  data-sub="true" data-on={activePreset === p.name} title={`Apply ${p.name}`} onClick={() => applyPreset(p)}><span /><span className="text-[var(--o-dim)]"><Layers className="h-3.5 w-3.5" strokeWidth={1.6} /></span><span className="truncate">{p.name}</span><span className="text-[var(--o-dim)]">{p.builtin ? 'built in' : 'saved'}</span></OBtn>
                    {!p.builtin && <OBtn kind="row" cls="ae-o-preset-del" keep="ae-o-preset-del"  disabled={readOnly} title={readOnly ? SAMPLE_OFF : `Delete preset ${p.name}`} aria-label={`Delete preset ${p.name}`} onClick={() => void deletePreset(p)}><Trash2 className="h-3.5 w-3.5" strokeWidth={1.8} /></OBtn>}
                  </div>
                ))}
                <div className="mt-2 flex gap-1.5 px-2">
                  <OInput cls="ae-o-text min-w-0 flex-1" keep="min-w-0 flex-1" value={presetName} maxLength={CONSOLE_LIMITS.presetName} disabled={readOnly} title={readOnly ? SAMPLE_OFF : undefined} onChange={(e) => { setPresetName(e.target.value); setReplaceAsk(null); }} onKeyDown={(e) => { if (e.key === 'Enter') { e.preventDefault(); void savePreset(); } }} placeholder="Save current as"  aria-label="Preset name" />
                  <OBtn kind="icon-sm" cls="ae-o-icon sm"  disabled={readOnly || !presetName.trim()} onClick={() => void savePreset()} title={readOnly ? SAMPLE_OFF : presetName.trim() ? 'Save the current controls as a preset' : 'Type a name first'} aria-label="Save preset"><Check className="h-4 w-4" strokeWidth={2} /></OBtn>
                </div>
                {replaceAsk && (
                  <div className="ae-o-replace" role="alert">
                    <span>Replace “{replaceAsk.name}”?</span>
                    <span className="flex gap-2"><OBtn kind="link" cls="ae-o-linkbtn"  onClick={() => void savePreset(replaceAsk)}>Replace</OBtn><OBtn kind="link" cls="ae-o-linkbtn"  onClick={() => setReplaceAsk(null)}>Keep</OBtn></span>
                  </div>
                )}
              </div>
            )}
            {tab === 'references' && (
              <div ref={refsRef} className="ae-o-tree px-3 text-[12px] text-[var(--o-muted)]" role="tabpanel" aria-labelledby="ae-o-tab-references">
                <div className="mb-1.5">Local file paths on this host, one per line. Uploaded fresh for every request.</div>
                {refField('referenceImages', 'Images', rule.maxReferenceImages, 3, '/path/hero.jpg')}
                {refField('referenceVideos', 'Videos', rule.maxReferenceVideos, 2, '/path/motion-ref.mp4')}
                {refField('referenceAudios', 'Audio', rule.maxReferenceAudios, 2, '/path/voice.mp3')}
                <div className="mb-1 mt-2 flex justify-between gap-2"><span>Start frame</span>{!rule.startFrame && <em className="not-italic text-[var(--o-dim)]">{c.startFrame ? 'kept, not sent' : 'not on this model'}</em>}</div>
                <OInput cls="ae-o-text w-full" keep="w-full" value={c.startFrame ?? ''} onChange={(e) => set('startFrame', e.target.value.trim() || null)} disabled={readOnly || (!rule.startFrame && !c.startFrame)} title={readOnly ? SAMPLE_OFF : !rule.startFrame ? `${c.model} has no start frame` : undefined} placeholder={rule.startFrame ? '/path/first-frame.jpg' : `${c.model} has no start frame`}  aria-label="Start frame path" />
                {c.startFrame && refStatus([c.startFrame])}
                <div className="mb-1 mt-2 flex justify-between gap-2"><span>End frame</span>{!rule.endFrame && <em className="not-italic text-[var(--o-dim)]">{c.endFrame ? 'kept, not sent' : 'not on this model'}</em>}</div>
                <OInput cls="ae-o-text w-full" keep="w-full" value={c.endFrame ?? ''} onChange={(e) => set('endFrame', e.target.value.trim() || null)} disabled={readOnly || (!rule.endFrame && !c.endFrame)} title={readOnly ? SAMPLE_OFF : !rule.endFrame ? `${c.model} has no end frame` : undefined} placeholder={rule.endFrame ? '/path/last-frame.jpg' : `${c.model} has no end frame`}  aria-label="End frame path" />
                {c.endFrame && refStatus([c.endFrame])}
              </div>
            )}
            {/* the tree foot keeps its shortcuts to the stage sheets (Q-MC-3) */}
            <div className="ae-o-tree-foot">
              {tool('describe', <Wand2 className="h-4 w-4" strokeWidth={1.8} />, 'Describe it', true)}
              {tool('dialogue', <MessageSquare className="h-4 w-4" strokeWidth={1.8} />, 'Dialogue', true)}
              {tool('lints', <AlertTriangle className="h-4 w-4" strokeWidth={1.8} />, 'Lints', true)}
              {tool('prompt', <FileText className="h-4 w-4" strokeWidth={1.8} />, 'Composed prompt', true)}
              {refsTool(<Images className="h-4 w-4" strokeWidth={1.8} />, true)}
            </div>
          </div>

          {/* stage */}
          <div className="ae-o-stage" data-asset-id={asset?.id} data-ring={ring > 0 ? 'on' : undefined} key={`stage-${ring}`}>
            {asset?.path && !health.dry && !stageError ? (
              asset.kind === 'video'
                ? <video key={asset.id} src={asset.path} controls className="ae-o-arrive" onError={() => setStageError('video')} />
                : <img key={asset.id} src={asset.path} alt={`Motion asset ${Math.min(assetIndex, stageAssets.length - 1) + 1}`} className="ae-o-arrive" onError={() => setStageError('image')} />
            ) : <div className="ae-o-grid absolute inset-0" />}
            {(() => {
              let msg: React.ReactNode = null;
              if (stageError) msg = <>Couldn&apos;t load this file · <OBtn kind="link" cls="ae-o-linkbtn"  onClick={() => openSheet('runs', { runsKind: 'motion' })}>Open in Runs</OBtn></>;
              else if (asset && readOnly) msg = `Sample asset · ${asset.model}${asset.durationSec ? ` · ${asset.durationSec}s` : ''}. The sample has no video file.`;
              else if (asset && health.dry) msg = 'Dry run placeholder delivered';
              else if (asset && !asset.path) msg = 'Delivered; the file is still downloading';
              else if (!asset && running.length) msg = `Generating ${running[0].title} · ${fmtMs(running[0].elapsedMs)} · ${titleCaseStatus(running[0].arcads?.lastStatus) || 'Waiting'}`;
              else if (!asset) msg = 'No motion video yet. Generate to see it here.';
              return msg && !panel ? <div className="ae-o-stage-note ae-o-arrive">{msg}</div> : null;
            })()}
            {!panel && (
              <div className="ae-o-hud">
                <OBtn kind="pill" title="Open the composed prompt" aria-label={`Open composed prompt, ${words}`} onClick={() => showPrompt(null)}><FileText className="h-3.5 w-3.5" strokeWidth={2} /> {words}</OBtn>
                <OBtn kind="pill" title="Open lints" aria-label={`Open lints, ${lintWord}`} data-err={blocking.length > 0} onClick={() => setPanel('lints')}><AlertTriangle className="h-3.5 w-3.5" strokeWidth={2} /> {lintWord}</OBtn>
                {beatPick !== null && firstVersion?.beats[beatPick] && <OBtn kind="pill" title="Back to whole script" aria-label={`${firstVersion.beats[beatPick].name} beat. Back to whole script`} onClick={() => pickBeat(null)}><Film className="h-3.5 w-3.5" strokeWidth={2} /> {firstVersion.beats[beatPick].name} beat</OBtn>}
                <OBtn kind="pill" title="Open Runs: delivered motion" aria-label={`${assetsWord}. Open Runs`} onClick={() => openSheet('runs', { runsKind: 'motion' })}><Images className="h-3.5 w-3.5" strokeWidth={2} /> <span key={flash} className={flash ? 'ae-o-popnum' : ''}>{assetsWord}{asset ? ` · ${Math.min(assetIndex, stageAssets.length - 1) + 1}` : ''}</span></OBtn>
              </div>
            )}
            {panel && (
              <div className="ae-o-sheet" role="region" aria-label={PANEL_TITLE[panel]}>
                <div className="ae-o-sheet-head"><span>{PANEL_TITLE[panel]}{panel === 'prompt' ? ` · ${words}` : ''}</span><span className="flex gap-3">{panel === 'prompt' && <OBtn kind="link" onClick={copyPrompt}>Copy</OBtn>}<OBtn kind="link" onClick={() => setPanel(null)}>Close</OBtn></span></div>
                {panel === 'prompt' && (<>
                  {promptFocus === 'output' && (
                    <div className="ae-o-fields">
                      <div className="mb-1 text-[var(--o-muted)]">Sent to Arcads as fields, not in the prompt:</div>
                      <ul>{BOUND_FIELDS.map((f) => (<li key={f.key}><span>{f.label}</span><span>{valueText(f.key, c[f.key])}</span></li>))}</ul>
                    </div>
                  )}
                  <div ref={promptRef} className="ae-o-promptbody whitespace-pre-wrap leading-relaxed">
                    {promptFocus && promptFocus !== 'output'
                      ? PART_ORDER.filter((p) => p === 'constraints' || !isHidden(p)).map((p) => (<div key={p} className={p === promptFocus ? 'ae-o-hl' : undefined}>{composed.parts[p]}</div>))
                      : composed.prompt}
                  </div>
                </>)}
                {panel === 'lints' && (allLints.length === 0 ? <div className="text-[var(--o-muted)]">No lints. The request and the prompt are clean.</div> : <ul className="space-y-1.5">{allLints.map((l, i) => (<li key={i} style={{ color: l.level === 'error' ? 'var(--o-red-text)' : 'var(--o-yellow)' }}>{l.level}: {l.message}</li>))}</ul>)}
                {panel === 'describe' && (<div className="flex flex-col gap-2"><OInput cls="ae-o-text w-full" keep="w-full" value={describe} maxLength={CONSOLE_LIMITS.subject} onChange={(e) => setDescribe(e.target.value)} onKeyDown={(e) => e.key === 'Enter' && applyDescribe()} placeholder="close-up handheld selfie in the kitchen, slow, 8s"  aria-label="Describe the shot" autoFocus /><div className="flex items-center justify-between text-[var(--o-muted)]"><span>Locked parts stay as they are. {describe.length} / {CONSOLE_LIMITS.subject}</span><OBtn kind="pill" cls="ae-o-pill" onClick={applyDescribe} disabled={!describe.trim()} title={describe.trim() ? 'Apply to the controls' : 'Type a description first'} >Apply</OBtn></div></div>)}
                {panel === 'dialogue' && (<>
                  {c.omitDialogue && <div className="mb-1.5 text-[11.5px] text-[var(--o-yellow)]">Dialogue is off: the text is kept but not sent. <OBtn kind="link" cls="ae-o-linkbtn"  onClick={() => set('omitDialogue', false)}>Turn it on</OBtn></div>}
                  <OTextarea cls="ae-o-text w-full resize-none" keep="w-full resize-none" value={c.dialogue} maxLength={CONSOLE_LIMITS.dialogue} onChange={(e) => set('dialogue', e.target.value)} rows={5} placeholder={approved.length ? 'From the approved script' : 'Spoken words, 2.5 per second'}  aria-label="Dialogue" />
                  <div className="mt-1.5 flex justify-between gap-3 text-[11.5px]"><span style={{ color: !dialogueFit || dialogueFit.fits ? 'var(--o-muted)' : 'var(--o-yellow)' }}>{dialogueFit?.note ?? ''}</span><span className="tabular-nums text-[var(--o-dim)]">{c.dialogue.length} / {CONSOLE_LIMITS.dialogue}</span></div>
                </>)}
                {panel === 'presets' && (<div className="flex flex-wrap gap-2">{presets.map((p) => (<OBtn kind="pill" cls="ae-o-pill" key={p.id}  data-on={activePreset === p.name} title={`Apply ${p.name}`} onClick={() => applyPreset(p)}>{p.name}{activePreset === p.name && presetEdited ? ' · edited' : ''}</OBtn>))}{presetsError && <span className="ae-o-inline-err">Couldn&apos;t load presets · <OBtn kind="link" cls="ae-o-linkbtn"  onClick={loadPresets}>Retry</OBtn></span>}</div>)}
                {panel === 'shortcuts' && (<ul className="ae-o-fields"><li className="text-[var(--o-muted)]"><span>Shortcuts are off while you type in a field.</span><span /></li>{CONSOLE_SHORTCUTS.map((s) => (<li key={s.keys}><span className="ae-o-kbd" data-mono-ok>{s.keys}</span><span className="text-right">{s.what}</span></li>))}</ul>)}
              </div>
            )}
          </div>

          {/* parameters */}
          <div className="ae-o-params">
            <div className="ae-o-params-top">
              {connected ? (
                <span className="ae-o-genwrap">
                  <OBtn kind="primary" cls="ae-o-generate"  data-busy={busy} disabled={!!block} title={block?.text ?? 'Opens the cost preview; nothing is sent without Confirm'} aria-describedby={block ? 'ae-o-block-caption' : undefined} onClick={() => { setGateError(null); setGate(true); }}><i /> {busy ? 'Sending' : `Generate${requests > 1 ? ` ×${requests}` : ''}`}</OBtn>
                  {block && !busy && <span id="ae-o-block-caption" className="ae-o-block-caption" title={block.text}>{block.text}{blockLabel(block.action, false) && <> · <OBtn kind="link" cls="ae-o-linkbtn"  onClick={() => fixBlock(block.action)}>{blockLabel(block.action, false)}</OBtn></>}</span>}
                </span>
              ) : (
                <span className="flex items-center gap-3"><OBtn kind="primary" cls="ae-o-generate"  disabled={readOnly} title={readOnly ? SAMPLE_OFF : 'Connect Arcads'} onClick={onConnect}><i /> Connect Arcads</OBtn><SetupLink href={health.arcads.setupHref} hint="ARCADS_BASIC_AUTH" label="Set up" /></span>
              )}
              {error && <span key={error} className="ae-o-shake text-[12px]" role="alert" style={{ color: 'var(--o-red-text)' }}>{error}</span>}
              <span className="ml-auto flex items-center gap-4">
                <span className="ae-o-file"><FileText className="h-[18px] w-[18px]" strokeWidth={1.6} />{approved.length > 1 ? (
                  <span className="ae-o-selwrap" title="Pick the approved script to generate from"><span>{firstScript ? scriptTitle(firstScript).slice(0, 30) : 'Script'}</span><OSelect value={firstScript?.id ?? ''} onChange={(e) => setScriptPick(e.target.value)} aria-label="Script">{approved.map((s) => (<option key={s.id} value={s.id}>{scriptTitle(s)}</option>))}</OSelect></span>
                ) : firstScript ? (
                  <OBtn kind="row" cls="ae-o-filebtn" keep="ae-o-filebtn"  title="Open this script in the Ad scripter" onClick={() => openSheet('scripter')}>{scriptTitle(firstScript).slice(0, 30)}</OBtn>
                ) : (
                  <OBtn kind="row" cls="ae-o-filebtn" keep="ae-o-filebtn"  disabled={readOnly || !campaign} title={readOnly ? SAMPLE_OFF : !campaign ? 'Open or create a campaign first' : 'Write and approve a script in the Ad scripter'} onClick={() => openSheet('scripter')}>No approved script · Write one</OBtn>
                )}</span>
                <span className="ae-o-pill lg" data-display-only="model picker frame" title={!modelHealth ? `${c.model}: not probed yet` : modelHealth.ok ? `${c.model}: probe delivered` : `${c.model}: probe failed, ${modelHealth.error}`}><span className="ae-o-dot" style={{ background: !modelHealth ? 'var(--o-dim)' : modelHealth.ok ? 'var(--o-green)' : 'var(--o-red-text)' }} /><Box className="h-4 w-4 text-[var(--o-muted)]" strokeWidth={1.7} /><span className="ae-o-selwrap"><span>{c.model}</span><OSelect value={c.model} onChange={(e) => set('model', e.target.value as VideoModel)} aria-label="Model">{VIDEO_MODELS.map((m) => (<option key={m} value={m}>{m}{UNVERIFIED_MODELS.includes(m) ? ' (limits inherited)' : ''}{health.arcads.health[m] ? (health.arcads.health[m].ok ? ' · probed ok' : ' · probe failed') : ''}</option>))}</OSelect></span></span>
              </span>
            </div>
            <div className="ae-o-cols">
              <ColumnBox col="shot" sphere="" hidden={isHidden} locked={locked} onEye={toggleOmit} onLock={toggleLock} menuOpen={menu === 'column:shot'} onMenu={(v) => setMenu(v ? 'column:shot' : null)} resetTo={presetObj?.name ?? null} onResetColumn={resetColumn} onShow={showPrompt}>
                <SelectRow label="Framing" value={c.framing} options={FRAMINGS} onChange={(v) => set('framing', v)} {...reset('framing')} />
                <SelectRow label="Movement" value={c.movement} options={MOVEMENTS} onChange={(v) => set('movement', v)} {...reset('movement')} />
                <SliderRow label="Camera height" value={HEIGHTS.indexOf(c.height)} min={0} max={HEIGHTS.length - 1} readout={c.height} edit="enum" options={HEIGHTS} onChange={(v) => set('height', HEIGHTS[v])} {...reset('height')} />
                <SliderRow label="Lens feel" value={Math.max(0, LENSES.indexOf(c.lensMm as (typeof LENSES)[number]))} min={0} max={LENSES.length - 1} readout={`${c.lensMm} mm`} edit="enum" options={LENSES.map((l) => `${l} mm`)} onChange={(v) => set('lensMm', LENSES[v])} {...reset('lensMm')} />
                <SliderRow label="Motion intensity" value={c.intensity} min={1} max={5} readout={INTENSITY_WORDS[c.intensity - 1]} edit="enum" options={INTENSITY_WORDS} onChange={(v) => set('intensity', v)} {...reset('intensity')} />
                {dRule.kind === 'enum' ? <SelectRow bound label="Duration" value={String(c.duration)} options={dRule.values.map(String)} suffix="s" onChange={(v) => set('duration', Number(v))} {...reset('duration')} /> : dRule.kind === 'range' ? <SliderRow bound label="Duration" value={c.duration} min={dRule.min} max={dRule.max} readout={`${c.duration}s`} edit="number" onChange={(v) => set('duration', v)} {...reset('duration')} /> : <SelectRow bound label="Duration" value="auto" options={['auto']} onChange={() => {}} disabled disabledReason={`${c.model} picks this itself`} />}
                <SelectRow bound label="Aspect" value={c.aspectRatio} options={ASPECTS.filter((a) => rule.aspects.includes(a))} onChange={(v) => set('aspectRatio', v)} {...reset('aspectRatio')} />
                {rule.resolutions.length > 0
                  ? <SelectRow bound label="Resolution" value={rule.resolutions.includes(c.resolution) ? c.resolution : rule.resolutions[0]} options={RESOLUTIONS.filter((r) => rule.resolutions.includes(r))} onChange={(v) => set('resolution', v)} {...reset('resolution')} />
                  : <SelectRow bound label="Resolution" value="auto" options={['auto']} onChange={() => {}} disabled disabledReason={`${c.model} picks this itself`} />}
              </ColumnBox>
              <ColumnBox col="look" sphere="gold" hidden={isHidden} locked={locked} onEye={toggleOmit} onLock={toggleLock} menuOpen={menu === 'column:look'} onMenu={(v) => setMenu(v ? 'column:look' : null)} resetTo={presetObj?.name ?? null} onResetColumn={resetColumn} onShow={showPrompt}>
                <SwatchRow label="Lighting" value={c.lighting} options={LIGHTING} attr="data-light" onChange={(v) => set('lighting', v)} {...reset('lighting')} />
                <SwatchRow label="Style" value={c.style} options={STYLES} attr="data-style" onChange={(v) => set('style', v)} {...reset('style')} />
                <SelectRow label="Setting" value={c.setting} options={SETTINGS} onChange={(v) => set('setting', v)} {...reset('setting')} />
                <SliderRow label="Imperfection" value={c.imperfection} min={0} max={9} readout={`${c.imperfection}`} edit="number" onChange={(v) => set('imperfection', v)} {...reset('imperfection')} />
                <SliderRow label="Skin realism" value={c.skinRealism} min={0} max={4} readout={`${c.skinRealism}`} edit="number" onChange={(v) => set('skinRealism', v)} {...reset('skinRealism')} />
                <SliderRow label="Motion cues" value={c.motionCues} min={3} max={6} readout={`${c.motionCues}`} edit="number" onChange={(v) => set('motionCues', v)} {...reset('motionCues')} />
                <SelectRow label="Product intro" value={c.intro} options={INTRO_OPTIONS.map((o) => o.value)} labels={Object.fromEntries(INTRO_OPTIONS.map((o) => [o.value, o.label]))} onChange={(v) => set('intro', v)} {...reset('intro')} />
                <SelectRow label="Tone" value={c.tone} options={TONE_OPTIONS.map((o) => o.value)} labels={Object.fromEntries(TONE_OPTIONS.map((o) => [o.value, o.label]))} onChange={(v) => set('tone', v)} {...reset('tone')} />
                <TextRow label="Subject" value={c.subject} placeholder="who is in frame" maxLength={CONSOLE_LIMITS.subject} onChange={(v) => set('subject', v)} {...reset('subject')} />
              </ColumnBox>
              <ColumnBox col="output" sphere="red" hidden={isHidden} locked={locked} onEye={toggleOmit} onLock={toggleLock} menuOpen={menu === 'column:output'} onMenu={(v) => setMenu(v ? 'column:output' : null)} resetTo={presetObj?.name ?? null} onResetColumn={resetColumn} onShow={showPrompt}>
                <SliderRow bound label="Variations" value={c.variations} min={1} max={10} readout={`${c.variations} req`} edit="number" onChange={(v) => set('variations', v)} {...reset('variations')} />
                <CheckRow bound label="Audio" on={c.audioEnabled && rule.audioToggle} disabled={!rule.audioToggle} disabledReason={rule.speaks ? `${c.model} always renders audio` : `${c.model} has no audio`} onChange={(v) => set('audioEnabled', v)} {...reset('audioEnabled')} />
                <CheckRow bound label="Enhance prompt" on={c.enhance} muted onChange={(v) => set('enhance', v)} hint="+8 credits" {...reset('enhance')} />
                <SelectRow bound label="Bitrate" value={c.bitrateMode} options={['standard', 'high'] as const} onChange={(v) => set('bitrateMode', v)} {...reset('bitrateMode')} />
                <CheckRow label="Dialogue" on={dialogueOn} onChange={(v) => { if (!v) set('omitDialogue', true); else { set('omitDialogue', false); if (!c.dialogue.trim()) setPanel('dialogue'); } }} />
                <TextRow label="Overlay text" value={c.onScreenText} placeholder="none" maxLength={CONSOLE_LIMITS.onScreenText} onChange={(v) => set('onScreenText', v)} {...reset('onScreenText')} />
                <SelectRow label="Text reveal" value={c.textReveal} options={TEXT_REVEALS} onChange={(v) => set('textReveal', v)} disabled={!c.onScreenText} disabledReason="Add overlay text first" {...reset('textReveal')} />
                <LinkRow bound label="References" value={plural(c.referenceImages.length + c.referenceVideos.length + c.referenceAudios.length, 'file')} onClick={goRefs} />
                <LinkRow bound label="Frames" value={c.startFrame && c.endFrame ? 'start + end' : c.startFrame ? 'start' : c.endFrame ? 'end' : 'none'} onClick={goRefs} disabled={!rule.startFrame && !rule.endFrame} title={!rule.startFrame && !rule.endFrame ? `${c.model} takes no start or end frame` : undefined} />
              </ColumnBox>
            </div>
          </div>
        </div>

        {/* node canvas */}
        <WorkflowCanvas campaign={campaign} health={health} runs={runs} readOnly={readOnly} consoleControls={c} onRuns={onRuns} onCampaign={onCampaign} onNotice={(m) => notify(m)} onConnect={onConnect}
          onChooseCampaign={readOnly ? undefined : openCampaignPicker} chooseLabel={campaigns.length ? 'Choose a campaign' : 'Create a campaign'} onScripter={readOnly ? undefined : () => openSheet('scripter')} />
      </div>

      {probeAsk && !gate && (
        <SpendGate
          id="motion-probe"
          title={`Probe ${c.model}`}
          dry={health.dry}
          items={[<>One cheapest request on this model with no campaign attached: {MODEL_RULES[c.model].resolutions[0] ?? 'default resolution'}, the shortest legal duration, no references. The verdict and the raw reply are recorded in Credits & health.</>]}
          balance={balance}
          session={{ spent: health.arcads.sessionSpent, cap: health.arcads.sessionCap }}
          block={readOnly ? { text: SAMPLE_OFF } : !connected ? { text: 'Connect Arcads first', actionLabel: 'Connect Arcads', onAction: () => { setProbeAsk(false); onConnect(); } } : null}
          capAsk={capAsk}
          error={gateError}
          busy={busy}
          confirmLabel="Run the probe"
          onConfirm={(override) => void probeThisModel(override)}
          onCancel={() => { setProbeAsk(false); setCapAsk(null); }}
        />
      )}
      {gate && (
        <SpendGate
          id="motion-gate"
          title="Submit to Arcads"
          dry={health.dry}
          items={[
            <><span className="text-[var(--o-text)]">{c.model}</span> · {dRule.kind === 'auto' ? 'auto length' : `${c.duration}s`} · {c.aspectRatio} · {rule.resolutions.length ? c.resolution : 'auto resolution'}</>,
            <>{plural(requests, 'request')}, each charged separately{c.enhance ? ', each +8 credits for enhancement' : ''}</>,
            <>{plural(uploads, 'reference upload')} per request</>,
          ]}
          balance={balance}
          session={{ spent: health.arcads.sessionSpent, cap: health.arcads.sessionCap }}
          block={block ? { text: block.text, actionLabel: blockLabel(block.action, true), onAction: block.action ? () => fixBlock(block.action) : undefined } : null}
          capAsk={capAsk}
          error={gateError}
          busy={busy}
          confirmLabel={`Confirm${requests > 1 ? ` ×${requests}` : ''}`}
          onConfirm={(override) => void submit(override)}
          onCancel={() => { setGate(false); setCapAsk(null); }}
        />
      )}
    </>
  );
}

/* ───────────────────────────── columns and rows ───────────────────────────── */

function ColumnBox({ col, sphere, hidden, locked, onEye, onLock, menuOpen, onMenu, resetTo, onResetColumn, onShow, children }: {
  col: Column; sphere: string; hidden: (p: Part) => boolean; locked: Set<Part>; onEye: (p: Part | null) => void; onLock: (p: Part | null) => void;
  menuOpen: boolean; onMenu: (open: boolean) => void; resetTo: string | null; onResetColumn: (c: Column) => void; onShow: (focus: Part | 'output') => void; children: React.ReactNode;
}) {
  const part = COLUMN_PART[col];
  const name = COLUMN_NAME[col];
  const off = part ? hidden(part) : false;
  return (
    <div>
      <div className="ae-o-colhead">
        <span className="relative inline-flex">
          <OBtn kind="pill" cls="ae-o-pill"  aria-haspopup="menu" aria-expanded={menuOpen} title={`${name} column menu`} onClick={() => onMenu(!menuOpen)}><span className={`ae-o-sphere ${sphere}`} />{name}</OBtn>
          {menuOpen && (
            <Popover id={`column-${col}`} label={`${name} column`} onClose={() => onMenu(false)} className="left-0 top-[46px]">
              <OBtn kind="row" role="menuitem" onClick={() => { onMenu(false); onResetColumn(col); }}><RotateCcw className="h-3.5 w-3.5" strokeWidth={2} />Reset {name} to {resetTo ?? 'defaults'}</OBtn>
              <OBtn kind="row" role="menuitem" onClick={() => { onMenu(false); onShow(part ?? 'output'); }}><FileText className="h-3.5 w-3.5" strokeWidth={2} />{part ? 'Show in composed prompt' : 'Show the fields sent to Arcads'}</OBtn>
            </Popover>
          )}
        </span>
        {part && (
          <span className="ae-o-tools-mini">
            <OBtn kind="icon-sm" title={off ? `Include ${part} in the prompt` : `Hide ${part} from the prompt`} aria-label={`${off ? 'Include' : 'Hide'} ${name}`} data-on={!off} onClick={() => onEye(part)}>{off ? <EyeOff className="h-4 w-4" strokeWidth={1.7} /> : <Eye className="h-4 w-4" strokeWidth={1.7} />}</OBtn>
            <OBtn kind="icon-sm" title={locked.has(part) ? 'Unlock: Describe it may change this part' : 'Protect: Describe it will not change this part'} aria-label={`${locked.has(part) ? 'Unlock' : 'Lock'} ${name}`} aria-pressed={locked.has(part)} data-on={locked.has(part)} onClick={() => onLock(part)}>{locked.has(part) ? <Lock className="h-4 w-4" strokeWidth={1.7} /> : <LockOpen className="h-4 w-4" strokeWidth={1.7} />}</OBtn>
          </span>
        )}
      </div>
      <div className="ae-o-col" style={{ opacity: off ? 0.5 : 1, transition: 'opacity 160ms' }}>{children}</div>
    </div>
  );
}
