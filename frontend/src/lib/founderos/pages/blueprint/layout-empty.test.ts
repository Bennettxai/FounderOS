import { describe, expect, it } from 'vitest';
import type { BlueprintGraph } from './graph';
import { buildHierarchy, expandedAtLevel, indexHierarchy } from './hierarchy';
import { layoutHierarchy } from './hierarchy-layout';

/** A department with no agents yet must not poison the layout (NaN widths
 *  once collapsed every department frame onto the origin in Everything mode). */
function graph(): BlueprintGraph {
  const nodes: BlueprintGraph['nodes'] = [
    { id: 'operator', kind: 'operator', name: 'the operator', layer: 0, status: 'live', blurb: 'The operator.', facts: {}, icon: 'user-round' },
    { id: 'agent-conductor', kind: 'wizard', name: 'Wizard', layer: 1, status: 'live', blurb: 'The super agent.', facts: {}, icon: 'wand-2' },
    { id: 'dept-sales', kind: 'department', name: 'Sales', layer: 2, status: 'live', blurb: 'pipeline', facts: {}, icon: 'folder' },
    { id: 'dept-finance', kind: 'department', name: 'Finances', layer: 2, status: 'live', blurb: 'books', facts: {}, icon: 'folder' },
    { id: 'dept-tech', kind: 'department', name: 'TECH', layer: 2, status: 'live', blurb: 'the OS', facts: {}, icon: 'folder' },
    { id: 'agent-drake', kind: 'agent', name: 'Drake', layer: 2, status: 'live', blurb: '', facts: { role: 'Sales' }, icon: 'bot' },
    { id: 'agent-yeat', kind: 'agent', name: 'Yeat', layer: 2, status: 'live', blurb: '', facts: { role: 'Auditor' }, icon: 'bot' },
  ];
  const edges: BlueprintGraph['edges'] = [
    { from: 'operator', to: 'agent-conductor', kind: 'commands' },
    { from: 'agent-conductor', to: 'agent-drake', kind: 'commands' },
    { from: 'agent-conductor', to: 'agent-yeat', kind: 'commands' },
    { from: 'agent-drake', to: 'dept-sales', kind: 'member-of' },
    { from: 'agent-yeat', to: 'dept-tech', kind: 'member-of' },
  ];
  return { nodes, edges, compiledAt: new Date().toISOString() } as BlueprintGraph;
}

describe('layoutHierarchy — an empty department never breaks the map', () => {
  it('every box is finite in Everything mode, and department frames do not pile onto one point', () => {
    const h = buildHierarchy(graph());
    const idx = indexHierarchy(h);
    const layout = layoutHierarchy(h, expandedAtLevel(idx, 3));
    const boxes = [...layout.boxes.values()];
    expect(boxes.length).toBeGreaterThan(3);
    for (const b of boxes) {
      for (const k of ['x', 'y', 'w', 'h'] as const) expect(Number.isFinite(b[k]), `${b.id}.${k}`).toBe(true);
      expect(b.w, `${b.id}.w`).toBeGreaterThan(0);
    }
    const depts = boxes.filter((b) => b.id.startsWith('dept-'));
    expect(depts.length).toBe(3);
    const spots = new Set(depts.map((b) => `${b.x},${b.y}`));
    expect(spots.size).toBe(3);
  });
});
