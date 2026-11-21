'use client';

/**
 * /trading under the Classic style (pages/trading.md §7): PageHeader, bordered
 * Classic cards on a 12-column grid, os-* tokens only, Geist Mono for times
 * and ids. Same model and behaviour as the GladOS tree (use-trading.ts).
 * Monitor only: no control here places an order (T13).
 */
import { useId, useMemo, useState, type PointerEvent, type ReactNode } from 'react';
import { Copy, RefreshCw, X } from 'lucide-react';
import { PageHeader } from '@/components/PageHeader';
import { SetupLink } from '@/components/SetupLink';
import { Badge, type BadgeTone } from '@/components/terminal';
import { useDialog } from '@/components/ui/use-dialog';
import { useMeltClose } from '@/lib/use-melt';
import { Layer } from '@/components/g/Overlays';
import { areaPaths, extent, nearestIndex, seriesPaths, spreadX } from '@/lib/chart-geometry';
import {
  ago, day, logRows, qty, signedUsd, stamp, toneOf, tradeLine, usd, whoOf, type ActivityFilter, type TradingModel,
} from '@/lib/trading-model';
import type { TradeActivity, TradingPosition } from '@/lib/schemas';
import { useTrading, type TradingUi } from './use-trading';
import type { TradingTreeProps } from './TradingG';

const statusTone: Record<TradeActivity['status'], BadgeTone> = { filled: 'ok', pending: 'default', cancelled: 'default', rejected: 'err' };
const statusLabel: Record<TradeActivity['status'], string> = { filled: 'Filled', pending: 'Pending', cancelled: 'Cancelled', rejected: 'Rejected' };
const verdictTone = (v: string): BadgeTone => (/buy|enter|long|take/i.test(v) ? 'ok' : /sell|exit|trim/i.test(v) ? 'warn' : /watch|hold/i.test(v) ? 'accent' : 'default');
const cap = (s: string) => s.charAt(0).toUpperCase() + s.slice(1);
const acctName = (id: string) => (id === 'agentic' ? 'Agentic' : cap(id));

function Pnl({ n, cents = true }: { n: number | null; cents?: boolean }) {
  if (n === null) return <span className="tabular-nums text-os-dim">—</span>;
  return (
    <span className="trd-c-pnl tabular-nums" data-tone={toneOf(n)}>
      {signedUsd(n, cents)}
    </span>
  );
}

function Card({ title, sub, right, children, className = '', id }: { title: string; sub?: ReactNode; right?: ReactNode; children: ReactNode; className?: string; id?: string }) {
  const hid = useId();
  return (
    <section aria-labelledby={id ?? hid} className={`trd-c-card rounded-lg-t border border-os-border bg-os-surface ${className}`}>
      <header className="trd-c-card-head">
        <div className="min-w-0">
          <h2 id={id ?? hid} className="text-[14px] font-medium text-os-text">
            {title}
          </h2>
          {sub != null && <p className="mt-0.5 text-[12px] text-os-muted">{sub}</p>}
        </div>
        {right}
      </header>
      {children}
    </section>
  );
}

function Empty({ children, sub }: { children: ReactNode; sub?: ReactNode }) {
  return (
    <div className="px-5 py-7 text-center">
      <p className="text-[13px] text-os-muted">{children}</p>
      {sub != null && <p className="mx-auto mt-1 max-w-[420px] text-[12px] text-os-dim">{sub}</p>}
    </div>
  );
}

