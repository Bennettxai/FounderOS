import { describe, expect, it } from 'vitest';
import { founderosCommandGroup, founderosHrefsFor } from './commandModules';

const personal = { enabled_builtin_modules: ['dashboard', 'founderos-os', 'founderos-trading', 'founderos-finances'] };

describe('FounderOS modules follow the workspace', () => {
	it('a workspace with no module list gets every FounderOS page', () => {
		expect(founderosHrefsFor({}).size).toBe(24);
	});

	it('a workspace with a list gets only its the operator pages', () => {
		expect([...founderosHrefsFor(personal)].sort()).toEqual(['/os', '/os/finances', '/os/trading']);
	});

	it('the Command page shows only that workspace’s the operator cards', () => {
		expect(founderosCommandGroup(personal).modules.map((m) => m.label)).toEqual(['Home', 'Finances', 'Trading']);
	});
});
