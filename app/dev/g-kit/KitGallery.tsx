'use client';

import { useEffect, useRef, useState, type ReactNode } from 'react';
import { Archive, Bell, Copy, Home, ImageOff, Inbox, MoreHorizontal, Pencil, Plus, Settings, Trash2 } from 'lucide-react';
import {
  GAsk,
  GBadge,
  GBigNumber,
  GButton,
  GCard,
  GCardRaised,
  GCheck,
  GChip,
  GDateTime,
  GDelta,
  GDialog,
  GDisplay,
  GDotNumber,
  GDrawer,
  GEmpty,
  GErrorLine,
  GEyebrow,
  GField,
  GHeroCard,
  GInput,
  GKeyValue,
  GList,
  GMenu,
  GPage,
  GPill,
  GPillNav,
  GPopover,
  GProgress,
  GRow,
  GSampleBadge,
  GSearch,
  GSectionHeader,
  GSegmented,
  GSelect,
  GSheet,
  GSkeleton,
  GStat,
  GSwatch,
  GTable,
  GTabs,
  GTag,
  GTextarea,
  GToggle,
  GTooltip,
  GWell,
  useToast,
} from '@/components/g';
import { DOT_CHARSET } from '@/components/g/logic';
import { deriveClassicTheme, groundFor } from '@/lib/appearance';

/**
 * Every lane-1d primitive in every state it defines, on fictional fixture
 * values (nothing here is real data; the page carries a Sample badge). The
 * Accent and Mode controls set the html attributes for this preview only;
 * the real choice lives in Settings (lane 2a).
 */

const ACCENT_OPTIONS = [
  { value: 'ice', label: 'Ice' },
  { value: 'lime', label: 'Lime' },
  { value: 'ember', label: 'Ember' },
  { value: 'mint', label: 'Mint' },
] as const;
const MODE_OPTIONS = [
  { value: 'dark', label: 'Dark' },
  { value: 'light', label: 'Light' },
] as const;
type Accent = (typeof ACCENT_OPTIONS)[number]['value'];
type Mode = (typeof MODE_OPTIONS)[number]['value'];

/** Preview the style on <html> while the gallery is mounted; restore on leave. */
function usePreviewAppearance(accent: Accent, mode: Mode) {
  const saved = useRef<Record<string, string | null> | null>(null);
  useEffect(() => {
    const html = document.documentElement;
    const keys = ['data-ui', 'data-accent', 'data-mode', 'data-ground'];
    saved.current = Object.fromEntries(keys.map((k) => [k, html.getAttribute(k)]));
    return () => {
      for (const [k, v] of Object.entries(saved.current ?? {})) {
        if (v === null) html.removeAttribute(k);
        else html.setAttribute(k, v);
      }
    };
  }, []);
  useEffect(() => {
    const html = document.documentElement;
    html.setAttribute('data-ui', 'glados');
    html.setAttribute('data-accent', accent);
    html.setAttribute('data-mode', mode);
    // the glass light reads the ground GladOS paints (lib/appearance.ts groundFor)
    html.setAttribute('data-ground', groundFor(deriveClassicTheme(accent, mode), mode, 'glados'));
  }, [accent, mode]);
}

function initial<T extends string>(attr: string, allowed: readonly { value: T }[], fallback: T): T {
  if (typeof document === 'undefined') return fallback;
  const v = document.documentElement.getAttribute(attr);
  return (allowed.find((o) => o.value === v)?.value ?? fallback) as T;
}

function Section({ id, title, caption, children }: { id: string; title: string; caption?: string; children: ReactNode }) {
  return (
    <section id={id} className="mt-9 scroll-mt-20" aria-labelledby={`${id}-h`}>
      <GSectionHeader id={`${id}-h`} title={title} caption={caption} />
      <div className="mt-5 grid grid-cols-[minmax(0,1fr)] gap-5">{children}</div>
    </section>
  );
}

function Row({ label, children }: { label: string; children: ReactNode }) {
  return (
    <div className="grid min-w-0 gap-2">
      <GEyebrow>{label}</GEyebrow>
      <div className="flex min-w-0 flex-wrap items-center gap-3">{children}</div>
    </div>
  );
}

