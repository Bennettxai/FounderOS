
/** Theme tokens read live from the canvas root (the document until mount), so the
 *  canvas re-keys with the theme. The root inherits every document token, and under
 *  the GladOS skin it re-points them at --g-* (glados-ui SPEC C28), so the minimap and
 *  library thumbs read the skin the page shows. */
let tokRoot: Element | null = null;
function tok(name: string, fallback: string): string {
  if (typeof document === 'undefined') return fallback;
  const v = getComputedStyle(tokRoot ?? document.documentElement).getPropertyValue(name).trim();
  return v || fallback;
}

function inkA(a: number): string {
  return `rgb(${tok('--wf-ink-rgb', '233 240 237')} / ${a})`;
}
function accentA(a: number): string {
  return `rgb(${tok('--accent-rgb', '0 255 171')} / ${a})`;
}
/* The workflow canvas editor. Ported from the design sandbox
   (~/Desktop/UI design examples/ai flowchart example 2/app.js) on 2026-09-15
   and wired to the real API through `EditorApi`. DOM-driven on purpose: it
   mounts into the static markup WorkflowCanvas.tsx renders and owns it
   until `destroy()`. Nothing in here fabricates data — runs come from
   POST /api/agents/:id/run, rules from /api/brand-deals/rules, versions from
   the version tables. */
/* eslint-disable */
import type { WorkflowInput } from '@/app/api/workflows/shared';
import { BRAND } from '@/lib/brand';
import type { AgentRun, BrandDealRules, ConfigVersion, Workflow, WorkflowVersion } from '@/lib/schemas';
import { graphToInput, workflowToGraph, YES_NO_KINDS, type Graph, type GraphEdge, type GraphNode } from '@/lib/workflow-graph';
import { CATEGORIES, HUE, ICONS, NODE_TYPES, RULE_LABEL, SUB_LABEL, icon, type NodeType } from './catalog';
import { onAppearanceChange } from '@/lib/appearance';
import { kitify } from './kit';

export type RulesState = { configured: true; rules: BrandDealRules; updatedAt: string } | { configured: false; reason: string };

export type EditorApi = {
  save(id: string, input: WorkflowInput & { note?: string }): Promise<{ workflow: Workflow; version: WorkflowVersion | null }>;
  create(input: WorkflowInput): Promise<{ workflow: Workflow }>;
  remove(id: string): Promise<void>;
  listVersions(id: string): Promise<WorkflowVersion[]>;
  restoreVersion(id: string, versionId: string): Promise<{ workflow: Workflow }>;
  saveRules(patch: Partial<BrandDealRules>, note: string): Promise<{ rules: BrandDealRules; version: ConfigVersion }>;
  listRuleVersions(): Promise<{ versions: ConfigVersion[]; backups: { file: string; at: string }[] }>;
  restoreRuleVersion(versionId: string): Promise<{ rules: BrandDealRules }>;
  runAgent(agentId: string): Promise<AgentRun>;
  draft(prompt: string): Promise<{ ok: boolean; draft?: WorkflowInput; error?: string }>;
};

export type EditorOptions = {
  workflows: Workflow[];
  agents: { id: string; name: string; status: string }[];
  people: string[];
  runs: AgentRun[];
  rules: RulesState;
  setupHref: string;
  initialId?: string | null;
  api: EditorApi;
  /** `g`: the GladOS skin (kit control classes on built DOM; tokens re-pointed in CSS). */
  skin?: 'g';
};

type Sub = { kind: 'model' | 'memory' | 'tool'; id: string };
const SUB_TYPE: Record<string, string> = { model: 'ai-model', memory: 'ai-memory', tool: 'ai-tool' };
const WORLD_OFF = 6000;

