'use client';

/**
 * /funnel under the GladOS style (glados-ui SPEC S02 §4.2, §5.5; X5). The old
 * layout, new widgets: Touches · New leads + Converted · Revenue on top, the
 * Pipeline (GFunnelStep) and the Insights hero below, the sources strip
 * under them. Same props, URL state and views as the Classic FunnelDashboard.
 * Cut (FR-F3, FR-F7): the five kebabs (their one unique action is each card's
 * ↗ button), the insight dashes and glass shapes, the y axis and caps.
 */

import { useEffect, useMemo, useState, useTransition } from 'react';
import { useRouter } from 'next/navigation';
import { ArrowUpRight, Calendar, Check, ChevronDown, ExternalLink, Link2, RefreshCw } from 'lucide-react';
import {
  GAsk, GBigNumber, GButton, GCard, GDelta, GDisplay, GDotMatrix, GDotNumber, GDrawer, GDriverBars, GEmpty, GFunnelStep, GHeroCard, GKeyValue,
  GLineChart, GMenu, GPage, GSampleBadge, GSectionHeader, GSegmented, GTable, GTag, type GColumn, type GTagTone,
} from '@/components/g';
import { CHANNEL_GLYPHS, DECAY_DAYS, FUNNEL_STAGES, decayFactor, funnelSummary, journeyMeta } from '@/lib/funnel';
import type { FunnelJourney, FunnelStage } from '@/lib/schemas';
import type { DashboardProps } from './FunnelDashboard';
import { FunnelChannelsG } from './FunnelChannelsG';
import { FunnelLinksG } from './FunnelLinksG';
import { LANES, type Compare, type Grain, type Lane, type Period, type View } from './state';
import {
  DAY, LANE_LABEL, PAGE_ROWS, PERIODS, WEEKDAYS, WEEKDAY_INITIALS, defaultStage as leakStage, fmtDate, fmtDay, laneOf, periodStats, stageIndex, stageLabel,
  touchSeries as buildTouchSeries, worstStep as findWorstStep,
} from './model';

const usd = (n: number) => `$${Math.round(n).toLocaleString('en-US')}`;
const num = (n: number) => n.toLocaleString('en-US');
const LANE_COLOR = { organic: 'series-1', ads: 'series-2', crm: 'series-3' } as const;

type Command = { token: string; hint: string; run: () => void };