export function TradingClassic({ model: m, pushPath, setup }: TradingTreeProps) {
  const ui = useTrading(m, pushPath);
  const p = m.payload;

  const right = (
    <div className="flex flex-wrap items-center justify-end gap-2">
      {m.fixture && (
        <span data-sample-badge="" title={`Sample data: ${m.fixture}`}>
          <Badge tone="warn">Sample</Badge>
        </span>
      )}
      {m.feedPill ? (
        <span title={m.feedPill.title}>
          <Badge tone={m.feedPill.tone}>
            <span className={`dot ${m.feedPill.tone === 'ok' ? 'ok' : 'warn'}`} aria-hidden />
            {m.feedPill.text}
          </Badge>
        </span>
      ) : (
        <SourceTag label="Robinhood feed" href={setup.robinhood} hint="Have the Markets agent push to /api/trading/snapshot with TRADING_PUSH_TOKEN" />
      )}
      {m.wallet === 'unconfigured' && <SourceTag label="Phantom wallet" href={setup.phantom} hint="Set PHANTOM_WALLET_ADDRESS (a public address, not a key)" />}
      {!m.unconfigured && (
        <button type="button" onClick={ui.refresh} disabled={ui.refreshing} className="c-btn" title="Reload from the store; the wallet read is cached 60 s">
          <RefreshCw size={13} strokeWidth={1.7} aria-hidden className={ui.refreshing ? 'trd-c-spin' : undefined} />
          Refresh
        </button>
      )}
    </div>
  );

  return (
    <div className="trd-c">
      <PageHeader title="Trading" eyebrow={m.subtitle ?? m.copy.subtitleEmpty} right={right} />

      {m.unconfigured ? (
        <ConnectCard ui={ui} setupHref={setup.robinhood} />
      ) : (
        <div className="trd-c-grid">
          {m.feed === 'none' ? (
            <div className="trd-c-span-8">
              <ConnectCard ui={ui} setupHref={setup.robinhood} />
            </div>
          ) : (
            <Sleeve m={m} ui={ui} />
          )}
          <Accounts m={m} setupPhantom={setup.phantom} />
          {m.feed !== 'none' && (
            <>
              <Reasoning m={m} ui={ui} />
              <Positions m={m} ui={ui} />
              <NextMove m={m} />
              <OpenOrders m={m} />
              <TradeLog m={m} ui={ui} />
            </>
          )}
          <Limits m={m} ui={ui} />
          <p className="trd-c-span-12 text-center text-[12px] text-os-dim">Only the agentic sleeve can be traded by an agent. Wallet read from a public RPC.</p>
        </div>
      )}

      {ui.openTrade && <TradeDialog trade={ui.openTrade} onClose={() => ui.setOpenTrade(null)} />}
      {ui.positionsOpen && (
        <Modal title="Positions" sub={`${p.positions.length} open · ${usd(m.investedUsd)} invested`} onClose={() => ui.setPositionsOpen(false)} wide>
          <PositionsTable rows={p.positions} />
        </Modal>
      )}
      {ui.notesOpen && p.analysis && (
        <Modal title="Agent reasoning" sub={`${p.analysis.agent} · ${stamp(p.analysis.at)}`} onClose={() => ui.setNotesOpen(false)}>
          <p className="whitespace-pre-wrap px-5 pb-5 text-[13px] leading-relaxed text-os-text">{p.analysis.notes}</p>
        </Modal>
      )}
      {ui.autopilot.confirmOpen && (
        <Modal title="Turn on autopilot?" onClose={() => ui.autopilot.setConfirmOpen(false)}>
          <p className="px-5 text-[13px] leading-relaxed text-os-muted">{m.copy.autopilotBody}</p>
          <div className="flex justify-end gap-2 px-5 py-4">
            <button type="button" className="c-btn" onClick={() => ui.autopilot.setConfirmOpen(false)}>
              Cancel
            </button>
            <button type="button" className="c-btn c-btn-primary" disabled={ui.autopilot.switching} onClick={() => void ui.autopilot.setAutopilot(true, true)}>
              Turn on
            </button>
          </div>
        </Modal>
      )}
    </div>
  );
}

function SourceTag({ label, href, hint }: { label: string; href: string; hint: string }) {
  return (
    <span className="inline-flex items-center gap-1.5" title={hint}>
      <Badge ghost>{label} · Not configured</Badge>
      <SetupLink href={href} hint={hint} />
    </span>
  );
}