export function mountWorkflowEditor(root: HTMLElement, opts: EditorOptions) {
  // React strict mode mounts twice in dev: tear down any previous instance
  // so DOM listeners never double up.
  const prev = (root as any).__glados;
  if (prev && typeof prev.destroy === 'function') prev.destroy();
  tokRoot = root;
  // No workflows: say so before anything looks up the editor markup. The empty
  // message replaces that markup, so a second mount (React strict mode in dev)
  // must not go looking for it either.
  if (!opts.workflows.length) {
    showEmptyLibrary(root);
    return { destroy() {} };
  }
  const ac = new AbortController();
  const on = (el: any, type: string, fn: (ev: any) => void, o?: boolean | AddEventListenerOptions): void => {
    el.addEventListener(type, fn, { ...(typeof o === 'boolean' ? { capture: o } : o || {}), signal: ac.signal });
  };
  const $ = <T extends Element = HTMLElement>(s: string, r: ParentNode = root) => r.querySelector(s) as T | null;
  const $$ = <T extends Element = HTMLElement>(s: string, r: ParentNode = root) => Array.from(r.querySelectorAll(s)) as T[];
  const need = <T extends Element = HTMLElement>(s: string) => {
    const el = $<T>(s);
    if (!el) throw new Error(`workflow canvas markup missing ${s}`);
    return el;
  };
  const clamp = (v: number, a: number, b: number) => Math.max(a, Math.min(b, v));
  const uid = (p: string) => `${p}${Math.random().toString(36).slice(2, 7)}`;
  const delay = (ms: number) => new Promise((r) => setTimeout(r, ms));
  const esc = (s: unknown) => String(s ?? '').replace(/[&<>"]/g, (c) => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;' })[c] as string);

  /* ── state ── */
  const S = {
    wfs: opts.workflows.map(workflowToGraph),
    wf: null as unknown as Graph,
    view: { x: 0, y: 0, k: 1 },
    sel: null as string | null,
    selEdge: null as string | null,
    history: [] as string[],
    future: [] as string[],
    current: '',
    sizes: {} as Record<string, { w: number; h: number }>,
    panel: { mode: 'cats', cat: null as string | null, q: '', focusSearch: false },
    pending: null as null | { from: string; port: string; at: { x: number; y: number } | null },
    running: false,
    paused: false,
    runStart: 0,
    timerId: 0 as unknown as ReturnType<typeof setInterval>,
    chatHidden: true,
    spaceDown: false,
    panelOpen: false,
    panelView: 'picker' as 'picker' | 'inspector',
    mode: 'library' as 'library' | 'editor',
    execFilter: 'all',
    rules: opts.rules,
    rulesDirty: {} as Record<string, true>,
    runs: opts.runs.slice(),
    dirty: false,
    saving: false,
  };

  const D = {
    canvas: need('#canvas'), world: need('#world'), nodes: need('#nodes'), edges: need<SVGSVGElement>('#edges'), edgesTemp: need<SVGSVGElement>('#edgesTemp'),
    labels: need('#edgeLabels'), gridGlow: need('#gridGlow'), gridAmbient: need('#gridAmbient'), panel: need('#panel'), toasts: need('#toasts'),
    modal: need('#modalRoot'), menu: need('#menuRoot'), ghost: need('#dragGhost'), minimap: need<HTMLCanvasElement>('#minimapCanvas'), minimapView: need('#minimapView'),
  };
  const agentsById = new Map(opts.agents.map((a) => [a.id, a]));

  /* ── graph helpers ── */
  const T = (n: GraphNode): NodeType => NODE_TYPES[n.type] ?? NODE_TYPES['ai-agent'];
  const isSubNode = (n: GraphNode) => n.id.includes(':') && !!T(n).ports.top;
  const node = (id: string) => S.wf.nodes.find((n) => n.id === id);
  const mainNodes = () => S.wf.nodes.filter((n) => !isSubNode(n));
  const size = (n: GraphNode) => S.sizes[n.id] || { w: T(n).shape === 'square' && !n.expanded ? 112 : 244, h: T(n).shape === 'square' && !n.expanded ? 112 : 82 };
  const subsOf = (n: GraphNode) => T(n).ports.subs || [];

  /** Agent nodes carry model/memory/tool as config strings; the canvas shows
   *  them as attached sub-nodes. Derived on load, folded back on save. */
  function expandSubs(g: Graph): Graph {
    const nodes = g.nodes.filter((n) => !isSubNode(n));
    const edges = g.edges.filter((e) => e.toPort !== 'top');
    for (const n of nodes.slice()) {
      const subs = subsOf(n);
      if (!subs.length) continue;
      subs.forEach((s, i) => {
        const label = n.config[s];
        if (typeof label !== 'string' || !label.trim()) return;
        const id = `${n.id}:${s}`;
        const t = NODE_TYPES[SUB_TYPE[s]];
        nodes.push({ id, type: SUB_TYPE[s], title: label, sub: t.sub, detail: '', owner: n.owner, ownerKind: 'agent', ownerAgentId: n.ownerAgentId, hours: 0, tools: [], status: 'ok', expanded: false, config: {}, automation: null, leakUsd: null, x: n.x - 80 + i * 160, y: n.y + 212 });
        edges.push({ id: `e-${n.id}-${id}-${s}`, from: n.id, to: id, fromPort: s, toPort: 'top', label: null });
      });
    }
    return { ...g, nodes, edges };
  }
  function collapseSubs(g: Graph): Graph {
    const nodes = g.nodes.filter((n) => !isSubNode(n)).map((n) => ({ ...n, config: { ...n.config } }));
    for (const n of nodes) {
      for (const s of subsOf(n)) {
        const sub = g.nodes.find((x) => x.id === `${n.id}:${s}`);
        const linked = g.edges.some((e) => e.from === n.id && e.fromPort === s);
        if (sub && linked) n.config[s] = sub.title;
        else delete n.config[s];
      }
    }
    return { ...g, nodes, edges: g.edges.filter((e) => e.toPort !== 'top') };
  }

  function anchor(n: GraphNode, port: string) {
    const { w, h } = size(n);
    if (port === 'in') return { x: n.x, y: n.y + h / 2 };
    if (port === 'out') return { x: n.x + w, y: n.y + h / 2 };
    if (port === 'yes') return { x: n.x + w, y: n.y + h * 0.34 };
    if (port === 'no') return { x: n.x + w, y: n.y + h * 0.66 };
    if (port === 'top') return { x: n.x + w / 2, y: n.y };
    const subs = subsOf(n);
    const i = subs.indexOf(port);
    if (i >= 0) return { x: n.x + (w * (i + 1)) / (subs.length + 1), y: n.y + h + 20 };
    return { x: n.x + w / 2, y: n.y + h / 2 };
  }
  const isVertical = (port: string) => !['in', 'out', 'yes', 'no'].includes(port);
  type Pt = { x: number; y: number };
  function pathFor(a: Pt, b: Pt, vertical: boolean) {
    if (vertical) { const dy = Math.max(40, Math.abs(b.y - a.y) * 0.5); return `M${a.x},${a.y} C${a.x},${a.y + dy} ${b.x},${b.y - dy} ${b.x},${b.y}`; }
    const dx = Math.max(60, Math.abs(b.x - a.x) * 0.5);
    return `M${a.x},${a.y} C${a.x + dx},${a.y} ${b.x - dx},${b.y} ${b.x},${b.y}`;
  }
  function bezierMid(a: Pt, b: Pt, vertical: boolean) {
    let c1: Pt, c2: Pt;
    if (vertical) { const dy = Math.max(40, Math.abs(b.y - a.y) * 0.5); c1 = { x: a.x, y: a.y + dy }; c2 = { x: b.x, y: b.y - dy }; }
    else { const dx = Math.max(60, Math.abs(b.x - a.x) * 0.5); c1 = { x: a.x + dx, y: a.y }; c2 = { x: b.x - dx, y: b.y }; }
    const t = 0.5, mt = 0.5;
    return { x: mt ** 3 * a.x + 3 * mt ** 2 * t * c1.x + 3 * mt * t ** 2 * c2.x + t ** 3 * b.x, y: mt ** 3 * a.y + 3 * mt ** 2 * t * c1.y + 3 * mt * t ** 2 * c2.y + t ** 3 * b.y };
  }

  /* ── view maths ── */
  const canvasRect = () => D.canvas.getBoundingClientRect();
  const panelW = () => (S.panelOpen ? 352 + 40 : 100);
  function toWorld(cx: number, cy: number) { const r = canvasRect(); return { x: (cx - r.left - S.view.x) / S.view.k, y: (cy - r.top - S.view.y) / S.view.k }; }
  function applyView() { D.world.style.transform = `translate(${S.view.x}px, ${S.view.y}px) scale(${S.view.k})`; drawMinimap(); }
  function zoomAt(cx: number, cy: number, factor: number) {
    const r = canvasRect(); const k = clamp(S.view.k * factor, 0.3, 2.2); const px = cx - r.left, py = cy - r.top;
    S.view.x = px - ((px - S.view.x) * k) / S.view.k; S.view.y = py - ((py - S.view.y) * k) / S.view.k; S.view.k = k; applyView();
  }
  function bbox() {
    if (!S.wf.nodes.length) return { x: 0, y: 0, w: 400, h: 300 };
    let x1 = Infinity, y1 = Infinity, x2 = -Infinity, y2 = -Infinity;
    for (const n of S.wf.nodes) { const { w, h } = size(n); x1 = Math.min(x1, n.x); y1 = Math.min(y1, n.y); x2 = Math.max(x2, n.x + w); y2 = Math.max(y2, n.y + h + (T(n).shape === 'square' ? 44 : 0) + (subsOf(n).length ? 44 : 0)); }
    return { x: x1, y: y1, w: x2 - x1, h: y2 - y1 };
  }
  /** The first fit can run before the frame has a size (CSS still landing,
   *  strict-mode remount); wait for a real layout instead of fitting to zero. */
  function fitWhenLaidOut(tries = 0) {
    const r = canvasRect();
    if (r.height < 200 && tries < 60) { requestAnimationFrame(() => fitWhenLaidOut(tries + 1)); return; }
    fitView(false);
  }
  function fitView(animate: boolean) {
    const r = canvasRect(); const b = bbox(); const pad = 70;
    const availW = r.width - panelW() - pad * 2, availH = r.height - 200 - pad * 2;
    const k = clamp(Math.min(availW / b.w, availH / b.h, 1.05), 0.78, 2.2);
    S.view.k = k; S.view.x = b.w * k > availW ? pad + 120 - b.x * k : pad + (availW - b.w * k) / 2 - b.x * k + 20; S.view.y = pad + (availH - b.h * k) / 2 - b.y * k;
    if (animate) { D.world.style.transition = 'transform .45s var(--ease)'; setTimeout(() => (D.world.style.transition = ''), 460); }
    applyView();
  }

  /* ── glow spotlight ── */
  function glowAt(wx: number, wy: number) { for (const el of [D.gridGlow, D.gridAmbient]) { el.style.setProperty('--gx', `${wx + WORLD_OFF}px`); el.style.setProperty('--gy', `${wy + WORLD_OFF}px`); el.classList.add('is-on'); } }
  function glowNode(n: GraphNode) { const { w, h } = size(n); glowAt(n.x + w / 2, n.y + h / 2); }
  function glowOff() { D.gridGlow.classList.remove('is-on'); D.gridAmbient.classList.remove('is-on'); }

  /* ── panel open/close ── */
  function openPanel(view?: 'picker' | 'inspector') { if (view) S.panelView = view; S.panelOpen = true; renderPanel(); D.panel.classList.add('is-open'); syncCapsules(); }
  function closePanel() { if (!S.panelOpen) return; S.panelOpen = false; D.panel.classList.remove('is-open'); syncCapsules(); }
  function syncCapsules() {
    const hasSel = !!(S.sel || S.selEdge);
    need('#btnPanelToggle').classList.toggle('is-open', S.panelOpen && S.panelView === 'picker');
    const st = need('#btnSettings'); st.classList.toggle('is-visible', hasSel); st.classList.toggle('is-open', S.panelOpen && S.panelView === 'inspector' && hasSel);
  }

  /* ── history (undo lands on the state BEFORE the mutation) ── */
  const snap = () => JSON.stringify({ nodes: S.wf.nodes, edges: S.wf.edges, name: S.wf.name, subtitle: S.wf.subtitle, ratioTarget: S.wf.ratioTarget });
  function commit() { S.history.push(S.current); if (S.history.length > 80) S.history.shift(); S.future = []; S.current = snap(); S.dirty = true; markDirty(); }
  function restore(json: string) { Object.assign(S.wf, JSON.parse(json)); S.current = json; S.sel = S.wf.nodes.some((n) => n.id === S.sel) ? S.sel : null; S.selEdge = null; S.dirty = true; markDirty(); renderAll(); }
  function undo() { if (!S.history.length) return toast('Nothing to undo', 'warn'); S.future.push(S.current); restore(S.history.pop()!); toast('Undone'); }
  function redo() { if (!S.future.length) return toast('Nothing to redo', 'warn'); S.history.push(S.current); restore(S.future.pop()!); toast('Redone'); }
  function markDirty() { const b = need('#btnSave'); b.textContent = S.saving ? 'Saving…' : S.dirty || Object.keys(S.rulesDirty).length ? 'Save changes' : 'Saved'; b.classList.toggle('is-dirty', S.dirty || Object.keys(S.rulesDirty).length > 0); }

  /* ── rules bindings ── */
  function boundKey(n: GraphNode, fieldKey: string): string | null { const b = n.config && (n.config as any).bindings; return b && b[fieldKey] ? String(b[fieldKey]) : null; }
  function liveRules(): BrandDealRules | null { return S.rules.configured ? S.rules.rules : null; }
  function fieldValue(n: GraphNode, key: string) {
    const rk = boundKey(n, key); const r = liveRules();
    if (rk && r && rk in r) { const v = (r as any)[rk]; return Array.isArray(v) ? v.join(', ') : v; }
    return (n.config as any)[key];
  }
  /** Returns true when the write went to the live rules (not the node). */
  function setFieldValue(n: GraphNode, key: string, value: unknown): boolean {
    const rk = boundKey(n, key); const r = liveRules() as any;
    if (rk && r) {
      const wasList = Array.isArray(r[rk]);
      r[rk] = wasList ? String(value).split(',').map((x) => x.trim()).filter(Boolean) : typeof r[rk] === 'number' ? Number(value) || 0 : value;
      S.rulesDirty[rk] = true; markDirty(); return true;
    }
    (n.config as any)[key] = value; return false;
  }

  /* ── rendering ── */
  function boundTag(n: GraphNode, key: string) {
    const rk = boundKey(n, key); if (!rk) return '';
    if (S.rules.configured) return `<span class="bound live" title="Reads and writes ~/.config/bdpilot/rules.json on the mini; the daemon reloads it within two minutes">live · rules.json</span>`;
    return `<span class="bound" title="${esc((S.rules as any).reason)}">not configured</span> <a class="setup" href="${esc(opts.setupHref)}">Set up ›</a>`;
  }
  function fieldHtml(n: GraphNode, f: NodeType['fields'][number], inline: boolean) {
    const v = fieldValue(n, f.key); const rk = boundKey(n, f.key);
    const cls = inline ? 'field' : 'fld';
    const label = `<label>${esc(f.label)}${boundTag(n, f.key)}</label>`;
    const dis = rk && !S.rules.configured ? 'disabled' : '';
    if (f.type === 'select') return `<div class="${cls}">${label}<div class="ctl"><select data-cfg="${f.key}" ${dis}>${(f.options || []).map((o) => `<option ${o === v ? 'selected' : ''}>${esc(o)}</option>`).join('')}</select>${icon('chevDown')}</div></div>`;
    if (f.type === 'toggle') return `<div class="${cls}">${label}<div class="sw ${v ? 'is-on' : ''}" data-cfg-toggle="${f.key}"><span>${v ? 'On' : 'Off'}</span><span class="knob"></span></div></div>`;
    if (f.type === 'number') return `<div class="${cls}">${label}<div class="ctl"><input type="${inline ? 'text' : 'number'}" data-cfg="${f.key}" value="${esc(v ?? '')}" ${dis}><span class="unit">${f.unit || ''}</span></div></div>`;
    if (!inline && (f.key === 'prompt' || f.key === 'criteria' || String(v ?? '').length > 90)) return `<div class="${cls}">${label}<div class="ctl"><textarea data-cfg="${f.key}" ${dis} ${rk === 'draft_rules' ? 'style="min-height:220px"' : ''}>${esc(v ?? '')}</textarea></div></div>`;
    return `<div class="${cls}">${label}<div class="ctl"><input type="text" data-cfg="${f.key}" value="${esc(v ?? '')}" ${dis}></div></div>`;
  }

  function nodeHtml(n: GraphNode) {
    const t = T(n); const sq = t.shape === 'square';
    const linked = new Set(S.wf.edges.flatMap((e) => (e.from === n.id ? [e.fromPort] : []).concat(e.to === n.id ? [e.toPort] : [])));
    const statusIcon = n.status === 'off' ? 'trash' : n.status === 'suggested' ? 'alert' : 'check';
    const cfg = n.config as any;
    const portTags: Record<string, string> = { yes: cfg.yes || 'yes', no: cfg.no || 'no' };
    let ports = '';
    if (t.ports.in) ports += `<div class="port in ${linked.has('in') ? 'is-linked' : ''}" data-port="in"></div>`;
    if (t.ports.top) ports += `<div class="port top ${linked.has('top') ? 'is-linked' : ''}" data-port="top"></div>`;
    if (t.ports.out) ports += `<div class="port out ${linked.has('out') ? 'is-linked' : ''}" data-port="out"></div>`;
    if (t.ports.outs) ports += t.ports.outs.map((p) => `<div class="port ${p} ${linked.has(p) ? 'is-linked' : ''}" data-port="${p}"><span class="ptag">${esc(portTags[p])}</span></div>`).join('');
    const subs = subsOf(n);
    const subsHtml = subs.length ? `<div class="subs">${subs.map((s, i) => `<div class="sub ${linked.has(s) ? 'is-linked' : ''}" style="left:${((i + 1) / (subs.length + 1)) * 100}%"><div class="stem"></div><div class="sport ${linked.has(s) ? 'is-linked' : ''}" data-port="${s}"></div><div class="sname">${SUB_LABEL[s]}</div><button class="sadd" data-sub="${s}" title="Add ${SUB_LABEL[s]}">${icon('plus')}</button></div>`).join('')}</div>` : '';
    const inlineFields = t.fields.slice(0, t.inline ? 3 : 2);
    const body = t.fields.length ? `<div class="body">${inlineFields.map((f) => fieldHtml(n, f, true)).join('')}</div>` : '';
    const caret = t.fields.length ? `<button class="caret" data-act="expand" title="Configure inline">${icon('chevDown')}</button>` : '';
    const presence = n.ownerAgentId ? agentsById.get(n.ownerAgentId) : null;
    const presenceDot = n.ownerKind === 'agent' ? `<span class="pres ${presence ? presence.status : 'unbound'}" title="${presence ? `${esc(presence.name)} · ${presence.status}` : 'not bound to a roster agent'}"></span>` : '';
    return `<div class="node ${sq ? 'square' : 'card'} ${S.sel === n.id ? 'is-selected' : ''} ${n.expanded ? 'is-expanded' : ''}" data-id="${n.id}" style="transform:translate(${n.x}px,${n.y}px)">
      <div class="frame">
        <div class="head"><div class="tile" data-hue="${HUE[n.type] || ''}">${icon(t.icon)}</div><div class="ttl"><div class="t">${esc(n.title)}</div><div class="s">${presenceDot}${esc(n.sub)}</div></div></div>
        <div class="status" data-s="${n.status}">${icon(statusIcon)}</div>${caret}${body}${ports}${subsHtml}
      </div>
      ${sq ? `<div class="label">${esc(n.title)}${n.status === 'off' ? '<div class="s">(Deactivated)</div>' : n.sub && n.sub !== t.sub ? `<div class="s">${esc(n.sub)}</div>` : ''}</div>` : ''}
    </div>`;
  }
  function renderNodes() {
    D.nodes.innerHTML = S.wf.nodes.map(nodeHtml).join('');
    for (const el of $$('.node', D.nodes)) { const f = $('.frame', el)!; S.sizes[el.dataset.id!] = { w: f.offsetWidth, h: f.offsetHeight }; }
  }
  function edgeClass(e: GraphEdge & { run?: string | null }) {
    const a = node(e.from), b = node(e.to); if (!a || !b) return '';
    const cls: string[] = [];
    if (isVertical(e.fromPort)) cls.push('subedge');
    if (a.status === 'off' || b.status === 'off') cls.push('off');
    else if (a.status === 'suggested' || b.status === 'suggested') cls.push('suggested');
    if (S.selEdge === e.id) cls.push('is-selected');
    if (e.run) cls.push(e.run === 'active' ? 'is-active' : 'is-done');
    return cls.join(' ');
  }
  function renderEdges() {
    const defs = `<defs>
      <radialGradient id="flare"><stop offset="0" stop-color="${tok('--wf-flare-core', '#9dffdc')}" stop-opacity="1"/><stop offset="0.35" stop-color="${tok('--accent', '#00ffab')}" stop-opacity="0.55"/><stop offset="1" stop-color="${tok('--accent', '#00ffab')}" stop-opacity="0"/></radialGradient>
      <radialGradient id="flareWhite"><stop offset="0" stop-color="${tok('--text', '#ffffff')}" stop-opacity="0.9"/><stop offset="0.4" stop-color="${tok('--text', '#e9f0ed')}" stop-opacity="0.35"/><stop offset="1" stop-color="${tok('--text', '#e9f0ed')}" stop-opacity="0"/></radialGradient>
      <radialGradient id="flareAmber"><stop offset="0" stop-color="${tok('--wf-hue-amber', '#ffe2a3')}" stop-opacity="0.9"/><stop offset="0.4" stop-color="${tok('--warn', '#f2b33d')}" stop-opacity="0.4"/><stop offset="1" stop-color="${tok('--warn', '#f2b33d')}" stop-opacity="0"/></radialGradient>
    </defs>`;
    let out = defs; let labels = '';
    for (const e of S.wf.edges as (GraphEdge & { run?: string | null })[]) {
      const a = node(e.from), b = node(e.to); if (!a || !b) continue;
      const v = isVertical(e.fromPort); const p1 = anchor(a, e.fromPort), p2 = anchor(b, e.toPort);
      const o = { x: p1.x + WORLD_OFF, y: p1.y + WORLD_OFF }, q = { x: p2.x + WORLD_OFF, y: p2.y + WORLD_OFF };
      const d = pathFor(o, q, v); const cls = edgeClass(e);
      const flare = cls.includes('off') ? '' : cls.includes('subedge') ? 'sub' : cls.includes('suggested') ? 'amber' : '';
      out += `<g class="${cls}" data-edge="${e.id}"><path class="glow" d="${d}"/><path class="line" d="${d}"/><path class="flow" d="${d}"/><path class="hit" d="${d}"/>${cls.includes('off') ? '' : `<circle class="flare ${flare}" cx="${q.x}" cy="${q.y}" r="${v ? 12 : 16}"/><circle class="flare ${flare}" cx="${o.x}" cy="${o.y}" r="${v ? 10 : 14}"/>`}</g>`;
      if (e.label) { const m = bezierMid(p1, p2, v); labels += `<div class="edge-label ${e.run === 'active' ? 'is-active' : ''} ${['yes', 'no'].includes(e.fromPort) ? 'branch' : ''}" data-edge="${e.id}" style="left:${m.x}px;top:${m.y}px" contenteditable="true" spellcheck="false">${esc(e.label)}</div>`; }
    }
    D.edges.innerHTML = out; D.labels.innerHTML = labels;
  }
  function renderAll() { renderNodes(); renderEdges(); renderPanel(); renderRatio(); drawMinimap(); renderTop(); syncCapsules(); }

  /* ── minimap ── */
  function drawMinimap() {
    const c = D.minimap, ctx = c.getContext('2d'); if (!ctx) return; ctx.clearRect(0, 0, c.width, c.height);
    const b = bbox(); const pad = 400; const bx = b.x - pad, by = b.y - pad, bw = b.w + pad * 2, bh = b.h + pad * 2;
    const k = Math.min(c.width / bw, c.height / bh); const ox = (c.width - bw * k) / 2, oy = (c.height - bh * k) / 2;
    for (const e of S.wf.edges) { const a = node(e.from), bn = node(e.to); if (!a || !bn) continue; const p = anchor(a, e.fromPort), q = anchor(bn, e.toPort); ctx.strokeStyle = accentA(0.35); ctx.lineWidth = 1; ctx.beginPath(); ctx.moveTo(ox + (p.x - bx) * k, oy + (p.y - by) * k); ctx.lineTo(ox + (q.x - bx) * k, oy + (q.y - by) * k); ctx.stroke(); }
    for (const n of S.wf.nodes) { const { w, h } = size(n); ctx.fillStyle = n.id === S.sel ? tok('--accent', '#00ffab') : n.ownerKind === 'human' ? tok('--wf-minimap-human', 'rgba(189,95,232,0.8)') : inkA(0.55); ctx.fillRect(ox + (n.x - bx) * k, oy + (n.y - by) * k, Math.max(3, w * k), Math.max(3, h * k)); }
    const r = canvasRect(); const vx = -S.view.x / S.view.k, vy = -S.view.y / S.view.k, vw = r.width / S.view.k, vh = r.height / S.view.k;
    Object.assign(D.minimapView.style, { left: `${ox + (vx - bx) * k}px`, top: `${oy + (vy - by) * k}px`, width: `${vw * k}px`, height: `${vh * k}px` });
    (D.minimap as any)._map = { bx, by, k, ox, oy };
  }

  /* ── ratio card ── */
  function ratioBasics() {
    const main = mainNodes().filter((n) => n.status !== 'off'); const agent = main.filter((n) => n.ownerKind === 'agent');
    const ratio = main.length ? agent.length / main.length : 0;
    const humanHours = main.filter((n) => n.ownerKind === 'human').reduce((s, n) => s + (Number(n.hours) || 0), 0);
    const totalHours = main.reduce((s, n) => s + (Number(n.hours) || 0), 0);
    const baseline = ratio < 1 ? humanHours / (1 - ratio) : totalHours;
    return { ratio, baseline };
  }
  const ratioStatAt = (t: number) => Math.round(ratioBasics().baseline * (1 - t) * 10) / 10;
  function fitRatioHead() { const head = need('.ratio-head'); head.classList.remove('is-tight'); if (head.scrollWidth > head.clientWidth + 1) head.classList.add('is-tight'); }
  function renderRatio() {
    const { ratio } = ratioBasics(); const shown = S.wf.ratioTarget != null ? S.wf.ratioTarget : ratio;
    need('#ratioStat').textContent = `Human hours: ${ratioStatAt(shown)}h/wk`; need('#ratioMiniPct').textContent = `${Math.round(shown * 100)}%`; fitRatioHead();
    const track = need('#ratioTrack'); const W = track.clientWidth - 24;
    need('#ratioLine').style.width = `${W * shown}px`; need('#ratioMarker').style.left = `${12 + W * shown}px`; drawRatioGrid(shown);
  }
  function drawRatioGrid(ratio: number) {
    const c = need<HTMLCanvasElement>('#ratioGrid'); const track = need('#ratioTrack'); const dpr = window.devicePixelRatio || 1;
    const w = track.clientWidth, h = track.clientHeight; if (!w) return;
    if (c.width !== Math.round(w * dpr)) { c.width = Math.round(w * dpr); c.height = Math.round(h * dpr); }
    const ctx = c.getContext('2d'); if (!ctx) return; ctx.setTransform(dpr, 0, 0, dpr, 0, 0); ctx.clearRect(0, 0, w, h);
    const mx = 12 + (w - 24) * ratio, my = h / 2; const step = 9;
    for (let y = 6; y < h - 3; y += step) for (let x = 7; x < w - 6; x += step) {
      const dx = (x - mx) / 74, dy = (y - my) / 30; const g = Math.exp(-(dx * dx) * 0.9) * (0.55 + 0.45 * Math.exp(-(dy * dy)));
      const r = 0.6 + 1.5 * g; const a = 0.07 + 0.3 * g;
      ctx.beginPath(); ctx.arc(x, y, r, 0, Math.PI * 2); ctx.fillStyle = inkA(+a.toFixed(3)); ctx.fill();
    }
  }
  {
    const m = need('#ratioMarker'), track = need('#ratioTrack'); let dragging = false;
    const setFrom = (clientX: number) => { const r = track.getBoundingClientRect(); const W = r.width - 24; const t = clamp((clientX - r.left - 12) / W, 0, 1); S.wf.ratioTarget = Math.round(t * 1000) / 1000; need('#ratioLine').style.width = `${W * t}px`; m.style.left = `${12 + W * t}px`; need('#ratioStat').textContent = `Human hours: ${ratioStatAt(t)}h/wk`; drawRatioGrid(t); };
    on(m, 'pointerdown', (ev) => { dragging = true; m.classList.add('is-dragging'); track.classList.add('is-dragging'); try { m.setPointerCapture(ev.pointerId); } catch {} setFrom(ev.clientX); ev.preventDefault(); });
    on(m, 'pointermove', (ev) => { if (dragging) setFrom(ev.clientX); });
    on(m, 'pointerup', () => { if (!dragging) return; dragging = false; m.classList.remove('is-dragging'); track.classList.remove('is-dragging'); commit(); renderRatio(); toast(`Target ${Math.round((S.wf.ratioTarget ?? 0) * 100)}% agent-owned · save to keep`); });
    on(m, 'dblclick', () => { S.wf.ratioTarget = null; commit(); renderRatio(); toast('Target cleared'); });
  }

  /* ── top bar ── */
  function renderTop() {
    if (S.mode === 'library') {
      need('#wfTitle').textContent = 'Workflows';
      need('#wfCrumb').innerHTML = `${S.wfs.length} mapped · click one to open it`;
      need('#wfSubtitle').textContent = '';
      return;
    }
    need('#wfTitle').textContent = S.wf.name;
    need('#wfCrumb').innerHTML = `${icon('chevLeft')}all workflows`;
    need('#wfSubtitle').textContent = S.wf.subtitle || 'No description';
    need('#chatId').textContent = `/ ${S.wf.id}`;
    markDirty();
  }

  /* ── library: a glass card per workflow with a mini-render of its tree ── */
  function thumbSvg(g: Graph, W: number, H: number): string {
    const main = g.nodes.filter((n) => !isSubNode(n));
    if (!main.length) return '';
    const dims = (n: GraphNode) => (T(n).shape === 'square' ? { w: 112, h: 112 } : { w: 244, h: 82 });
    let x1 = Infinity, y1 = Infinity, x2 = -Infinity, y2 = -Infinity;
    for (const n of main) { const d = dims(n); x1 = Math.min(x1, n.x); y1 = Math.min(y1, n.y); x2 = Math.max(x2, n.x + d.w); y2 = Math.max(y2, n.y + d.h); }
    const pad = 14; const k = Math.min((W - pad * 2) / (x2 - x1), (H - pad * 2) / (y2 - y1), 0.42);
    const ox = pad + ((W - pad * 2) - (x2 - x1) * k) / 2 - x1 * k, oy = pad + ((H - pad * 2) - (y2 - y1) * k) / 2 - y1 * k;
    const P = (x: number, y: number) => ({ x: ox + x * k, y: oy + y * k });
    const byId = new Map(main.map((n) => [n.id, n]));
    let out = '';
    for (const e of g.edges.filter((e) => e.toPort === 'in')) {
      const a = byId.get(e.from), b = byId.get(e.to); if (!a || !b) continue; const da = dims(a), db = dims(b);
      const ay = e.fromPort === 'yes' ? 0.34 : e.fromPort === 'no' ? 0.66 : 0.5;
      const p1 = P(a.x + da.w, a.y + da.h * ay), p2 = P(b.x, b.y + db.h / 2); const dx = Math.max(12, Math.abs(p2.x - p1.x) * 0.5);
      out += `<path d="M${p1.x},${p1.y} C${p1.x + dx},${p1.y} ${p2.x - dx},${p2.y} ${p2.x},${p2.y}" fill="none" stroke="${accentA(0.55)}" stroke-width="1.2"/><circle cx="${p2.x}" cy="${p2.y}" r="2" fill="${tok('--accent', '#00ffab')}"/>`;
    }
    for (const n of main) {
      const d = dims(n); const p = P(n.x, n.y); const w = d.w * k, h = d.h * k; const t = T(n);
      const fill = n.ownerKind === 'human' ? tok('--wf-thumb-human-bg', 'rgba(52,36,70,0.95)') : tok('--wf-thumb-bg', 'rgba(23,38,34,0.97)'); const stroke = n.ownerKind === 'human' ? tok('--wf-thumb-human-line', 'rgba(189,95,232,0.5)') : tok('--wf-thumb-line', 'rgba(43,61,56,1)');
      const ic = Math.min(14, h * 0.5); const iconSvg = `<svg x="${p.x + (t.shape === 'square' ? (w - ic) / 2 : 8)}" y="${p.y + (h - ic) / 2}" width="${ic}" height="${ic}" viewBox="0 0 24 24" fill="none" stroke="${n.ownerKind === 'human' ? tok('--wf-hue-violet', '#cfa4ff') : tok('--text', '#e9f0ed')}" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round">${ICONS[t.icon] || ICONS.circle}</svg>`;
      const maxChars = Math.floor(w / 6);
      const label = t.shape !== 'square' && w > 70 ? `<text x="${p.x + 8 + ic + 6}" y="${p.y + h / 2 + 3.5}" font-size="${Math.max(8, Math.min(10, w / 9))}" fill="${tok('--text', '#e9f0ed')}">${esc(n.title.length > maxChars ? n.title.slice(0, Math.max(1, maxChars - 1)) + '…' : n.title)}</text>` : '';
      out += `<rect x="${p.x}" y="${p.y}" width="${w}" height="${h}" rx="${Math.min(8, h * 0.18)}" fill="${fill}" stroke="${stroke}"/>${iconSvg}${label}`;
    }
    return `<svg viewBox="0 0 ${W} ${H}" preserveAspectRatio="xMidYMid meet" style="font-family: var(--sans)">${out}</svg>`;
  }
  function renderLibrary() {
    const grid = need('#libGrid');
    grid.innerHTML = S.wfs.map((w) => {
      const main = w.nodes.filter((n) => !isSubNode(n)); const agent = main.filter((n) => n.ownerKind === 'agent').length;
      const hours = main.filter((n) => n.ownerKind === 'human').reduce((a, n) => a + (Number(n.hours) || 0), 0);
      return `<button class="lib-card" data-open-lib="${esc(w.id)}"><div class="lib-thumb">${thumbSvg(w, 380, 190)}</div><div class="lib-body"><div class="lib-name">${esc(w.name)}</div><div class="lib-sub">${esc(w.subtitle || '')}</div><div class="lib-meta"><span><b>${main.length}</b> steps</span><span><b>${agent}</b> agent-owned</span><span><b>${hours}h</b>/wk human</span>${main.some((n) => n.automation?.state === 'suggested') ? '<span>suggested automations</span>' : ''}</div></div></button>`;
    }).join('') + `<button class="lib-card lib-new" data-new-lib>${icon('plus')}New workflow</button>`;
  }
  function showLibrary() {
    S.mode = 'library'; closePanel(); root.classList.add('is-library');
    $$('.view').forEach((v) => v.classList.toggle('is-active', v.id === 'viewLibrary'));
    renderLibrary(); renderTop();
    try { history.replaceState(null, '', '/workflows'); } catch {}
  }
  function showEditor() {
    S.mode = 'editor'; root.classList.remove('is-library');
    $$('.tab').forEach((b) => b.classList.toggle('is-active', b.dataset.tab === 'editor'));
    $$('.view').forEach((v) => v.classList.toggle('is-active', v.id === 'viewEditor'));
  }
  on(root, 'click', (ev: MouseEvent) => {
    const t = ev.target as HTMLElement;
    const card = t.closest('[data-open-lib]') as HTMLElement | null; if (card) { loadWorkflow(card.dataset.openLib!); return; }
    if (t.closest('[data-new-lib]')) need('#btnNewWorkflow').click();
  });

  /* ── right panel ── */
  function itemHtml(typeId: string) {
    const t = NODE_TYPES[typeId];
    return `<div class="item" data-type="${typeId}" draggable="false"><div class="tile" data-hue="${HUE[typeId] || ''}">${icon(t.icon)}</div><div class="it"><div class="l">${esc(t.label)}</div><div class="s">${esc(t.sub)}</div></div><button class="add" data-add="${typeId}" title="Add">${icon('plus')}</button></div>`;
  }
  const catsHtml = () => CATEGORIES.map((c) => `<div class="cat" data-cat="${c.id}"><div class="cic">${icon(c.icon)}</div><div class="ct"><div class="l">${esc(c.label)}</div><div class="s">${esc(c.sub)}</div></div><div class="go">${icon('arrowRight')}</div></div>`).join('');
  const searchHtml = (q: string) => { const hits = Object.keys(NODE_TYPES).filter((k) => { const t = NODE_TYPES[k]; return (t.label + ' ' + t.sub + ' ' + t.cat).toLowerCase().includes(q); }); return hits.length ? hits.map(itemHtml).join('') : `<div class="empty">No node matches "${esc(q)}".</div>`; };
  function renderPanel() {
    const P = D.panel; const insp = S.panelView === 'inspector'; const sel = insp && S.sel ? node(S.sel) : null;
    const top = `<div class="panel-top"><button class="sq ${!insp ? 'is-on' : ''}" data-panel="picker" title="Add node">${icon('plus')}</button><button class="sq" data-panel="search" title="Search nodes">${icon('search')}</button><button class="sq" data-panel="library" title="All workflows">${icon('grid')}</button></div>`;
    if (insp && S.selEdge) {
      const e = S.wf.edges.find((x) => x.id === S.selEdge)!; const a = node(e.from)!, b = node(e.to)!;
      P.innerHTML = `${top}<h3>Connection</h3><div class="sec"><div class="sec-t">Route</div><div class="kv"><span>From</span><span>${esc(a.title)} · ${esc(e.fromPort)}</span></div><div class="kv"><span>To</span><span>${esc(b.title)}</span></div></div><div class="sec"><div class="fld"><label>Label on the connector</label><div class="ctl"><input type="text" data-edge-label value="${esc(e.label || '')}" placeholder="e.g. approved"></div></div></div><div class="insp-actions"><button class="btn danger" data-act="delete-edge">${icon('trash')}Remove connection</button></div>`;
      return;
    }
    if (sel) {
      const t = T(sel); const fields = t.fields.map((f) => fieldHtml(sel, f, false)).join(''); const auto = sel.automation;
      const agentOpts = opts.agents.map((a) => `<option value="${a.id}" ${a.id === sel.ownerAgentId ? 'selected' : ''}>${esc(a.name)} · ${a.status}</option>`).join('');
      const owner = sel.ownerKind === 'human'
        ? `<select data-owner>${[...new Set([sel.owner, ...opts.people])].map((o) => `<option ${o === sel.owner ? 'selected' : ''}>${esc(o)}</option>`).join('')}</select>${icon('chevDown')}`
        : `<select data-owner-agent><option value="">— not bound —</option>${agentOpts}</select>${icon('chevDown')}`;
      const bound = sel.ownerAgentId ? agentsById.get(sel.ownerAgentId) : null;
      P.innerHTML = `${top}
        <div class="insp-head"><div class="tile" data-hue="${HUE[sel.type] || ''}">${icon(t.icon)}</div><div class="ih"><input data-title value="${esc(sel.title)}"><div class="k">${esc(t.label)} · <span class="mono">${esc(sel.id)}</span></div></div></div>
        <div class="panel-scroll">
          <div class="sec"><div class="sec-t">What it does</div><div class="fld"><div class="ctl"><textarea data-detail placeholder="Describe this step">${esc(sel.detail)}</textarea></div></div></div>
          <div class="sec"><div class="sec-t">Configuration</div>${fields || '<div class="empty">No settings for this node.</div>'}</div>
          <div class="sec"><div class="sec-t">Owner</div>
            <div class="fld"><div class="seg"><button data-kind="agent" class="${sel.ownerKind === 'agent' ? 'is-on' : ''}">Agent</button><button data-kind="human" class="${sel.ownerKind === 'human' ? 'is-on' : ''}">Human</button></div></div>
            <div class="fld"><label>${sel.ownerKind === 'human' ? 'Person' : 'Roster agent'}</label><div class="owner-row"><div class="avatar ${sel.ownerKind}">${esc(sel.owner.slice(0, 2).toUpperCase())}</div><div class="ctl" style="flex:1">${owner}</div></div>${sel.ownerKind === 'agent' && !bound ? `<div class="hint" style="margin:8px 0 0">Not bound to a roster agent, so Test workflow cannot run it.</div>` : ''}</div>
            <div class="fld"><label>Hours per week</label><div class="ctl"><input type="number" data-hours value="${esc(sel.hours)}" min="0" step="0.5"><span class="unit">h/wk</span></div></div>
          </div>
          <div class="sec"><div class="sec-t">Automation</div>
            ${auto ? `<div class="kv"><span>State</span><span class="pill ${auto.state}">${auto.state}</span></div><div class="fld" style="margin-top:10px"><label>What it does</label><div class="ctl"><input type="text" data-auto-title value="${esc(auto.title)}"></div></div>${auto.state === 'suggested' ? `<button class="btn primary block" data-act="promote">${icon('zap')}Mark as live</button>` : ''}` : `<div class="sw" data-act="add-auto"><span>Suggest an automation here</span><span class="knob"></span></div>`}
          </div>
          <div class="sec"><div class="sec-t">Runs</div>${runsFor(sel).slice(0, 3).map((r) => `<div class="kv"><span class="mono">${esc(r.startedAt.slice(0, 16).replace('T', ' '))}</span><span class="pill ${r.ok ? 'success' : 'failed'}">${r.ok ? 'ok' : 'failed'}</span></div>`).join('') || `<div class="empty" style="padding:8px 0">No recorded runs${sel.ownerAgentId ? '' : ' (not bound to an agent)'}.</div>`}</div>
        </div>
        <div class="insp-actions"><button class="btn" data-act="dup">${icon('copy')}Duplicate</button><button class="btn" data-act="toggle-off">${icon(sel.status === 'off' ? 'play' : 'pause')}${sel.status === 'off' ? 'Activate' : 'Deactivate'}</button><button class="btn danger" data-act="delete" style="margin-left:auto">${icon('trash')}</button></div>`;
      return;
    }
    const q = S.panel.q.trim().toLowerCase();
    let body = '';
    if (q) body = searchHtml(q);
    else if (S.panel.mode === 'items' && S.panel.cat) { const c = CATEGORIES.find((x) => x.id === S.panel.cat)!; body = `<div class="back" data-back>${icon('chevLeft')}${esc(c.label)}</div>` + Object.keys(NODE_TYPES).filter((k) => NODE_TYPES[k].cat === c.id).map(itemHtml).join(''); }
    else body = catsHtml();
    const hint = S.pending ? `<div class="hint">Connecting from <b>${esc(node(S.pending.from)?.title)}</b> · ${SUB_LABEL[S.pending.port] || S.pending.port}. Pick what happens next.</div>` : `<div class="hint">Click to add after the selected node, or drag onto the canvas.</div>`;
    P.innerHTML = `${top}<h3>What happens next?</h3><div class="search">${icon('search')}<input id="panelSearch" placeholder="Search nodes…" value="${esc(S.panel.q)}" autocomplete="off"></div>${hint}<div class="panel-scroll">${body}</div>`;
    if (S.panel.focusSearch) { $<HTMLInputElement>('#panelSearch')?.focus(); S.panel.focusSearch = false; }
  }
  function runsFor(n: GraphNode): AgentRun[] { return n.ownerAgentId ? S.runs.filter((r) => r.agentId === n.ownerAgentId).sort((a, b) => (a.startedAt < b.startedAt ? 1 : -1)) : []; }

  /* ── mutations ── */
  function newNode(typeId: string, at: { x: number; y: number }): GraphNode {
    const t = NODE_TYPES[typeId]; const config: Record<string, unknown> = {};
    for (const f of t.fields) config[f.key] = f.default !== undefined ? f.default : f.options ? f.options[0] : '';
    const id = `${S.wf.id}-${uid('n')}`;
    return { id, type: typeId, title: t.label, sub: t.sub, detail: '', owner: t.ownerKind === 'human' ? opts.people[0] || 'Operator' : t.label, ownerKind: t.ownerKind, ownerAgentId: null, hours: t.hours, tools: [], status: 'ok', expanded: false, config, automation: null, leakUsd: null, x: at.x, y: at.y };
  }
  function addNode(typeId: string, at: { x: number; y: number }, connectFrom: { from: string; port: string } | null) {
    const t = NODE_TYPES[typeId]; const n = newNode(typeId, at);
    if (connectFrom) {
      const from = node(connectFrom.from);
      if (from) {
        const port = connectFrom.port; const toPort = isVertical(port) ? 'top' : 'in';
        if (toPort === 'top' && t.ports.top) { n.id = `${from.id}:${port}`; S.wf.edges = S.wf.edges.filter((e) => !(e.from === from.id && e.fromPort === port)); S.wf.nodes = S.wf.nodes.filter((x) => x.id !== n.id); }
        if ((toPort === 'top' && t.ports.top) || (toPort === 'in' && t.ports.in)) S.wf.edges.push({ id: `e-${from.id}-${n.id}-${port}`, from: from.id, to: n.id, fromPort: port, toPort, label: toPort === 'in' ? '1 item' : null });
      }
    }
    S.wf.nodes.push(n); commit(); S.sel = n.id; S.selEdge = null; S.pending = null; S.panel.mode = 'cats'; S.panel.q = '';
    renderAll(); glowNode(n); toast(`Added ${t.label}`);
    return n;
  }
  function placeAfter(fromId: string, port: string) {
    const from = node(fromId)!; const { w, h } = size(from);
    if (isVertical(port)) { const a = anchor(from, port); return { x: a.x - 56, y: a.y + 60 }; }
    const taken = S.wf.edges.filter((e) => e.from === fromId && !isVertical(e.fromPort)).length;
    return { x: from.x + w + 90, y: from.y + (h - 82) / 2 + taken * 150 };
  }
  function removeNode(id: string) {
    const n = node(id); if (!n) return;
    S.wf.nodes = S.wf.nodes.filter((x) => x.id !== id && !x.id.startsWith(`${id}:`));
    S.wf.edges = S.wf.edges.filter((e) => e.from !== id && e.to !== id && !e.to.startsWith(`${id}:`));
    commit(); if (S.sel === id) { S.sel = null; if (S.panelView === 'inspector') closePanel(); } renderAll(); glowOff(); toast(`Removed ${n.title}`);
  }
  function connect(fromId: string, fromPort: string, toId: string, toPort: string) {
    if (fromId === toId) return false;
    const vertical = isVertical(fromPort);
    if (vertical !== (toPort === 'top')) { toast('That port does not accept this connection', 'warn'); return false; }
    if (S.wf.edges.some((e) => e.from === fromId && e.to === toId && e.fromPort === fromPort)) return false;
    if (toPort === 'in' && S.wf.edges.some((e) => e.to === toId && e.toPort === 'in')) { toast('That step already has an incoming connection (one parent per step)', 'warn'); return false; }
    if (toPort === 'top' && S.wf.edges.some((e) => e.to === toId)) { toast('Already attached to an agent', 'warn'); return false; }
    if (!vertical && wouldCycle(fromId, toId)) { toast('That would loop the flow back on itself', 'warn'); return false; }
    S.wf.edges = S.wf.edges.filter((e) => !(vertical && e.from === fromId && e.fromPort === fromPort));
    S.wf.edges.push({ id: `e-${fromId}-${toId}-${fromPort}`, from: fromId, to: toId, fromPort, toPort, label: vertical ? null : '1 item' }); commit(); renderAll(); toast('Connected'); return true;
  }
  function wouldCycle(from: string, to: string) { const seen = new Set<string>(); const q = [to]; while (q.length) { const id = q.shift()!; if (id === from) return true; if (seen.has(id)) continue; seen.add(id); for (const e of S.wf.edges) if (e.from === id && e.toPort === 'in') q.push(e.to); } return false; }

  /* ── pointer interactions ── */
  let drag: any = null;
  const onCanvasDown = (ev: PointerEvent) => {
    const target = ev.target as HTMLElement;
    if (ev.button === 1 || (ev.button === 0 && (S.spaceDown || ev.altKey))) { startPan(ev); return; }
    if (ev.button !== 0) return;
    const port = target.closest('[data-port]') as HTMLElement | null;
    if (port) { const nEl = port.closest('.node') as HTMLElement; startConnect(ev, nEl.dataset.id!, port.dataset.port!); return; }
    const sadd = target.closest('.sadd') as HTMLElement | null;
    if (sadd) { const nEl = sadd.closest('.node') as HTMLElement; openPendingPicker(nEl.dataset.id!, sadd.dataset.sub!); return; }
    if (target.closest('.caret')) { const nEl = target.closest('.node') as HTMLElement; const n = node(nEl.dataset.id!)!; n.expanded = !n.expanded; S.sel = n.id; S.selEdge = null; renderAll(); glowNode(n); return; }
    if (target.closest('.body') || target.closest('.edge-label')) return;
    const nEl = target.closest('.node') as HTMLElement | null;
    if (nEl) { startNodeDrag(ev, nEl.dataset.id!); return; }
    const edgeEl = target.closest('[data-edge]') as HTMLElement | null;
    if (edgeEl) { S.selEdge = edgeEl.dataset.edge!; S.sel = null; renderAll(); return; }
    startPan(ev);
  };
  function startPan(ev: PointerEvent) { D.canvas.classList.add('is-panning'); try { D.canvas.setPointerCapture(ev.pointerId); } catch {} drag = { kind: 'pan', sx: ev.clientX, sy: ev.clientY, vx: S.view.x, vy: S.view.y, moved: false }; }
  function startNodeDrag(ev: PointerEvent, id: string) {
    const n = node(id)!; try { D.canvas.setPointerCapture(ev.pointerId); } catch {}
    drag = { kind: 'node', id, sx: ev.clientX, sy: ev.clientY, nx: n.x, ny: n.y, moved: false, el: $(`.node[data-id="${CSS.escape(id)}"]`) };
    if (S.sel !== id || S.selEdge) { S.sel = id; S.selEdge = null; renderAll(); drag.el = $(`.node[data-id="${CSS.escape(id)}"]`); }
    glowNode(n);
  }
  function startConnect(ev: PointerEvent, id: string, port: string) { try { D.canvas.setPointerCapture(ev.pointerId); } catch {} D.canvas.classList.add('is-connecting'); drag = { kind: 'connect', from: id, port, moved: false, hot: null }; }
  const onCanvasMove = (ev: PointerEvent) => {
    if (!drag) return;
    if (drag.kind === 'pan') { S.view.x = drag.vx + ev.clientX - drag.sx; S.view.y = drag.vy + ev.clientY - drag.sy; drag.moved = drag.moved || Math.hypot(ev.clientX - drag.sx, ev.clientY - drag.sy) > 3; applyView(); return; }
    if (drag.kind === 'node') {
      const n = node(drag.id)!; n.x = drag.nx + (ev.clientX - drag.sx) / S.view.k; n.y = drag.ny + (ev.clientY - drag.sy) / S.view.k;
      if (!drag.moved && Math.hypot(ev.clientX - drag.sx, ev.clientY - drag.sy) > 3) { drag.moved = true; drag.el?.classList.add('is-dragging'); }
      if (drag.el) drag.el.style.transform = `translate(${n.x}px,${n.y}px)`; renderEdges(); glowNode(n); return;
    }
    if (drag.kind === 'connect') {
      drag.moved = true; const from = node(drag.from)!; const a = anchor(from, drag.port); const w = toWorld(ev.clientX, ev.clientY); const vertical = isVertical(drag.port);
      D.edgesTemp.innerHTML = `<path d="${pathFor({ x: a.x + WORLD_OFF, y: a.y + WORLD_OFF }, { x: w.x + WORLD_OFF, y: w.y + WORLD_OFF }, vertical)}"/>`;
      const under = document.elementFromPoint(ev.clientX, ev.clientY); const nEl = under && (under.closest('.node') as HTMLElement | null); const want = vertical ? 'top' : 'in';
      $$('.port.is-hot, .sport.is-hot').forEach((p) => p.classList.remove('is-hot')); $$('.node.is-target').forEach((p) => p.classList.remove('is-target')); drag.hot = null;
      if (nEl && nEl.dataset.id !== drag.from) { const p = $(`[data-port="${want}"]`, nEl); if (p) { p.classList.add('is-hot'); nEl.classList.add('is-target'); drag.hot = { id: nEl.dataset.id, port: want }; } }
    }
  };
  const onCanvasUp = (ev: PointerEvent) => {
    if (!drag) return; const d = drag; drag = null; D.canvas.classList.remove('is-panning', 'is-connecting');
    if (d.kind === 'pan') { if (!d.moved) { S.sel = null; S.selEdge = null; S.pending = null; renderAll(); glowOff(); closePanel(); } return; }
    if (d.kind === 'node') { d.el?.classList.remove('is-dragging'); if (d.moved) { commit(); renderAll(); } renderRatio(); return; }
    if (d.kind === 'connect') {
      D.edgesTemp.innerHTML = ''; $$('.node.is-target').forEach((p) => p.classList.remove('is-target'));
      if (d.hot) { connect(d.from, d.port, d.hot.id, d.hot.port); return; }
      if (d.moved) { const w = toWorld(ev.clientX, ev.clientY); openPendingPicker(d.from, d.port, { x: w.x - 20, y: w.y - 40 }); }
    }
  };
  function openPendingPicker(fromId: string, port: string, at?: { x: number; y: number }) {
    S.pending = { from: fromId, port, at: at || null }; S.sel = null; S.selEdge = null;
    S.panel.mode = isVertical(port) ? 'items' : 'cats'; S.panel.cat = isVertical(port) ? 'ai' : null; S.panel.q = ''; S.panel.focusSearch = !isVertical(port);
    renderAll(); openPanel('picker'); const n = node(fromId); if (n) glowNode(n);
  }
  const onWheel = (ev: WheelEvent) => { ev.preventDefault(); if (ev.ctrlKey || ev.metaKey) { zoomAt(ev.clientX, ev.clientY, Math.exp(-ev.deltaY * 0.0022)); return; } S.view.x -= ev.deltaX; S.view.y -= ev.deltaY; applyView(); };
  const onDbl = (ev: MouseEvent) => { const t = ev.target as HTMLElement; const nEl = t.closest('.node') as HTMLElement | null; if (!nEl || t.closest('.body')) return; const n = node(nEl.dataset.id!)!; S.sel = n.id; S.selEdge = null; renderAll(); glowNode(n); openPanel('inspector'); };
  on(D.canvas, 'pointerdown', onCanvasDown); on(D.canvas, 'pointermove', onCanvasMove); on(D.canvas, 'pointerup', onCanvasUp);
  on(D.canvas, 'wheel', onWheel, { passive: false }); on(D.canvas, 'dblclick', onDbl);

  on(D.nodes, 'change', (ev) => { const c = (ev.target as HTMLElement).closest('[data-cfg]') as HTMLInputElement | null; if (!c) return; const n = node((c.closest('.node') as HTMLElement).dataset.id!)!; const live = setFieldValue(n, c.dataset.cfg!, c.value); commit(); renderPanel(); if (live) toast(`${RULE_LABEL[boundKey(n, c.dataset.cfg!)!]} changed · save to write rules.json`); });
  on(D.nodes, 'click', (ev) => { const t = (ev.target as HTMLElement).closest('[data-cfg-toggle]') as HTMLElement | null; if (!t) return; const n = node((t.closest('.node') as HTMLElement).dataset.id!)!; (n.config as any)[t.dataset.cfgToggle!] = !(n.config as any)[t.dataset.cfgToggle!]; commit(); renderAll(); });
  on(D.labels, 'keydown', (ev) => { if (ev.key === 'Enter') { ev.preventDefault(); (ev.target as HTMLElement).blur(); } });
  on(D.labels, 'blur', (ev) => { const el = (ev.target as HTMLElement).closest('.edge-label') as HTMLElement | null; if (!el) return; const e = S.wf.edges.find((x) => x.id === el.dataset.edge); const v = el.textContent!.trim(); if (e && v !== e.label) { e.label = v || null; commit(); renderEdges(); } }, true);
  on(D.labels, 'pointerdown', (ev) => ev.stopPropagation());

  /* ── palette drag + clicks ── */
  let pal: any = null;
  on(D.panel, 'pointerdown', (ev) => { const item = (ev.target as HTMLElement).closest('.item') as HTMLElement | null; if (!item || (ev.target as HTMLElement).closest('.add')) return; pal = { type: item.dataset.type, sx: ev.clientX, sy: ev.clientY, moved: false }; try { D.panel.setPointerCapture(ev.pointerId); } catch {} });
  on(D.panel, 'pointermove', (ev) => {
    if (!pal) return;
    if (!pal.moved && Math.hypot(ev.clientX - pal.sx, ev.clientY - pal.sy) > 6) { pal.moved = true; D.ghost.innerHTML = `${icon(NODE_TYPES[pal.type].icon)}${esc(NODE_TYPES[pal.type].label)}`; D.ghost.classList.add('is-on'); }
    if (pal.moved) { D.ghost.style.left = `${ev.clientX}px`; D.ghost.style.top = `${ev.clientY}px`; const r = canvasRect(); const over = ev.clientX < r.right - (S.panelOpen ? 380 : 0) && ev.clientX > r.left && ev.clientY > r.top && ev.clientY < r.bottom; D.canvas.classList.toggle('is-dropping', over); }
  });
  on(D.panel, 'pointerup', (ev) => {
    if (!pal) return; const p = pal; pal = null; D.ghost.classList.remove('is-on'); D.canvas.classList.remove('is-dropping');
    if (p.moved) { const r = canvasRect(); if (ev.clientX > r.left && ev.clientX < r.right && ev.clientY > r.top && ev.clientY < r.bottom) { const w = toWorld(ev.clientX, ev.clientY); const t = NODE_TYPES[p.type]; const sq = t.shape === 'square'; addNode(p.type, { x: w.x - (sq ? 56 : 118), y: w.y - (sq ? 56 : 41) }, S.pending); } return; }
    addFromPicker(p.type);
  });
  function addFromPicker(typeId: string) {
    const t = NODE_TYPES[typeId];
    if (S.pending) { const at = S.pending.at || placeAfter(S.pending.from, S.pending.port); addNode(typeId, at, S.pending); return; }
    if (S.sel) { const from = node(S.sel)!; const port = T(from).ports.out ? 'out' : T(from).ports.outs ? T(from).ports.outs![0] : t.ports.top && subsOf(from).length ? subsOf(from)[0] : null; if (port) { addNode(typeId, placeAfter(from.id, port), { from: from.id, port }); return; } }
    const r = canvasRect(); const c = toWorld(r.left + (r.width - panelW()) / 2, r.top + r.height / 2 - 60); addNode(typeId, { x: c.x - 118, y: c.y - 41 }, null);
  }
  on(D.panel, 'click', (ev) => {
    const t = ev.target as HTMLElement;
    const add = t.closest('[data-add]') as HTMLElement | null; if (add) { addFromPicker(add.dataset.add!); return; }
    const cat = t.closest('[data-cat]') as HTMLElement | null; if (cat) { S.panel.mode = 'items'; S.panel.cat = cat.dataset.cat!; renderPanel(); return; }
    if (t.closest('[data-back]')) { S.panel.mode = 'cats'; S.panel.cat = null; renderPanel(); return; }
    const pb = t.closest('[data-panel]') as HTMLElement | null;
    if (pb) { if (pb.dataset.panel === 'library') return openLibrary(); S.panel.mode = 'cats'; S.panel.q = ''; S.panel.focusSearch = pb.dataset.panel === 'search'; openPanel('picker'); return; }
    const act = t.closest('[data-act]') as HTMLElement | null; if (act) { inspectorAction(act.dataset.act!); return; }
    const kind = t.closest('[data-kind]') as HTMLElement | null; if (kind) { const n = node(S.sel!)!; n.ownerKind = kind.dataset.kind as 'human' | 'agent'; if (n.ownerKind === 'human') { n.ownerAgentId = null; n.owner = opts.people[0] || 'Operator'; n.hours = Math.max(1, n.hours); } else n.owner = n.title; n.sub = n.owner; commit(); renderAll(); return; }
    const sw = t.closest('[data-cfg-toggle]') as HTMLElement | null; if (sw) { const n = node(S.sel!)!; (n.config as any)[sw.dataset.cfgToggle!] = !(n.config as any)[sw.dataset.cfgToggle!]; commit(); renderAll(); }
  });
  on(D.panel, 'input', (ev) => {
    const t = ev.target as HTMLInputElement;
    if (t.id === 'panelSearch') { S.panel.q = t.value; const scroll = $('.panel-scroll', D.panel); if (scroll) { const q = S.panel.q.trim().toLowerCase(); scroll.innerHTML = q ? searchHtml(q) : catsHtml(); } return; }
    if (!S.sel && !S.selEdge) return;
    if (t.matches('[data-edge-label]')) { const e = S.wf.edges.find((x) => x.id === S.selEdge)!; e.label = t.value || null; renderEdges(); return; }
    const n = node(S.sel!)!;
    if (t.matches('[data-title]')) { n.title = t.value; if (n.ownerKind === 'agent' && !n.ownerAgentId) { n.owner = t.value; n.sub = t.value; } renderNodes(); renderEdges(); renderRatio(); return; }
    if (t.matches('[data-detail]')) { n.detail = t.value; return; }
    if (t.matches('[data-owner]')) { n.owner = t.value; n.sub = t.value; renderNodes(); return; }
    if (t.matches('[data-hours]')) { n.hours = Number(t.value) || 0; renderRatio(); return; }
    if (t.matches('[data-auto-title]')) { n.automation!.title = t.value; return; }
    if (t.matches('[data-cfg]')) { setFieldValue(n, t.dataset.cfg!, t.value); renderNodes(); renderEdges(); }
  });
  on(D.panel, 'change', (ev) => {
    const t = ev.target as HTMLSelectElement;
    if (t.matches('[data-owner-agent]') && S.sel) { const n = node(S.sel)!; const a = agentsById.get(t.value); n.ownerAgentId = a ? a.id : null; n.owner = a ? a.name : n.title; n.sub = n.owner; commit(); renderAll(); return; }
    if (S.sel || S.selEdge) commit();
  });
  function inspectorAction(act: string) {
    const n = S.sel ? node(S.sel) : null;
    if (act === 'delete' && n) return removeNode(n.id);
    if (act === 'delete-edge') { S.wf.edges = S.wf.edges.filter((e) => e.id !== S.selEdge); S.selEdge = null; commit(); closePanel(); renderAll(); return toast('Connection removed'); }
    if (act === 'dup' && n) { const c: GraphNode = JSON.parse(JSON.stringify(n)); c.id = `${S.wf.id}-${uid('n')}`; c.x += 40; c.y += 120; c.title = `${n.title} copy`; S.wf.nodes.push(c); commit(); S.sel = c.id; renderAll(); glowNode(c); return toast('Duplicated'); }
    if (act === 'toggle-off' && n) { n.status = n.status === 'off' ? (n.automation?.state === 'suggested' ? 'suggested' : 'ok') : 'off'; n.expanded = false; commit(); renderAll(); return toast(n.status === 'off' ? `${n.title} deactivated` : `${n.title} active`); }
    if (act === 'promote' && n && n.automation) { n.automation.state = 'live'; n.status = 'ok'; commit(); renderAll(); return toast(`${n.automation.title} marked live`); }
    if (act === 'add-auto' && n) { n.automation = { title: `Automate: ${n.title}`, state: 'suggested', recoveredUsd: 0 }; n.status = 'suggested'; commit(); renderAll(); return toast('Automation suggested'); }
  }

  /* ── minimap + HUD ── */
  {
    const mm = need('#minimap'); let mmDrag = false;
    const centerAt = (clientX: number, clientY: number) => { const m = (D.minimap as any)._map; if (!m) return; const r = D.minimap.getBoundingClientRect(); const wx = (clientX - r.left - m.ox) / m.k + m.bx, wy = (clientY - r.top - m.oy) / m.k + m.by; const cr = canvasRect(); S.view.x = cr.width / 2 - wx * S.view.k; S.view.y = cr.height / 2 - wy * S.view.k; applyView(); };
    on(mm, 'pointerdown', (ev: PointerEvent) => { mmDrag = true; try { mm.setPointerCapture(ev.pointerId); } catch {} centerAt(ev.clientX, ev.clientY); ev.preventDefault(); });
    on(mm, 'pointermove', (ev: PointerEvent) => { if (mmDrag) centerAt(ev.clientX, ev.clientY); });
    on(mm, 'pointerup', () => { mmDrag = false; });
    on(mm, 'pointercancel', () => { mmDrag = false; });
  }
  const onRootClick = (ev: Event) => {
    const b = (ev.target as HTMLElement).closest('[data-action]') as HTMLElement | null; if (!b) return; const a = b.dataset.action; const r = canvasRect();
    if (a === 'fit') fitView(true);
    if (a === 'zoom-in') zoomAt(r.left + (r.width - 380) / 2, r.top + r.height / 2, 1.2);
    if (a === 'zoom-out') zoomAt(r.left + (r.width - 380) / 2, r.top + r.height / 2, 1 / 1.2);
    if (a === 'undo') undo();
    if (a === 'delete') { if (S.sel) removeNode(S.sel); else if (S.selEdge) inspectorAction('delete-edge'); else toast('Select a node or connection first', 'warn'); }
  };
  on(root, 'click', onRootClick);

  /* ── keyboard ── */
  const onKey = (ev: KeyboardEvent) => {
    if (!root.isConnected) return;
    const target = ev.target as HTMLElement; const typing = ['INPUT', 'TEXTAREA', 'SELECT'].includes(target.tagName) || target.isContentEditable;
    if (ev.key === 'Escape') { if (D.modal.classList.contains('is-open')) return closeModal(); if (D.menu.classList.contains('is-open')) return closeMenu(); if (typing) return target.blur(); if (!S.sel && !S.selEdge && S.panelOpen) return closePanel(); S.sel = null; S.selEdge = null; S.pending = null; renderAll(); glowOff(); closePanel(); return; }
    if (typing) return;
    if (!root.contains(document.activeElement) && document.activeElement !== document.body) return;
    if (ev.key === ' ') { S.spaceDown = true; D.canvas.style.cursor = 'grab'; ev.preventDefault(); }
    if ((ev.metaKey || ev.ctrlKey) && ev.key.toLowerCase() === 'z') { ev.preventDefault(); ev.shiftKey ? redo() : undo(); }
    if ((ev.metaKey || ev.ctrlKey) && ev.key.toLowerCase() === 'd') { ev.preventDefault(); if (S.sel) inspectorAction('dup'); }
    if ((ev.metaKey || ev.ctrlKey) && ev.key.toLowerCase() === 's') { ev.preventDefault(); save(); }
    if ((ev.metaKey || ev.ctrlKey) && ev.key.toLowerCase() === 'k') { ev.preventDefault(); S.panel.focusSearch = true; openPanel('picker'); }
    if (ev.key === 'Delete' || ev.key === 'Backspace') { if (S.sel) removeNode(S.sel); else if (S.selEdge) inspectorAction('delete-edge'); }
    if (ev.key === 'f' || ev.key === 'F') fitView(true);
    if (ev.key === '+' || ev.key === '=') { const r = canvasRect(); zoomAt(r.left + r.width / 2, r.top + r.height / 2, 1.2); }
    if (ev.key === '-') { const r = canvasRect(); zoomAt(r.left + r.width / 2, r.top + r.height / 2, 1 / 1.2); }
    if (ev.key === 'Enter' && S.sel) { const n = node(S.sel)!; if (T(n).fields.length) { n.expanded = !n.expanded; renderAll(); } }
    if (ev.key === ',' && (S.sel || S.selEdge)) { S.panelOpen && S.panelView === 'inspector' ? closePanel() : openPanel('inspector'); }
  };
  const onKeyUp = (ev: KeyboardEvent) => { if (ev.key === ' ') { S.spaceDown = false; D.canvas.style.cursor = ''; } };
  const onResize = () => { renderRatio(); drawMinimap(); };
  const onDocDown = (ev: PointerEvent) => { if (D.menu.classList.contains('is-open') && !(ev.target as HTMLElement).closest('.menu')) closeMenu(); };
  on(window, 'keydown', onKey); on(window, 'keyup', onKeyUp); on(window, 'resize', onResize); on(document, 'pointerdown', onDocDown, true);

  /* ── toasts / modals / menus ── */
  function toast(msg: string, kind?: string) { const t = document.createElement('div'); t.className = `toast ${kind || ''}`; t.innerHTML = `${icon(kind === 'warn' ? 'alert' : kind === 'err' ? 'x' : 'check')}<span>${esc(msg)}</span>`; D.toasts.appendChild(t); setTimeout(() => { t.classList.add('leave'); setTimeout(() => t.remove(), 260); }, kind === 'err' ? 5000 : 2400); }
  function openModal(html: string, cls?: string) { D.modal.innerHTML = `<div class="modal ${cls || ''}">${html}<button class="x" data-close>${icon('x')}</button></div>`; D.modal.classList.add('is-open'); }
  function closeModal() { D.modal.classList.remove('is-open'); D.modal.innerHTML = ''; }
  on(D.modal, 'click', (ev) => { if (ev.target === D.modal || (ev.target as HTMLElement).closest('[data-close]')) closeModal(); });
  function openMenu(anchorEl: HTMLElement, html: string) { const r = anchorEl.getBoundingClientRect(); D.menu.innerHTML = `<div class="menu">${html}</div>`; D.menu.classList.add('is-open'); const m = $('.menu', D.menu)!; m.style.position = 'fixed'; m.style.top = `${r.bottom + 8}px`; m.style.left = `${Math.min(r.left, window.innerWidth - 250)}px`; }
  function closeMenu() { D.menu.classList.remove('is-open'); D.menu.innerHTML = ''; }

  /* ── top bar behaviours ── */
  on(need('#wfTitle'), 'keydown', (ev) => { if ((ev as KeyboardEvent).key === 'Enter') { ev.preventDefault(); (ev.target as HTMLElement).blur(); } });
  on(need('#wfTitle'), 'blur', (ev) => { const v = (ev.target as HTMLElement).textContent!.trim(); if (v && v !== S.wf.name) { S.wf.name = v; commit(); renderTop(); toast('Renamed · save to keep'); } else renderTop(); });
  on(need('#wfCrumb'), 'click', () => { if (S.mode === 'editor') showLibrary(); });
  on(need('#btnSave'), 'click', () => save());
  on(need('#btnHistory'), 'click', openHistory);
  on(need('#btnMore'), 'click', (ev) => openMenu(ev.currentTarget as HTMLElement, `<button data-more="dup">${icon('copy')}Duplicate workflow</button><button data-more="export">${icon('external')}Export JSON</button><button data-more="fit">${icon('fit')}Fit to view<span class="kbd">F</span></button><div class="sep"></div><button data-more="delete" class="danger">${icon('trash')}Delete workflow</button>`));
  on(D.menu, 'click', (ev) => { const m = (ev.target as HTMLElement).closest('[data-more]') as HTMLElement | null; if (!m) return; closeMenu(); moreAction(m.dataset.more!); });
  on(need('#btnNewWorkflow'), 'click', () => openModal(`<h4>New workflow</h4><p>Starts with one step so the canvas is never empty. Saved to the real workflows table.</p><div class="fld"><label>Name</label><div class="ctl"><input type="text" id="newName" placeholder="e.g. Podcast guest pipeline"></div></div><div class="fld"><label>One line on what it does</label><div class="ctl"><input type="text" id="newSubtitle" placeholder="Optional"></div></div><div class="actions"><button class="btn" data-close>Cancel</button><button class="btn primary" data-create-wf>Create</button></div>`));

  async function save() {
    if (S.saving) return;
    const collapsed = collapseSubs(S.wf); const { input, problems } = graphToInput(collapsed);
    if (!input) { toast(problems[0], 'warn'); return; }
    S.saving = true; markDirty();
    try {
      const dirtyRules = Object.keys(S.rulesDirty) as (keyof BrandDealRules)[];
      if (dirtyRules.length && S.rules.configured) {
        const patch: Partial<BrandDealRules> = {}; for (const k of dirtyRules) (patch as any)[k] = (S.rules.rules as any)[k];
        const res = await opts.api.saveRules(patch, `canvas: ${dirtyRules.map((k) => RULE_LABEL[k] || k).join(', ')}`);
        S.rules = { configured: true, rules: res.rules, updatedAt: new Date().toISOString() }; S.rulesDirty = {};
        toast(`rules.json written · v${res.version.version} · the daemon reloads within two minutes`);
      }
      if (S.dirty) {
        const res = await opts.api.save(S.wf.id, { ...input, note: 'saved from the canvas' });
        replaceGraph(workflowToGraph(res.workflow)); S.dirty = false;
        toast(res.version ? `Saved · v${res.version.version}` : 'Saved · no changes');
      }
    } catch (e) { toast(`Save failed: ${(e as Error).message}`, 'err'); }
    S.saving = false; markDirty();
  }
  function replaceGraph(g: Graph) {
    const idx = S.wfs.findIndex((w) => w.id === g.id); if (idx >= 0) S.wfs[idx] = g; else S.wfs.push(g);
    const view = { ...S.view }; loadWorkflow(g.id); S.view = view; applyView();
  }
  async function openHistory() {
    openModal(`<h4>Version history</h4><p>Loading…</p>`);
    try {
      const versions = await opts.api.listVersions(S.wf.id);
      const hasRules = S.wf.nodes.some((n) => boundKey(n, 'prompt') || boundKey(n, 'criteria') || boundKey(n, 'auditEveryDays') || boundKey(n, 'uniqueHints'));
      const rv = hasRules ? await opts.api.listRuleVersions().catch(() => ({ versions: [], backups: [] })) : { versions: [], backups: [] };
      openModal(`<h4>Version history</h4><p>Every save records a version. Restoring writes a new version on top; nothing is deleted.</p>
        <div class="sec-t">Workflow · ${esc(S.wf.name)}</div>
        <div class="hist">${versions.length ? versions.map((v, i) => `<div class="h ${i === 0 ? 'is-current' : ''}" data-version="${esc(v.id)}"><span class="d"></span>v${v.version} · ${esc(v.note || 'saved')}<span class="mono">${esc(v.createdAt.slice(0, 16).replace('T', ' '))}</span></div>`).join('') : '<div class="empty">No versions yet. Save once.</div>'}</div>
        ${hasRules ? `<div class="sec-t" style="margin-top:16px">Daemon rules · rules.json on the mini</div><div class="hist">${rv.versions.length ? rv.versions.map((v, i) => `<div class="h ${i === 0 ? 'is-current' : ''}" data-rule-version="${esc(v.id)}"><span class="d"></span>v${v.version} · ${esc(v.note || 'written')}<span class="mono">${esc(v.createdAt.slice(0, 16).replace('T', ' '))}</span></div>`).join('') : `<div class="empty">${S.rules.configured ? 'No rule versions recorded yet.' : esc((S.rules as any).reason)}</div>`}${rv.backups.length ? `<div class="hint" style="margin:8px 0 0">${rv.backups.length} file backup${rv.backups.length > 1 ? 's' : ''} also sit next to rules.json on the mini.</div>` : ''}</div>` : ''}`);
    } catch (e) { openModal(`<h4>Version history</h4><p>${esc((e as Error).message)}</p>`); }
  }
  on(D.modal, 'click', async (ev) => {
    const t = ev.target as HTMLElement;
    const w = t.closest('[data-open-wf]') as HTMLElement | null; if (w) { closeModal(); loadWorkflow(w.dataset.openWf!); return; }
    const v = t.closest('[data-version]') as HTMLElement | null;
    if (v) { closeModal(); try { const res = await opts.api.restoreVersion(S.wf.id, v.dataset.version!); replaceGraph(workflowToGraph(res.workflow)); S.dirty = false; markDirty(); fitView(true); toast('Version restored'); } catch (e) { toast(`Restore failed: ${(e as Error).message}`, 'err'); } return; }
    const rvEl = t.closest('[data-rule-version]') as HTMLElement | null;
    if (rvEl) { closeModal(); try { const res = await opts.api.restoreRuleVersion(rvEl.dataset.ruleVersion!); S.rules = { configured: true, rules: res.rules, updatedAt: new Date().toISOString() }; S.rulesDirty = {}; renderAll(); toast('Rules restored · the daemon reloads within two minutes'); } catch (e) { toast(`Restore failed: ${(e as Error).message}`, 'err'); } return; }
    if (t.closest('[data-create-wf]')) { const name = ($('#newName', D.modal) as HTMLInputElement).value.trim(); const subtitle = ($('#newSubtitle', D.modal) as HTMLInputElement).value.trim(); if (!name) return toast('Give it a name', 'warn'); closeModal(); await createWorkflow(name, subtitle); return; }
    if (t.closest('[data-confirm-delete]')) { closeModal(); await deleteWorkflow(); return; }
    if (t.closest('[data-apply-draft]')) { const g = (D.modal as any)._draft as Graph | undefined; closeModal(); if (g) { S.wf.nodes = g.nodes; S.wf.edges = g.edges; S.wf.subtitle = g.subtitle || S.wf.subtitle; commit(); renderAll(); fitView(true); toast('Draft applied · save to keep it'); } return; }
  });
  async function createWorkflow(name: string, subtitle: string) {
    try {
      const res = await opts.api.create({ name, subtitle, steps: [{ title: 'First step', detail: 'Describe what starts this workflow.', ownerKind: 'human', owner: opts.people[0] || 'Operator', hoursPerWeek: 1, tools: [], automation: null, branchFromIndex: null, branchCondition: null, kind: 'human-task', config: {}, pos: { x: 80, y: 260 } }] });
      S.wfs.push(workflowToGraph(res.workflow)); loadWorkflow(res.workflow.id); S.panel.focusSearch = true; openPanel('picker'); toast(`Created ${name}`);
    } catch (e) { toast(`Create failed: ${(e as Error).message}`, 'err'); }
  }
  async function deleteWorkflow() {
    const id = S.wf.id;
    try { await opts.api.remove(id); S.wfs = S.wfs.filter((w) => w.id !== id); if (!S.wfs.length) { await createWorkflow('Untitled workflow', ''); } else loadWorkflow(S.wfs[0].id); toast('Workflow deleted'); }
    catch (e) { toast(`Delete failed: ${(e as Error).message}`, 'err'); }
  }
  function moreAction(a: string) {
    if (a === 'fit') fitView(true);
    if (a === 'dup') { const { input } = graphToInput(collapseSubs(S.wf)); if (!input) return toast('Fix the flow before duplicating', 'warn'); opts.api.create({ ...input, name: `${S.wf.name} (copy)` }).then((res) => { S.wfs.push(workflowToGraph(res.workflow)); loadWorkflow(res.workflow.id); toast('Workflow duplicated'); }).catch((e) => toast(`Duplicate failed: ${e.message}`, 'err')); }
    if (a === 'export') { const blob = new Blob([JSON.stringify(collapseSubs(S.wf), null, 2)], { type: 'application/json' }); const a2 = document.createElement('a'); a2.href = URL.createObjectURL(blob); a2.download = `${S.wf.id}.json`; a2.click(); toast('Exported'); }
    if (a === 'delete') openModal(`<h4>Delete ${esc(S.wf.name)}?</h4><p>Removes the workflow row. Its version history stays in the database.</p><div class="actions"><button class="btn" data-close>Cancel</button><button class="btn danger" data-confirm-delete>${icon('trash')}Delete</button></div>`);
  }
  function openLibrary() {
    openModal(`<h4>Workflows</h4><p>${S.wfs.length} mapped. Pick one to open it on the canvas.</p><div class="wf-grid">${S.wfs.map((w) => `<button class="wf-card ${w.id === S.wf.id ? 'is-current' : ''}" data-open-wf="${esc(w.id)}"><div class="n">${esc(w.name)}</div><div class="s">${esc(w.subtitle || 'No description')}</div><div class="m"><span class="d"></span>${w.nodes.filter((n) => !isSubNode(n)).length} steps · ${w.nodes.filter((n) => n.ownerKind === 'agent' && !isSubNode(n)).length} agent</div></button>`).join('')}</div>`);
  }

  /* ── tabs + lists ── */
  on(need('#tabs'), 'click', (ev) => { const t = (ev.target as HTMLElement).closest('.tab') as HTMLElement | null; if (!t) return; switchTab(t.dataset.tab!); });
  function switchTab(tab: string) {
    $$('.tab').forEach((b) => b.classList.toggle('is-active', b.dataset.tab === tab));
    need('#viewLibrary').classList.remove('is-active'); need('#viewEditor').classList.toggle('is-active', tab === 'editor'); need('#viewExecutions').classList.toggle('is-active', tab === 'executions'); need('#viewTests').classList.toggle('is-active', tab === 'tests');
    if (tab === 'executions') renderExecutions(); if (tab === 'tests') renderTests(); if (tab === 'editor') { renderRatio(); drawMinimap(); }
  }
  function wfRuns(): AgentRun[] { const ids = new Set(mainNodes().map((n) => n.ownerAgentId).filter(Boolean)); return S.runs.filter((r) => ids.has(r.agentId)).sort((a, b) => (a.startedAt < b.startedAt ? 1 : -1)); }
  function renderExecutions() {
    const all = wfRuns(); const rows = all.filter((r) => S.execFilter === 'all' || (S.execFilter === 'success' ? r.ok : !r.ok));
    need('#execSummary').textContent = all.length ? `${all.length} recorded run${all.length > 1 ? 's' : ''} · ${all.filter((r) => r.ok).length} ok · ${all.filter((r) => !r.ok).length} failed` : 'No recorded runs for this workflow’s agents yet.';
    $$('[data-exec-filter]').forEach((b) => b.classList.toggle('is-on', b.dataset.execFilter === S.execFilter));
    const note = S.wf.id === 'wf-brand-deals' ? `<div class="hint" style="margin:0 0 12px">The live pipeline runs as bdpilot on the mini and keeps its own log at ~/.config/bdpilot/bdpilot.log. Rows here are ${BRAND.name} agent runs (Test workflow, or POST /api/agents/:id/run).</div>` : '';
    const agentName = (id: string) => agentsById.get(id)?.name ?? id;
    need('#execTable').innerHTML = note + `<div class="row head"><span>Run</span><span>Agent</span><span>Status</span><span>Started</span><span>Duration</span><span>Summary</span><span></span></div>` + (rows.length ? rows.map((r) => `<div class="row" data-run="${esc(r.id)}"><span class="mono">${esc(r.id.slice(0, 12))}</span><span>${esc(agentName(r.agentId))}</span><span><span class="pill ${r.ok ? 'success' : 'failed'}">${r.ok ? 'ok' : 'failed'}</span></span><span class="mono">${esc(r.startedAt.slice(0, 16).replace('T', ' '))}</span><span class="mono">${esc(duration(r))}</span><span class="muted">${esc(r.summary.slice(0, 80))}</span><span class="go">${icon('chevRight')}</span></div><div class="trace" data-trace="${esc(r.id)}"><div class="tr">${esc(r.summary)}</div></div>`).join('') : '<div class="empty">Nothing recorded. Test workflow runs every bound agent and records a run per agent.</div>');
  }
  const duration = (r: AgentRun) => { const ms = Date.parse(r.finishedAt) - Date.parse(r.startedAt); return isNaN(ms) ? '' : ms < 1000 ? `${ms}ms` : `${(ms / 1000).toFixed(1)}s`; };
  on(need('#viewExecutions'), 'click', (ev) => {
    const t = ev.target as HTMLElement;
    const f = t.closest('[data-exec-filter]') as HTMLElement | null; if (f) { S.execFilter = f.dataset.execFilter!; renderExecutions(); return; }
    const r = t.closest('[data-run]') as HTMLElement | null; if (!r) return; const tr = $(`[data-trace="${CSS.escape(r.dataset.run!)}"]`); tr?.classList.toggle('is-open'); r.classList.toggle('is-on', !!tr?.classList.contains('is-open')); r.setAttribute('aria-expanded', String(!!tr?.classList.contains('is-open')));
  });
  function renderTests() {
    need('#testSummary').textContent = 'Workflow tests are not built yet.';
    need('#testTable').innerHTML = `<div class="empty" style="text-align:left;padding:24px">A workflow test will replay one recorded inbound through the bound agents and check which branch it takes. Nothing is recorded or simulated here until that exists.</div>`;
  }

  /* ── chat dock → drafting assistant ── */
  on(need('#btnChat'), 'click', () => { S.chatHidden = !S.chatHidden; need('#dockChat').classList.toggle('is-hidden', S.chatHidden); need('#btnChat').textContent = S.chatHidden ? 'Show chat' : 'Hide chat'; });
  need('#dockChat').classList.toggle('is-hidden', S.chatHidden); need('#btnChat').textContent = S.chatHidden ? 'Show chat' : 'Hide chat';
  function chatMsg(text: string, who: string) { const l = need('#chatLog'); const m = document.createElement('div'); m.className = `msg ${who}`; m.innerHTML = text; l.appendChild(m); l.scrollTop = l.scrollHeight; }
  on(need('#chatForm'), 'submit', async (ev) => {
    ev.preventDefault(); const inp = need<HTMLInputElement>('#chatInput'); const v = inp.value.trim(); if (!v) return; inp.value = ''; chatMsg(esc(v), 'me');
    chatMsg('Drafting with Claude on this machine…', 'ai');
    try {
      const res = await opts.api.draft(v);
      if (!res.ok || !res.draft) { chatMsg(esc(res.error || 'The drafting assistant is not available on this machine.'), 'ai'); return; }
      const g = workflowToGraph({ id: S.wf.id, name: res.draft.name, subtitle: res.draft.subtitle, revenueUsd: S.wf.revenueUsd, order: S.wf.order, steps: res.draft.steps.map((s, i) => ({ id: `${S.wf.id}-d${i + 1}`, title: s.title, detail: s.detail, ownerKind: s.ownerKind, owner: s.owner, ownerAgentId: opts.agents.find((a) => a.name.toLowerCase() === s.owner.toLowerCase())?.id ?? null, hoursPerWeek: s.hoursPerWeek, tools: s.tools, edgeLabel: null, leakUsd: null, automation: s.automation, branch: s.branchFromIndex != null ? { from: `${S.wf.id}-d${s.branchFromIndex + 1}`, condition: s.branchCondition || 'branch' } : null, kind: s.kind, config: s.config ?? {}, pos: null })) });
      (D.modal as any)._draft = g;
      chatMsg(`Drafted <b>${g.nodes.length} steps</b>: ${g.nodes.map((n) => esc(n.title)).join(', ')}.`, 'ai');
      openModal(`<h4>Apply the draft?</h4><p>Replaces the canvas with ${g.nodes.length} drafted steps. Nothing is saved until you press Save.</p><div class="hist">${g.nodes.map((n) => `<div class="h"><span class="d"></span>${esc(n.title)}<span class="mono">${esc(n.owner)}</span></div>`).join('')}</div><div class="actions"><button class="btn" data-close>Keep mine</button><button class="btn primary" data-apply-draft>Apply draft</button></div>`);
    } catch (e) { chatMsg(esc((e as Error).message), 'ai'); }
  });

  /* ── run: real agent runs, in flow order ── */
  function setTimer(ms: number) { const s = Math.floor(ms / 1000); need('#timerText').textContent = `${String(Math.floor(s / 60)).padStart(2, '0')}:${String(s % 60).padStart(2, '0')}`; }
  function log(text: string, kind?: string) { const b = need('#logsBody'); const ts = new Date().toLocaleTimeString([], { hour12: false }); const l = document.createElement('div'); l.className = `log ${kind || ''}`; l.innerHTML = `<span class="ts">${ts}</span><span>${text}</span>`; b.appendChild(l); b.scrollTop = b.scrollHeight; return l; }
  on(need('#btnLogsClose'), 'click', () => need('#dockLogs').classList.add('is-hidden'));
  on(need('#btnLogs'), 'click', () => { const d = need('#dockLogs'); d.classList.toggle('is-open'); need('#btnLogs').innerHTML = icon(d.classList.contains('is-open') ? 'chevDown' : 'chevUp'); });
  on(need('#btnPause'), 'click', () => { if (!S.running) return; S.paused = !S.paused; need('#btnPause').innerHTML = icon(S.paused ? 'play' : 'pause'); toast(S.paused ? 'Run paused' : 'Run resumed', S.paused ? 'warn' : ''); });
  async function waitWhilePaused() { while (S.paused) await delay(120); }
  function setRun(n: GraphNode, state: string | null) { (n as any).run = state; const el = $(`.node[data-id="${CSS.escape(n.id)}"]`); if (!el) return; el.classList.remove('is-running', 'is-done', 'is-waiting', 'is-failed'); if (state) el.classList.add(`is-${state}`); const st = $('.status', el)!; if (state === 'running') { st.dataset.s = 'running'; st.innerHTML = icon('refresh'); } else if (state === 'done') { st.dataset.s = 'ok'; st.innerHTML = icon('check'); } else if (state === 'waiting') { st.dataset.s = 'suggested'; st.innerHTML = icon('hourglass'); } else if (state === 'failed') { st.dataset.s = 'error'; st.innerHTML = icon('alert'); } }
  function setEdgeRun(e: GraphEdge, state: string | null) { (e as any).run = state; renderEdges(); }
  function clearRun() { for (const n of S.wf.nodes) setRun(n, null); for (const e of S.wf.edges) (e as any).run = null; renderEdges(); }
  async function runWorkflow() {
    if (S.running) return toast('A run is already in progress', 'warn');
    S.running = true; S.paused = false; need('#btnPause').innerHTML = icon('pause'); clearRun();
    const btn = need('#btnTest'); btn.textContent = 'Running…'; btn.classList.add('is-running');
    const logs = need('#dockLogs'); logs.classList.remove('is-hidden'); logs.classList.add('is-open'); need('#btnLogs').innerHTML = icon('chevDown'); need('#logsBody').innerHTML = '';
    // the timer shows only while a test runs; at rest "00:00" said nothing (H29)
    need('#dockBar').classList.add('is-timing');
    S.runStart = Date.now(); setTimer(0); S.timerId = setInterval(() => { if (!S.paused) setTimer(Date.now() - S.runStart); }, 250);
    const incomingMain = (id: string) => S.wf.edges.filter((e) => e.to === id && e.toPort === 'in');
    const roots = mainNodes().filter((n) => !incomingMain(n.id).length && n.status !== 'off');
    if (!roots.length) { toast('No starting step', 'warn'); return endRun(0, 0); }
    log(`Run started · ${esc(S.wf.name)} · every bound agent is executed for real through POST /api/agents/:id/run`, 'ok');
    const visited = new Set<string>(); const queue = roots.slice(); let count = 0, failed = 0;
    while (queue.length) {
      const n = queue.shift()!; if (visited.has(n.id) || n.status === 'off') continue; visited.add(n.id); await waitWhilePaused();
      const incoming = incomingMain(n.id); for (const e of incoming) setEdgeRun(e, 'active');
      setRun(n, 'running'); glowNode(n); count++; need('#logsTitle').textContent = `Latest logs from ${n.title}`;
      const subs = S.wf.edges.filter((e) => e.from === n.id && isVertical(e.fromPort)).map((e) => node(e.to)).filter(Boolean) as GraphNode[]; for (const s of subs) setRun(s, 'running');
      let ok = true;
      if (n.ownerKind === 'human') { log(`${esc(n.title)} · manual step owned by ${esc(n.owner)} · not simulated`, 'warn'); await delay(300); }
      else if (!n.ownerAgentId) { log(`${esc(n.title)} · not bound to a roster agent · skipped`, 'warn'); ok = false; await delay(300); }
      else {
        log(`${esc(n.title)} · running ${esc(agentsById.get(n.ownerAgentId)?.name ?? n.ownerAgentId)}…`);
        try { const run = await opts.api.runAgent(n.ownerAgentId); S.runs.unshift(run); ok = run.ok; log(`${esc(n.title)} · ${run.ok ? 'ok' : 'failed'} · ${esc(run.summary)}`, run.ok ? 'ok' : 'err'); }
        catch (e) { ok = false; log(`${esc(n.title)} · ${esc((e as Error).message)}`, 'err'); }
      }
      if (!ok) failed++;
      setRun(n, ok ? 'done' : 'failed'); for (const s of subs) setRun(s, 'done'); for (const e of incoming) setEdgeRun(e, 'done');
      let outs = S.wf.edges.filter((e) => e.from === n.id && e.toPort === 'in');
      if (YES_NO_KINDS.has(n.type)) {
        const cfg = n.config as any; const yes = outs.filter((e) => e.fromPort === 'yes'); const no = outs.filter((e) => e.fromPort === 'no');
        const chosen = yes.length ? yes : no; const label = yes.length ? cfg.yes || 'yes' : cfg.no || 'no';
        log(`${esc(n.title)} → following "${esc(label)}" (the outcome is not decided here; the run walks the first wired branch)`, 'warn'); outs = chosen;
      }
      for (const e of outs) { const nx = node(e.to); if (nx) queue.push(nx); }
      await delay(180);
    }
    log(`Run complete · ${count} steps · ${failed} failed · ${need('#timerText').textContent}`, failed ? 'warn' : 'ok');
    endRun(count, failed);
  }
  function endRun(count: number, failed: number) {
    S.running = false; clearInterval(S.timerId); need('#dockBar').classList.remove('is-timing'); const btn = need('#btnTest'); btn.textContent = 'Test workflow'; btn.classList.remove('is-running');
    if (count) toast(`Run complete · ${count} steps${failed ? ` · ${failed} failed` : ''}`, failed ? 'warn' : ''); renderRatio();
    setTimeout(() => { if (!S.running) { clearRun(); if (S.sel) glowNode(node(S.sel)!); else glowOff(); } }, 6000);
  }
  on(need('#btnTest'), 'click', () => runWorkflow());

  /* ── ratio widget collapse ── */
  on(need('#ratioMini'), 'click', () => { need('#dockRatio').classList.remove('is-collapsed'); renderRatio(); });
  on(need('#btnRatioCollapse'), 'click', () => need('#dockRatio').classList.add('is-collapsed'));

  /* ── capsules ── */
  on(need('#btnPanelToggle'), 'click', () => { if (S.panelOpen && S.panelView === 'picker') { closePanel(); return; } S.panel.mode = 'cats'; S.panel.q = ''; S.panel.focusSearch = true; openPanel('picker'); });
  on(need('#btnSettings'), 'click', () => { if (!S.sel && !S.selEdge) return; if (S.panelOpen && S.panelView === 'inspector') { closePanel(); return; } openPanel('inspector'); });

  /* ── static icons ── */
  need('#btnNewWorkflow').innerHTML = icon('plus') + (opts.skin === 'g' ? '<span class="wfc-g-label">New workflow</span>' : ''); need('#btnHistory').innerHTML = icon('history'); need('#btnMore').innerHTML = icon('more');
  need('[data-action="fit"]').innerHTML = icon('fit'); $$('[data-action="zoom-in"]').forEach((b) => (b.innerHTML = icon('plus'))); $$('[data-action="zoom-out"]').forEach((b) => (b.innerHTML = icon('minus'))); $$('[data-action="undo"]').forEach((b) => (b.innerHTML = icon('undo')));
  need('#ratioArrowMini').innerHTML = icon('chevRight'); need('#btnRatioCollapse').innerHTML = icon('chevDown');
  need('#btnPanelToggle').innerHTML = icon('plus'); need('#btnSettings').innerHTML = icon('settings'); need('.chat-send').innerHTML = icon('arrowRight'); need('#btnPause').innerHTML = icon('pause'); need('#btnLogs').innerHTML = icon('chevUp'); need('#btnLogsClose').innerHTML = icon('x'); need('#ratioArrow').innerHTML = icon('chevRight');

  /* ── load / boot ── */
  function loadWorkflow(id: string) {
    const base = S.wfs.find((w) => w.id === id) || S.wfs[0]; if (!base) return; S.sizes = {};
    S.wf = expandSubs(JSON.parse(JSON.stringify(base)));
    S.history = []; S.future = []; S.current = snap(); S.sel = null; S.selEdge = null; S.pending = null; S.panel = { mode: 'cats', cat: null, q: '', focusSearch: false }; S.execFilter = 'all'; S.dirty = false; S.rulesDirty = {};
    clearRun(); need('#chatLog').innerHTML = ''; chatMsg(`This is <b>${esc(S.wf.name)}</b>. Describe the workflow you want and the drafting assistant proposes steps for review.`, 'ai');
    showEditor(); renderAll(); fitWhenLaidOut(); glowOff(); closePanel();
    try { history.replaceState(null, '', `/workflows?wf=${encodeURIComponent(S.wf.id)}`); } catch {}
    if (need('#viewExecutions').classList.contains('is-active')) renderExecutions(); if (need('#viewTests').classList.contains('is-active')) renderTests();
  }
  if (!S.wfs.length) { showEmptyLibrary(root); return { destroy() {} }; }
  if (opts.initialId && S.wfs.some((w) => w.id === opts.initialId)) loadWorkflow(opts.initialId);
  else { S.wf = expandSubs(JSON.parse(JSON.stringify(S.wfs[0]))); S.current = snap(); showLibrary(); }
  const onBeforeUnload = (ev: BeforeUnloadEvent) => { if (S.dirty || Object.keys(S.rulesDirty).length) { ev.preventDefault(); ev.returnValue = ''; } };
  on(window, 'beforeunload', onBeforeUnload);
  // Appearance switched while mounted (S04 edge 9, X3): the minimap and the
  // library thumbs are drawn with computed colours, so draw them again.
  const offAppearance = onAppearanceChange(() => {
    try {
      drawMinimap();
      if (need('#viewLibrary').classList.contains('is-active')) renderLibrary();
    } catch {}
  });
  let kitObserver: MutationObserver | null = null;
  if (opts.skin === 'g') {
    kitify(root);
    kitObserver = new MutationObserver((records) => {
      for (const r of records) r.addedNodes.forEach((n) => { if (n.nodeType === 1) { kitify(n as Element); const el = n as HTMLElement; if (el.matches?.('button, a[href], input, select, textarea')) kitify(el.parentElement ?? el); } });
    });
    kitObserver.observe(root, { childList: true, subtree: true });
  }
  const handle = {
    destroy() {
      ac.abort();
      offAppearance();
      kitObserver?.disconnect();
      clearInterval(S.timerId);
      delete (root as any).__glados;
    },
  };
  (root as any).__glados = { S, api: { addNode, connect, removeNode, undo, redo, save, runWorkflow, switchTab, fitView, node, anchor }, destroy: handle.destroy };
  return handle;
}

/**
 * No workflows: only the library view stays, with the empty line in its grid.
 * The editor markup goes (as the old whole-root replace did, so no unwired
 * control is left behind), but the library view and its React-rendered slot
 * (the F10 Scheduled card) survive, and a second mount (strict mode) finds
 * them again. React removes `.wfc` itself on unmount, never these children.
 */
function showEmptyLibrary(root: HTMLElement) {
  root.classList.add('is-library', 'is-empty');
  for (const child of [...root.children]) if (child.id !== 'viewLibrary') child.remove();
  root.querySelectorAll<HTMLElement>('.view').forEach((v) => v.classList.toggle('is-active', v.id === 'viewLibrary'));
  const grid = root.querySelector<HTMLElement>('#libGrid');
  if (grid) grid.innerHTML = '<div class="empty" style="padding:60px">No workflows in the database. Run the seed, or create one from the API.</div>';
}
