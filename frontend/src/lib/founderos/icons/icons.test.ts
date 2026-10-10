import { execSync } from 'node:child_process';
import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { render } from '@testing-library/svelte';
import { describe, expect, it } from 'vitest';
import * as icons from './index';

// FounderOS v1 draws lucide-react 0.441 glyphs; lucide-svelte 0.562 changed 38 of
// the icons the the operator pages use (Brain, Mail, Search, Users, Megaphone, ...).
// The the operator views import this set, generated from 0.441's icon nodes.
describe('the operator icon set', () => {
	it('draws the 0.441 Megaphone, not the redrawn one', () => {
		const { container } = render(icons.Megaphone, { size: 15 });
		const ds = [...container.querySelectorAll('path')].map((p) => p.getAttribute('d'));
		expect(ds).toEqual(['m3 11 18-5v12L3 14v-3z', 'M11.6 16.8a3 3 0 1 1-5.8-1.6']);
		const svg = container.querySelector('svg')!;
		expect(svg.getAttribute('width')).toBe('15');
		expect(svg.classList.contains('lucide-megaphone')).toBe(true);
	});

	it('no the operator view imports lucide-svelte directly', () => {
		const root = resolve(__dirname, '../../..');
		const hits = execSync(`grep -rl "from 'lucide-svelte'" lib/founderos "routes/(founderos)" || true`, { cwd: root }).toString().trim();
		expect(hits.split('\n').filter((f) => f && !f.startsWith('lib/founderos/icons/'))).toEqual([]);
	});
});
