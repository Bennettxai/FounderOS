import { render } from '@testing-library/svelte';
import { createRawSnippet } from 'svelte';
import { describe, expect, it } from 'vitest';
import ToggleChip from './ToggleChip.svelte';

// A chip is one line: "→ Optimal Engine" must not wrap and squeeze its row
// mates (prod's chips never wrap).
describe('ToggleChip', () => {
	it('stays on one line and does not shrink', () => {
		const children = createRawSnippet(() => ({ render: () => '<span>→ Optimal Engine</span>' }));
		const { container } = render(ToggleChip, { on: false, children });
		const chip = container.querySelector('.bn-toggle-chip')!;
		expect(chip.classList.contains('whitespace-nowrap')).toBe(true);
		expect(chip.classList.contains('shrink-0')).toBe(true);
	});
});
