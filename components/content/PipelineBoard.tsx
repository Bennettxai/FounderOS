'use client';

/**
 * The Pipelines section — CONTENT-V3-SCAFFOLD.md §6. Saved pipelines as
 * glass cards (same chrome language as /workflows' WorkflowTree): name,
 * description, node count, a small inline flow glyph, and live run state
 * derived straight from the production queue — a node only shows a status
 * dot once a real task with matching pipelineId/nodeId exists, nothing is
 * animated that isn't backed by a queue row. Run opens a cost-gated confirm
 * (ordered node list, per-node ~credits, total, campaign select) before
 * anything queues; Run as form is the same confirm behind a plain-field
 * front door for a pipeline's input nodes (the Weavy App-Mode pattern —
 * graph for builders, form for daily use).
 *
 * Props contract (documented for the orchestrator wiring this into
 * app/content/page.tsx):
 *   pipelines — the saved pipeline list (GET /api/content/pipelines)
 *   tasks     — the production queue, so run state can be derived per node
 *   campaigns — for the Run confirm's campaign select
 * This component owns no data fetching — it is pure props in, and calls
 * router.refresh() after a mutation so the parent server component re-reads
 * the store on the next render, exactly like ProductionQueue/WorkflowTree.
 */

import { useMemo, useState } from 'react';
import { useRouter } from 'next/navigation';
import { FileText, Pencil, Play, Plus, Send, Sparkles, Upload, Wand2, X, type LucideIcon } from 'lucide-react';
import { Dot } from '@/components/terminal';
import { estimateCredits, type ProductionTask } from '@/lib/content-schema';
import type { Campaign } from '@/lib/content-campaigns';
import type { Pipeline, PipelineInput, PipelineNode, PipelineNodeKind } from '@/lib/content-pipelines';
import { PipelineBuilder } from '@/components/content/PipelineBuilder';
import { useMeltClose } from '@/lib/use-melt';

export type PipelineBoardProps = {
  pipelines: Pipeline[];
  tasks: ProductionTask[];
  campaigns: Campaign[];
};

type BuilderTarget = { mode: 'new' } | { mode: 'edit'; pipeline: Pipeline } | { mode: 'draft'; input: PipelineInput } | null;
type RunTarget = { pipeline: Pipeline; asForm: boolean } | null;

const KIND_ICON: Record<PipelineNodeKind, LucideIcon> = {
  input: Upload,
  generate: Sparkles,
  transform: Wand2,
  output: Send,
};

const STATUS_DOT: Record<ProductionTask['status'], string> = {
  queued: 'available',
  generating: 'warn',
  delivered: 'ok',
  blocked: 'off',
};

/** Longest-path column layout over the visual edges (Kahn's algorithm —
 *  terminates cleanly even if a pipeline somehow carries a cycle; any node
 *  that never gets a topological position lands in a trailing column
 *  instead of hanging the render). Multiple nodes sharing a column stack
 *  vertically — branches stacked, per the spec. */
function flowColumns(nodes: PipelineNode[], edges: { from: string; to: string }[]): PipelineNode[][] {
  const byId = new Map(nodes.map((n) => [n.id, n]));
  const adjacency = new Map<string, string[]>();
  const indegree = new Map<string, number>(nodes.map((n) => [n.id, 0]));
  for (const e of edges) {
    if (!byId.has(e.from) || !byId.has(e.to)) continue;
    indegree.set(e.to, (indegree.get(e.to) ?? 0) + 1);
    const list = adjacency.get(e.from);
    if (list) list.push(e.to);
    else adjacency.set(e.from, [e.to]);
  }
  const depth = new Map<string, number>();
  const indegreeLeft = new Map(indegree);
  const queue: string[] = nodes.filter((n) => (indegree.get(n.id) ?? 0) === 0).map((n) => n.id);
  queue.forEach((id) => depth.set(id, 0));
  const order: string[] = [];
  let i = 0;
  while (i < queue.length) {
    const id = queue[i++];
    order.push(id);
    const d = depth.get(id) ?? 0;
    for (const next of adjacency.get(id) ?? []) {
      depth.set(next, Math.max(depth.get(next) ?? 0, d + 1));
      const left = (indegreeLeft.get(next) ?? 0) - 1;
      indegreeLeft.set(next, left);
      if (left === 0) queue.push(next);
    }
  }
  const placed = new Set(order);
  const orphans = nodes.filter((n) => !placed.has(n.id));
  const maxDepth = order.length > 0 ? Math.max(...order.map((id) => depth.get(id) ?? 0)) : -1;
  const columns: PipelineNode[][] = Array.from({ length: Math.max(maxDepth + 1, 0) }, () => []);
  for (const id of order) columns[depth.get(id) ?? 0].push(byId.get(id)!);
  if (orphans.length > 0) columns.push(orphans);
  return columns.filter((c) => c.length > 0);
}

