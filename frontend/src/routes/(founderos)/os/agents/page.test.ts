import { render, waitFor } from '@testing-library/svelte';
import { beforeEach, describe, expect, it, vi } from 'vitest';

vi.mock('$lib/founderos/api', () => ({
	founderosFetch: vi.fn(),
	FounderosApiError: class extends Error {},
	founderosUrl: (p: string) => `/api/founderos${p}`
}));

import { founderosFetch } from '$lib/founderos/api';
import Page from './+page.svelte';

const fetchMock = vi.mocked(founderosFetch);

beforeEach(() => {
	fetchMock.mockReset();
});

describe('/os/agents route', () => {
	it('is the real page, not the porting placeholder, and loads the board view', async () => {
		fetchMock.mockImplementation(() => new Promise(() => {}));
		const { getByText, queryByText } = render(Page);
		expect(getByText('Real Agents')).toBeTruthy();
		expect(queryByText('Porting in progress')).toBeNull();
		await waitFor(() => expect(fetchMock).toHaveBeenCalledWith('/pages/agents/view'));
		expect(getByText('reading the board…')).toBeTruthy();
	});
});
