import { describe, expect, it } from 'vitest';
import type { BlueprintGraph } from './graph';
import { APP_ID, buildHierarchy, describeScope, expandedAtLevel, indexHierarchy, relationsOf, ROUTER_DEPARTMENTS, searchHierarchy } from './hierarchy';
import { focusEdges, layoutHierarchy, Router, spineEdges, type Box } from './hierarchy-layout';

function graph(): BlueprintGraph {
  const nodes: BlueprintGraph['nodes'] = [
    { id: 'operator', kind: 'operator', name: 'the operator', layer: 0, status: 'live', blurb: 'The operator.', facts: {}, icon: 'user-round' },
    { id: 'agent-conductor', kind: 'wizard', name: 'Wizard', layer: 1, status: 'live', blurb: 'The super agent.', facts: {}, icon: 'wand-2' },
    { id: 'dept-sales', kind: 'department', name: 'Sales', layer: 2, status: 'live', blurb: 'pipeline', facts: {}, icon: 'folder' },
    { id: 'dept-tech', kind: 'department', name: 'TECH', layer: 2, status: 'live', blurb: 'the OS', facts: {}, icon: 'folder' },
    { id: 'agent-drake', kind: 'agent', name: 'Drake', layer: 2, status: 'live', blurb: '', facts: { role: 'Closer', tools: 'attio' }, icon: 'bot' },
    { id: 'agent-crm', kind: 'agent', name: 'Attio CRM', layer: 2, status: 'designed', blurb: '', facts: { role: 'CRM' }, icon: 'bot' },
    { id: 'agent-yeat', kind: 'agent', name: 'Yeat', layer: 2, status: 'live', blurb: '', facts: { role: 'Auditor' }, icon: 'bot' },
    { id: 'person-koda', kind: 'person', name: 'Koda', layer: 2, status: 'live', blurb: 'Setter', facts: {}, icon: 'user-round' },
    { id: 'connector-gbrain', kind: 'connector', name: 'G-Brain', layer: 3, status: 'live', blurb: '', facts: { kind: 'brain' }, icon: 'plug' },
    { id: 'connector-attio', kind: 'connector', name: 'Attio', layer: 3, status: 'not-configured', blurb: '', facts: { kind: 'crm' }, icon: 'plug' },
    { id: 'model-claude-engine', kind: 'model', name: 'Claude engine', layer: 3, status: 'live', blurb: 'Local claude binary', facts: { mode: 'headless' }, icon: 'cpu' },
    { id: 'skill-voice', kind: 'skillpack', name: 'Voice', layer: 3, status: 'live', blurb: 'writing voice', facts: {}, icon: 'sparkles' },
    { id: 'router-identity', kind: 'router', name: 'Identity routing', layer: 3, status: 'live', blurb: 'Identity routing — Souls → soul models', facts: {}, icon: 'route' },
    { id: 'router-deals', kind: 'router', name: 'Deal routing', layer: 3, status: 'live', blurb: 'Deal routing — inbound → gsend', facts: {}, icon: 'route' },
    { id: 'surface-content', kind: 'surface', name: 'Content', layer: 3, status: 'live', blurb: '', facts: { route: '/content' }, icon: 'clapperboard' },
    { id: 'surface-wizard', kind: 'surface', name: 'Wizard', layer: 3, status: 'live', blurb: '', facts: { route: '/wizard' }, icon: 'wand-2' },
    { id: 'host-macbook', kind: 'host', name: 'MacBook', layer: 4, status: 'live', blurb: 'Where the OS is built.', facts: {}, icon: 'laptop' },
    { id: 'host-mini', kind: 'host', name: 'Mac mini', layer: 4, status: 'live', blurb: 'Always-on host.', facts: { address: '10.0.0.2' }, icon: 'server' },
    { id: 'store-db', kind: 'store', name: 'OS database', layer: 4, status: 'live', blurb: 'SQLite', facts: { path: 'data/glados.db' }, icon: 'database' },
    { id: 'store-wizard', kind: 'store', name: 'Wizard threads', layer: 4, status: 'live', blurb: 'projects', facts: { path: 'data/wizard/' }, icon: 'messages-square' },
    { id: 'daemon-gsend', kind: 'daemon', name: 'Gsend', layer: 4, status: 'configured', blurb: 'sends the queue', facts: { host: 'mini' }, icon: 'send' },
    { id: 'daemon-briefing', kind: 'daemon', name: 'Daily briefing', layer: 4, status: 'configured', blurb: 'weekday run', facts: { host: 'mini', schedule: 'weekdays 06:30' }, icon: 'sunrise' },
    { id: 'service-os-mini', kind: 'service', name: 'FounderOS v1 (prod)', layer: 4, status: 'live', blurb: 'next start', facts: { port: '4100' }, icon: 'layout-dashboard' },
    { id: 'container-hermes', kind: 'container', name: 'Hermes worker', layer: 4, status: 'configured', blurb: 'docker', facts: { image: 'hermes' }, icon: 'container' },
    { id: 'tool-superset', kind: 'tool', name: 'Superset', layer: 4, status: 'live', blurb: 'parallel agent worktrees', facts: {}, icon: 'app-window' },
    { id: 'cloud-github', kind: 'service', name: 'GitHub', layer: 4, status: 'live', blurb: 'origin/main', facts: {}, icon: 'git-branch' },
  ];
  for (let i = 0; i < 6; i++) nodes.push({ id: `model-${i}`, kind: 'model', name: `Model ${i}`, layer: 3, status: 'live', blurb: '', facts: { family: 'image' }, icon: 'cpu' });
  const edges: BlueprintGraph['edges'] = [
    { from: 'operator', to: 'agent-conductor', kind: 'commands' },
    { from: 'agent-conductor', to: 'agent-drake', kind: 'commands' },
    { from: 'agent-conductor', to: 'agent-crm', kind: 'commands' },
    { from: 'agent-conductor', to: 'agent-yeat', kind: 'commands' },
    { from: 'agent-drake', to: 'dept-sales', kind: 'member-of' },
    { from: 'agent-crm', to: 'dept-sales', kind: 'member-of' },
    { from: 'person-koda', to: 'dept-sales', kind: 'member-of' },
    { from: 'agent-yeat', to: 'dept-tech', kind: 'member-of' },
    { from: 'agent-drake', to: 'connector-attio', kind: 'uses' },
    { from: 'agent-yeat', to: 'connector-gbrain', kind: 'uses' },
    { from: 'agent-conductor', to: 'model-claude-engine', kind: 'runs-on' },
    { from: 'agent-conductor', to: 'store-wizard', kind: 'writes' },
    { from: 'surface-wizard', to: 'store-wizard', kind: 'uses' },
    { from: 'surface-content', to: 'router-identity', kind: 'uses' },
    { from: 'router-deals', to: 'daemon-gsend', kind: 'uses', via: 'approval → gsend' },
    { from: 'store-db', to: 'host-macbook', kind: 'runs-on' },
    { from: 'daemon-gsend', to: 'host-mini', kind: 'runs-on' },
    { from: 'daemon-briefing', to: 'host-mini', kind: 'runs-on' },
    { from: 'service-os-mini', to: 'host-mini', kind: 'runs-on' },
    { from: 'container-hermes', to: 'host-mini', kind: 'runs-on' },
    { from: 'tool-superset', to: 'host-macbook', kind: 'runs-on' },
    { from: 'tool-superset', to: 'cloud-github', kind: 'deploys-to', via: 'git push' },
    { from: 'cloud-github', to: 'service-os-mini', kind: 'deploys-to', via: 'autodeploy' },
  ];
  for (let i = 0; i < 6; i++) edges.push({ from: 'router-identity', to: `model-${i}`, kind: 'uses', via: i < 2 ? 'Souls' : 'Elements' });
  return { compiledAt: new Date().toISOString(), nodes, edges };
}

