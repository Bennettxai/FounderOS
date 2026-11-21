'use client';

/**
 * The pipeline builder — CONTENT-V3-SCAFFOLD.md §6. A full-width overlay,
 * same chrome and open/close idioms as WorkflowBuilder (glass backdrop,
 * Escape/backdrop-click to close, two-step delete confirm), but the canvas
 * itself follows WorkflowTree's visual language instead of WorkflowBuilder's
 * flat step list: a branching left-to-right flowchart of clickable node
 * chips, click a node to open its detail in a side panel. This is NOT a
 * free-form drag canvas — nodes insert at named points ("add after this
 * node") and connections are explicit toggles, a structured flowchart with
 * insert points, like the workflows builder.
 *
 * validatePipeline() is server-only (lib/content-pipelines.ts pulls in
 * node:fs), so this component never imports it directly — Save/PUT sends
 * the draft to the API, and a 400 response's `problems` array renders
 * inline exactly where a client-side check would have.
 */
import { useEffect, useMemo, useRef, useState, type Dispatch, type SetStateAction } from 'react';
import { useRouter } from 'next/navigation';
import { Plus, Send, Sparkles, Trash2, Upload, Wand2, X, type LucideIcon } from 'lucide-react';
import { ParamFields, type ParamValues } from '@/components/content/ParamFields';
import {
  MODEL_FAMILIES,
  ROLE_KIND,
  ROLE_LABELS,
  defaultParams,
  findModel,
  modelsByFamily,
  type HfMediaRole,
} from '@/lib/higgsfield/catalog';
import type {
  InputKind,
  MediaBinding,
  MediaSourceType,
  Pipeline,
  PipelineEdge,
  PipelineInput,
  PipelineNode,
  PipelineNodeKind,
} from '@/lib/content-pipelines';
import { useMeltClose } from '@/lib/use-melt';
import { Layer } from '@/components/g/Overlays';
import { pushEscape } from '@/lib/escape-stack';

const NODE_KINDS: PipelineNodeKind[] = ['input', 'generate', 'transform', 'output'];
const NODE_KIND_LABEL: Record<PipelineNodeKind, string> = {
  input: 'Input',
  generate: 'Generate',
  transform: 'Transform',
  output: 'Output',
};
const KIND_ICON: Record<PipelineNodeKind, LucideIcon> = {
  input: Upload,
  generate: Sparkles,
  transform: Wand2,
  output: Send,
};
const MEDIA_SOURCE_TYPES: MediaSourceType[] = ['upload', 'library', 'identity', 'text', 'node'];
const MEDIA_SOURCE_LABEL: Record<MediaSourceType, string> = {
  upload: 'Upload',
  library: 'Library asset',
  identity: 'Identity asset',
  text: 'Literal text',
  node: 'Upstream node',
};

function newNodeId(kind: PipelineNodeKind): string {
  const rand = typeof crypto !== 'undefined' && 'randomUUID' in crypto ? crypto.randomUUID() : `${Date.now()}-${Math.random()}`;
  return `${kind}-${rand.slice(0, 8)}`;
}

function emptyNode(kind: PipelineNodeKind): PipelineNode {
  return {
    id: newNodeId(kind),
    kind,
    label: kind === 'input' ? 'New input' : kind === 'output' ? 'Deliver to campaign' : 'New node',
    model: null,
    params: {},
    aspectRatio: null,
    duration: null,
    prompt: null,
    mediaBindings: [],
    inputKind: kind === 'input' ? 'image' : null,
  };
}

/** Same longest-path column layout as PipelineBoard's flow glyph, at
 *  builder scale — duplicated locally rather than imported to avoid a
 *  circular module dependency (PipelineBoard already imports this file). */