function ConnectCard({ ui, setupHref }: { ui: TradingUi; setupHref: string }) {
  return (
    <section aria-label="Connect the Markets agent feed" className="trd-c-card items-center rounded-lg-t border border-os-border bg-os-surface px-5 py-8 text-center">
      <Badge ghost>Not configured</Badge>
      <p className="mt-3 text-[14px] text-os-text">Connect the Markets agent feed. Its first push fills this page.</p>
      <p className="mx-auto mt-1 max-w-[460px] text-[12.5px] text-os-muted">
        The agent posts account snapshots, positions and trades with its push token. The token stays in the agent; only the URL goes here.
      </p>
      <div className="mx-auto mt-4 flex w-full min-w-0 max-w-[520px] items-center gap-2 rounded-[8px] border border-os-border bg-os-bg2 py-1.5 pl-3 pr-1.5">
        <code className="min-w-0 flex-1 truncate text-left font-mono text-[12px] text-os-text" data-copy="">
          {ui.pushUrl}
        </code>
        <button type="button" className="c-btn h-[28px] px-2.5 text-[12px]" onClick={ui.copyPushUrl} aria-label="Copy push URL">
          <Copy size={12} strokeWidth={1.7} aria-hidden />
          Copy
        </button>
      </div>
      <div className="mt-4 flex justify-center">
        <SetupLink href={setupHref} hint="TRADING_PUSH_TOKEN" />
      </div>
    </section>
  );
}

/** Line + area over the account's captures (lib/chart-geometry), hover readout. */
function ValueChart({ points, label }: { points: Array<{ at: string; valueUsd: number }>; label: string }) {
  const W = 640;
  const H = 180;
  const pad = 6;
  const [hover, setHover] = useState<number | null>(null);
  const values = points.map((x) => x.valueUsd);
  const geo = useMemo(() => {
    const [lo0, hi0] = extent(values, false);
    const span = hi0 - lo0 || Math.max(1, Math.abs(hi0) * 0.02);
    const lo = lo0 - span * 0.08;
    const hi = hi0 + span * 0.08;
    const xs = spreadX(values.length, pad, W - pad);
    const yOf = (v: number) => pad + (1 - (v - lo) / (hi - lo)) * (H - pad * 2);
    return { xs, yOf, line: seriesPaths(values, xs, yOf), area: areaPaths(values, xs, yOf, H) };
  }, [values]);
  if (points.length < 2) return <Empty>{points.length === 1 ? 'One capture so far. The line starts with the next push.' : 'No captures for this account yet.'}</Empty>;
  const first = points[0];
  const last = points[points.length - 1];
  const change = last.valueUsd - first.valueUsd;
  const onMove = (e: PointerEvent<SVGSVGElement>) => {
    const r = e.currentTarget.getBoundingClientRect();
    setHover(nearestIndex(geo.xs, ((e.clientX - r.left) / r.width) * W));
  };
  const h = hover !== null ? points[hover] : null;
  return (
    <div className="trd-c-chart">
      <svg viewBox={`0 0 ${W} ${H}`} preserveAspectRatio="none" role="img" aria-label={label} onPointerMove={onMove} onPointerLeave={() => setHover(null)}>
        {geo.area.map((d, i) => (
          <path key={`a${i}`} d={d} className="trd-c-area" />
        ))}
        {geo.line.map((s, i) => (
          <path key={`l${i}`} d={s.d} className="trd-c-line" vectorEffect="non-scaling-stroke" />
        ))}
        {hover !== null && <line x1={geo.xs[hover]} x2={geo.xs[hover]} y1={0} y2={H} className="trd-c-rule" vectorEffect="non-scaling-stroke" />}
      </svg>
      {hover !== null && h && (
        <span className="trd-c-dot" style={{ left: `${(geo.xs[hover] / W) * 100}%`, top: `${(geo.yOf(h.valueUsd) / H) * 100}%` }} aria-hidden />
      )}
      <div className="trd-c-chart-foot mt-2 font-mono text-[11px] text-os-dim">
        <span>{day(first.at)}</span>
        <span className="trd-c-pnl trd-c-chart-read" data-tone={toneOf(change)}>
          {h ? `${stamp(h.at)} · ${usd(h.valueUsd, true)}` : `${signedUsd(change)} over ${points.length} captures`}
        </span>
        <span className="text-right">{day(last.at)}</span>
      </div>
    </div>
  );
}

function Stat({ label, value, tone }: { label: string; value: ReactNode; tone?: 'up' | 'down' | 'flat' }) {
  return (
    <div className="trd-c-stat">
      <div className="text-[11.5px] text-os-muted">{label}</div>
      <div className="mt-1 text-[18px] font-normal tabular-nums text-os-text trd-c-pnl" data-tone={tone}>
        {value}
      </div>
    </div>
  );
}

