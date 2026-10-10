import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { describe, expect, it } from 'vitest';

// The kit's .bn-caret::after draws a '▌' after page titles. The filter box's
// blinking caret must not wear that class, or a white block sits beside it.
describe('Brand Deals filter caret', () => {
	it('uses its own class, not the kit title caret', () => {
		const src = readFileSync(resolve(__dirname, 'DealBoard.svelte'), 'utf8');
		expect(src).not.toMatch(/class="bn-caret\b/);
		expect(src).not.toMatch(/^\s*\.bn-caret\s*\{/m);
		expect(src).toMatch(/class="bn-bd-caret\b/);
	});
});