function flowColumns(nodes: PipelineNode[], edges: PipelineEdge[]): PipelineNode[][] {
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

/** A node's output media kind, for the source-picker's own type filter —
 *  mirrors nodeOutputKind() in lib/content-pipelines.ts. */
function nodeOutputKind(node: PipelineNode): 'image' | 'video' | 'audio' | 'text' | null {
  if (node.kind === 'input') return node.inputKind;
  if ((node.kind === 'generate' || node.kind === 'transform') && node.model) return findModel(node.model)?.output ?? null;
  return null;
}

function eligibleUpstreamNodes(nodes: PipelineNode[], selfId: string, role: string): PipelineNode[] {
  const roleKind = ROLE_KIND[role as HfMediaRole];
  if (!roleKind) return [];
  return nodes.filter((n) => n.id !== selfId && nodeOutputKind(n) === roleKind);
}

export function PipelineBuilder({
  pipeline,
  draft,
  onClose,
}: {
  /** Non-null opens the builder in edit mode (PUT on save). */
  pipeline: Pipeline | null;
  /** A drafted-but-unsaved PipelineInput (from "Draft with chat") to
   *  prefill a new pipeline with — ignored when `pipeline` is set. */
  draft?: PipelineInput | null;
  onClose: () => void;
}) {
  const router = useRouter();
  // Close melts the face (lib/use-melt); the board unmounts it once it has melted.
  const melt = useMeltClose(onClose);
  const close = melt.close;
  const editing = pipeline !== null;
  const initial = pipeline ?? draft ?? null;

  const [name, setName] = useState(initial?.name ?? '');
  const [description, setDescription] = useState(initial?.description ?? '');
  const [nodes, setNodes] = useState<PipelineNode[]>(initial?.nodes ?? []);
  const [edges, setEdges] = useState<PipelineEdge[]>(initial?.edges ?? []);
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [problems, setProblems] = useState<string[]>([]);
  const [deleteArmed, setDeleteArmed] = useState(false);
  const [deleting, setDeleting] = useState(false);

  useEffect(() => {
    const prevOverflow = document.body.style.overflow;
    document.body.style.overflow = 'hidden';
    function onKey(e: KeyboardEvent) {
      if (e.key === 'Escape') close();
    }
    window.addEventListener('keydown', onKey);
    return () => {
      document.body.style.overflow = prevOverflow;
      window.removeEventListener('keydown', onKey);
    };
  }, [close]);

  const columns = useMemo(() => flowColumns(nodes, edges), [nodes, edges]);
  const selectedNode = nodes.find((n) => n.id === selectedId) ?? null;

  function addNode(afterId: string | null, kind: PipelineNodeKind) {
    const node = emptyNode(kind);
    setNodes((prev) => {
      if (afterId === null) return [...prev, node];
      const idx = prev.findIndex((n) => n.id === afterId);
      if (idx === -1) return [...prev, node];
      return [...prev.slice(0, idx + 1), node, ...prev.slice(idx + 1)];
    });
    if (afterId !== null) setEdges((prev) => [...prev, { from: afterId, to: node.id }]);
    setSelectedId(node.id);
  }

  function deleteNode(id: string) {
    setNodes((prev) =>
      prev
        .filter((n) => n.id !== id)
        .map((n) => ({ ...n, mediaBindings: n.mediaBindings.filter((b) => !(b.source.type === 'node' && b.source.ref === id)) })),
    );
    setEdges((prev) => prev.filter((e) => e.from !== id && e.to !== id));
    setSelectedId((cur) => (cur === id ? null : cur));
  }

  function buildPayload(): PipelineInput {
    // Drop any media binding whose ref was never filled in (e.g. a fresh
    // "Upstream node" row before a node is picked) — the schema requires a
    // non-empty ref, and an incomplete row is honestly "not set" rather
    // than something to send.
    return {
      name: name.trim(),
      description: description.trim(),
      nodes: nodes.map((n) => ({ ...n, mediaBindings: n.mediaBindings.filter((b) => b.source.ref.trim().length > 0) })),
      edges,
    };
  }

  async function submit() {
    setError(null);
    setProblems([]);
    if (!name.trim()) {
      setError('Give the pipeline a name.');
      return;
    }
    setBusy(true);
    try {
      const res = await fetch(editing ? `/api/content/pipelines/${pipeline!.id}` : '/api/content/pipelines', {
        method: editing ? 'PUT' : 'POST',
        headers: { 'content-type': 'application/json' },
        body: JSON.stringify(buildPayload()),
      });
      const body = await res.json().catch(() => null);
      if (!res.ok) {
        if (Array.isArray(body?.problems)) {
          setProblems(body.problems as string[]);
          setError('Fix the problems below before saving.');
        } else {
          setError(typeof body?.error === 'string' ? body.error : `save failed (HTTP ${res.status})`);
        }
        return;
      }
      router.refresh();
      close();
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setBusy(false);
    }
  }

  async function confirmDelete() {
    if (!pipeline) return;
    if (!deleteArmed) {
      setDeleteArmed(true);
      return;
    }
    setDeleting(true);
    setError(null);
    try {
      const res = await fetch(`/api/content/pipelines/${pipeline.id}`, { method: 'DELETE' });
      if (!res.ok) throw new Error(`delete failed (HTTP ${res.status})`);
      router.refresh();
      close();
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
      setDeleteArmed(false);
    } finally {
      setDeleting(false);
    }
  }

  // Portalled to <body>: under a frosted ancestor the fixed layer would be trapped.
  return (
    <Layer>
      <div className="fixed inset-0 z-[100] flex items-center justify-center p-4" role="presentation" {...melt.layer}>
        {/* The scrim is the face's sibling (glass brief §3.2): it frosts and dims the page; the face frosts on its own and melts on close. */}
        <div className="gl-scrim" aria-hidden onClick={close} />
        <div
          ref={melt.faceRef}
          onAnimationEnd={melt.onAnimationEnd}
          className={`glass-panel glass-face--strong gl-materialise relative flex w-full max-w-[1180px] flex-col overflow-hidden${melt.melting ? ' is-melting' : ''}`}
          style={{ maxHeight: '92vh' }}
          role="dialog"
          aria-modal="true"
          aria-label={editing ? `Edit ${pipeline!.name}` : 'New pipeline'}
        >
          <button
            onClick={close}
            aria-label="Close"
            className="absolute right-4 top-4 z-10 grid h-8 w-8 place-items-center rounded-full border border-[rgb(var(--mist-rgb) / 0.14)] bg-black/30 text-os-dim transition-colors hover:text-os-text"
          >
            <X className="h-4 w-4" />
          </button>

          <div className="flex-1 overflow-y-auto px-7 py-7">
            <h2 className="pr-10 text-[18px] font-medium leading-tight">{editing ? 'Edit pipeline' : 'New pipeline'}</h2>

            <section className="mt-5 grid gap-3 sm:grid-cols-2">
              <label className="flex flex-col gap-1.5">
                <span className="glass-label">Name</span>
                <input
                  value={name}
                  onChange={(e) => setName(e.target.value)}
                  placeholder="e.g. Thumbnail pack"
                  className="glass-chip border-none px-3 py-2.5 text-[13px] text-os-text outline-none placeholder:text-os-dim"
                />
              </label>
              <label className="flex flex-col gap-1.5">
                <span className="glass-label">Description</span>
                <input
                  value={description}
                  onChange={(e) => setDescription(e.target.value)}
                  placeholder="What this pipeline delivers, in one line"
                  className="glass-chip border-none px-3 py-2.5 text-[13px] text-os-text outline-none placeholder:text-os-dim"
                />
              </label>
            </section>

            <section className="mt-6 grid gap-5 lg:grid-cols-[minmax(0,1fr)_360px]">
              <div className="overflow-x-auto">
                <div className="mb-3 flex items-center justify-between">
                  <span className="glass-label">Flow</span>
                  <InsertMenu onPick={(kind) => addNode(nodes.at(-1)?.id ?? null, kind)} label="Add node" />
                </div>
                {nodes.length === 0 ? (
                  <div className="flex flex-col items-center gap-3 py-10 text-center">
                    <p className="text-[12.5px] text-os-dim">No nodes yet.</p>
                    <InsertMenu onPick={(kind) => addNode(null, kind)} label="Add the first node" />
                  </div>
                ) : (
                  <div className="flex min-w-max items-start gap-3 pb-2">
                    {columns.map((col, ci) => (
                      <div key={ci} className="flex items-center">
                        {ci > 0 && <span className="mx-1 h-px w-[18px] shrink-0" style={{ background: 'rgba(255,255,255,0.14)' }} />}
                        <div className="flex flex-col gap-2.5">
                          {col.map((node) => (
                            <div key={node.id} className="flex items-center gap-1.5">
                              <NodeChip node={node} selected={node.id === selectedId} onSelect={() => setSelectedId(node.id)} onDelete={() => deleteNode(node.id)} />
                              <InsertMenu compact onPick={(kind) => addNode(node.id, kind)} />
                            </div>
                          ))}
                        </div>
                      </div>
                    ))}
                  </div>
                )}
              </div>

              {selectedNode ? (
                <NodeDetailPanel node={selectedNode} nodes={nodes} edges={edges} setNodes={setNodes} setEdges={setEdges} onClose={() => setSelectedId(null)} />
              ) : (
                <div className="glass-panel-violet flex min-h-[220px] flex-col items-center justify-center gap-1 px-5 py-8 text-center">
                  <span className="text-[12px] text-os-dim">Select a node to edit it.</span>
                </div>
              )}
            </section>

            {problems.length > 0 && (
              <section className="mt-5">
                <span className="glass-label">Problems</span>
                <ul className="mt-2 flex flex-col gap-1.5">
                  {problems.map((p, i) => (
                    <li key={i} className="flex items-start gap-2 text-[12px] leading-relaxed text-os-muted">
                      <span className="mt-[6px] h-1.5 w-1.5 shrink-0 rounded-full" style={{ background: 'var(--accent)' }} />
                      {p}
                    </li>
                  ))}
                </ul>
              </section>
            )}

            {error && (
              <p className="mt-4 flex items-center gap-2 text-[12.5px] text-os-muted">
                <span className="dot err" />
                {error}
              </p>
            )}
          </div>

          <div className="flex shrink-0 items-center justify-between gap-3 border-t border-[rgb(var(--mist-rgb) / 0.1)] px-6 py-4">
            <div>
              {editing && (
                <button
                  type="button"
                  onClick={confirmDelete}
                  disabled={deleting}
                  className="flex items-center gap-1.5 rounded-sm-t border border-[color-mix(in_oklab,var(--err)_35%,transparent)] bg-[color-mix(in_oklab,var(--err)_9%,transparent)] px-3 py-2 text-[12.5px] text-os-err transition-colors"
                >
                  <Trash2 className="h-3.5 w-3.5" />
                  {deleting ? 'Deleting…' : deleteArmed ? 'Really delete? Click again' : 'Delete pipeline'}
                </button>
              )}
            </div>
            <div className="flex items-center gap-2">
              <button type="button" onClick={close} className="c-btn">
                Cancel
              </button>
              <button type="button" onClick={submit} disabled={busy} className="c-btn c-btn-primary">
                {busy ? 'Saving…' : 'Save'}
              </button>
            </div>
          </div>
        </div>
      </div>
    </Layer>
  );
}

function InsertMenu({ onPick, compact, label }: { onPick: (kind: PipelineNodeKind) => void; compact?: boolean; label?: string }) {
  const [open, setOpen] = useState(false);
  const root = useRef<HTMLDivElement>(null);
  // Outside press or Escape closes the menu (the GPopover pattern). No fixed click-away: inside the
  // modal's frosted face it would be trapped to the face instead of covering the page.
  useEffect(() => {
    if (!open) return;
    const off = pushEscape(() => setOpen(false));
    const onDown = (e: PointerEvent) => {
      if (root.current && !root.current.contains(e.target as Node)) setOpen(false);
    };
    document.addEventListener('pointerdown', onDown);
    return () => {
      off();
      document.removeEventListener('pointerdown', onDown);
    };
  }, [open]);
  return (
    <div ref={root} className="relative">
      <button
        type="button"
        onClick={() => setOpen((v) => !v)}
        aria-label={compact ? 'Insert node' : label}
        className={compact ? 'glass-chip grid h-8 w-8 shrink-0 place-items-center text-os-dim hover:text-os-text' : 'c-btn'}
      >
        <Plus className="h-3 w-3" strokeWidth={1.8} /> {!compact && (label ?? 'Add node')}
      </button>
      {open && (
        <div className="c-pop absolute left-0 top-full z-20 mt-1 flex flex-col gap-0.5 p-1">
          {NODE_KINDS.map((kind) => (
            <button
              key={kind}
              type="button"
              onClick={() => {
                onPick(kind);
                setOpen(false);
              }}
              className="whitespace-nowrap rounded-[6px] px-2.5 py-1.5 text-left text-[11.5px] text-os-muted transition-colors hover:bg-white/5 hover:text-os-text"
            >
              {NODE_KIND_LABEL[kind]}
            </button>
          ))}
        </div>
      )}
    </div>
  );
}

function NodeChip({
  node,
  selected,
  onSelect,
  onDelete,
}: {
  node: PipelineNode;
  selected: boolean;
  onSelect: () => void;
  onDelete: () => void;
}) {
  const Icon = KIND_ICON[node.kind];
  return (
    <div
      className="glass-chip flex items-center gap-2 px-3 py-2 transition-colors"
      style={selected ? { borderColor: 'var(--accent-line)' } : undefined}
    >
      <button type="button" onClick={onSelect} className="flex min-w-0 items-center gap-2 text-left">
        <Icon className="h-3.5 w-3.5 shrink-0" style={{ color: selected ? 'var(--accent)' : 'var(--text-2)' }} strokeWidth={1.8} />
        <span className="max-w-[130px] truncate text-[12px] text-os-text">{node.label}</span>
      </button>
      <button type="button" onClick={onDelete} aria-label={`Delete ${node.label}`} className="shrink-0 text-os-dim hover:text-os-err">
        <X className="h-3 w-3" strokeWidth={1.8} />
      </button>
    </div>
  );
}

function NodeDetailPanel({
  node,
  nodes,
  edges,
  setNodes,
  setEdges,
  onClose,
}: {
  node: PipelineNode;
  nodes: PipelineNode[];
  edges: PipelineEdge[];
  setNodes: Dispatch<SetStateAction<PipelineNode[]>>;
  setEdges: Dispatch<SetStateAction<PipelineEdge[]>>;
  onClose: () => void;
}) {
  const model = node.model ? findModel(node.model) : undefined;

  function patch(next: Partial<PipelineNode>) {
    setNodes((prev) => prev.map((n) => (n.id === node.id ? { ...n, ...next } : n)));
  }

  function onModelChange(modelId: string) {
    const nextModel = findModel(modelId);
    if (!nextModel) {
      patch({ model: null });
      return;
    }
    const preservedBatch: Record<string, number> = typeof node.params.batch === 'number' ? { batch: node.params.batch } : {};
    const validRoles = new Set<string>(nextModel.medias.map((m) => m.role));
    patch({
      model: modelId,
      params: { ...defaultParams(nextModel), ...preservedBatch },
      mediaBindings: node.mediaBindings.filter((b) => validRoles.has(b.role)),
      aspectRatio: node.aspectRatio && nextModel.aspectRatios.includes(node.aspectRatio) ? node.aspectRatio : null,
      duration: null,
    });
  }

  function setFixedBinding(role: string, next: MediaBinding | null) {
    const others = node.mediaBindings.filter((b) => b.role !== role);
    patch({ mediaBindings: next ? [...others, next] : others });
    if (next?.source.type === 'node' && next.source.ref) {
      setEdges((prev) => (prev.some((e) => e.from === next.source.ref && e.to === node.id) ? prev : [...prev, { from: next.source.ref, to: node.id }]));
    }
  }

  function setIndexedBinding(index: number, next: MediaBinding | null) {
    const bindings = [...node.mediaBindings];
    if (next === null) bindings.splice(index, 1);
    else bindings[index] = next;
    patch({ mediaBindings: bindings });
    if (next?.source.type === 'node' && next.source.ref) {
      setEdges((prev) => (prev.some((e) => e.from === next.source.ref && e.to === node.id) ? prev : [...prev, { from: next.source.ref, to: node.id }]));
    }
  }

  const otherNodes = nodes.filter((n) => n.id !== node.id);

  return (
    <div className="glass-panel-violet flex flex-col gap-4 overflow-y-auto px-5 py-5" style={{ maxHeight: '62vh' }}>
      <div className="flex items-start justify-between gap-2">
        <span className="glass-label">Node</span>
        <button onClick={onClose} aria-label="Close node detail" className="glass-chip grid h-6 w-6 place-items-center text-os-dim hover:text-os-text">
          <X className="h-3 w-3" strokeWidth={1.8} />
        </button>
      </div>

      <label className="flex flex-col gap-1.5">
        <span className="text-[10.5px] text-os-dim">Label</span>
        <input
          value={node.label}
          onChange={(e) => patch({ label: e.target.value })}
          className="glass-chip border-none px-3 py-2 text-[13px] text-os-text outline-none"
        />
      </label>

      <label className="flex flex-col gap-1.5">
        <span className="text-[10.5px] text-os-dim">Kind</span>
        <select
          value={node.kind}
          onChange={(e) => {
            const kind = e.target.value as PipelineNodeKind;
            if (kind === 'input') patch({ kind, model: null, mediaBindings: [], inputKind: node.inputKind ?? 'image' });
            else if (kind === 'output') patch({ kind, model: null, inputKind: null });
            else patch({ kind, inputKind: null });
          }}
          className="glass-chip border-none px-3 py-2 text-[13px] text-os-text outline-none"
        >
          {NODE_KINDS.map((k) => (
            <option key={k} value={k}>
              {NODE_KIND_LABEL[k]}
            </option>
          ))}
        </select>
      </label>

      {node.kind === 'input' && (
        <label className="flex flex-col gap-1.5">
          <span className="text-[10.5px] text-os-dim">Input kind</span>
          <select
            value={node.inputKind ?? 'image'}
            onChange={(e) => patch({ inputKind: e.target.value as InputKind })}
            className="glass-chip border-none px-3 py-2 text-[13px] text-os-text outline-none"
          >
            <option value="image">Image</option>
            <option value="video">Video</option>
            <option value="audio">Audio</option>
            <option value="text">Text (URL / literal value)</option>
          </select>
        </label>
      )}

      {(node.kind === 'generate' || node.kind === 'transform') && (
        <>
          <label className="flex flex-col gap-1.5">
            <span className="text-[10.5px] text-os-dim">Model</span>
            <select value={node.model ?? ''} onChange={(e) => onModelChange(e.target.value)} className="glass-chip border-none px-3 py-2 text-[13px] text-os-text outline-none">
              <option value="">Pick a model…</option>
              {MODEL_FAMILIES.map((family) => {
                const models = modelsByFamily(family);
                if (models.length === 0) return null;
                return (
                  <optgroup key={family} label={family}>
                    {models.map((m) => (
                      <option key={m.id} value={m.id}>
                        {m.name} ({m.id})
                      </option>
                    ))}
                  </optgroup>
                );
              })}
            </select>
          </label>

          {model && (
            <>
              {model.aspectRatios.length > 0 && (
                <label className="flex flex-col gap-1.5">
                  <span className="text-[10.5px] text-os-dim">Aspect ratio</span>
                  <select
                    value={node.aspectRatio ?? ''}
                    onChange={(e) => patch({ aspectRatio: e.target.value || null })}
                    className="glass-chip border-none px-3 py-2 text-[13px] text-os-text outline-none"
                  >
                    <option value="">Not set</option>
                    {model.aspectRatios.map((a) => (
                      <option key={a} value={a}>
                        {a}
                      </option>
                    ))}
                  </select>
                </label>
              )}

              {(model.durations || model.durationRange) && (
                <label className="flex flex-col gap-1.5">
                  <span className="text-[10.5px] text-os-dim">Duration (seconds)</span>
                  {model.durations ? (
                    <select
                      value={node.duration ?? ''}
                      onChange={(e) => patch({ duration: e.target.value ? Number(e.target.value) : null })}
                      className="glass-chip border-none px-3 py-2 text-[13px] text-os-text outline-none"
                    >
                      <option value="">Not set</option>
                      {model.durations.map((d) => (
                        <option key={d} value={d}>
                          {d}s
                        </option>
                      ))}
                    </select>
                  ) : (
                    <input
                      type="number"
                      min={model.durationRange!.min}
                      max={model.durationRange!.max}
                      value={node.duration ?? ''}
                      onChange={(e) => patch({ duration: e.target.value ? Number(e.target.value) : null })}
                      className="glass-chip border-none px-3 py-2 text-[13px] text-os-text outline-none"
                    />
                  )}
                </label>
              )}

              <label className="flex flex-col gap-1.5">
                <span className="text-[10.5px] text-os-dim">Batch (parallel runs)</span>
                <input
                  type="number"
                  min={1}
                  max={8}
                  value={typeof node.params.batch === 'number' ? node.params.batch : 1}
                  onChange={(e) => patch({ params: { ...node.params, batch: Math.max(1, Math.min(8, Number(e.target.value) || 1)) } })}
                  className="glass-chip w-24 border-none px-3 py-2 text-[13px] text-os-text outline-none"
                />
              </label>

              <label className="flex flex-col gap-1.5">
                <span className="text-[10.5px] text-os-dim">Prompt</span>
                <textarea
                  value={node.prompt ?? ''}
                  onChange={(e) => patch({ prompt: e.target.value })}
                  rows={3}
                  className="glass-chip resize-none border-none px-3 py-2 text-[12.5px] leading-relaxed text-os-text outline-none"
                />
              </label>

              <div className="flex flex-col gap-1">
                <span className="text-[10.5px] text-os-dim">Parameters</span>
                <ParamFields
                  model={model}
                  values={node.params as ParamValues}
                  onChange={(paramName, value) => patch({ params: { ...node.params, [paramName]: value } })}
                />
              </div>

              {model.medias.length > 0 && (
                <div className="flex flex-col gap-2.5">
                  <span className="text-[10.5px] text-os-dim">Media bindings</span>
                  {model.medias.map((slot) => (
                    <MediaBindingRow
                      key={slot.role}
                      role={slot.role}
                      binding={node.mediaBindings.find((b) => b.role === slot.role)}
                      nodes={nodes}
                      selfId={node.id}
                      onChange={(next) => setFixedBinding(slot.role, next)}
                    />
                  ))}
                </div>
              )}
            </>
          )}
        </>
      )}

      {node.kind === 'output' && (
        <div className="flex flex-col gap-2.5">
          <span className="text-[10.5px] text-os-dim">Media bindings</span>
          {node.mediaBindings.map((b, i) => (
            <MediaBindingRow
              key={i}
              role={b.role}
              editableRole
              binding={b}
              nodes={nodes}
              selfId={node.id}
              onRoleChange={(role) => setIndexedBinding(i, { ...b, role })}
              onChange={(next) => setIndexedBinding(i, next)}
            />
          ))}
          <button
            type="button"
            onClick={() => patch({ mediaBindings: [...node.mediaBindings, { role: 'image_references', source: { type: 'node', ref: '' } }] })}
            className="c-btn self-start"
          >
            <Plus className="h-3 w-3" strokeWidth={1.8} /> Add media binding
          </button>
        </div>
      )}

      {otherNodes.length > 0 && (
        <div className="flex flex-col gap-1.5">
          <span className="text-[10.5px] text-os-dim">Connects to</span>
          <div className="flex flex-wrap gap-1.5">
            {otherNodes.map((n) => {
              const connected = edges.some((e) => e.from === node.id && e.to === n.id);
              return (
                <button
                  key={n.id}
                  type="button"
                  onClick={() =>
                    setEdges((prev) =>
                      connected ? prev.filter((e) => !(e.from === node.id && e.to === n.id)) : [...prev, { from: node.id, to: n.id }],
                    )
                  }
                  className={`rounded-full border px-2.5 py-1 text-[10.5px] transition-colors ${
                    connected
                      ? 'border-[var(--accent-line)] bg-[var(--accent-soft)] text-os-accent'
                      : 'border-os-border text-os-dim hover:text-os-muted'
                  }`}
                >
                  {n.label}
                </button>
              );
            })}
          </div>
        </div>
      )}
    </div>
  );
}

function MediaBindingRow({
  role,
  editableRole,
  binding,
  nodes,
  selfId,
  onRoleChange,
  onChange,
}: {
  role: string;
  editableRole?: boolean;
  binding: MediaBinding | undefined;
  nodes: PipelineNode[];
  selfId: string;
  onRoleChange?: (role: string) => void;
  onChange: (next: MediaBinding | null) => void;
}) {
  const sourceType = binding?.source.type ?? '';
  const upstream = eligibleUpstreamNodes(nodes, selfId, role);

  return (
    <div className="flex flex-wrap items-start gap-1.5">
      {editableRole ? (
        <select
          value={role}
          onChange={(e) => onRoleChange?.(e.target.value)}
          className="glass-chip shrink-0 border-none px-2 py-2 text-[11px] text-os-text outline-none"
        >
          {(Object.keys(ROLE_LABELS) as HfMediaRole[]).map((r) => (
            <option key={r} value={r}>
              {ROLE_LABELS[r]}
            </option>
          ))}
        </select>
      ) : (
        <span className="w-[118px] shrink-0 pt-2 text-[11px] text-os-dim">{ROLE_LABELS[role as HfMediaRole] ?? role}</span>
      )}

      <select
        value={sourceType}
        onChange={(e) => {
          const type = e.target.value as MediaSourceType | '';
          if (!type) {
            onChange(null);
            return;
          }
          if (type === 'node') {
            onChange({ role, source: { type: 'node', ref: upstream[0]?.id ?? '' } });
            return;
          }
          onChange({ role, source: { type, ref: '' } });
        }}
        className="glass-chip shrink-0 border-none px-2.5 py-2 text-[11.5px] text-os-text outline-none"
      >
        <option value="">— none —</option>
        {MEDIA_SOURCE_TYPES.map((t) => (
          <option key={t} value={t}>
            {MEDIA_SOURCE_LABEL[t]}
          </option>
        ))}
      </select>

      {sourceType === 'node' ? (
        <select
          value={binding?.source.ref ?? ''}
          onChange={(e) => onChange({ role, source: { type: 'node', ref: e.target.value } })}
          className="glass-chip min-w-0 flex-1 border-none px-2.5 py-2 text-[11.5px] text-os-text outline-none"
        >
          <option value="">Pick a node…</option>
          {upstream.map((n) => (
            <option key={n.id} value={n.id}>
              {n.label}
            </option>
          ))}
        </select>
      ) : sourceType ? (
        <input
          value={binding?.source.ref ?? ''}
          onChange={(e) => onChange({ role, source: { type: sourceType as MediaSourceType, ref: e.target.value } })}
          placeholder={
            sourceType === 'identity'
              ? 'Soul/Element id from the Identity section'
              : sourceType === 'text'
                ? 'Literal value'
                : 'Path under /content-media/'
          }
          className="glass-chip min-w-0 flex-1 border-none px-2.5 py-2 text-[11.5px] text-os-text outline-none placeholder:text-os-dim"
        />
      ) : null}
    </div>
  );
}
