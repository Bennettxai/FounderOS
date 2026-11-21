import { Suspense, type ComponentProps } from 'react';
import { createGBrainProvider, readStoreNotes } from '@/lib/connectors/gbrain';
import { BRAND } from '@/lib/brand';
import { attioClients } from '@/lib/connectors/attio';
import { readVaultNotes } from '@/lib/connectors/obsidian';
import type { RosterClient } from '@/lib/schemas';
import { buildBrainGraph } from '@/lib/brain-graph';
import { buildKnowledgeGraph } from '@/lib/knowledge-graph';
import { distillMemoryGraph, type MemoryGraph } from '@/lib/memory-core';
import { foldersToClusters } from '@/lib/brain-viz';
import { getDb } from '@/lib/data';
import { PageHeader } from '@/components/PageHeader';
import { BrainCore } from '@/components/BrainCore';
import { GBrainRadialLazy } from '@/components/GBrainRadialLazy';
import { SetupLink } from '@/components/SetupLink';
import { loadObservatoryData } from '@/lib/gbrain-observatory';
import { PillarRadar } from '@/components/PillarRadar';
import { pillarRadarAxes } from '@/lib/pillar-radar';
import { BrainGraphView } from '@/components/BrainGraphView';
import { CardSkeleton } from '@/components/RouteSkeleton';
import { BrainDock } from '@/components/BrainDock';
import { Dot, SectionHead } from '@/components/terminal';
import { storageLayers, supabaseState } from '@/lib/brain-layers';
import { formatCompact } from '@/lib/format-number';
import { getAppearance } from '@/lib/appearance-server';
import { treeFor } from '@/lib/appearance';
import { GPage, GSectionHeader } from '@/components/g';
import { GStyleProvider } from '@/components/system-g/style-context';
import { BrainSection, DoctorHero, FolderBars, PillarHealth, RadialStage, StorageLayers } from '@/components/brain/BrainG';
import { BrainAbout } from '@/components/brain/BrainAbout';

export const dynamic = 'force-dynamic';

const CHECK_DOT: Record<string, string> = {
  ok: 'ok',
  warn: 'warn',
  error: 'err',
};

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

function Stage({
  step,
  title,
  caption,
  children,
}: {
  step: string;
  title: string;
  caption: React.ReactNode;
  children: React.ReactNode;
}) {
  return (
    <section className="min-w-0 flex-1 rounded-lg-t border border-os-border bg-os-surface p-6">
      <div className="flex items-center gap-4">
        <span className="flex h-7 w-7 shrink-0 items-center justify-center rounded-sm-t bg-os-accent text-xs font-medium text-os-ink">
          {step}
        </span>
        <div className="min-w-0">
          <h2 className="text-sm font-medium">{title}</h2>
          <div className="text-[12.5px] text-os-dim [overflow-wrap:anywhere]">{caption}</div>
        </div>
      </div>
      <div className="mt-5">{children}</div>
    </section>
  );
}

function Arrow({ label }: { label: string }) {
  return (
    <div className="flex shrink-0 items-center justify-center self-stretch px-1 py-3 xl:flex-col">
      <div className="flex items-center gap-1 xl:flex-col">
        <span className="hidden h-px w-6 bg-os-border-strong xl:block xl:h-6 xl:w-px" />
        <span className="text-[11.5px] text-os-dim xl:[writing-mode:vertical-rl]">
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
      className={`flex-1 rounded-md-t border px-3 py-3.5 ${
        dashed ? 'border-dashed border-os-border' : 'glass-well'
      }`}
    >
      <div className="text-xs font-medium">{title}</div>
      <div className="mt-0.5 text-[12.5px] leading-relaxed text-os-dim">{detail}</div>
    </div>
  );
}

// The client roster prefers live Attio deals (a cached real pull, lib/connectors/attio.ts) and
// falls back to the local funnel, read fresh.
async function clientRoster(db: ReturnType<typeof getDb>): Promise<RosterClient[]> {
  const live = await attioClients();
  return live.state === 'connected' && live.clients.length > 0
    ? live.clients
    : db.funnel.journeys().map((j) => ({
        id: j.id,
        name: j.name,
        venture: j.venture,
        status: j.status,
        amountUsd: j.amountUsd,
        source: 'funnel' as const,
      }));
}