export function FunnelDashboardG({ journeys, archived, nowIso, source, connectors, pushNow, saveNow, initial, sampleOffer = false, channels, trakyo }: DashboardProps) {
  const router = useRouter();
  const [refreshing, startRefresh] = useTransition();
  const now = useMemo(() => new Date(nowIso), [nowIso]);
  const summary = useMemo(() => funnelSummary(journeys), [journeys]);
  const defaultStage = useMemo(() => leakStage(summary), [summary]);

  const [view, setView] = useState<View>(initial.view ?? 'overview');
  const [period, setPeriod] = useState<Period>(initial.period ?? 90);
  const [grain, setGrain] = useState<Grain>(initial.grain ?? 'monthly');
  const [compare, setCompare] = useState<Compare>(initial.compare ?? 'previous');
  const [stage, setStage] = useState<FunnelStage>(initial.stage ?? defaultStage);
  const [segment, setSegment] = useState<FunnelStage | undefined>(initial.stage);
  const [lane, setLane] = useState<Lane | undefined>(initial.lane);
  const [selected, setSelected] = useState<string | undefined>(initial.lead);
  const [copied, setCopied] = useState(false);
  const [explore, setExplore] = useState('');
  const [note, setNote] = useState<string | null>(null);
  const [shown, setShown] = useState(PAGE_ROWS);

  // The URL mirrors the view, so a copied link reproduces it (AC02.12).
  useEffect(() => {
    const p = new URLSearchParams(window.location.search);
    const put = (k: string, v: string | number | undefined, def?: string | number) => {
      if (v === undefined || v === def) p.delete(k);
      else p.set(k, String(v));
    };
    put('view', view, 'overview');
    put('stage', stage, defaultStage);
    put('period', period, 90);
    put('grain', grain, 'monthly');
    put('compare', compare, 'previous');
    put('lane', lane);
    put('lead', selected);
    const qs = p.toString();
    const next = `${window.location.pathname}${qs ? `?${qs}` : ''}`;
    if (next !== `${window.location.pathname}${window.location.search}`) window.history.replaceState(null, '', next);
  }, [view, stage, defaultStage, period, grain, compare, lane, selected]);

  useEffect(() => setShown(PAGE_ROWS), [view, segment, lane]);

  const stats = useMemo(() => periodStats(journeys, now, period, compare), [journeys, now, period, compare]);
  const touches = useMemo(() => buildTouchSeries([...journeys, ...archived], now, grain), [journeys, archived, now, grain]);
  const conversion = summary.clients > 0 ? Math.round((summary.converted / summary.clients) * 100) : 0;
  const leak = useMemo(() => findWorstStep(summary), [summary]);
  const openValue = journeys.filter((j) => j.status === stage).reduce((s, j) => s + (j.amountUsd ?? 0), 0);

  const byId = useMemo(() => new Map([...journeys, ...archived].map((j) => [j.id, j])), [journeys, archived]);
  const archivedIds = useMemo(() => new Set(archived.map((j) => j.id)), [archived]);
  const pushList = pushNow.map((id) => byId.get(id)).filter(Boolean) as FunnelJourney[];
  const saveList = saveNow.map((id) => byId.get(id)).filter(Boolean) as FunnelJourney[];
  const selectedLead = selected ? byId.get(selected) : undefined;
  const tableJourneys = journeys.filter((j) => (!segment || j.status === segment) && (!lane || laneOf(j) === lane));
  const stageCounts = new Map<FunnelStage, number>();
  for (const j of journeys) stageCounts.set(j.status, (stageCounts.get(j.status) ?? 0) + 1);
  const laneCounts: Record<Lane, number> = { organic: 0, ads: 0, crm: 0 };
  for (const j of journeys) laneCounts[laneOf(j)]++;

  const goJourneys = (s?: FunnelStage, l?: Lane) => {
    setSegment(s);
    setLane(l);
    if (s) setStage(s);
    setView('journeys');
  };

  const commands: Command[] = [
    ...FUNNEL_STAGES.map((s) => ({ token: `/${s.id.replace('_', '-')}`, hint: `${s.label} segment`, run: () => goJourneys(s.id, lane) })),
    ...LANES.map((l) => ({ token: `/${l}`, hint: `${LANE_LABEL[l]} entries`, run: () => goJourneys(segment, l) })),
    { token: '/push', hint: 'Hot and moving', run: () => setView('push') },
    { token: '/save', hint: 'Fading toward the archive', run: () => setView('save') },
    { token: '/archive', hint: `Decayed past ${DECAY_DAYS} d`, run: () => setView('archive') },
    { token: '/overview', hint: 'Back to the board', run: () => setView('overview') },
  ];
  const slash = explore.match(/\/([\w-]*)$/);
  const matching = slash ? commands.filter((c) => c.token.slice(1).startsWith(slash[1].toLowerCase())) : [];
  const runCommand = (c: Command) => { c.run(); setExplore(''); setNote(null); };
  const submitExplore = (line: string) => {
    const hit = commands.find((c) => line.toLowerCase().includes(c.token));
    if (hit) runCommand(hit);
    else setNote('No token in that line. Type / to see where this bar can jump.');
  };

  const copyLink = async () => {
    try {
      await navigator.clipboard.writeText(window.location.href);
      setCopied(true);
      setTimeout(() => setCopied(false), 1500);
    } catch {
      window.prompt('Copy this link', window.location.href);
    }
  };

  const crm = connectors.find((c) => c.ok && (c.id === 'attio' || c.id === 'ghl'));
  const crmHref = crm ? (crm.id === 'ghl' ? 'https://app.gohighlevel.com' : 'https://app.attio.com') : undefined;
  const attioHref = connectors.find((c) => c.id === 'attio')?.href ?? '/integrations';
  const cmpLabel = stats.cmp ? `${fmtDay(stats.cmp[0], compare === 'year')} – ${fmtDay(stats.cmp[1] - DAY, compare === 'year')}` : null;
  const vsLabel = compare === 'year' ? 'vs last year' : 'vs previous period';
  const periodWindow = PERIODS.find((p) => p.id === period)?.label ?? 'Selected period';
  const compareOptions: { id: Compare; label: string }[] = [
    { id: 'previous', label: compare === 'previous' && cmpLabel ? cmpLabel : 'Previous period' },
    { id: 'year', label: compare === 'year' && cmpLabel ? cmpLabel : 'Same period last year' },
    { id: 'none', label: 'No comparison' },
  ];

  const NAV: { id: View; label: string; n?: number }[] = [
    { id: 'overview', label: 'Overview' },
    { id: 'push', label: 'Push now', n: pushList.length },
    { id: 'save', label: 'Save now', n: saveList.length },
    { id: 'journeys', label: 'Journeys', n: journeys.length },
    { id: 'archive', label: 'Archive', n: archived.length },
  ];

  const seeJourneys = (label: string, run: () => void) => (
    <GButton variant="icon" size="sm" aria-label={`See these journeys: ${label}`} title="See these journeys" onClick={run}>
      <ArrowUpRight size={15} strokeWidth={1.75} aria-hidden />
    </GButton>
  );

  const countDelta = (cur: number, prev: number) => {
    const d = cur - prev;
    return (
      <GTag tone={d > 0 ? 'ok' : d < 0 ? 'err' : 'neutral'} title={`${num(prev)} ${vsLabel.replace('vs ', 'in the ')}`}>
        {d >= 0 ? '+' : '−'}{num(Math.abs(d))} {vsLabel}
      </GTag>
    );
  };

  /* ── overview cards ── */
  const touchCard = (
    <GCard p={24} className="fn-g-touches" aria-labelledby="fn-g-touches">
      <GSectionHeader
        id="fn-g-touches"
        size={22}
        title="Touches"
        caption={grain === 'monthly' ? 'Last 6 months' : 'Last 12 weeks'}
        action={seeJourneys('Touches', () => goJourneys())}
      />
      <div className="fn-g-grain">
        <GSegmented<Grain> label="Touches chart grain" value={grain} onChange={setGrain} options={[{ value: 'monthly', label: 'Monthly' }, { value: 'weekly', label: 'Weekly' }]} />
      </div>
      {touches.series.every((v) => v === 0) ? (
        <GEmpty inline title={`No touches in the last ${grain === 'monthly' ? '6 months' : '12 weeks'}`} />
      ) : (
        <GLineChart
          variant="step"
          height={170}
          labels={touches.longLabels}
          axisLabels={touches.labels}
          series={[{ id: 'touches', label: 'Touches', values: touches.series, color: 'accent' }]}
          peakLabel={(i, v) => `${num(v)} touches · ${touches.labels[i]}`}
          tip={(i) => {
            const total = touches.series.reduce((a, b) => a + b, 0);
            return { title: touches.longLabels[i], lines: [`${num(touches.series[i])} touches`, `${total > 0 ? Math.round((touches.series[i] / total) * 100) : 0}% of the ${grain === 'monthly' ? '6 months' : '12 weeks'}`] };
          }}
          label={`Touches per ${grain === 'monthly' ? 'month' : 'week'}, active and archived journeys`}
        />
      )}
    </GCard>
  );

  const dotCard = (kind: 'leads' | 'converted') => {
    const leads = kind === 'leads';
    const n = leads ? stats.leadsIn.length : stats.convIn.length;
    const cols = leads ? stats.leadCols : stats.convCols;
    const peak = leads ? stats.leadPeak : stats.convPeak;
    const noun = leads ? 'leads' : 'conversions';
    return (
      <GCard p={20} className="fn-g-sub" aria-labelledby={`fn-g-${kind}`}>
        <GSectionHeader
          id={`fn-g-${kind}`}
          size={22}
          title={leads ? 'New leads' : 'Converted'}
          caption={periodWindow}
          action={seeJourneys(leads ? 'New leads' : 'Converted', () => goJourneys(leads ? 'first_touch' : 'converted'))}
        />
        <div className="fn-g-stat">
          <GDotNumber value={num(n)} size={5} label={`${n} ${noun}, ${periodWindow.toLowerCase()}`} />
          <div className="fn-g-dots">
            <GDotMatrix
              columns={cols.map((v, i) => ({ id: WEEKDAYS[i], label: WEEKDAY_INITIALS[i], value: v, title: `${WEEKDAYS[i]}s · ${num(v)} ${noun}` }))}
              rows={6}
              color={leads ? 'accent' : 'accent-2'}
              label={`${leads ? 'New leads' : 'Conversions'} by weekday, ${periodWindow.toLowerCase()}`}
            />
          </div>
          <div className="fn-g-stat-side">
            {peak && <GTag>Peak {peak}</GTag>}
            {stats.cmp && countDelta(n, leads ? stats.leadsPrev.length : stats.convPrev.length)}
          </div>
        </div>
      </GCard>
    );
  };

  const laneMax = Math.max(1, ...Object.values(stats.byLane));
  const revenueCard = (
    <GCard p={24} className="fn-g-revenue" aria-labelledby="fn-g-revenue">
      <GSectionHeader id="fn-g-revenue" size={22} title="Revenue" caption={periodWindow} action={seeJourneys('Revenue', () => goJourneys('converted'))} />
      <div className="fn-g-rev-num">
        <GBigNumber value={usd(stats.revenue)} title={`Revenue from deals converted in the ${periodWindow.toLowerCase()}`} />
        {stats.cmp && (stats.revenue > 0 || stats.revenuePrev > 0) && <GDelta cur={stats.revenue} prev={stats.revenuePrev} money window={vsLabel} />}
      </div>
      <GDriverBars
        labelWidth={88}
        max={laneMax}
        label="Revenue by entry lane"
        rows={LANES.map((l) => ({
          id: l,
          label: LANE_LABEL[l],
          value: stats.byLane[l],
          display: usd(stats.byLane[l]),
          tone: LANE_COLOR[l],
          title: `See the ${LANE_LABEL[l].toLowerCase()} deals that closed: ${usd(stats.byLane[l])}`,
          onClick: () => goJourneys('converted', l),
        }))}
      />
      <p className="fn-g-cap">{stats.convIn.length} closed in the period, split by how each deal entered.</p>
    </GCard>
  );

  const pipelineCard = (
    <GCard p={24} className="fn-g-pipeline" aria-labelledby="fn-g-pipeline">
      <GSectionHeader id="fn-g-pipeline" size={22} title="Pipeline" caption="All active journeys" action={seeJourneys('Pipeline', () => goJourneys(stage))} />
      <div className="fn-g-chart">
        {summary.clients === 0 ? (
          <GEmpty kind="unconfigured" title="No journeys yet." body="Connect a CRM and the funnel fills itself." setupHref={attioHref} setupHint="Key Attio or GoHighLevel and the board reads the live pipeline" />
        ) : (
          <GFunnelStep
            stages={summary.stages.map((s, i) => {
              const conv = s.conversionFromPrev;
              const value = s.stage === stage ? openValue : journeys.filter((j) => j.status === s.stage).reduce((a, j) => a + (j.amountUsd ?? 0), 0);
              return {
                id: s.stage,
                label: stageLabel(s.stage),
                count: s.total,
                conversion: conv === null ? null : conv / 100,
                tip: {
                  title: `${stageLabel(s.stage)} · ${num(s.total)}`,
                  lines: [
                    `Organic ${num(s.organic)} · Ads ${num(s.ads)}`,
                    ...(conv !== null && i > 0 ? [`Drop-off ${Math.round(100 - conv)}% from ${stageLabel(summary.stages[i - 1].stage)}`] : []),
                    ...(value > 0 ? [`Open value ${usd(value)}`] : []),
                  ],
                },
              };
            })}
            selected={stage}
            onSelect={(id) => setStage(id as FunnelStage)}
            countNoun={['journey', 'journeys']}
            tipHint="Click to focus this stage"
            label="Pipeline by stage, every active journey"
          />
        )}
      </div>
      <GAsk mode="filter" value={explore} onValueChange={(v) => { setExplore(v); setNote(null); }} onSubmit={submitExplore} label="Jump to a stage or a view with a slash token" placeholder="Type / to jump to a stage or a view" className="fn-g-ask" />
      {(matching.length > 0 || explore === '/') && (
        <div className="fn-g-slash" role="group" aria-label="Slash tokens">
          {(matching.length > 0 ? matching : commands).map((c) => (
            <GButton key={c.token} size="sm" title={c.hint} onClick={() => runCommand(c)}>{c.token}</GButton>
          ))}
        </div>
      )}
      {note && <p className="fn-g-cap">{note}</p>}
    </GCard>
  );

  const insights = (
    <GHeroCard
      variant="a"
      className="fn-g-insights"
      caption="Converted, all active journeys"
      value={summary.clients === 0 ? null : `${conversion}%`}
      label={summary.clients === 0 ? 'Nothing to read yet' : `${conversion}% converted`}
      status={summary.clients === 0 ? 'Nothing to read yet' : `${summary.converted} of ${summary.clients} leads converted`}
      title="Converted journeys over every active journey"
      strip={
        <div className="fn-g-ins glass-solid">
          <p className="fn-g-ins-head">
            {summary.clients === 0
              ? 'Connect Attio or GHL and this card starts reading the pipeline.'
              : leak
                ? `The leak is ${stageLabel(leak.from)} → ${stageLabel(leak.to)}, ${leak.drop}% drop at that step.`
                : 'No step stands out as the leak.'}
          </p>
          {summary.clients > 0 && (
            <p className="fn-g-cap">
              {pushList.length} hot lead{pushList.length === 1 ? '' : 's'} in motion, {saveList.length} fading toward the archive, {archived.length} already there.
            </p>
          )}
          <div className="fn-g-actions">
            {leak && <GButton size="sm" onClick={() => goJourneys(leak.from)}>Who is stuck at {stageLabel(leak.from).toLowerCase()}</GButton>}
            {saveList.length > 0 && <GButton size="sm" onClick={() => setView('save')}>Save the {saveList.length} fading</GButton>}
          </div>
        </div>
      }
    />
  );

  /* ── tables ── */
  const quietTone = (j: FunnelJourney): GTagTone => {
    const meta = journeyMeta(j, now);
    if (meta.state === 'stalled') return 'err';
    return decayFactor(meta.daysSinceLastTouch, j.status) > 0 ? 'warn' : 'neutral';
  };
  const quietTitle = (j: FunnelJourney) => {
    const meta = journeyMeta(j, now);
    const d = decayFactor(meta.daysSinceLastTouch, j.status);
    return d > 0 ? `Fading: ${Math.round(d * 100)}% of the way to the archive` : meta.state === 'stalled' ? 'Stalled: quiet past a week before converting' : 'Days since the last touch';
  };
  const leadCol: GColumn<FunnelJourney> = {
    key: 'lead',
    label: 'Lead',
    sortValue: (j) => j.person ?? j.name,
    render: (j) => (
      <span className="fn-g-lead">
        <span className="fn-g-lead-name">{j.person ?? j.name}</span>
        {j.company && <span className="fn-g-cap">{j.company}</span>}
      </span>
    ),
  };
  const stageCol: GColumn<FunnelJourney> = { key: 'stage', label: 'Stage', sortValue: (j) => stageIndex(j.status), render: (j) => <span className="fn-g-muted">{stageLabel(j.status)}</span> };
  const quietCol: GColumn<FunnelJourney> = {
    key: 'quiet',
    label: 'Quiet',
    numeric: true,
    sortValue: (j) => journeyMeta(j, now).daysSinceLastTouch,
    title: 'Days since the last touch',
    render: (j) => <GTag tone={quietTone(j)} title={quietTitle(j)}>{journeyMeta(j, now).daysSinceLastTouch} d</GTag>,
  };
  const likelyCol: GColumn<FunnelJourney> = {
    key: 'likelihood',
    label: 'Likelihood',
    numeric: true,
    sortValue: (j) => j.likelihood,
    render: (j) => (
      <span className="fn-g-likely">
        <span className="fn-g-likely-bar"><i style={{ width: `${Math.max(0, Math.min(100, j.likelihood))}%` }} /></span>
        {j.likelihood}%
      </span>
    ),
  };
  const valueCol: GColumn<FunnelJourney> = {
    key: 'value',
    label: 'Value',
    numeric: true,
    sortValue: (j) => j.amountUsd ?? 0,
    render: (j) => ((j.amountUsd ?? 0) > 0 ? <span className={j.status === 'converted' ? undefined : 'fn-g-muted'}>{usd(j.amountUsd ?? 0)}{j.status === 'converted' ? '' : ' open'}</span> : <span className="fn-g-muted">—</span>),
  };
  const contactCol: GColumn<FunnelJourney> = { key: 'contact', label: 'Contact', render: (j) => <Contact j={j} /> };
  const lastTouchCol: GColumn<FunnelJourney> = {
    key: 'last',
    label: 'Last touch',
    render: (j) => {
      const t = j.touches[j.touches.length - 1];
      return t ? (
        <span className="fn-g-last" title={`${stageLabel(t.stage)} · via ${t.source} · ${t.at}`}>
          <span className="fn-g-glyph" aria-hidden>{CHANNEL_GLYPHS[t.channel] ?? '·'}</span>
          <span className="fn-g-last-text">{t.label}</span>
          <span className="fn-g-cap">{fmtDay(Date.parse(`${t.at}T00:00:00Z`))} · {j.touches.length} touch{j.touches.length === 1 ? '' : 'es'}</span>
        </span>
      ) : <span className="fn-g-muted">—</span>;
    },
  };
  const entryCol: GColumn<FunnelJourney> = { key: 'entry', label: 'Entry', sortValue: (j) => laneOf(j), render: (j) => <span className="fn-g-muted">{LANE_LABEL[laneOf(j)]}</span> };

  const table = (rows: FunnelJourney[], cols: GColumn<FunnelJourney>[], caption: string, empty: string) => (
    <GTable
      caption={caption}
      columns={cols}
      rows={rows}
      rowKey={(j) => j.id}
      onRowClick={(j) => setSelected(j.id)}
      rowLabel={(j) => `Open ${j.person ?? j.name}`}
      selectedKey={selected ?? null}
      empty={empty}
      sticky={false}
      className="fn-g-table"
    />
  );

  return (
    <GPage className="fn-g">
      <div className="fn-g-top">
        <GSegmented<View>
          label="Funnel views"
          size="md"
          value={view}
          onChange={setView}
          options={NAV.map((n) => ({ value: n.id, label: <>{n.label}{n.n !== undefined && <span className="fn-g-navn">{n.n}</span>}</> }))}
          className="fn-g-nav"
        />
        <div className="fn-g-top-actions">
          <GButton variant="icon" aria-label="Refresh from the sources" title="Re-read the CRM and local journeys" busy={refreshing} onClick={() => startRefresh(() => router.refresh())}>
            <RefreshCw size={15} strokeWidth={1.75} aria-hidden />
          </GButton>
          {crmHref ? (
            <GButton href={crmHref} external title={`Open ${crm?.name} in a new tab`}>
              <ExternalLink size={13} strokeWidth={1.75} aria-hidden /> Open {crm?.name}
            </GButton>
          ) : (
            <GButton size="sm" href={attioHref} title="Key Attio or GoHighLevel and the board reads the live pipeline">Connect a CRM</GButton>
          )}
        </div>
      </div>

      <div className="fn-g-title">
        <div className="fn-g-title-l">
          <GDisplay>Funnel</GDisplay>
          <GButton variant="icon" size="sm" onClick={copyLink} title="Copy a link to exactly this view" aria-label="Copy a link to this view">
            {copied ? <Check size={14} strokeWidth={1.75} aria-hidden /> : <Link2 size={14} strokeWidth={1.75} aria-hidden />}
          </GButton>
          {copied && <span className="fn-g-cap" role="status">Link copied</span>}
          {source.kind === 'sample' && <GSampleBadge fixture="components/funnel/sample.ts" />}
        </div>
        <div className="fn-g-filters">
          <GMenu
            label="Period"
            align="start"
            items={PERIODS.map((p) => ({ id: String(p.id), label: p.label, icon: p.id === period ? <Check size={13} aria-hidden /> : undefined, onSelect: () => setPeriod(p.id) }))}
            trigger={(t) => (
              <GButton {...t} aria-label={`Period: ${periodWindow}`}>
                <Calendar size={13} strokeWidth={1.75} aria-hidden /> {periodWindow} <ChevronDown size={13} strokeWidth={1.75} aria-hidden />
              </GButton>
            )}
          />
          <span className="fn-g-cap">compared to</span>
          {period === 0 ? (
            <span className="fn-g-cap" title="All time has no earlier window to compare against">nothing</span>
          ) : (
            <GMenu
              label="Comparison window"
              align="end"
              items={compareOptions.map((o) => ({ id: o.id, label: o.label, icon: o.id === compare ? <Check size={13} aria-hidden /> : undefined, onSelect: () => setCompare(o.id) }))}
              trigger={(t) => (
                <GButton {...t} aria-label={`Comparison window: ${compareOptions.find((o) => o.id === compare)?.label}`}>
                  <Calendar size={13} strokeWidth={1.75} aria-hidden /> {compareOptions.find((o) => o.id === compare)?.label} <ChevronDown size={13} strokeWidth={1.75} aria-hidden />
                </GButton>
              )}
            />
          )}
        </div>
      </div>

      {view === 'overview' && (
        <div className="fn-g-grid">
          <div className="fn-g-c3">{touchCard}</div>
          <div className="fn-g-c4">{dotCard('leads')}{dotCard('converted')}</div>
          <div className="fn-g-c2">{revenueCard}</div>
          <div className="fn-g-c1">{pipelineCard}</div>
          <div className="fn-g-c5">{insights}</div>
          {trakyo && <div className="fn-g-c6"><FunnelLinksG configured={trakyo.configured} setupHref={trakyo.setupHref} /></div>}
          {channels && <div className="fn-g-c7"><FunnelChannelsG card={channels} /></div>}
        </div>
      )}

      {(view === 'push' || view === 'save') && (
        <div className="fn-g-two">
          {(
            [
              { id: 'push', title: 'Push now', sub: 'Hot and moving. Close them.', tone: 'ok', list: pushList, empty: 'No hot leads in motion right now' },
              { id: 'save', title: 'Save now', sub: 'Fading toward the archive, highest likelihood first', tone: 'err', list: saveList, empty: 'Nothing fading. Every lead is fresh.' },
            ] as const
          ).map((q) => (
            <GCard key={q.id} p={24} className="fn-g-panel" data-quiet={view !== q.id || undefined} aria-labelledby={`fn-g-${q.id}`}>
              <GSectionHeader id={`fn-g-${q.id}`} size={22} title={<span className="fn-g-panel-title"><GTag tone={q.tone} dot size="xs">{q.list.length}</GTag>{q.title}</span>} caption={q.sub} />
              {table(q.list, [leadCol, stageCol, quietCol, likelyCol, valueCol, contactCol], q.title, q.empty)}
            </GCard>
          ))}
        </div>
      )}

      {view === 'journeys' && (
        <GCard p={24} className="fn-g-panel" aria-labelledby="fn-g-journeys">
          <GSectionHeader id="fn-g-journeys" size={22} title="Journeys" caption={`${tableJourneys.length} of ${journeys.length} leads · click a row for the dossier`} />
          <div className="fn-g-segs">
            <GSegmented<string>
              label="Stage"
              value={segment ?? 'all'}
              onChange={(v) => { if (v === 'all') setSegment(undefined); else { setSegment(v as FunnelStage); setStage(v as FunnelStage); } }}
              options={[{ value: 'all', label: `All stages ${journeys.length}` }, ...FUNNEL_STAGES.map((s) => ({ value: s.id, label: `${s.label} ${stageCounts.get(s.id) ?? 0}` }))]}
            />
            <GSegmented<string>
              label="Entry"
              value={lane ?? 'any'}
              onChange={(v) => setLane(v === 'any' ? undefined : (v as Lane))}
              options={[{ value: 'any', label: 'Any entry' }, ...LANES.map((l) => ({ value: l, label: `${LANE_LABEL[l]} ${laneCounts[l]}` }))]}
            />
          </div>
          {table(tableJourneys.slice(0, shown), [leadCol, stageCol, quietCol, likelyCol, entryCol, valueCol, lastTouchCol, contactCol], 'Journey data', 'No leads match those filters.')}
          {tableJourneys.length > shown && (
            <div className="fn-g-more">
              <GButton size="sm" onClick={() => setShown((n) => n + PAGE_ROWS)}>Show {Math.min(PAGE_ROWS, tableJourneys.length - shown)} more</GButton>
              <span className="fn-g-cap">{shown} of {tableJourneys.length} shown</span>
            </div>
          )}
        </GCard>
      )}

      {view === 'archive' && (
        <GCard p={24} className="fn-g-panel" aria-labelledby="fn-g-archive">
          <GSectionHeader id="fn-g-archive" size={22} title="Archive" caption={`Quiet past ${DECAY_DAYS} days · click a row for the dossier`} />
          {table(archived.slice(0, shown), [leadCol, { ...stageCol, label: 'Reached' }, quietCol, likelyCol, lastTouchCol, contactCol], 'Archived journeys', `Nothing decayed. No lead has sat quiet past ${DECAY_DAYS} days.`)}
          {archived.length > shown && (
            <div className="fn-g-more">
              <GButton size="sm" onClick={() => setShown((n) => n + PAGE_ROWS)}>Show {Math.min(PAGE_ROWS, archived.length - shown)} more</GButton>
            </div>
          )}
        </GCard>
      )}

      <div className="fn-g-sources">
        {source.kind === 'live' ? (
          <GTag tone="ok" dot>Live · {source.label}</GTag>
        ) : source.kind === 'db' ? (
          <GTag tone="warn" dot>Local snapshot · CRM not keyed</GTag>
        ) : (
          <GTag tone="warn" dot>Sample data · CRM not keyed and no journeys stored</GTag>
        )}
        {source.kind !== 'live' && <GButton size="sm" href={attioHref} title="Key Attio or GHL to replace this with the live pipeline">Set up</GButton>}
        {sampleOffer && <GButton size="sm" href="/funnel?sample=1" title="See the page on a badged sample set; nothing is stored">Preview with sample data</GButton>}
        {source.kind === 'sample' && <GButton size="sm" href="/funnel" title="Back to the empty, honest state">Leave sample data</GButton>}
        <span className="fn-g-sep" aria-hidden />
        {connectors.map((c) =>
          c.ok ? (
            <GTag key={c.id} tone="ok" dot title={c.detail}>{c.name}{c.count != null ? ` ${c.count}` : ''}</GTag>
          ) : (
            <GButton key={c.id} size="sm" href={c.href} title={`${c.detail} · set up`}>
              <span className="fn-g-off" aria-hidden /> {c.name}
            </GButton>
          ),
        )}
      </div>

      {selectedLead && (
        <LeadDrawerG
          j={selectedLead}
          now={now}
          archived={archivedIds.has(selectedLead.id)}
          onClose={() => setSelected(undefined)}
          onStage={(s) => {
            setSelected(undefined);
            setStage(s);
            setView('overview');
          }}
        />
      )}
    </GPage>
  );
}

