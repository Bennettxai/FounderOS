import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { describe, expect, it } from 'vitest';

// FounderOS v1's CommsThreePane uses the compact 16px .kbd for its key hints.
describe('Comms key hints', () => {
	it('use the compact keycap', () => {
		const src = readFileSync(resolve(__dirname, 'ThreePane.svelte'), 'utf8');
		const caps = src.match(/<Kbd[^>]*>/g) ?? [];
		expect(caps.length).toBeGreaterThan(0);
		for (const c of caps) expect(c).toBe('<Kbd size="sm">');
	});
});
