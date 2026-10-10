import { describe, expect, it } from 'vitest';
import { getEnabledModuleIds, getModuleCatalog, getModuleGroups } from './workspaceModules';

describe('FounderOS v1 inside BusinessOS', () => {
  it('is a built-in module whose Home opens the /os view', () => {
    const mod = getModuleCatalog().find((m) => m.id === 'founderos-os');
    expect(mod).toMatchObject({ href: '/os', label: 'Home', group: 'FounderOS' });
  });

  it('is enabled by default, its group right after Operate', () => {
    const ids = getEnabledModuleIds({});
    expect(ids).toContain('founderos-os');
    const labels = getModuleGroups({}).map((g) => g.label);
    expect(labels.slice(0, 2)).toEqual(['Operate', 'FounderOS']);
  });
});