/** The knowledge graph waits on the Attio roster; it streams in behind <Suspense> (SPEC 4.1 a4) so
 *  the rest of the page never waits on Attio. */
async function GraphWithClients({ roster, ...props }: Omit<ComponentProps<typeof BrainGraphView>, 'clients'> & { roster: Promise<RosterClient[]> }) {
  return <BrainGraphView {...props} clients={await roster} />;
}

// The memory constellation distills the whole brain-store (parse + local PCA
// over the full note set) — too heavy to redo per request on a force-dynamic page, so
// cache per server process with a short TTL. Never throws: an unreadable
// store yields undefined and the graph falls back to the plain the operator dot.
let memoryCache: { at: number; value: MemoryGraph | undefined } | null = null;
const MEMORY_TTL_MS = 5 * 60_000;

function memoryConstellation(): MemoryGraph | undefined {
  if (memoryCache && Date.now() - memoryCache.at < MEMORY_TTL_MS) return memoryCache.value;
  let value: MemoryGraph | undefined;
  try {
    // the operator's memory = the brain-store PLUS the Notes vault (the Claude
    // Archive is the bulk of it). Store notes win path collisions; each source
    // keeps its own folders so the constellation clusters by real structure.
    const store = readStoreNotes();
    const seen = new Set(store.map((n) => n.path));
    const vault = readVaultNotes().filter((n) => !seen.has(n.path));
    // the Chat Archive is spotlighted: a guaranteed
    // slice of the page cap and its cluster centered in the disc
    const distilled = distillMemoryGraph(buildBrainGraph([...store, ...vault]), {
      centerFolder: 'Chat Archive',
    });
    value = distilled.nodes.length > 0 ? distilled : undefined;
  } catch {
    value = undefined;
  }
  memoryCache = { at: Date.now(), value };
  return value;
}