function Sleeve({ m, ui }: { m: TradingModel; ui: TradingUi }) {
  const a = m.payload.view.agent;
  const s = m.series.find((x) => x.accountId === ui.acct) ?? m.series[0];
  return (
    <Card
      className="trd-c-span-8"
      title="Agent sleeve"
      sub={a.hasActed ? `${a.tradeCount} ${a.tradeCount === 1 ? 'trade' : 'trades'} · last ${a.lastTrade ? ago(a.lastTrade.at, m.now) : ''}` : 'The agent has not traded yet'}
      right={
        m.series.length > 1 ? (
          <div role="radiogroup" aria-label="Account shown in the chart" className="flex gap-1.5">
            {m.series.map((x) => (
              <button
                key={x.accountId}
                type="button"
                role="radio"
                aria-checked={ui.acct === x.accountId}
                className="trd-c-chip c-chip"
                onClick={() => ui.setAcct(x.accountId)}
              >
                {x.label}
              </button>
            ))}
          </div>
        ) : undefined
      }
    >
      <div className="px-5">
        <ValueChart points={s?.points ?? []} label={`${s?.label ?? 'Account'} value per capture`} />
      </div>
      {m.hasAgentic ? (
        <div className="trd-c-stats">
          <Stat label="Deployed" value={usd(a.deployedUsd, true)} />
          <Stat label="Idle cash" value={a.idleCashUsd === null ? '—' : usd(a.idleCashUsd, true)} />
          <Stat label="Unrealized" value={signedUsd(a.unrealizedPnlUsd)} tone={toneOf(a.unrealizedPnlUsd)} />
          <Stat label="Agent trades" value={a.tradeCount} />
        </div>
      ) : (
        <Empty sub="The agent's stats appear once a push carries the account with id agentic.">No agentic account in the feed.</Empty>
      )}
    </Card>
  );
}

function Accounts({ m, setupPhantom }: { m: TradingModel; setupPhantom: string }) {
  const hasWallet = m.accounts.some((a) => a.kind === 'wallet');
  const brokerage = m.accounts.filter((a) => a.kind === 'brokerage');
  return (
    <Card className="trd-c-span-4" title="Accounts" sub={`${m.accounts.length} ${m.accounts.length === 1 ? 'source' : 'sources'}`}>
      <div className="px-5 pb-5">
        <div className="text-[30px] font-normal leading-none tracking-[-0.01em] tabular-nums text-os-text">{m.totalUsd === null ? '—' : usd(m.totalUsd, true)}</div>
        {m.dayPnlUsd !== null && (
          <div className="mt-2 text-[12.5px] text-os-muted">
            <Pnl n={m.dayPnlUsd} /> today · {usd(m.investedUsd)} in {m.payload.positions.length} {m.payload.positions.length === 1 ? 'position' : 'positions'}
          </div>
        )}
        <ul className="mt-5 space-y-4">
          {m.accounts.map((a) => (
            <li key={a.id} title={a.error}>
              <div className="flex items-baseline justify-between gap-3 text-[12.5px]">
                <span className="min-w-0 truncate text-os-text">
                  {a.label} <span className="text-os-muted">· {a.note}</span>
                </span>
                <span className="shrink-0 tabular-nums text-os-text">{a.valueUsd === null ? '—' : usd(a.valueUsd, true)}</span>
              </div>
              <div className="trd-c-meter mt-1.5" role="img" aria-label={`${a.label}: ${Math.round(a.frac * 100)}% of the total`}>
                <i data-agent={a.agentMayTrade || undefined} style={{ width: `${Math.max(a.valueUsd === null ? 0 : 1.5, a.frac * 100)}%` }} />
              </div>
            </li>
          ))}
          {!hasWallet && (
            <li className="flex items-center justify-between gap-3 text-[12.5px]">
              <span className="text-os-muted">Phantom · not configured</span>
              <SetupLink href={setupPhantom} hint="Set PHANTOM_WALLET_ADDRESS" className="shrink-0 whitespace-nowrap" />
            </li>
          )}
        </ul>
      </div>
      {brokerage.length > 0 && (
        <dl className="mt-auto border-t border-[var(--hairline)] px-5 py-3 text-[12.5px]">
          {[
            ['Cash', usd(brokerage.reduce((t, a) => t + (a.cashUsd ?? 0), 0), true)],
            ['Buying power', usd(brokerage.reduce((t, a) => t + (a.buyingPowerUsd ?? 0), 0), true)],
          ].map(([k, v]) => (
            <div key={k} className="flex justify-between gap-3 py-0.5">
              <dt className="text-os-muted">{k}</dt>
              <dd className="tabular-nums text-os-text">{v}</dd>
            </div>
          ))}
          <div className="flex justify-between gap-3 py-0.5">
            <dt className="text-os-muted">Total P&amp;L</dt>
            <dd>
              <Pnl n={m.payload.accounts.reduce((t, a) => t + a.totalPnlUsd, 0)} />
            </dd>
          </div>
        </dl>
      )}
    </Card>
  );
}

