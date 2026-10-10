import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { describe, expect, it } from 'vitest';

// FounderOS v1 app/layout.tsx loads JetBrains Mono at 400/500/600/700. Without
// 600 every font-semibold falls through to 700 and the port reads bolder.
describe('app.html font weights', () => {
	it('loads JetBrains Mono at every weight production loads', () => {
		const html = readFileSync(resolve(__dirname, '../../app.html'), 'utf8');
		const m = html.match(/family=JetBrains\+Mono:wght@([\d;]+)/);
		expect(m?.[1].split(';')).toEqual(['400', '500', '600', '700']);
	});
});