export default async function BrainPage() {
  const provider = createGBrainProvider();
  const db = getDb();
  // Started now, awaited only inside the graph's Suspense boundary.
  const roster = clientRoster(db);
  const overview = await provider.overview();
  const { store, doctor } = overview;
  // Supabase page/chunk counts come only from `engine stats` (H33): no CLI,
  // or a stats call that fails, means no counts, never remembered ones.
  const stats = doctor.connected ? await provider.stats().catch(() => null) : null;
  const knowledgeGraph = buildKnowledgeGraph(db.agents.all(), db.departments.all(), db.people.all(), db.sopTasks.all());
  const maxFiles = Math.max(1, ...store.folders.map((f) => f.files));
  const clusters = foldersToClusters(store.folders);
  const storeShort = store.path.replace(process.env.HOME ?? '', '~');

  const lastBrainRun = db.agentRuns.byAgent('data-agent')[0];
  // latest run per agent (oldest first so the LAST write per id is the newest)
  const runsByAgent = Object.fromEntries(
    db.agentRuns
      .recent(300)
      .reverse()
      .map((r) => [r.agentId, r]),
  );
  const warnings = doctor.checks.filter((c) => c.status !== 'ok');
  const supabaseCheck = doctor.checks.find((c) => /supabase|database/i.test(c.name));
  const fallbackActive = supabaseCheck ? supabaseCheck.status !== 'ok' : !doctor.connected;

  const layers = storageLayers({ overview, stats, storeShort });

  // Two trees, one fetch (glados-ui SPEC X5, S05 F05.10 to F05.17).
  if (treeFor(getAppearance(), '/brain') === 'glados') {
    const observatory = loadObservatoryData();
    return (
      <GPage className="kg-page kg-brain">
        <GSectionHeader
          size="page"
          title="Optimal Engine"
          caption={`Knowledge core · ${store.totalFiles} pages · ${store.totalFiles > 0 ? 'brain-store mounted' : 'brain-store not found'}`}
          action={
            <div className="kg-brain-head">
              {/* retrieval notices from the radial portal in here (GBrainRadial) */}
              <div id="gbrain-notice-slot" className="kg-notice-slot" />
              <BrainAbout storeShort={storeShort} />
            </div>
          }
        />
        <RadialStage data={observatory} />
        <BrainSection id="kgraph-head" title="Knowledge graph" caption={`${knowledgeGraph.nodes.length} nodes: agents, departments, people, SOP tasks and the memory constellation`}>
          <GStyleProvider>
            <Suspense fallback={<CardSkeleton height={520} label="Loading the knowledge graph" />}>
              <GraphWithClients
                graph={knowledgeGraph}
                agents={db.agents.all()}
                departments={db.departments.all()}
                people={db.people.all()}
                tasks={db.sopTasks.all()}
                memory={memoryConstellation()}
                roster={roster}
                runsByAgent={runsByAgent}
              />
            </Suspense>
          </GStyleProvider>
        </BrainSection>
        <div className="kg-brain-row" data-split="5-7">
          <DoctorHero doctor={doctor} lastRun={lastBrainRun ? relativeTime(lastBrainRun.finishedAt) : null} fallbackActive={fallbackActive} />
          <PillarHealth axes={pillarRadarAxes(db.departments.all(), db.agents.all(), db.sopTasks.all(), runsByAgent)} />
        </div>
        <div className="kg-brain-row" data-split="6-6">
          <StorageLayers layers={layers} />
          {store.totalFiles > 0 && <FolderBars folders={store.folders} total={store.totalFiles} />}
        </div>
      </GPage>
    );
  }

  return (
    <div>
      {/* Capture floats over the graph and folds down to a line at the bottom
          middle — see BrainDock. The header's right slot is now the landing
          area for retrieval notices, which the radial portals up into: they
          belong ABOVE the canvas window, level with the title, rather than
          sitting inside the frame competing with the graph. */}
      <PageHeader
        eyebrow="Knowledge core"
        title="Optimal Engine"
        caret
        rightWide
        // h-0: the notice floats at title height without adding header height,
        // so it cannot push the canvas — and the capture card at the bottom of
        // the canvas — down off the fold.
        right={<div id="gbrain-notice-slot" className="relative z-40 h-0 w-full" />}
      />

      {/* The living core — the Example 6 radial (same engine as
          the archived /design/gbrain-3): real folders, pages, chunks, live retrieval. */}
      {/* Full-bleed: the centered max-w container leaves gutters at some
          widths, so negative padding-margins alone never reach the page edge.
          Size the frame off the viewport minus the fixed sidebar (--sidebar-w) and
          the conductor dock, and center it with a margin (not a transform —
          a transform would trap the radial's position:fixed fullscreen). */}
      <section
        className="relative mt-2 h-[calc(100vh-130px)] min-h-[560px] [overflow-x:clip]"
        style={{
          width: 'calc(100vw - var(--sidebar-w) - var(--conductor-w, 0px))',
          marginLeft: 'calc((100% - (100vw - var(--sidebar-w) - var(--conductor-w, 0px))) / 2)',
        }}
      >
        <GBrainRadialLazy data={loadObservatoryData()} />
        <BrainDock />
      </section>

      <section className="mt-8">
        <SectionHead label="Knowledge graph" count={`${knowledgeGraph.nodes.length} nodes`} />
        <Suspense fallback={<CardSkeleton height={520} label="Loading the knowledge graph" />}>
          <GraphWithClients
            graph={knowledgeGraph}
            agents={db.agents.all()}
            departments={db.departments.all()}
            people={db.people.all()}
            tasks={db.sopTasks.all()}
            memory={memoryConstellation()}
            roster={roster}
            runsByAgent={runsByAgent}
          />
        </Suspense>
      </section>

      {/* Optimal Engine knowledge core: the PILLAR SPIDER CHART on the LEFT, the
          radar/health monitor on the RIGHT — a 50/50 split of the row.
          Stacks on narrow screens. */}
      <div className="mt-5 grid grid-cols-1 items-stretch gap-5 lg:grid-cols-2">
        <div className="glass-panel-violet flex min-h-[480px] flex-col overflow-hidden">
          <div className="flex items-baseline justify-between px-5 pt-4">
            <span className="glass-label">Pillar health</span>
            <span className="text-[11.5px] text-os-dim">live roster + runs + SOP coverage</span>
          </div>
          <PillarRadar
            axes={pillarRadarAxes(db.departments.all(), db.agents.all(), db.sopTasks.all(), runsByAgent)}
            health={doctor.healthScore}
            warnings={warnings.length}
          />
        </div>

        <div className="glass-panel-violet flex min-h-[480px] flex-col overflow-hidden">
          {/* annotations as a real header row — at half width the old absolute
              corners collided with the radar's ring labels */}
          <div className="flex items-start justify-between px-5 pt-4 text-[11.5px] leading-normal text-os-dim">
            <div className="flex flex-col gap-1">
              <span className="glass-label">Doctor</span>
              <span>{doctor.connected ? (warnings.length > 0 ? 'warnings' : 'ok') : 'unreachable'} · {lastBrainRun ? `last run ${relativeTime(lastBrainRun.finishedAt)}` : 'no agent runs yet'}</span>
            </div>
            <div className="flex flex-col gap-1 text-right">
              <span className="glass-label">Hybrid search</span>
              <span>{doctor.connected ? 'verified' : 'degraded'} · {fallbackActive ? 'local fallback' : 'Supabase reachable'}</span>
            </div>
          </div>
          <div className="grid flex-1 place-items-center">
            <div className="w-full max-w-[540px]">
              <BrainCore
                clusters={clusters}
                health={doctor.healthScore}
                doctor={doctor}
                fallbackActive={fallbackActive}
                version={doctor.version ?? null}
                supabasePages={stats?.pages ?? null}
                supabaseState={supabaseState(doctor, stats).val}
              />
            </div>
          </div>
        </div>

      </div>

      {/* Core status: storage layers + doctor-health footer, full width. */}
      <div className="glass-panel mt-5 flex flex-col overflow-hidden">
        <div className="glass-label border-b border-[var(--hairline)] px-5 py-4">
          Storage layers
        </div>
        <div className="flex flex-1 flex-col divide-y divide-os-border">
          {layers.map((layer) => (
            <div key={layer.name} className="flex flex-1 items-center gap-4 px-3.5 py-3">
              <Dot state={layer.state} pulse={layer.state === 'connected'} />
              <div className="min-w-0 flex-1">
                <div className="truncate text-[13.5px] font-medium">{layer.name}</div>
                <div className="truncate font-mono text-[11.5px] text-os-dim">{layer.sub}</div>
              </div>
              <span className="shrink-0 text-[12.5px] text-os-muted">{layer.val}</span>
            </div>
          ))}
        </div>
        <div className="flex items-center justify-between border-t border-os-border px-3.5 py-3 text-[12.5px]">
          <span className="text-os-dim">
            <b className="font-medium text-os-muted">Doctor</b> — health {doctor.healthScore ?? '—'}/100
          </span>
          <span className="flex items-center gap-2.5 text-os-muted">
            <span className={`dot ${warnings.length > 0 ? 'warn' : doctor.connected ? 'ok' : 'err'}`} />
            {doctor.connected ? (warnings.length > 0 ? `${warnings.length} warnings` : 'all green') : 'offline'}
          </span>
        </div>
      </div>

      {/* The pipeline: where knowledge lives and how it becomes searchable */}
      <section className="mt-8">
        <SectionHead label="Pipeline" count={`${store.totalFiles} pages on disk`} />
        <div className="flex flex-col gap-3 xl:flex-row xl:items-stretch">
          <Stage step="1" title="Markdown brain-store" caption={<span className="font-mono text-[12px]">{storeShort}</span>}>
            <div className="text-xs text-os-muted">
              {store.totalFiles} pages on disk, plain <span className="font-mono text-os-text">.md</span> files —
              the source of truth. <code className="font-mono text-[12.5px]">engine sync</code> walks the git repo and
              pushes changed pages up.
            </div>
            <ul className="mt-4 space-y-2.5">
              {store.folders.map((folder) => (
                <li key={folder.name} className="flex items-center gap-3">
                  <span className="w-24 shrink-0 truncate font-mono text-[12.5px] text-os-muted">{folder.name}</span>
                  <span
                    className="h-2 rounded-sm bg-[var(--ramp-1)]"
                    style={{
                      width: `${Math.max(6, (folder.files / maxFiles) * 100)}%`,
                      opacity: 0.25 + 0.55 * (folder.files / maxFiles),
                    }}
                  />
                  <span className="font-mono text-[12.5px] text-os-dim">{folder.files}</span>
                </li>
              ))}
            </ul>
          </Stage>

          <Arrow label="sync · import" />

          <Stage step="2" title="engine CLI" caption="chunk · embed · route — the engine between disk and database">
            <div className="flex items-baseline gap-3">
              <span className="text-[30px] font-normal tabular-nums">{doctor.healthScore ?? '—'}</span>
              <span className="text-xs text-os-dim">/ 100 health{doctor.connected ? '' : ' · CLI unreachable'}</span>
            </div>
            <ul className="mt-4 space-y-2.5">
              {doctor.checks.map((check) => (
                <li key={check.name} className="flex items-start gap-3 text-[12.5px]">
                  <span className={`dot mt-1 ${CHECK_DOT[check.status] ?? 'err'}`} />
                  <span className="text-os-muted">
                    <span className="font-medium text-os-text">{check.name}</span> — {check.message}
                  </span>
                </li>
              ))}
              {doctor.checks.length === 0 && (
                <li className="flex items-center gap-3 rounded-md-t border border-dashed border-os-border px-3 py-3 text-[12.5px] text-os-dim">
                  <span className="dot err" /> Doctor offline
                  <SetupLink href="/setup" hint={doctor.detail} className="ml-auto" />
                </li>
              )}
            </ul>
            <div className="mt-4 flex flex-wrap gap-1">
              {['put', 'get', 'query', 'search', 'sync', 'import', 'export', 'doctor'].map((cmd) => (
                <span key={cmd} className="glass-chip rounded-full px-1.5 py-0.5 font-mono text-[11.5px] text-os-muted">
                  {cmd}
                </span>
              ))}
            </div>
          </Stage>

          <Arrow label="embed · upsert" />

          <Stage step="3" title="Supabase Postgres + pgvector" caption='"Second Brain" · ZeroEntropy embeddings'>
            {stats ? (
              <div className="grid grid-cols-2 gap-3" title="From engine stats, read when this page loaded">
                <div className="glass-well px-3 py-3.5">
                  <div className="text-[22px] font-normal tabular-nums">{formatCompact(stats.pages)}</div>
                  <div className="text-[11.5px] text-os-dim">Pages · engine stats</div>
                </div>
                <div className="glass-well px-3 py-3.5">
                  <div className="text-[22px] font-normal tabular-nums">{formatCompact(stats.chunks)}</div>
                  <div className="text-[11.5px] text-os-dim">Chunks · engine stats</div>
                </div>
              </div>
            ) : (
              <p className="rounded-md-t border border-dashed border-os-border px-3 py-3.5 text-[12.5px] text-os-dim">
                Page and chunk counts unavailable: {doctor.connected ? 'engine stats did not answer' : 'the engine CLI is unreachable'}.
              </p>
            )}
            <div className="mt-4 space-y-2.5 text-[12.5px] leading-relaxed text-os-muted">
              <p>
                Each page is split into chunks; every chunk gets a ZeroEntropy embedding stored in a{' '}
                <code className="font-mono">vector</code> column. Postgres holds both the text (tsvector) and the
                vectors, so one database answers keyword and semantic queries.
              </p>
              <p className="text-os-dim">
                Free tier pauses on idle — when hybrid queries fail, unpause from the Supabase dashboard. The
                brain-store on disk keeps working regardless.
              </p>
            </div>
          </Stage>
        </div>
      </section>

      {/* How a query actually resolves */}
      <section className="mt-8">
        <SectionHead label="Query path" />
        <p className="mb-4 text-xs text-os-dim">
          What happens when an agent calls <code className="font-mono">engine query</code> — hybrid retrieval with an
          honest fallback.
        </p>
        <div className="flex flex-col gap-3 xl:flex-row xl:items-stretch">
          <FlowStep title="Question" detail="Natural-language query from you or an agent run." />
          <Arrow label="expand" />
          <FlowStep title="Query expansion" detail="The CLI rewrites the question into search variants (skip with --no-expand)." />
          <Arrow label="fan out" />
          <div className="flex flex-1 flex-col gap-3">
            <FlowStep title="Keyword search" detail="Postgres tsvector full-text match over chunk text." />
            <FlowStep title="Vector search" detail="pgvector nearest-neighbor over ZeroEntropy embeddings." />
          </div>
          <Arrow label="merge" />
          <FlowStep title="RRF fusion" detail="Reciprocal-rank fusion merges both result lists into one ranking." />
          <Arrow label="answer" />
          <FlowStep title="Ranked snippets" detail="Top pages with snippets, returned to the agent." />
        </div>
        <div className="mt-3 flex flex-col gap-3 xl:flex-row xl:items-stretch">
          <FlowStep
            dashed
            title="Fallback: local grep"
            detail={`If Supabase is paused or unreachable, ${BRAND.name} greps the markdown brain-store directly — fewer smarts, zero downtime.`}
          />
        </div>
      </section>
    </div>
  );
}