/** Client-safe mirror of lib/content-pipelines.ts's topologicalTaskOrder +
 *  runPreview — pure over the pipeline's own data (no fs), so the Run
 *  confirm can show real numbers before the server does the authoritative
 *  compile on Confirm. */
function localPreview(pipeline: Pick<Pipeline, 'nodes'>): {
  nodes: { nodeId: string; label: string; model: string | null; credits: number }[];
  totalCredits: number;
} {
  const byId = new Map(pipeline.nodes.map((n) => [n.id, n]));
  const genTransform = pipeline.nodes.filter((n) => n.kind === 'generate' || n.kind === 'transform');
  const deps = new Map<string, Set<string>>();
  for (const n of genTransform) {
    const set = new Set<string>();
    for (const b of n.mediaBindings) {
      if (b.source.type !== 'node') continue;
      const upstream = byId.get(b.source.ref);
      if (upstream && (upstream.kind === 'generate' || upstream.kind === 'transform')) set.add(upstream.id);
    }
    deps.set(n.id, set);
  }
  const visited = new Set<string>();
  const order: PipelineNode[] = [];
  function visit(id: string) {
    if (visited.has(id)) return;
    visited.add(id);
    for (const dep of deps.get(id) ?? []) visit(dep);
    const node = byId.get(id);
    if (node) order.push(node);
  }
  genTransform.forEach((n) => visit(n.id));

  const nodes = order.map((n) => {
    const rawBatch = n.params.batch;
    const batch = typeof rawBatch === 'number' && rawBatch >= 1 && rawBatch <= 8 ? Math.round(rawBatch) : 1;
    const credits = n.model ? estimateCredits({ model: n.model, batch }) : 0;
    return { nodeId: n.id, label: n.label, model: n.model, credits };
  });
  return { nodes, totalCredits: nodes.reduce((sum, x) => sum + x.credits, 0) };
}

export function PipelineBoard({ pipelines, tasks, campaigns }: PipelineBoardProps) {
  const [builderTarget, setBuilderTarget] = useState<BuilderTarget>(null);
  const [runTarget, setRunTarget] = useState<RunTarget>(null);

  return (
    <div>
      <div className="mb-4 flex items-center justify-between gap-3">
        <span className="text-[12.5px] text-os-dim">
          {pipelines.length} pipeline{pipelines.length === 1 ? '' : 's'} saved
        </span>
      </div>

      <div className="grid grid-cols-1 gap-5 md:grid-cols-2 xl:grid-cols-3">
        {pipelines.map((pipeline) => (
          <PipelineCard
            key={pipeline.id}
            pipeline={pipeline}
            tasks={tasks}
            onEdit={() => setBuilderTarget({ mode: 'edit', pipeline })}
            onRun={() => setRunTarget({ pipeline, asForm: false })}
            onRunForm={() => setRunTarget({ pipeline, asForm: true })}
          />
        ))}
        <NewPipelineCard onClick={() => setBuilderTarget({ mode: 'new' })} />
        <DraftChatCard onDrafted={(input) => setBuilderTarget({ mode: 'draft', input })} />
      </div>

      {builderTarget && (
        <PipelineBuilder
          pipeline={builderTarget.mode === 'edit' ? builderTarget.pipeline : null}
          draft={builderTarget.mode === 'draft' ? builderTarget.input : null}
          onClose={() => setBuilderTarget(null)}
        />
      )}

      {runTarget && (
        <RunModal
          pipeline={runTarget.pipeline}
          campaigns={campaigns}
          asForm={runTarget.asForm}
          onClose={() => setRunTarget(null)}
        />
      )}
    </div>
  );
}

