import { describe, expect, test } from 'vitest';
import { readFileSync, readdirSync } from 'node:fs';
import { Spotlight, PageSpotlight } from '@/components/Spotlight';
const read = (p: string) => readFileSync(p, 'utf8');
describe('public demo refresh', () => {
  test('no cursor glow is rendered anywhere', () => {
    expect(Spotlight()).toBeNull();
    expect(PageSpotlight()).toBeNull();
    expect(read('app/layout.tsx')).not.toContain('<PageSpotlight />');
  });
  test('home uses the current slab with the generic demo operator', () => {
    expect(read('app/page.tsx')).toContain('os-slab');
    expect(read('app/page.tsx')).not.toMatch(/Bennett|Merydian|bennettspooner/);
  });
});

function sourceFiles(dir: string): string[] {
  return readdirSync(dir, { withFileTypes: true }).flatMap((e) => {
    const name = `${dir}/${e.name}`;
    return e.isDirectory() ? sourceFiles(name) : /\.(tsx?|css)$/.test(name) ? [name] : [];
  });
}
test('demo application source excludes private operator names, brands and hosts', () => {
  const privateNames = /bennett|merydian|larps-mac|tail090dce|clue-agent|agency accelerant/i;
  for (const file of ['app', 'components', 'lib'].flatMap(sourceFiles)) {
    expect(read(file), file).not.toMatch(privateNames);
  }
});
