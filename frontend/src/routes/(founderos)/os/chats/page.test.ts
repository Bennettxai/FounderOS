import { render, waitFor } from '@testing-library/svelte';
import { beforeEach, describe, expect, it, vi } from 'vitest';

vi.mock('$lib/founderos/api', () => ({
	founderosFetch: vi.fn(),
	FounderosApiError: class extends Error {},
	isGuardRefusal: () => false,
	founderosUrl: (p: string) => `/api/founderos${p}`
}));

import { founderosFetch } from '$lib/founderos/api';
import Page from './+page.svelte';

const fetchMock = vi.mocked(founderosFetch);

beforeEach(() => {
	fetchMock.mockReset();
});

describe('/os/chats route', () => {
	it('is the real chat hub (not a 404, not the porting placeholder) and loads its view', async () => {
		fetchMock.mockImplementation(() => new Promise(() => {}));
		const { getByText, queryByText } = render(Page);
		expect(getByText('Chats')).toBeTruthy();
		expect(queryByText('Porting in progress')).toBeNull();
		await waitFor(() => expect(fetchMock).toHaveBeenCalledWith('/pages/chats'));
	});
});