function PipelineCard({
  pipeline,
  tasks,
  onEdit,
  onRun,
  onRunForm,
}: {
  pipeline: Pipeline;
  tasks: ProductionTask[];
  onEdit: () => void;
  onRun: () => void;
  onRunForm: () => void;
}) {
  const columns = useMemo(() => flowColumns(pipeline.nodes, pipeline.edges), [pipeline.nodes, pipeline.edges]);
  const hasInputs = pipeline.nodes.some((n) => n.kind === 'input');

  return (
    <div className="glass-panel hoverable flex flex-col gap-4 px-5 py-4">
      <div className="flex items-start justify-between gap-3">
        <div className="min-w-0">
          <div className="truncate text-[15px] font-medium">{pipeline.name}</div>
          {pipeline.description && (
            <p className="mt-1 line-clamp-2 text-[12px] leading-relaxed text-os-dim">{pipeline.description}</p>
          )}
        </div>
        <button
          type="button"
          onClick={onEdit}
          aria-label={`Edit ${pipeline.name}`}
          className="glass-chip grid h-7 w-7 shrink-0 place-items-center text-os-dim transition-colors hover:text-os-text"
        >
          <Pencil className="h-3.5 w-3.5" strokeWidth={1.8} />
        </button>
      </div>

      {/* The glyph can outgrow the card; a fade tells the eye it scrolls
          instead of reading as a hard clip. */}
      <div
        className="overflow-x-auto"
        style={{
          maskImage: 'linear-gradient(90deg, #000 88%, transparent)',
          WebkitMaskImage: 'linear-gradient(90deg, #000 88%, transparent)',
        }}
      >
        <FlowGlyph columns={columns} tasks={tasks} pipelineId={pipeline.id} />
      </div>

      <div className="flex items-center justify-between gap-2 border-t border-[var(--hairline)] pt-3">
        <span className="shrink-0 font-mono text-[10.5px] text-os-dim">
          {pipeline.nodes.length} node{pipeline.nodes.length === 1 ? '' : 's'}
        </span>
        <div className="flex items-center gap-1.5">
          {hasInputs && (
            <button type="button" onClick={onRunForm} className="c-btn">
              <FileText className="h-3 w-3" strokeWidth={1.8} /> Run as form
            </button>
          )}
          <button type="button" onClick={onRun} className="c-btn c-btn-primary">
            <Play className="h-3 w-3" strokeWidth={1.8} /> Run
          </button>
        </div>
      </div>
    </div>
  );
}

function FlowGlyph({ columns, tasks, pipelineId }: { columns: PipelineNode[][]; tasks: ProductionTask[]; pipelineId: string }) {
  if (columns.length === 0) {
    return <p className="text-[11px] text-os-dim">No nodes yet</p>;
  }
  return (
    <div className="flex min-w-max items-center gap-1 pb-1">
      {columns.map((col, i) => (
        <div key={i} className="flex items-center">
          {i > 0 && <span className="h-px w-[14px] shrink-0" style={{ background: 'rgba(255,255,255,0.12)' }} />}
          <div className="flex flex-col gap-1.5">
            {col.map((node) => (
              <FlowNodeChip key={node.id} node={node} tasks={tasks} pipelineId={pipelineId} />
            ))}
          </div>
        </div>
      ))}
    </div>
  );
}

function FlowNodeChip({ node, tasks, pipelineId }: { node: PipelineNode; tasks: ProductionTask[]; pipelineId: string }) {
  const Icon = KIND_ICON[node.kind];
  // Most recent matching task — pipelines are re-run over time, and only
  // the latest run's status is meaningful on the card glyph.
  const task = tasks.filter((t) => t.pipelineId === pipelineId && t.nodeId === node.id).at(-1);
  return (
    <span className="glass-chip flex items-center gap-1.5 px-2 py-1" title={node.label}>
      <Icon className="h-3 w-3 shrink-0 text-os-dim" strokeWidth={1.8} />
      <span className="max-w-[92px] truncate font-mono text-[10px] text-os-muted">{node.label}</span>
      {task && <Dot state={STATUS_DOT[task.status]} />}
    </span>
  );
}

function NewPipelineCard({ onClick }: { onClick: () => void }) {
  return (
    <button
      type="button"
      onClick={onClick}
      className="glass-panel hoverable flex min-h-[180px] flex-col items-center justify-center gap-2 px-5 py-6 text-os-dim transition-colors hover:text-os-text"
    >
      <Plus className="h-5 w-5" strokeWidth={1.8} />
      <span className="text-[13px]">New pipeline</span>
    </button>
  );
}