/** Contact links: plain prose links (mail, WhatsApp, CRM), never buttons. */
function Contact({ j, full = false }: { j: FunnelJourney; full?: boolean }) {
  const digits = j.phone?.replace(/[^\d]/g, '');
  const isGhl = j.url?.includes('gohighlevel');
  const crm = isGhl ? 'GoHighLevel' : 'Attio';
  const links: { href: string; label: string; ext?: boolean; title?: string }[] = [];
  if (j.email) links.push({ href: `mailto:${j.email}`, label: full ? j.email : 'Email', title: j.email });
  if (digits) links.push({ href: `https://wa.me/${digits}`, label: full ? `WhatsApp ${j.phone}` : 'WhatsApp', ext: true, title: j.phone ?? undefined });
  if (j.phone && full) links.push({ href: `sms:${j.phone}`, label: 'SMS' });
  if (j.url) links.push({ href: j.url, label: full ? `Open in ${crm}` : crm, ext: true });
  if (j.linkedin && full) links.push({ href: j.linkedin, label: 'LinkedIn', ext: true });
  if (links.length === 0) return <span className="fn-g-muted">{full ? 'No contact channel on record' : '—'}</span>;
  return (
    <ul className="fn-g-contact" data-full={full || undefined}>
      {links.map((l) => (
        <li key={l.href}>
          <a href={l.href} target={l.ext ? '_blank' : undefined} rel={l.ext ? 'noopener noreferrer' : undefined} title={l.title}>{l.label}</a>
        </li>
      ))}
    </ul>
  );
}