function Reasoning({ m, ui }: { m: TradingModel; ui: TradingUi }) {
  const a = m.payload.analysis;
  return (
    <Card className="trd-c-span-4" title="Agent reasoning" sub={a ? `Examined ${a.examined} · ${a.signals} ${a.signals === 1 ? 'signal' : 'signals'} · ${ago(a.at, m.now)}` : 'No run recorded'}>
      {!a ? (
        <Empty>No analysis pushed yet. The agent writes here every run, whether or not it trades.</Empty>
      ) : (
        <div className="pb-2">
          {a.notes && (
            <div className="px-5 pb-3">
              <p className="line-clamp-2 text-[12.5px] leading-relaxed text-os-muted">{a.notes}</p>
              {a.notes.length > 140 && (
                <button type="button" className="mt-1 text-[12px] text-os-accent hover:underline" onClick={() => ui.setNotesOpen(true)}>
                  More
                </button>
              )}
            </div>
          )}
          <ul className="divide-y divide-[var(--hairline)] border-t border-[var(--hairline)]">
            {a.rows.slice(0, 6).map((r) => (
              <li key={r.ticker} className="flex items-start gap-3 px-5 py-2.5" title={r.score === null ? 'Not scored' : `Score ${r.score.toFixed(2)}`}>
                <span className="w-14 shrink-0 font-mono text-[12px] text-os-text">{r.ticker}</span>
                <span className="min-w-0 flex-1 text-[12px] text-os-muted">{r.reason}</span>
                <Badge tone={verdictTone(r.verdict)}>{cap(r.verdict)}</Badge>
              </li>
            ))}
          </ul>
        </div>
      )}
    </Card>
  );
}

function PositionsTable({ rows }: { rows: TradingPosition[] }) {
  return (
    <div className="trd-c-table">
      <div className="trd-c-tr trd-c-th" role="row">
        <span>Symbol</span>
        <span className="text-right">Qty</span>
        <span className="text-right">Value</span>
        <span className="text-right">P&amp;L</span>
      </div>
      {rows.map((r) => (
        <div key={`${r.accountId}:${r.symbol}`} className="trd-c-tr" role="row">
          <span className="min-w-0">
            <span className="font-mono text-[12.5px] text-os-text">{r.symbol}</span>
            <span className="ml-2 text-[11px] text-os-dim">{acctName(r.accountId)}</span>
          </span>
          <span className="text-right tabular-nums text-os-muted">{qty(r.quantity)}</span>
          <span className="text-right tabular-nums text-os-text">{usd(r.marketValueUsd, true)}</span>
          <span className="text-right">
            <Pnl n={r.unrealizedPnlUsd} />
          </span>
        </div>
      ))}
    </div>
  );
}