function DraftChatCard({ onDrafted }: { onDrafted: (input: PipelineInput) => void }) {
  const [brief, setBrief] = useState('');
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [unavailable, setUnavailable] = useState(false);

  async function draft() {
    const trimmed = brief.trim();
    if (!trimmed) return;
    setBusy(true);
    setError(null);
    setUnavailable(false);
    try {
      const res = await fetch('/api/content/pipelines/draft', {
        method: 'POST',
        headers: { 'content-type': 'application/json' },
        body: JSON.stringify({ brief: trimmed }),
      });
      const body = await res.json().catch(() => null);
      if (!res.ok) {
        setUnavailable(res.status === 503);
        setError(typeof body?.error === 'string' ? body.error : `drafting failed (HTTP ${res.status})`);
        return;
      }
      onDrafted(body.pipeline as PipelineInput);
      setBrief('');
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="glass-panel flex min-h-[180px] flex-col gap-3 px-5 py-5">
      <span className="glass-label">Draft with chat</span>
      <textarea
        value={brief}
        onChange={(e) => setBrief(e.target.value)}
        rows={4}
        placeholder="Describe the pipeline and I'll draft it…"
        className="glass-chip flex-1 resize-none border-none px-3 py-2.5 text-[12.5px] leading-relaxed text-os-text outline-none placeholder:text-os-dim"
      />
      <button type="button" onClick={draft} disabled={busy || !brief.trim()} className="c-btn c-btn-primary self-start">
        <Sparkles className="h-3 w-3" strokeWidth={1.8} /> {busy ? 'Drafting…' : 'Draft'}
      </button>
      {error && (
        <p className="text-[11.5px] text-os-dim">{unavailable ? 'Drafting assistant unavailable — build manually.' : error}</p>
      )}
    </div>
  );
}

function RunModal({
  pipeline,
  campaigns,
  asForm,
  onClose,
}: {
  pipeline: Pipeline;
  campaigns: Campaign[];
  asForm: boolean;
  onClose: () => void;
}) {
  const router = useRouter();
  // Close melts the face (lib/use-melt); the board unmounts it once it has melted.
  const melt = useMeltClose(onClose);
  const close = melt.close;
  const inputNodes = useMemo(() => pipeline.nodes.filter((n) => n.kind === 'input'), [pipeline.nodes]);
  const startOnForm = asForm && inputNodes.length > 0;
  const [step, setStep] = useState<'form' | 'confirm'>(startOnForm ? 'form' : 'confirm');
  const [inputValues, setInputValues] = useState<Record<string, string>>({});
  const [campaignId, setCampaignId] = useState(campaigns[0]?.id ?? '');
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const preview = useMemo(() => localPreview(pipeline), [pipeline]);

  async function confirm() {
    if (!campaignId) {
      setError('Pick a campaign.');
      return;
    }
    setBusy(true);
    setError(null);
    try {
      const res = await fetch(`/api/content/pipelines/${pipeline.id}/run`, {
        method: 'POST',
        headers: { 'content-type': 'application/json' },
        body: JSON.stringify({ campaignId }),
      });
      if (!res.ok) {
        const body = await res.json().catch(() => null);
        throw new Error(typeof body?.error === 'string' ? body.error : `run failed (HTTP ${res.status})`);
      }
      router.refresh();
      close();
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="fixed inset-0 z-[100] flex items-center justify-center p-6" role="presentation" {...melt.layer}>
      {/* The scrim is the face's sibling (glass brief §3.2): it frosts and dims the page; the face frosts on its own and melts on close. */}
      <div className="gl-scrim" aria-hidden onClick={close} />
      <div
        ref={melt.faceRef}
        onAnimationEnd={melt.onAnimationEnd}
        className={`glass-panel glass-face--strong gl-materialise relative flex w-full max-w-[560px] flex-col overflow-hidden${melt.melting ? ' is-melting' : ''}`}
        style={{ maxHeight: '85vh' }}
        role="dialog"
        aria-modal="true"
        aria-label={`Run ${pipeline.name}`}
      >
        <button
          onClick={close}
          aria-label="Close"
          className="absolute right-4 top-4 z-10 grid h-8 w-8 place-items-center rounded-full border border-[rgb(var(--mist-rgb) / 0.14)] bg-black/30 text-os-dim transition-colors hover:text-os-text"
        >
          <X className="h-4 w-4" />
        </button>

        <div className="flex-1 overflow-y-auto px-6 py-6">
          <h2 className="pr-10 text-[16px] font-medium leading-tight">
            {step === 'form' ? `${pipeline.name} — inputs` : `Run ${pipeline.name}`}
          </h2>

          {step === 'form' && (
            <div className="mt-5 flex flex-col gap-4">
              {inputNodes.map((n) => (
                <div key={n.id} className="flex flex-col gap-1.5">
                  <span className="glass-label">{n.label}</span>
                  {n.inputKind === 'text' ? (
                    <input
                      value={inputValues[n.id] ?? ''}
                      onChange={(e) => setInputValues((v) => ({ ...v, [n.id]: e.target.value }))}
                      placeholder="Paste the value"
                      className="glass-chip border-none px-3 py-2 text-[13px] text-os-text outline-none placeholder:text-os-dim"
                    />
                  ) : (
                    <p className="text-[12px] text-os-dim">
                      Provide the {n.inputKind ?? 'reference'} for this input during fulfillment — the operator picks it up
                      when this run reaches it.
                    </p>
                  )}
                </div>
              ))}
              <p className="text-[11px] text-os-dim">
                This queues the pipeline exactly as drawn — the operator resolves each input against the values above during
                fulfillment.
              </p>
            </div>
          )}

          {step === 'confirm' && (
            <div className="mt-5 flex flex-col gap-3">
              {preview.nodes.length === 0 ? (
                <p className="text-[12.5px] text-os-dim">This pipeline has no generate or transform nodes to run.</p>
              ) : (
                <ol className="flex flex-col gap-2">
                  {preview.nodes.map((n, i) => (
                    <li key={n.nodeId} className="glass-chip flex items-center justify-between gap-3 px-3 py-2">
                      <span className="flex min-w-0 items-center gap-2 text-[12.5px] text-os-text">
                        <span className="shrink-0 font-mono text-[10.5px] text-os-dim">{i + 1}</span>
                        <span className="truncate">{n.label}</span>
                        <span className="shrink-0 font-mono text-[10.5px] text-os-dim">· {n.model}</span>
                      </span>
                      <span className="shrink-0 font-mono text-[11.5px] tabular-nums text-os-muted">~{n.credits} cr</span>
                    </li>
                  ))}
                </ol>
              )}
              <label className="flex flex-col gap-1.5">
                <span className="glass-label">Campaign</span>
                <select
                  value={campaignId}
                  onChange={(e) => setCampaignId(e.target.value)}
                  className="glass-chip border-none px-3 py-2 text-[13px] text-os-text outline-none"
                >
                  <option value="">Pick a campaign…</option>
                  {campaigns.map((c) => (
                    <option key={c.id} value={c.id}>
                      {c.name}
                    </option>
                  ))}
                </select>
              </label>
              <div className="flex items-center justify-between border-t border-[var(--hairline)] pt-3">
                <span className="text-[12px] text-os-dim">Total</span>
                <span className="font-mono text-[13px] tabular-nums text-os-text">~{preview.totalCredits} credits</span>
              </div>
            </div>
          )}

          {error && (
            <p className="mt-4 flex items-center gap-2 text-[12.5px] text-os-muted">
              <span className="dot err" />
              {error}
            </p>
          )}
        </div>

        <div className="flex shrink-0 items-center justify-end gap-2 border-t border-[rgb(var(--mist-rgb) / 0.1)] px-6 py-4">
          <button type="button" onClick={close} className="c-btn">
            Cancel
          </button>
          {step === 'form' ? (
            <button type="button" onClick={() => setStep('confirm')} className="c-btn c-btn-primary">
              Continue
            </button>
          ) : (
            <button
              type="button"
              onClick={confirm}
              disabled={busy || !campaignId || preview.nodes.length === 0}
              className="c-btn c-btn-primary"
            >
              {busy ? 'Queuing…' : 'Confirm & queue'}
            </button>
          )}
        </div>
      </div>
    </div>
  );
}