function LeadDrawerG({ j, now, archived, onClose, onStage }: { j: FunnelJourney; now: Date; archived: boolean; onClose: () => void; onStage: (s: FunnelStage) => void }) {
  const meta = journeyMeta(j, now);
  const decay = decayFactor(meta.daysSinceLastTouch, j.status);
  const reached = stageIndex(j.status);
  const lane = laneOf(j);
  const status = archived
    ? `Archived · quiet ${meta.daysSinceLastTouch} d, past the ${DECAY_DAYS}-day line`
    : meta.state === 'converted'
      ? 'Converted'
      : meta.state === 'stalled'
        ? `Stalled · quiet ${meta.daysSinceLastTouch} d`
        : decay > 0
          ? `Fading · quiet ${meta.daysSinceLastTouch} d, ${Math.round(decay * 100)}% of the way to the archive`
          : `Active · quiet ${meta.daysSinceLastTouch} d`;
  return (
    <GDrawer open onClose={onClose} title={j.person ?? j.name} width={460} className="fn-g-drawer">
      <p className="fn-g-cap fn-g-dsub">{[j.company, j.role].filter(Boolean).join(' · ') || j.name}</p>
      <div className="fn-g-reached" role="img" aria-label={`Reached ${stageLabel(j.status)}`}>
        {FUNNEL_STAGES.map((s, i) => <i key={s.id} data-on={i <= reached || undefined} title={s.label} />)}
      </div>
      <div className="fn-g-dsec">
        <GSegmented<string>
          label="Focus a stage on the board"
          value={j.status}
          onChange={(v) => onStage(v as FunnelStage)}
          options={FUNNEL_STAGES.map((s) => ({ value: s.id, label: s.label, title: `${s.label}: focus this stage on the board` }))}
        />
      </div>
      <GKeyValue
        className="fn-g-facts"
        items={[
          { label: 'Status', value: status },
          { label: 'Relationship', value: <span className="fn-g-cap-ink">{j.relationship}</span> },
          { label: 'Likelihood', value: `${j.likelihood}%` },
          { label: 'Entry', value: LANE_LABEL[lane] },
          { label: 'Value', value: `${(j.amountUsd ?? 0) > 0 ? usd(j.amountUsd ?? 0) : '—'}${j.product ? ` · ${j.product}` : ''}${j.status !== 'converted' && (j.amountUsd ?? 0) > 0 ? ' (open)' : ''}` },
          { label: 'First seen', value: fmtDate(j.touches[0]?.at ?? j.createdAt) },
          { label: 'Deal', value: j.name },
        ]}
      />
      <h3 className="fn-g-dlabel">Contact</h3>
      <Contact j={j} full />
      <h3 className="fn-g-dlabel">Journey · {j.touches.length} touches</h3>
      <ol className="fn-g-rail">
        {j.touches.map((t) => (
          <li key={t.id}>
            <span className="fn-g-rail-dot" aria-hidden />
            <span className="fn-g-rail-when">{fmtDate(t.at)}</span>
            <span className="fn-g-rail-what"><span className="fn-g-glyph" aria-hidden>{CHANNEL_GLYPHS[t.channel] ?? '·'}</span> {t.label}</span>
            <span className="fn-g-cap">{stageLabel(t.stage)} · {t.source}</span>
          </li>
        ))}
        {j.status !== 'converted' && <li data-next=""><span className="fn-g-rail-dot" aria-hidden /><span className="fn-g-cap">Next touch is yours</span></li>}
      </ol>
    </GDrawer>
  );
}
