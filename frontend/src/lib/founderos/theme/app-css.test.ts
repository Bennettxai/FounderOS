// app.css once carried an UNLAYERED `img, video, canvas, svg, iframe
// { max-width: 100%; height: auto }`. Unlayered CSS beats every Tailwind v4
// layer, so `h-4`, `max-w-[…]` and friends were silently ignored on svgs and
// images across the app (round-1 finding: the operator pages had to fall back to
// inline styles). It must live in the base layer, where utilities win.
import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { describe, expect, it } from 'vitest';

const css = readFileSync(resolve(__dirname, '../../../app.css'), 'utf8');

/** The CSS outside every @layer block (brace-matched). */
function unlayered(src: string): string {
	let out = '';
	let i = 0;
	while (i < src.length) {
		const at = src.indexOf('@layer', i);
		if (at < 0) {
			out += src.slice(i);
			break;
		}
		out += src.slice(i, at);
		const open = src.indexOf('{', at);
		const semi = src.indexOf(';', at);
		if (semi >= 0 && semi < open) {
			i = semi + 1; // `@layer a, b;` statement
			continue;
		}
		let depth = 0;
		let j = open;
		for (; j < src.length; j++) {
			if (src[j] === '{') depth++;
			else if (src[j] === '}' && --depth === 0) break;
		}
		i = j + 1;
	}
	return out;
}

describe('app.css media sizing rule', () => {
	it('is not unlayered, so Tailwind utilities can size svgs and images', () => {
		expect(unlayered(css)).not.toMatch(/(^|[\s,}])svg\s*[,{][^}]*max-width:\s*100%/);
		expect(unlayered(css)).not.toMatch(/(^|[\s,}])img\s*,\s*video\s*,\s*canvas\s*,\s*svg/);
	});

	it('still caps media at its container, from the base layer', () => {
		expect(css).toMatch(/@layer base \{\s*\/\*[^*]*\*\/\s*img,\s*video,\s*canvas,\s*svg,\s*iframe \{\s*max-width: 100%;\s*height: auto;\s*\}\s*\}/);
	});
});
