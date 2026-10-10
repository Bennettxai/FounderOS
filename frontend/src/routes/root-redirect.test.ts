import { render } from '@testing-library/svelte';
import { describe, expect, it, vi } from 'vitest';

const goto = vi.fn();
vi.mock('$app/navigation', () => ({ goto: (...a: unknown[]) => goto(...a) }));

import Root from './+page.svelte';

describe('/', () => {
	it('opens Founder OS on the operator console (which handles sign-in)', () => {
		render(Root);
		expect(goto).toHaveBeenCalledWith('/os', { replaceState: true });
	});
});
