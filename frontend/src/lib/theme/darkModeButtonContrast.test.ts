import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { describe, expect, it } from 'vitest';

const agentsPage = readFileSync(
	resolve(process.cwd(), 'src/routes/(app)/agents/+page.svelte'),
	'utf8'
);
const knowledgePage = readFileSync(
	resolve(process.cwd(), 'src/routes/(app)/knowledge/+page.svelte'),
	'utf8'
);

describe('dark-mode primary action contrast', () => {
	it('uses the adaptive primary button foreground in the Agents module', () => {
		expect(agentsPage).toMatch(
			/\.btn-primary\s*\{[^}]*background:\s*var\(--bos-v2-button-primary\)[^}]*color:\s*var\(--bos-v2-button-pureWhiteText\)/s
		);
	});

	it('uses the adaptive primary button pair for the Knowledge cloud CTA', () => {
		expect(knowledgePage).toMatch(
			/\.kb-cloud-gate-btn\s*\{[^}]*color:\s*var\(--bos-v2-button-pureWhiteText\)[^}]*background:\s*var\(--bos-v2-button-primary\)/s
		);
	});
});
