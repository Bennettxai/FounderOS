import type { CSSProperties, ReactNode } from 'react';
import Link from 'next/link';
import { createGBrainProvider } from '@/lib/connectors/gbrain';
import { foldersToClusters } from '@/lib/brain-viz';
import { getDb } from '@/lib/data';
import { BrainCore } from '@/components/BrainCore';
import { PillarRadar } from '@/components/PillarRadar';
import { pillarRadarAxes } from '@/lib/pillar-radar';
import { Dot } from '@/components/terminal';
import { DoctorRerun } from '@/components/DoctorRerun';
import { DoctorRunProvider } from '@/components/DoctorRun';
import { DoctorChecks } from '@/components/DoctorChecks';
import { Slab, SlabTitle, SlabCard, BigStat, Chip, MeterStack, InsightCard, PILL } from '@/components/slab';
import { StepLine, DotMatrix } from '@/components/slab-charts';
import { doctorVolume, doctorLayers, type StorageLayer } from '@/lib/doctor-volume';

export const dynamic = 'force-dynamic';

const WINDOW_DAYS = 14;

function relativeTime(iso: string): string {
  const ms = Date.now() - new Date(iso).getTime();
  if (!Number.isFinite(ms) || ms < 0) return iso;
  const minutes = Math.floor(ms / 60_000);
  if (minutes < 1) return 'just now';
  if (minutes < 60) return `${minutes}m ago`;
  const hours = Math.floor(minutes / 60);
  if (hours < 24) return `${hours}h ago`;
  return `${Math.floor(hours / 24)}d ago`;
}

/** The Brand Deals status pill, in the layer's honest status color. */
function pillStyle(state: StorageLayer['state']): CSSProperties {
  const hue = state === 'connected' ? 'var(--ok)' : state === 'error' ? 'var(--err)' : 'var(--warn)';
  return { background: `color-mix(in oklab, ${hue} 16%, transparent)`, color: hue };
}

function Stage({
  step,
  title,
  caption,
  children,
}: {
  step: string;
  title: string;
  caption: string;
  children: ReactNode;
}) {
  return (
    <section className="flex-1 rounded-panel border border-os-border bg-os-surface2 p-5">
      <div className="flex items-center gap-3">
        <span className="flex h-7 w-7 shrink-0 items-center justify-center rounded-ctl bg-os-accent font-mono text-xs font-bold text-os-ink">
          {step}
        </span>
        <div>
          <h3 className="text-[14px] font-semibold">{title}</h3>
          <div className="font-mono text-[10.5px] text-os-dim">{caption}</div>
        </div>
      </div>
      <div className="mt-4">{children}</div>
    </section>
  );
}

function Arrow({ label }: { label: string }) {
  return (
    <div className="flex shrink-0 items-center justify-center self-stretch px-1 py-2 xl:flex-col">
      <div className="flex items-center gap-1 xl:flex-col">
        <span className="hidden h-px w-6 bg-os-border-strong xl:block xl:h-6 xl:w-px" />
        <span className="font-mono text-[10px] uppercase tracking-widest text-os-dim xl:[writing-mode:vertical-rl]">
          {label}
        </span>
        <span className="text-os-muted xl:rotate-90">→</span>
      </div>
    </div>
  );
}

function FlowStep({ title, detail, dashed = false }: { title: string; detail: string; dashed?: boolean }) {
  return (
    <div
      className={`flex-1 rounded-panel border px-3 py-2.5 ${
        dashed ? 'border-dashed border-os-border' : 'border-os-border bg-os-surface2'
      }`}
    >
      <div className="text-xs font-semibold">{title}</div>
      <div className="mt-0.5 text-[11px] leading-relaxed text-os-dim">{detail}</div>
    </div>
  );
}

