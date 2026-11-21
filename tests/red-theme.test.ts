import { existsSync, readFileSync } from 'node:fs';
import { join } from 'node:path';
import { describe, expect, test } from 'vitest';

const root = process.cwd();
const read = (p: string) => readFileSync(join(root, p), 'utf8');

describe('red theme — globals.css contract', () => {
  const css = read('app/globals.css');
  // The [data-theme='red'] block, not the bare :root/console block above it.
  const redBlockStart = css.indexOf(":root[data-theme='red'] {");
  const redBlock = css.slice(redBlockStart, css.indexOf('\n}', redBlockStart));

  test("[data-theme='red'] block exists and precedes the component-token additions", () => {
    expect(redBlockStart).toBeGreaterThan(-1);
  });

  test('carries the warm data ramp', () => {
    expect(redBlock).toMatch(/--ramp-1:\s*#ff4557/);
    expect(redBlock).toMatch(/--ramp-2:\s*#ff8a3d/);
    expect(redBlock).toMatch(/--ramp-3:\s*#ffc861/);
    expect(redBlock).toMatch(/--ramp-4:\s*#ffe9c2/);
  });

  test('carries the warm status triad', () => {
    expect(redBlock).toMatch(/--ok:\s*#ff8a3d/); // orange: never yellow text
    expect(redBlock).toMatch(/--warn:\s*#ffc861/);
    expect(redBlock).toMatch(/--err:\s*#ff3355/);
  });

  test('brain, funnel, glass, tile and workflow tokens are all re-keyed warm', () => {
    for (const v of ['--brain-1', '--funnel-s0', '--glass-a', '--glass-v-top', '--tile-glow-a-rgb', '--social-linkedin', '--wf-rail-a', '--wf-on-a', '--accent-rgb']) {
      expect(redBlock, v).toContain(`${v}:`);
    }
    expect(redBlock).not.toMatch(/#(6e96f2|6656e3|bd5fe8|00ffab|3fd968)/);
  });

  test('workflows canvas stylesheet carries no mint/blue/violet literals', () => {
    // the canvas ships on the workflows-canvas branch first; skip where it has not landed
    if (!existsSync(join(root, 'components/workflow-canvas/canvas.css'))) return;
    const wf = read('components/workflow-canvas/canvas.css');
    expect(wf).not.toMatch(/rgba\(0, ?255, ?171/);
    expect(wf).not.toMatch(/#(00ffab|6e96f2|6656e3|0c3f31|d9fff1|06c98a)/);
    const engine = read('components/workflow-canvas/engine.ts');
    // literals may remain only as tok() fallbacks, never as the drawn value
    expect(engine).not.toMatch(/strokeStyle = 'rgba\(0,255,171/);
    expect(engine).not.toMatch(/fill="#00ffab"|stop-color="#00ffab"|\? '#00ffab'/);
  });

  test('glass chrome rides tokens, no literal teal', () => {
    const glass = css.slice(css.indexOf('.glass-panel {'), css.indexOf('.glass-chip {'));
    expect(glass).toContain('var(--glass-a)');
    expect(glass).toContain('var(--glass-v-top)');
    expect(glass).not.toMatch(/rgba\(30, 52, 48|rgba\(178, 166, 255/);
  });

  test('scrollbars are hidden everywhere', () => {
    expect(css).toMatch(/scrollbar-width:\s*none/);
    expect(css).toMatch(/::-webkit-scrollbar \{\s*display: none/);
  });

  test('hoverable/nav cold-hover blue is re-toned under red', () => {
    // glass faces are excluded from the glow (their hover moves the glass knobs, glass phase 3)
    expect(css).toMatch(/\[data-theme='red'\] \.hoverable:not\([^{]*\):hover \{/);
    expect(css).toContain("[data-theme='red'] .os-sidebar a:hover");
  });
});

describe('red theme — banned literals removed at their old call sites', () => {
  // Each entry: a file we converted, and the exact green/blue/teal/cyan
  // literal(s) that used to live there. These are content greps scoped to
  // ONE file each, so they can't false-positive on the console token
  // definitions in globals.css (a different file entirely) or on literals
  // that were deliberately left because they aren't a banned hue (gold,
  // pink/magenta, or a sub-20-unit-channel-spread neutral gray).
  const cases: Array<{ file: string; banned: string[] }> = [
    {
      file: 'components/Home6Console.tsx',
      banned: ['#43b3a2', '#5fd3a2', '#7dffd4', '#8f7ce0', '#d9e4fb', 'rgba(26,58,55', 'rgba(38,62,60', 'rgba(210,230,225'],
    },
    {
      file: 'components/GBrainRadial.tsx',
      banned: ['#7dffd4', '#0a5a41', '#eafff6', '#dffff2', '#8fffd9', 'rgba(10,58,42', 'rgba(8,42,32', 'rgba(16,30,26', 'rgba(210,230,225'],
    },
    {
      file: 'components/SkillHierarchy.tsx',
      banned: ['#0a5a41', '#eafff6', 'rgba(10,58,42', 'rgba(210,230,225'],
    },
    {
      file: 'components/KnowledgeGraph.tsx',
      banned: ['#2a2c4e', '#121428', '#05060f'],
    },
    {
      file: 'components/knowledge/KnowledgeCanvas.tsx',
      banned: ['#00ac7c', 'rgba(10,56,42', 'rgba(190,220,212'],
    },
    {
      file: 'components/knowledge/DriftRail.tsx',
      banned: ['#8fe3a5', 'rgba(63,217,104', 'rgba(178,166,255'],
    },
    {
      file: 'app/agents/AgentRoster.tsx',
      banned: ['#23a94d'],
    },
    {
      file: 'components/brand-deals/DealBoard.tsx',
      banned: ["'#8fb4f6'", 'in oklab, #bd5fe8', 'in oklab, #6e96f2'],
    },
    {
      file: 'components/PipelineChart.tsx',
      banned: ['#221d4e'],
    },
    {
      file: 'lib/knowledge-views.ts',
      banned: ["'#4a83db'", "'#00ac7c'", "'#8dffd4'", "'#3ce8b0'", "'#00c489'", "'#0b8a63'", "'#12563f'"],
    },
    {
      file: 'components/knowledge/KnowledgePanels.tsx',
      banned: ["'#f04e52'", "'#f2b33d'", "'#6e96f2'"],
    },
  ];

  for (const { file, banned } of cases) {
    test(`${file} no longer contains its banned literals`, () => {
      const src = read(file);
      for (const literal of banned) {
        expect(src, `${file} should not contain ${literal}`).not.toContain(literal);
      }
    });
  }
});

describe('red theme — former purple exceptions now ride the tile-glow tokens ', () => {
  test('DealBoard insight card and /agents hero tiles carry no literal violet/blue', () => {
    // FounderOS keeps its own /agents and /social (founderos/, tests/founderos-screens.test.ts).
    for (const file of ['components/brand-deals/DealBoard.tsx']) {
      const src = read(file);
      expect(src, file).not.toMatch(/rgba\((102,86,227|110,150,242|189,95,232)/);
      expect(src, file).toContain('--tile-glow-a-rgb');
    }
  });

  test('lib/use-palette red set has no purple at all', () => {
    const src = read('lib/use-palette.ts');
    const red = src.slice(src.indexOf('const RED'), src.indexOf('observatory', src.indexOf('const RED')));
    expect(red).not.toMatch(/#(c084fc|a78bfa|8b7ff0|6656e3|bd5fe8)/);
  });
});
