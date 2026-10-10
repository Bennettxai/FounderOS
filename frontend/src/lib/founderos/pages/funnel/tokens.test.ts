import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { describe, expect, it } from 'vitest';

// FounderOS v1's funnel controls are `tracking-wide` (0.025em); 0.05em made each
// venture/filter chip 3-4px wider and pushed the status row right.
describe('funnel control chips', () => {
	it('track like prod tracking-wide', () => {
		const css = readFileSync(resolve(__dirname, 'tokens.css'), 'utf8');
		const ctl = css.match(/\.bn-fn \.fn-ctl \{([^}]*)\}/)?.[1] ?? '';
		expect(ctl).toMatch(/letter-spacing:\s*0\.025em/);
	});
});

// FounderOS v1 globals.css .funnel-halo / .funnel-hub-ring: the halo grows 0.5→2
// and is gone by 70%; hub rings breathe (scale + opacity) over 5.5s; under
// reduced motion both stop and the halo is hidden.
describe('funnel heartbeat motion', () => {
	const css = readFileSync(resolve(__dirname, 'tokens.css'), 'utf8');
	it('halo keyframes match prod', () => {
		const kf = css.match(/@keyframes bn-fn-halo \{([\s\S]*?)\n\}/)?.[1] ?? '';
		expect(kf).toMatch(/0% \{ transform: scale\(0\.5\); opacity: 0\.65; \}/);
		expect(kf).toMatch(/70%,\s*100% \{ transform: scale\(2\); opacity: 0; \}/);
	});
	it('hub rings breathe over 5.5s', () => {
		expect(css).toMatch(/\.bn-fn \.fn-hub-ring \{[^}]*animation: bn-fn-hub 5\.5s ease-in-out infinite/);
		const kf = css.match(/@keyframes bn-fn-hub \{([\s\S]*?)\n\}/)?.[1] ?? '';
		expect(kf).toMatch(/scale\(1\.06\)/);
	});
	it('reduced motion hides the halo', () => {
		const rm = css.match(/@media \(prefers-reduced-motion: reduce\) \{([\s\S]*?)\n\}/)?.[1] ?? '';
		expect(rm).toMatch(/\.bn-fn \.fn-halo \{[^}]*opacity: 0/);
	});
});