function Positions({ m, ui }: { m: TradingModel; ui: TradingUi }) {
  const pos = m.payload.positions;
  const sizes = m.payload.view.sizes;
  const most = Math.max(1, ...sizes.map((b) => b.count));
  return (
    <Card
      className="trd-c-span-4"
      title="Positions"
      sub={pos.length ? `${pos.length} open · ${usd(m.investedUsd)} invested` : 'Nothing held'}
      right={
        pos.length > 5 ? (
          <button type="button" className="c-btn h-[28px] px-2.5 text-[12px]" onClick={() => ui.setPositionsOpen(true)}>
            See all
          </button>
        ) : undefined
      }
    >
      {pos.length === 0 ? (
        <Empty>No open positions.</Empty>
      ) : (
        <>
          <div className="trd-c-buckets px-5 pb-4" role="img" aria-label={sizes.map((b) => `${b.label}: ${b.count}`).join(', ')}>
            {sizes.map((b) => (
              <div key={b.label} className="trd-c-bucket">
                <div className="trd-c-bucket-bar">
                  <i style={{ height: `${(b.count / most) * 100}%` }} data-zero={b.count === 0 || undefined} />
                </div>
                <div className="mt-1 text-center font-mono text-[10.5px] text-os-dim">{b.label}</div>
                <div className="text-center text-[11.5px] tabular-nums text-os-muted">{b.count}</div>
              </div>
            ))}
          </div>
          <PositionsTable rows={pos.slice(0, 5)} />
        </>
      )}
    </Card>
  );
}

function NextMove({ m }: { m: TradingModel }) {
  const a = m.payload.view.agent;
  const last = m.lastAgentTrade;
  const notes = m.payload.analysis?.notes;
  return (
    <section aria-labelledby="trd-c-next" className="trd-c-span-4 trd-c-next rounded-lg-t border">
      <p id="trd-c-next" className="text-[12px] font-medium text-os-accent">
        Agent · next move
      </p>
      <div className="mt-3 text-[40px] font-normal leading-none tabular-nums text-os-text">{a.tradeCount}</div>
      <p className="mt-2 text-[13px] text-os-text">{a.tradeCount === 1 ? 'trade' : 'trades'} filled in the sleeve.</p>
      <p className="mt-2 line-clamp-4 text-[12.5px] leading-relaxed text-os-muted">
        {notes || (last ? `Last: ${last.action} ${tradeLine(last)}, ${ago(last.at, m.now)}. ${last.rationale}` : 'The agent has not reported a run yet.')}
      </p>
      {m.limits.saved && (
        <p className="mt-3 font-mono text-[11px] text-os-dim">
          Autopilot {m.limits.saved.autopilot ? 'on' : 'off'} · {Math.round((a.deployedUsd / Math.max(1, m.limits.saved.maxDeployedCapitalUsd)) * 100)}% of the autonomy budget deployed
        </p>
      )}
    </section>
  );
}

function OpenOrders({ m }: { m: TradingModel }) {
  const orders = m.payload.openOrders;
  return (
    <Card className="trd-c-span-12" title="Open orders" sub={orders.length ? `${orders.length} working` : 'Nothing working'}>
      {orders.length === 0 ? (
        <Empty>No live orders at the broker.</Empty>
      ) : (
        <ul className="divide-y divide-[var(--hairline)] border-t border-[var(--hairline)]">
          {orders.map((o) => (
            <li key={o.id} className="trd-c-row">
              <Badge tone={o.side === 'buy' ? 'ok' : 'warn'}>{o.side === 'buy' ? 'Buy' : 'Sell'}</Badge>
              <div className="min-w-0 flex-1">
                <div className="text-[13px] text-os-text">
                  <span className="font-mono">{o.symbol}</span>{' '}
                  {o.dollarAmountUsd !== null ? usd(o.dollarAmountUsd, true) : `${qty(o.quantity)}${o.limitPriceUsd !== null ? ` @ ${usd(o.limitPriceUsd, true)}` : ''}`}
                </div>
                <div className="text-[12px] text-os-muted">
                  {cap(o.type)} · {o.filledQuantity ? `${qty(o.filledQuantity)} filled · ` : ''}
                  {o.placedAgent || 'Unknown'}
                </div>
              </div>
              <span className="c-chip">{cap(o.state)}</span>
              <span className="trd-c-when font-mono text-[11px] text-os-dim">{ago(o.createdAt, m.now)}</span>
            </li>
          ))}
        </ul>
      )}
    </Card>
  );
}