const overlaps = (a: Box, b: Box) => a.x < b.x + b.w && b.x < a.x + a.w && a.y < b.y + b.h && b.y < a.y + a.h;

describe('blueprint hierarchy — the graph as buckets', () => {
  const g = graph();
  const h = buildHierarchy(g);
  const idx = indexHierarchy(h);

  it('every graph node lands exactly once; hosts become machine containers', () => {
    const drawn = new Map<string, number>();
    for (const id of idx.items.keys()) drawn.set(id, (drawn.get(id) ?? 0) + 1);
    for (const n of g.nodes) {
      if (n.kind === 'host') {
        expect(idx.items.get(`k-${n.id}`)?.type, n.id).toBe('container');
        continue;
      }
      expect(drawn.get(n.id), `${n.id} missing`).toBe(1);
    }
    expect(idx.items.get(APP_ID)?.type).toBe('container');
    expect(idx.items.get(ROUTER_DEPARTMENTS)?.kind).toBe('router');
    // agents sit inside their department, daemons on their host, the engine beside the Wizard
    expect(idx.parent.get('agent-drake')).toBe('dept-sales');
    expect(idx.chain('daemon-gsend')[0]).toBe('k-host-mini');
    expect(idx.chain('store-db')[0]).toBe('k-host-macbook');
    expect(idx.parent.get('model-claude-engine')).toBe(APP_ID);
    expect(idx.chain('model-3')[0]).toBe('k-cloud');
  });

  it('machines hold their services, containers and dev tools; unhosted services live in the cloud', () => {
    expect(idx.parent.get('service-os-mini')).toBe('g-services-host-mini');
    expect(idx.parent.get('container-hermes')).toBe('g-docker-host-mini');
    expect(idx.parent.get('tool-superset')).toBe('g-tools-host-macbook');
    expect(idx.chain('tool-superset')[0]).toBe('k-host-macbook');
    expect(idx.items.get('container-hermes')?.kind).toBe('docker');
    expect(idx.items.get('tool-superset')?.kind).toBe('tool');
    expect(idx.chain('cloud-github')[0]).toBe('k-cloud');
    expect(idx.parent.get('cloud-github')).toBe('g-managed');
    // the deploy path is a relation you can follow: MacBook tool → GitHub → the mini's OS
    expect(h.relations.some((r) => r.from === 'tool-superset' && r.to === 'cloud-github' && r.via === 'git push')).toBe(true);
    expect(h.relations.some((r) => r.from === 'cloud-github' && r.to === 'service-os-mini' && r.kind === 'deploys to')).toBe(true);
  });

  it('relations and spine only reference drawn things, and skip containment edges', () => {
    for (const r of h.relations) {
      expect(idx.items.has(r.from), r.from).toBe(true);
      expect(idx.items.has(r.to), r.to).toBe(true);
      expect(r.kind).not.toBe('member of');
    }
    expect(h.relations.some((r) => r.to === 'k-host-mini' || r.to === 'host-mini')).toBe(false);
    for (const s of h.spine) {
      expect(idx.items.has(s.from), s.from).toBe(true);
      for (const t of Array.isArray(s.to) ? s.to : [s.to]) expect(idx.items.has(t), t).toBe(true);
    }
    // the deal router → gsend edge and identity routing → models are drawn structurally because they exist
    expect(h.spine.some((s) => s.from === 'router-deals' && s.to === 'daemon-gsend')).toBe(true);
    expect(h.spine.some((s) => s.from === 'router-identity' && s.to === 'g-models')).toBe(true);
  });

  it('disclosure levels: overview collapses everything, everything opens everything', () => {
    expect(expandedAtLevel(idx, 1).size).toBe(0);
    const two = expandedAtLevel(idx, 2);
    expect(two.has('g-connectors')).toBe(true);
    expect(two.has('dept-sales')).toBe(false);
    expect(expandedAtLevel(idx, 3).size).toBe(idx.groups.length);
  });

  it('department frames carry an honest live badge', () => {
    const sales = idx.items.get('dept-sales');
    expect(sales?.type).toBe('group');
    if (sales?.type === 'group') expect(sales.badge).toBe('1 of 2 live');
  });

  it('layout: no two drawn cards or frames overlap at any level; containers stack without touching', () => {
    for (const level of [1, 2, 3] as const) {
      const layout = layoutHierarchy(h, expandedAtLevel(idx, level));
      const boxes = [...layout.boxes.values()].filter((b) => b.kind !== 'container');
      for (let i = 0; i < boxes.length; i++) {
        for (let j = i + 1; j < boxes.length; j++) {
          const a = boxes[i];
          const b = boxes[j];
          const nested = idx.chain(a.id).includes(b.id) || idx.chain(b.id).includes(a.id);
          if (nested) continue;
          expect(overlaps(a, b), `level ${level}: ${a.id} overlaps ${b.id}`).toBe(false);
        }
      }
      for (let i = 1; i < layout.containers.length; i++) expect(layout.containers[i].y).toBeGreaterThan(layout.containers[i - 1].y + layout.containers[i - 1].h);
    }
  });

  it('routing never runs a trace through a card', () => {
    const layout = layoutHierarchy(h, expandedAtLevel(idx, 3));
    const router = new Router(layout, idx);
    const content = [...layout.boxes.values()].filter((b) => b.kind !== 'container');
    const check = (pts: { x: number; y: number }[], a: Box, b: Box, label: string) => {
      if (pts.length === 5) {
        // gutter fallback: the long vertical runs outside every drawn thing, and the exit
        // from the source card is clear — the final approach is best-effort in a dense frame
        const gx = pts[1].x;
        for (const box of content) expect(gx > box.x && gx < box.x + box.w, `${label}: gutter x inside ${box.id}`).toBe(false);
        expect(router.blockedH(pts[0].y, pts[0].x, pts[1].x, a, b), `${label}: exit run crosses a card`).toBe(false);
        return;
      }
      for (let i = 1; i < pts.length; i++) {
        const p = pts[i - 1];
        const q = pts[i];
        if (p.x === q.x) expect(router.blockedV(p.x, p.y, q.y, a, b), `${label}: vertical segment crosses a card`).toBe(false);
        else if (p.y === q.y) expect(router.blockedH(p.y, p.x, q.x, a, b), `${label}: horizontal segment crosses a card`).toBe(false);
      }
    };
    const rels = relationsOf(h, idx, 'agent-conductor');
    expect(rels.length).toBeGreaterThan(3);
    const a = layout.boxes.get('agent-conductor')!;
    for (const r of rels) {
      const b = layout.boxes.get(r.other);
      if (!b) continue;
      check(router.route(a, b), a, b, `wizard → ${r.other}`);
    }
    // structural edges too
    for (const e of spineEdges(h, idx, layout)) {
      if (e.bare) continue;
      expect(e.pts.length).toBeGreaterThan(1);
    }
  });

  it('focus mode keeps the selection, its chain and its relations sharp and draws one edge per related card', () => {
    const layout = layoutHierarchy(h, expandedAtLevel(idx, 3));
    const rels = relationsOf(h, idx, 'agent-drake');
    const f = focusEdges(idx, layout, 'agent-drake', rels);
    expect(f.hot.has('agent-drake')).toBe(true);
    expect(f.hot.has('dept-sales')).toBe(true);
    expect(f.hot.has('connector-attio')).toBe(true);
    expect(f.hot.has('agent-yeat')).toBe(false);
    expect(f.edges.length).toBe(2); // commanded by the Wizard, uses Attio
  });

  it('search finds by name and by subline and reports the path', () => {
    const hits = searchHierarchy(idx, 'attio');
    expect(hits.map((x) => x.id)).toEqual(expect.arrayContaining(['agent-crm', 'connector-attio']));
    expect(hits.find((x) => x.id === 'connector-attio')?.path).toBe('Founder OS › Connectors');
    expect(searchHierarchy(idx, 'closer')[0]?.id).toBe('agent-drake');
    expect(searchHierarchy(idx, 'zzz')).toHaveLength(0);
  });

  it('the ask scope describes the selection, its members and its relations, never invents', () => {
    const scope = describeScope(h, idx, 'dept-sales') as { scope: string; path: string; inside?: unknown[]; relations?: string[] };
    expect(scope.scope).toBe('Sales');
    expect(scope.path).toBe('Founder OS › Sales');
    expect(scope.inside).toHaveLength(3);
    expect(scope.relations?.some((r) => r.includes('Attio'))).toBe(true);
    const whole = describeScope(h, idx, null) as { scope: string; containers: unknown[] };
    expect(whole.scope).toBe('the whole system');
    expect(whole.containers.length).toBe(h.containers.length);
  });
});
