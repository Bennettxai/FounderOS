import { describe, expect, it } from 'vitest';
import { getModuleCatalog } from '$lib/config/workspaceModules';
import { FOUNDEROS_COMMAND_GROUP } from './commandModules';

describe('FounderOS section on the Command page', () => {
	it('shows a card for every FounderOS module in the sidebar, same order, each described', () => {
		const sidebar = getModuleCatalog().filter((m) => m.group === 'FounderOS');
		expect(FOUNDEROS_COMMAND_GROUP.label).toBe('FounderOS');
		expect(FOUNDEROS_COMMAND_GROUP.modules.map((m) => [m.label, m.href])).toEqual(
			sidebar.map((m) => [m.label, m.href])
		);
		for (const m of FOUNDEROS_COMMAND_GROUP.modules) expect(m.desc.length).toBeGreaterThan(20);
	});
});