type Deal = { id: string; name: string; stage: string; value: number | null };
const DEALS: Deal[] = [
  { id: 'd1', name: 'Northwind desk lamp', stage: 'Proposal', value: 12000 },
  { id: 'd2', name: 'Lumen notebook', stage: 'Signed', value: 48500 },
  { id: 'd3', name: 'Harbor coffee', stage: 'Lead', value: null },
  { id: 'd4', name: 'Atlas backpack', stage: 'Proposal', value: 7300 },
];

export function KitGallery() {
  const [accent, setAccent] = useState<Accent>('ice');
  const [mode, setMode] = useState<Mode>('dark');
  // Start from whatever the page already has (Settings, or ?mode=), after hydration.
  useEffect(() => {
    setAccent(initial('data-accent', ACCENT_OPTIONS, 'ice'));
    setMode(initial('data-mode', MODE_OPTIONS, 'dark'));
  }, []);
  usePreviewAppearance(accent, mode);
  const { toast } = useToast();

  const [range, setRange] = useState('30d');
  const [view, setView] = useState('list');
  const [tab, setTab] = useState('engine');
  const [chips, setChips] = useState({ open: true, mine: false, flagged: false });
  const [toggle, setToggle] = useState(true);
  const [checks, setChecks] = useState({ a: true, b: false });
  const [text, setText] = useState('');
  const [bad, setBad] = useState('not-an-email');
  const [notes, setNotes] = useState('A caption that grows as you type.\nUp to eight lines, then it scrolls.');
  const [query, setQuery] = useState('');
  const [ask, setAsk] = useState('');
  const [filter, setFilter] = useState('');
  const [swatch, setSwatch] = useState('a');
  const [selected, setSelected] = useState('d2');
  const [sheet, setSheet] = useState(false);
  const [drawer, setDrawer] = useState<number>(0);
  const [dock, setDock] = useState(false);
  const [dialog, setDialog] = useState(false);
  const [busy, setBusy] = useState(false);
  const [pop, setPop] = useState(false);

  const filtered = DEALS.filter((d) => d.name.toLowerCase().includes(filter.toLowerCase()));

  return (
    <GPage>
      <div className="mx-auto max-w-[1180px] px-6 py-8">
        <GSectionHeader
          size="page"
          title="Kit gallery"
          caption="Every G primitive from lane 1d in each state it defines. Dev only; not in the nav."
          action={<GSampleBadge fixture="kit gallery fixtures (fictional brands and numbers)" />}
        />
        <div className="mt-6 flex flex-wrap items-center gap-4">
          <GSegmented label="Accent preview" size="md" options={ACCENT_OPTIONS} value={accent} onChange={setAccent} />
          <GSegmented label="Mode preview" size="md" options={MODE_OPTIONS} value={mode} onChange={setMode} />
        </div>

        <Section id="frame" title="Frame and type" caption="GDisplay, GSectionHeader sizes, GEyebrow">
          <GCard>
            <GDisplay size="title">A title at 28, light</GDisplay>
            <div className="mt-5 grid gap-5">
              <GSectionHeader size={30} title="Section 30" caption="One muted caption line" action={<GButton variant="sm">See all</GButton>} />
              <GSectionHeader title="Section 26" caption="The default section size" />
              <GSectionHeader size={22} title="Section 22" />
              <GEyebrow>Eyebrow: says what the headline is</GEyebrow>
            </div>
          </GCard>
        </Section>

        <Section id="surfaces" title="Surfaces" caption="GCard, GCardRaised, GWell, GHeroCard (value, action, unconfigured)">
          <div className="grid gap-5 md:grid-cols-3">
            <GCard>
              <p className="text-[15px]">Card</p>
              <p className="mt-1 text-[11.5px] opacity-80">Static: no hover lift.</p>
              <GWell className="mt-4">A well inside a card</GWell>
            </GCard>
            <GCard onClick={() => toast({ text: 'Card opened', tone: 'ok' })} aria-label="Open the card">
              <p className="text-[15px]">Clickable card</p>
              <p className="mt-1 text-[11.5px] opacity-80">Lifts 1px on hover; the whole card is the button.</p>
            </GCard>
            <GCardRaised p={16}>
              <p className="text-[15px]">Raised card</p>
              <GWell className="mt-4">Well inside raised: card fill</GWell>
            </GCardRaised>
          </div>
          <div className="grid gap-5 md:grid-cols-3">
            <GHeroCard variant="a" caption="Revenue · 30 days" value="$631K" status="12 deals closed" title="Fixture value" strip={<GDelta cur={631} prev={590} window="vs previous 30 days" />} />
            <GHeroCard variant="b" caption="Inbound · 24h" value={48} status="Open the inbox" onClick={() => toast({ text: 'Would open the inbox' })} />
            <GHeroCard variant="c" caption="Payments" setupHref="/integrations#stripe" setupHint="Fixture: STRIPE_SECRET_KEY is not set" />
          </div>
          <div className="grid gap-5 md:grid-cols-2">
            <GCard media={<div className="flex h-full items-center justify-center"><ImageOff size={20} strokeWidth={1.5} aria-label="No image" /></div>} mediaCaption="Media slot: broken media shows the well and an icon">
              <p className="text-[15px]">Card with media</p>
            </GCard>
            <div className="flex flex-wrap gap-5">
              <div className="w-[200px]">
                <GHeroCard variant="a" size="sm" caption="200px hero" value="$631K" status="Dots auto-fit the width" />
              </div>
              <div className="w-[150px]">
                <GHeroCard variant="b" size="sm" caption="150px hero" value="$1,240,000" status="Below dot size 3: text" />
              </div>
            </div>
          </div>
        </Section>

        <Section id="numbers" title="Numbers" caption="GDotNumber glyphs and auto-fit, GBigNumber, GDelta on lib/delta, GStat states">
          <GCard>
            <Row label="Every C7 glyph, sizes 3 to 7">
              <div className="grid w-full min-w-0 gap-3 overflow-x-auto pb-1">
                {([3, 4, 5, 6, 7] as const).map((s) => (
                  <div key={s} className="flex flex-wrap gap-x-6 gap-y-3">
                    <GDotNumber value={DOT_CHARSET.slice(0, 10)} size={s} />
                    <GDotNumber value={DOT_CHARSET.slice(10)} size={s} />
                  </div>
                ))}
              </div>
            </Row>
            <div className="mt-6 grid gap-3 overflow-x-auto">
              <GEyebrow>Auto-fit of $631K at 120, 225, 290, 400 px (never overflows)</GEyebrow>
              {[120, 225, 290, 400].map((w) => (
                <GWell key={w} className="!p-2">
                  <div style={{ width: w }}>
                    <GDotNumber value="$631K" size="auto" />
                  </div>
                </GWell>
              ))}
            </div>
            <div className="mt-6 grid gap-4">
              <Row label="No glyph for a character: plain text, never a dropped digit">
                <GDotNumber value="1.2 days" size={5} />
                <GDotNumber value="2.29x" size={5} />
                <GDotNumber value={null} size={5} />
              </Row>
              <Row label="GBigNumber">
                <GBigNumber value="1m 37s" unit="elapsed" />
                <GBigNumber value={1240000} unit="views" />
              </Row>
              <Row label="GDelta: pct, flat, absolute above 999%, absolute when prev is 0, money">
                <GDelta cur={101.4} prev={100} window="vs previous 30 days" />
                <GDelta cur={88} prev={100} />
                <GDelta cur={100.01} prev={100} />
                <GDelta cur={375000} prev={22000} money />
                <GDelta cur={12} prev={0} />
                <span className="text-[11.5px]">
                  Small base (prev 2, min 5): <GDelta cur={19} prev={2} min={5} />
                  hidden
                </span>
              </Row>
            </div>
          </GCard>
          <GCard>
            <GEyebrow>GStat: dots up to 6 characters, text beyond (C6)</GEyebrow>
            <div className="mt-4 grid grid-cols-2 gap-5 sm:grid-cols-4">
              <GStat value={1284} label="Leads" source="Fixture · leads, all time" href="/dev/g-kit#numbers" badge extra={<GDelta cur={1284} prev={1190} />} />
              <GStat value="$48.5K" label="Pipeline" source="Fixture · open deals" onClick={() => toast({ text: 'Would open the pipeline' })} />
              <GStat value={1240000} label="Reach" source="Fixture · reach" />
              <GStat value="12,431,000" label="Long value" source="Fixture · over six characters renders as text" />
              <GStat value={0} label="Replies" source="Fixture · a true zero at 55%" />
              <GStat value={null} label="Unknown" source="Fixture · null renders a dash, never 0" />
              <GStat value={null} label="Loading" source="Fixture" state="loading" />
              <GStat value={312} label="Errored" source="Fixture" state="error" />
              <GStat value={null} label="Payouts" source="Fixture · payouts" state="unconfigured" missing="STRIPE_SECRET_KEY" />
            </div>
          </GCard>
        </Section>

        <Section id="controls" title="Controls" caption="GButton, GChip, GSegmented, GTabs, GBadge, GTag, GToggle, GCheck, GPill, GPillNav">
          <GCard>
            <div className="grid gap-5">
              <Row label="GButton">
                <GButton>Default</GButton>
                <GButton variant="ink">Ink, one per view</GButton>
                <GButton variant="danger">Delete</GButton>
                <GButton variant="sm">Small</GButton>
                <GButton size="compact">Compact</GButton>
                <GButton variant="icon" aria-label="Add">
                  <Plus size={16} strokeWidth={1.75} aria-hidden />
                </GButton>
                <GButton busy={busy} onClick={() => { setBusy(true); setTimeout(() => setBusy(false), 1500); }}>
                  Busy keeps width
                </GButton>
                <GButton disabled disabledReason="Fixture: nothing to send yet">Disabled</GButton>
                <GButton href="/dev/g-kit#overlays">Link as button</GButton>
              </Row>
              <Row label="GChip (aria-pressed)">
                <GChip pressed={chips.open} onPressedChange={(v) => setChips({ ...chips, open: v })}>Open only</GChip>
                <GChip pressed={chips.mine} onPressedChange={(v) => setChips({ ...chips, mine: v })} count={4}>Mine</GChip>
                <GChip pressed={chips.flagged} onPressedChange={(v) => setChips({ ...chips, flagged: v })} size="compact">Compact</GChip>
                <GChip pressed={false} onPressedChange={() => {}} disabled disabledReason="Fixture: no flagged items">Disabled</GChip>
              </Row>
              <Row label="GSegmented sm, md, compact; overflow fades">
                <GSegmented label="Range" options={[{ value: '7d', label: '7d' }, { value: '30d', label: '30d' }, { value: '90d', label: '90d' }, { value: 'all', label: 'All', disabled: true, title: 'Fixture: no history yet' }]} value={range} onChange={setRange} />
                <GSegmented label="View" size="md" options={[{ value: 'list', label: 'List' }, { value: 'board', label: 'Board' }]} value={view} onChange={setView} />
                <GSegmented label="View compact" size="compact" options={[{ value: 'list', label: 'List' }, { value: 'board', label: 'Board' }]} value={view} onChange={setView} />
                <div style={{ width: 180 }}>
                  <GSegmented
                    label="Sources"
                    options={['Instagram', 'TikTok', 'YouTube', 'LinkedIn', 'Email'].map((s) => ({ value: s, label: s }))}
                    value="Instagram"
                    onChange={() => {}}
                  />
                </div>
              </Row>
              <Row label="GTabs (title, ?tab= in the URL)">
                <GTabs label="Engine views" param="tab" value={tab} onChange={setTab} tabs={[{ value: 'engine', label: 'Engine' }, { value: 'runs', label: 'Runs' }, { value: 'off', label: 'Archived', disabled: true }]} />
              </Row>
              <Row label="GBadge, GTag tones, dot, xs">
                <GBadge>Headline</GBadge>
                <GTag>Neutral</GTag>
                <GTag tone="accent" dot>Accent</GTag>
                <GTag tone="ok" dot>All nominal</GTag>
                <GTag tone="warn" dot>2 need attention</GTag>
                <GTag tone="err" dot>Failed</GTag>
                <GTag tone="outline">Outline</GTag>
                <GTag size="xs">xs edge label</GTag>
              </Row>
              <Row label="GToggle, GCheck">
                <GToggle checked={toggle} onCheckedChange={setToggle}>Auto-refresh</GToggle>
                <GToggle checked={false} onCheckedChange={() => {}} disabled disabledReason="Fixture: needs a key">Disabled</GToggle>
                <GCheck checked={checks.a} onCheckedChange={(v) => setChecks({ ...checks, a: v })}>Reviewed</GCheck>
                <GCheck checked={checks.b} onCheckedChange={(v) => setChecks({ ...checks, b: v })} aria-label="Mark done" />
              </Row>
            </div>
          </GCard>
          <div className="grid gap-5 md:grid-cols-[232px_1fr]">
            <GPillNav
              label="Gallery sections"
              items={[
                { id: 'home', label: 'Overview', icon: <Home />, href: '/dev/g-kit#frame', active: true },
                { id: 'inbox', label: 'Inbox', icon: <Inbox />, badge: 12, href: '/dev/g-kit#lists' },
                { id: 'alerts', label: 'Alerts', icon: <Bell />, badge: 'Live', href: '/dev/g-kit#states' },
                { id: 'settings', label: 'Settings', icon: <Settings />, href: '/dev/g-kit#inputs' },
              ]}
            />
            <GCard>
              <GEyebrow>GPill alone (button form)</GEyebrow>
              <div className="mt-3 max-w-[260px]">
                <GPill label="Archive" icon={<Archive />} badge={3} onClick={() => toast({ text: 'Would open the archive' })} />
              </div>
            </GCard>
          </div>
        </Section>

        <Section id="inputs" title="Inputs" caption="GInput, GField, GTextarea, GSelect, GDateTime, GSearch, GAsk, GSwatch">
          <GCard>
            <div className="grid gap-5 md:grid-cols-2">
              <GField label="Name" hint="Shown on invoices">
                <GInput value={text} onChange={(e) => setText(e.target.value)} placeholder="Northwind" />
              </GField>
              <GField label="Email" error="Enter an address like name@example.com">
                <GInput invalid value={bad} onChange={(e) => setBad(e.target.value)} />
              </GField>
              <GField label="Disabled">
                <GInput disabled value="Locked" readOnly title="Fixture: set by the server" />
              </GField>
              <GField label="Compact">
                <GInput size="compact" placeholder="28px" />
              </GField>
              <GField label="Caption (grows to 8 lines)">
                <GTextarea value={notes} onChange={(e) => setNotes(e.target.value)} />
              </GField>
              <GField label="Stage">
                <GSelect defaultValue="proposal" options={[{ value: 'lead', label: 'Lead' }, { value: 'proposal', label: 'Proposal' }, { value: 'signed', label: 'Signed' }]} />
              </GField>
              <GField label="Schedule">
                <GDateTime defaultValue="2026-10-12T09:30" />
              </GField>
              <GField label="Large">
                <GInput size="lg" placeholder="A 17/300 pill field" />
              </GField>
            </div>
            <div className="mt-6 grid gap-5">
              <GSearch value={query} onValueChange={setQuery} placeholder="Search (Escape clears)" aria-label="Search the gallery" />
              <GAsk label="Ask" value={ask} onValueChange={setAsk} placeholder="Ask anything (send needs 3 characters)" onSubmit={(v) => toast({ text: `Fixture: would send "${v}"` })} />
              <GAsk label="Ask busy" value="Summarise this week" onValueChange={() => {}} busy />
              <GAsk label="Ask error" value="Draft a reply" onValueChange={() => {}} error="Fixture: the model is not configured" />
              <GAsk label="Filter deals" mode="filter" value={filter} onValueChange={setFilter} placeholder="Filter deals by name" />
              <p className="text-[11.5px]">{filtered.length} of {DEALS.length} fixture deals match.</p>
            </div>
            <div className="mt-6" role="radiogroup" aria-label="Fill preview">
              <GEyebrow>GSwatch (the picker passes each accent&apos;s hero-a; here, this accent&apos;s three heroes)</GEyebrow>
              <div className="mt-3 flex gap-3">
                {(['a', 'b', 'c'] as const).map((k) => (
                  <GSwatch key={k} name={`Hero ${k.toUpperCase()}`} fill={`var(--g-hero-${k})`} selected={swatch === k} onSelect={() => setSwatch(k)} />
                ))}
              </div>
            </div>
          </GCard>
        </Section>

        <Section id="lists" title="Lists and tables" caption="GList, GRow, GTable (sortable, sticky), GKeyValue">
          <div className="grid gap-5 md:grid-cols-2">
            <GCard>
              <GList label="Fixture deals">
                {DEALS.map((d) => (
                  <GRow
                    key={d.id}
                    leading={<GCheck checked={selected === d.id} onCheckedChange={() => setSelected(d.id)} aria-label={`Select ${d.name}`} />}
                    title={d.name}
                    meta={d.stage}
                    trailing={d.value == null ? '—' : `$${d.value.toLocaleString('en-US')}`}
                    onClick={() => setSelected(d.id)}
                    selected={selected === d.id}
                  />
                ))}
                <GRow title="Archived deal" meta="Read only" disabled disabledReason="Fixture: archived" onClick={() => {}} />
              </GList>
            </GCard>
            <GCard>
              <GList divided label="Divided list">
                <GRow title="Divided rows" meta="--g-line between rows" />
                <GRow title="Static row" meta="No action, no hover" trailing="12" />
                <GRow title="Link row" meta="Opens a section" href="/dev/g-kit#states" />
              </GList>
              <div className="mt-5">
                <GKeyValue
                  items={[
                    { label: 'Owner', value: 'Fixture owner' },
                    { label: 'Created', value: 'Oct 2, 2026' },
                    { label: 'Value', value: '$48,500', title: 'Fixture value' },
                  ]}
                />
              </div>
            </GCard>
          </div>
          <GCard>
            <GTable
              caption="Fixture deals"
              showCaption
              rows={DEALS}
              rowKey={(r) => r.id}
              sort={{ key: 'value', dir: 'desc' }}
              columns={[
                { key: 'name', label: 'Deal', sortValue: (r) => r.name, render: (r) => r.name },
                { key: 'stage', label: 'Stage', sortValue: (r) => r.stage, render: (r) => <GTag size="xs">{r.stage}</GTag> },
                { key: 'value', label: 'Value', numeric: true, sortValue: (r) => r.value, render: (r) => (r.value == null ? '—' : `$${r.value.toLocaleString('en-US')}`), title: 'Fixture USD' },
              ]}
            />
            <div className="mt-5">
              <GTable caption="Empty table" rows={[] as Deal[]} rowKey={(r) => r.id} columns={[{ key: 'name', label: 'Deal', render: (r) => r.name }]} empty={<GEmpty inline title="No deals yet" body="Deals appear when the first lead converts." />} />
            </div>
          </GCard>
        </Section>

        <Section id="states" title="States" caption="GEmpty kinds, GProgress, GSkeleton, GErrorLine, GSampleBadge">
          <div className="grid gap-5 md:grid-cols-3">
            <GCard>
              <GEmpty title="No campaigns yet" body="Start one to see its runs here." action={{ label: 'New campaign', onClick: () => toast({ text: 'Would open the composer' }) }} />
            </GCard>
            <GCard>
              <GEmpty kind="unconfigured" title="Payments are not connected" body="Connect Stripe to see revenue." setupHref="/integrations#stripe" setupHint="Fixture: STRIPE_SECRET_KEY is not set" />
            </GCard>
            <GCard>
              <GEmpty kind="error" title="Could not load runs" body="Fixture: the request timed out." onRetry={() => toast({ text: 'Fixture: retried', tone: 'ok' })} />
            </GCard>
          </div>
          <GCard>
            <div className="grid gap-4">
              <GProgress text="Fixture: generating three angles" />
              <div className="grid grid-cols-[1fr_120px_60px] items-center gap-4">
                <GSkeleton height={14} />
                <GSkeleton height={34} shape="pill" />
                <GSkeleton width={40} height={40} shape="pill" />
              </div>
              <GSkeleton height={120} shape="nested" />
              <GErrorLine>Fixture: the last sync failed 3 minutes ago.</GErrorLine>
              <GErrorLine align="left" onRetry={() => toast({ text: 'Fixture: retried' })}>
                Fixture: with a retry
              </GErrorLine>
              <div>
                <GSampleBadge fixture="kit gallery" />
              </div>
            </div>
          </GCard>
        </Section>

        <Section id="overlays" title="Overlays" caption="GSheet, GDrawer (back stack, docked, swipe), GDialog, GPopover, GMenu, GTooltip, GToast">
          <GCard>
            <div className="flex flex-wrap items-center gap-3">
              <GButton onClick={() => setSheet(true)}>Open sheet</GButton>
              <GButton onClick={() => setDrawer(1)}>Open drawer</GButton>
              <GButton onClick={() => setDock((v) => !v)}>{dock ? 'Close docked drawer' : 'Open docked drawer'}</GButton>
              <GButton variant="danger" onClick={() => setDialog(true)}>
                <Trash2 size={14} strokeWidth={1.75} aria-hidden /> Delete project
              </GButton>
              <GPopover
                open={pop}
                onOpenChange={setPop}
                label="Rename"
                trigger={(p) => (
                  <GButton {...p}>
                    <Pencil size={14} strokeWidth={1.75} aria-hidden /> Rename
                  </GButton>
                )}
              >
                <div className="grid gap-3">
                  <GField label="New name">
                    <GInput defaultValue="Northwind desk lamp" />
                  </GField>
                  <div className="flex justify-end gap-2">
                    <GButton variant="sm" onClick={() => setPop(false)}>Cancel</GButton>
                    <GButton variant="ink" size="sm" onClick={() => { setPop(false); toast({ text: 'Fixture: renamed', tone: 'ok' }); }}>Save</GButton>
                  </div>
                </div>
              </GPopover>
              <GMenu
                label="Deal actions"
                trigger={(p) => (
                  <GButton variant="icon" aria-label="More actions" {...p}>
                    <MoreHorizontal size={16} strokeWidth={1.75} aria-hidden />
                  </GButton>
                )}
                items={[
                  { id: 'copy', label: 'Copy link', icon: <Copy size={14} strokeWidth={1.75} />, onSelect: () => toast({ text: 'Fixture: link copied', tone: 'ok' }) },
                  { id: 'archive', label: 'Archive', icon: <Archive size={14} strokeWidth={1.75} />, disabled: true, disabledReason: 'Fixture: already archived' },
                  { id: 'del', label: 'Delete', icon: <Trash2 size={14} strokeWidth={1.75} />, danger: true, onSelect: () => setDialog(true) },
                ]}
              />
              <GTooltip content="Fixture: last synced 3 minutes ago from the seed store">
                <GButton variant="sm">Hover or focus me</GButton>
              </GTooltip>
              <GButton
                variant="sm"
                onClick={() => toast({ text: 'Removed from queue', tone: 'ok', ms: 5000, action: { label: 'Undo', onClick: () => toast({ text: 'Restored' }) } })}
              >
                Toast with Undo
              </GButton>
              <GButton variant="sm" onClick={() => toast({ text: 'Fixture: could not save', tone: 'err' })}>
                Error toast
              </GButton>
            </div>
          </GCard>
          <GTabsAnchorDemo />
        </Section>
      </div>

      <GSheet
        open={sheet}
        onClose={() => setSheet(false)}
        title="Deal details"
        caption="Detail moves into sheets, not onto the surface"
        width={720}
        footer={
          <>
            <GButton onClick={() => setSheet(false)}>Close</GButton>
            <GButton variant="ink" onClick={() => { setSheet(false); toast({ text: 'Fixture: saved', tone: 'ok' }); }}>Save</GButton>
          </>
        }
      >
        <GKeyValue items={DEALS.map((d) => ({ key: d.id, label: d.name, value: d.value == null ? '—' : `$${d.value.toLocaleString('en-US')}` }))} />
      </GSheet>
      <GDrawer
        open={drawer > 0}
        onClose={() => setDrawer(0)}
        title={drawer > 1 ? 'Works with' : 'Agent'}
        onBack={drawer > 1 ? () => setDrawer(drawer - 1) : undefined}
        width={480}
      >
        <p className="text-[13.5px]">Level {drawer} of the back stack. On touch, swipe right 80px to close.</p>
        <div className="mt-4">
          <GList>
            <GRow title="Open the next level" meta="Pushes onto the back stack" onClick={() => setDrawer(drawer + 1)} />
          </GList>
        </div>
      </GDrawer>
      <GDrawer open={dock} onClose={() => setDock(false)} title="Inspector" modal={false} width={360}>
        <p className="text-[13.5px]">Docked: no scrim, no focus trap; the page stays usable. Escape closes it.</p>
      </GDrawer>
      <GDialog
        open={dialog}
        onClose={() => setDialog(false)}
        title="Delete this project?"
        tone="danger"
        confirmLabel="Delete"
        onConfirm={() => { setDialog(false); toast({ text: 'Fixture: nothing was deleted (gallery)' }); }}
      >
        Its runs and drafts go too. This cannot be undone.
      </GDialog>
    </GPage>
  );
}

function GTabsAnchorDemo() {
  const ids = ['anchor-one', 'anchor-two', 'anchor-three'];
  return (
    <GCard>
      <GTabs variant="anchor" label="Anchor sections" tabs={ids.map((id, i) => ({ value: id, label: ['One', 'Two', 'Three'][i] }))} />
      <div className="mt-4 grid gap-4">
        {ids.map((id, i) => (
          <GWell key={id}>
            <div id={id} className="scroll-mt-24 py-6 text-[13.5px]">
              Anchor section {['one', 'two', 'three'][i]}: the tab tracks the section in view.
            </div>
          </GWell>
        ))}
      </div>
    </GCard>
  );
}