function TradeLog({ m, ui }: { m: TradingModel; ui: TradingUi }) {
  const c = m.payload.view.activityCounts;
  const rows = logRows(m, ui.log);
  const filters: { value: ActivityFilter; label: string }[] = [
    { value: 'all', label: 'All' },
    { value: 'agent', label: 'Agent' },
    { value: 'you', label: 'You' },
    { value: 'rejected', label: 'Rejected' },
  ];
  return (
    <Card
      className="trd-c-span-12"
      title="Trade log"
      sub={c.all ? `${rows.length} of ${c.all}` : undefined}
      right={
        c.all ? (
          <div role="radiogroup" aria-label="Filter the trade log" className="trd-c-chips">
            {filters.map((f) => (
              <button key={f.value} type="button" role="radio" aria-checked={ui.log === f.value} className="trd-c-chip c-chip" onClick={() => ui.setLog(f.value)}>
                {f.label} <span className="tabular-nums text-os-dim">{c[f.value]}</span>
              </button>
            ))}
          </div>
        ) : undefined
      }
    >
      {c.all === 0 ? (
        <Empty>No trades yet.</Empty>
      ) : rows.length === 0 ? (
        <Empty>Nothing matches this filter.</Empty>
      ) : (
        <ul className="divide-y divide-[var(--hairline)] border-t border-[var(--hairline)]">
          {rows.map((t) => (
            <li key={t.id}>
              <button type="button" className="trd-c-row trd-c-row-btn hoverable" onClick={() => ui.setOpenTrade(t)}>
                <span className="trd-c-side text-[11.5px] font-medium" data-side={t.action}>
                  {t.action === 'buy' ? 'Buy' : 'Sell'}
                </span>
                <span className="min-w-0 flex-1 text-left">
                  <span className="flex flex-wrap items-center gap-2 text-[13px] text-os-text">
                    <span className="font-mono">{tradeLine(t)}</span>
                    <Badge tone={statusTone[t.status]}>{statusLabel[t.status]}</Badge>
                  </span>
                  <span className="mt-0.5 block truncate text-[12px] text-os-muted">{t.rationale || 'No rationale given'}</span>
                </span>
                <span className="trd-c-when shrink-0 text-right text-[11.5px] text-os-muted">
                  {whoOf(t)} · {acctName(t.accountId)}
                  <span className="block font-mono text-[11px] text-os-dim">{ago(t.at, m.now)}</span>
                </span>
              </button>
            </li>
          ))}
        </ul>
      )}
    </Card>
  );
}

function Modal({ title, sub, onClose, children, wide = false }: { title: string; sub?: string; onClose: () => void; children: ReactNode; wide?: boolean }) {
  const titleId = useId();
  // Escape, the scrim and the close button melt the face (lib/use-melt); the page unmounts it once it has melted.
  const dlg = useDialog({ open: true, onClose: () => melt.close(), labelledBy: titleId });
  const melt = useMeltClose(onClose, dlg.ref);
  // Portalled to <body> (a frosted ancestor would trap the fixed layer); a box-less .trd-c wrapper keeps the
  // page's scoped rules (.trd-c .c-btn:disabled) on the modal's controls.
  return (
    <Layer>
      <div className="trd-c contents">
        <div className="trd-c-layer" {...melt.layer}>
          {/* The scrim is the face's sibling (glass brief §3.2): it frosts and dims the page; the face frosts on its own. */}
          <div className="gl-scrim" aria-hidden onClick={melt.close} />
          <div
            {...dlg.props}
            onAnimationEnd={melt.onAnimationEnd}
            className={`c-pop glass-face--strong gl-materialise trd-c-modal relative w-full ${wide ? 'max-w-[720px]' : 'max-w-[480px]'}${melt.melting ? ' is-melting' : ''}`}
          >
            <div className="flex items-start justify-between gap-3 px-5 pb-3 pt-4">
              <div className="min-w-0">
                <h2 id={titleId} className="text-[15px] font-medium text-os-text">
                  {title}
                </h2>
                {sub && <p className="mt-0.5 font-mono text-[11.5px] text-os-dim">{sub}</p>}
              </div>
              <button type="button" data-dialog-close onClick={melt.close} aria-label="Close" className="rounded-[6px] p-1 text-os-muted hover:text-os-text">
                <X size={15} strokeWidth={1.7} aria-hidden />
              </button>
            </div>
            <div className="max-h-[70vh] overflow-y-auto">{children}</div>
          </div>
        </div>
      </div>
    </Layer>
  );
}

