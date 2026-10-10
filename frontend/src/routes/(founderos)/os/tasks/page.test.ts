import { render, waitFor } from '@testing-library/svelte';
import { existsSync } from 'node:fs';
import { resolve } from 'node:path';
import { beforeEach, describe, expect, it, vi } from 'vitest';

vi.mock('$lib/founderos/api', async (orig) => ({ ...(await orig<typeof import('$lib/founderos/api')>()), founderosFetch: vi.fn() }));

import { founderosFetch } from '$lib/founderos/api';
import Page from './+page.svelte';

const fetchMock = vi.mocked(founderosFetch);

// braces: a function returned from beforeEach is run as its teardown
beforeEach(() => {
	fetchMock.mockReset();
});

// FounderOS v1 (the demo) keeps /tasks as its own page (app/tasks/page.tsx),
// not the later build's redirect into the Agents tab.
describe('/os/tasks route', () => {
	it('is a real page, not a redirect', () => {
		expect(existsSync(resolve(__dirname, '+page.svelte'))).toBe(true);
		expect(existsSync(resolve(__dirname, '+page.ts'))).toBe(false);
	});

	it('loads the tasks view and says so while it reads', async () => {
		fetchMock.mockImplementation(() => new Promise(() => {}));
		const { getByText } = render(Page);
		await waitFor(() => expect(fetchMock).toHaveBeenCalledWith('/pages/tasks'));
		expect(getByText('loading the task queues…')).toBeTruthy();
	});

	it('a failed read is an honest error line, not a blank page', async () => {
		fetchMock.mockRejectedValue(new Error('backend down'));
		const { findByText } = render(Page);
		expect(await findByText('Could not load tasks: backend down')).toBeTruthy();
	});
});