// The Doctor: everything that reports on G-Brain's health, split off
// the knowledge-graph tab so that view can stay a single, uncluttered screen.
//
// 2026-09-24: the whole view sits in the Brand Deals slab (components/slab).
// Every number in the hero and second row comes from lib/doctor-volume, fed
// with the same doctor, store, radar and run rows the panels below render.
export default async function DoctorPage() {
  const overview = await createGBrainProvider().overview();
  const { store, doctor } = overview;
  const db = getDb();
  const maxFiles = Math.max(1, ...store.folders.map((f) => f.files));
  const clusters = foldersToClusters(store.folders);
  const storeShort = store.path.replace(process.env.HOME ?? '', '~');

  const brainRuns = db.agentRuns.byAgent('data-agent');
  const lastBrainRun = brainRuns[0];
  // latest run per agent (oldest first so the LAST write per id is the newest)
  const runsByAgent = Object.fromEntries(
    db.agentRuns
      .recent(300)
      .reverse()
      .map((r) => [r.agentId, r]),
  );
  const axes = pillarRadarAxes(db.departments.all(), db.agents.all(), db.sopTasks.all(), runsByAgent);
  const { layers, fallbackActive } = doctorLayers({ doctor, store, storeShort });
  const v = doctorVolume({ doctor, store, axes, runs: brainRuns, days: WINDOW_DAYS });
  const warnings = v.counts.total - v.counts.ok;
  const layersLive = layers.filter((l) => l.state === 'connected').length;
  const statusTone = doctor.connected ? (warnings > 0 ? 'warn' : 'ok') : 'err';

  return (
    <DoctorRunProvider>
      <Slab>
        <SlabTitle
          eyebrow="engine health"
          title="Doctor"
          meta={`${storeShort} · ${store.totalFiles} pages · ${lastBrainRun ? `last run ${relativeTime(lastBrainRun.finishedAt)} · data-agent` : 'no agent runs yet'}`}
          right={
            <>
              <Chip tone={statusTone}>
                {doctor.connected ? (warnings > 0 ? `${warnings} warnings` : 'all green') : 'unreachable'}
              </Chip>
              <Link href="/brain" className={PILL}>
                G-Brain
              </Link>
              <DoctorRerun />
            </>
          }
        />

        {/* Hero row, Brand Deals' shape: the pillar radar + the volume card */}
        <div className="grid grid-cols-[2fr_1fr] gap-6 max-[1200px]:grid-cols-1">
          <SlabCard i={1} title="Pillar Health" sub="live roster + runs + SOP coverage" className="flex flex-col overflow-hidden">
            <div data-lens="r" aria-label="pillar health radar" className="flex min-h-[440px] flex-1 flex-col">
              <PillarRadar axes={axes} health={doctor.healthScore} warnings={warnings} />
            </div>
          </SlabCard>

          <SlabCard i={2} title="Health Volume" sub="out of 100" className="flex flex-col">
            <div className="flex flex-1 flex-col px-6 pb-6 pt-3">
              <BigStat
                value={v.headline ?? undefined}
                display={v.headline == null ? ' - ' : undefined}
                unit="/100"
                chips={v.chips}
                caption={v.caption}
              />
              <MeterStack meters={v.meters} foot={v.foot} />
            </div>
          </SlabCard>
        </div>

        {/* Second row: brain runs, the store's shape, THE gradient card */}
        <div className="mt-6 grid grid-cols-3 gap-6 max-[1200px]:grid-cols-1">
          <SlabCard i={3} title="Brain Runs" sub={`data-agent · last ${WINDOW_DAYS} days`} className="pb-2">
            <div className="px-6 pb-2 pt-3">
              <BigStat
                size={30}
                value={v.runsInWindow}
                chips={v.failedInWindow > 0 ? [{ tone: 'err', text: `${v.failedInWindow} failed` }] : []}
                caption={lastBrainRun ? `last run ${relativeTime(lastBrainRun.finishedAt)}` : 'no agent runs yet'}
              />
            </div>
            <StepLine series={v.series} hue="var(--send-activity)" unit=" runs" empty={`No data-agent runs in the last ${WINDOW_DAYS} days.`} />
          </SlabCard>

          <SlabCard i={4} title="Brain Store" sub={`${store.folders.length} folders`}>
            {/* folder names run long, so the number sits above the matrix */}
            <div className="flex flex-col gap-5 px-6 pb-6 pt-3">
              <BigStat
                size={30}
                value={v.store.total}
                unit="pages"
                caption={v.store.top ? `largest · ${v.store.top.name} · ${v.store.top.files} pages` : 'no markdown on disk yet'}
              />
              <DotMatrix cols={v.store.cols} hue="var(--brain-2)" />
            </div>
          </SlabCard>

          <InsightCard
            i={5}
            badge="Needs you"
            value={v.insight.value ?? undefined}
            display={v.insight.value == null ? ' - ' : undefined}
            headline={v.insight.headline}
            body={v.insight.body}
            frac={v.insight.frac}
          />
        </div>

        {/* Third row: the doctor core beside the storage layers */}
        <div className="mt-6 grid grid-cols-1 gap-6 lg:grid-cols-2">
          <SlabCard i={6} title="Doctor Core" sub={doctor.connected ? (warnings > 0 ? 'warnings' : 'ok') : 'unreachable'} className="flex flex-col overflow-hidden">
            <div data-lens="r" className="brain-stage relative mt-4 flex min-h-[440px] flex-1 flex-col">
              <div className="flex items-start justify-between px-6 pt-3.5 font-mono text-[10px] leading-normal text-os-dim">
                <span>{lastBrainRun ? `last run ${relativeTime(lastBrainRun.finishedAt)} · data-agent` : 'no agent runs yet'}</span>
                <div className="flex flex-col gap-1 text-right">
                  <span>
                    <b className="font-medium text-os-muted">hybrid search</b> {doctor.connected ? 'verified' : 'degraded'}
                  </span>
                  <span>{fallbackActive ? 'local fallback active' : 'supabase reachable'}</span>
                </div>
              </div>
              <div className="grid flex-1 place-items-center">
                <div className="w-full max-w-[540px]">
                  <BrainCore clusters={clusters} health={doctor.healthScore} doctor={doctor} fallbackActive={fallbackActive} />
                </div>
              </div>
            </div>
          </SlabCard>

          {/* Core status: storage layers + doctor-health footer */}
          <SlabCard i={7} title="Storage layers" sub={`${layersLive}/${layers.length} live`} className="flex flex-col">
            <div className="mt-4 flex flex-1 flex-col border-t border-os-border">
              {layers.map((layer) => (
                <div
                  key={layer.name}
                  data-lens="r"
                  className="pressable is-row flex flex-1 items-center gap-4 border-b border-os-border px-6 py-4 last:border-0"
                >
                  <Dot state={layer.state} pulse={layer.state === 'connected'} />
                  <div className="min-w-0 flex-1">
                    <div className="truncate text-[13.5px] font-medium">{layer.name}</div>
                    <div className="truncate font-mono text-[11px] text-os-dim">{layer.sub}</div>
                  </div>
                  <span
                    className="shrink-0 rounded-full px-2.5 py-0.5 font-mono text-[10.5px] uppercase tracking-[0.12em]"
                    style={pillStyle(layer.state)}
                  >
                    {layer.val}
                  </span>
                </div>
              ))}
            </div>
            <div className="flex items-center justify-between border-t border-os-border px-6 py-3.5 font-mono text-[11px]">
              <span className="text-os-dim">
                <b className="font-medium text-os-muted">doctor</b> · health {doctor.healthScore ?? ' - '}/100
              </span>
              <span className={warnings > 0 ? 'text-os-warn' : doctor.connected ? 'text-os-ok' : 'text-os-err'}>
                {doctor.connected ? (warnings > 0 ? `${warnings} warnings` : 'all green') : 'offline'}
              </span>
            </div>
          </SlabCard>
        </div>

        {/* The pipeline: where knowledge lives and how it becomes searchable */}
        <SlabCard i={8} title="Pipeline" sub={`${store.totalFiles} pages on disk`} className="mt-6">
          <div className="flex flex-col gap-2 px-6 pb-6 pt-4 xl:flex-row xl:items-stretch">
            <Stage step="1" title="Markdown brain-store" caption={storeShort}>
              <div className="text-xs text-os-muted">
                {store.totalFiles} pages on disk, plain <span className="font-semibold text-os-text">.md</span> files,
                the source of truth. <code className="font-mono text-[11px]">gbrain sync</code> walks the git repo and
                pushes changed pages up.
              </div>
              <ul className="mt-3 space-y-1.5">
                {store.folders.map((folder) => (
                  <li key={folder.name} className="flex items-center gap-2">
                    <span className="w-24 shrink-0 truncate font-mono text-[11px] text-os-muted">{folder.name}</span>
                    <span
                      className="h-2 rounded-full bg-os-accent"
                      style={{
                        width: `${Math.max(6, (folder.files / maxFiles) * 100)}%`,
                        opacity: 0.25 + 0.55 * (folder.files / maxFiles),
                      }}
                    />
                    <span className="font-mono text-[11px] text-os-dim">{folder.files}</span>
                  </li>
                ))}
              </ul>
            </Stage>

            <Arrow label="sync · import" />

            <Stage step="2" title="gbrain CLI" caption="chunk · embed · route, the engine between disk and database">
              <DoctorChecks
                checks={doctor.checks}
                healthScore={doctor.healthScore}
                connected={doctor.connected}
                detail={doctor.detail}
              />
              <div className="mt-3 flex flex-wrap gap-1.5">
                {['put', 'get', 'query', 'search', 'sync', 'import', 'export', 'doctor'].map((cmd) => (
                  <span key={cmd} data-lens="c" className="rounded-full border border-os-border px-2.5 py-0.5 font-mono text-[10.5px] text-os-muted">
                    {cmd}
                  </span>
                ))}
              </div>
            </Stage>

            <Arrow label="embed · upsert" />

            <Stage step="3" title="Supabase Postgres + pgvector" caption='"Second Brain" · Ollama bge-m3 embeddings'>
              <div className="grid grid-cols-2 gap-2">
                <div className="rounded-panel border border-os-border bg-os-surface px-3 py-2.5">
                  <div className="text-[30px] font-semibold leading-none tracking-[-0.03em] tabular-nums">918</div>
                  <div className="mt-1.5 font-mono text-[10px] uppercase tracking-wider text-os-dim">pages · last known</div>
                </div>
                <div className="rounded-panel border border-os-border bg-os-surface px-3 py-2.5">
                  <div className="text-[30px] font-semibold leading-none tracking-[-0.03em] tabular-nums">11k</div>
                  <div className="mt-1.5 font-mono text-[10px] uppercase tracking-wider text-os-dim">chunks · last known</div>
                </div>
              </div>
              <div className="mt-3 space-y-1.5 text-[11px] leading-relaxed text-os-muted">
                <p>
                  Each page is split into chunks; every chunk gets a local bge-m3 embedding stored in a{' '}
                  <code className="font-mono">vector</code> column. Postgres holds both the text (tsvector) and the
                  vectors, so one database answers keyword and semantic queries.
                </p>
                <p className="text-os-dim">
                  Free tier pauses on idle  -  when hybrid queries fail, unpause from the Supabase dashboard. The
                  brain-store on disk keeps working regardless.
                </p>
              </div>
            </Stage>
          </div>
        </SlabCard>

        {/* How a query actually resolves */}
        <SlabCard i={9} title="Query path" sub="hybrid retrieval, honest fallback" className="mt-6">
          <div className="px-6 pb-6 pt-3">
            <p className="mb-4 text-[13px] text-os-dim">
              What happens when an agent calls <code className="font-mono">gbrain query</code>, hybrid retrieval with an
              honest fallback.
            </p>
            <div className="flex flex-col gap-2 lg:flex-row lg:items-stretch">
              <FlowStep title="Question" detail="Natural-language query from you or an agent run." />
              <Arrow label="expand" />
              <FlowStep title="Query expansion" detail="The CLI rewrites the question into search variants (skip with --no-expand)." />
              <Arrow label="fan out" />
              <div className="flex flex-1 flex-col gap-2">
                <FlowStep title="Keyword search" detail="Postgres tsvector full-text match over chunk text." />
                <FlowStep title="Vector search" detail="pgvector nearest-neighbor over Ollama bge-m3 embeddings (1024d)." />
              </div>
              <Arrow label="merge" />
              <FlowStep title="RRF fusion" detail="Reciprocal-rank fusion merges both result lists into one ranking." />
              <Arrow label="answer" />
              <FlowStep title="Ranked snippets" detail="Top pages with snippets, returned to the agent." />
            </div>
            <div className="mt-2 flex flex-col gap-2 lg:flex-row lg:items-stretch">
              <FlowStep
                dashed
                title="Fallback: local grep"
                detail="If Supabase is paused or unreachable, FOUNDER OS greps the markdown brain-store directly  -  fewer smarts, zero downtime."
              />
            </div>
          </div>
        </SlabCard>
      </Slab>
    </DoctorRunProvider>
  );
}