function TradeDialog({ trade: t, onClose }: { trade: TradeActivity; onClose: () => void }) {
  const rows: [string, ReactNode][] = [
    ['Order', <span key="o" className="font-mono">{tradeLine(t)}</span>],
    ['Notional', usd(t.quantity * t.priceUsd, true)],
    ['Status', <Badge key="s" tone={statusTone[t.status]}>{statusLabel[t.status]}</Badge>],
    ['Who', whoOf(t)],
    ['Account', t.accountId === 'agentic' ? 'Agentic sleeve' : acctName(t.accountId)],
  ];
  return (
    <Modal title={`${t.action === 'buy' ? 'Buy' : 'Sell'} ${t.symbol}`} sub={stamp(t.at)} onClose={onClose}>
      <dl className="px-5">
        {rows.map(([k, v]) => (
          <div key={k} className="flex items-center justify-between gap-3 border-b border-[var(--hairline)] py-2 text-[13px]">
            <dt className="text-os-muted">{k}</dt>
            <dd className="text-os-text">{v}</dd>
          </div>
        ))}
      </dl>
      <p className="m-5 rounded-[8px] border border-os-border bg-os-bg2 p-3 text-[13px] leading-relaxed text-os-text">{t.rationale || 'No rationale given.'}</p>
    </Modal>
  );
}

function Limits({ m, ui }: { m: TradingModel; ui: TradingUi }) {
  const L = ui.limits;
  return (
    <Card className="trd-c-span-12" title="Agent limits" sub={m.limits.savedLabel}>
      <form
        noValidate
        onSubmit={(e) => {
          e.preventDefault();
          void L.save();
        }}
      >
        <div className="trd-c-fields">
          {m.limits.fields.map((f) => {
            const err = L.fieldError(f.key);
            return (
              <label key={f.key} className="trd-c-field" htmlFor={`trd-c-${f.key}`}>
                <span className="min-w-0 flex-1">
                  <span className="block text-[13px] text-os-text">{f.label}</span>
                  <span className={`block text-[11.5px] ${err ? 'text-os-err' : 'text-os-dim'}`} role={err ? 'alert' : undefined}>
                    {err ?? f.hint}
                  </span>
                </span>
                <span className="trd-c-input-wrap" data-invalid={err ? '' : undefined}>
                  {f.unit === '$' && <span className="text-os-dim">$</span>}
                  <input
                    id={`trd-c-${f.key}`}
                    inputMode="decimal"
                    value={L.values[f.key] ?? ''}
                    placeholder="Not set"
                    aria-invalid={Boolean(err)}
                    disabled={m.readOnly}
                    title={m.readOnly ? (m.readOnlyReason ?? undefined) : `${f.min.toLocaleString('en-US')} to ${f.max.toLocaleString('en-US')}`}
                    onChange={(e) => L.setValue(f.key, e.target.value)}
                    onBlur={() => L.blur(f.key)}
                    className="trd-c-input tabular-nums"
                  />
                  {f.unit === '%' && <span className="text-os-dim">%</span>}
                </span>
              </label>
            );
          })}
        </div>
        <div className="flex flex-wrap items-center gap-3 border-t border-[var(--hairline)] px-5 py-3.5">
          <button
            type="button"
            role="switch"
            aria-checked={ui.autopilot.on}
            aria-label="Autopilot"
            disabled={Boolean(ui.autopilot.reason) || ui.autopilot.switching}
            title={ui.autopilot.reason}
            onClick={() => void ui.autopilot.setAutopilot(!ui.autopilot.on)}
            className="trd-c-switch"
          >
            <i aria-hidden />
          </button>
          <div className="min-w-0 flex-1">
            <div className="text-[13px] text-os-text">Autopilot</div>
            <div className="text-[11.5px] text-os-dim">
              {ui.autopilot.on ? 'On: the Markets agent may place real orders inside these limits.' : 'Off: the daily run proposes and logs every order as a dry run; nothing reaches the broker.'}
            </div>
          </div>
          {L.saveError && (
            <span role="alert" className="text-[12px] text-os-err">
              {L.saveError}
            </span>
          )}
          <button type="submit" className="c-btn c-btn-primary" disabled={Boolean(L.saveReason) || L.saving} title={L.saveReason}>
            {L.saving ? 'Saving' : 'Save'}
          </button>
        </div>
      </form>
    </Card>
  );
}
